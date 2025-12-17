package batchutil

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LockID maps a name to a stable advisory lock identifier.
func LockID(name string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(name))
	return int64(h.Sum64())
}

// TryAdvisoryLock tries to acquire a Postgres advisory lock and returns an unlock function on success.
func TryAdvisoryLock(ctx context.Context, pool *pgxpool.Pool, name string) (locked bool, unlock func(context.Context) error, err error) {
	if pool == nil {
		return false, nil, fmt.Errorf("pool is nil")
	}
	lockID := LockID(name)

	if err := pool.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, lockID).Scan(&locked); err != nil {
		return false, nil, fmt.Errorf("try advisory lock: %w", err)
	}
	if !locked {
		return false, nil, nil
	}

	return true, func(ctx context.Context) error {
		var released bool
		if err := pool.QueryRow(ctx, `SELECT pg_advisory_unlock($1)`, lockID).Scan(&released); err != nil {
			return fmt.Errorf("advisory unlock: %w", err)
		}
		if !released {
			return fmt.Errorf("advisory unlock: not held")
		}
		return nil
	}, nil
}

