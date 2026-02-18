package hatena

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// FeedKind identifies a predefined Hatena RSS feed.
type FeedKind string

const (
	// FeedKindEntryList represents https://b.hatena.ne.jp/entrylist?mode=rss.
	FeedKindEntryList FeedKind = "entrylist"
	// FeedKindHotEntry represents https://b.hatena.ne.jp/hotentry?mode=rss.
	FeedKindHotEntry FeedKind = "hotentry"
)

const (
	defaultBookmarkEndpoint = "https://bookmark.hatenaapis.com/count/entries"
	defaultUserAgent        = "hateblog-bot/1.0"
	defaultHTTPTimeout      = 10 * time.Second
	maxURLUpperBound        = 50
)

var defaultFeeds = map[FeedKind]string{
	FeedKindEntryList: "https://b.hatena.ne.jp/entrylist?mode=rss",
	FeedKindHotEntry:  "https://b.hatena.ne.jp/hotentry?mode=rss",
}

// ClientConfig represents knobs required to talk to Hatena APIs.
type ClientConfig struct {
	HTTPClient            *http.Client
	FeedURLs              map[FeedKind]string
	BookmarkCountEndpoint string
	BookmarkCountMaxURLs  int
	UserAgent             string
}

// Client aggregates Hatena RSS とブックマーク件数APIを扱う。
type Client struct {
	httpClient            *http.Client
	feedURLs              map[FeedKind]string
	bookmarkCountEndpoint string
	bookmarkCountMaxURLs  int
	userAgent             string
}

// NewClient wires a client with sane defaults.
func NewClient(cfg ClientConfig) *Client {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: defaultHTTPTimeout,
		}
	}

	feedURLs := make(map[FeedKind]string, len(defaultFeeds))
	for kind, url := range defaultFeeds {
		feedURLs[kind] = url
	}
	for kind, url := range cfg.FeedURLs {
		if url == "" {
			continue
		}
		feedURLs[kind] = url
	}

	maxURLs := cfg.BookmarkCountMaxURLs
	if maxURLs <= 0 || maxURLs > maxURLUpperBound {
		maxURLs = maxURLUpperBound
	}

	userAgent := cfg.UserAgent
	if userAgent == "" {
		userAgent = defaultUserAgent
	}

	return &Client{
		httpClient:            httpClient,
		feedURLs:              feedURLs,
		bookmarkCountEndpoint: fallback(cfg.BookmarkCountEndpoint, defaultBookmarkEndpoint),
		bookmarkCountMaxURLs:  maxURLs,
		userAgent:             userAgent,
	}
}

// Feed represents RSS channel metadata and entries.
type Feed struct {
	Title       string
	Description string
	Link        string
	Entries     []FeedEntry
}

// FeedEntry represents a single RSS item.
type FeedEntry struct {
	Title         string
	URL           string
	Excerpt       string
	Content       string
	Subjects      []string
	BookmarkCount int
	PublishedAt   time.Time
}

// FetchFeedByKind retrieves a predefined Hatena RSS feed.
func (c *Client) FetchFeedByKind(ctx context.Context, kind FeedKind) (*Feed, error) {
	url, ok := c.feedURLs[kind]
	if !ok {
		return nil, fmt.Errorf("unknown feed kind: %s", kind)
	}
	return c.FetchFeed(ctx, url)
}

// FetchFeed grabs entries from the given RSS URL.
func (c *Client) FetchFeed(ctx context.Context, feedURL string) (*Feed, error) {
	if feedURL == "" {
		return nil, errors.New("feed url is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.httpClient.Do(req) // #nosec G704
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("rss request failed: status %d", resp.StatusCode)
	}
	return parseRSS(resp.Body)
}

// GetBookmarkCounts returns bookmark counts for the provided URLs (chunked when necessary).
func (c *Client) GetBookmarkCounts(ctx context.Context, urls []string) (map[string]int, error) {
	return c.fetchBookmarkCounts(ctx, urls)
}

func fallback(value, def string) string {
	if strings.TrimSpace(value) == "" {
		return def
	}
	return value
}
