package handler

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"

	domainEntry "hateblog/internal/domain/entry"
	usecaseEntry "hateblog/internal/usecase/entry"
)

func TestEntryHandler_NewEntries(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockResult     usecaseEntry.ListResult
		mockError      error
		wantStatus     int
		wantEntryCount int
		wantTotal      int64
		wantLimit      int
		wantOffset     int
	}{
		{
			name:        "success with default parameters",
			queryParams: "?date=20240101",
			mockResult: buildTestListResult([]*domainEntry.Entry{
				newTestEntry(uuid.New(), "Entry 1", 100),
				newTestEntry(uuid.New(), "Entry 2", 50),
			}, 2),
			wantStatus:     http.StatusOK,
			wantEntryCount: 2,
			wantTotal:      2,
			wantLimit:      defaultLimit,
			wantOffset:     0,
		},
		{
			name:        "success with custom limit and offset",
			queryParams: "?date=20240101&limit=10&offset=5&min_users=0",
			mockResult: buildTestListResult([]*domainEntry.Entry{
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
				newTestEntry(uuid.New(), "Entry 16", 1),
			}, 16),
			wantStatus:     http.StatusOK,
			wantEntryCount: 10,
			wantTotal:      16,
			wantLimit:      10,
			wantOffset:     5,
		},
		{
			name:        "success with min_users filter",
			queryParams: "?date=20240101&min_users=50",
			mockResult: buildTestListResult([]*domainEntry.Entry{
				newTestEntry(uuid.New(), "High bookmarks", 100),
			}, 1),
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
			wantTotal:      1,
			wantLimit:      defaultLimit,
			wantOffset:     0,
		},
		{
			name:        "success with empty result",
			queryParams: "?date=20240101",
			mockResult:  buildTestListResult([]*domainEntry.Entry{}, 0),
			wantStatus:  http.StatusOK,
			wantEntryCount: 0,
			wantTotal:      0,
			wantLimit:      defaultLimit,
			wantOffset:     0,
		},
		{
			name:        "error: missing date parameter",
			queryParams: "",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: invalid date format",
			queryParams: "?date=2024-01-01",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: date too short",
			queryParams: "?date=202401",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: date not a valid date",
			queryParams: "?date=20241301",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: invalid limit",
			queryParams: "?date=20240101&limit=abc",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: limit too small",
			queryParams: "?date=20240101&limit=0",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: limit too large",
			queryParams: fmt.Sprintf("?date=20240101&limit=%d", domainEntry.MaxLimit+1),
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: invalid offset",
			queryParams: "?date=20240101&offset=xyz",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: negative offset",
			queryParams: "?date=20240101&offset=-1",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: invalid min_users",
			queryParams: "?date=20240101&min_users=abc",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: negative min_users",
			queryParams: "?date=20240101&min_users=-1",
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockEntryRepository{
				entries: tt.mockResult.Entries,
				total:   tt.mockResult.Total,
			}
			if tt.mockError != nil {
				mockRepo.listFunc = func(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, error) {
					return nil, tt.mockError
				}
			}

			service := newTestEntryService(mockRepo)
			handler := NewEntryHandler(service, testAPIBasePath)
			ts := newTestServer(RouterConfig{
				EntryHandler: handler,
			})
			defer ts.Close()

			resp := ts.get(t, apiPath("/entries/new"+tt.queryParams))
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
				if entry.FaviconURL == "" {
					t.Error("entry FaviconURL should not be empty")
				}
				if len(entry.Tags) == 0 {
					t.Error("entry should have tags")
				}
			}
		})
	}
}

