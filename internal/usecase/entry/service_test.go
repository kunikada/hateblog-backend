package entry

import (
	"context"
	"testing"
	"time"

	domainEntry "hateblog/internal/domain/entry"
	"hateblog/internal/domain/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type stubEntryRepo struct {
	listResult []*domainEntry.Entry
	listErr    error

	listCalls int
}

func (s *stubEntryRepo) Get(ctx context.Context, id domainEntry.ID) (*domainEntry.Entry, error) {
	return nil, nil
}
func (s *stubEntryRepo) List(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, error) {
	s.listCalls++
	return s.listResult, s.listErr
}
func (s *stubEntryRepo) Count(ctx context.Context, query domainEntry.ListQuery) (int64, error) {
	return 0, nil
}
func (s *stubEntryRepo) Create(ctx context.Context, entry *domainEntry.Entry) error {
	return nil
}
func (s *stubEntryRepo) Update(ctx context.Context, entry *domainEntry.Entry) error {
	return nil
}
func (s *stubEntryRepo) Delete(ctx context.Context, id domainEntry.ID) error {
	return nil
}
func (s *stubEntryRepo) ListArchiveCounts(ctx context.Context, minBookmarkCount int) ([]repository.ArchiveCount, error) {
	return nil, nil
}

type stubDayCache struct {
	store    map[string][]*domainEntry.Entry
	getCalls int
	setCalls int
}

func newStubDayCache() *stubDayCache {
	return &stubDayCache{store: make(map[string][]*domainEntry.Entry)}
}

func (c *stubDayCache) Get(ctx context.Context, date string) ([]*domainEntry.Entry, bool, error) {
	c.getCalls++
	v, ok := c.store[date]
	return v, ok, nil
}

func (c *stubDayCache) Set(ctx context.Context, date string, entries []*domainEntry.Entry) error {
	c.setCalls++
	c.store[date] = entries
	return nil
}

type stubTagCache struct {
	store map[string][]*domainEntry.Entry
}

func (c *stubTagCache) Get(ctx context.Context, tag string) ([]*domainEntry.Entry, bool, error) {
	v, ok := c.store[tag]
	return v, ok, nil
}

func (c *stubTagCache) Set(ctx context.Context, tag string, entries []*domainEntry.Entry) error {
	c.store[tag] = entries
	return nil
}

func TestListNewEntriesUsesDayCache(t *testing.T) {
	dayCache := newStubDayCache()
	tagCache := &stubTagCache{store: map[string][]*domainEntry.Entry{}}
	now := time.Now()
	dayCache.store["20250105"] = []*domainEntry.Entry{{
		ID:            uuid.New(),
		Title:         "cached",
		URL:           "https://example.com",
		BookmarkCount: 10,
		PostedAt:      now,
	}}

	repo := &stubEntryRepo{}
	svc := NewService(repo, dayCache, tagCache, nil)

	out, err := svc.ListNewEntries(context.Background(), DayListParams{
		Date:             "20250105",
		MinBookmarkCount: 0,
		Limit:            25,
		Offset:           0,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), out.Total)
	require.Len(t, out.Entries, 1)
	require.Equal(t, "cached", out.Entries[0].Title)
	require.Equal(t, 0, repo.listCalls)
	require.Equal(t, 1, dayCache.getCalls)
}

func TestListHotEntriesStoresDayCacheAndSorts(t *testing.T) {
	dayCache := newStubDayCache()
	tagCache := &stubTagCache{store: map[string][]*domainEntry.Entry{}}
	now := time.Now()
	entries := []*domainEntry.Entry{
		{
			ID:            uuid.New(),
			Title:         "a",
			URL:           "https://example.com/a",
			BookmarkCount: 10,
			PostedAt:      now.Add(-1 * time.Hour),
		},
		{
			ID:            uuid.New(),
			Title:         "b",
			URL:           "https://example.com/b",
			BookmarkCount: 50,
			PostedAt:      now,
		},
	}
	repo := &stubEntryRepo{listResult: entries}
	svc := NewService(repo, dayCache, tagCache, nil)

	out, err := svc.ListHotEntries(context.Background(), DayListParams{
		Date:             "20250105",
		MinBookmarkCount: 5,
		Limit:            25,
		Offset:           0,
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), out.Total)
	require.Len(t, out.Entries, 2)
	require.Equal(t, "b", out.Entries[0].Title)
	require.Equal(t, 1, repo.listCalls)
	require.Equal(t, 1, dayCache.setCalls)
	require.Contains(t, dayCache.store, "20250105")
}
