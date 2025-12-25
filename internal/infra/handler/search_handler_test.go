package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	domainEntry "hateblog/internal/domain/entry"
)

func TestSearchHandler_SearchEntries(t *testing.T) {
	entry1 := newTestEntry(uuid.New(), "Go Programming Tutorial", 100)
	entry2 := newTestEntry(uuid.New(), "Advanced Go Patterns", 75)

	tests := []struct {
		name           string
		queryParams    string
		mockEntries    []*domainEntry.Entry
		mockTotal      int64
		mockError      error
		wantStatus     int
		wantQuery      string
		wantEntryCount int
		wantTotal      int64
		wantLimit      int
		wantOffset     int
	}{
		{
			name:           "success with default parameters",
			queryParams:    "?q=golang",
			mockEntries:    []*domainEntry.Entry{entry1, entry2},
			mockTotal:      2,
			wantStatus:     http.StatusOK,
			wantQuery:      "golang",
			wantEntryCount: 2,
			wantTotal:      2,
			wantLimit:      defaultLimit,
			wantOffset:     0,
		},
		{
			name:           "success with custom limit and offset",
			queryParams:    "?q=programming&limit=10&offset=5",
			mockEntries:    []*domainEntry.Entry{entry1},
			mockTotal:      100,
			wantStatus:     http.StatusOK,
			wantQuery:      "programming",
			wantEntryCount: 1,
			wantTotal:      100,
			wantLimit:      10,
			wantOffset:     5,
		},
		{
			name:           "success with min_users filter",
			queryParams:    "?q=tutorial&min_users=50",
			mockEntries:    []*domainEntry.Entry{entry1},
			mockTotal:      1,
			wantStatus:     http.StatusOK,
			wantQuery:      "tutorial",
			wantEntryCount: 1,
			wantTotal:      1,
			wantLimit:      defaultLimit,
			wantOffset:     0,
		},
		{
			name:           "success with empty result",
			queryParams:    "?q=nonexistent",
			mockEntries:    []*domainEntry.Entry{},
			mockTotal:      0,
			wantStatus:     http.StatusOK,
			wantQuery:      "nonexistent",
			wantEntryCount: 0,
			wantTotal:      0,
			wantLimit:      defaultLimit,
			wantOffset:     0,
		},
		{
			name:           "success with trimmed query",
			queryParams:    "?q=%20%20golang%20%20",
			mockEntries:    []*domainEntry.Entry{entry1},
			mockTotal:      1,
			wantStatus:     http.StatusOK,
			wantQuery:      "golang",
			wantEntryCount: 1,
			wantTotal:      1,
			wantLimit:      defaultLimit,
			wantOffset:     0,
		},
		{
			name:        "error: empty query",
			queryParams: "?q=",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: whitespace only query",
			queryParams: "?q=%20%20%20",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: missing query parameter",
			queryParams: "",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: query too long",
			queryParams: fmt.Sprintf("?q=%s", strings.Repeat("a", 501)),
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: invalid limit",
			queryParams: "?q=test&limit=abc",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: limit too small",
			queryParams: "?q=test&limit=0",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: limit too large",
			queryParams: fmt.Sprintf("?q=test&limit=%d", domainEntry.MaxLimit+1),
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: invalid offset",
			queryParams: "?q=test&offset=xyz",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: negative offset",
			queryParams: "?q=test&offset=-1",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: invalid min_users",
			queryParams: "?q=test&min_users=abc",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: negative min_users",
			queryParams: "?q=test&min_users=-1",
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEntryRepo := &mockEntryRepository{
				entries: tt.mockEntries,
				total:   tt.mockTotal,
			}
			if tt.mockError != nil {
				mockEntryRepo.listFunc = func(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, error) {
					return nil, tt.mockError
				}
			}

			mockHistoryRepo := &mockSearchHistoryRepository{}
			service := newTestSearchService(mockEntryRepo, mockHistoryRepo)
			handler := NewSearchHandler(service, testAPIBasePath)

			ts := newTestServer(RouterConfig{
				SearchHandler: handler,
			})
			defer ts.Close()

			resp := ts.get(t, apiPath("/search"+tt.queryParams))
			defer resp.Body.Close()

			if tt.wantStatus != http.StatusOK {
				assertErrorResponse(t, resp, tt.wantStatus)
				return
			}

			assertStatus(t, resp, http.StatusOK)
			assertContentType(t, resp, "application/json")

			var result searchResponse
			decodeJSON(t, resp, &result)

			if result.Query != tt.wantQuery {
				t.Errorf("query = %q, want %q", result.Query, tt.wantQuery)
			}
			if len(result.Entries) != tt.wantEntryCount {
				t.Errorf("got %d entries, want %d", len(result.Entries), tt.wantEntryCount)
			}
			assertPagination(t, result.Total, result.Limit, result.Offset,
				tt.wantTotal, tt.wantLimit, tt.wantOffset)

			// Verify entry structure for non-empty results
			if len(result.Entries) > 0 {
				entry := result.Entries[0]
				if entry.ID == uuid.Nil {
					t.Error("entry ID should not be nil")
				}
				if entry.Title == "" {
					t.Error("entry title should not be empty")
				}
				if entry.URL == "" {
					t.Error("entry URL should not be empty")
				}
			}
		})
	}
}

