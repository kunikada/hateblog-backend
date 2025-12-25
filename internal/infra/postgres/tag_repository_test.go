package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hateblog/internal/domain/tag"
)

func TestTagRepository_Upsert(t *testing.T) {
	pool, terminate := setupPostgres(t)
	defer terminate()

	ctx := context.Background()
	require.NoError(t, applyTestMigrations(ctx, pool))

	repo := NewTagRepository(pool)

	t.Run("inserts new tag", func(t *testing.T) {
		cleanupTables(t, pool)

		tg := testTag("golang")
		err := repo.Upsert(ctx, tg)
		require.NoError(t, err)

		// Verify tag was created
		got, err := repo.Get(ctx, tg.ID)
		require.NoError(t, err)
		assertTagEqual(t, tg, got)
	})

	t.Run("updates existing tag by name", func(t *testing.T) {
		cleanupTables(t, pool)

		tg := testTag("golang")
		insertTag(t, pool, tg)

		// Try to upsert with same name but different ID
		tg2 := testTag("golang")
		err := repo.Upsert(ctx, tg2)
		require.NoError(t, err)

		// Should still have the original ID
		assert.Equal(t, tg.ID, tg2.ID)
	})

	t.Run("normalizes tag name", func(t *testing.T) {
		cleanupTables(t, pool)

		tg := &tag.Tag{
			ID:   uuid.New(),
			Name: "  GoLang  ",
		}
		err := repo.Upsert(ctx, tg)
		require.NoError(t, err)

		// Name should be normalized to lowercase and trimmed
		assert.Equal(t, "golang", tg.Name)
	})

	t.Run("generates ID if not provided", func(t *testing.T) {
		cleanupTables(t, pool)

		tg := &tag.Tag{
			ID:   uuid.Nil,
			Name: "python",
		}
		err := repo.Upsert(ctx, tg)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, tg.ID)
	})

	t.Run("returns error for nil tag", func(t *testing.T) {
		err := repo.Upsert(ctx, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "tag is nil")
	})

	t.Run("returns error for empty name", func(t *testing.T) {
		tg := &tag.Tag{
			ID:   uuid.New(),
			Name: "",
		}
		err := repo.Upsert(ctx, tg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "tag name is required")
	})
}

func TestTagRepository_Get(t *testing.T) {
	pool, terminate := setupPostgres(t)
	defer terminate()

	ctx := context.Background()
	require.NoError(t, applyTestMigrations(ctx, pool))

	repo := NewTagRepository(pool)

	t.Run("gets tag by ID", func(t *testing.T) {
		cleanupTables(t, pool)

		tg := testTag("golang")
		insertTag(t, pool, tg)

		got, err := repo.Get(ctx, tg.ID)
		require.NoError(t, err)
		assertTagEqual(t, tg, got)
	})

	t.Run("returns error for non-existent tag", func(t *testing.T) {
		cleanupTables(t, pool)

		_, err := repo.Get(ctx, uuid.New())
		require.Error(t, err)
		require.Contains(t, err.Error(), "get tag")
	})

	t.Run("returns error for nil UUID", func(t *testing.T) {
		_, err := repo.Get(ctx, uuid.Nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "tag id is required")
	})
}

func TestTagRepository_GetByName(t *testing.T) {
	pool, terminate := setupPostgres(t)
	defer terminate()

	ctx := context.Background()
	require.NoError(t, applyTestMigrations(ctx, pool))

	repo := NewTagRepository(pool)

	t.Run("gets tag by name", func(t *testing.T) {
		cleanupTables(t, pool)

		tg := testTag("golang")
		insertTag(t, pool, tg)

		got, err := repo.GetByName(ctx, "golang")
		require.NoError(t, err)
		assertTagEqual(t, tg, got)
	})

	t.Run("normalizes name before lookup", func(t *testing.T) {
		cleanupTables(t, pool)

		tg := testTag("golang")
		insertTag(t, pool, tg)

		// Try to get with different casing
		got, err := repo.GetByName(ctx, "  GoLang  ")
		require.NoError(t, err)
		assertTagEqual(t, tg, got)
	})

	t.Run("returns error for non-existent tag", func(t *testing.T) {
		cleanupTables(t, pool)

		_, err := repo.GetByName(ctx, "nonexistent")
		require.Error(t, err)
		require.Contains(t, err.Error(), "get tag by name")
	})

	t.Run("returns error for empty name", func(t *testing.T) {
		_, err := repo.GetByName(ctx, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "tag name is required")
	})
}

