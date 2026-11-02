package hatena

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

// bookmarkCountMaxQueryLen はクエリ文字列の上限バイト数。
// APIはリクエストURL全体が約8KBを超えると件数に関係なく414を返すため、余裕を持たせている。
const bookmarkCountMaxQueryLen = 7000

func (c *Client) fetchBookmarkCounts(ctx context.Context, urls []string) (map[string]int, error) {
	result := make(map[string]int, len(urls))
	cleanList := dedupeURLs(urls)
	if len(cleanList) == 0 {
		return result, nil
	}

	for _, chunk := range c.chunkURLs(ctx, cleanList) {
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

	resp, err := c.httpClient.Do(req) // #nosec G704
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

// chunkURLs は件数とエンコード後のクエリ長の両方が上限を超えないようにURLを分割する。
// 単独で上限を超えるURLはスキップする。
func (c *Client) chunkURLs(ctx context.Context, urls []string) [][]string {
	var chunks [][]string
	var chunk []string
	queryLen := 0
	for _, u := range urls {
		paramLen := len("url=") + len(url.QueryEscape(u))
		if paramLen > bookmarkCountMaxQueryLen {
			slog.DebugContext(ctx, "hatena: skip bookmark count url exceeding query length limit", "url", u, "length", paramLen)
			continue
		}
		// 2件目以降は区切りの "&" 分を加算する。
		if len(chunk) >= c.bookmarkCountMaxURLs || (len(chunk) > 0 && queryLen+1+paramLen > bookmarkCountMaxQueryLen) {
			chunks = append(chunks, chunk)
			chunk = nil
			queryLen = 0
		}
		if len(chunk) > 0 {
			queryLen++
		}
		chunk = append(chunk, u)
		queryLen += paramLen
	}
	if len(chunk) > 0 {
		chunks = append(chunks, chunk)
	}
	return chunks
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
