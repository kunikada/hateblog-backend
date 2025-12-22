package postgres

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"hateblog/internal/domain/entry"
	"hateblog/internal/domain/tag"
)

// setupPostgres starts a PostgreSQL container, runs migrations, and returns a pool and cleanup function.
func setupPostgres(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()
	container, err := tcpostgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16"),
		tcpostgres.WithDatabase("test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
	)
	if err != nil {
		t.Skipf("skipping postgres integration test: %v", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	return pool, func() {
		pool.Close()
		_ = container.Terminate(context.Background())
	}
}

// applyTestMigrations runs the schema migrations from the migrations directory.
func applyTestMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	files := []string{
		"migrations/000001_create_entries.up.sql",
		"migrations/000002_create_tags.up.sql",
		"migrations/000003_create_entry_tags.up.sql",
	}
	for _, file := range files {
		if err := executeSQLFile(ctx, pool, file); err != nil {
			return err
		}
	}
	// Create tag_view_history table (not in migration files yet)
	return createTagViewHistoryTable(ctx, pool)
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

// createTagViewHistoryTable creates the tag_view_history table if it doesn't exist.
func createTagViewHistoryTable(ctx context.Context, pool *pgxpool.Pool) error {
	query := `CREATE TABLE IF NOT EXISTS tag_view_history (
		tag_id UUID NOT NULL,
		viewed_at DATE NOT NULL,
		count INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (tag_id, viewed_at),
		FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
	)`
	_, err := pool.Exec(ctx, query)
	return err
}

// cleanupTables removes all data from test tables.
func cleanupTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()

	tables := []string{
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
func insertEntryTag(t *testing.T, pool *pgxpool.Pool, entryID uuid.UUID, tagID uuid.UUID, score float64) {
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
