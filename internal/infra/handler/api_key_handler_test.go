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

	"hateblog/internal/domain/api_key"
	usecaseAPIKey "hateblog/internal/usecase/api_key"
)

func TestAPIKeyHandler_CreateAPIKey(t *testing.T) {
	futureTime := time.Now().Add(24 * time.Hour)
	pastTime := time.Now().Add(-1 * time.Hour)
	longName := string(make([]byte, 101))
	longDesc := string(make([]byte, 501))

	tests := []struct {
		name       string
		body       interface{}
		mockError  error
		wantStatus int
		checkResp  bool
	}{
		{
			name:       "success with minimal fields",
			body:       createAPIKeyRequest{},
			wantStatus: http.StatusCreated,
			checkResp:  true,
		},
		{
			name: "success with name",
			body: createAPIKeyRequest{
				Name: stringPtr("Test API Key"),
			},
			wantStatus: http.StatusCreated,
			checkResp:  true,
		},
		{
			name: "success with description",
			body: createAPIKeyRequest{
				Description: stringPtr("Test API key for development"),
			},
			wantStatus: http.StatusCreated,
			checkResp:  true,
		},
		{
			name: "success with all fields",
			body: createAPIKeyRequest{
				Name:        stringPtr("Production API Key"),
				Description: stringPtr("API key for production environment"),
				ExpiresAt:   &futureTime,
			},
			wantStatus: http.StatusCreated,
			checkResp:  true,
		},
		{
			name: "error: name too long",
			body: createAPIKeyRequest{
				Name: &longName,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "error: description too long",
			body: createAPIKeyRequest{
				Description: &longDesc,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "error: expires_at in the past",
			body: createAPIKeyRequest{
				ExpiresAt: &pastTime,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "error: invalid JSON",
			body:       `{"name": 123}`,
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
			mockRepo := &mockAPIKeyRepository{
				err: tt.mockError,
			}
			service := usecaseAPIKey.NewService(mockRepo, "test_")
			handler := NewAPIKeyHandler(service)

			ts := newTestServer(RouterConfig{
				APIKeyHandler: handler,
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

			req, err := http.NewRequest(http.MethodPost, ts.URL+apiPath("/api-keys"), bytes.NewReader(body))
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

			if tt.checkResp && resp.StatusCode == http.StatusCreated {
				var result apiKeyResponse
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				// Verify ID is valid UUID
				if result.ID == uuid.Nil {
					t.Error("ID should not be nil UUID")
				}

				// Verify key has correct format
				if result.Key == "" {
					t.Error("key should not be empty")
				}
				if len(result.Key) < 10 {
					t.Errorf("key length = %d, should be at least 10", len(result.Key))
				}

				// Verify created_at is set
				if result.CreatedAt.IsZero() {
					t.Error("created_at should be set")
				}

				// Verify optional fields
				req := tt.body.(createAPIKeyRequest)
				if req.Name != nil && result.Name != nil {
					if *result.Name != *req.Name {
						t.Errorf("name = %q, want %q", *result.Name, *req.Name)
					}
				}
				if req.Description != nil && result.Description != nil {
					if *result.Description != *req.Description {
						t.Errorf("description = %q, want %q", *result.Description, *req.Description)
					}
				}
				if req.ExpiresAt != nil && result.ExpiresAt != nil {
					// Allow small time difference due to rounding
					diff := req.ExpiresAt.Sub(*result.ExpiresAt)
					if diff < 0 {
						diff = -diff
					}
					if diff > time.Second {
						t.Errorf("expires_at difference too large: %v", diff)
					}
				}
			}
		})
	}
}

func TestAPIKeyHandler_NilService(t *testing.T) {
	handler := NewAPIKeyHandler(nil)

	ts := newTestServer(RouterConfig{
		APIKeyHandler: handler,
	})
	defer ts.Close()

	body, _ := json.Marshal(createAPIKeyRequest{
		Name: stringPtr("Test"),
	})

	req, _ := http.NewRequest(http.MethodPost, ts.URL+apiPath("/api-keys"), bytes.NewReader(body))
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

func TestAPIKeyHandler_RepositoryError(t *testing.T) {
	mockRepo := &mockAPIKeyRepository{
		err: fmt.Errorf("redis connection error"),
	}
	service := usecaseAPIKey.NewService(mockRepo, "test_")
	handler := NewAPIKeyHandler(service)

	ts := newTestServer(RouterConfig{
		APIKeyHandler: handler,
	})
	defer ts.Close()

	body, _ := json.Marshal(createAPIKeyRequest{
		Name: stringPtr("Test"),
	})

	req, _ := http.NewRequest(http.MethodPost, ts.URL+apiPath("/api-keys"), bytes.NewReader(body))
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

func TestAPIKeyHandler_ResponseFormat(t *testing.T) {
	mockRepo := &mockAPIKeyRepository{}
	service := usecaseAPIKey.NewService(mockRepo, "hb_live_")
	handler := NewAPIKeyHandler(service)

	router := NewRouter(RouterConfig{
		APIKeyHandler: handler,
		APIBasePath:   testAPIBasePath,
	})

	name := "Test Key"
	desc := "Test description"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	body, _ := json.Marshal(createAPIKeyRequest{
		Name:        &name,
		Description: &desc,
		ExpiresAt:   &expiresAt,
	})

	req := httptest.NewRequest(http.MethodPost, apiPath("/api-keys"), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	// Verify Content-Type
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
	}

	var result apiKeyResponse
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify all fields are present
	if result.ID == uuid.Nil {
		t.Error("ID should not be nil UUID")
	}

	if result.Key == "" {
		t.Error("key should not be empty")
	}

	// Key should have prefix
	if len(result.Key) < len("hb_live_") || result.Key[:8] != "hb_live_" {
		t.Errorf("key should start with 'hb_live_', got %q", result.Key)
	}

	if result.Name == nil || *result.Name != name {
		t.Errorf("name = %v, want %q", result.Name, name)
	}

	if result.Description == nil || *result.Description != desc {
		t.Errorf("description = %v, want %q", result.Description, desc)
	}

	if result.CreatedAt.IsZero() {
		t.Error("created_at should be set")
	}

	if result.ExpiresAt == nil {
		t.Error("expires_at should be set")
	}
}

func TestAPIKeyHandler_MetadataExtraction(t *testing.T) {
	mockRepo := &mockAPIKeyRepository{}
	service := usecaseAPIKey.NewService(mockRepo, "test_")
	handler := NewAPIKeyHandler(service)

	router := NewRouter(RouterConfig{
		APIKeyHandler: handler,
		APIBasePath:   testAPIBasePath,
	})

	body, _ := json.Marshal(createAPIKeyRequest{})

	req := httptest.NewRequest(http.MethodPost, apiPath("/api-keys"), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Test-Agent/1.0")
	req.Header.Set("Referer", "https://example.com")
	req.RemoteAddr = "192.168.1.100:12345"
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	// Verify that metadata was extracted (we can't directly check it in the response,
	// but we can verify the handler didn't fail)
	if mockRepo.storedKey == nil {
		t.Fatal("API key should have been stored")
	}

	// Verify metadata was captured
	if mockRepo.storedKey.CreatedIP == nil {
		t.Error("CreatedIP should be set")
	}
	if mockRepo.storedKey.CreatedUserAgent == nil {
		t.Error("CreatedUserAgent should be set")
	}
	if mockRepo.storedKey.CreatedReferrer == nil {
		t.Error("CreatedReferrer should be set")
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		want       string
	}{
		{
			name:       "IPv4 with port",
			remoteAddr: "192.168.1.100:12345",
			want:       "192.168.1.100",
		},
		{
			name:       "IPv6 with port",
			remoteAddr: "[2001:db8::1]:8080",
			want:       "[2001:db8::1]",
		},
		{
			name:       "localhost with port",
			remoteAddr: "127.0.0.1:54321",
			want:       "127.0.0.1",
		},
		{
			name:       "no port",
			remoteAddr: "192.168.1.100",
			want:       "192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractIP(tt.remoteAddr)
			if got != tt.want {
				t.Errorf("extractIP(%q) = %q, want %q", tt.remoteAddr, got, tt.want)
			}
		})
	}
}

// mockAPIKeyRepository implements repository.APIKeyRepository for testing.
type mockAPIKeyRepository struct {
	storedKey *api_key.APIKey
	err       error
}

func (m *mockAPIKeyRepository) Store(ctx context.Context, k *api_key.APIKey) error {
	if m.err != nil {
		return m.err
	}
	m.storedKey = k
	return nil
}

func (m *mockAPIKeyRepository) GetByID(ctx context.Context, id api_key.ID) (*api_key.APIKey, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.storedKey != nil && m.storedKey.ID == id {
		return m.storedKey, nil
	}
	return nil, fmt.Errorf("api key not found")
}
