package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v10"
)

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
	Host            string        `env:"DB_HOST" envDefault:"localhost"`
	Port            int           `env:"DB_PORT" envDefault:"5432"`
	User            string        `env:"DB_USER" envDefault:"hateblog"`
	Password        string        `env:"DB_PASSWORD" envDefault:"hateblog"`
	Database        string        `env:"DB_NAME" envDefault:"hateblog"`
	SSLMode         string        `env:"DB_SSLMODE" envDefault:"disable"`
	MaxConns        int32         `env:"DB_MAX_CONNS" envDefault:"25"`
	MinConns        int32         `env:"DB_MIN_CONNS" envDefault:"5"`
	MaxConnLifetime time.Duration `env:"DB_MAX_CONN_LIFETIME" envDefault:"1h"`
	MaxConnIdleTime time.Duration `env:"DB_MAX_CONN_IDLE_TIME" envDefault:"30m"`
	ConnectTimeout  time.Duration `env:"DB_CONNECT_TIMEOUT" envDefault:"10s"`
}

// ConnectionString returns the PostgreSQL connection string
func (d DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
		d.Host, d.Port, d.User, d.Password, d.Database, d.SSLMode,
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
	Environment      string        `env:"APP_ENV" envDefault:"development"`
	LogLevel         string        `env:"APP_LOG_LEVEL" envDefault:"info"`
	LogFormat        string        `env:"APP_LOG_FORMAT" envDefault:"text"` // text or json
	CacheTTL         time.Duration `env:"APP_CACHE_TTL" envDefault:"1h"`
	FaviconCacheTTL  time.Duration `env:"APP_FAVICON_CACHE_TTL" envDefault:"168h"` // 7 days
	EnableMetrics    bool          `env:"APP_ENABLE_METRICS" envDefault:"true"`
	EnableCORS       bool          `env:"APP_ENABLE_CORS" envDefault:"true"`
	AllowedOrigins   []string      `env:"APP_ALLOWED_ORIGINS" envSeparator:"," envDefault:"*"`
	APIKeyRequired   bool          `env:"APP_API_KEY_REQUIRED" envDefault:"false"`
	MasterAPIKey     string        `env:"APP_MASTER_API_KEY" envDefault:""`
}

// IsDevelopment returns true if the environment is development
func (a AppConfig) IsDevelopment() bool {
	return a.Environment == "development"
}

// IsProduction returns true if the environment is production
func (a AppConfig) IsProduction() bool {
	return a.Environment == "production"
}

// ExternalConfig holds external API configuration
type ExternalConfig struct {
	// Yahoo! Keyphrase Extraction API
	YahooAPIKey string `env:"YAHOO_API_KEY" envDefault:""`

	// Google Favicon API settings
	FaviconAPITimeout time.Duration `env:"FAVICON_API_TIMEOUT" envDefault:"3s"`
	FaviconRateLimit  time.Duration `env:"FAVICON_RATE_LIMIT" envDefault:"1s"`

	// Hatena Bookmark API settings
	HatenaAPITimeout  time.Duration `env:"HATENA_API_TIMEOUT" envDefault:"10s"`
	HatenaMaxURLs     int           `env:"HATENA_MAX_URLS" envDefault:"50"`
	HatenaRSSFeedURLs []string      `env:"HATENA_RSS_FEED_URLS" envSeparator:"|" envDefault:"https://b.hatena.ne.jp/entrylist?sort=hot&mode=rss&threshold=5|https://feeds.feedburner.com/hatena/b/hotentry"`
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
	validEnvs := map[string]bool{
		"development": true,
		"staging":     true,
		"production":  true,
	}
	if !validEnvs[c.App.Environment] {
		return fmt.Errorf("invalid environment: %s (must be development, staging, or production)",
			c.App.Environment)
	}

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

	// Validate API key requirement
	if c.App.APIKeyRequired && c.App.MasterAPIKey == "" {
		return fmt.Errorf("master API key is required when API key authentication is enabled")
	}

	return nil
}
