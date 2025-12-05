package database

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_InvalidConfig(t *testing.T) {
	logger := slog.Default()
	ctx := context.Background()

	cfg := Config{
		ConnectionString: "invalid connection string",
		MaxConns:         10,
		MinConns:         2,
		MaxConnLifetime:  1 * time.Hour,
		MaxConnIdleTime:  30 * time.Minute,
		ConnectTimeout:   10 * time.Second,
	}

	db, err := New(ctx, cfg, logger)
	require.Error(t, err)
	assert.Nil(t, db)
}

// Note: Integration tests that require actual database connection
// should be placed in separate _integration_test.go files and use
// testcontainers for PostgreSQL

func TestConfig(t *testing.T) {
	cfg := Config{
		ConnectionString: "host=localhost port=5432 user=test password=test dbname=test",
		MaxConns:         25,
		MinConns:         5,
		MaxConnLifetime:  1 * time.Hour,
		MaxConnIdleTime:  30 * time.Minute,
		ConnectTimeout:   10 * time.Second,
	}

	assert.Equal(t, int32(25), cfg.MaxConns)
	assert.Equal(t, int32(5), cfg.MinConns)
	assert.Equal(t, 1*time.Hour, cfg.MaxConnLifetime)
	assert.Equal(t, 30*time.Minute, cfg.MaxConnIdleTime)
	assert.Equal(t, 10*time.Second, cfg.ConnectTimeout)
}
