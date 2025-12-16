package favicon

import (
	"context"
	"errors"
	"log/slog"
)

// Fetcher retrieves favicons from external services.
type Fetcher interface {
	Fetch(ctx context.Context, domain string) ([]byte, string, error)
}

// Cache defines cache operations for favicons.
type Cache interface {
	BuildKey(domain string) (string, error)
	Get(ctx context.Context, key string) ([]byte, string, bool, error)
	Set(ctx context.Context, key string, data []byte, contentType string) error
}

// Service coordinates favicon fetching and caching.
type Service struct {
	fetcher Fetcher
	cache   Cache
	logger  *slog.Logger
}

// NewService builds a favicon service.
func NewService(fetcher Fetcher, cache Cache, logger *slog.Logger) *Service {
	return &Service{
		fetcher: fetcher,
		cache:   cache,
		logger:  logger,
	}
}

// Fetch returns a favicon for the given domain, using cache when possible.
func (s *Service) Fetch(ctx context.Context, domain string) ([]byte, string, error) {
	if s.fetcher == nil || s.cache == nil {
		return nil, "", errors.New("favicon service not initialized")
	}
	key, err := s.cache.BuildKey(domain)
	if err != nil {
		return nil, "", err
	}

	if data, contentType, ok, err := s.cache.Get(ctx, key); err == nil && ok {
		return data, contentType, nil
	} else if err != nil {
		s.logDebug("favicon cache get failed", err)
	}

	data, contentType, err := s.fetcher.Fetch(ctx, domain)
	if err != nil {
		return nil, "", err
	}

	if err := s.cache.Set(ctx, key, data, contentType); err != nil {
		s.logDebug("favicon cache set failed", err)
	}

	return data, contentType, nil
}

func (s *Service) logDebug(msg string, err error) {
	if s.logger != nil && err != nil {
		s.logger.Debug(msg, "error", err.Error())
	}
}
