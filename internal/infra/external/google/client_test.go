package google

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchSuccess(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/icon", r.URL.Path)
		require.Equal(t, "example.com", r.URL.Query().Get("domain"))
		require.Equal(t, "128", r.URL.Query().Get("sz"))
		w.Header().Set("Content-Type", "image/x-icon")
		_, _ = w.Write([]byte{0x01, 0x02})
	}))
	defer server.Close()

	client := NewClient(Config{
		HTTPClient: server.Client(),
		BaseURL:    server.URL + "/icon",
		Size:       128,
		UserAgent:  "test-agent",
	})

	data, contentType, err := client.Fetch(context.Background(), "https://example.com/page")
	require.NoError(t, err)
	require.Equal(t, []byte{0x01, 0x02}, data)
	require.Equal(t, "image/x-icon", contentType)
}

func TestFetchBadStatus(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient(Config{
		HTTPClient: server.Client(),
		BaseURL:    server.URL,
	})

	_, _, err := client.Fetch(context.Background(), "example.com")
	require.Error(t, err)
}

func TestFetchInvalidDomain(t *testing.T) {
	client := NewClient(Config{})
	_, _, err := client.Fetch(context.Background(), "bad host")
	require.Error(t, err)
}
