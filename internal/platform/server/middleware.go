package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"hateblog/internal/platform/cache"
)

// RequestLogger returns a middleware that logs HTTP requests
func RequestLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Process request
			next.ServeHTTP(ww, r)

			// Log request
			duration := time.Since(start)
			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"query", r.URL.RawQuery,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", duration.Milliseconds(),
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)
		})
	}
}

// Recoverer returns a middleware that recovers from panics
func Recoverer(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					logger.Error("panic recovered",
						"error", rvr,
						"method", r.Method,
						"path", r.URL.Path,
					)

					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// CORS returns a middleware that handles CORS
func CORS(allowedOrigins []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				if origin != "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else if len(allowedOrigins) > 0 {
					w.Header().Set("Access-Control-Allow-Origin", allowedOrigins[0])
				}
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Authorization, X-API-Key")
				w.Header().Set("Access-Control-Max-Age", "3600")
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// APIKeyAuth returns a middleware that validates API keys
func APIKeyAuth(validAPIKey string, logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")

			if apiKey == "" {
				logger.Warn("missing API key", "path", r.URL.Path, "remote_addr", r.RemoteAddr)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"UNAUTHORIZED","message":"Missing API key"}`))
				return
			}

			if apiKey != validAPIKey {
				logger.Warn("invalid API key", "path", r.URL.Path, "remote_addr", r.RemoteAddr)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"UNAUTHORIZED","message":"Invalid API key"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders returns a middleware that adds security headers
func SecurityHeaders() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Content-Security-Policy", "default-src 'self'")

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitConfig configures the Redis-backed rate limiter.
type RateLimitConfig struct {
	Cache  *cache.Cache
	Limit  int
	Window time.Duration
	Logger *slog.Logger

	Prefix string
	Skip   func(r *http.Request) bool
	Key    func(r *http.Request) (string, error)
}

// RateLimit returns a middleware that enforces a simple fixed-window counter per key.
func RateLimit(cfg RateLimitConfig) func(next http.Handler) http.Handler {
	if cfg.Limit <= 0 || cfg.Window <= 0 || cfg.Cache == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	prefix := strings.TrimSpace(cfg.Prefix)
	if prefix == "" {
		prefix = "ratelimit"
	}
	logger := cfg.Logger

	skip := cfg.Skip
	if skip == nil {
		skip = func(*http.Request) bool { return false }
	}
	keyFunc := cfg.Key
	if keyFunc == nil {
		keyFunc = func(r *http.Request) (string, error) {
			ip := clientIP(r)
			if ip == "" {
				return "", fmt.Errorf("client ip is empty")
			}
			apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
			if apiKey != "" {
				return "key:" + apiKey, nil
			}
			return "ip:" + ip, nil
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			key, err := keyFunc(r)
			if err != nil {
				if logger != nil {
					logger.Debug("rate limit key build failed", "error", err, "path", r.URL.Path)
				}
				next.ServeHTTP(w, r)
				return
			}
			redisKey := prefix + ":" + key

			count, err := cfg.Cache.IncrementWithTTL(r.Context(), redisKey, cfg.Window)
			if err != nil {
				if logger != nil {
					logger.Debug("rate limit counter failed", "error", err, "path", r.URL.Path)
				}
				next.ServeHTTP(w, r)
				return
			}

			if int(count) > cfg.Limit {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(cfg.Window.Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "rate limit exceeded",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
