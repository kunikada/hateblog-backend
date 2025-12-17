package handler

import (
	"net/http"

	usecaseArchive "hateblog/internal/usecase/archive"
)

const defaultArchiveMinUsers = 5

// ArchiveHandler exposes archive endpoints.
type ArchiveHandler struct {
	service *usecaseArchive.Service
}

// NewArchiveHandler builds an ArchiveHandler.
func NewArchiveHandler(service *usecaseArchive.Service) *ArchiveHandler {
	return &ArchiveHandler{service: service}
}

// RegisterRoutes wires archive endpoints.
func (h *ArchiveHandler) RegisterRoutes(r chiRouter) {
	r.Get("/archive", h.handleArchive)
}

func (h *ArchiveHandler) handleArchive(w http.ResponseWriter, r *http.Request) {
	minUsers, err := readQueryInt(r, "min_users", 0, 10000, defaultArchiveMinUsers)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	items, err := h.service.List(r.Context(), minUsers)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	resp := archiveResponse{
		Items: make([]archiveItemResponse, 0, len(items)),
	}
	for _, item := range items {
		resp.Items = append(resp.Items, archiveItemResponse{
			Date:  item.Date.Format("2006-01-02"),
			Count: item.Count,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

type archiveResponse struct {
	Items []archiveItemResponse `json:"items"`
}

type archiveItemResponse struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}