func TestSearchHandler_SearchEntries_ServiceError(t *testing.T) {
	mockEntryRepo := &mockEntryRepository{
		listFunc: func(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, error) {
			return nil, fmt.Errorf("database error")
		},
	}

	mockHistoryRepo := &mockSearchHistoryRepository{}
	service := newTestSearchService(mockEntryRepo, mockHistoryRepo)
	handler := NewSearchHandler(service, testAPIBasePath)

	ts := newTestServer(RouterConfig{
		SearchHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/search?q=golang"))
	defer resp.Body.Close()

	errResp := assertErrorResponse(t, resp, http.StatusBadRequest)
	// Search service errors are returned as bad request
	if errResp["error"] == "" {
		t.Error("error message should not be empty")
	}
}

func TestSearchHandler_SearchEntries_RecordHistory(t *testing.T) {
	historyRecorded := false
	recordedQuery := ""

	mockEntryRepo := &mockEntryRepository{
		entries: []*domainEntry.Entry{newTestEntry(uuid.New(), "Entry", 100)},
		total:   1,
	}

	mockHistoryRepo := &mockSearchHistoryRepository{
		recordFunc: func(ctx context.Context, query string, searchedAt time.Time) error {
			historyRecorded = true
			recordedQuery = query
			return nil
		},
	}

	service := newTestSearchService(mockEntryRepo, mockHistoryRepo)
	handler := NewSearchHandler(service, testAPIBasePath)

	ts := newTestServer(RouterConfig{
		SearchHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/search?q=golang"))
	defer resp.Body.Close()

	assertStatus(t, resp, http.StatusOK)

	if !historyRecorded {
		t.Error("search history should be recorded")
	}
	if recordedQuery != "golang" {
		t.Errorf("recorded query = %q, want %q", recordedQuery, "golang")
	}
}

func TestSearchHandler_SearchEntries_HistoryError(t *testing.T) {
	mockEntryRepo := &mockEntryRepository{
		entries: []*domainEntry.Entry{newTestEntry(uuid.New(), "Entry", 100)},
		total:   1,
	}

	mockHistoryRepo := &mockSearchHistoryRepository{
		recordFunc: func(ctx context.Context, query string, searchedAt time.Time) error {
			return fmt.Errorf("failed to record history")
		},
	}

	service := newTestSearchService(mockEntryRepo, mockHistoryRepo)
	handler := NewSearchHandler(service, testAPIBasePath)

	ts := newTestServer(RouterConfig{
		SearchHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/search?q=golang"))
	defer resp.Body.Close()

	// Should still return 200 even if recording history fails
	assertStatus(t, resp, http.StatusOK)
}

func TestSearchHandler_ResponseFormat(t *testing.T) {
	entryID := uuid.New()
	tagID := uuid.New()

	entry := newTestEntry(entryID, "Go Programming", 100)
	entry.Tags = []domainEntry.Tagging{
		newTestTagging(tagID, "golang", 0.9),
	}

	mockEntryRepo := &mockEntryRepository{
		entries: []*domainEntry.Entry{entry},
		total:   1,
	}

	mockHistoryRepo := &mockSearchHistoryRepository{}
	service := newTestSearchService(mockEntryRepo, mockHistoryRepo)
	handler := NewSearchHandler(service, testAPIBasePath)

	ts := newTestServer(RouterConfig{
		SearchHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/search?q=golang"))
	defer resp.Body.Close()

	assertStatus(t, resp, http.StatusOK)

	var result searchResponse
	decodeJSON(t, resp, &result)

	if result.Query != "golang" {
		t.Errorf("query = %q, want %q", result.Query, "golang")
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Entries))
	}

	got := result.Entries[0]
	if got.ID != entryID {
		t.Errorf("ID = %v, want %v", got.ID, entryID)
	}
	if got.Title != "Go Programming" {
		t.Errorf("Title = %q, want %q", got.Title, "Go Programming")
	}
	if got.BookmarkCount != 100 {
		t.Errorf("BookmarkCount = %d, want %d", got.BookmarkCount, 100)
	}
	if len(got.Tags) != 1 {
		t.Fatalf("Tags count = %d, want 1", len(got.Tags))
	}
	if got.Tags[0].TagID != tagID {
		t.Errorf("Tags[0].TagID = %v, want %v", got.Tags[0].TagID, tagID)
	}
	if got.Tags[0].Name != "golang" {
		t.Errorf("Tags[0].Name = %q, want %q", got.Tags[0].Name, "golang")
	}
}

func TestSearchHandler_BoundaryValues(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		limit      int
		wantStatus int
	}{
		{
			name:       "minimum valid query length",
			query:      "a",
			limit:      defaultLimit,
			wantStatus: http.StatusOK,
		},
		{
			name:       "maximum valid query length",
			query:      strings.Repeat("a", 500),
			limit:      defaultLimit,
			wantStatus: http.StatusOK,
		},
		{
			name:       "query exceeds max length",
			query:      strings.Repeat("a", 501),
			limit:      defaultLimit,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "minimum valid limit",
			query:      "test",
			limit:      1,
			wantStatus: http.StatusOK,
		},
		{
			name:       "maximum valid limit",
			query:      "test",
			limit:      domainEntry.MaxLimit,
			wantStatus: http.StatusOK,
		},
		{
			name:       "limit below minimum",
			query:      "test",
			limit:      0,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "limit above maximum",
			query:      "test",
			limit:      domainEntry.MaxLimit + 1,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEntryRepo := &mockEntryRepository{
				entries: []*domainEntry.Entry{},
				total:   0,
			}

			mockHistoryRepo := &mockSearchHistoryRepository{}
			service := newTestSearchService(mockEntryRepo, mockHistoryRepo)
			handler := NewSearchHandler(service, testAPIBasePath)

			ts := newTestServer(RouterConfig{
				SearchHandler: handler,
			})
			defer ts.Close()

			path := apiPath(fmt.Sprintf("/search?q=%s&limit=%d", tt.query, tt.limit))
			resp := ts.get(t, path)
			defer resp.Body.Close()

			if tt.wantStatus == http.StatusOK {
				assertStatus(t, resp, http.StatusOK)
			} else {
				assertErrorResponse(t, resp, tt.wantStatus)
			}
		})
	}
}

func TestSearchHandler_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{
			name:       "query with spaces",
			query:      "hello world",
			wantStatus: http.StatusOK,
		},
		{
			name:       "query with special characters",
			query:      "go@2.0",
			wantStatus: http.StatusOK,
		},
		{
			name:       "query with unicode",
			query:      "日本語",
			wantStatus: http.StatusOK,
		},
		{
			name:       "query with symbols",
			query:      "c++",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEntryRepo := &mockEntryRepository{
				entries: []*domainEntry.Entry{},
				total:   0,
			}

			mockHistoryRepo := &mockSearchHistoryRepository{}
			service := newTestSearchService(mockEntryRepo, mockHistoryRepo)
			handler := NewSearchHandler(service, testAPIBasePath)

			ts := newTestServer(RouterConfig{
				SearchHandler: handler,
			})
			defer ts.Close()

			path := apiPath(fmt.Sprintf("/search?q=%s", url.QueryEscape(tt.query)))
			resp := ts.get(t, path)
			defer resp.Body.Close()

			assertStatus(t, resp, tt.wantStatus)
		})
	}
}
