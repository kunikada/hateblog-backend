package favicon

import (
	"context"
	"encoding/base64"
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

// Limiter throttles external favicon fetches.
type Limiter interface {
	Allow(ctx context.Context, domain string) (bool, error)
}

var (
	// ErrNotInitialized indicates missing dependencies.
	ErrNotInitialized = errors.New("favicon service not initialized")
	// ErrRateLimited indicates rate limit exceeded for the domain.
	ErrRateLimited = errors.New("favicon rate limit exceeded")
)

// Service coordinates favicon fetching and caching.
type Service struct {
	fetcher      Fetcher
	cache        Cache
	limiter      Limiter
	logger       *slog.Logger
	fallbackData []byte
	fallbackType string
}

// NewService builds a favicon service.
func NewService(fetcher Fetcher, cache Cache, limiter Limiter, logger *slog.Logger) *Service {
	return &Service{
		fetcher:      fetcher,
		cache:        cache,
		limiter:      limiter,
		logger:       logger,
		fallbackData: defaultFaviconFallback,
		fallbackType: "image/png",
	}
}

// Fetch returns a favicon for the given domain, using cache when possible.
func (s *Service) Fetch(ctx context.Context, domain string) ([]byte, string, error) {
	if s.fetcher == nil || s.cache == nil {
		return nil, "", ErrNotInitialized
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

	if s.limiter != nil {
		if allowed, err := s.limiter.Allow(ctx, domain); err != nil {
			s.logDebug("favicon rate limit check failed", err)
		} else if !allowed {
			return nil, "", ErrRateLimited
		}
	}

	data, contentType, err := s.fetcher.Fetch(ctx, domain)
	if err != nil {
		s.logDebug("favicon fetch failed", err)
		return s.fallback()
	}

	if err := s.cache.Set(ctx, key, data, contentType); err != nil {
		s.logDebug("favicon cache set failed", err)
	}

	return data, contentType, nil
}

func (s *Service) fallback() ([]byte, string, error) {
	if len(s.fallbackData) == 0 {
		return nil, "", errors.New("fallback icon unavailable")
	}
	return s.fallbackData, s.fallbackType, nil
}

func (s *Service) logDebug(msg string, err error) {
	if s.logger != nil && err != nil {
		s.logger.Debug(msg, "error", err.Error())
	}
}

var defaultFaviconFallback = mustDecodeBase64("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGMAAQAABQABDQottAAAAABJRU5ErkJggg==")

func mustDecodeBase64(value string) []byte {
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		panic("invalid fallback favicon data")
	}
	return data
}
