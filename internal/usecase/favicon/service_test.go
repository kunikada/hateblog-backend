package favicon

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchUsesCache(t *testing.T) {
	cache := &mockCache{
		key:     "favicon:example.com",
		getData: []byte{1},
		getType: "image/png",
	}
	service := NewService(&mockFetcher{}, cache, nil, nil)

	data, ctype, err := service.Fetch(context.Background(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []byte{1}, data)
	require.Equal(t, "image/png", ctype)
	require.False(t, cache.setCalled)
}

func TestFetchMissFetchesAndCaches(t *testing.T) {
	cache := &mockCache{key: "favicon:example.com"}
	fetcher := &mockFetcher{
		data:  []byte{9},
		ctype: "image/x-icon",
	}
	service := NewService(fetcher, cache, nil, nil)

	data, ctype, err := service.Fetch(context.Background(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []byte{9}, data)
	require.Equal(t, "image/x-icon", ctype)
	require.True(t, cache.setCalled)
}

func TestFetchMissingDeps(t *testing.T) {
	service := NewService(nil, nil, nil, nil)
	_, _, err := service.Fetch(context.Background(), "example.com")
	require.Error(t, err)
}

func TestFetchRespectsRateLimit(t *testing.T) {
	cache := &mockCache{key: "favicon:example.com"}
	limiter := &mockLimiter{allow: false}
	service := NewService(&mockFetcher{}, cache, limiter, nil)

	_, _, err := service.Fetch(context.Background(), "example.com")
	require.ErrorIs(t, err, ErrRateLimited)
}

func TestFetchFallbackOnError(t *testing.T) {
	cache := &mockCache{key: "favicon:example.com"}
	fetcher := &mockFetcher{err: errors.New("boom")}
	service := NewService(fetcher, cache, nil, nil)

	data, ctype, err := service.Fetch(context.Background(), "example.com")
	require.NoError(t, err)
	require.Equal(t, defaultFaviconFallback, data)
	require.Equal(t, "image/png", ctype)
}

type mockFetcher struct {
	data  []byte
	ctype string
	err   error
}

func (m *mockFetcher) Fetch(ctx context.Context, domain string) ([]byte, string, error) {
	if m.err != nil {
		return nil, "", m.err
	}
	return m.data, m.ctype, nil
}

type mockCache struct {
	key       string
	getData   []byte
	getType   string
	setCalled bool
}

func (m *mockCache) BuildKey(domain string) (string, error) {
	return m.key, nil
}

func (m *mockCache) Get(ctx context.Context, key string) ([]byte, string, bool, error) {
	if len(m.getData) == 0 {
		return nil, "", false, nil
	}
	return m.getData, m.getType, true, nil
}

func (m *mockCache) Set(ctx context.Context, key string, data []byte, contentType string) error {
	m.setCalled = true
	return nil
}

type mockLimiter struct {
	allow bool
	err   error
}

func (m *mockLimiter) Allow(ctx context.Context, domain string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.allow, nil
}
