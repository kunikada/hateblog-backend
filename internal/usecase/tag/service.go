package tag

import (
	"context"
	"fmt"
	"time"

	"hateblog/internal/domain/tag"
)

// Repository describes DB operations required by the tag service.
type Repository interface {
	GetByName(ctx context.Context, name string) (*tag.Tag, error)
	List(ctx context.Context, limit, offset int) ([]tag.Tag, error)
	IncrementViewHistory(ctx context.Context, tagID tag.ID, viewedAt time.Time) error
}

// Service exposes tag operations.
type Service struct {
	repo  Repository
	cache ListCache
}

// ListCache stores tag list payloads.
type ListCache interface {
	Get(ctx context.Context, limit, offset int, out any) (bool, error)
	Set(ctx context.Context, limit, offset int, value any) error
}

// NewService builds a tag service.
func NewService(repo Repository, cache ListCache) *Service {
	return &Service{repo: repo, cache: cache}
}

// GetByName returns tag metadata.
func (s *Service) GetByName(ctx context.Context, name string) (*tag.Tag, error) {
	norm := tag.NormalizeName(name)
	if norm == "" {
		return nil, fmt.Errorf("tag is required")
	}
	return s.repo.GetByName(ctx, norm)
}

// List returns tags sorted by name.
func (s *Service) List(ctx context.Context, limit, offset int) ([]tag.Tag, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	if s.cache != nil {
		var cached []tag.Tag
		ok, err := s.cache.Get(ctx, limit, offset, &cached)
		if err != nil {
			return nil, err
		}
		if ok {
			return cached, nil
		}
	}
	tags, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	if s.cache != nil {
		if err := s.cache.Set(ctx, limit, offset, tags); err != nil {
			return tags, nil
		}
	}
	return tags, nil
}

// RecordView increments the view counter for the tag.
func (s *Service) RecordView(ctx context.Context, tagID tag.ID, viewedAt time.Time) error {
	return s.repo.IncrementViewHistory(ctx, tagID, viewedAt)
}
