package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	usecaseAPIKey "hateblog/internal/usecase/api_key"
)

// APIKeyHandler handles /api-keys endpoints.
type APIKeyHandler struct {
	service *usecaseAPIKey.Service
}

// NewAPIKeyHandler creates an APIKeyHandler.
func NewAPIKeyHandler(service *usecaseAPIKey.Service) *APIKeyHandler {
	return &APIKeyHandler{service: service}
}

// RegisterRoutes wires API key routes.
func (h *APIKeyHandler) RegisterRoutes(r chiRouter) {
	r.Post("/api-keys", h.handleCreateAPIKey)
}

func (h *APIKeyHandler) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusInternalServerError, errServiceUnavailable)
		return
	}

	var req createAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	// Validate field lengths
	if req.Name != nil && len(*req.Name) > 100 {
		writeError(w, http.StatusBadRequest, errors.New("name must be at most 100 characters"))
		return
	}

	if req.Description != nil && len(*req.Description) > 500 {
		writeError(w, http.StatusBadRequest, errors.New("description must be at most 500 characters"))
		return
	}

	// Validate expires_at is not in the past
	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now()) {
		writeError(w, http.StatusBadRequest, errors.New("expires_at must be in the future"))
		return
	}

	// Extract metadata from request
	ip := extractIP(r.RemoteAddr)
	userAgent := r.UserAgent()
	referrer := r.Referer()

	// Generate API key
	generated, err := h.service.GenerateAPIKey(r.Context(), usecaseAPIKey.GenerateParams{
		Name:        req.Name,
		Description: req.Description,
		ExpiresAt:   req.ExpiresAt,
		CreatedIP:   &ip,
		CreatedUA:   &userAgent,
		CreatedRef:  &referrer,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Return success response
	writeJSON(w, http.StatusCreated, apiKeyResponse{
		ID:          generated.ID,
		Key:         generated.Key,
		Name:        generated.Name,
		Description: generated.Description,
		CreatedAt:   generated.CreatedAt,
		ExpiresAt:   generated.ExpiresAt,
	})
}

func extractIP(remoteAddr string) string {
	// Extract IP from RemoteAddr (format: "IP:port")
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		return remoteAddr[:idx]
	}
	return remoteAddr
}

type createAPIKeyRequest struct {
	Name        *string    `json:"name"`
	Description *string    `json:"description"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

type apiKeyResponse struct {
	ID          uuid.UUID  `json:"id"`
	Key         string     `json:"key"`
	Name        *string    `json:"name,omitempty"`
	Description *string    `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}
