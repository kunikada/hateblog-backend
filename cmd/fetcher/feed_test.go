package main

import (
	"context"
	"errors"
	"testing"

	"hateblog/internal/infra/external/hatena"
)

type stubFeedFetcher struct {
	calls []string
}

func (s *stubFeedFetcher) FetchFeed(_ context.Context, url string) (*hatena.Feed, error) {
	s.calls = append(s.calls, url)
	switch url {
	case "failed":
		return nil, errors.New("HTTP 404")
	case "empty":
		return &hatena.Feed{}, nil
	default:
		return &hatena.Feed{Entries: []hatena.FeedEntry{{Title: "entry", URL: url}}}, nil
	}
}

func TestFetchEntriesFeedIsolation(t *testing.T) {
	tests := []struct {
		name    string
		urls    []string
		want    int
		wantErr bool
	}{
		{name: "failure before success", urls: []string{"failed", "first"}, want: 1},
		{name: "failure after success", urls: []string{"first", "failed"}, want: 1},
		{name: "failure between successes", urls: []string{"first", "failed", "second"}, want: 2},
		{name: "all failed", urls: []string{"failed", "failed"}, wantErr: true},
		{name: "empty successful feed", urls: []string{"failed", "empty"}},
		{name: "no configured feeds", urls: []string{" "}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &stubFeedFetcher{}
			entries, err := fetchEntries(context.Background(), client, tt.urls, 300)
			if (err != nil) != tt.wantErr {
				t.Fatalf("fetchEntries() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(entries) != tt.want {
				t.Fatalf("got %d entries, want %d", len(entries), tt.want)
			}
			wantCalls := 0
			for _, url := range tt.urls {
				if url != " " {
					wantCalls++
				}
			}
			if len(client.calls) != wantCalls {
				t.Fatalf("got %d feed requests, want %d", len(client.calls), wantCalls)
			}
		})
	}
}
