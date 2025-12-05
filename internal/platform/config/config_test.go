package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct{
		name    string
		envVars map[string]string
		wantErr bool
		check   func(*testing.T, *Config)
	}{
		{
			name: "default configuration",
			envVars: map[string]string{},
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "0.0.0.0", cfg.Server.Host)
				assert.Equal(t, 8080, cfg.Server.Port)
				assert.Equal(t, "localhost", cfg.Database.Host)
				assert.Equal(t, 5432, cfg.Database.Port)
				assert.Equal(t, "development", cfg.App.Environment)
			},
		},
		{
			name: "custom configuration",
			envVars: map[string]string{
				"SERVER_PORT":   "9000",
				"DB_HOST":       "db.example.com",
				"DB_PORT":       "5433",
				"APP_ENV":       "production",
				"APP_LOG_LEVEL": "debug",
			},
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 9000, cfg.Server.Port)
				assert.Equal(t, "db.example.com", cfg.Database.Host)
				assert.Equal(t, 5433, cfg.Database.Port)
				assert.Equal(t, "production", cfg.App.Environment)
				assert.Equal(t, "debug", cfg.App.LogLevel)
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
			name: "invalid environment",
			envVars: map[string]string{
				"APP_ENV": "invalid",
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
			name: "API key required without master key",
			envVars: map[string]string{
				"APP_API_KEY_REQUIRED": "true",
			},
			wantErr: true,
		},
		{
			name: "API key required with master key",
			envVars: map[string]string{
				"APP_API_KEY_REQUIRED": "true",
				"APP_MASTER_API_KEY":   "test-key",
			},
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				assert.True(t, cfg.App.APIKeyRequired)
				assert.Equal(t, "test-key", cfg.App.MasterAPIKey)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer func() {
				// Clean up environment variables
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
	expected := "host=localhost port=5432 user=user password=pass dbname=dbname sslmode=disable connect_timeout=10"
	assert.Equal(t, expected, cfg.ConnectionString())
}

func TestRedisConfig_Address(t *testing.T) {
	cfg := RedisConfig{
		Host: "redis.example.com",
		Port: 6380,
	}
	assert.Equal(t, "redis.example.com:6380", cfg.Address())
}

func TestAppConfig_IsDevelopment(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		want        bool
	}{
		{"development", "development", true},
		{"production", "production", false},
		{"staging", "staging", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := AppConfig{Environment: tt.environment}
			assert.Equal(t, tt.want, cfg.IsDevelopment())
		})
	}
}

func TestAppConfig_IsProduction(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		want        bool
	}{
		{"development", "development", false},
		{"production", "production", true},
		{"staging", "staging", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := AppConfig{Environment: tt.environment}
			assert.Equal(t, tt.want, cfg.IsProduction())
		})
	}
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
					Environment: "development",
					LogLevel:    "info",
					LogFormat:   "text",
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
					Environment: "development",
					LogLevel:    "info",
					LogFormat:   "text",
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
