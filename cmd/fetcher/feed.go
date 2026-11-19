package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"hateblog/internal/infra/external/hatena"
)

type feedItem struct {
	Title         string
	URL           string
	Excerpt       string
	Subject       string
	BookmarkCount int
	PostedAt      time.Time
}

type feedFetcher interface {
	FetchFeed(context.Context, string) (*hatena.Feed, error)
}

func fetchEntries(ctx context.Context, client feedFetcher, feedURLs []string, max int) ([]feedItem, error) {
	if max <= 0 {
		max = 1
	}
	seen := make(map[string]feedItem, max)
	succeeded := 0
	var failures []error
	for _, raw := range feedURLs {
		u := strings.TrimSpace(raw)
		if u == "" {
			continue
		}
		feed, err := client.FetchFeed(ctx, u)
		if err != nil {
			failures = append(failures, fmt.Errorf("feed %s: %w", u, err))
			slog.ErrorContext(ctx, "fetch feed failed; continuing with other feeds", "url", u, "err", err)
			continue
		}
		succeeded++
		for _, e := range feed.Entries {
			url := strings.TrimSpace(e.URL)
			if url == "" {
				continue
			}
			if _, ok := seen[url]; ok {
				continue
			}
			subject := strings.Join(e.Subjects, ",")
			seen[url] = feedItem{
				Title:         strings.TrimSpace(e.Title),
				URL:           url,
				Excerpt:       strings.TrimSpace(e.Excerpt),
				Subject:       strings.TrimSpace(subject),
				BookmarkCount: e.BookmarkCount,
				PostedAt:      e.PublishedAt.In(time.Local),
			}
			if len(seen) >= max {
				break
			}
		}
		if len(seen) >= max {
			break
		}
	}

	if succeeded == 0 {
		if len(failures) > 0 {
			return nil, fmt.Errorf("all feeds failed: %w", errors.Join(failures...))
		}
		return nil, fmt.Errorf("no feed URLs configured")
	}

	items := make([]feedItem, 0, len(seen))
	for _, v := range seen {
		items = append(items, v)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].PostedAt.After(items[j].PostedAt)
	})
	return items, nil
}

