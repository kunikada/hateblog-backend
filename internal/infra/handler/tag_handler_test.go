package handler

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"

	domainEntry "hateblog/internal/domain/entry"
	domainTag "hateblog/internal/domain/tag"
	usecaseEntry "hateblog/internal/usecase/entry"
)

func TestTagHandler_GetEntriesByTag(t *testing.T) {
	tagID := uuid.New()
	tagName := "programming"

	entry1 := newTestEntry(uuid.New(), "Programming Entry 1", 100)
	entry2 := newTestEntry(uuid.New(), "Programming Entry 2", 50)

	tests := []struct {
		name            string
		tagPath         string
		queryParams     string
		mockTag         *domainTag.Tag
		mockTagError    error
		mockEntries     []*domainEntry.Entry
		mockTotal       int64
		wantStatus      int
		wantEntryCount  int
		wantTotal       int64
		wantLimit       int
		wantOffset      int
	}{
		{
			name:           "success with default parameters",
			tagPath:        tagName,
			queryParams:    "",
			mockTag:        newTestTag(tagID, tagName),
			mockEntries:    []*domainEntry.Entry{entry1, entry2},
			mockTotal:      2,
			wantStatus:     http.StatusOK,
			wantEntryCount: 2,
			wantTotal:      2,
			wantLimit:      defaultTagLimit,
			wantOffset:     0,
		},
		{
			name:           "success with custom limit and offset",
			tagPath:        tagName,
			queryParams:    "?limit=10&offset=5",
			mockTag:        newTestTag(tagID, tagName),
			mockEntries:    []*domainEntry.Entry{entry1},
			mockTotal:      100,
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
			wantTotal:      100,
			wantLimit:      10,
			wantOffset:     5,
		},
		{
			name:           "success with min_users filter",
			tagPath:        tagName,
			queryParams:    "?min_users=50",
			mockTag:        newTestTag(tagID, tagName),
			mockEntries:    []*domainEntry.Entry{entry1},
			mockTotal:      1,
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
			wantTotal:      1,
			wantLimit:      defaultTagLimit,
			wantOffset:     0,
		},
		{
			name:           "success with empty result",
			tagPath:        tagName,
			queryParams:    "",
			mockTag:        newTestTag(tagID, tagName),
			mockEntries:    []*domainEntry.Entry{},
			mockTotal:      0,
			wantStatus:     http.StatusOK,
			wantEntryCount: 0,
			wantTotal:      0,
			wantLimit:      defaultTagLimit,
			wantOffset:     0,
		},
		{
			name:         "error: tag not found",
			tagPath:      "nonexistent",
			queryParams:  "",
			mockTagError: fmt.Errorf("tag not found"),
			wantStatus:   http.StatusNotFound,
		},
		{
			name:        "error: empty tag",
			tagPath:     "",
			queryParams: "",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: invalid limit",
			tagPath:     tagName,
			queryParams: "?limit=abc",
			mockTag:     newTestTag(tagID, tagName),
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: limit too small",
			tagPath:     tagName,
			queryParams: "?limit=0",
			mockTag:     newTestTag(tagID, tagName),
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: limit too large",
			tagPath:     tagName,
			queryParams: fmt.Sprintf("?limit=%d", maxTagLimit+1),
			mockTag:     newTestTag(tagID, tagName),
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: invalid offset",
			tagPath:     tagName,
			queryParams: "?offset=xyz",
			mockTag:     newTestTag(tagID, tagName),
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: negative offset",
			tagPath:     tagName,
			queryParams: "?offset=-1",
			mockTag:     newTestTag(tagID, tagName),
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: invalid min_users",
			tagPath:     tagName,
			queryParams: "?min_users=abc",
			mockTag:     newTestTag(tagID, tagName),
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: negative min_users",
			tagPath:     tagName,
			queryParams: "?min_users=-1",
			mockTag:     newTestTag(tagID, tagName),
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTagRepo := &mockTagRepository{
				getByNameFunc: func(ctx context.Context, name string) (*domainTag.Tag, error) {
					if tt.mockTagError != nil {
						return nil, tt.mockTagError
					}
					return tt.mockTag, nil
				},
			}

			mockEntryRepo := &mockEntryRepository{
				entries: tt.mockEntries,
				total:   tt.mockTotal,
			}

			tagService := newTestTagService(mockTagRepo)
			entryService := newTestEntryService(mockEntryRepo)
			handler := NewTagHandler(tagService, entryService)

			ts := newTestServer(RouterConfig{
				TagHandler: handler,
			})
			defer ts.Close()

			path := fmt.Sprintf("/tags/%s/entries%s", tt.tagPath, tt.queryParams)
			resp := ts.get(t, path)
			defer resp.Body.Close()

			if tt.wantStatus != http.StatusOK {
				assertErrorResponse(t, resp, tt.wantStatus)
				return
			}

			result := assertEntryListResponse(t, resp)
			if len(result.Entries) != tt.wantEntryCount {
				t.Errorf("got %d entries, want %d", len(result.Entries), tt.wantEntryCount)
			}
			assertPagination(t, result.Total, result.Limit, result.Offset,
				tt.wantTotal, tt.wantLimit, tt.wantOffset)
		})
	}
}

