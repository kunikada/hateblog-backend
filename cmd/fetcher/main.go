package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"hateblog/internal/domain/tag"
	"hateblog/internal/infra/external/hatena"
	"hateblog/internal/infra/external/yahoo"
	"hateblog/internal/infra/postgres"
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
		lockName          = flag.String("lock", "fetcher", "advisory lock name")
		maxEntries        = flag.Int("max-entries", 300, "maximum number of unique entries to process per run")
		noTags            = flag.Bool("no-tags", false, "disable Yahoo keyphrase tagging even when YAHOO_APP_ID is set")
		tagTopN           = flag.Int("tag-top", 5, "max number of tags to attach per inserted entry")
		yahooMinInterval  = flag.Duration("yahoo-interval", 200*time.Millisecond, "minimum interval between Yahoo API requests")
		executionDeadline = flag.Duration("deadline", 5*time.Minute, "overall execution deadline")
	)
	flag.Parse()

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
	log.Info("fetcher started", "max_entries", *maxEntries, "deadline", *executionDeadline)
	defer func() {
		if ctx.Err() == context.DeadlineExceeded {
			log.Error("fetcher deadline exceeded", "elapsed", time.Since(startedAt), "err", ctx.Err())
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

	locked, unlock, err := batchutil.TryAdvisoryLock(ctx, db.Pool, *lockName)
	if err != nil {
		db.Close()
		log.Error("lock failed", "err", err)
		return 1
	}
	if !locked {
		db.Close()
		log.Info("lock not acquired; skip")
		return 0
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := unlock(unlockCtx); err != nil {
			log.Warn("unlock failed", "err", err)
		}
		db.Close()
	}()

	httpClient := &http.Client{Timeout: cfg.External.HatenaAPITimeout}
	hatenaClient := hatena.NewClient(hatena.ClientConfig{HTTPClient: httpClient})

	feedEntries, err := fetchEntries(ctx, hatenaClient, cfg.External.HatenaRSSFeedURLs, *maxEntries)
	if err != nil {
		log.Error("fetch entries failed", "err", err)
		return 1
	}
	log.Info("fetched entries", "count", len(feedEntries))

	tagRepo := postgres.NewTagRepository(db.Pool)
	yahooClient := yahoo.NewClient(yahoo.ClientConfig{
		AppID: cfg.External.YahooAPIKey,
	})

	inserted := 0
	skipped := 0
	for _, item := range feedEntries {
		select {
		case <-ctx.Done():
			log.Error("deadline exceeded", "err", ctx.Err())
			return 1
		default:
		}

		_, ok, err := insertEntry(ctx, db.Pool, item)
		if err != nil {
			log.Error("insert entry failed", "url", item.URL, "err", err)
			return 1
		}
		if !ok {
			skipped++
			continue
		}
		inserted++

	}

	tagged := 0
	if !*noTags && strings.TrimSpace(cfg.External.YahooAPIKey) != "" && *tagTopN > 0 {
		untagged, err := fetchUntaggedEntries(ctx, db.Pool, *maxEntries)
		if err != nil {
			log.Error("fetch untagged entries failed", "err", err)
			return 1
		}
		for _, entry := range untagged {
			select {
			case <-ctx.Done():
				log.Error("deadline exceeded", "err", ctx.Err())
				return 1
			default:
			}

			item := feedItem{
				Title:   entry.Title,
				URL:     entry.URL,
				Excerpt: entry.Excerpt,
			}
			tagCount, err := attachTags(ctx, tagRepo, db.Pool, yahooClient, entry.ID, item, *tagTopN)
			if err != nil {
				if _, ok := yahoo.IsTooManyRequests(err); ok {
					log.Warn("tagging stopped due to rate limit", "url", entry.URL, "err", err)
					break
				}
				log.Error("attach tags failed", "url", entry.URL, "err", err)
				return 1
			}
			if tagCount > 0 {
				tagged++
			}
			if *yahooMinInterval > 0 {
				time.Sleep(*yahooMinInterval)
			}
		}
	}

	today := timeutil.DateInLocation(timeutil.Now())
	if err := refreshArchiveCountsForDay(ctx, db.Pool, today); err != nil {
		log.Error("refresh archive counts failed", "day", today.Format("2006-01-02"), "err", err)
		return 1
	}

	log.Info("fetcher finished", "inserted", inserted, "skipped", skipped, "tagged", tagged, "elapsed", time.Since(startedAt))
	return 0
}

type feedItem struct {
	Title         string
	URL           string
	Excerpt       string
	Subject       string
	BookmarkCount int
	PostedAt      time.Time
}

func fetchEntries(ctx context.Context, client *hatena.Client, feedURLs []string, max int) ([]feedItem, error) {
	if max <= 0 {
		max = 1
	}
	seen := make(map[string]feedItem, max)
	for _, raw := range feedURLs {
		u := strings.TrimSpace(raw)
		if u == "" {
			continue
		}
		feed, err := client.FetchFeed(ctx, u)
		if err != nil {
			return nil, err
		}
		for _, e := range feed.Entries {
			url := strings.TrimSpace(e.URL)
			if url == "" {
				continue
			}
			if _, ok := seen[url]; ok {
				continue
			}
			subject := strings.Join(e.Subjects, ",")
			seen[url] = feedItem{
				Title:         strings.TrimSpace(e.Title),
				URL:           url,
				Excerpt:       strings.TrimSpace(e.Excerpt),
				Subject:       strings.TrimSpace(subject),
				BookmarkCount: e.BookmarkCount,
				PostedAt:      timeutil.InLocation(e.PublishedAt),
			}
			if len(seen) >= max {
				break
			}
		}
		if len(seen) >= max {
			break
		}
	}

	items := make([]feedItem, 0, len(seen))
	for _, v := range seen {
		items = append(items, v)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].PostedAt.After(items[j].PostedAt)
	})
	return items, nil
}

