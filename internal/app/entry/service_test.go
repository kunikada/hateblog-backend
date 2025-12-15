package entry

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	domainEntry "hateblog/internal/domain/entry"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type stubEntryRepo struct {
	listResult  []*domainEntry.Entry
	listErr     error
	countResult int64
	countErr    error

	listCalls  int
	countCalls int
}

func (s *stubEntryRepo) Get(ctx context.Context, id domainEntry.ID) (*domainEntry.Entry, error) {
	return nil, nil
}
func (s *stubEntryRepo) List(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, error) {
	s.listCalls++
	return s.listResult, s.listErr
}
func (s *stubEntryRepo) Count(ctx context.Context, query domainEntry.ListQuery) (int64, error) {
	s.countCalls++
	return s.countResult, s.countErr
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

type stubCache struct {
	store map[string][]byte
}

func newStubCache() *stubCache {
	return &stubCache{store: make(map[string][]byte)}
}

func (c *stubCache) BuildKey(query domainEntry.ListQuery) (string, error) {
	return string(query.Sort), nil
}

func (c *stubCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	v, ok := c.store[key]
	return v, ok, nil
}

func (c *stubCache) Set(ctx context.Context, key string, payload []byte) error {
	c.store[key] = payload
	return nil
}

func TestListNewEntriesUsesCache(t *testing.T) {
	cache := newStubCache()
	result := ListResult{
		Entries: []*domainEntry.Entry{{
			ID:            uuid.New(),
			Title:         "title",
			URL:           "https://example.com",
			BookmarkCount: 10,
			PostedAt:      time.Now(),
			Tags:          []domainEntry.Tagging{},
		}},
		Total: 100,
	}
	payload, err := json.Marshal(result)
	require.NoError(t, err)
	cache.store["new"] = payload

	repo := &stubEntryRepo{}
	svc := NewService(repo, cache, nil)

	out, err := svc.ListNewEntries(context.Background(), ListParams{})
	require.NoError(t, err)
	require.Equal(t, result.Total, out.Total)
	require.Len(t, out.Entries, len(result.Entries))
	require.Equal(t, result.Entries[0].ID, out.Entries[0].ID)
	require.Equal(t, result.Entries[0].Title, out.Entries[0].Title)
	require.Equal(t, 0, repo.listCalls)
	require.Equal(t, 0, repo.countCalls)
}

func TestListHotEntriesStoresCache(t *testing.T) {
	cache := newStubCache()
	entries := []*domainEntry.Entry{{
		ID:            uuid.New(),
		Title:         "title",
		URL:           "https://example.com",
		BookmarkCount: 20,
		PostedAt:      time.Now(),
	}}
	repo := &stubEntryRepo{
		listResult:  entries,
		countResult: 10,
	}
	svc := NewService(repo, cache, nil)

	result, err := svc.ListHotEntries(context.Background(), ListParams{MinBookmarkCount: 5})
	require.NoError(t, err)
	require.Equal(t, entries, result.Entries)
	require.Equal(t, int64(10), result.Total)
	require.Equal(t, 1, repo.listCalls)
	require.Equal(t, 1, repo.countCalls)
	require.Contains(t, cache.store, "hot")
}
