package cache

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config holds Redis cache configuration
type Config struct {
	Address      string
	Password     string // #nosec G117
	DB           int
	MaxRetries   int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int
}

// Cache wraps redis.Client with additional functionality
type Cache struct {
	client *redis.Client
	logger *slog.Logger
}

// New creates a new Redis cache client
func New(cfg Config, logger *slog.Logger) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Address,
		Password:     cfg.Password,
		DB:           cfg.DB,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("redis connection established",
		"address", cfg.Address,
		"db", cfg.DB,
	)

	return &Cache{
		client: client,
		logger: logger,
	}, nil
}

// Close closes the Redis connection
func (c *Cache) Close() error {
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("failed to close Redis connection: %w", err)
	}
	c.logger.Info("redis connection closed")
	return nil
}

// HealthCheck performs a health check on Redis
func (c *Cache) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.Ping(ctx).Err()
}

// Get retrieves a value from cache
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrCacheMiss
	}
	if err != nil {
		c.logger.Error("failed to get cache", "key", key, "error", err)
		return "", fmt.Errorf("failed to get cache: %w", err)
	}
	return val, nil
}

// GetBytes retrieves a raw value from cache.
func (c *Cache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	val, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrCacheMiss
	}
	if err != nil {
		c.logger.Error("failed to get cache bytes", "key", key, "error", err)
		return nil, fmt.Errorf("failed to get cache bytes: %w", err)
	}
	return val, nil
}

// Set sets a value in cache with TTL
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		if isContextDoneError(err) {
			c.logger.Debug("cache set aborted by context", "key", key, "error", err)
			return fmt.Errorf("failed to set cache: %w", err)
		}
		c.logger.Error("failed to set cache", "key", key, "error", err)
		return fmt.Errorf("failed to set cache: %w", err)
	}
	return nil
}

func isContextDoneError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// Delete deletes a value from cache
func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		c.logger.Error("failed to delete cache", "keys", keys, "error", err)
		return fmt.Errorf("failed to delete cache: %w", err)
	}
	return nil
}

// DeleteByPattern deletes keys that match the pattern using SCAN.
func (c *Cache) DeleteByPattern(ctx context.Context, pattern string, batchSize int64) (int64, error) {
	if batchSize <= 0 {
		batchSize = 500
	}
	var cursor uint64
	var deleted int64
	for {
		keys, next, err := c.client.Scan(ctx, cursor, pattern, batchSize).Result()
		if err != nil {
			c.logger.Error("failed to scan keys", "pattern", pattern, "error", err)
			return deleted, fmt.Errorf("failed to scan keys: %w", err)
		}
		if len(keys) > 0 {
			n, err := c.client.Del(ctx, keys...).Result()
			if err != nil {
				c.logger.Error("failed to delete keys", "pattern", pattern, "error", err)
				return deleted, fmt.Errorf("failed to delete keys: %w", err)
			}
			deleted += n
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return deleted, nil
}

// Exists checks if a key exists in cache
func (c *Cache) Exists(ctx context.Context, keys ...string) (int64, error) {
	count, err := c.client.Exists(ctx, keys...).Result()
	if err != nil {
		c.logger.Error("failed to check cache existence", "keys", keys, "error", err)
		return 0, fmt.Errorf("failed to check cache existence: %w", err)
	}
	return count, nil
}

// Expire sets a TTL on an existing key
func (c *Cache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if err := c.client.Expire(ctx, key, ttl).Err(); err != nil {
		c.logger.Error("failed to set expiration", "key", key, "error", err)
		return fmt.Errorf("failed to set expiration: %w", err)
	}
	return nil
}

// Increment increments a counter in cache
func (c *Cache) Increment(ctx context.Context, key string) (int64, error) {
	val, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		c.logger.Error("failed to increment cache", "key", key, "error", err)
		return 0, fmt.Errorf("failed to increment cache: %w", err)
	}
	return val, nil
}

var incrementWithTTLScript = redis.NewScript(`
local v = redis.call('INCR', KEYS[1])
if v == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return v
`)

// IncrementWithTTL increments a counter and sets TTL when the key is created.
func (c *Cache) IncrementWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	if ttl <= 0 {
		return 0, fmt.Errorf("ttl must be positive")
	}
	val, err := incrementWithTTLScript.Run(ctx, c.client, []string{key}, ttl.Milliseconds()).Int64()
	if err != nil {
		c.logger.Error("failed to increment cache with ttl", "key", key, "error", err)
		return 0, fmt.Errorf("failed to increment cache with ttl: %w", err)
	}
	return val, nil
}

// SetNX sets a value only if the key does not exist
func (c *Cache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	ok, err := c.client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		c.logger.Error("failed to setnx cache", "key", key, "error", err)
		return false, fmt.Errorf("failed to setnx cache: %w", err)
	}
	return ok, nil
}

// FlushDB flushes all keys in the current database (use with caution)
func (c *Cache) FlushDB(ctx context.Context) error {
	if err := c.client.FlushDB(ctx).Err(); err != nil {
		c.logger.Error("failed to flush database", "error", err)
		return fmt.Errorf("failed to flush database: %w", err)
	}
	c.logger.Warn("redis database flushed")
	return nil
}

// GetClient returns the underlying Redis client for advanced operations
func (c *Cache) GetClient() *redis.Client {
	return c.client
}

// Stats returns Redis pool statistics
func (c *Cache) Stats() *redis.PoolStats {
	return c.client.PoolStats()
}

// LogStats logs current Redis pool statistics
func (c *Cache) LogStats() {
	stats := c.Stats()
	c.logger.Debug("redis pool stats",
		"hits", stats.Hits,
		"misses", stats.Misses,
		"timeouts", stats.Timeouts,
		"total_conns", stats.TotalConns,
		"idle_conns", stats.IdleConns,
		"stale_conns", stats.StaleConns,
	)
}
