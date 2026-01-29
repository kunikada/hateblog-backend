package favicon

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
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
	data, contentType, _, err := s.FetchWithCacheStatus(ctx, domain)
	return data, contentType, err
}

// FetchWithCacheStatus returns a favicon and cache hit info.
func (s *Service) FetchWithCacheStatus(ctx context.Context, domain string) ([]byte, string, bool, error) {
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
		data, fallbackType, fallbackErr := s.fallback()
		return data, fallbackType, false, fallbackErr
	}

	if isGoogleDefaultFavicon(data) {
		data = googleDefaultFallback
		contentType = googleDefaultFallbackType
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

var defaultFaviconFallback = mustDecodeBase64("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGMAAQAABQABDQottAAAAABJRU5ErkJggg==")
var googleDefaultFallback = mustDecodeBase64("iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAMAAAAoLQ9TAAAAYFBMVEX////9/f3Z2dnX19fr6+va2trW1tb+/v78/PyJiYnb29vc3Ny1tbWFhYWMjIzT09PY2Ni+vr64uLiPj4+3t7eIiIinp6erq6uLi4vu7u77+/vMzMyKioqpqam5ubn39/f5jK5fAAAAZ0lEQVQYla2PRxKAIBAElyCwJnTNmP7/S0ULxbt9mulbA3gYF5xBgMlEeJTU/houIk6FafSzXANiUdrrVTUReYFYNlyolugRiF1P9BE4/CrUGIl2MgDOdUEwc9dpPS/rZuHtPRP3sA5/ewnKoV106QAAAABJRU5ErkJggg==")
var googleDefaultFallbackType = "image/png"
var googleDefaultMD5 = mustDecodeMD5("3ca64f83fdcf25135d87e08af65e68c9")

func mustDecodeBase64(value string) []byte {
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		panic("invalid fallback favicon data")
	}
	return data
}

func mustDecodeMD5(value string) [16]byte {
	data, err := hex.DecodeString(value)
	if err != nil || len(data) != md5.Size {
		panic("invalid md5 hash")
	}
	var sum [16]byte
	copy(sum[:], data)
	return sum
}

func isGoogleDefaultFavicon(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	return md5.Sum(data) == googleDefaultMD5
}
