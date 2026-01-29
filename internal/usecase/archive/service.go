package archive

import (
	"context"

	domainArchive "hateblog/internal/domain/archive"
	"hateblog/internal/domain/repository"
)

// Repository defines entry aggregation operations required by the service.
type Repository interface {
	ListArchiveCounts(ctx context.Context, minBookmarkCount int) ([]repository.ArchiveCount, error)
}

// Service exposes archive use cases.
type Service struct {
	repo  Repository
	cache Cache
}

// Cache stores archive aggregates.
type Cache interface {
	Get(ctx context.Context, minUsers int, out any) (bool, error)
	Set(ctx context.Context, minUsers int, value any) error
}

// NewService builds an archive service.
func NewService(repo Repository, cache Cache) *Service {
	return &Service{repo: repo, cache: cache}
}

// List returns aggregated counts sorted by date desc.
func (s *Service) List(ctx context.Context, minBookmarkCount int) ([]repository.ArchiveCount, error) {
	items, _, err := s.ListWithCacheStatus(ctx, minBookmarkCount)
	return items, err
}

// ListWithCacheStatus returns aggregates with cache hit info for diagnostics.
func (s *Service) ListWithCacheStatus(ctx context.Context, minBookmarkCount int) ([]repository.ArchiveCount, bool, error) {
	if err := domainArchive.ValidateMinUsers(minBookmarkCount); err != nil {
		return nil, false, err
	}
	if s.cache != nil {
		var cached []repository.ArchiveCount
		ok, err := s.cache.Get(ctx, minBookmarkCount, &cached)
		if err != nil {
			return nil, false, err
		}
		if ok {
			return cached, true, nil
		}
	}
	items, err := s.repo.ListArchiveCounts(ctx, minBookmarkCount)
	if err != nil {
		return nil, false, err
	}
	if s.cache != nil {
		if err := s.cache.Set(ctx, minBookmarkCount, items); err != nil {
			return items, false, nil
		}
	}
	return items, false, nil
}
