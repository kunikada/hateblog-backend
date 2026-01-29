package handler

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	domainEntry "hateblog/internal/domain/entry"
	domainTag "hateblog/internal/domain/tag"
)

func TestTagHandler_GetEntriesByTag(t *testing.T) {
	tagID := uuid.New()
	tagName := "programming"

	entry1 := newTestEntry(uuid.New(), "Programming Entry 1", 100)
	entry2 := newTestEntry(uuid.New(), "Programming Entry 2", 50)

	tests := []struct {
		name           string
		tagPath        string
		queryParams    string
		mockTag        *domainTag.Tag
		mockTagError   error
		mockEntries    []*domainEntry.Entry
		mockTotal      int64
		wantStatus     int
		wantEntryCount int
		wantTotal      int64
		wantLimit      int
		wantOffset     int
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
			name:        "success with custom limit and offset",
			tagPath:     tagName,
			queryParams: "?limit=10&offset=5&min_users=0",
			mockTag:     newTestTag(tagID, tagName),
			mockEntries: []*domainEntry.Entry{
				newTestEntry(uuid.New(), "Entry 1", 100),
				newTestEntry(uuid.New(), "Entry 2", 90),
				newTestEntry(uuid.New(), "Entry 3", 80),
				newTestEntry(uuid.New(), "Entry 4", 70),
				newTestEntry(uuid.New(), "Entry 5", 60),
				newTestEntry(uuid.New(), "Entry 6", 50),
				newTestEntry(uuid.New(), "Entry 7", 40),
				newTestEntry(uuid.New(), "Entry 8", 30),
				newTestEntry(uuid.New(), "Entry 9", 20),
				newTestEntry(uuid.New(), "Entry 10", 10),
				newTestEntry(uuid.New(), "Entry 11", 5),
				newTestEntry(uuid.New(), "Entry 12", 3),
				newTestEntry(uuid.New(), "Entry 13", 2),
				newTestEntry(uuid.New(), "Entry 14", 1),
				newTestEntry(uuid.New(), "Entry 15", 1),
			},
			mockTotal:      15,
			wantStatus:     http.StatusOK,
			wantEntryCount: 10,
			wantTotal:      15,
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
		{
			name:        "error: invalid sort",
			tagPath:     tagName,
			queryParams: "?sort=popular",
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
			handler := NewTagHandler(tagService, entryService, testAPIBasePath)

			ts := newTestServer(RouterConfig{
				TagHandler: handler,
			})
			defer ts.Close()

			path := apiPath(fmt.Sprintf("/tags/entries/%s%s", tt.tagPath, tt.queryParams))
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
	handler := NewTagHandler(tagService, entryService, testAPIBasePath)

	ts := newTestServer(RouterConfig{
		TagHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/tags/entries/programming"))
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
	handler := NewTagHandler(tagService, entryService, testAPIBasePath)

	ts := newTestServer(RouterConfig{
		TagHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/tags/entries/programming"))
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
	handler := NewTagHandler(tagService, entryService, testAPIBasePath)

	ts := newTestServer(RouterConfig{
		TagHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/tags/entries/programming"))
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
			handler := NewTagHandler(tagService, entryService, testAPIBasePath)

			ts := newTestServer(RouterConfig{
				TagHandler: handler,
			})
			defer ts.Close()

			path := apiPath(fmt.Sprintf("/tags/entries/test?limit=%d", tt.limit))
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

func TestTagHandler_ListTags(t *testing.T) {
	tag1 := newTestTag(uuid.New(), "programming")
	tag2 := newTestTag(uuid.New(), "golang")
	tag3 := newTestTag(uuid.New(), "tech")

	tests := []struct {
		name        string
		queryParams string
		mockTags    []domainTag.Tag
		mockError   error
		wantStatus  int
		wantCount   int
		wantLimit   int
		wantOffset  int
	}{
		{
			name:        "success with default parameters",
			queryParams: "",
			mockTags: []domainTag.Tag{
				*tag1, *tag2, *tag3,
			},
			wantStatus: http.StatusOK,
			wantCount:  3,
			wantLimit:  defaultTagListLimit,
			wantOffset: 0,
		},
		{
			name:        "success with custom limit",
			queryParams: "?limit=10",
			mockTags: []domainTag.Tag{
				*tag1, *tag2,
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
			wantLimit:  10,
			wantOffset: 0,
		},
		{
			name:        "success with limit and offset",
			queryParams: "?limit=20&offset=5",
			mockTags: []domainTag.Tag{
				*tag1,
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
			wantLimit:  20,
			wantOffset: 5,
		},
		{
			name:        "success with empty result",
			queryParams: "",
			mockTags:    []domainTag.Tag{},
			wantStatus:  http.StatusOK,
			wantCount:   0,
			wantLimit:   defaultTagListLimit,
			wantOffset:  0,
		},
		{
			name:        "error: invalid limit format",
			queryParams: "?limit=abc",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: limit too small",
			queryParams: "?limit=0",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: limit too large",
			queryParams: "?limit=201",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: invalid offset format",
			queryParams: "?offset=xyz",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: negative offset",
			queryParams: "?offset=-1",
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockTagRepository{
				tags: tt.mockTags,
				err:  tt.mockError,
			}
			if tt.mockError != nil {
				mockRepo.listFunc = func(ctx context.Context, limit, offset int) ([]domainTag.Tag, error) {
					return nil, tt.mockError
				}
			}

			tagService := newTestTagService(mockRepo)
			handler := NewTagHandler(tagService, nil, testAPIBasePath)

			ts := newTestServer(RouterConfig{
				TagHandler: handler,
			})
			defer ts.Close()

			resp := ts.get(t, apiPath("/tags"+tt.queryParams))
			defer resp.Body.Close()

			if tt.wantStatus != http.StatusOK {
				assertErrorResponse(t, resp, tt.wantStatus)
				return
			}

			assertStatus(t, resp, http.StatusOK)
			assertContentType(t, resp, "application/json")

			var result tagsResponse
			decodeJSON(t, resp, &result)

			if len(result.Tags) != tt.wantCount {
				t.Errorf("got %d tags, want %d", len(result.Tags), tt.wantCount)
			}
			if result.Limit != tt.wantLimit {
				t.Errorf("limit = %d, want %d", result.Limit, tt.wantLimit)
			}
			if result.Offset != tt.wantOffset {
				t.Errorf("offset = %d, want %d", result.Offset, tt.wantOffset)
			}

			// Verify tag structure
			for i, tag := range result.Tags {
				if tag.ID == uuid.Nil {
					t.Errorf("tag %d: ID should not be nil", i)
				}
				if tag.Name == "" {
					t.Errorf("tag %d: Name should not be empty", i)
				}
			}
		})
	}
}

func TestTagHandler_ListTags_ServiceError(t *testing.T) {
	mockRepo := &mockTagRepository{
		listFunc: func(ctx context.Context, limit, offset int) ([]domainTag.Tag, error) {
			return nil, fmt.Errorf("database error")
		},
	}

	tagService := newTestTagService(mockRepo)
	handler := NewTagHandler(tagService, nil, testAPIBasePath)

	ts := newTestServer(RouterConfig{
		TagHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/tags"))
	defer resp.Body.Close()

	assertErrorResponse(t, resp, http.StatusInternalServerError)
}

func TestTagHandler_ListTags_NilService(t *testing.T) {
	handler := NewTagHandler(nil, nil, testAPIBasePath)

	ts := newTestServer(RouterConfig{
		TagHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/tags"))
	defer resp.Body.Close()

	assertErrorResponse(t, resp, http.StatusInternalServerError)
}