func TestEntryHandler_NewEntries_ServiceError(t *testing.T) {
	mockRepo := &mockEntryRepository{
		listFunc: func(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, error) {
			return nil, fmt.Errorf("database error")
		},
	}

	service := newTestEntryService(mockRepo)
	handler := NewEntryHandler(service, testAPIBasePath)
	ts := newTestServer(RouterConfig{
		EntryHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/entries/new?date=20240101"))
	defer resp.Body.Close()

	errResp := assertErrorResponse(t, resp, http.StatusInternalServerError)
	if errResp["error"] != "internal error" {
		t.Errorf("error message = %q, want %q", errResp["error"], "internal error")
	}
}

func TestEntryHandler_HotEntries(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockResult     usecaseEntry.ListResult
		mockError      error
		wantStatus     int
		wantEntryCount int
		wantTotal      int64
		wantLimit      int
		wantOffset     int
	}{
		{
			name:        "success with default parameters",
			queryParams: "?date=20240101",
			mockResult: buildTestListResult([]*domainEntry.Entry{
				newTestEntry(uuid.New(), "Hot Entry 1", 1000),
				newTestEntry(uuid.New(), "Hot Entry 2", 500),
			}, 2),
			wantStatus:     http.StatusOK,
			wantEntryCount: 2,
			wantTotal:      2,
			wantLimit:      defaultLimit,
			wantOffset:     0,
		},
		{
			name:        "success with custom limit",
			queryParams: "?date=20240101&limit=50",
			mockResult: buildTestListResult([]*domainEntry.Entry{
				newTestEntry(uuid.New(), "Hot Entry", 1000),
			}, 1),
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
			wantTotal:      1,
			wantLimit:      50,
			wantOffset:     0,
		},
		{
			name:        "success with pagination",
			queryParams: "?date=20240101&limit=20&offset=40&min_users=0",
			mockResult: func() usecaseEntry.ListResult {
				entries := make([]*domainEntry.Entry, 50)
				for i := 0; i < 50; i++ {
					entries[i] = newTestEntry(uuid.New(), fmt.Sprintf("Entry %d", i+1), 1000-i*10)
				}
				return buildTestListResult(entries, 50)
			}(),
			wantStatus:     http.StatusOK,
			wantEntryCount: 10,
			wantTotal:      50,
			wantLimit:      20,
			wantOffset:     40,
		},
		{
			name:        "error: missing date",
			queryParams: "",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: invalid date",
			queryParams: "?date=invalid",
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockEntryRepository{
				entries: tt.mockResult.Entries,
				total:   tt.mockResult.Total,
			}
			if tt.mockError != nil {
				mockRepo.listFunc = func(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, error) {
					return nil, tt.mockError
				}
			}

			service := newTestEntryService(mockRepo)
			handler := NewEntryHandler(service, testAPIBasePath)
			ts := newTestServer(RouterConfig{
				EntryHandler: handler,
			})
			defer ts.Close()

			resp := ts.get(t, apiPath("/entries/hot"+tt.queryParams))
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

func TestEntryHandler_HotEntries_ServiceError(t *testing.T) {
	mockRepo := &mockEntryRepository{
		listFunc: func(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, error) {
			return nil, fmt.Errorf("cache error")
		},
	}

	service := newTestEntryService(mockRepo)
	handler := NewEntryHandler(service, testAPIBasePath)
	ts := newTestServer(RouterConfig{
		EntryHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/entries/hot?date=20240101"))
	defer resp.Body.Close()

	errResp := assertErrorResponse(t, resp, http.StatusInternalServerError)
	if errResp["error"] != "internal error" {
		t.Errorf("error message = %q, want %q", errResp["error"], "internal error")
	}
}

func TestEntryHandler_ResponseFormat(t *testing.T) {
	entryID := uuid.New()
	tagID := uuid.New()

	excerpt := "This is a test excerpt"
	subject := "technology"

	entry := newTestEntry(entryID, "Test Entry", 100)
	entry.Excerpt = excerpt
	entry.Subject = subject
	entry.Tags = []domainEntry.Tagging{
		newTestTagging(tagID, "tech", 90),
		newTestTagging(uuid.New(), "programming", 80),
	}

	mockRepo := &mockEntryRepository{
		entries: []*domainEntry.Entry{entry},
		total:   1,
	}

	service := newTestEntryService(mockRepo)
	handler := NewEntryHandler(service, testAPIBasePath)
	ts := newTestServer(RouterConfig{
		EntryHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/entries/new?date=20240101"))
	defer resp.Body.Close()

	result := assertEntryListResponse(t, resp)
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Entries))
	}

	got := result.Entries[0]
	if got.ID != entryID {
		t.Errorf("ID = %v, want %v", got.ID, entryID)
	}
	if got.Title != "Test Entry" {
		t.Errorf("Title = %q, want %q", got.Title, "Test Entry")
	}
	if got.BookmarkCount != 100 {
		t.Errorf("BookmarkCount = %d, want %d", got.BookmarkCount, 100)
	}
	if got.Excerpt == nil || *got.Excerpt != excerpt {
		t.Errorf("Excerpt = %v, want %q", got.Excerpt, excerpt)
	}
	if got.Subject == nil || *got.Subject != subject {
		t.Errorf("Subject = %v, want %q", got.Subject, subject)
	}
	if len(got.Tags) != 2 {
		t.Fatalf("Tags count = %d, want 2", len(got.Tags))
	}
	if got.Tags[0].TagID != tagID {
		t.Errorf("Tags[0].TagID = %v, want %v", got.Tags[0].TagID, tagID)
	}
	if got.Tags[0].Name != "tech" {
		t.Errorf("Tags[0].Name = %q, want %q", got.Tags[0].Name, "tech")
	}
	if got.Tags[0].Score != 90 {
		t.Errorf("Tags[0].Score = %d, want %d", got.Tags[0].Score, 90)
	}
	if got.FaviconURL == "" {
		t.Error("FaviconURL should not be empty")
	}
}

func TestEntryHandler_BoundaryValues(t *testing.T) {
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
			limit:      domainEntry.MaxLimit,
			wantStatus: http.StatusOK,
		},
		{
			name:       "limit below minimum",
			limit:      0,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "limit above maximum",
			limit:      domainEntry.MaxLimit + 1,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockEntryRepository{
				entries: []*domainEntry.Entry{},
				total:   0,
			}

			service := newTestEntryService(mockRepo)
			handler := NewEntryHandler(service, testAPIBasePath)
			ts := newTestServer(RouterConfig{
				EntryHandler: handler,
			})
			defer ts.Close()

			path := apiPath(fmt.Sprintf("/entries/new?date=20240101&limit=%d", tt.limit))
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
