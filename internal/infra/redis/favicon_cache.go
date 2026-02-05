package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"hateblog/internal/pkg/hostname"
	"hateblog/internal/platform/cache"
)

// FaviconCache caches favicon binaries in Redis.
type FaviconCache struct {
	client cacheClient
	ttl    time.Duration
}

type cacheClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

// NewFaviconCache constructs a cache wrapper.
func NewFaviconCache(client cacheClient, ttl time.Duration) *FaviconCache {
	return &FaviconCache{
		client: client,
		ttl:    ttl,
	}
}

// BuildKey normalizes the domain and returns cache key.
func (c *FaviconCache) BuildKey(domain string) (string, error) {
	host, err := hostname.Normalize(domain)
	if err != nil {
		return "", err
	}
	return "favicon:" + host, nil
}

// Get retrieves favicon from cache.
func (c *FaviconCache) Get(ctx context.Context, key string) ([]byte, string, bool, error) {
	payload, err := c.client.Get(ctx, key)
	if err != nil {
		if errors.Is(err, cache.ErrCacheMiss) {
			return nil, "", false, nil
		}
		return nil, "", false, err
	}
	var cached struct {
		ContentType string `json:"content_type"`
		Data        []byte `json:"data"`
	}
	if err := json.Unmarshal([]byte(payload), &cached); err != nil {
		return nil, "", false, fmt.Errorf("favicon cache decode: %w", err)
	}
	if len(cached.Data) == 0 {
		return nil, "", false, nil
	}
	return cached.Data, cached.ContentType, true, nil
}

// Set saves a favicon in cache.
func (c *FaviconCache) Set(ctx context.Context, key string, data []byte, contentType string) error {
	payload, err := json.Marshal(struct {
		ContentType string `json:"content_type"`
		Data        []byte `json:"data"`
	}{
		ContentType: contentType,
		Data:        data,
	})
	if err != nil {
		return fmt.Errorf("favicon cache encode: %w", err)
	}
	return c.client.Set(ctx, key, payload, c.ttl)
}
