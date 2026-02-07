package ranking

import (
	"context"
	"testing"
	"time"

	domainEntry "hateblog/internal/domain/entry"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type stubEntryRepo struct {
	lastQuery domainEntry.ListQuery
}

func (s *stubEntryRepo) List(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, error) {
	s.lastQuery = query
	return []*domainEntry.Entry{{
		ID:            uuid.New(),
		Title:         "example",
		URL:           "https://example.com",
		BookmarkCount: 10,
		PostedAt:      time.Now(),
	}}, nil
}

func (s *stubEntryRepo) Count(ctx context.Context, query domainEntry.ListQuery) (int64, error) {
	return 1, nil
}

func TestYearlyRankingClampsLimit(t *testing.T) {
	repo := &stubEntryRepo{}
	svc := NewService(repo, nil, nil, nil)

	result, err := svc.Yearly(context.Background(), 2024, 1000, 0, -5)
	require.NoError(t, err)
	require.Len(t, result.Entries, 1)
	require.Equal(t, 1000, repo.lastQuery.Limit)
	require.Equal(t, 100, repo.lastQuery.MaxLimitOverride)
	require.Equal(t, 0, repo.lastQuery.MinBookmarkCount)
	require.False(t, repo.lastQuery.PostedAtFrom.IsZero())
	require.False(t, repo.lastQuery.PostedAtTo.IsZero())
}

func TestWeeklyRankingRejectsInvalidWeek(t *testing.T) {
	repo := &stubEntryRepo{}
	svc := NewService(repo, nil, nil, nil)

	_, err := svc.Weekly(context.Background(), 2024, 54, 10, 0, 0)
	require.Error(t, err)
}