func TestTagHandler_GetEntriesByTag_ServiceError(t *testing.T) {
	tagID := uuid.New()
	tagName := "programming"

	mockTagRepo := &mockTagRepository{
		getByNameFunc: func(ctx context.Context, name string) (*domainTag.Tag, error) {
			return newTestTag(tagID, tagName), nil
		},
	}

	mockEntryRepo := &mockEntryRepository{
		listFunc: func(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, error) {
			return nil, fmt.Errorf("database error")
		},
	}

	tagService := newTestTagService(mockTagRepo)
	entryService := newTestEntryService(mockEntryRepo)
	handler := NewTagHandler(tagService, entryService)

	ts := newTestServer(RouterConfig{
		TagHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, "/tags/programming/entries")
	defer resp.Body.Close()

	errResp := assertErrorResponse(t, resp, http.StatusInternalServerError)
	if errResp["error"] != "internal error" {
		t.Errorf("error message = %q, want %q", errResp["error"], "internal error")
	}
}

func TestTagHandler_GetEntriesByTag_RecordView(t *testing.T) {
	tagID := uuid.New()
	tagName := "programming"

	viewRecorded := false
	mockTagRepo := &mockTagRepository{
		getByNameFunc: func(ctx context.Context, name string) (*domainTag.Tag, error) {
			return newTestTag(tagID, tagName), nil
		},
		incrementViewHistoryFunc: func(ctx context.Context, id domainTag.ID, viewedAt time.Time) error {
			if id != tagID {
				t.Errorf("recorded view for tag ID %v, want %v", id, tagID)
			}
			viewRecorded = true
			return nil
		},
	}

	mockEntryRepo := &mockEntryRepository{
		entries: []*domainEntry.Entry{newTestEntry(uuid.New(), "Entry", 100)},
		total:   1,
	}

	tagService := newTestTagService(mockTagRepo)
	entryService := newTestEntryService(mockEntryRepo)
	handler := NewTagHandler(tagService, entryService)

	ts := newTestServer(RouterConfig{
		TagHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, "/tags/programming/entries")
	defer resp.Body.Close()

	assertStatus(t, resp, http.StatusOK)

	if !viewRecorded {
		t.Error("tag view should be recorded")
	}
}

func TestTagHandler_GetEntriesByTag_RecordViewError(t *testing.T) {
	tagID := uuid.New()
	tagName := "programming"

	mockTagRepo := &mockTagRepository{
		getByNameFunc: func(ctx context.Context, name string) (*domainTag.Tag, error) {
			return newTestTag(tagID, tagName), nil
		},
		incrementViewHistoryFunc: func(ctx context.Context, id domainTag.ID, viewedAt time.Time) error {
			return fmt.Errorf("failed to record view")
		},
	}

	mockEntryRepo := &mockEntryRepository{
		entries: []*domainEntry.Entry{newTestEntry(uuid.New(), "Entry", 100)},
		total:   1,
	}

	tagService := newTestTagService(mockTagRepo)
	entryService := newTestEntryService(mockEntryRepo)
	handler := NewTagHandler(tagService, entryService)

	ts := newTestServer(RouterConfig{
		TagHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, "/tags/programming/entries")
	defer resp.Body.Close()

	// Should still return 200 even if recording view fails
	assertStatus(t, resp, http.StatusOK)
}

func TestTagHandler_BoundaryValues(t *testing.T) {
	tagID := uuid.New()
	tagName := "test"

	tests := []struct {
		name       string
		limit      int
		wantStatus int
	}{
		{
			name:       "minimum valid limit",
			limit:      1,
			wantStatus: http.StatusOK,
		},
		{
			name:       "maximum valid limit",
			limit:      maxTagLimit,
			wantStatus: http.StatusOK,
		},
		{
			name:       "limit below minimum",
			limit:      0,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "limit above maximum",
			limit:      maxTagLimit + 1,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTagRepo := &mockTagRepository{
				getByNameFunc: func(ctx context.Context, name string) (*domainTag.Tag, error) {
					return newTestTag(tagID, tagName), nil
				},
			}

			mockEntryRepo := &mockEntryRepository{
				entries: []*domainEntry.Entry{},
				total:   0,
			}

			tagService := newTestTagService(mockTagRepo)
			entryService := newTestEntryService(mockEntryRepo)
			handler := NewTagHandler(tagService, entryService)

			ts := newTestServer(RouterConfig{
				TagHandler: handler,
			})
			defer ts.Close()

			path := fmt.Sprintf("/tags/test/entries?limit=%d", tt.limit)
			resp := ts.get(t, path)
			defer resp.Body.Close()

			if tt.wantStatus == http.StatusOK {
				assertEntryListResponse(t, resp)
			} else {
				assertErrorResponse(t, resp, tt.wantStatus)
			}
		})
	}
}
