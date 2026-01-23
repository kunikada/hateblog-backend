package handler

import (
	"errors"
	"net/http"

	"hateblog/internal/pkg/hostname"
	usecaseFavicon "hateblog/internal/usecase/favicon"
)

// FaviconHandler handles favicon proxy requests.
type FaviconHandler struct {
	service *usecaseFavicon.Service
}

// NewFaviconHandler creates a new handler.
func NewFaviconHandler(service *usecaseFavicon.Service) *FaviconHandler {
	return &FaviconHandler{service: service}
}

// RegisterRoutes registers favicon endpoint routes.
func (h *FaviconHandler) RegisterRoutes(r chiRouter) {
	r.Get("/favicons", h.handleGetFavicon)
}

func (h *FaviconHandler) handleGetFavicon(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	if domain == "" {
		writeError(w, r, http.StatusBadRequest, errMissingDomain)
		return
	}

	host, err := hostname.Normalize(domain)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}

	data, contentType, err := h.service.Fetch(r.Context(), host)
	if err != nil {
		if errors.Is(err, usecaseFavicon.ErrRateLimited) {
			writeError(w, r, http.StatusTooManyRequests, err)
			return
		}
		writeError(w, r, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

var errMissingDomain = errors.New("domain is required")
