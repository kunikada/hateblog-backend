package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"hateblog/internal/domain/entry"
	"hateblog/internal/platform/cache"
)

// EntryListCache stores serialized entry lists in Redis.
type EntryListCache struct {
	client *cache.Cache
	ttl    time.Duration
}

// NewEntryListCache builds a cache wrapper with the provided TTL (default 1h when ttl<=0).
func NewEntryListCache(client *cache.Cache, ttl time.Duration) *EntryListCache {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &EntryListCache{
		client: client,
		ttl:    ttl,
	}
}

// BuildKey normalizes the query and returns a deterministic cache key.
func (c *EntryListCache) BuildKey(query entry.ListQuery) (string, error) {
	q := query
	if err := q.Normalize(); err != nil {
		return "", err
	}
	var from, to string
	if !q.PostedAtFrom.IsZero() {
		from = q.PostedAtFrom.Format(time.RFC3339Nano)
	}
	if !q.PostedAtTo.IsZero() {
		to = q.PostedAtTo.Format(time.RFC3339Nano)
	}

	parts := []string{
		string(q.Sort),
		strconv.Itoa(q.MinBookmarkCount),
		strconv.Itoa(q.Offset),
		strconv.Itoa(q.Limit),
		strings.Join(q.Tags, ","),
		from,
		to,
		q.Keyword,
		strconv.Itoa(q.MaxLimitOverride),
	}

	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return "entry:list:" + hex.EncodeToString(sum[:]), nil
}

// Get fetches the cached payload.
func (c *EntryListCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	val, err := c.client.Get(ctx, key)
	if err != nil {
		if errors.Is(err, cache.ErrCacheMiss) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return []byte(val), true, nil
}

// Set saves payload to cache with TTL.
func (c *EntryListCache) Set(ctx context.Context, key string, payload []byte) error {
	return c.client.Set(ctx, key, payload, c.ttl)
}

// Delete invalidates a cache key.
func (c *EntryListCache) Delete(ctx context.Context, key string) error {
	return c.client.Delete(ctx, key)
}

// WithTTL overrides TTL for chaining (useful in tests).
func (c *EntryListCache) WithTTL(ttl time.Duration) *EntryListCache {
	if ttl <= 0 {
		panic("ttl must be positive")
	}
	return &EntryListCache{
		client: c.client,
		ttl:    ttl,
	}
}
