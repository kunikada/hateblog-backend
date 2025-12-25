package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	domainEntry "hateblog/internal/domain/entry"
	usecaseRanking "hateblog/internal/usecase/ranking"
)

func TestRankingHandler_Yearly(t *testing.T) {
	entry1 := newTestEntry(uuid.New(), "Popular Entry 2024", 500)
	entry2 := newTestEntry(uuid.New(), "Another Entry 2024", 300)

	tests := []struct {
		name           string
		queryParams    string
		mockResult     usecaseRanking.Result
		mockError      error
		wantStatus     int
		wantEntryCount int
		wantYear       int
		wantTotal      int64
	}{
		{
			name:        "success with default parameters",
			queryParams: "?year=2024",
			mockResult: usecaseRanking.Result{
				Entries: []*domainEntry.Entry{entry1, entry2},
				Total:   2,
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 2,
			wantYear:       2024,
			wantTotal:      2,
		},
		{
			name:        "success with custom limit",
			queryParams: "?year=2024&limit=10",
			mockResult: usecaseRanking.Result{
				Entries: []*domainEntry.Entry{entry1},
				Total:   1,
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
			wantYear:       2024,
			wantTotal:      1,
		},
		{
			name:        "success with min_users filter",
			queryParams: "?year=2024&min_users=100",
			mockResult: usecaseRanking.Result{
				Entries: []*domainEntry.Entry{entry1, entry2},
				Total:   2,
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 2,
			wantYear:       2024,
			wantTotal:      2,
		},
		{
			name:        "success with empty result",
			queryParams: "?year=2024",
			mockResult: usecaseRanking.Result{
				Entries: []*domainEntry.Entry{},
				Total:   0,
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 0,
			wantYear:       2024,
			wantTotal:      0,
		},
		{
			name:        "error: missing year parameter",
			queryParams: "",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: invalid year format",
			queryParams: "?year=abc",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: year too small",
			queryParams: "?year=1999",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: year too large",
			queryParams: "?year=10000",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: invalid limit",
			queryParams: "?year=2024&limit=abc",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: limit too small",
			queryParams: "?year=2024&limit=0",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: limit too large",
			queryParams: "?year=2024&limit=1001",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: invalid min_users",
			queryParams: "?year=2024&min_users=abc",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: negative min_users",
			queryParams: "?year=2024&min_users=-1",
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRankingRepository{
				result: tt.mockResult,
				err:    tt.mockError,
			}
			service := usecaseRanking.NewService(mockRepo, nil, nil, nil)
			handler := NewRankingHandler(service, testAPIBasePath)

			ts := newTestServer(RouterConfig{
				RankingHandler: handler,
			})
			defer ts.Close()

			resp := ts.get(t, apiPath("/rankings/yearly"+tt.queryParams))
			defer resp.Body.Close()

			if tt.wantStatus != http.StatusOK {
				assertErrorResponse(t, resp, tt.wantStatus)
				return
			}

			assertStatus(t, resp, http.StatusOK)
			assertContentType(t, resp, "application/json")

			var result rankingResponse
			decodeJSON(t, resp, &result)

			if result.PeriodType != "yearly" {
				t.Errorf("period_type = %q, want %q", result.PeriodType, "yearly")
			}
			if result.Year != tt.wantYear {
				t.Errorf("year = %d, want %d", result.Year, tt.wantYear)
			}
			if len(result.Entries) != tt.wantEntryCount {
				t.Errorf("got %d entries, want %d", len(result.Entries), tt.wantEntryCount)
			}
			if result.Total != tt.wantTotal {
				t.Errorf("total = %d, want %d", result.Total, tt.wantTotal)
			}

			// Verify ranking numbers
			for i, entry := range result.Entries {
				if entry.Rank != i+1 {
					t.Errorf("entry %d: rank = %d, want %d", i, entry.Rank, i+1)
				}
			}
		})
	}
}

func TestRankingHandler_Monthly(t *testing.T) {
	entry1 := newTestEntry(uuid.New(), "Popular Entry January", 500)
	entry2 := newTestEntry(uuid.New(), "Another Entry January", 300)

	tests := []struct {
		name           string
		queryParams    string
		mockResult     usecaseRanking.Result
		mockError      error
		wantStatus     int
		wantEntryCount int
		wantYear       int
		wantMonth      int
		wantTotal      int64
	}{
		{
			name:        "success with default parameters",
			queryParams: "?year=2025&month=1",
			mockResult: usecaseRanking.Result{
				Entries: []*domainEntry.Entry{entry1, entry2},
				Total:   2,
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 2,
			wantYear:       2025,
			wantMonth:      1,
			wantTotal:      2,
		},
		{
			name:        "success with December",
			queryParams: "?year=2025&month=12",
			mockResult: usecaseRanking.Result{
				Entries: []*domainEntry.Entry{entry1},
				Total:   1,
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
			wantYear:       2025,
			wantMonth:      12,
			wantTotal:      1,
		},
		{
			name:        "success with custom limit and min_users",
			queryParams: "?year=2025&month=6&limit=50&min_users=20",
			mockResult: usecaseRanking.Result{
				Entries: []*domainEntry.Entry{entry1, entry2},
				Total:   2,
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 2,
			wantYear:       2025,
			wantMonth:      6,
			wantTotal:      2,
		},
		{
			name:        "error: missing year parameter",
			queryParams: "?month=1",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: missing month parameter",
			queryParams: "?year=2025",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: invalid month format",
			queryParams: "?year=2025&month=abc",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: month too small",
			queryParams: "?year=2025&month=0",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: month too large",
			queryParams: "?year=2025&month=13",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: limit too large",
			queryParams: "?year=2025&month=1&limit=101",
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRankingRepository{
				result: tt.mockResult,
				err:    tt.mockError,
			}
			service := usecaseRanking.NewService(mockRepo, nil, nil, nil)
			handler := NewRankingHandler(service, testAPIBasePath)

			ts := newTestServer(RouterConfig{
				RankingHandler: handler,
			})
			defer ts.Close()

			resp := ts.get(t, apiPath("/rankings/monthly"+tt.queryParams))
			defer resp.Body.Close()

			if tt.wantStatus != http.StatusOK {
				assertErrorResponse(t, resp, tt.wantStatus)
				return
			}

			assertStatus(t, resp, http.StatusOK)
			assertContentType(t, resp, "application/json")

			var result rankingResponse
			decodeJSON(t, resp, &result)

			if result.PeriodType != "monthly" {
				t.Errorf("period_type = %q, want %q", result.PeriodType, "monthly")
			}
			if result.Year != tt.wantYear {
				t.Errorf("year = %d, want %d", result.Year, tt.wantYear)
			}
			if result.Month == nil || *result.Month != tt.wantMonth {
				t.Errorf("month = %v, want %d", result.Month, tt.wantMonth)
			}
			if len(result.Entries) != tt.wantEntryCount {
				t.Errorf("got %d entries, want %d", len(result.Entries), tt.wantEntryCount)
			}
			if result.Total != tt.wantTotal {
				t.Errorf("total = %d, want %d", result.Total, tt.wantTotal)
			}
		})
	}
}

func TestRankingHandler_Weekly(t *testing.T) {
	entry1 := newTestEntry(uuid.New(), "Popular Entry Week 1", 500)

	tests := []struct {
		name           string
		queryParams    string
		mockResult     usecaseRanking.Result
		mockError      error
		wantStatus     int
		wantEntryCount int
		wantYear       int
		wantWeek       int
		wantTotal      int64
	}{
		{
			name:        "success with week 1",
			queryParams: "?year=2025&week=1",
			mockResult: usecaseRanking.Result{
				Entries: []*domainEntry.Entry{entry1},
				Total:   1,
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
			wantYear:       2025,
			wantWeek:       1,
			wantTotal:      1,
		},
		{
			name:        "success with week 52",
			queryParams: "?year=2025&week=52",
			mockResult: usecaseRanking.Result{
				Entries: []*domainEntry.Entry{entry1},
				Total:   1,
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
			wantYear:       2025,
			wantWeek:       52,
			wantTotal:      1,
		},
		{
			name:        "success with custom limit",
			queryParams: "?year=2025&week=10&limit=20&min_users=50",
			mockResult: usecaseRanking.Result{
				Entries: []*domainEntry.Entry{entry1},
				Total:   1,
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
			wantYear:       2025,
			wantWeek:       10,
			wantTotal:      1,
		},
		{
			name:        "error: missing year parameter",
			queryParams: "?week=1",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: missing week parameter",
			queryParams: "?year=2025",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: week too small",
			queryParams: "?year=2025&week=0",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: week too large",
			queryParams: "?year=2025&week=54",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "error: invalid week format",
			queryParams: "?year=2025&week=abc",
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRankingRepository{
				result: tt.mockResult,
				err:    tt.mockError,
			}
			service := usecaseRanking.NewService(mockRepo, nil, nil, nil)
			handler := NewRankingHandler(service, testAPIBasePath)

			ts := newTestServer(RouterConfig{
				RankingHandler: handler,
			})
			defer ts.Close()

			resp := ts.get(t, apiPath("/rankings/weekly"+tt.queryParams))
			defer resp.Body.Close()

			if tt.wantStatus != http.StatusOK {
				assertErrorResponse(t, resp, tt.wantStatus)
				return
			}

			assertStatus(t, resp, http.StatusOK)
			assertContentType(t, resp, "application/json")

			var result rankingResponse
			decodeJSON(t, resp, &result)

			if result.PeriodType != "weekly" {
				t.Errorf("period_type = %q, want %q", result.PeriodType, "weekly")
			}
			if result.Year != tt.wantYear {
				t.Errorf("year = %d, want %d", result.Year, tt.wantYear)
			}
			if result.Week == nil || *result.Week != tt.wantWeek {
				t.Errorf("week = %v, want %d", result.Week, tt.wantWeek)
			}
			if len(result.Entries) != tt.wantEntryCount {
				t.Errorf("got %d entries, want %d", len(result.Entries), tt.wantEntryCount)
			}
			if result.Total != tt.wantTotal {
				t.Errorf("total = %d, want %d", result.Total, tt.wantTotal)
			}
		})
	}
}

// mockRankingRepository implements usecaseRanking.Repository for testing.
type mockRankingRepository struct {
	result usecaseRanking.Result
	err    error
}

func (m *mockRankingRepository) List(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result.Entries, nil
}

func (m *mockRankingRepository) Count(ctx context.Context, query domainEntry.ListQuery) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.result.Total, nil
}
