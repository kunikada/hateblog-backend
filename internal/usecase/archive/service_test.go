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

func TestServiceListRejectsInvalidMinUsers(t *testing.T) {
	items := []repository.ArchiveCount{
		{Date: time.Now(), Count: 10},
	}
	svc := NewService(&stubRepo{items: items}, nil)

	_, err := svc.List(context.Background(), 7)
	require.Error(t, err)
}
