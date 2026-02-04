package ranking

import (
	"context"
	"time"

	domainEntry "hateblog/internal/domain/entry"
	"hateblog/internal/pkg/apptime"
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
	result, _, err := s.YearlyWithCacheStatus(ctx, year, limit, minUsers)
	return result, err
}

// YearlyWithCacheStatus returns ranking entries and cache hit info.
func (s *Service) YearlyWithCacheStatus(ctx context.Context, year, limit, minUsers int) (Result, bool, error) {
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
			return Result{}, false, err
		}
		if ok {
			return Result{
				Entries: sliceLimit(cached.Entries, limit),
				Total:   cached.Total,
			}, true, nil
		}
	}
	from, to, err := apptime.YearRange(year)
	if err != nil {
		return Result{}, false, err
	}
	all, total, err := s.listEntriesAndCount(ctx, from, to, max, minUsers)
	if err != nil {
		return Result{}, false, err
	}
	if s.yearlyCache != nil {
		_ = s.yearlyCache.Set(ctx, year, minUsers, rankingCachePayload{Entries: all, Total: total})
	}
	return Result{
		Entries: sliceLimit(all, limit),
		Total:   total,
	}, false, nil
}

// Monthly returns ranking entries for the given year/month.
func (s *Service) Monthly(ctx context.Context, year, month, limit, minUsers int) (Result, error) {
	result, _, err := s.MonthlyWithCacheStatus(ctx, year, month, limit, minUsers)
	return result, err
}

// MonthlyWithCacheStatus returns ranking entries and cache hit info.
func (s *Service) MonthlyWithCacheStatus(ctx context.Context, year, month, limit, minUsers int) (Result, bool, error) {
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
			return Result{}, false, err
		}
		if ok {
			return Result{
				Entries: sliceLimit(cached.Entries, limit),
				Total:   cached.Total,
			}, true, nil
		}
	}
	from, to, err := apptime.MonthRange(year, month)
	if err != nil {
		return Result{}, false, err
	}
	all, total, err := s.listEntriesAndCount(ctx, from, to, max, minUsers)
	if err != nil {
		return Result{}, false, err
	}
	if s.monthlyCache != nil {
		_ = s.monthlyCache.Set(ctx, year, month, minUsers, rankingCachePayload{Entries: all, Total: total})
	}
	return Result{
		Entries: sliceLimit(all, limit),
		Total:   total,
	}, false, nil
}

// Weekly returns ranking entries for the given ISO week.
func (s *Service) Weekly(ctx context.Context, year, week, limit, minUsers int) (Result, error) {
	result, _, err := s.WeeklyWithCacheStatus(ctx, year, week, limit, minUsers)
	return result, err
}

// WeeklyWithCacheStatus returns ranking entries and cache hit info.
func (s *Service) WeeklyWithCacheStatus(ctx context.Context, year, week, limit, minUsers int) (Result, bool, error) {
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
			return Result{}, false, err
		}
		if ok {
			return Result{
				Entries: sliceLimit(cached.Entries, limit),
				Total:   cached.Total,
			}, true, nil
		}
	}
	from, to, err := apptime.ISOWeekRange(year, week)
	if err != nil {
		return Result{}, false, err
	}
	all, total, err := s.listEntriesAndCount(ctx, from, to, max, minUsers)
	if err != nil {
		return Result{}, false, err
	}
	if s.weeklyCache != nil {
		_ = s.weeklyCache.Set(ctx, year, week, minUsers, rankingCachePayload{Entries: all, Total: total})
	}
	return Result{
		Entries: sliceLimit(all, limit),
		Total:   total,
	}, false, nil
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

func sliceLimit(entries []*domainEntry.Entry, limit int) []*domainEntry.Entry {
	if limit <= 0 {
		return entries
	}
	if limit >= len(entries) {
		return entries
	}
	return entries[:limit]
}
