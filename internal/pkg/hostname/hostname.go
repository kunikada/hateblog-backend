package hostname

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var hostPattern = regexp.MustCompile(`^[a-z0-9.-]+$`)

// Normalize validates and lowercases a hostname (or URL) for consistent use.
func Normalize(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("hostname is required")
	}
	if strings.Contains(value, "://") {
		u, err := url.Parse(value)
		if err != nil || u.Host == "" {
			return "", fmt.Errorf("invalid hostname: %s", raw)
		}
		value = u.Host
	}
	value = strings.TrimSuffix(strings.ToLower(value), ".")
	if value == "" {
		return "", fmt.Errorf("invalid hostname: %s", raw)
	}
	if strings.ContainsAny(value, "/ \\") {
		return "", fmt.Errorf("invalid hostname: %s", raw)
	}
	if !hostPattern.MatchString(value) {
		return "", fmt.Errorf("invalid hostname: %s", raw)
	}
	return value, nil
}
