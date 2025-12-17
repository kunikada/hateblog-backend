package entry

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	domainEntry "hateblog/internal/domain/entry"
	"hateblog/internal/domain/repository"
)

// ListCache defines cache operations required by the service.
type ListCache interface {
	BuildKey(query domainEntry.ListQuery) (string, error)
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, payload []byte) error
}

// Service orchestrates entry use cases.
type Service struct {
	repo   repository.EntryRepository
	cache  ListCache
	logger *slog.Logger
}

// ListResult represents query outcome.
type ListResult struct {
	Entries []*domainEntry.Entry `json:"entries"`
	Total   int64                `json:"total"`
}

// ListParams represents user filters.
type ListParams struct {
	Tags             []string
	MinBookmarkCount int
	Offset           int
	Limit            int
}

// NewService instantiates the service.
func NewService(repo repository.EntryRepository, cache ListCache, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		cache:  cache,
		logger: logger,
	}
}

// ListNewEntries returns entries ordered by posted_at DESC.
func (s *Service) ListNewEntries(ctx context.Context, params ListParams) (ListResult, error) {
	return s.listEntries(ctx, domainEntry.SortNew, params)
}

// ListHotEntries returns entries ordered by bookmark count.
func (s *Service) ListHotEntries(ctx context.Context, params ListParams) (ListResult, error) {
	return s.listEntries(ctx, domainEntry.SortHot, params)
}

func (s *Service) listEntries(ctx context.Context, sort domainEntry.SortType, params ListParams) (ListResult, error) {
	var empty ListResult
	query := domainEntry.ListQuery{
		Tags:             params.Tags,
		MinBookmarkCount: params.MinBookmarkCount,
		Offset:           params.Offset,
		Limit:            params.Limit,
		Sort:             sort,
	}

	var cacheKey string
	if s.cache != nil {
		key, err := s.cache.BuildKey(query)
		if err != nil {
			return empty, err
		}
		cacheKey = key
		if payload, ok, err := s.cache.Get(ctx, key); err == nil && ok {
			var cached ListResult
			if err := json.Unmarshal(payload, &cached); err == nil {
				return cached, nil
			}
			s.logDebug("failed to unmarshal cache payload", err)
		} else if err != nil {
			s.logDebug("cache lookup failed", err)
		}
	}

	entries, err := s.repo.List(ctx, query)
	if err != nil {
		return empty, err
	}

	total, err := s.repo.Count(ctx, query)
	if err != nil {
		return empty, err
	}

	result := ListResult{
		Entries: entries,
		Total:   total,
	}

	if s.cache != nil && cacheKey != "" {
		payload, err := json.Marshal(result)
		if err != nil {
			s.logDebug("failed to marshal entries for cache", err)
			return result, nil
		}
		if err := s.cache.Set(ctx, cacheKey, payload); err != nil {
			s.logDebug("cache set failed", err)
		}
	}

	return result, nil
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
