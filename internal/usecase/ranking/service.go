package ranking

import (
	"context"
	"fmt"
	"time"

	domainEntry "hateblog/internal/domain/entry"
)

// Repository describes entry operations required for ranking computations.
type Repository interface {
	List(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, error)
	Count(ctx context.Context, query domainEntry.ListQuery) (int64, error)
}

// Service provides ranking queries.
type Service struct {
	repo         Repository
	yearlyCache  CacheYearly
	monthlyCache CacheMonthly
	weeklyCache  CacheWeekly
}

// Result bundles ranking entries and totals.
type Result struct {
	Entries []*domainEntry.Entry
	Total   int64
}

// NewService creates a ranking service.
func NewService(repo Repository, yearly CacheYearly, monthly CacheMonthly, weekly CacheWeekly) *Service {
	return &Service{
		repo:         repo,
		yearlyCache:  yearly,
		monthlyCache: monthly,
		weeklyCache:  weekly,
	}
}

// Yearly returns ranking entries for the given year.
func (s *Service) Yearly(ctx context.Context, year, limit, minUsers int) (Result, error) {
	if limit <= 0 {
		limit = domainEntry.DefaultLimit
	}
	if minUsers < 0 {
		minUsers = 0
	}
	const max = 1000
	if s.yearlyCache != nil {
		var cached rankingCachePayload
		ok, err := s.yearlyCache.Get(ctx, year, minUsers, &cached)
		if err != nil {
			return Result{}, err
		}
		if ok {
			return Result{
				Entries: sliceLimit(cached.Entries, limit),
				Total:   cached.Total,
			}, nil
		}
	}
	from, to, err := yearRange(year)
	if err != nil {
		return Result{}, err
	}
	all, total, err := s.listEntriesAndCount(ctx, from, to, max, minUsers)
	if err != nil {
		return Result{}, err
	}
	if s.yearlyCache != nil {
		_ = s.yearlyCache.Set(ctx, year, minUsers, rankingCachePayload{Entries: all, Total: total})
	}
	return Result{
		Entries: sliceLimit(all, limit),
		Total:   total,
	}, nil
}

// Monthly returns ranking entries for the given year/month.
func (s *Service) Monthly(ctx context.Context, year, month, limit, minUsers int) (Result, error) {
	if limit <= 0 {
		limit = domainEntry.DefaultLimit
	}
	if minUsers < 0 {
		minUsers = 0
	}
	const max = 100
	if s.monthlyCache != nil {
		var cached rankingCachePayload
		ok, err := s.monthlyCache.Get(ctx, year, month, minUsers, &cached)
		if err != nil {
			return Result{}, err
		}
		if ok {
			return Result{
				Entries: sliceLimit(cached.Entries, limit),
				Total:   cached.Total,
			}, nil
		}
	}
	from, to, err := monthRange(year, month)
	if err != nil {
		return Result{}, err
	}
	all, total, err := s.listEntriesAndCount(ctx, from, to, max, minUsers)
	if err != nil {
		return Result{}, err
	}
	if s.monthlyCache != nil {
		_ = s.monthlyCache.Set(ctx, year, month, minUsers, rankingCachePayload{Entries: all, Total: total})
	}
	return Result{
		Entries: sliceLimit(all, limit),
		Total:   total,
	}, nil
}

// Weekly returns ranking entries for the given ISO week.
func (s *Service) Weekly(ctx context.Context, year, week, limit, minUsers int) (Result, error) {
	if limit <= 0 {
		limit = domainEntry.DefaultLimit
	}
	if minUsers < 0 {
		minUsers = 0
	}
	const max = 100
	if s.weeklyCache != nil {
		var cached rankingCachePayload
		ok, err := s.weeklyCache.Get(ctx, year, week, minUsers, &cached)
		if err != nil {
			return Result{}, err
		}
		if ok {
			return Result{
				Entries: sliceLimit(cached.Entries, limit),
				Total:   cached.Total,
			}, nil
		}
	}
	from, to, err := isoWeekRange(year, week)
	if err != nil {
		return Result{}, err
	}
	all, total, err := s.listEntriesAndCount(ctx, from, to, max, minUsers)
	if err != nil {
		return Result{}, err
	}
	if s.weeklyCache != nil {
		_ = s.weeklyCache.Set(ctx, year, week, minUsers, rankingCachePayload{Entries: all, Total: total})
	}
	return Result{
		Entries: sliceLimit(all, limit),
		Total:   total,
	}, nil
}

// CacheYearly stores yearly ranking payloads.
type CacheYearly interface {
	Get(ctx context.Context, year, minUsers int, out any) (bool, error)
	Set(ctx context.Context, year, minUsers int, value any) error
}

// CacheMonthly stores monthly ranking payloads.
type CacheMonthly interface {
	Get(ctx context.Context, year, month, minUsers int, out any) (bool, error)
	Set(ctx context.Context, year, month, minUsers int, value any) error
}

// CacheWeekly stores weekly ranking payloads.
type CacheWeekly interface {
	Get(ctx context.Context, year, week, minUsers int, out any) (bool, error)
	Set(ctx context.Context, year, week, minUsers int, value any) error
}

type rankingCachePayload struct {
	Entries []*domainEntry.Entry `json:"entries"`
	Total   int64                `json:"total"`
}

func (s *Service) listEntriesAndCount(ctx context.Context, from, to time.Time, limit, minUsers int) ([]*domainEntry.Entry, int64, error) {
	query := domainEntry.ListQuery{
		Sort:             domainEntry.SortHot,
		Limit:            limit,
		MaxLimitOverride: limit,
		PostedAtFrom:     from,
		PostedAtTo:       to,
		MinBookmarkCount: minUsers,
	}
	entries, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.Count(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

func yearRange(year int) (time.Time, time.Time, error) {
	if year < 2000 || year > 9999 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid year: %d", year)
	}
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	return start, start.AddDate(1, 0, 0), nil
}

func sliceLimit(entries []*domainEntry.Entry, limit int) []*domainEntry.Entry {
	if limit <= 0 {
		return entries
	}
	if limit >= len(entries) {
		return entries
	}
	return entries[:limit]
}

func monthRange(year, month int) (time.Time, time.Time, error) {
	if month < 1 || month > 12 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid month: %d", month)
	}
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	return start, start.AddDate(0, 1, 0), nil
}

func isoWeekRange(year, week int) (time.Time, time.Time, error) {
	if week < 1 || week > 53 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid week: %d", week)
	}
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)
	weekday := int(jan4.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := jan4.AddDate(0, 0, -(weekday - 1))
	start = start.AddDate(0, 0, (week-1)*7)
	isoYear, isoWeek := start.ISOWeek()
	if isoYear != year || isoWeek != week {
		return time.Time{}, time.Time{}, fmt.Errorf("week %d out of range for year %d", week, year)
	}
	return start, start.AddDate(0, 0, 7), nil
}
