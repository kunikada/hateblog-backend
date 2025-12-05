package logger

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "text format with info level",
			config: Config{
				Level:  LevelInfo,
				Format: FormatText,
			},
		},
		{
			name: "json format with debug level",
			config: Config{
				Level:  LevelDebug,
				Format: FormatJSON,
			},
		},
		{
			name: "text format with error level",
			config: Config{
				Level:  LevelError,
				Format: FormatText,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.config)
			require.NotNil(t, logger)
			assert.IsType(t, &slog.Logger{}, logger)
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name  string
		level Level
		want  slog.Level
	}{
		{"debug", LevelDebug, slog.LevelDebug},
		{"info", LevelInfo, slog.LevelInfo},
		{"warn", LevelWarn, slog.LevelWarn},
		{"error", LevelError, slog.LevelError},
		{"invalid", "invalid", slog.LevelInfo}, // Should default to info
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLevel(tt.level)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCreateHandler(t *testing.T) {
	tests := []struct {
		name   string
		format Format
		level  slog.Level
	}{
		{"json handler", FormatJSON, slog.LevelInfo},
		{"text handler", FormatText, slog.LevelDebug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := createHandler(tt.format, tt.level)
			require.NotNil(t, handler)
		})
	}
}

func TestWithContext(t *testing.T) {
	logger := New(Config{
		Level:  LevelInfo,
		Format: FormatText,
	})

	contextLogger := WithContext(logger, "key", "value")
	require.NotNil(t, contextLogger)
	assert.IsType(t, &slog.Logger{}, contextLogger)
}

func TestSetDefault(t *testing.T) {
	logger := New(Config{
		Level:  LevelInfo,
		Format: FormatText,
	})

	// Should not panic
	assert.NotPanics(t, func() {
		SetDefault(logger)
	})
}
