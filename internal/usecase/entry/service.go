package entry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	domainEntry "hateblog/internal/domain/entry"
	"hateblog/internal/domain/repository"
)

type DayEntriesCache interface {
	Get(ctx context.Context, date string) ([]*domainEntry.Entry, bool, error)
	Set(ctx context.Context, date string, entries []*domainEntry.Entry) error
}

type TagEntriesCache interface {
	Get(ctx context.Context, tagName string) ([]*domainEntry.Entry, bool, error)
	Set(ctx context.Context, tagName string, entries []*domainEntry.Entry) error
}

// Service orchestrates entry use cases.
type Service struct {
	repo          repository.EntryRepository
	dayCache      DayEntriesCache
	tagEntries    TagEntriesCache
	logger        *slog.Logger
	jstLocation   *time.Location
	maxAllResults int
}

// ListResult represents query outcome.
type ListResult struct {
	Entries []*domainEntry.Entry `json:"entries"`
	Total   int64                `json:"total"`
}

// DayListParams represents user filters for /entries endpoints.
type DayListParams struct {
	Date             string
	MinBookmarkCount int
	Offset           int
	Limit            int
}

// TagListParams represents user filters for /tags/{tag}/entries.
type TagListParams struct {
	MinBookmarkCount int
	Offset           int
	Limit            int
}

// NewService instantiates the service.
func NewService(repo repository.EntryRepository, dayCache DayEntriesCache, tagEntriesCache TagEntriesCache, logger *slog.Logger) *Service {
	jst, _ := time.LoadLocation("Asia/Tokyo")
	return &Service{
		repo:          repo,
		dayCache:      dayCache,
		tagEntries:    tagEntriesCache,
		logger:        logger,
		jstLocation:   jst,
		maxAllResults: 100000,
	}
}

// ListNewEntries returns entries ordered by posted_at DESC.
func (s *Service) ListNewEntries(ctx context.Context, params DayListParams) (ListResult, error) {
	return s.listDayEntries(ctx, domainEntry.SortNew, params)
}

// ListHotEntries returns entries ordered by bookmark count.
func (s *Service) ListHotEntries(ctx context.Context, params DayListParams) (ListResult, error) {
	return s.listDayEntries(ctx, domainEntry.SortHot, params)
}

// ListTagEntries returns tag entries ordered by posted_at DESC.
func (s *Service) ListTagEntries(ctx context.Context, tagName string, params TagListParams) (ListResult, error) {
	if tagName == "" {
		return ListResult{}, fmt.Errorf("tag is required")
	}
	all, err := s.loadAllTagEntries(ctx, tagName)
	if err != nil {
		return ListResult{}, err
	}
	filtered := filterByMinUsers(all, params.MinBookmarkCount)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].PostedAt.After(filtered[j].PostedAt)
	})
	total := int64(len(filtered))
	paged := paginate(filtered, params.Offset, params.Limit)
	return ListResult{Entries: paged, Total: total}, nil
}

func (s *Service) listDayEntries(ctx context.Context, sortType domainEntry.SortType, params DayListParams) (ListResult, error) {
	var empty ListResult
	if params.Date == "" {
		return empty, fmt.Errorf("date is required")
	}
	all, err := s.loadAllDayEntries(ctx, params.Date)
	if err != nil {
		return empty, err
	}
	filtered := filterByMinUsers(all, params.MinBookmarkCount)
	switch sortType {
	case domainEntry.SortHot:
		sort.Slice(filtered, func(i, j int) bool {
			if filtered[i].BookmarkCount != filtered[j].BookmarkCount {
				return filtered[i].BookmarkCount > filtered[j].BookmarkCount
			}
			return filtered[i].PostedAt.After(filtered[j].PostedAt)
		})
	default:
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].PostedAt.After(filtered[j].PostedAt)
		})
	}
	total := int64(len(filtered))
	paged := paginate(filtered, params.Offset, params.Limit)
	return ListResult{Entries: paged, Total: total}, nil
}

func (s *Service) logDebug(msg string, err error) {
	if s.logger == nil || err == nil {
		return
	}
	var attrs []any
	var unwrapped error = err
	for {
		if unwrapped == nil {
			break
		}
		attrs = append(attrs, "error", unwrapped.Error())
		unwrapped = errors.Unwrap(unwrapped)
	}
	s.logger.Debug(msg, attrs...)
}

func (s *Service) loadAllDayEntries(ctx context.Context, date string) ([]*domainEntry.Entry, error) {
	if s.dayCache != nil {
		if cached, ok, err := s.dayCache.Get(ctx, date); err == nil && ok {
			return cached, nil
		} else if err != nil {
			s.logDebug("day cache lookup failed", err)
		}
	}
	from, to, err := jstDayRange(date, s.jstLocation)
	if err != nil {
		return nil, err
	}
	query := domainEntry.ListQuery{
		Sort:             domainEntry.SortNew,
		Limit:            s.maxAllResults,
		MaxLimitOverride: s.maxAllResults,
		PostedAtFrom:     from,
		PostedAtTo:       to,
	}
	entries, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}
	if s.dayCache != nil {
		if err := s.dayCache.Set(ctx, date, entries); err != nil {
			s.logDebug("day cache set failed", err)
		}
	}
	return entries, nil
}

func (s *Service) loadAllTagEntries(ctx context.Context, tagName string) ([]*domainEntry.Entry, error) {
	if s.tagEntries != nil {
		if cached, ok, err := s.tagEntries.Get(ctx, tagName); err == nil && ok {
			return cached, nil
		} else if err != nil {
			s.logDebug("tag entries cache lookup failed", err)
		}
	}
	query := domainEntry.ListQuery{
		Tags:             []string{tagName},
		Sort:             domainEntry.SortNew,
		Limit:            s.maxAllResults,
		MaxLimitOverride: s.maxAllResults,
	}
	entries, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}
	if s.tagEntries != nil {
		if err := s.tagEntries.Set(ctx, tagName, entries); err != nil {
			s.logDebug("tag entries cache set failed", err)
		}
	}
	return entries, nil
}

func filterByMinUsers(entries []*domainEntry.Entry, minUsers int) []*domainEntry.Entry {
	if minUsers < 0 {
		minUsers = 0
	}
	if minUsers == 0 {
		return entries
	}
	out := make([]*domainEntry.Entry, 0, len(entries))
	for _, e := range entries {
		if e.BookmarkCount >= minUsers {
			out = append(out, e)
		}
	}
	return out
}

func paginate(entries []*domainEntry.Entry, offset, limit int) []*domainEntry.Entry {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = domainEntry.DefaultLimit
	}
	if offset >= len(entries) {
		return nil
	}
	end := offset + limit
	if end > len(entries) {
		end = len(entries)
	}
	return entries[offset:end]
}

func jstDayRange(date string, loc *time.Location) (time.Time, time.Time, error) {
	if loc == nil {
		loc = time.UTC
	}
	start, err := time.ParseInLocation("20060102", date, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid date: %s", date)
	}
	return start.UTC(), start.AddDate(0, 0, 1).UTC(), nil
}
