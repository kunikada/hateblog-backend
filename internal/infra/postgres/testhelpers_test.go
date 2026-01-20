package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
		connStr = "postgresql://hateblog:changeme@postgres:5432/hateblog?sslmode=disable"
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
	if err := applyTestMigrations(ctx, pool); err != nil {
		pool.Close()
		t.Fatalf("failed to apply migrations: %v", err)
	}

	return pool, func() {
		pool.Close()
	}
}

// applyTestMigrations runs the schema migrations from the migrations directory.
func applyTestMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Check if migrations already applied by looking for the latest table
	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = 'archive_counts'
	`).Scan(&count)
	if err == nil && count > 0 {
		// Migrations already applied
		return nil
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

	files := []string{
		filepath.Join(migrationsDir, "000001_create_entries.up.sql"),
		filepath.Join(migrationsDir, "000002_create_tags.up.sql"),
		filepath.Join(migrationsDir, "000003_create_entry_tags.up.sql"),
		filepath.Join(migrationsDir, "000004_create_click_metrics.up.sql"),
		filepath.Join(migrationsDir, "000005_create_tag_view_history.up.sql"),
		filepath.Join(migrationsDir, "000006_create_search_history.up.sql"),
		filepath.Join(migrationsDir, "000008_enable_pg_bigm.up.sql"),
		filepath.Join(migrationsDir, "000009_create_fulltext_indexes.up.sql"),
		filepath.Join(migrationsDir, "000012_create_archive_counts.up.sql"),
	}
	for _, file := range files {
		if err := executeSQLFile(ctx, pool, file); err != nil {
			return err
		}
	}
	return nil
}

// executeSQLFile reads and executes a SQL file.
func executeSQLFile(ctx context.Context, pool *pgxpool.Pool, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	statements := splitStatements(string(content))
	for _, stmt := range statements {
		if stmt == "" {
			continue
		}
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("exec %s: %w", path, err)
		}
	}
	return nil
}

// splitStatements splits SQL content into individual statements.
func splitStatements(sql string) []string {
	lines := strings.Split(sql, "\n")
	var builder strings.Builder
	var statements []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		builder.WriteString(line)
		if strings.HasSuffix(line, ";") {
			stmt := strings.TrimSuffix(builder.String(), ";")
			statements = append(statements, strings.TrimSpace(stmt))
			builder.Reset()
		} else {
			builder.WriteString("\n")
		}
	}

	// capture remaining
	if residual := strings.TrimSpace(builder.String()); residual != "" {
		statements = append(statements, residual)
	}
	return statements
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
SELECT DATE(posted_at) AS day, t.threshold, COUNT(1)
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

	const query = `
		INSERT INTO entries (id, title, url, posted_at, bookmark_count, excerpt, subject, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := pool.Exec(ctx, query,
		e.ID,
		e.Title,
		e.URL,
		e.PostedAt,
		e.BookmarkCount,
		nullString(e.Excerpt),
		nullString(e.Subject),
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
