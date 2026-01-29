package search

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	domainEntry "hateblog/internal/domain/entry"
)

// EntryRepository defines entry access required for search.
type EntryRepository interface {
	List(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, error)
	Count(ctx context.Context, query domainEntry.ListQuery) (int64, error)
}

// HistoryRepository records search queries.
type HistoryRepository interface {
	Record(ctx context.Context, query string, searchedAt time.Time) error
}

// Params defines search filters.
type Params struct {
	MinBookmarkCount int
	Limit            int
	Offset           int
}

// Result bundles search results.
type Result struct {
	Query   string
	Entries []*domainEntry.Entry
	Total   int64
	Limit   int
	Offset  int
}

// Service performs search operations.
type Service struct {
	entries EntryRepository
	history HistoryRepository
	cache   ResultCache
	logger  *slog.Logger
}

// ResultCache caches search results.
type ResultCache interface {
	Get(ctx context.Context, query string, minUsers, limit, offset int, out any) (bool, error)
	Set(ctx context.Context, query string, minUsers, limit, offset int, value any) error
}

// NewService builds a search service.
func NewService(entries EntryRepository, history HistoryRepository, cache ResultCache, logger *slog.Logger) *Service {
	return &Service{
		entries: entries,
		history: history,
		cache:   cache,
		logger:  logger,
	}
}

// Search executes a keyword search.
func (s *Service) Search(ctx context.Context, query string, params Params) (Result, error) {
	result, _, err := s.SearchWithCacheStatus(ctx, query, params)
	return result, err
}

// SearchWithCacheStatus executes a keyword search and returns cache hit info.
func (s *Service) SearchWithCacheStatus(ctx context.Context, query string, params Params) (Result, bool, error) {
	norm := strings.TrimSpace(query)
	if norm == "" {
		return Result{}, false, fmt.Errorf("q is required")
	}
	if len(norm) > 500 {
		return Result{}, false, fmt.Errorf("q must be <= 500 characters")
	}
	limit := params.Limit
	if limit <= 0 {
		limit = domainEntry.DefaultLimit
	}
	if limit > domainEntry.MaxLimit {
		limit = domainEntry.MaxLimit
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}
	minUsers := params.MinBookmarkCount
	if minUsers < 0 {
		minUsers = 0
	}

	if s.cache != nil {
		var cached Result
		ok, err := s.cache.Get(ctx, norm, minUsers, limit, offset, &cached)
		if err != nil {
			s.logDebug("failed to get search cache", err)
		} else if ok {
			if s.history != nil {
				if err := s.history.Record(ctx, norm, time.Now()); err != nil {
					s.logDebug("failed to record search history", err)
				}
			}
			return cached, true, nil
		}
	}

	queryParams := domainEntry.ListQuery{
		Keyword:          norm,
		Limit:            limit,
		Offset:           offset,
		Sort:             domainEntry.SortHot,
		MinBookmarkCount: minUsers,
	}

	entries, total, err := s.listAndCount(ctx, queryParams)
	if err != nil {
		return Result{}, false, err
	}

	if s.history != nil {
		if err := s.history.Record(ctx, norm, time.Now()); err != nil {
			s.logDebug("failed to record search history", err)
		}
	}

	result := Result{
		Query:   norm,
		Entries: entries,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}

	if s.cache != nil {
		if err := s.cache.Set(ctx, norm, minUsers, limit, offset, result); err != nil {
			s.logDebug("failed to set search cache", err)
		}
	}

	return result, false, nil
}

func (s *Service) listAndCount(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, int64, error) {
	type listAndCounter interface {
		ListAndCount(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, int64, error)
	}
	if repo, ok := any(s.entries).(listAndCounter); ok {
		return repo.ListAndCount(ctx, query)
	}
	entries, err := s.entries.List(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.entries.Count(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

func (s *Service) logDebug(msg string, err error) {
	if s.logger != nil && err != nil {
		s.logger.Debug(msg, "error", err)
	}
}
