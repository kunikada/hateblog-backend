package migration

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Config holds migration configuration.
type Config struct {
	DatabaseURL    string
	MigrationsPath string // Path to migrations directory (e.g., "file://migrations")
	Logger         *slog.Logger
}

// Runner handles database migrations.
type Runner struct {
	migrate *migrate.Migrate
	logger  *slog.Logger
}

// New creates a new migration runner.
func New(cfg Config) (*Runner, error) {
	m, err := migrate.New(cfg.MigrationsPath, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return &Runner{
		migrate: m,
		logger:  cfg.Logger,
	}, nil
}

// Up runs all available migrations.
func (r *Runner) Up() error {
	if err := r.migrate.Up(); err != nil && err != migrate.ErrNoChange {
		version, dirty, versionErr := r.migrate.Version()
		switch {
		case versionErr == nil:
			return fmt.Errorf("migration up failed at version=%d dirty=%t: %w", version, dirty, err)
		case errors.Is(versionErr, migrate.ErrNilVersion):
			return fmt.Errorf("migration up failed before version tracking was created: %w", err)
		default:
			return fmt.Errorf("migration up failed: %w (version check failed: %v)", err, versionErr)
		}
	}

	version, dirty, err := r.migrate.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	if err == migrate.ErrNilVersion {
		r.logger.Info("no migrations applied")
	} else {
		r.logger.Info("migrations applied successfully", "version", version, "dirty", dirty)
	}

	return nil
}

// Down rolls back one migration.
func (r *Runner) Down() error {
	if err := r.migrate.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration down failed: %w", err)
	}
	return nil
}

// Force sets the migration version without running migrations.
func (r *Runner) Force(version int) error {
	if err := r.migrate.Force(version); err != nil {
		return fmt.Errorf("migration force failed: %w", err)
	}
	return nil
}

// Version returns the current migration version.
func (r *Runner) Version() (uint, bool, error) {
	version, dirty, err := r.migrate.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}
	if err == migrate.ErrNilVersion {
		return 0, false, nil
	}
	return version, dirty, nil
}

// Close closes the migration runner.
func (r *Runner) Close() error {
	srcErr, dbErr := r.migrate.Close()
	if srcErr != nil {
		return fmt.Errorf("failed to close source: %w", srcErr)
	}
	if dbErr != nil {
		return fmt.Errorf("failed to close database: %w", dbErr)
	}
	return nil
}
