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
		lockName          = flag.String("lock", "", "advisory lock name (default: updater)")
		limit             = flag.Int("limit", 50, "max entries to update per run")
		executionDeadline = flag.Duration("deadline", 3*time.Minute, "overall execution deadline")
	)
	flag.Parse()

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
	loc, err := time.LoadLocation(cfg.App.TimeZone)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to load timezone %s: %v\n", cfg.App.TimeZone, err)
		return 1
	}
	time.Local = loc

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
	log.Info("updater started", "limit", *limit, "deadline", *executionDeadline)
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
		jobLock = "updater"
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

	httpClient := &http.Client{Timeout: cfg.External.HatenaAPITimeout}
	hatenaClient := hatena.NewClient(hatena.ClientConfig{
		HTTPClient:           httpClient,
		BookmarkCountMaxURLs: cfg.External.HatenaMaxURLs,
	})

	buckets := []struct {
		name  string
		where string
	}{
		{name: "posted-7d", where: "posted_at > NOW() - INTERVAL '7 days'"},
		{name: "posted-30d", where: "posted_at > NOW() - INTERVAL '30 days'"},
		{name: "posted-365d", where: "posted_at > NOW() - INTERVAL '365 days'"},
		{name: "posted-older", where: ""},
	}

	var totalTargets int
	var totalUpdated int
	var totalMissing int
	for _, bucket := range buckets {
		urls, err := selectTargetURLs(ctx, db.Pool, bucket.where, *limit)
		if err != nil {
			log.Error("select targets failed", "bucket", bucket.name, "err", err)
			return 1
		}
		if len(urls) == 0 {
			log.Info("no targets", "bucket", bucket.name)
			continue
		}

		counts, err := hatenaClient.GetBookmarkCounts(ctx, urls)
		if err != nil {
			log.Error("fetch bookmark counts failed", "bucket", bucket.name, "err", err)
			return 1
		}

		updated, missing, err := applyCounts(ctx, db.Pool, urls, counts)
		if err != nil {
			log.Error("apply counts failed", "bucket", bucket.name, "err", err)
			return 1
		}

		totalTargets += len(urls)
		totalUpdated += updated
		totalMissing += missing
		log.Info("updater bucket finished", "bucket", bucket.name, "targets", len(urls), "updated", updated, "missing", missing)
	}

	log.Info("updater bucket phase finished", "targets", totalTargets, "updated", totalUpdated, "missing", totalMissing)

	// HTTP→HTTPS URL正規化
	const httpNormalizeLimit = 25
	httpEntries, err := selectHTTPEntries(ctx, db.Pool, httpNormalizeLimit)
	if err != nil {
		log.Error("select http entries failed", "err", err)
		return 1
	}
	if len(httpEntries) == 0 {
		log.Info("no http entries to normalize")
	} else {
		normalized, merged, err := normalizeHTTPURLs(ctx, db.Pool, hatenaClient, httpEntries, log)
		if err != nil {
			log.Error("http normalization failed", "err", err)
			return 1
		}
		log.Info("http normalization finished", "targets", len(httpEntries), "updated", normalized, "merged", merged)
	}

	log.Info("updater finished", "targets", totalTargets, "updated", totalUpdated, "missing", totalMissing, "elapsed", time.Since(startedAt))
	return 0
}

