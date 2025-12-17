package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// RouterConfig bundles handler dependencies.
type RouterConfig struct {
	EntryHandler   *EntryHandler
	ArchiveHandler *ArchiveHandler
	RankingHandler *RankingHandler
	TagHandler     *TagHandler
	SearchHandler  *SearchHandler
	MetricsHandler *MetricsHandler
	FaviconHandler *FaviconHandler
	HealthHandler  *HealthHandler
}

// NewRouter wires handlers and middlewares.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	if cfg.EntryHandler != nil {
		cfg.EntryHandler.RegisterRoutes(r)
	}
	if cfg.ArchiveHandler != nil {
		cfg.ArchiveHandler.RegisterRoutes(r)
	}
	if cfg.RankingHandler != nil {
		cfg.RankingHandler.RegisterRoutes(r)
	}
	if cfg.TagHandler != nil {
		cfg.TagHandler.RegisterRoutes(r)
	}
	if cfg.SearchHandler != nil {
		cfg.SearchHandler.RegisterRoutes(r)
	}
	if cfg.MetricsHandler != nil {
		cfg.MetricsHandler.RegisterRoutes(r)
	}
	if cfg.FaviconHandler != nil {
		cfg.FaviconHandler.RegisterRoutes(r)
	}
	if cfg.HealthHandler != nil {
		r.Get("/health", cfg.HealthHandler.ServeHTTP)
	}
	return r
}
