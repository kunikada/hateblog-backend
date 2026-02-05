package archive

import (
	"context"
	"time"

	domainArchive "hateblog/internal/domain/archive"
	"hateblog/internal/domain/repository"
)

// Repository defines entry aggregation operations required by the service.
type Repository interface {
	ListArchiveCounts(ctx context.Context, minBookmarkCount int) ([]repository.ArchiveCount, error)
}

// Service exposes archive use cases.
type Service struct {
	repo     Repository
	cache    Cache
	timeFunc func() time.Time
}

// Cache stores archive aggregates with separate today/past data.
type Cache interface {
	GetToday(ctx context.Context, minUsers int, out any) (bool, error)
	SetToday(ctx context.Context, minUsers int, value any) error
	GetPast(ctx context.Context, minUsers int, out any) (bool, error)
	SetPast(ctx context.Context, minUsers int, value any) error
}

// NewService builds an archive service.
func NewService(repo Repository, cache Cache) *Service {
	return &Service{repo: repo, cache: cache, timeFunc: time.Now}
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

	if s.cache == nil {
		items, err := s.repo.ListArchiveCounts(ctx, minBookmarkCount)
		if err != nil {
			return nil, false, err
		}
		return items, false, nil
	}

	// Try to get from cache (today + past)
	items, cacheHit, err := s.getFromCache(ctx, minBookmarkCount)
	if err != nil {
		return nil, false, err
	}
	if cacheHit {
		return items, true, nil
	}

	// Cache miss: fetch from DB and split into today/past
	items, err = s.repo.ListArchiveCounts(ctx, minBookmarkCount)
	if err != nil {
		return nil, false, err
	}

	// Store in cache (ignore errors)
	_ = s.storeInCache(ctx, minBookmarkCount, items)

	return items, false, nil
}

// getFromCache retrieves today and past data from cache and merges them.
func (s *Service) getFromCache(ctx context.Context, minUsers int) ([]repository.ArchiveCount, bool, error) {
	today := s.todayDate()

	var todayData *repository.ArchiveCount
	ok, err := s.cache.GetToday(ctx, minUsers, &todayData)
	if err != nil {
		return nil, false, err
	}
	todayHit := ok

	var pastData []repository.ArchiveCount
	ok, err = s.cache.GetPast(ctx, minUsers, &pastData)
	if err != nil {
		return nil, false, err
	}
	pastHit := ok

	// Both must hit for a complete cache hit
	if !todayHit || !pastHit {
		return nil, false, nil
	}

	// Merge: today first (if exists and matches today's date), then past
	var result []repository.ArchiveCount
	if todayData != nil && s.isSameDate(todayData.Date, today) {
		result = append(result, *todayData)
	}
	result = append(result, pastData...)

	return result, true, nil
}

// storeInCache splits items into today/past and stores separately.
func (s *Service) storeInCache(ctx context.Context, minUsers int, items []repository.ArchiveCount) error {
	today := s.todayDate()

	var todayData *repository.ArchiveCount
	var pastData []repository.ArchiveCount

	for i := range items {
		if s.isSameDate(items[i].Date, today) {
			todayData = &items[i]
		} else {
			pastData = append(pastData, items[i])
		}
	}

	if err := s.cache.SetToday(ctx, minUsers, todayData); err != nil {
		return err
	}
	if err := s.cache.SetPast(ctx, minUsers, pastData); err != nil {
		return err
	}
	return nil
}

// todayDate returns today's date at midnight in JST.
func (s *Service) todayDate() time.Time {
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	now := s.timeFunc().In(jst)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// isSameDate checks if two times represent the same calendar date.
func (s *Service) isSameDate(t1, t2 time.Time) bool {
	return t1.Year() == t2.Year() && t1.Month() == t2.Month() && t1.Day() == t2.Day()
}