func TestTagRepository_List(t *testing.T) {
	pool, terminate := setupPostgres(t)
	defer terminate()

	ctx := context.Background()
	require.NoError(t, applyTestMigrations(ctx, pool))

	repo := NewTagRepository(pool)

	t.Run("lists all tags ordered by name", func(t *testing.T) {
		cleanupTables(t, pool)

		// Insert tags in non-alphabetical order
		tag1 := testTag("python")
		tag2 := testTag("golang")
		tag3 := testTag("rust")
		insertTag(t, pool, tag1)
		insertTag(t, pool, tag2)
		insertTag(t, pool, tag3)

		tags, err := repo.List(ctx, 100, 0)
		require.NoError(t, err)
		require.Len(t, tags, 3)

		// Should be ordered alphabetically
		assert.Equal(t, "golang", tags[0].Name)
		assert.Equal(t, "python", tags[1].Name)
		assert.Equal(t, "rust", tags[2].Name)
	})

	t.Run("paginates results", func(t *testing.T) {
		cleanupTables(t, pool)

		// Insert 5 tags
		for _, name := range []string{"alpha", "bravo", "charlie", "delta", "echo"} {
			tg := testTag(name)
			insertTag(t, pool, tg)
		}

		// Get first page
		page1, err := repo.List(ctx, 2, 0)
		require.NoError(t, err)
		require.Len(t, page1, 2)
		assert.Equal(t, "alpha", page1[0].Name)
		assert.Equal(t, "bravo", page1[1].Name)

		// Get second page
		page2, err := repo.List(ctx, 2, 2)
		require.NoError(t, err)
		require.Len(t, page2, 2)
		assert.Equal(t, "charlie", page2[0].Name)
		assert.Equal(t, "delta", page2[1].Name)
	})

	t.Run("uses default limit when limit is 0", func(t *testing.T) {
		cleanupTables(t, pool)

		for i := 0; i < 5; i++ {
			tg := testTag(string(rune('a' + i)))
			insertTag(t, pool, tg)
		}

		tags, err := repo.List(ctx, 0, 0)
		require.NoError(t, err)
		require.Len(t, tags, 5)
	})

	t.Run("handles negative offset", func(t *testing.T) {
		cleanupTables(t, pool)

		tg := testTag("golang")
		insertTag(t, pool, tg)

		tags, err := repo.List(ctx, 10, -1)
		require.NoError(t, err)
		require.Len(t, tags, 1)
	})

	t.Run("returns empty slice when no results", func(t *testing.T) {
		cleanupTables(t, pool)

		tags, err := repo.List(ctx, 10, 0)
		require.NoError(t, err)
		require.Len(t, tags, 0)
	})
}

func TestTagRepository_Delete(t *testing.T) {
	pool, terminate := setupPostgres(t)
	defer terminate()

	ctx := context.Background()
	require.NoError(t, applyTestMigrations(ctx, pool))

	repo := NewTagRepository(pool)

	t.Run("deletes tag successfully", func(t *testing.T) {
		cleanupTables(t, pool)

		tg := testTag("golang")
		insertTag(t, pool, tg)

		err := repo.Delete(ctx, tg.ID)
		require.NoError(t, err)

		// Verify deletion
		_, err = repo.Get(ctx, tg.ID)
		require.Error(t, err)
	})

	t.Run("deletes tag with entry relationships (cascade)", func(t *testing.T) {
		cleanupTables(t, pool)

		// Create entry and tag
		e := testEntry()
		tg := testTag("golang")
		insertEntry(t, pool, e)
		insertTag(t, pool, tg)
		insertEntryTag(t, pool, e.ID, tg.ID, 0)

		// Delete tag
		err := repo.Delete(ctx, tg.ID)
		require.NoError(t, err)

		// Verify entry-tag relationship was deleted
		var count int
		err = pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM entry_tags WHERE tag_id = $1", tg.ID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("deletes tag with view history (cascade)", func(t *testing.T) {
		cleanupTables(t, pool)

		tg := testTag("golang")
		insertTag(t, pool, tg)

		// Add view history
		_, err := pool.Exec(ctx,
			"INSERT INTO tag_view_history (tag_id, viewed_at, count) VALUES ($1, $2, $3)",
			tg.ID, time.Now().UTC().Truncate(24*time.Hour), 5)
		require.NoError(t, err)

		// Delete tag
		err = repo.Delete(ctx, tg.ID)
		require.NoError(t, err)

		// Verify view history was deleted
		var count int
		err = pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM tag_view_history WHERE tag_id = $1", tg.ID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("returns error for nil UUID", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.Nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "tag id is required")
	})

	t.Run("succeeds for non-existent tag", func(t *testing.T) {
		cleanupTables(t, pool)

		err := repo.Delete(ctx, uuid.New())
		require.NoError(t, err)
	})
}

