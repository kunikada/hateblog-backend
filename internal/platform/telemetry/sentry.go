package telemetry

import (
	"fmt"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"

	"hateblog/internal/platform/config"
)

const defaultSentryEnvironment = "production"

// InitSentry initializes Sentry and returns whether it is enabled.
func InitSentry(cfg config.SentryConfig) (bool, error) {
	dsn := strings.TrimSpace(cfg.DSN)
	if dsn == "" {
		return false, nil
	}
	environment := strings.TrimSpace(cfg.Environment)
	if environment == "" {
		environment = defaultSentryEnvironment
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      environment,
		Release:          strings.TrimSpace(cfg.Release),
		AttachStacktrace: true,
	}); err != nil {
		return false, fmt.Errorf("init sentry: %w", err)
	}
	return true, nil
}

// Flush waits for buffered events to be delivered.
func Flush(timeout time.Duration) {
	sentry.Flush(timeout)
}

// Recover captures a panic and reports it to Sentry.
func Recover() {
	sentry.Recover()
}
