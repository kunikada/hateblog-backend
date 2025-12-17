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
	repo Repository
}

// NewService builds a tag service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
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
	return s.repo.List(ctx, limit, offset)
}

// RecordView increments the view counter for the tag.
func (s *Service) RecordView(ctx context.Context, tagID tag.ID, viewedAt time.Time) error {
	return s.repo.IncrementViewHistory(ctx, tagID, viewedAt)
}
