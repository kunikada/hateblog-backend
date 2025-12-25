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

	APIBasePath       string
	Middlewares       []func(http.Handler) http.Handler
	PrometheusHandler http.Handler
}

// NewRouter wires handlers and middlewares.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	for _, mw := range cfg.Middlewares {
		if mw == nil {
			continue
		}
		r.Use(mw)
	}

	apiBasePath := normalizeAPIBasePath(cfg.APIBasePath)
	if apiBasePath == "" {
		apiBasePath = "/"
	}
	r.Route(apiBasePath, func(api chi.Router) {
		if cfg.EntryHandler != nil {
			cfg.EntryHandler.RegisterRoutes(api)
		}
		if cfg.ArchiveHandler != nil {
			cfg.ArchiveHandler.RegisterRoutes(api)
		}
		if cfg.RankingHandler != nil {
			cfg.RankingHandler.RegisterRoutes(api)
		}
		if cfg.TagHandler != nil {
			cfg.TagHandler.RegisterRoutes(api)
		}
		if cfg.SearchHandler != nil {
			cfg.SearchHandler.RegisterRoutes(api)
		}
		if cfg.MetricsHandler != nil {
			cfg.MetricsHandler.RegisterRoutes(api)
		}
		if cfg.FaviconHandler != nil {
			cfg.FaviconHandler.RegisterRoutes(api)
		}
		if cfg.HealthHandler != nil {
			api.Get("/health", cfg.HealthHandler.ServeHTTP)
		}
	})
	return r
}
