package handler

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"hateblog/internal/domain/repository"
	usecaseArchive "hateblog/internal/usecase/archive"
)

func TestArchiveHandler_List(t *testing.T) {
	date1, _ := time.Parse("2006-01-02", "2025-01-15")
	date2, _ := time.Parse("2006-01-02", "2025-01-14")
	date3, _ := time.Parse("2006-01-02", "2025-01-13")

	tests := []struct {
		name        string
		queryParams string
		mockItems   []repository.ArchiveCount
		mockError   error
		wantStatus  int
		wantCount   int
	}{
		{
			name:        "success with default parameters",
			queryParams: "",
			mockItems: []repository.ArchiveCount{
				{Date: date1, Count: 150},
				{Date: date2, Count: 120},
				{Date: date3, Count: 100},
			},
			wantStatus: http.StatusOK,
			wantCount:  3,
		},
		{
			name:        "success with min_users filter",
			queryParams: "?min_users=10",
			mockItems: []repository.ArchiveCount{
				{Date: date1, Count: 50},
				{Date: date2, Count: 30},
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{
			name:        "success with empty result",
			queryParams: "",
			mockItems:   []repository.ArchiveCount{},
			wantStatus:  http.StatusOK,
			wantCount:   0,
		},
		{
			name:        "error: invalid min_users format",
			queryParams: "?min_users=abc",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: negative min_users",
			queryParams: "?min_users=-1",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: min_users not allowed",
			queryParams: "?min_users=7",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: min_users not allowed zero",
			queryParams: "?min_users=0",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: min_users too large",
			queryParams: "?min_users=10001",
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockArchiveRepository{
				items: tt.mockItems,
				err:   tt.mockError,
			}
			service := usecaseArchive.NewService(mockRepo, nil)
			handler := NewArchiveHandler(service)

			ts := newTestServer(RouterConfig{
				ArchiveHandler: handler,
			})
			defer ts.Close()

			resp := ts.get(t, apiPath("/archive"+tt.queryParams))
			defer resp.Body.Close()

			if tt.wantStatus != http.StatusOK {
				assertErrorResponse(t, resp, tt.wantStatus)
				return
			}

			assertStatus(t, resp, http.StatusOK)
			assertContentType(t, resp, "application/json")
			if got := resp.Header.Get(cacheStatusHeader); got != cacheStatusMiss {
				t.Errorf("cache header = %q, want %q", got, cacheStatusMiss)
			}

			var result archiveResponse
			decodeJSON(t, resp, &result)

			if len(result.Items) != tt.wantCount {
				t.Errorf("got %d items, want %d", len(result.Items), tt.wantCount)
			}

			// Verify date format
			for i, item := range result.Items {
				if item.Date == "" {
					t.Errorf("item %d: date is empty", i)
				}
				if _, err := time.Parse("2006-01-02", item.Date); err != nil {
					t.Errorf("item %d: invalid date format %q: %v", i, item.Date, err)
				}
				if item.Count < 0 {
					t.Errorf("item %d: count = %d, should be non-negative", i, item.Count)
				}
			}
		})
	}
}

func TestArchiveHandler_ServiceError(t *testing.T) {
	mockRepo := &mockArchiveRepository{
		err: fmt.Errorf("database error"),
	}
	service := usecaseArchive.NewService(mockRepo, nil)
	handler := NewArchiveHandler(service)

	ts := newTestServer(RouterConfig{
		ArchiveHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/archive"))
	defer resp.Body.Close()

	assertErrorResponse(t, resp, http.StatusInternalServerError)
}

func TestArchiveHandler_ResponseFormat(t *testing.T) {
	date1, _ := time.Parse("2006-01-02", "2025-12-20")
	mockRepo := &mockArchiveRepository{
		items: []repository.ArchiveCount{
			{Date: date1, Count: 100},
		},
	}
	service := usecaseArchive.NewService(mockRepo, nil)
	handler := NewArchiveHandler(service)

	ts := newTestServer(RouterConfig{
		ArchiveHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/archive"))
	defer resp.Body.Close()

	assertStatus(t, resp, http.StatusOK)
	if got := resp.Header.Get(cacheStatusHeader); got != cacheStatusMiss {
		t.Errorf("cache header = %q, want %q", got, cacheStatusMiss)
	}

	var result archiveResponse
	decodeJSON(t, resp, &result)

	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}

	item := result.Items[0]
	if item.Date != "2025-12-20" {
		t.Errorf("date = %q, want %q", item.Date, "2025-12-20")
	}
	if item.Count != 100 {
		t.Errorf("count = %d, want %d", item.Count, 100)
	}
}

// mockArchiveRepository implements usecaseArchive.Repository for testing.
type mockArchiveRepository struct {
	items []repository.ArchiveCount
	err   error
}

func (m *mockArchiveRepository) ListArchiveCounts(ctx context.Context, minBookmarkCount int) ([]repository.ArchiveCount, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.items, nil
}
