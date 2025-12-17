package handler

import (
	"errors"
	"net/http"

	appFavicon "hateblog/internal/app/favicon"
	"hateblog/internal/pkg/hostname"
)

// FaviconHandler handles favicon proxy requests.
type FaviconHandler struct {
	service *appFavicon.Service
}

// NewFaviconHandler creates a new handler.
func NewFaviconHandler(service *appFavicon.Service) *FaviconHandler {
	return &FaviconHandler{service: service}
}

// RegisterRoutes registers favicon endpoint routes.
func (h *FaviconHandler) RegisterRoutes(r chiRouter) {
	r.Get("/favicons", h.handleGetFavicon)
}

func (h *FaviconHandler) handleGetFavicon(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	if domain == "" {
		writeError(w, http.StatusBadRequest, errMissingDomain)
		return
	}

	host, err := hostname.Normalize(domain)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	data, contentType, err := h.service.Fetch(r.Context(), host)
	if err != nil {
		if errors.Is(err, appFavicon.ErrRateLimited) {
			writeError(w, http.StatusTooManyRequests, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

var errMissingDomain = errors.New("domain is required")
