package archive

import (
	"context"
	"testing"
	"time"

	"hateblog/internal/domain/repository"

	"github.com/stretchr/testify/require"
)

type stubRepo struct {
	items []repository.ArchiveCount
}

func (s *stubRepo) ListArchiveCounts(ctx context.Context, minBookmarkCount int) ([]repository.ArchiveCount, error) {
	return s.items, nil
}

func TestServiceListNormalizesMinUsers(t *testing.T) {
	items := []repository.ArchiveCount{
		{Date: time.Now(), Count: 10},
	}
	svc := NewService(&stubRepo{items: items}, nil)

	got, err := svc.List(context.Background(), -10)
	require.NoError(t, err)
	require.Equal(t, items, got)
}
