package hatena

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) fetchBookmarkCounts(ctx context.Context, urls []string) (map[string]int, error) {
	result := make(map[string]int, len(urls))
	cleanList := dedupeURLs(urls)
	if len(cleanList) == 0 {
		return result, nil
	}

	for start := 0; start < len(cleanList); start += c.bookmarkCountMaxURLs {
		end := start + c.bookmarkCountMaxURLs
		if end > len(cleanList) {
			end = len(cleanList)
		}
		chunk := cleanList[start:end]
		part, err := c.requestBookmarkCount(ctx, chunk)
		if err != nil {
			return nil, err
		}
		for u, count := range part {
			result[u] = count
		}
	}
	return result, nil
}

func (c *Client) requestBookmarkCount(ctx context.Context, urls []string) (map[string]int, error) {
	if len(urls) == 0 {
		return map[string]int{}, nil
	}

	endpoint, err := url.Parse(c.bookmarkCountEndpoint)
	if err != nil {
		return nil, fmt.Errorf("hatena: invalid bookmark endpoint: %w", err)
	}

	query := endpoint.Query()
	for _, u := range urls {
		query.Add("url", u)
	}
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("hatena: bookmark count request failed: status %d", resp.StatusCode)
	}

	var payload map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("hatena: failed to decode bookmark count: %w", err)
	}
	if payload == nil {
		payload = map[string]int{}
	}
	return payload, nil
}

func dedupeURLs(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	result := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, u := range input {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		result = append(result, u)
	}
	return result
}
