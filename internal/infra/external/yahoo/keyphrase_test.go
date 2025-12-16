package yahoo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractKeyphrasesSuccess(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.Equal(t, "agent", r.Header.Get("User-Agent"))
		require.Equal(t, "appid", r.Header.Get("X-Yahoo-App-Id"))

		var req keyphraseRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, keyphraseMethod, req.Method)
		require.Equal(t, "Hello world", req.Params.Query)

		_ = json.NewEncoder(w).Encode(keyphraseResponse{
			Result: &keyphraseResult{
				Phrases: []Keyphrase{
					{Text: "Go", Score: 100},
					{Text: "world", Score: 50},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		HTTPClient: server.Client(),
		Endpoint:   server.URL,
		AppID:      "appid",
		UserAgent:  "agent",
	})

	phrases, err := client.Extract(context.Background(), "Hello world")
	require.NoError(t, err)
	require.Equal(t, []Keyphrase{
		{Text: "Go", Score: 100},
		{Text: "world", Score: 50},
	}, phrases)
}

func TestExtractKeyphrasesReturnsNilWhenEmptyText(t *testing.T) {
	client := NewClient(ClientConfig{})
	phrases, err := client.Extract(context.Background(), "   ")
	require.NoError(t, err)
	require.Nil(t, phrases)
}

func TestExtractKeyphrasesRequiresYahooAppID(t *testing.T) {
	client := NewClient(ClientConfig{})
	_, err := client.Extract(context.Background(), "text")
	require.Error(t, err)
	require.Contains(t, err.Error(), "app id is required")
}

func TestExtractKeyphrasesAPIErrors(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(keyphraseResponse{
			Error: &keyphraseErrorBody{
				Code:    400,
				Message: "bad request",
			},
		})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		HTTPClient: server.Client(),
		Endpoint:   server.URL,
		AppID:      "appid",
	})

	_, err := client.Extract(context.Background(), "text")
	require.Error(t, err)
	require.Contains(t, err.Error(), "api error")
}
