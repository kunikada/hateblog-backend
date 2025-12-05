package server

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestLogger(t *testing.T) {
	logger := slog.Default()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	})

	middleware := RequestLogger(logger)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "test", rec.Body.String())
}

func TestRecoverer(t *testing.T) {
	logger := slog.Default()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	middleware := Recoverer(logger)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Should not panic
	assert.NotPanics(t, func() {
		wrappedHandler.ServeHTTP(rec, req)
	})

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestCORS(t *testing.T) {
	tests := []struct {
		name            string
		allowedOrigins  []string
		requestOrigin   string
		method          string
		expectAllowCORS bool
	}{
		{
			name:            "wildcard allows any origin",
			allowedOrigins:  []string{"*"},
			requestOrigin:   "https://example.com",
			method:          http.MethodGet,
			expectAllowCORS: true,
		},
		{
			name:            "specific origin allowed",
			allowedOrigins:  []string{"https://example.com"},
			requestOrigin:   "https://example.com",
			method:          http.MethodGet,
			expectAllowCORS: true,
		},
		{
			name:            "origin not allowed",
			allowedOrigins:  []string{"https://example.com"},
			requestOrigin:   "https://evil.com",
			method:          http.MethodGet,
			expectAllowCORS: false,
		},
		{
			name:            "preflight request",
			allowedOrigins:  []string{"*"},
			requestOrigin:   "https://example.com",
			method:          http.MethodOptions,
			expectAllowCORS: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := CORS(tt.allowedOrigins)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			if tt.method == http.MethodOptions {
				assert.Equal(t, http.StatusNoContent, rec.Code)
			}

			if tt.expectAllowCORS {
				assert.NotEmpty(t, rec.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

func TestAPIKeyAuth(t *testing.T) {
	logger := slog.Default()
	validAPIKey := "test-api-key"

	tests := []struct {
		name           string
		apiKey         string
		expectedStatus int
	}{
		{
			name:           "valid API key",
			apiKey:         validAPIKey,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing API key",
			apiKey:         "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid API key",
			apiKey:         "invalid-key",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := APIKeyAuth(validAPIKey, logger)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := SecurityHeaders()
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", rec.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", rec.Header().Get("Referrer-Policy"))
	assert.NotEmpty(t, rec.Header().Get("Content-Security-Policy"))
}
