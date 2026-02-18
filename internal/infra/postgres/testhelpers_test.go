package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"hateblog/internal/domain/entry"
	"hateblog/internal/domain/tag"
)

// setupPostgres connects to a PostgreSQL database for testing.
// Uses TEST_POSTGRES_URL environment variable if set, otherwise uses default connection for Dev Container.
// Automatically runs migrations if needed.
func setupPostgres(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	connStr := os.Getenv("TEST_POSTGRES_URL")
	if connStr == "" {
		// Default connection for Dev Container (postgres, redis are running via docker-compose)
		connStr = "postgresql://hateblog:changeme@postgres:5432/hateblog_test?sslmode=disable"
	}
	if err := validateTestDBConnectionString(connStr); err != nil {
		t.Fatalf("invalid TEST_POSTGRES_URL: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Skipf("failed to connect to test database: %v", err)
	}

	// Verify connection with retries (container might still be starting)
	for i := 0; i < 30; i++ {
		if err := pool.Ping(ctx); err == nil {
			break
		}
		if i == 29 {
			pool.Close()
			t.Skipf("failed to ping test database after retries: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Apply migrations if needed
	if err := applyTestMigrations(ctx, pool, connStr); err != nil {
		pool.Close()
		t.Fatalf("failed to apply migrations: %v", err)
	}

	return pool, func() {
		pool.Close()
	}
}

func validateTestDBConnectionString(connStr string) error {
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return fmt.Errorf("parse connection string: %w", err)
	}
	if cfg.ConnConfig == nil {
		return fmt.Errorf("connection config is empty")
	}
	if cfg.ConnConfig.Database == "hateblog" {
		return fmt.Errorf("database 'hateblog' is not allowed for tests; use a dedicated test database")
	}
	return nil
}

// applyTestMigrations runs the schema migrations from the migrations directory.
func applyTestMigrations(ctx context.Context, pool *pgxpool.Pool, connStr string) error {
	// Always reset schema to avoid stale structures between test runs.
	if _, err := pool.Exec(ctx, "DROP SCHEMA IF EXISTS public CASCADE"); err != nil {
		return fmt.Errorf("drop schema: %w", err)
	}
	if _, err := pool.Exec(ctx, "CREATE SCHEMA public"); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	if _, err := pool.Exec(ctx, "GRANT ALL ON SCHEMA public TO public"); err != nil {
		return fmt.Errorf("grant schema: %w", err)
	}

	// Find the migrations directory from current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Try to find migrations directory starting from cwd and going up
	migrationsDir := filepath.Join(cwd, "migrations")
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(migrationsDir); err == nil {
			break
		}
		cwd = filepath.Dir(cwd)
		migrationsDir = filepath.Join(cwd, "migrations")
	}
	if _, err := os.Stat(migrationsDir); err != nil {
		return fmt.Errorf("migrations directory not found: %w", err)
	}

	absMigrationsDir, err := filepath.Abs(migrationsDir)
	if err != nil {
		return fmt.Errorf("resolve migrations directory: %w", err)
	}

	m, err := migrate.New("file://"+absMigrationsDir, connStr)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}

	return nil
}

// cleanupTables removes all data from test tables.
func cleanupTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()

	tables := []string{
		"archive_counts",
		"entry_tags",
		"tag_view_history",
		"entries",
		"tags",
	}

	for _, table := range tables {
		_, err := pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		require.NoError(t, err, "failed to cleanup table %s", table)
	}
}

// refreshArchiveCounts rebuilds archive_counts from entries for tests.
func refreshArchiveCounts(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()

	if _, err := pool.Exec(ctx, "TRUNCATE TABLE archive_counts"); err != nil {
		require.NoError(t, err, "failed to truncate archive_counts")
	}

	const query = `
INSERT INTO archive_counts (day, threshold, count)
SELECT DATE(created_at) AS day, t.threshold, COUNT(1)
FROM entries
CROSS JOIN (VALUES (5), (10), (50), (100), (500), (1000)) AS t(threshold)
WHERE entries.bookmark_count >= t.threshold
GROUP BY day, t.threshold`
	if _, err := pool.Exec(ctx, query); err != nil {
		require.NoError(t, err, "failed to rebuild archive_counts")
	}
}

// testEntry creates a test entry with default values.
func testEntry(overrides ...func(*entry.Entry)) *entry.Entry {
	e := &entry.Entry{
		ID:            uuid.New(),
		URL:           fmt.Sprintf("https://example.com/test-article-%s", uuid.New().String()),
		Title:         "Test Article",
		Excerpt:       "This is a test excerpt",
		Subject:       "Test Subject",
		BookmarkCount: 10,
		PostedAt:      time.Now().UTC().Truncate(time.Microsecond),
		CreatedAt:     time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:     time.Now().UTC().Truncate(time.Microsecond),
	}

	for _, override := range overrides {
		override(e)
	}

	return e
}

// testTag creates a test tag with default values.
func testTag(name string) *tag.Tag {
	return &tag.Tag{
		ID:   uuid.New(),
		Name: tag.NormalizeName(name),
	}
}

// insertEntry inserts an entry directly into the database.
func insertEntry(t *testing.T, pool *pgxpool.Pool, e *entry.Entry) {
	t.Helper()

	ctx := context.Background()

	searchText := entry.BuildSearchText(e.Title, e.Excerpt, e.URL)

	const query = `
		INSERT INTO entries (id, title, url, posted_at, bookmark_count, excerpt, subject, search_text, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := pool.Exec(ctx, query,
		e.ID,
		e.Title,
		e.URL,
		e.PostedAt,
		e.BookmarkCount,
		nullString(e.Excerpt),
		nullString(e.Subject),
		nullString(searchText),
		e.CreatedAt,
		e.UpdatedAt,
	)
	require.NoError(t, err)
}

// insertTag inserts a tag directly into the database.
func insertTag(t *testing.T, pool *pgxpool.Pool, tg *tag.Tag) {
	t.Helper()

	ctx := context.Background()

	const query = `INSERT INTO tags (id, name, created_at) VALUES ($1, $2, $3)`
	_, err := pool.Exec(ctx, query, tg.ID, tg.Name, time.Now().UTC())
	require.NoError(t, err)
}

// insertEntryTag inserts an entry-tag relationship directly into the database.
func insertEntryTag(t *testing.T, pool *pgxpool.Pool, entryID uuid.UUID, tagID uuid.UUID, score int) {
	t.Helper()

	ctx := context.Background()

	const query = `INSERT INTO entry_tags (entry_id, tag_id, score) VALUES ($1, $2, $3)`
	_, err := pool.Exec(ctx, query, entryID, tagID, score)
	require.NoError(t, err)
}

// nullString returns nil if the string is empty, otherwise returns the string.
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// assertEntryEqual compares two entries for equality.
func assertEntryEqual(t *testing.T, expected, actual *entry.Entry) {
	t.Helper()

	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.URL, actual.URL)
	require.Equal(t, expected.Title, actual.Title)
	require.Equal(t, expected.Excerpt, actual.Excerpt)
	require.Equal(t, expected.Subject, actual.Subject)
	require.Equal(t, expected.BookmarkCount, actual.BookmarkCount)
	require.WithinDuration(t, expected.PostedAt, actual.PostedAt, time.Second)
	require.WithinDuration(t, expected.CreatedAt, actual.CreatedAt, time.Second)
	require.WithinDuration(t, expected.UpdatedAt, actual.UpdatedAt, time.Second)
}

// assertTagEqual compares two tags for equality.
func assertTagEqual(t *testing.T, expected, actual *tag.Tag) {
	t.Helper()

	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.Name, actual.Name)
}
