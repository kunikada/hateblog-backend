package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		check   func(*testing.T, *Config)
	}{
		{
			name:    "default configuration",
			envVars: map[string]string{},
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				assert.True(t, cfg.App.CacheEnabled)
				assert.Equal(t, "0.0.0.0", cfg.Server.Host)
				assert.Equal(t, 8080, cfg.Server.Port)
				assert.Equal(t, "localhost", cfg.Database.Host)
				assert.Equal(t, 5432, cfg.Database.Port)
				assert.Equal(t, DefaultAPIBasePath, cfg.App.APIBasePath)
				assert.Equal(t, time.Duration(0), cfg.App.APIKeyTTL)
			},
		},
		{
			name: "custom configuration",
			envVars: map[string]string{
				"SERVER_PORT":     "9000",
				"POSTGRES_HOST":   "db.example.com",
				"POSTGRES_PORT":   "5433",
				"APP_LOG_LEVEL":   "debug",
				"APP_API_KEY_TTL": "30m",
			},
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 9000, cfg.Server.Port)
				assert.Equal(t, "db.example.com", cfg.Database.Host)
				assert.Equal(t, 5433, cfg.Database.Port)
				assert.Equal(t, "debug", cfg.App.LogLevel)
				assert.Equal(t, 30*time.Minute, cfg.App.APIKeyTTL)
			},
		},
		{
			name: "cache disabled",
			envVars: map[string]string{
				"APP_CACHE_ENABLED": "false",
			},
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				assert.False(t, cfg.App.CacheEnabled)
			},
		},
		{
			name: "invalid port",
			envVars: map[string]string{
				"SERVER_PORT": "70000",
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			envVars: map[string]string{
				"APP_LOG_LEVEL": "invalid",
			},
			wantErr: true,
		},
		{
			name: "API key required with custom prefix",
			envVars: map[string]string{
				"APP_API_KEY_REQUIRED": "true",
				"API_KEY_PREFIX":       "custom_",
			},
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				assert.True(t, cfg.App.APIKeyRequired)
				assert.Equal(t, "custom_", cfg.App.APIKeyPrefix)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := clearTestEnv()
			defer restore()

			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.envVars {
					os.Unsetenv(k)
				}
			}()

			cfg, err := Load()
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)

			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func clearTestEnv() func() {
	keys := []string{
		"SERVER_HOST", "SERVER_PORT", "SERVER_READ_TIMEOUT", "SERVER_WRITE_TIMEOUT", "SERVER_IDLE_TIMEOUT",
		"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB", "POSTGRES_SSLMODE",
		"POSTGRES_MAX_CONNS", "POSTGRES_MIN_CONNS", "POSTGRES_MAX_CONN_LIFETIME", "POSTGRES_MAX_CONN_IDLE_TIME", "POSTGRES_CONNECT_TIMEOUT",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB", "REDIS_MAX_RETRIES",
		"REDIS_DIAL_TIMEOUT", "REDIS_READ_TIMEOUT", "REDIS_WRITE_TIMEOUT", "REDIS_POOL_SIZE", "REDIS_MIN_IDLE_CONNS",
		"APP_LOG_LEVEL", "APP_LOG_FORMAT", "APP_TIMEZONE", "APP_CACHE_ENABLED", "APP_FAVICON_CACHE_TTL",
		"APP_ENABLE_METRICS", "APP_API_BASE_PATH",
		"APP_API_KEY_REQUIRED", "API_KEY_PREFIX", "APP_API_KEY_TTL", "APP_MASTER_API_KEY",
	}
	prev := make(map[string]string, len(keys))
	for _, k := range keys {
		prev[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	return func() {
		for k, v := range prev {
			if v == "" {
				os.Unsetenv(k)
				continue
			}
			os.Setenv(k, v)
		}
	}
}

func TestServerConfig_Address(t *testing.T) {
	cfg := ServerConfig{
		Host: "127.0.0.1",
		Port: 8080,
	}
	assert.Equal(t, "127.0.0.1:8080", cfg.Address())
}

func TestDatabaseConfig_ConnectionString(t *testing.T) {
	cfg := DatabaseConfig{
		Host:           "localhost",
		Port:           5432,
		User:           "user",
		Password:       "pass",
		Database:       "dbname",
		SSLMode:        "disable",
		ConnectTimeout: 10 * time.Second,
	}
	expected := "postgresql://user:pass@localhost:5432/dbname?sslmode=disable&connect_timeout=10"
	assert.Equal(t, expected, cfg.ConnectionString())
}

func TestRedisConfig_Address(t *testing.T) {
	cfg := RedisConfig{
		Host: "redis.example.com",
		Port: 6380,
	}
	assert.Equal(t, "redis.example.com:6380", cfg.Address())
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					Host:     "localhost",
					User:     "user",
					Database: "dbname",
					MaxConns: 25,
					MinConns: 5,
				},
				Redis: RedisConfig{
					Host: "localhost",
					DB:   0,
				},
				App: AppConfig{
					LogLevel:  "info",
					LogFormat: "text",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid DB max/min conns",
			config: &Config{
				Server: ServerConfig{Port: 8080},
				Database: DatabaseConfig{
					Host:     "localhost",
					User:     "user",
					Database: "dbname",
					MaxConns: 5,
					MinConns: 10,
				},
				Redis: RedisConfig{Host: "localhost"},
				App: AppConfig{
					LogLevel:  "info",
					LogFormat: "text",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
