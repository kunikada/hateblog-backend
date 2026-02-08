package handler

import (
	"errors"
	"net/http"
	"strings"

	domainEntry "hateblog/internal/domain/entry"
	usecaseSearch "hateblog/internal/usecase/search"
)

// SearchHandler serves /search endpoint.
type SearchHandler struct {
	service     *usecaseSearch.Service
	apiBasePath string
}

// NewSearchHandler builds a SearchHandler.
func NewSearchHandler(service *usecaseSearch.Service, apiBasePath string) *SearchHandler {
	return &SearchHandler{
		service:     service,
		apiBasePath: normalizeAPIBasePath(apiBasePath),
	}
}

// RegisterRoutes adds search routes.
func (h *SearchHandler) RegisterRoutes(r chiRouter) {
	r.Get("/search", h.handleSearch)
}

func (h *SearchHandler) handleSearch(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, r, http.StatusInternalServerError, errServiceUnavailable)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeError(w, r, http.StatusBadRequest, errors.New("q is required"))
		return
	}

	limit, err := readQueryInt(r, "limit", 1, 100, defaultLimit)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	offset, err := readQueryInt(r, "offset", 0, -1, 0)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	minUsers, err := readQueryInt(r, "min_users", 0, 10000, defaultMin)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	sortType, err := readQuerySort(r, "sort", domainEntry.SortHot)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}

	result, cacheHit, err := h.service.SearchWithCacheStatus(r.Context(), q, usecaseSearch.Params{
		MinBookmarkCount: minUsers,
		Limit:            limit,
		Offset:           offset,
		Sort:             sortType,
	})
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
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
		resp.Entries = append(resp.Entries, toEntryResponse(ent, h.apiBasePath))
	}

	setCacheStatusHeader(w, cacheHit)
	writeJSON(w, http.StatusOK, resp)
}

type searchResponse struct {
	Query   string          `json:"query"`
	Entries []entryResponse `json:"entries"`
	Total   int64           `json:"total"`
	Limit   int             `json:"limit"`
	Offset  int             `json:"offset"`
}
