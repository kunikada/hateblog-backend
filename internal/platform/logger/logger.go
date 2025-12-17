package logger

import (
	"log/slog"
	"os"
	"strings"

	"github.com/lmittmann/tint"
)

// Level represents log level
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// Format represents log output format
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Config holds logger configuration
type Config struct {
	Level  Level
	Format Format
}

// New creates a new structured logger with the given configuration
func New(cfg Config) *slog.Logger {
	level := parseLevel(cfg.Level)
	handler := createHandler(cfg.Format, level)
	return slog.New(handler)
}

// parseLevel converts string log level to slog.Level
func parseLevel(level Level) slog.Level {
	switch strings.ToLower(string(level)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// createHandler creates appropriate handler based on format
func createHandler(format Format, level slog.Level) slog.Handler {
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug, // Add source file info only in debug mode
	}

	switch format {
	case FormatJSON:
		return slog.NewJSONHandler(os.Stdout, opts)
	case FormatText:
		// Use tint handler for pretty console output in development
		return tint.NewHandler(os.Stdout, &tint.Options{
			Level:      level,
			TimeFormat: "15:04:05",
			AddSource:  opts.AddSource,
		})
	default:
		// Default to text format
		return tint.NewHandler(os.Stdout, &tint.Options{
			Level:      level,
			TimeFormat: "15:04:05",
			AddSource:  opts.AddSource,
		})
	}
}

// SetDefault sets the default logger for the application
func SetDefault(logger *slog.Logger) {
	slog.SetDefault(logger)
}

// WithContext creates a child logger with additional context fields
func WithContext(logger *slog.Logger, args ...any) *slog.Logger {
	return logger.With(args...)
}
