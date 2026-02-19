package cache

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_InvalidConfig(t *testing.T) {
	logger := slog.Default()

	// Invalid address should cause connection failure
	cfg := Config{
		Address:      "invalid:9999",
		Password:     "",
		DB:           0,
		MaxRetries:   1,
		DialTimeout:  1 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		PoolSize:     5,
		MinIdleConns: 1,
	}

	cache, err := New(cfg, logger)
	require.Error(t, err)
	assert.Nil(t, cache)
}

// Note: Integration tests that require actual Redis connection
// should be placed in separate _integration_test.go files and use
// testcontainers for Redis

func TestConfig(t *testing.T) {
	cfg := Config{
		Address:      "localhost:6379",
		Password:     "",
		DB:           0,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	}

	assert.Equal(t, "localhost:6379", cfg.Address)
	assert.Equal(t, "", cfg.Password)
	assert.Equal(t, 0, cfg.DB)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, 5*time.Second, cfg.DialTimeout)
	assert.Equal(t, 3*time.Second, cfg.ReadTimeout)
	assert.Equal(t, 3*time.Second, cfg.WriteTimeout)
	assert.Equal(t, 10, cfg.PoolSize)
	assert.Equal(t, 5, cfg.MinIdleConns)
}

func TestErrCacheMiss(t *testing.T) {
	assert.Error(t, ErrCacheMiss)
	assert.Equal(t, "cache miss", ErrCacheMiss.Error())
}

func TestIsContextDoneError(t *testing.T) {
	assert.True(t, isContextDoneError(context.Canceled))
	assert.True(t, isContextDoneError(context.DeadlineExceeded))
	assert.True(t, isContextDoneError(fmt.Errorf("wrapped: %w", context.Canceled)))
	assert.False(t, isContextDoneError(nil))
	assert.False(t, isContextDoneError(ErrCacheMiss))
}
