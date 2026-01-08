package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainEntry "hateblog/internal/domain/entry"
	"hateblog/internal/domain/tag"
)

func TestEntryRepository_Create(t *testing.T) {
	pool, terminate := setupPostgres(t)
	defer terminate()

	ctx := context.Background()
	require.NoError(t, applyTestMigrations(ctx, pool))

	repo := NewEntryRepository(pool)

	t.Run("creates entry successfully", func(t *testing.T) {
		cleanupTables(t, pool)

		e := testEntry()
		err := repo.Create(ctx, e)
		require.NoError(t, err)

		// Verify entry was created
		got, err := repo.Get(ctx, e.ID)
		require.NoError(t, err)
		assertEntryEqual(t, e, got)
	})

	t.Run("generates ID if not provided", func(t *testing.T) {
		cleanupTables(t, pool)

		e := testEntry()
		e.ID = uuid.Nil

		err := repo.Create(ctx, e)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, e.ID)
	})

	t.Run("sets timestamps if not provided", func(t *testing.T) {
		cleanupTables(t, pool)

		e := testEntry()
		e.CreatedAt = time.Time{}
		e.UpdatedAt = time.Time{}

		err := repo.Create(ctx, e)
		require.NoError(t, err)
		require.False(t, e.CreatedAt.IsZero())
		require.False(t, e.UpdatedAt.IsZero())
	})

	t.Run("returns error for nil entry", func(t *testing.T) {
		err := repo.Create(ctx, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "entry is nil")
	})
}

func TestEntryRepository_Get(t *testing.T) {
	pool, terminate := setupPostgres(t)
	defer terminate()

	ctx := context.Background()
	require.NoError(t, applyTestMigrations(ctx, pool))

	repo := NewEntryRepository(pool)

	t.Run("gets entry by ID", func(t *testing.T) {
		cleanupTables(t, pool)

		e := testEntry()
		insertEntry(t, pool, e)

		got, err := repo.Get(ctx, e.ID)
		require.NoError(t, err)
		assertEntryEqual(t, e, got)
	})

	t.Run("gets entry with tags", func(t *testing.T) {
		cleanupTables(t, pool)

		// Create entry
		e := testEntry()
		insertEntry(t, pool, e)

		// Create tags
		tag1 := testTag("golang")
		tag2 := testTag("testing")
		insertTag(t, pool, tag1)
		insertTag(t, pool, tag2)

		// Link tags to entry
		insertEntryTag(t, pool, e.ID, tag1.ID, 0)
		insertEntryTag(t, pool, e.ID, tag2.ID, 0)

		// Get entry
		got, err := repo.Get(ctx, e.ID)
		require.NoError(t, err)
		require.Len(t, got.Tags, 2)

		// Verify tags are loaded (order may vary)
		tagIDs := []uuid.UUID{got.Tags[0].TagID, got.Tags[1].TagID}
		assert.Contains(t, tagIDs, tag1.ID)
		assert.Contains(t, tagIDs, tag2.ID)
	})

	t.Run("returns error for non-existent entry", func(t *testing.T) {
		cleanupTables(t, pool)

		_, err := repo.Get(ctx, uuid.New())
		require.Error(t, err)
		require.Contains(t, err.Error(), "entry not found")
	})

	t.Run("returns error for nil UUID", func(t *testing.T) {
		_, err := repo.Get(ctx, uuid.Nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "entry id is required")
	})
}

