package handler

import (
	"errors"
	"net/http"
	"strings"

	appSearch "hateblog/internal/app/search"
)

// SearchHandler serves /search endpoint.
type SearchHandler struct {
	service *appSearch.Service
}

// NewSearchHandler builds a SearchHandler.
func NewSearchHandler(service *appSearch.Service) *SearchHandler {
	return &SearchHandler{service: service}
}

// RegisterRoutes adds search routes.
func (h *SearchHandler) RegisterRoutes(r chiRouter) {
	r.Get("/search", h.handleSearch)
}

func (h *SearchHandler) handleSearch(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusInternalServerError, errServiceUnavailable)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeError(w, http.StatusBadRequest, errors.New("q is required"))
		return
	}

	limit, err := readQueryInt(r, "limit", 1, maxTagLimit, defaultLimit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	offset, err := readQueryInt(r, "offset", 0, 0, 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	minUsers, err := readQueryInt(r, "min_users", 0, 10000, defaultMin)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.Search(r.Context(), q, appSearch.Params{
		MinBookmarkCount: minUsers,
		Limit:            limit,
		Offset:           offset,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	resp := searchResponse{
		Query:   result.Query,
		Entries: make([]entryResponse, 0, len(result.Entries)),
		Total:   result.Total,
		Limit:   result.Limit,
		Offset:  result.Offset,
	}
	for _, ent := range result.Entries {
		resp.Entries = append(resp.Entries, toEntryResponse(ent))
	}

	writeJSON(w, http.StatusOK, resp)
}

type searchResponse struct {
	Query   string          `json:"query"`
	Entries []entryResponse `json:"entries"`
	Total   int64           `json:"total"`
	Limit   int             `json:"limit"`
	Offset  int             `json:"offset"`
}
