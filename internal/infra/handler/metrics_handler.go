package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	domainEntry "hateblog/internal/domain/entry"
	usecaseMetrics "hateblog/internal/usecase/metrics"
)

// MetricsHandler handles /metrics endpoints.
type MetricsHandler struct {
	service *usecaseMetrics.Service
}

// NewMetricsHandler creates a MetricsHandler.
func NewMetricsHandler(service *usecaseMetrics.Service) *MetricsHandler {
	return &MetricsHandler{service: service}
}

// RegisterRoutes wires metrics routes.
func (h *MetricsHandler) RegisterRoutes(r chiRouter) {
	r.Post("/metrics/clicks", h.handleRecordClick)
}

func (h *MetricsHandler) handleRecordClick(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, r, http.StatusInternalServerError, errServiceUnavailable)
		return
	}
	var req clickMetricsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	if req.EntryID == uuid.Nil {
		writeError(w, r, http.StatusBadRequest, errors.New("entry_id is required"))
		return
	}
	if err := h.service.RecordClick(r.Context(), domainEntry.ID(req.EntryID)); err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusCreated, metricsResponse{
		Success: true,
		Message: "click recorded",
	})
}

type clickMetricsRequest struct {
	EntryID   uuid.UUID `json:"entry_id"`
	Referrer  *string   `json:"referrer"`
	UserAgent *string   `json:"user_agent"`
}

type metricsResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
