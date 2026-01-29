package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	domainEntry "hateblog/internal/domain/entry"
)

func readQueryInt(r *http.Request, key string, min, max, def int) (int, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return def, nil
	}
	return parseInt(key, raw, min, max)
}

func requireQueryInt(r *http.Request, key string, min, max int) (int, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	return parseInt(key, raw, min, max)
}

func parseInt(name, value string, min, max int) (int, error) {
	v, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	if v < min {
		return 0, fmt.Errorf("%s must be >= %d", name, min)
	}
	if max > 0 && v > max {
		return 0, fmt.Errorf("%s must be <= %d", name, max)
	}
	return v, nil
}

func readQuerySort(r *http.Request, key string, def domainEntry.SortType) (domainEntry.SortType, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return def, nil
	}
	switch domainEntry.SortType(raw) {
	case domainEntry.SortNew, domainEntry.SortHot:
		return domainEntry.SortType(raw), nil
	default:
		return "", fmt.Errorf("%s must be one of new, hot", key)
	}
}
