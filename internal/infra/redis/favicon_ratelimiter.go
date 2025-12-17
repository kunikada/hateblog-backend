package redis

import (
	"context"
	"time"

	"hateblog/internal/pkg/hostname"
)

type limiterClient interface {
	SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error)
}

// FaviconRateLimiter ensures domains are fetched at most once per window.
type FaviconRateLimiter struct {
	client limiterClient
	window time.Duration
}

// NewFaviconRateLimiter creates a new limiter.
func NewFaviconRateLimiter(client limiterClient, window time.Duration) *FaviconRateLimiter {
	if window <= 0 {
		window = time.Second
	}
	return &FaviconRateLimiter{
		client: client,
		window: window,
	}
}

// Allow returns true when the domain can be fetched.
func (l *FaviconRateLimiter) Allow(ctx context.Context, domain string) (bool, error) {
	host, err := hostname.Normalize(domain)
	if err != nil {
		return false, err
	}
	key := "favicon:ratelimit:" + host
	return l.client.SetNX(ctx, key, 1, l.window)
}
