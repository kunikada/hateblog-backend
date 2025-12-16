package redis

import (
	"context"
	"testing"
	"time"

	"hateblog/internal/platform/cache"

	"github.com/stretchr/testify/require"
)

func TestFaviconCacheSetAndGet(t *testing.T) {
	client := &mockCache{store: make(map[string]string)}
	fc := NewFaviconCache(client, time.Minute)

	key, err := fc.BuildKey("Example.com")
	require.NoError(t, err)

	require.NoError(t, fc.Set(context.Background(), key, []byte{1, 2}, "image/png"))

	data, ctype, ok, err := fc.Get(context.Background(), key)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []byte{1, 2}, data)
	require.Equal(t, "image/png", ctype)
}

func TestFaviconCacheMiss(t *testing.T) {
	client := &mockCache{store: make(map[string]string)}
	fc := NewFaviconCache(client, time.Minute)

	_, _, ok, err := fc.Get(context.Background(), "favicon:missing")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestFaviconCacheInvalidDomain(t *testing.T) {
	client := &mockCache{store: make(map[string]string)}
	fc := NewFaviconCache(client, time.Minute)

	_, err := fc.BuildKey("bad host")
	require.Error(t, err)
}

type mockCache struct {
	store map[string]string
}

func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	if val, ok := m.store[key]; ok {
		return val, nil
	}
	return "", cache.ErrCacheMiss
}

func (m *mockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	switch v := value.(type) {
	case []byte:
		m.store[key] = string(v)
	case string:
		m.store[key] = v
	default:
		return nil
	}
	return nil
}
