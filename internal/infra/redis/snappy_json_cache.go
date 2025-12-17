package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/golang/snappy"

	"hateblog/internal/platform/cache"
)

type bytesCacheClient interface {
	GetBytes(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

type snappyJSONCache struct {
	client bytesCacheClient
	ttl    time.Duration
}

func newSnappyJSONCache(client bytesCacheClient, ttl time.Duration) *snappyJSONCache {
	return &snappyJSONCache{client: client, ttl: ttl}
}

func (c *snappyJSONCache) Get(ctx context.Context, key string, out any) (bool, error) {
	payload, err := c.client.GetBytes(ctx, key)
	if err != nil {
		if errors.Is(err, cache.ErrCacheMiss) {
			return false, nil
		}
		return false, err
	}
	jsonData, err := snappy.Decode(nil, payload)
	if err != nil {
		return false, fmt.Errorf("snappy decode: %w", err)
	}
	if err := json.Unmarshal(jsonData, out); err != nil {
		return false, fmt.Errorf("json decode: %w", err)
	}
	return true, nil
}

func (c *snappyJSONCache) Set(ctx context.Context, key string, value any) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("json encode: %w", err)
	}
	payload := snappy.Encode(nil, jsonData)
	return c.client.Set(ctx, key, payload, c.ttl)
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
