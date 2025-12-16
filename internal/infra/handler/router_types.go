package handler

import "net/http"

// chiRouter allows registering handlers without importing chi in tests.
type chiRouter interface {
	Get(pattern string, handlerFn http.HandlerFunc)
}
