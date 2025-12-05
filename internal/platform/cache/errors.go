package cache

import "errors"

var (
	// ErrCacheMiss is returned when a key is not found in cache
	ErrCacheMiss = errors.New("cache miss")
)
