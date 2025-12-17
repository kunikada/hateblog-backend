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
	logger  *slog.Logger
}

// NewService builds a search service.
func NewService(entries EntryRepository, history HistoryRepository, logger *slog.Logger) *Service {
	return &Service{
		entries: entries,
		history: history,
		logger:  logger,
	}
}

// Search executes a keyword search.
func (s *Service) Search(ctx context.Context, query string, params Params) (Result, error) {
	norm := strings.TrimSpace(query)
	if norm == "" {
		return Result{}, fmt.Errorf("q is required")
	}
	if len(norm) > 500 {
		return Result{}, fmt.Errorf("q must be <= 500 characters")
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

	queryParams := domainEntry.ListQuery{
		Keyword:          norm,
		Limit:            limit,
		Offset:           offset,
		Sort:             domainEntry.SortHot,
		MinBookmarkCount: minUsers,
	}

	entries, err := s.entries.List(ctx, queryParams)
	if err != nil {
		return Result{}, err
	}
	total, err := s.entries.Count(ctx, queryParams)
	if err != nil {
		return Result{}, err
	}

	if s.history != nil {
		if err := s.history.Record(ctx, norm, time.Now()); err != nil {
			s.logDebug("failed to record search history", err)
		}
	}

	return Result{
		Query:   norm,
		Entries: entries,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}, nil
}

func (s *Service) logDebug(msg string, err error) {
	if s.logger != nil && err != nil {
		s.logger.Debug(msg, "error", err)
	}
}
