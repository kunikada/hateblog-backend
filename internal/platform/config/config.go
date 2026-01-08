package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v10"
)

// DefaultAPIBasePath is the fallback base path for the HTTP API.
const DefaultAPIBasePath = "/api/v1"

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	Server ServerConfig

	// Database configuration
	Database DatabaseConfig

	// Redis configuration
	Redis RedisConfig

	// Application configuration
	App AppConfig

	// External API configuration
	External ExternalConfig

	// Sentry configuration
	Sentry SentryConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host         string        `env:"SERVER_HOST" envDefault:"0.0.0.0"`
	Port         int           `env:"SERVER_PORT" envDefault:"8080"`
	ReadTimeout  time.Duration `env:"SERVER_READ_TIMEOUT" envDefault:"10s"`
	WriteTimeout time.Duration `env:"SERVER_WRITE_TIMEOUT" envDefault:"10s"`
	IdleTimeout  time.Duration `env:"SERVER_IDLE_TIMEOUT" envDefault:"60s"`
}

// Address returns the server address in host:port format
func (s ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host            string        `env:"POSTGRES_HOST" envDefault:"localhost"`
	Port            int           `env:"POSTGRES_PORT" envDefault:"5432"`
	User            string        `env:"POSTGRES_USER" envDefault:"hateblog"`
	Password        string        `env:"POSTGRES_PASSWORD" envDefault:"hateblog"`
	Database        string        `env:"POSTGRES_DB" envDefault:"hateblog"`
	SSLMode         string        `env:"POSTGRES_SSLMODE" envDefault:"disable"`
	MaxConns        int32         `env:"POSTGRES_MAX_CONNS" envDefault:"25"`
	MinConns        int32         `env:"POSTGRES_MIN_CONNS" envDefault:"5"`
	MaxConnLifetime time.Duration `env:"POSTGRES_MAX_CONN_LIFETIME" envDefault:"1h"`
	MaxConnIdleTime time.Duration `env:"POSTGRES_MAX_CONN_IDLE_TIME" envDefault:"30m"`
	ConnectTimeout  time.Duration `env:"POSTGRES_CONNECT_TIMEOUT" envDefault:"10s"`
}

// ConnectionString returns the PostgreSQL connection string in URL format
func (d DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s&connect_timeout=%d",
		d.User, d.Password, d.Host, d.Port, d.Database, d.SSLMode,
		int(d.ConnectTimeout.Seconds()),
	)
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host         string        `env:"REDIS_HOST" envDefault:"localhost"`
	Port         int           `env:"REDIS_PORT" envDefault:"6379"`
	Password     string        `env:"REDIS_PASSWORD" envDefault:""`
	DB           int           `env:"REDIS_DB" envDefault:"0"`
	MaxRetries   int           `env:"REDIS_MAX_RETRIES" envDefault:"3"`
	DialTimeout  time.Duration `env:"REDIS_DIAL_TIMEOUT" envDefault:"5s"`
	ReadTimeout  time.Duration `env:"REDIS_READ_TIMEOUT" envDefault:"3s"`
	WriteTimeout time.Duration `env:"REDIS_WRITE_TIMEOUT" envDefault:"3s"`
	PoolSize     int           `env:"REDIS_POOL_SIZE" envDefault:"10"`
	MinIdleConns int           `env:"REDIS_MIN_IDLE_CONNS" envDefault:"5"`
}

// Address returns the Redis address in host:port format
func (r RedisConfig) Address() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// AppConfig holds application-specific configuration
type AppConfig struct {
	LogLevel        string        `env:"APP_LOG_LEVEL" envDefault:"info"`
	LogFormat       string        `env:"APP_LOG_FORMAT" envDefault:"text"` // text or json
	TimeZone        string        `env:"APP_TIMEZONE" envDefault:"Asia/Tokyo"`
	CacheEnabled    bool          `env:"APP_CACHE_ENABLED" envDefault:"true"`
	FaviconCacheTTL time.Duration `env:"APP_FAVICON_CACHE_TTL" envDefault:"168h"` // 7 days
	EnableMetrics   bool          `env:"APP_ENABLE_METRICS" envDefault:"true"`
	APIBasePath     string        `env:"APP_API_BASE_PATH" envDefault:"/api/v1"`
	APIKeyRequired  bool          `env:"APP_API_KEY_REQUIRED" envDefault:"false"`
	APIKeyPrefix    string        `env:"API_KEY_PREFIX" envDefault:"hb_live_"`
	APIKeyTTL       time.Duration `env:"APP_API_KEY_TTL" envDefault:"0"`

	RateLimitEnabled     bool          `env:"APP_RATE_LIMIT_ENABLED" envDefault:"false"`
	RateLimitWindow      time.Duration `env:"APP_RATE_LIMIT_WINDOW" envDefault:"1m"`
	RateLimitMaxRequests int           `env:"APP_RATE_LIMIT_MAX_REQUESTS" envDefault:"120"`
}

// ExternalConfig holds external API configuration
type ExternalConfig struct {
	// Yahoo! Keyphrase Extraction API
	YahooAPIKey string `env:"YAHOO_APP_ID" envDefault:""`

	// Google Favicon API settings
	FaviconAPITimeout time.Duration `env:"FAVICON_API_TIMEOUT" envDefault:"3s"`
	FaviconRateLimit  time.Duration `env:"FAVICON_RATE_LIMIT" envDefault:"1s"`

	// Hatena Bookmark API settings
	HatenaAPITimeout  time.Duration `env:"HATENA_API_TIMEOUT" envDefault:"10s"`
	HatenaMaxURLs     int           `env:"HATENA_MAX_URLS" envDefault:"50"`
	HatenaRSSFeedURLs []string      `env:"HATENA_RSS_FEED_URLS" envSeparator:"|" envDefault:"https://b.hatena.ne.jp/entrylist?sort=hot&mode=rss&threshold=5|https://feeds.feedburner.com/hatena/b/hotentry"`
}

// SentryConfig holds Sentry configuration
type SentryConfig struct {
	DSN         string `env:"SENTRY_DSN" envDefault:""`
	Environment string `env:"SENTRY_ENVIRONMENT" envDefault:""`
	Release     string `env:"SENTRY_RELEASE" envDefault:""`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Parse environment variables into config struct
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate performs validation on the configuration
func (c *Config) Validate() error {
	// Validate server configuration
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// Validate database configuration
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}
	if c.Database.MaxConns < c.Database.MinConns {
		return fmt.Errorf("database max connections (%d) must be >= min connections (%d)",
			c.Database.MaxConns, c.Database.MinConns)
	}

	// Validate Redis configuration
	if c.Redis.Host == "" {
		return fmt.Errorf("redis host is required")
	}
	if c.Redis.DB < 0 || c.Redis.DB > 15 {
		return fmt.Errorf("invalid redis database: %d (must be 0-15)", c.Redis.DB)
	}

	// Validate app configuration
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.App.LogLevel] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)",
			c.App.LogLevel)
	}

	validLogFormats := map[string]bool{
		"text": true,
		"json": true,
	}
	if !validLogFormats[c.App.LogFormat] {
		return fmt.Errorf("invalid log format: %s (must be text or json)",
			c.App.LogFormat)
	}

	if c.App.RateLimitEnabled {
		if c.App.RateLimitWindow <= 0 {
			return fmt.Errorf("rate limit window must be positive")
		}
		if c.App.RateLimitMaxRequests <= 0 {
			return fmt.Errorf("rate limit max requests must be positive")
		}
	}

	return nil
}
