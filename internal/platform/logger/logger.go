package logger

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"

	"hateblog/internal/pkg/timeutil"
)

// Level represents log level
type Level string

const (
	// LevelDebug enables debug logs.
	LevelDebug Level = "debug"
	// LevelInfo enables info logs.
	LevelInfo Level = "info"
	// LevelWarn enables warning logs.
	LevelWarn Level = "warn"
	// LevelError enables error logs.
	LevelError Level = "error"
)

// Format represents log output format
type Format string

const (
	// FormatText renders logs as colored text.
	FormatText Format = "text"
	// FormatJSON renders logs as JSON.
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

	// ReplaceAttr function to convert time to application timezone
	replaceAttr := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			if t, ok := a.Value.Any().(time.Time); ok {
				return slog.Time(slog.TimeKey, t.In(timeutil.Location()))
			}
		}
		return a
	}

	switch format {
	case FormatJSON:
		return slog.NewJSONHandler(os.Stdout, opts)
	case FormatText:
		// Use tint handler for pretty console output in development
		return tint.NewHandler(os.Stdout, &tint.Options{
			Level:       level,
			TimeFormat:  "2006-01-02 15:04:05",
			AddSource:   opts.AddSource,
			NoColor:     true,
			ReplaceAttr: replaceAttr,
		})
	default:
		// Default to text format
		return tint.NewHandler(os.Stdout, &tint.Options{
			Level:       level,
			TimeFormat:  "2006-01-02 15:04:05",
			AddSource:   opts.AddSource,
			NoColor:     true,
			ReplaceAttr: replaceAttr,
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
