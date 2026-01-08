package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds database connection configuration
type Config struct {
	ConnectionString string
	MaxConns         int32
	MinConns         int32
	MaxConnLifetime  time.Duration
	MaxConnIdleTime  time.Duration
	ConnectTimeout   time.Duration
	TimeZone         string
}

// DB wraps pgxpool.Pool with additional functionality
type DB struct {
	*pgxpool.Pool
	logger *slog.Logger
}

// New creates a new database connection pool
func New(ctx context.Context, cfg Config, logger *slog.Logger) (*DB, error) {
	// Parse connection string
	poolCfg, err := pgxpool.ParseConfig(cfg.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Configure pool settings
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout
	if cfg.TimeZone != "" {
		if poolCfg.ConnConfig.RuntimeParams == nil {
			poolCfg.ConnConfig.RuntimeParams = map[string]string{}
		}
		poolCfg.ConnConfig.RuntimeParams["timezone"] = cfg.TimeZone
	}

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("database connection established",
		"max_conns", cfg.MaxConns,
		"min_conns", cfg.MinConns,
	)

	return &DB{
		Pool:   pool,
		logger: logger,
	}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	db.Pool.Close()
	db.logger.Info("database connection closed")
}

// HealthCheck performs a health check on the database
func (db *DB) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return db.Pool.Ping(ctx)
}

// Stats returns connection pool statistics
func (db *DB) Stats() *pgxpool.Stat {
	return db.Pool.Stat()
}

// LogStats logs current connection pool statistics
func (db *DB) LogStats() {
	stats := db.Stats()
	db.logger.Debug("database pool stats",
		"acquire_count", stats.AcquireCount(),
		"acquired_conns", stats.AcquiredConns(),
		"canceled_acquire_count", stats.CanceledAcquireCount(),
		"constructing_conns", stats.ConstructingConns(),
		"empty_acquire_count", stats.EmptyAcquireCount(),
		"idle_conns", stats.IdleConns(),
		"max_conns", stats.MaxConns(),
		"total_conns", stats.TotalConns(),
	)
}
