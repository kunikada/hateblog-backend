package handler

import (
	"context"
	"net/http"
	"time"
)

// HealthChecker defines dependencies that can be health-checked.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// HealthHandler handles /health endpoint.
type HealthHandler struct {
	DB    HealthChecker
	Cache HealthChecker
}

// ServeHTTP responds with dependency status.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type component struct {
		Name   string `json:"name"`
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	status := http.StatusOK
	components := []component{}

	if h.DB != nil {
		if err := h.DB.HealthCheck(ctx); err != nil {
			status = http.StatusServiceUnavailable
			components = append(components, component{Name: "database", Status: "unhealthy", Error: err.Error()})
		} else {
			components = append(components, component{Name: "database", Status: "healthy"})
		}
	}

	if h.Cache != nil {
		if err := h.Cache.HealthCheck(ctx); err != nil {
			status = http.StatusServiceUnavailable
			components = append(components, component{Name: "redis", Status: "unhealthy", Error: err.Error()})
		} else {
			components = append(components, component{Name: "redis", Status: "healthy"})
		}
	}

	writeJSON(w, status, map[string]any{
		"status":     statusLabel(status),
		"components": components,
		"checked_at": time.Now().UTC(),
	})
}

func statusLabel(code int) string {
	if code == http.StatusOK {
		return "healthy"
	}
	return "unhealthy"
}
