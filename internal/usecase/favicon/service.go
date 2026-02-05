package favicon

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"

	"hateblog/internal/pkg/hostname"
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
	SetNegative(ctx context.Context, key string) error
	IsNegative(ctx context.Context, key string) (bool, error)
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
func (s *Service) Fetch(ctx context.Context, domain string) ([]byte, string, bool, error) {
	if s.fetcher == nil {
		return nil, "", false, ErrNotInitialized
	}

	var (
		key string
		err error
	)

	if s.cache != nil {
		key, err = s.cache.BuildKey(domain)
		if err != nil {
			return nil, "", false, err
		}

		// Check negative cache first
		if negative, err := s.cache.IsNegative(ctx, key); err != nil {
			s.logDebug("favicon negative cache check failed", err)
		} else if negative {
			data, fallbackType, fallbackErr := s.fallback()
			return data, fallbackType, true, fallbackErr
		}

		if data, contentType, ok, err := s.cache.Get(ctx, key); err == nil && ok {
			return data, contentType, true, nil
		} else if err != nil {
			s.logDebug("favicon cache get failed", err)
		}
	} else {
		if _, err := hostname.Normalize(domain); err != nil {
			return nil, "", false, err
		}
	}

	if s.limiter != nil {
		if allowed, err := s.limiter.Allow(ctx, domain); err != nil {
			s.logDebug("favicon rate limit check failed", err)
		} else if !allowed {
			return nil, "", false, ErrRateLimited
		}
	}

	data, contentType, err := s.fetcher.Fetch(ctx, domain)
	if err != nil {
		s.logDebug("favicon fetch failed", err)
		// Save negative cache to avoid repeated requests
		if s.cache != nil {
			if negErr := s.cache.SetNegative(ctx, key); negErr != nil {
				s.logDebug("favicon negative cache set failed", negErr)
			}
		}
		data, fallbackType, fallbackErr := s.fallback()
		return data, fallbackType, false, fallbackErr
	}

	if s.cache != nil {
		if err := s.cache.Set(ctx, key, data, contentType); err != nil {
			s.logDebug("favicon cache set failed", err)
		}
	}

	return data, contentType, false, nil
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

var defaultFaviconFallback = mustDecodeBase64("iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAABN0lEQVR4AdxTO46DMBCdIckF0uYGkdJFuUAKoIDb8LkCl4EKCW5BBQJRAA0FnAAB3p2RnOUjdvtFGo/93ptn45EVOPjyPBdZlgnKBxKGVwZd14miKLhwnmcWUCajNE0F8QwuBjagXUjU9z1M07Sgf6aICMSTjkIyiud5QlHYR2J/5svlAmEYChJype/7QPENwjiOhO8CEaFtW6jrGsqyhNPpxBo2oNn5fObjx3EMURSBPJUQAqqq4hiGgaSr+BhIlO6AhEEQQJIk0DQNIKKkd3lnIBWICHQquT7KhwZHBVv8PxgYhvFp2fb/jtbU2tfrxbRyv9/Rsix0HAeJYPSXQdM01HUdr9cr93Z1ia7rstH7/ebe00NCRLjdbkCFFFvvlYEkn88n2raNpmmCqqr4eDx4N8kv8xcAAAD//3cFwBYAAAAGSURBVAMA+OeNBadagYoAAAAASUVORK5CYII=")

func mustDecodeBase64(value string) []byte {
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		panic("invalid fallback favicon data")
	}
	return data
}
