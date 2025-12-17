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
	repo Repository
}

// Result bundles ranking entries and totals.
type Result struct {
	Entries []*domainEntry.Entry
	Total   int64
}

// NewService creates a ranking service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Yearly returns ranking entries for the given year.
func (s *Service) Yearly(ctx context.Context, year, limit, minUsers int) (Result, error) {
	from, to, err := yearRange(year)
	if err != nil {
		return Result{}, err
	}
	return s.list(ctx, from, to, limit, minUsers)
}

// Monthly returns ranking entries for the given year/month.
func (s *Service) Monthly(ctx context.Context, year, month, limit, minUsers int) (Result, error) {
	from, to, err := monthRange(year, month)
	if err != nil {
		return Result{}, err
	}
	return s.list(ctx, from, to, limit, minUsers)
}

// Weekly returns ranking entries for the given ISO week.
func (s *Service) Weekly(ctx context.Context, year, week, limit, minUsers int) (Result, error) {
	from, to, err := isoWeekRange(year, week)
	if err != nil {
		return Result{}, err
	}
	return s.list(ctx, from, to, limit, minUsers)
}

func (s *Service) list(ctx context.Context, from, to time.Time, limit, minUsers int) (Result, error) {
	if limit <= 0 {
		limit = domainEntry.DefaultLimit
	}
	if minUsers < 0 {
		minUsers = 0
	}
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
		return Result{}, err
	}
	total, err := s.repo.Count(ctx, query)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Entries: entries,
		Total:   total,
	}, nil
}

func yearRange(year int) (time.Time, time.Time, error) {
	if year < 2000 || year > 9999 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid year: %d", year)
	}
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	return start, start.AddDate(1, 0, 0), nil
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
