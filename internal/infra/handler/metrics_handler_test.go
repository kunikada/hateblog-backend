package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	domainEntry "hateblog/internal/domain/entry"
	usecaseMetrics "hateblog/internal/usecase/metrics"
)

func TestMetricsHandler_RecordClick(t *testing.T) {
	validEntryID := uuid.New()

	tests := []struct {
		name       string
		body       interface{}
		mockError  error
		wantStatus int
		wantMsg    string
	}{
		{
			name: "success with valid entry_id",
			body: clickMetricsRequest{
				EntryID: validEntryID,
			},
			wantStatus: http.StatusCreated,
			wantMsg:    "click recorded",
		},
		{
			name: "success with all fields",
			body: clickMetricsRequest{
				EntryID:   validEntryID,
				Referrer:  stringPtr("https://example.com"),
				UserAgent: stringPtr("Mozilla/5.0"),
			},
			wantStatus: http.StatusCreated,
			wantMsg:    "click recorded",
		},
		{
			name: "error: empty entry_id",
			body: clickMetricsRequest{
				EntryID: uuid.Nil,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "error: invalid JSON",
			body:       `{"entry_id": "invalid-uuid"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "error: empty body",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "error: malformed JSON",
			body:       `{invalid json`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEntryRepo := &mockEntryRepository{
				entries: []*domainEntry.Entry{newTestEntry(validEntryID, "Test Entry", 100)},
			}
			mockClickRepo := &mockClickMetricsRepository{
				err: tt.mockError,
			}
			service := usecaseMetrics.NewService(mockEntryRepo, mockClickRepo)
			handler := NewMetricsHandler(service)

			ts := newTestServer(RouterConfig{
				MetricsHandler: handler,
			})
			defer ts.Close()

			var body []byte
			var err error
			if str, ok := tt.body.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("failed to marshal body: %v", err)
				}
			}

			req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/metrics/clicks", bytes.NewReader(body))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("failed to send request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusCreated {
				var result metricsResponse
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if !result.Success {
					t.Errorf("success = false, want true")
				}
				if result.Message != tt.wantMsg {
					t.Errorf("message = %q, want %q", result.Message, tt.wantMsg)
				}
			}
		})
	}
}

func TestMetricsHandler_ServiceError(t *testing.T) {
	validEntryID := uuid.New()
	mockEntryRepo := &mockEntryRepository{
		entries: []*domainEntry.Entry{},
	}
	mockClickRepo := &mockClickMetricsRepository{
		err: fmt.Errorf("database error"),
	}
	service := usecaseMetrics.NewService(mockEntryRepo, mockClickRepo)
	handler := NewMetricsHandler(service)

	ts := newTestServer(RouterConfig{
		MetricsHandler: handler,
	})
	defer ts.Close()

	body, _ := json.Marshal(clickMetricsRequest{
		EntryID: validEntryID,
	})

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/metrics/clicks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestMetricsHandler_NilService(t *testing.T) {
	handler := NewMetricsHandler(nil)

	ts := newTestServer(RouterConfig{
		MetricsHandler: handler,
	})
	defer ts.Close()

	body, _ := json.Marshal(clickMetricsRequest{
		EntryID: uuid.New(),
	})

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/metrics/clicks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestMetricsHandler_ResponseFormat(t *testing.T) {
	entryID := uuid.New()
	mockEntryRepo := &mockEntryRepository{
		entries: []*domainEntry.Entry{newTestEntry(entryID, "Test Entry", 100)},
	}
	mockClickRepo := &mockClickMetricsRepository{}
	service := usecaseMetrics.NewService(mockEntryRepo, mockClickRepo)
	handler := NewMetricsHandler(service)

	router := NewRouter(RouterConfig{
		MetricsHandler: handler,
	})

	body, _ := json.Marshal(clickMetricsRequest{
		EntryID: entryID,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/clicks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var result metricsResponse
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !result.Success {
		t.Errorf("success = false, want true")
	}
	if result.Message != "click recorded" {
		t.Errorf("message = %q, want %q", result.Message, "click recorded")
	}
}

// mockClickMetricsRepository implements usecaseMetrics.ClickRepository for testing.
type mockClickMetricsRepository struct {
	err error
}

func (m *mockClickMetricsRepository) Increment(ctx context.Context, entryID domainEntry.ID, clickedAt time.Time) error {
	if m.err != nil {
		return m.err
	}
	return nil
}

func stringPtr(s string) *string {
	return &s
}
