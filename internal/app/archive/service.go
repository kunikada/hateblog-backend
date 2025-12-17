package archive

import (
	"context"

	"hateblog/internal/domain/repository"
)

// Repository defines entry aggregation operations required by the service.
type Repository interface {
	ListArchiveCounts(ctx context.Context, minBookmarkCount int) ([]repository.ArchiveCount, error)
}

// Service exposes archive use cases.
type Service struct {
	repo Repository
}

// NewService builds an archive service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List returns aggregated counts sorted by date desc.
func (s *Service) List(ctx context.Context, minBookmarkCount int) ([]repository.ArchiveCount, error) {
	if minBookmarkCount < 0 {
		minBookmarkCount = 0
	}
	return s.repo.ListArchiveCounts(ctx, minBookmarkCount)
}
