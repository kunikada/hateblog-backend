package postgres

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	domainEntry "hateblog/internal/domain/entry"
	"hateblog/internal/domain/tag"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestEntryRepository_ListAndCount(t *testing.T) {
	pool, terminate := setupPostgres(t)
	defer terminate()

	ctx := context.Background()
	require.NoError(t, applyTestMigrations(ctx, pool))

	repo := NewEntryRepository(pool)
	tagRepo := NewTagRepository(pool)

	goTag, err := tag.New(uuid.New(), "Go")
	require.NoError(t, err)
	require.NoError(t, tagRepo.Upsert(ctx, &goTag))

	now := time.Now().UTC()
	entry1, err := domainEntry.New(domainEntry.Params{
		ID:            uuid.New(),
		URL:           "https://example.com/1",
		Title:         "Entry1",
		BookmarkCount: 50,
		PostedAt:      now.Add(-1 * time.Hour),
		CreatedAt:     now.Add(-2 * time.Hour),
		UpdatedAt:     now.Add(-2 * time.Hour),
	})
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, entry1))

	entry2, err := domainEntry.New(domainEntry.Params{
		ID:            uuid.New(),
		URL:           "https://example.com/2",
		Title:         "Entry2",
		BookmarkCount: 10,
		PostedAt:      now,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, entry2))

	// attach tags
	_, err = pool.Exec(ctx, `INSERT INTO entry_tags (entry_id, tag_id, score) VALUES ($1, $2, $3)`, entry1.ID, goTag.ID, 0.9)
	require.NoError(t, err)

	result, err := repo.List(ctx, domainEntry.ListQuery{
		Limit: 2,
		Sort:  domainEntry.SortHot,
	})
	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, entry1.ID, result[0].ID)
	require.Len(t, result[0].Tags, 1)
	require.Equal(t, goTag.ID, result[0].Tags[0].TagID)

	resultFiltered, err := repo.List(ctx, domainEntry.ListQuery{
		MinBookmarkCount: 20,
		Limit:            5,
		Sort:             domainEntry.SortHot,
	})
	require.NoError(t, err)
	require.Len(t, resultFiltered, 1)
	require.Equal(t, entry1.ID, resultFiltered[0].ID)

	count, err := repo.Count(ctx, domainEntry.ListQuery{
		MinBookmarkCount: 20,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	entries, total, err := repo.ListAndCount(ctx, domainEntry.ListQuery{
		Limit: 2,
		Sort:  domainEntry.SortHot,
	})
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, int64(2), total)

	entries, total, err = repo.ListAndCount(ctx, domainEntry.ListQuery{
		Keyword: "%' OR 1=1 --",
		Limit:   10,
	})
	require.NoError(t, err)
	require.Len(t, entries, 0)
	require.Equal(t, int64(0), total)
}

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
	return nil
}

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
