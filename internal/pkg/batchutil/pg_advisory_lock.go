package batchutil

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LockID maps a name to a stable advisory lock identifier.
func LockID(name string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(name))
	sum := h.Sum64()
	// #nosec G115 -- mask to 63-bit to fit advisory lock range.
	return int64(sum & math.MaxInt64)
}

// TryAdvisoryLock holds a dedicated connection until the returned unlock function is called.
func TryAdvisoryLock(ctx context.Context, pool *pgxpool.Pool, name string) (locked bool, unlock func(context.Context) error, err error) {
	if pool == nil {
		return false, nil, fmt.Errorf("pool is nil")
	}
	lockID := LockID(name)
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return false, nil, fmt.Errorf("acquire advisory lock connection: %w", err)
	}
	release := func(discard bool) {
		if discard {
			closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			// An uncertain lock state must not be returned to the pool.
			_ = conn.Hijack().Close(closeCtx)
			return
		}
		conn.Release()
	}

	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, lockID).Scan(&locked); err != nil {
		release(true)
		return false, nil, fmt.Errorf("try advisory lock: %w", err)
	}
	if !locked {
		release(false)
		return false, nil, nil
	}

	var once sync.Once
	var unlockErr error
	return true, func(ctx context.Context) error {
		once.Do(func() {
			var released bool
			if err := conn.QueryRow(ctx, `SELECT pg_advisory_unlock($1)`, lockID).Scan(&released); err != nil {
				unlockErr = fmt.Errorf("advisory unlock: %w", err)
			} else if !released {
				unlockErr = fmt.Errorf("advisory unlock: not held")
			}
			release(unlockErr != nil)
		})
		return unlockErr
	}, nil
}
