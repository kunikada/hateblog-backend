package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	domainTag "hateblog/internal/domain/tag"
	usecaseEntry "hateblog/internal/usecase/entry"
	usecaseTag "hateblog/internal/usecase/tag"
)

const (
	defaultTagLimit     = 25
	maxTagLimit         = 100
	defaultTagListLimit = 50
	maxTagListLimit     = 200
)

// TagHandler exposes tag endpoints.
type TagHandler struct {
	tagService   *usecaseTag.Service
	entryService *usecaseEntry.Service
	apiBasePath  string
}

// NewTagHandler builds a TagHandler.
func NewTagHandler(tagService *usecaseTag.Service, entryService *usecaseEntry.Service, apiBasePath string) *TagHandler {
	return &TagHandler{
		tagService:   tagService,
		entryService: entryService,
		apiBasePath:  normalizeAPIBasePath(apiBasePath),
	}
}

// RegisterRoutes wires tag endpoints.
func (h *TagHandler) RegisterRoutes(r chiRouter) {
	r.Get("/tags", h.handleListTags)
	r.Get("/tags/{tag}/entries", h.handleTagEntries)
}

func (h *TagHandler) handleListTags(w http.ResponseWriter, r *http.Request) {
	if h.tagService == nil {
		writeError(w, http.StatusInternalServerError, errServiceUnavailable)
		return
	}
	limit, err := readQueryInt(r, "limit", 1, maxTagListLimit, defaultTagListLimit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	offset, err := readQueryInt(r, "offset", 0, 0, 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	tags, err := h.tagService.List(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	resp := tagsResponse{
		Tags:   make([]tagItemResponse, 0, len(tags)),
		Limit:  limit,
		Offset: offset,
	}
	for _, t := range tags {
		resp.Tags = append(resp.Tags, tagItemResponse{
			ID:   t.ID,
			Name: t.Name,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *TagHandler) handleTagEntries(w http.ResponseWriter, r *http.Request) {
	if h.tagService == nil || h.entryService == nil {
		writeError(w, http.StatusInternalServerError, errServiceUnavailable)
		return
	}
	rawTag := chi.URLParam(r, "tag")
	if rawTag == "" {
		writeError(w, http.StatusBadRequest, errInvalidTag)
		return
	}

	tagEntity, err := h.tagService.GetByName(r.Context(), rawTag)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	limit, err := readQueryInt(r, "limit", 1, maxTagLimit, defaultTagLimit)
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

	result, err := h.entryService.ListTagEntries(r.Context(), tagEntity.Name, usecaseEntry.TagListParams{
		MinBookmarkCount: minUsers,
		Limit:            limit,
		Offset:           offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if err := h.tagService.RecordView(r.Context(), tagEntity.ID, time.Now()); err != nil {
		slog.Default().Warn("failed to record tag view", "tag", tagEntity.Name, "error", err)
	}

	writeJSON(w, http.StatusOK, buildEntryListResponse(result, limit, offset, h.apiBasePath))
}

var (
	errServiceUnavailable = errors.New("service unavailable")
	errInvalidTag         = errors.New("tag is required")
)

type tagsResponse struct {
	Tags   []tagItemResponse `json:"tags"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}

type tagItemResponse struct {
	ID   domainTag.ID `json:"id"`
	Name string       `json:"name"`
}