func insertEntry(ctx context.Context, pool *pgxpool.Pool, item feedItem) (id uuid.UUID, inserted bool, err error) {
	if pool == nil {
		return uuid.Nil, false, fmt.Errorf("pool is nil")
	}
	if strings.TrimSpace(item.URL) == "" {
		return uuid.Nil, false, fmt.Errorf("url is required")
	}
	if item.PostedAt.IsZero() {
		return uuid.Nil, false, fmt.Errorf("posted_at is required")
	}

	now := timeutil.Now()
	const q = `
INSERT INTO entries (title, url, posted_at, bookmark_count, excerpt, subject, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (url) DO UPDATE SET
	title = EXCLUDED.title,
	posted_at = EXCLUDED.posted_at,
	bookmark_count = EXCLUDED.bookmark_count,
	excerpt = EXCLUDED.excerpt,
	subject = EXCLUDED.subject,
	updated_at = EXCLUDED.updated_at
RETURNING id`

	var entryID uuid.UUID
	row := pool.QueryRow(ctx, q,
		item.Title,
		item.URL,
		item.PostedAt,
		item.BookmarkCount,
		nullableText(item.Excerpt),
		nullableText(item.Subject),
		now,
		now,
	)
	if err := row.Scan(&entryID); err != nil {
		if err == pgx.ErrNoRows {
			return uuid.Nil, false, nil
		}
		return uuid.Nil, false, err
	}
	return entryID, true, nil
}

func refreshArchiveCountsForDay(ctx context.Context, pool *pgxpool.Pool, day time.Time) error {
	if pool == nil {
		return fmt.Errorf("pool is nil")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	const deleteQuery = `
DELETE FROM archive_counts
WHERE day = $1`
	if _, err = tx.Exec(ctx, deleteQuery, day); err != nil {
		return err
	}

	const insertQuery = `
INSERT INTO archive_counts (day, bookmark_count, count)
SELECT DATE(posted_at) AS day, bookmark_count, COUNT(1)
FROM entries
WHERE DATE(posted_at) = $1
GROUP BY day, bookmark_count`
	if _, err = tx.Exec(ctx, insertQuery, day); err != nil {
		return err
	}

	err = tx.Commit(ctx)
	return err
}

func nullableText(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func attachTags(
	ctx context.Context,
	tagRepo *postgres.TagRepository,
	pool *pgxpool.Pool,
	yahooClient *yahoo.Client,
	entryID uuid.UUID,
	item feedItem,
	topN int,
) (int, error) {
	if pool == nil {
		return 0, fmt.Errorf("pool is nil")
	}
	input := strings.TrimSpace(strings.Join([]string{item.Title, item.Excerpt}, "\n"))
	phrases, err := yahooClient.Extract(ctx, input)
	if err != nil {
		return 0, err
	}
	if len(phrases) == 0 {
		return 0, nil
	}

	sort.Slice(phrases, func(i, j int) bool { return phrases[i].Score > phrases[j].Score })
	if topN > len(phrases) {
		topN = len(phrases)
	}
	phrases = phrases[:topN]

	maxScore := 0
	for _, p := range phrases {
		if p.Score > maxScore {
			maxScore = p.Score
		}
	}

	added := 0
	for _, p := range phrases {
		name := tag.NormalizeName(p.Text)
		if name == "" {
			continue
		}
		t := &tag.Tag{Name: name}
		if err := tagRepo.Upsert(ctx, t); err != nil {
			return added, err
		}

		score := 0.0
		if maxScore > 0 {
			score = float64(p.Score) / float64(maxScore)
		}
		if score < 0 {
			score = 0
		}
		if score > 1 {
			score = 1
		}

		const q = `
INSERT INTO entry_tags (entry_id, tag_id, score)
VALUES ($1, $2, $3)
ON CONFLICT (entry_id, tag_id) DO NOTHING`
		if _, err := pool.Exec(ctx, q, entryID, t.ID, score); err != nil {
			return added, err
		}
		added++
	}
	return added, nil
}

type tagEntry struct {
	ID      uuid.UUID
	URL     string
	Title   string
	Excerpt string
}

func fetchUntaggedEntries(ctx context.Context, pool *pgxpool.Pool, limit int) ([]tagEntry, error) {
	if pool == nil {
		return nil, fmt.Errorf("pool is nil")
	}
	if limit <= 0 {
		limit = 1
	}
	const q = `
SELECT e.id, e.url, e.title, e.excerpt
FROM entries e
WHERE NOT EXISTS (
	SELECT 1 FROM entry_tags et WHERE et.entry_id = e.id
)
ORDER BY e.posted_at DESC
LIMIT $1`
	rows, err := pool.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]tagEntry, 0, limit)
	for rows.Next() {
		var entry tagEntry
		var excerpt *string
		if err := rows.Scan(&entry.ID, &entry.URL, &entry.Title, &excerpt); err != nil {
			return nil, err
		}
		if excerpt != nil {
			entry.Excerpt = *excerpt
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}
