package google

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"hateblog/internal/pkg/hostname"
)

const (
	defaultBaseURL = "https://www.google.com/s2/favicons"
	defaultSize    = 64
	defaultUA      = "hateblog-bot/1.0"
)

// Client fetches favicons via the Google Favicon API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	size       int
	userAgent  string
}

// Config configures the Client.
type Config struct {
	HTTPClient *http.Client
	BaseURL    string
	Size       int
	UserAgent  string
}

// NewClient builds a Google favicon client.
func NewClient(cfg Config) *Client {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		base = defaultBaseURL
	}
	size := cfg.Size
	if size <= 0 {
		size = defaultSize
	}
	ua := strings.TrimSpace(cfg.UserAgent)
	if ua == "" {
		ua = defaultUA
	}
	return &Client{
		httpClient: httpClient,
		baseURL:    base,
		size:       size,
		userAgent:  ua,
	}
}

// Fetch retrieves a favicon binary and content type.
func (c *Client) Fetch(ctx context.Context, domain string) ([]byte, string, error) {
	host, err := hostname.Normalize(domain)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildURL(host), nil)
	if err != nil {
		return nil, "", err
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	resp, err := c.httpClient.Do(req) // #nosec G704
	if err != nil {
		return nil, "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return nil, "", fmt.Errorf("google favicon: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}

	return body, contentType, nil
}

func (c *Client) buildURL(domain string) string {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return c.baseURL
	}
	query := base.Query()
	query.Set("domain", domain)
	if c.size > 0 {
		query.Set("sz", strconv.Itoa(c.size))
	}
	base.RawQuery = query.Encode()
	return base.String()
}
