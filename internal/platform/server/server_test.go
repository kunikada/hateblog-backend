package server

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	logger := slog.Default()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := Config{
		Address:      "127.0.0.1:8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	srv := New(cfg, handler, logger)
	require.NotNil(t, srv)
	assert.Equal(t, cfg.Address, srv.Address())
}

func TestServer_Shutdown(t *testing.T) {
	logger := slog.Default()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := Config{
		Address:      "127.0.0.1:0", // Use port 0 for automatic port assignment
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		IdleTimeout:  1 * time.Second,
	}

	srv := New(cfg, handler, logger)

	// Start server in goroutine
	go func() {
		_ = srv.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := srv.Shutdown(ctx)
	require.NoError(t, err)
}

func TestConfig(t *testing.T) {
	cfg := Config{
		Address:      "0.0.0.0:8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	assert.Equal(t, "0.0.0.0:8080", cfg.Address)
	assert.Equal(t, 10*time.Second, cfg.ReadTimeout)
	assert.Equal(t, 10*time.Second, cfg.WriteTimeout)
	assert.Equal(t, 60*time.Second, cfg.IdleTimeout)
}