func selectTargetURLs(ctx context.Context, pool *pgxpool.Pool, where string, limit int) ([]string, error) {
	if pool == nil {
		return nil, fmt.Errorf("pool is nil")
	}

	const base = `
SELECT url
FROM entries
%s
ORDER BY updated_at ASC
LIMIT $1`

	whereClause := ""
	if strings.TrimSpace(where) != "" {
		whereClause = "WHERE " + where
	}
	rows, err := pool.Query(ctx, fmt.Sprintf(base, whereClause), limit)
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

// selectHTTPEntries は url が http: で始まるエントリーの id と url を updated_at ASC で抽出する。
func selectHTTPEntries(ctx context.Context, pool *pgxpool.Pool, limit int) ([]httpEntry, error) {
	if pool == nil {
		return nil, fmt.Errorf("pool is nil")
	}

	const q = `
SELECT id, url
FROM entries
WHERE url LIKE 'http://%'
ORDER BY updated_at ASC
LIMIT $1`

	rows, err := pool.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("query http entries: %w", err)
	}
	defer rows.Close()

	var entries []httpEntry
	for rows.Next() {
		var e httpEntry
		if err := rows.Scan(&e.id, &e.url); err != nil {
			return nil, fmt.Errorf("scan http entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

type httpEntry struct {
	id  string
	url string
}

// normalizeHTTPURLs は http エントリーに対して http/https 両方のブックマーク件数を取得し、
// 多い方の URL・件数でエントリーを更新する。
// httpsエントリーが既にDBに存在する場合は、件数が多い方を残してもう片方を削除する。
func normalizeHTTPURLs(ctx context.Context, pool *pgxpool.Pool, client *hatena.Client, entries []httpEntry, log logger) (updated int, deleted int, err error) {
	if len(entries) == 0 {
		return 0, 0, nil
	}

	// http URL と https URL の両方を用意
	allURLs := make([]string, 0, len(entries)*2)
	httpsURLs := make([]string, 0, len(entries))
	for _, e := range entries {
		httpsURL := "https://" + strings.TrimPrefix(e.url, "http://")
		allURLs = append(allURLs, e.url)
		allURLs = append(allURLs, httpsURL)
		httpsURLs = append(httpsURLs, httpsURL)
	}

	// DBにhttpsエントリーが既に存在するか確認
	existingHTTPS, err := findExistingEntries(ctx, pool, httpsURLs)
	if err != nil {
		return 0, 0, fmt.Errorf("find existing https entries: %w", err)
	}

	counts, err := client.GetBookmarkCounts(ctx, allURLs)
	if err != nil {
		return 0, 0, fmt.Errorf("fetch bookmark counts for http normalization: %w", err)
	}

	now := time.Now()
	for _, e := range entries {
		httpsURL := "https://" + strings.TrimPrefix(e.url, "http://")
		httpCount := counts[e.url]
		httpsCount := counts[httpsURL]

		bestCount := httpCount
		if httpsCount > httpCount {
			bestCount = httpsCount
		}

		if existing, ok := existingHTTPS[httpsURL]; ok {
			// httpsエントリーが既にDBに存在する → マージして片方を削除
			// httpsエントリーを残し、httpエントリーを削除する
			if err := mergeAndDelete(ctx, pool, existing.id, e.id, bestCount, now); err != nil {
				return updated, deleted, fmt.Errorf("merge entries: %w", err)
			}
			updated++
			deleted++
			log.Debug("merged http into https entry", "http_id", e.id, "https_id", existing.id, "url", httpsURL, "count", bestCount)
		} else {
			// httpsエントリーが存在しない → 件数比較でURL書き換え
			bestURL := e.url
			if httpsCount > httpCount {
				bestURL = httpsURL
			}
			const q = `
UPDATE entries
SET url = $1,
    bookmark_count = $2,
    updated_at = $3
WHERE id = $4`
			tag, err := pool.Exec(ctx, q, bestURL, bestCount, now, e.id)
			if err != nil {
				return updated, deleted, fmt.Errorf("update http entry: %w", err)
			}
			if tag.RowsAffected() > 0 {
				updated++
			}
		}
	}
	return updated, deleted, nil
}

// findExistingEntries は指定URLのエントリーがDBに存在するか確認し、存在するものを返す。
func findExistingEntries(ctx context.Context, pool *pgxpool.Pool, urls []string) (map[string]httpEntry, error) {
	if len(urls) == 0 {
		return nil, nil
	}

	const q = `SELECT id, url FROM entries WHERE url = ANY($1)`
	rows, err := pool.Query(ctx, q, urls)
	if err != nil {
		return nil, fmt.Errorf("query existing entries: %w", err)
	}
	defer rows.Close()

	result := make(map[string]httpEntry, len(urls))
	for rows.Next() {
		var e httpEntry
		if err := rows.Scan(&e.id, &e.url); err != nil {
			return nil, fmt.Errorf("scan existing entry: %w", err)
		}
		result[e.url] = e
	}
	return result, rows.Err()
}

// mergeAndDelete は残すエントリーのbookmark_countを更新し、不要なエントリーを削除する。
func mergeAndDelete(ctx context.Context, pool *pgxpool.Pool, keepID, deleteID string, count int, now time.Time) error {
	const updateQ = `
UPDATE entries
SET bookmark_count = $1,
    updated_at = $2
WHERE id = $3`
	if _, err := pool.Exec(ctx, updateQ, count, now, keepID); err != nil {
		return fmt.Errorf("update kept entry: %w", err)
	}

	const deleteQ = `DELETE FROM entries WHERE id = $1`
	if _, err := pool.Exec(ctx, deleteQ, deleteID); err != nil {
		return fmt.Errorf("delete merged entry: %w", err)
	}
	return nil
}

// logger はログ出力のインタフェース。
type logger interface {
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
}

func applyCounts(ctx context.Context, pool *pgxpool.Pool, urls []string, counts map[string]int) (updated int, missing int, err error) {
	if pool == nil {
		return 0, 0, fmt.Errorf("pool is nil")
	}
	now := time.Now()

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
