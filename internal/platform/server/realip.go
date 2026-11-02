package server

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ParseTrustedProxies parses CIDRs or single IP addresses into prefixes.
func ParseTrustedProxies(values []string) ([]netip.Prefix, error) {
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if strings.Contains(v, "/") {
			prefix, err := netip.ParsePrefix(v)
			if err != nil {
				return nil, fmt.Errorf("invalid trusted proxy %q: %w", v, err)
			}
			prefixes = append(prefixes, prefix.Masked())
			continue
		}
		addr, err := netip.ParseAddr(v)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy %q: %w", v, err)
		}
		addr = addr.Unmap()
		prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
	}
	return prefixes, nil
}

// RealIP returns a middleware that sets r.RemoteAddr to the client IP
// only when the request comes from a trusted proxy.
// X-Forwarded-For is walked from right to left and the first untrusted address is used;
// X-Real-IP is used only when X-Forwarded-For is absent.
func RealIP(trustedProxies []netip.Prefix) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if len(trustedProxies) == 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			peer, ok := parseAddr(r.RemoteAddr)
			if ok && isTrusted(peer, trustedProxies) {
				if ip, found := forwardedClientIP(r, trustedProxies); found {
					r.RemoteAddr = ip.String()
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func forwardedClientIP(r *http.Request, trustedProxies []netip.Prefix) (netip.Addr, bool) {
	if values := r.Header.Values("X-Forwarded-For"); len(values) > 0 {
		hops := strings.Split(strings.Join(values, ","), ",")
		for i := len(hops) - 1; i >= 0; i-- {
			addr, ok := parseAddr(hops[i])
			if !ok {
				// A malformed hop means the chain beyond this point cannot be trusted.
				return netip.Addr{}, false
			}
			if !isTrusted(addr, trustedProxies) {
				return addr, true
			}
		}
		return netip.Addr{}, false
	}
	return parseAddr(r.Header.Get("X-Real-IP"))
}

func parseAddr(s string) (netip.Addr, bool) {
	s = strings.TrimSpace(s)
	if host, _, err := net.SplitHostPort(s); err == nil {
		s = host
	}
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Addr{}, false
	}
	return addr.Unmap(), true
}

func isTrusted(addr netip.Addr, trustedProxies []netip.Prefix) bool {
	for _, prefix := range trustedProxies {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
