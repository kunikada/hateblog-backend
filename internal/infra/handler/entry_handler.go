package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	domainEntry "hateblog/internal/domain/entry"
	"hateblog/internal/domain/tag"
	usecaseEntry "hateblog/internal/usecase/entry"
)

const (
	defaultLimit = 25
	defaultMin   = 5
)

// EntryHandler exposes entry endpoints.
type EntryHandler struct {
	service *usecaseEntry.Service
}

// NewEntryHandler creates a new EntryHandler.
func NewEntryHandler(service *usecaseEntry.Service) *EntryHandler {
	return &EntryHandler{service: service}
}

// RegisterRoutes registers entry handlers on the router.
func (h *EntryHandler) RegisterRoutes(r chiRouter) {
	r.Get("/entries/new", h.handleNewEntries)
	r.Get("/entries/hot", h.handleHotEntries)
}

func (h *EntryHandler) handleNewEntries(w http.ResponseWriter, r *http.Request) {
	params, err := buildDayListParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.ListNewEntries(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, buildEntryListResponse(result, params.Limit, params.Offset))
}

func (h *EntryHandler) handleHotEntries(w http.ResponseWriter, r *http.Request) {
	params, err := buildDayListParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.ListHotEntries(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, buildEntryListResponse(result, params.Limit, params.Offset))
}

func buildEntryListResponse(result usecaseEntry.ListResult, limit, offset int) entryListResponse {
	resp := entryListResponse{
		Entries: make([]entryResponse, 0, len(result.Entries)),
		Total:   result.Total,
		Limit:   limit,
		Offset:  offset,
	}

	for _, ent := range result.Entries {
		resp.Entries = append(resp.Entries, toEntryResponse(ent))
	}

	return resp
}

func toEntryResponse(ent *domainEntry.Entry) entryResponse {
	resp := entryResponse{
		ID:            ent.ID,
		Title:         ent.Title,
		URL:           ent.URL,
		BookmarkCount: ent.BookmarkCount,
		PostedAt:      ent.PostedAt,
		Tags:          make([]entryTagResponse, 0, len(ent.Tags)),
		CreatedAt:     ent.CreatedAt,
		UpdatedAt:     ent.UpdatedAt,
		FaviconURL:    buildFaviconURL(ent.URL),
	}

	if ent.Excerpt != "" {
		text := ent.Excerpt
		resp.Excerpt = &text
	}
	if ent.Subject != "" {
		subject := ent.Subject
		resp.Subject = &subject
	}

	for _, tagging := range ent.Tags {
		resp.Tags = append(resp.Tags, entryTagResponse{
			TagID: tagging.TagID,
			Name:  tagging.Name,
			Score: tagging.Score,
		})
	}

	return resp
}

func buildDayListParams(r *http.Request) (usecaseEntry.DayListParams, error) {
	params := usecaseEntry.DayListParams{
		Limit:  defaultLimit,
		Offset: 0,
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		return usecaseEntry.DayListParams{}, fmt.Errorf("date is required")
	}
	if !isValidDate(date) {
		return usecaseEntry.DayListParams{}, fmt.Errorf("date must be YYYYMMDD")
	}
	params.Date = date

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return usecaseEntry.DayListParams{}, fmt.Errorf("invalid limit")
		}
		if limit < 1 || limit > domainEntry.MaxLimit {
			return usecaseEntry.DayListParams{}, fmt.Errorf("limit must be between 1 and %d", domainEntry.MaxLimit)
		}
		params.Limit = limit
	} else {
		params.Limit = defaultLimit
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return usecaseEntry.DayListParams{}, fmt.Errorf("offset must be >= 0")
		}
		params.Offset = offset
	}

	minUsers := defaultMin
	if minStr := r.URL.Query().Get("min_users"); minStr != "" {
		min, err := strconv.Atoi(minStr)
		if err != nil || min < 0 {
			return usecaseEntry.DayListParams{}, fmt.Errorf("min_users must be >= 0")
		}
		minUsers = min
	}
	params.MinBookmarkCount = minUsers

	return params, nil
}

func isValidDate(value string) bool {
	if len(value) != 8 {
		return false
	}
	_, err := time.Parse("20060102", value)
	return err == nil
}

func buildFaviconURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	host := u.Hostname()
	return fmt.Sprintf("/api/v1/favicons?domain=%s", host)
}

// entryListResponse matches EntryListResponse schema.
type entryListResponse struct {
	Entries []entryResponse `json:"entries"`
	Total   int64           `json:"total"`
	Limit   int             `json:"limit"`
	Offset  int             `json:"offset"`
}

type entryResponse struct {
	ID            domainEntry.ID     `json:"id"`
	Title         string             `json:"title"`
	URL           string             `json:"url"`
	PostedAt      time.Time          `json:"posted_at"`
	BookmarkCount int                `json:"bookmark_count"`
	Excerpt       *string            `json:"excerpt,omitempty"`
	Subject       *string            `json:"subject,omitempty"`
	Tags          []entryTagResponse `json:"tags"`
	FaviconURL    string             `json:"favicon_url"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

type entryTagResponse struct {
	TagID tag.ID  `json:"tag_id"`
	Name  string  `json:"tag_name"`
	Score float64 `json:"score"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	message := "internal error"
	if status == http.StatusBadRequest && err != nil {
		message = err.Error()
	}
	if status >= 500 {
		message = "internal error"
	}
	writeJSON(w, status, map[string]string{"error": message})
}
