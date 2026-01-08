package yahoo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	defaultEndpoint = "https://jlp.yahooapis.jp/KeyphraseService/V2/extract"
	userAgent       = "hateblog-bot/1.0"
	keyphraseMethod = "jlp.keyphraseservice.extract"
)

// ClientConfig controls Yahoo API接続設定.
type ClientConfig struct {
	HTTPClient *http.Client
	Endpoint   string
	AppID      string
	UserAgent  string
}

// Client wraps Yahoo! JAPAN Keyphrase API.
type Client struct {
	httpClient *http.Client
	endpoint   string
	appID      string
	userAgent  string
}

// NewClient creates a new keyphrase client.
func NewClient(cfg ClientConfig) *Client {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	ua := strings.TrimSpace(cfg.UserAgent)
	if ua == "" {
		ua = userAgent
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	return &Client{
		httpClient: httpClient,
		endpoint:   endpoint,
		appID:      cfg.AppID,
		userAgent:  ua,
	}
}

// Keyphrase represents Yahoo keyphrase output.
type Keyphrase struct {
	Text  string `json:"text"`
	Score int    `json:"score"`
}

// Extract returns keyphrases for the provided text.
func (c *Client) Extract(ctx context.Context, text string) ([]Keyphrase, error) {
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	if c.appID == "" {
		return nil, errors.New("yahoo: app id is required")
	}

	reqPayload := keyphraseRequest{
		ID:      fmt.Sprintf("hateblog-%d", time.Now().UnixNano()),
		JSONRPC: "2.0",
		Method:  keyphraseMethod,
		Params: keyphraseParams{
			Query: text,
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(reqPayload); err != nil {
		return nil, fmt.Errorf("yahoo: failed to encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("X-Yahoo-App-Id", c.appID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		retryAfter := retryAfterDuration(resp.Header.Get("Retry-After"))
		_ = resp.Body.Close()
		return nil, &StatusError{StatusCode: resp.StatusCode, RetryAfter: retryAfter}
	}

	var res keyphraseResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("yahoo: failed to decode response: %w", err)
	}
	_ = resp.Body.Close()

	if res.Error != nil {
		return nil, fmt.Errorf("yahoo: api error %d: %s", res.Error.Code, res.Error.Message)
	}
	if res.Result == nil {
		return nil, errors.New("yahoo: missing result")
	}

	phrases := make([]Keyphrase, 0, len(res.Result.Phrases))
	for _, phrase := range res.Result.Phrases {
		if strings.TrimSpace(phrase.Text) == "" {
			continue
		}
		phrases = append(phrases, Keyphrase{
			Text:  strings.TrimSpace(phrase.Text),
			Score: phrase.Score,
		})
	}
	return phrases, nil
}

type keyphraseRequest struct {
	ID      string          `json:"id"`
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  keyphraseParams `json:"params"`
}

type keyphraseParams struct {
	Query string `json:"q"`
}

type keyphraseResponse struct {
	ID      string              `json:"id"`
	JSONRPC string              `json:"jsonrpc"`
	Result  *keyphraseResult    `json:"result"`
	Error   *keyphraseErrorBody `json:"error"`
}

type keyphraseResult struct {
	Phrases []Keyphrase `json:"phrases"`
}

type keyphraseErrorBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type StatusError struct {
	StatusCode int
	RetryAfter time.Duration
}

func (e *StatusError) Error() string {
	if e == nil {
		return "yahoo: status error"
	}
	return fmt.Sprintf("yahoo: keyphrase request failed: status %d", e.StatusCode)
}

func IsTooManyRequests(err error) (time.Duration, bool) {
	var statusErr *StatusError
	if !errors.As(err, &statusErr) {
		return 0, false
	}
	if statusErr.StatusCode != http.StatusTooManyRequests {
		return 0, false
	}
	return statusErr.RetryAfter, true
}

func retryAfterDuration(raw string) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if seconds, err := time.ParseDuration(raw + "s"); err == nil && seconds > 0 {
		return seconds
	}
	if t, err := http.ParseTime(raw); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}
