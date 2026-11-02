package hatena

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetBookmarkCountsChunksAndDedupes(t *testing.T) {
	t.Parallel()
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "test-agent", r.Header.Get("User-Agent"))
		urls := r.URL.Query()["url"]
		require.LessOrEqual(t, len(urls), 2)
		payload := make(map[string]int, len(urls))
		for _, u := range urls {
			payload[u] = len(u)
		}
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer server.Close()

	httpClient := server.Client()
	httpClient.Timeout = time.Second

	client := NewClient(ClientConfig{
		HTTPClient:            httpClient,
		BookmarkCountEndpoint: server.URL,
		BookmarkCountMaxURLs:  2,
		UserAgent:             "test-agent",
	})

	ctx := context.Background()
	counts, err := client.GetBookmarkCounts(ctx, []string{
		"https://example.com/a",
		"https://example.com/b",
		"https://example.com/c",
		"https://example.com/a",
		"  ",
	})
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Equal(t, map[string]int{
		"https://example.com/a": len("https://example.com/a"),
		"https://example.com/b": len("https://example.com/b"),
		"https://example.com/c": len("https://example.com/c"),
	}, counts)
}

func TestGetBookmarkCountsHTTPError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		HTTPClient:            server.Client(),
		BookmarkCountEndpoint: server.URL,
	})

	_, err := client.GetBookmarkCounts(context.Background(), []string{"https://example.com"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bookmark count request failed")
}

func TestGetBookmarkCountsChunksByQueryLength(t *testing.T) {
	t.Parallel()
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.LessOrEqual(t, len(r.URL.RawQuery), bookmarkCountMaxQueryLen)
		urls := r.URL.Query()["url"]
		payload := make(map[string]int, len(urls))
		for _, u := range urls {
			payload[u] = 1
		}
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		HTTPClient:            server.Client(),
		BookmarkCountEndpoint: server.URL,
	})

	// 1件あたり約3000バイトのため、件数上限内でも2件ずつに分割される。上限超えの1件はスキップされる。
	tooLong := "https://example.com/" + strings.Repeat("x", bookmarkCountMaxQueryLen)
	urls := []string{tooLong}
	for _, c := range "abc" {
		urls = append(urls, "https://example.com/"+strings.Repeat(string(c), 3000))
	}

	counts, err := client.GetBookmarkCounts(context.Background(), urls)
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Len(t, counts, 3)
	require.NotContains(t, counts, tooLong)
}
