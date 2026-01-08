package metrics

import (
	"context"
	"fmt"
	"time"

	domainEntry "hateblog/internal/domain/entry"
	"hateblog/internal/pkg/timeutil"
)

// EntryRepository provides entry metadata lookup.
type EntryRepository interface {
	Get(ctx context.Context, id domainEntry.ID) (*domainEntry.Entry, error)
}

// ClickRepository stores click counts.
type ClickRepository interface {
	Increment(ctx context.Context, entryID domainEntry.ID, clickedAt time.Time) error
}

// Service records click metrics.
type Service struct {
	entries EntryRepository
	clicks  ClickRepository
}

// NewService builds a metrics service.
func NewService(entries EntryRepository, clicks ClickRepository) *Service {
	return &Service{
		entries: entries,
		clicks:  clicks,
	}
}

// RecordClick validates entry existence and increments click count.
func (s *Service) RecordClick(ctx context.Context, id domainEntry.ID) error {
	if s.entries == nil || s.clicks == nil {
		return fmt.Errorf("metrics service not initialized")
	}
	if id == (domainEntry.ID{}) {
		return fmt.Errorf("entry_id is required")
	}
	if _, err := s.entries.Get(ctx, id); err != nil {
		return fmt.Errorf("entry not found: %w", err)
	}
	if err := s.clicks.Increment(ctx, id, timeutil.Now()); err != nil {
		return err
	}
	return nil
}