func TestTagRepository_IncrementViewHistory(t *testing.T) {
	pool, terminate := setupPostgres(t)
	defer terminate()

	ctx := context.Background()
	require.NoError(t, applyTestMigrations(ctx, pool))

	repo := NewTagRepository(pool)

	t.Run("increments view count for new date", func(t *testing.T) {
		cleanupTables(t, pool)

		tg := testTag("golang")
		insertTag(t, pool, tg)

		viewedAt := time.Now().UTC()
		err := repo.IncrementViewHistory(ctx, tg.ID, viewedAt)
		require.NoError(t, err)

		// Verify count was set to 1
		var count int
		date := viewedAt.Truncate(24 * time.Hour)
		err = pool.QueryRow(ctx,
			"SELECT count FROM tag_view_history WHERE tag_id = $1 AND viewed_at = $2",
			tg.ID, date).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("increments view count for existing date", func(t *testing.T) {
		cleanupTables(t, pool)

		tg := testTag("golang")
		insertTag(t, pool, tg)

		viewedAt := time.Now().UTC()
		date := viewedAt.Truncate(24 * time.Hour)

		// First increment
		err := repo.IncrementViewHistory(ctx, tg.ID, viewedAt)
		require.NoError(t, err)

		// Second increment
		err = repo.IncrementViewHistory(ctx, tg.ID, viewedAt)
		require.NoError(t, err)

		// Verify count was incremented to 2
		var count int
		err = pool.QueryRow(ctx,
			"SELECT count FROM tag_view_history WHERE tag_id = $1 AND viewed_at = $2",
			tg.ID, date).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("truncates timestamp to date", func(t *testing.T) {
		cleanupTables(t, pool)

		tg := testTag("golang")
		insertTag(t, pool, tg)

		// Increment with different times on same day
		now := time.Now().UTC()
		err := repo.IncrementViewHistory(ctx, tg.ID, now)
		require.NoError(t, err)

		err = repo.IncrementViewHistory(ctx, tg.ID, now.Add(5*time.Hour))
		require.NoError(t, err)

		// Should have only one row with count 2
		var rows int
		err = pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM tag_view_history WHERE tag_id = $1", tg.ID).Scan(&rows)
		require.NoError(t, err)
		assert.Equal(t, 1, rows)

		var count int
		date := now.Truncate(24 * time.Hour)
		err = pool.QueryRow(ctx,
			"SELECT count FROM tag_view_history WHERE tag_id = $1 AND viewed_at = $2",
			tg.ID, date).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("handles different dates separately", func(t *testing.T) {
		cleanupTables(t, pool)

		tg := testTag("golang")
		insertTag(t, pool, tg)

		today := time.Now().UTC()
		yesterday := today.Add(-24 * time.Hour)

		err := repo.IncrementViewHistory(ctx, tg.ID, today)
		require.NoError(t, err)

		err = repo.IncrementViewHistory(ctx, tg.ID, yesterday)
		require.NoError(t, err)

		// Should have two separate rows
		var rows int
		err = pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM tag_view_history WHERE tag_id = $1", tg.ID).Scan(&rows)
		require.NoError(t, err)
		assert.Equal(t, 2, rows)
	})

	t.Run("returns error for nil UUID", func(t *testing.T) {
		err := repo.IncrementViewHistory(ctx, uuid.Nil, time.Now())
		require.Error(t, err)
		require.Contains(t, err.Error(), "tag id is required")
	})

	t.Run("returns error for non-existent tag", func(t *testing.T) {
		cleanupTables(t, pool)

		err := repo.IncrementViewHistory(ctx, uuid.New(), time.Now())
		require.Error(t, err)
	})
}
