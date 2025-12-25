package handler

import "strings"

func normalizeAPIBasePath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	if trimmed != "/" {
		trimmed = strings.TrimRight(trimmed, "/")
	}
	return trimmed
}

func joinAPIPath(basePath, route string) string {
	normalized := normalizeAPIBasePath(basePath)
	if normalized == "" || normalized == "/" {
		if strings.HasPrefix(route, "/") {
			return route
		}
		return "/" + route
	}
	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}
	return normalized + route
}
