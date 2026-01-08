package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"hateblog/internal/infra/external/hatena"
	"hateblog/internal/pkg/batchutil"
	"hateblog/internal/pkg/timeutil"
	"hateblog/internal/platform/config"
	"hateblog/internal/platform/database"
	platformLogger "hateblog/internal/platform/logger"
	"hateblog/internal/platform/telemetry"
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		tier              = flag.String("tier", "", "update tier: high|medium|low|round")
		lockName          = flag.String("lock", "", "advisory lock name (default: updater-<tier>)")
		limit             = flag.Int("limit", 50, "max entries to update per run")
		executionDeadline = flag.Duration("deadline", 3*time.Minute, "overall execution deadline")
	)
	flag.Parse()

	if strings.TrimSpace(*tier) == "" {
		_, _ = fmt.Fprintln(os.Stderr, "--tier is required (high|medium|low|round)")
		return 2
	}
	if *limit <= 0 {
		*limit = 50
	}

	ctx, cancel := context.WithTimeout(context.Background(), *executionDeadline)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := timeutil.SetLocation(cfg.App.TimeZone); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}

	sentryEnabled, err := telemetry.InitSentry(cfg.Sentry)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	} else if sentryEnabled {
		defer telemetry.Flush(2 * time.Second)
		defer telemetry.Recover()
	}

	log := platformLogger.New(platformLogger.Config{
		Level:  platformLogger.Level(cfg.App.LogLevel),
		Format: platformLogger.Format(cfg.App.LogFormat),
	})
	if sentryEnabled {
		log = platformLogger.WrapWithSentry(log)
	}
	platformLogger.SetDefault(log)
	startedAt := time.Now()
	log.Info("updater started", "tier", *tier, "limit", *limit, "deadline", *executionDeadline)
	defer func() {
		if ctx.Err() == context.DeadlineExceeded {
			log.Error("updater deadline exceeded", "elapsed", time.Since(startedAt), "err", ctx.Err())
		}
	}()

	db, err := database.New(ctx, database.Config{
		ConnectionString: cfg.Database.ConnectionString(),
		MaxConns:         cfg.Database.MaxConns,
		MinConns:         cfg.Database.MinConns,
		MaxConnLifetime:  cfg.Database.MaxConnLifetime,
		MaxConnIdleTime:  cfg.Database.MaxConnIdleTime,
		ConnectTimeout:   cfg.Database.ConnectTimeout,
		TimeZone:         cfg.App.TimeZone,
	}, log)
	if err != nil {
		log.Error("db connect failed", "err", err)
		return 1
	}
	defer db.Close()

	jobLock := strings.TrimSpace(*lockName)
	if jobLock == "" {
		jobLock = "updater-" + strings.ToLower(strings.TrimSpace(*tier))
	}

	locked, unlock, err := batchutil.TryAdvisoryLock(ctx, db.Pool, jobLock)
	if err != nil {
		log.Error("lock failed", "err", err)
		return 1
	}
	if !locked {
		log.Info("lock not acquired; skip")
		return 0
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := unlock(unlockCtx); err != nil {
			log.Warn("unlock failed", "err", err)
		}
	}()

	urls, err := selectTargetURLs(ctx, db.Pool, strings.ToLower(strings.TrimSpace(*tier)), *limit)
	if err != nil {
		log.Error("select targets failed", "err", err)
		return 1
	}
	if len(urls) == 0 {
		log.Info("no targets")
		return 0
	}

	httpClient := &http.Client{Timeout: cfg.External.HatenaAPITimeout}
	hatenaClient := hatena.NewClient(hatena.ClientConfig{
		HTTPClient:           httpClient,
		BookmarkCountMaxURLs: cfg.External.HatenaMaxURLs,
	})

	counts, err := hatenaClient.GetBookmarkCounts(ctx, urls)
	if err != nil {
		log.Error("fetch bookmark counts failed", "err", err)
		return 1
	}

	updated, missing, err := applyCounts(ctx, db.Pool, urls, counts)
	if err != nil {
		log.Error("apply counts failed", "err", err)
		return 1
	}

	log.Info("updater finished", "tier", *tier, "targets", len(urls), "updated", updated, "missing", missing, "elapsed", time.Since(startedAt))
	return 0
}

func selectTargetURLs(ctx context.Context, pool *pgxpool.Pool, tier string, limit int) ([]string, error) {
	if pool == nil {
		return nil, fmt.Errorf("pool is nil")
	}

	var where string
	switch tier {
	case "high":
		where = `(posted_at > NOW() - INTERVAL '30 days' OR bookmark_count >= 100)`
	case "medium":
		where = `(posted_at > NOW() - INTERVAL '90 days' OR bookmark_count >= 20)`
	case "low":
		where = `(posted_at > NOW() - INTERVAL '180 days' OR bookmark_count >= 10)`
	case "round":
		where = `bookmark_count >= 5`
	default:
		return nil, fmt.Errorf("unknown tier: %s", tier)
	}

	const base = `
SELECT url
FROM entries
WHERE %s
ORDER BY updated_at ASC
LIMIT $1`

	rows, err := pool.Query(ctx, fmt.Sprintf(base, where), limit)
	if err != nil {
		return nil, fmt.Errorf("query targets: %w", err)
	}
	defer rows.Close()

	var urls []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, fmt.Errorf("scan url: %w", err)
		}
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		urls = append(urls, u)
	}
	return urls, rows.Err()
}

func applyCounts(ctx context.Context, pool *pgxpool.Pool, urls []string, counts map[string]int) (updated int, missing int, err error) {
	if pool == nil {
		return 0, 0, fmt.Errorf("pool is nil")
	}
	now := timeutil.Now()

	for _, u := range urls {
		count, ok := counts[u]
		var countParam any
		if ok {
			countParam = count
		} else {
			missing++
			countParam = nil
		}
		const q = `
UPDATE entries
SET bookmark_count = COALESCE($1, bookmark_count),
	updated_at = $2
WHERE url = $3`
		tag, err := pool.Exec(ctx, q, countParam, now, u)
		if err != nil {
			return updated, missing, fmt.Errorf("update entry: %w", err)
		}
		if tag.RowsAffected() > 0 {
			updated++
		}
	}
	return updated, missing, nil
}