func TestEntryRepository_Update(t *testing.T) {
	pool, terminate := setupPostgres(t)
	defer terminate()

	ctx := context.Background()
	require.NoError(t, applyTestMigrations(ctx, pool))

	repo := NewEntryRepository(pool)

	t.Run("updates entry successfully", func(t *testing.T) {
		cleanupTables(t, pool)

		// Create entry
		e := testEntry()
		insertEntry(t, pool, e)

		// Update entry
		e.Title = "Updated Title"
		e.BookmarkCount = 100
		e.Excerpt = "Updated excerpt"
		e.UpdatedAt = time.Now().UTC()

		err := repo.Update(ctx, e)
		require.NoError(t, err)

		// Verify update
		got, err := repo.Get(ctx, e.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Title", got.Title)
		assert.Equal(t, 100, got.BookmarkCount)
		assert.Equal(t, "Updated excerpt", got.Excerpt)
	})

	t.Run("sets updated_at if not provided", func(t *testing.T) {
		cleanupTables(t, pool)

		e := testEntry()
		insertEntry(t, pool, e)

		e.Title = "Updated Title"
		e.UpdatedAt = time.Time{}

		err := repo.Update(ctx, e)
		require.NoError(t, err)
		require.False(t, e.UpdatedAt.IsZero())
	})

	t.Run("returns error for nil entry", func(t *testing.T) {
		err := repo.Update(ctx, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "entry is nil")
	})

	t.Run("returns error for nil UUID", func(t *testing.T) {
		e := testEntry()
		e.ID = uuid.Nil

		err := repo.Update(ctx, e)
		require.Error(t, err)
		require.Contains(t, err.Error(), "entry id is required")
	})
}

func TestEntryRepository_Delete(t *testing.T) {
	pool, terminate := setupPostgres(t)
	defer terminate()

	ctx := context.Background()
	require.NoError(t, applyTestMigrations(ctx, pool))

	repo := NewEntryRepository(pool)

	t.Run("deletes entry successfully", func(t *testing.T) {
		cleanupTables(t, pool)

		e := testEntry()
		insertEntry(t, pool, e)

		err := repo.Delete(ctx, e.ID)
		require.NoError(t, err)

		// Verify deletion
		_, err = repo.Get(ctx, e.ID)
		require.Error(t, err)
	})

	t.Run("deletes entry with tags (cascade)", func(t *testing.T) {
		cleanupTables(t, pool)

		// Create entry with tags
		e := testEntry()
		insertEntry(t, pool, e)

		tag1 := testTag("golang")
		insertTag(t, pool, tag1)
		insertEntryTag(t, pool, e.ID, tag1.ID, 0)

		// Delete entry
		err := repo.Delete(ctx, e.ID)
		require.NoError(t, err)

		// Verify entry-tag relationship was deleted
		var count int
		err = pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM entry_tags WHERE entry_id = $1", e.ID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("returns error for nil UUID", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.Nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "entry id is required")
	})

	t.Run("succeeds for non-existent entry", func(t *testing.T) {
		cleanupTables(t, pool)

		err := repo.Delete(ctx, uuid.New())
		require.NoError(t, err)
	})
}

func TestEntryRepository_List(t *testing.T) {
	pool, terminate := setupPostgres(t)
	defer terminate()

	ctx := context.Background()
	require.NoError(t, applyTestMigrations(ctx, pool))

	repo := NewEntryRepository(pool)

	t.Run("lists all entries with default query", func(t *testing.T) {
		cleanupTables(t, pool)

		// Insert entries
		e1 := testEntry(func(e *domainEntry.Entry) {
			e.PostedAt = time.Now().UTC().Add(-1 * time.Hour)
		})
		e2 := testEntry(func(e *domainEntry.Entry) {
			e.PostedAt = time.Now().UTC()
		})
		insertEntry(t, pool, e1)
		insertEntry(t, pool, e2)

		// List entries
		entries, err := repo.List(ctx, domainEntry.ListQuery{})
		require.NoError(t, err)
		require.Len(t, entries, 2)

		// Should be ordered by posted_at DESC
		assert.Equal(t, e2.ID, entries[0].ID)
		assert.Equal(t, e1.ID, entries[1].ID)
	})

	t.Run("filters by tags", func(t *testing.T) {
		cleanupTables(t, pool)

		// Create entries
		e1 := testEntry()
		e2 := testEntry()
		insertEntry(t, pool, e1)
		insertEntry(t, pool, e2)

		// Create tags
		golangTag := testTag("golang")
		pythonTag := testTag("python")
		insertTag(t, pool, golangTag)
		insertTag(t, pool, pythonTag)

		// Link tags
		insertEntryTag(t, pool, e1.ID, golangTag.ID, 0)
		insertEntryTag(t, pool, e2.ID, pythonTag.ID, 0)

		// Filter by golang tag
		entries, err := repo.List(ctx, domainEntry.ListQuery{
			Tags: []string{"golang"},
		})
		require.NoError(t, err)
		require.Len(t, entries, 1)
		assert.Equal(t, e1.ID, entries[0].ID)
	})

	t.Run("filters by keyword", func(t *testing.T) {
		cleanupTables(t, pool)

		e1 := testEntry(func(e *domainEntry.Entry) {
			e.Title = "Learning Go"
			e.Excerpt = "Go basics"
			e.Subject = "intro"
			e.URL = "https://example.com/golang/intro"
		})
		e2 := testEntry(func(e *domainEntry.Entry) {
			e.Title = "Python Tutorial"
		})
		insertEntry(t, pool, e1)
		insertEntry(t, pool, e2)

		entries, err := repo.List(ctx, domainEntry.ListQuery{
			Keyword: "golang",
		})
		require.NoError(t, err)
		require.Len(t, entries, 1)
		assert.Equal(t, e1.ID, entries[0].ID)
	})

	t.Run("filters by date range", func(t *testing.T) {
		cleanupTables(t, pool)

		now := time.Now().UTC()
		e1 := testEntry(func(e *domainEntry.Entry) {
			e.PostedAt = now.Add(-48 * time.Hour)
		})
		e2 := testEntry(func(e *domainEntry.Entry) {
			e.PostedAt = now.Add(-24 * time.Hour)
		})
		e3 := testEntry(func(e *domainEntry.Entry) {
			e.PostedAt = now
		})
		insertEntry(t, pool, e1)
		insertEntry(t, pool, e2)
		insertEntry(t, pool, e3)

		entries, err := repo.List(ctx, domainEntry.ListQuery{
			PostedAtFrom: now.Add(-36 * time.Hour),
			PostedAtTo:   now.Add(-12 * time.Hour),
		})
		require.NoError(t, err)
		require.Len(t, entries, 1)
		assert.Equal(t, e2.ID, entries[0].ID)
	})

	t.Run("filters by min bookmark count", func(t *testing.T) {
		cleanupTables(t, pool)

		e1 := testEntry(func(e *domainEntry.Entry) {
			e.BookmarkCount = 5
		})
		e2 := testEntry(func(e *domainEntry.Entry) {
			e.BookmarkCount = 50
		})
		e3 := testEntry(func(e *domainEntry.Entry) {
			e.BookmarkCount = 100
		})
		insertEntry(t, pool, e1)
		insertEntry(t, pool, e2)
		insertEntry(t, pool, e3)

		entries, err := repo.List(ctx, domainEntry.ListQuery{
			MinBookmarkCount: 50,
		})
		require.NoError(t, err)
		require.Len(t, entries, 2)
	})

	t.Run("sorts by hot (bookmark count)", func(t *testing.T) {
		cleanupTables(t, pool)

		e1 := testEntry(func(e *domainEntry.Entry) {
			e.BookmarkCount = 10
			e.PostedAt = time.Now().UTC()
		})
		e2 := testEntry(func(e *domainEntry.Entry) {
			e.BookmarkCount = 100
			e.PostedAt = time.Now().UTC().Add(-1 * time.Hour)
		})
		insertEntry(t, pool, e1)
		insertEntry(t, pool, e2)

		entries, err := repo.List(ctx, domainEntry.ListQuery{
			Sort: domainEntry.SortHot,
		})
		require.NoError(t, err)
		require.Len(t, entries, 2)

		// Should be ordered by bookmark_count DESC
		assert.Equal(t, e2.ID, entries[0].ID)
		assert.Equal(t, e1.ID, entries[1].ID)
	})

	t.Run("paginates results", func(t *testing.T) {
		cleanupTables(t, pool)

		// Insert 5 entries
		for i := 0; i < 5; i++ {
			e := testEntry(func(e *domainEntry.Entry) {
				e.PostedAt = time.Now().UTC().Add(-time.Duration(i) * time.Hour)
			})
			insertEntry(t, pool, e)
		}

		// Get first page
		page1, err := repo.List(ctx, domainEntry.ListQuery{
			Limit:  2,
			Offset: 0,
		})
		require.NoError(t, err)
		require.Len(t, page1, 2)

		// Get second page
		page2, err := repo.List(ctx, domainEntry.ListQuery{
			Limit:  2,
			Offset: 2,
		})
		require.NoError(t, err)
		require.Len(t, page2, 2)

		// Ensure different results
		assert.NotEqual(t, page1[0].ID, page2[0].ID)
	})

	t.Run("returns empty slice when no results", func(t *testing.T) {
		cleanupTables(t, pool)

		entries, err := repo.List(ctx, domainEntry.ListQuery{})
		require.NoError(t, err)
		require.Len(t, entries, 0)
	})
}

func TestEntryRepository_Count(t *testing.T) {
	pool, terminate := setupPostgres(t)
	defer terminate()

	ctx := context.Background()
	require.NoError(t, applyTestMigrations(ctx, pool))

	repo := NewEntryRepository(pool)

	t.Run("counts all entries", func(t *testing.T) {
		cleanupTables(t, pool)

		insertEntry(t, pool, testEntry())
		insertEntry(t, pool, testEntry())
		insertEntry(t, pool, testEntry())

		count, err := repo.Count(ctx, domainEntry.ListQuery{})
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("counts with filters", func(t *testing.T) {
		cleanupTables(t, pool)

		e1 := testEntry(func(e *domainEntry.Entry) {
			e.BookmarkCount = 10
		})
		e2 := testEntry(func(e *domainEntry.Entry) {
			e.BookmarkCount = 50
		})
		e3 := testEntry(func(e *domainEntry.Entry) {
			e.BookmarkCount = 100
		})
		insertEntry(t, pool, e1)
		insertEntry(t, pool, e2)
		insertEntry(t, pool, e3)

		count, err := repo.Count(ctx, domainEntry.ListQuery{
			MinBookmarkCount: 50,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})

	t.Run("returns zero when no results", func(t *testing.T) {
		cleanupTables(t, pool)

		count, err := repo.Count(ctx, domainEntry.ListQuery{})
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestEntryRepository_ListAndCount(t *testing.T) {
	pool, terminate := setupPostgres(t)
	defer terminate()

	ctx := context.Background()
	require.NoError(t, applyTestMigrations(ctx, pool))

	cleanupTables(t, pool)

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
		Excerpt:       "Entry1 excerpt",
		Subject:       "entry1",
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
		Excerpt:       "Entry2 excerpt",
		Subject:       "entry2",
		BookmarkCount: 10,
		PostedAt:      now,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, entry2))

	// attach tags
	_, err = pool.Exec(ctx, `INSERT INTO entry_tags (entry_id, tag_id, score) VALUES ($1, $2, $3)`, entry1.ID, goTag.ID, 0)
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

	t.Run("returns total count even when page is empty", func(t *testing.T) {
		cleanupTables(t, pool)

		// Insert 3 entries
		for i := 0; i < 3; i++ {
			insertEntry(t, pool, testEntry())
		}

		// Try to get page beyond results
		entries, total, err := repo.ListAndCount(ctx, domainEntry.ListQuery{
			Limit:  10,
			Offset: 10,
		})
		require.NoError(t, err)
		require.Len(t, entries, 0)
		assert.Equal(t, int64(3), total)
	})

	t.Run("applies filters to both list and count", func(t *testing.T) {
		cleanupTables(t, pool)

		for i := 0; i < 5; i++ {
			e := testEntry(func(e *domainEntry.Entry) {
				e.BookmarkCount = i * 10
			})
			insertEntry(t, pool, e)
		}

		entries, total, err := repo.ListAndCount(ctx, domainEntry.ListQuery{
			MinBookmarkCount: 20,
			Limit:            10,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		require.Len(t, entries, 3)
	})
}

func TestEntryRepository_ListArchiveCounts(t *testing.T) {
	pool, terminate := setupPostgres(t)
	defer terminate()

	ctx := context.Background()
	require.NoError(t, applyTestMigrations(ctx, pool))

	repo := NewEntryRepository(pool)

	t.Run("aggregates entries by date", func(t *testing.T) {
		cleanupTables(t, pool)

		now := time.Now().UTC()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

		// Insert entries on different days
		e1 := testEntry(func(e *domainEntry.Entry) {
			e.PostedAt = today
		})
		e2 := testEntry(func(e *domainEntry.Entry) {
			e.PostedAt = today
		})
		e3 := testEntry(func(e *domainEntry.Entry) {
			e.PostedAt = today.Add(-24 * time.Hour)
		})
		insertEntry(t, pool, e1)
		insertEntry(t, pool, e2)
		insertEntry(t, pool, e3)
		refreshArchiveCounts(t, pool)

		counts, err := repo.ListArchiveCounts(ctx, 0)
		require.NoError(t, err)
		require.Len(t, counts, 2)

		// Should be ordered by date DESC
		assert.Equal(t, today, counts[0].Date)
		assert.Equal(t, 2, counts[0].Count)
		assert.Equal(t, today.Add(-24*time.Hour), counts[1].Date)
		assert.Equal(t, 1, counts[1].Count)
	})

	t.Run("filters by min bookmark count", func(t *testing.T) {
		cleanupTables(t, pool)

		now := time.Now().UTC()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

		e1 := testEntry(func(e *domainEntry.Entry) {
			e.PostedAt = today
			e.BookmarkCount = 10
		})
		e2 := testEntry(func(e *domainEntry.Entry) {
			e.PostedAt = today
			e.BookmarkCount = 100
		})
		insertEntry(t, pool, e1)
		insertEntry(t, pool, e2)
		refreshArchiveCounts(t, pool)

		counts, err := repo.ListArchiveCounts(ctx, 50)
		require.NoError(t, err)
		require.Len(t, counts, 1)
		assert.Equal(t, 1, counts[0].Count)
	})

	t.Run("returns empty slice when no results", func(t *testing.T) {
		cleanupTables(t, pool)
		refreshArchiveCounts(t, pool)

		counts, err := repo.ListArchiveCounts(ctx, 0)
		require.NoError(t, err)
		require.Len(t, counts, 0)
	})

	t.Run("handles negative min bookmark count", func(t *testing.T) {
		cleanupTables(t, pool)

		insertEntry(t, pool, testEntry())
		refreshArchiveCounts(t, pool)

		counts, err := repo.ListArchiveCounts(ctx, -1)
		require.NoError(t, err)
		require.Len(t, counts, 1)
	})
}
