package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter wires handlers and middlewares.
func NewRouter(entryHandler *EntryHandler, faviconHandler *FaviconHandler, healthHandler *HealthHandler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	if entryHandler != nil {
		entryHandler.RegisterRoutes(r)
	}
	if faviconHandler != nil {
		faviconHandler.RegisterRoutes(r)
	}
	if healthHandler != nil {
		r.Get("/health", healthHandler.ServeHTTP)
	}
	return r
}
