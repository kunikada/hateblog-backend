package search

import (
	"context"
	"errors"
	"testing"
	"time"

	domainEntry "hateblog/internal/domain/entry"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeEntryRepo struct {
	lastQuery domainEntry.ListQuery
}

func (f *fakeEntryRepo) List(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, error) {
	f.lastQuery = query
	return []*domainEntry.Entry{{
		ID:            uuid.New(),
		Title:         "match",
		URL:           "https://example.com",
		BookmarkCount: 10,
		PostedAt:      time.Now(),
	}}, nil
}

func (f *fakeEntryRepo) Count(ctx context.Context, query domainEntry.ListQuery) (int64, error) {
	return 1, nil
}

type fakeHistory struct {
	err error
}

func (f *fakeHistory) Record(ctx context.Context, query string, searchedAt time.Time) error {
	return f.err
}

func TestSearchValidatesQuery(t *testing.T) {
	repo := &fakeEntryRepo{}
	history := &fakeHistory{err: errors.New("boom")}
	svc := NewService(repo, history, nil)

	_, err := svc.Search(context.Background(), " ", Params{})
	require.Error(t, err)

	result, err := svc.Search(context.Background(), "Go", Params{Limit: 200})
	require.NoError(t, err)
	require.Equal(t, "Go", result.Query)
	require.Equal(t, domainEntry.MaxLimit, repo.lastQuery.Limit)
}
