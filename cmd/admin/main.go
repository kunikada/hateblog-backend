package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	infraPostgres "hateblog/internal/infra/postgres"
	infraRedis "hateblog/internal/infra/redis"
	"hateblog/internal/pkg/timeutil"
	"hateblog/internal/platform/cache"
	"hateblog/internal/platform/config"
	"hateblog/internal/platform/database"
	"hateblog/internal/platform/logger"
	"hateblog/internal/platform/telemetry"
	usecaseArchive "hateblog/internal/usecase/archive"
	usecaseEntry "hateblog/internal/usecase/entry"
	usecaseRanking "hateblog/internal/usecase/ranking"
	usecaseSearch "hateblog/internal/usecase/search"
	usecaseTag "hateblog/internal/usecase/tag"
)

func main() {
	if err := run(context.Background(), os.Args); err != nil {
		slog.Error("admin command failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) < 2 {
		printUsage()
		return fmt.Errorf("missing command")
	}
	switch args[1] {
	case "cache":
		return runCache(ctx, args[2:])
	case "archive":
		return runArchive(ctx, args[2:])
	default:
		printUsage()
		return fmt.Errorf("unknown command: %s", args[1])
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  admin cache purge --pattern 'hateblog:entries:*' --yes")
	fmt.Fprintln(os.Stderr, "  admin cache warmup --dates 20250105,20250106 --tags go,web --yearly 2024,2025 --min-users 5,10,50")
	fmt.Fprintln(os.Stderr, "  admin archive rebuild --yes")
}

func runCache(ctx context.Context, args []string) error {
	if len(args) < 1 {
		printUsage()
		return fmt.Errorf("missing cache subcommand")
	}
	switch args[0] {
	case "purge":
		return runCachePurge(ctx, args[1:])
	case "warmup":
		return runCacheWarmup(ctx, args[1:])
	default:
		printUsage()
		return fmt.Errorf("unknown cache subcommand: %s", args[0])
	}
}

func runArchive(ctx context.Context, args []string) error {
	if len(args) < 1 {
		printUsage()
		return fmt.Errorf("missing archive subcommand")
	}
	switch args[0] {
	case "rebuild":
		return runArchiveRebuild(ctx, args[1:])
	default:
		printUsage()
		return fmt.Errorf("unknown archive subcommand: %s", args[0])
	}
}

func runArchiveRebuild(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("archive rebuild", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	yes := fs.Bool("yes", false, "required confirmation")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*yes {
		return fmt.Errorf("--yes is required")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := timeutil.SetLocation(cfg.App.TimeZone); err != nil {
		return fmt.Errorf("load timezone: %w", err)
	}

	sentryEnabled, err := telemetry.InitSentry(cfg.Sentry)
	if err != nil {
		return fmt.Errorf("init sentry: %w", err)
	}
	if sentryEnabled {
		defer telemetry.Flush(2 * time.Second)
		defer telemetry.Recover()
	}

	log := logger.New(logger.Config{
		Level:  logger.Level(cfg.App.LogLevel),
		Format: logger.Format(cfg.App.LogFormat),
	})
	if sentryEnabled {
		log = logger.WrapWithSentry(log)
	}
	logger.SetDefault(log)

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
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	if err := rebuildArchiveCounts(ctx, db.Pool); err != nil {
		return fmt.Errorf("rebuild archive counts: %w", err)
	}

	log.Info("archive rebuild completed")
	return nil
}

func runCachePurge(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("cache purge", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pattern := fs.String("pattern", "", "delete keys by pattern (must start with 'hateblog:')")
	batchSize := fs.Int64("batch-size", 500, "SCAN batch size")
	yes := fs.Bool("yes", false, "required confirmation")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*yes {
		return fmt.Errorf("--yes is required")
	}
	if strings.TrimSpace(*pattern) == "" {
		return fmt.Errorf("--pattern is required")
	}
	if !strings.HasPrefix(*pattern, "hateblog:") {
		return fmt.Errorf("pattern must start with 'hateblog:'")
	}

	cfg, log, redisClient, closeAll, sentryEnabled, err := connect(ctx)
	_ = cfg
	if err != nil {
		return err
	}
	defer closeAll()
	if sentryEnabled {
		defer telemetry.Recover()
	}

	deleted, err := redisClient.DeleteByPattern(ctx, *pattern, *batchSize)
	if err != nil {
		return err
	}
	log.Info("cache purge completed", "pattern", *pattern, "deleted", deleted)
	return nil
}

func runCacheWarmup(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("cache warmup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dates := fs.String("dates", "", "comma-separated YYYYMMDD list (required)")
	tags := fs.String("tags", "", "comma-separated tag names")
	minUsers := fs.String("min-users", "5,10,50,100,500,1000", "comma-separated min_users list for caches that vary by min_users")
	yearly := fs.String("yearly", "", "comma-separated years for yearly rankings")
	monthly := fs.String("monthly", "", "comma-separated YYYY-MM for monthly rankings")
	weekly := fs.String("weekly", "", "comma-separated YYYY-WW (ISO week) for weekly rankings")
	searchQueries := fs.String("search", "", "comma-separated search queries to warm")
	yes := fs.Bool("yes", false, "required confirmation")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*yes {
		return fmt.Errorf("--yes is required")
	}
	dateList := splitCSV(*dates)
	if len(dateList) == 0 {
		return fmt.Errorf("--dates is required")
	}

	cfg, log, redisClient, closeAll, sentryEnabled, err := connect(ctx)
	if err != nil {
		return err
	}
	defer closeAll()
	if sentryEnabled {
		defer telemetry.Recover()
	}

	if !cfg.App.CacheEnabled {
		return fmt.Errorf("cache is disabled (APP_CACHE_ENABLED=false)")
	}

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
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	entryRepo := infraPostgres.NewEntryRepository(db.Pool)
	tagRepo := infraPostgres.NewTagRepository(db.Pool)
	searchHistoryRepo := infraPostgres.NewSearchHistoryRepository(db.Pool)

	dayEntriesCache := infraRedis.NewDayEntriesCache(redisClient)
	tagEntriesCache := infraRedis.NewTagEntriesCache(redisClient)
	searchCache := infraRedis.NewSearchCache(redisClient)
	tagsListCache := infraRedis.NewTagsListCache(redisClient)
	archiveCache := infraRedis.NewArchiveCache(redisClient)
	yearlyRankingCache := infraRedis.NewYearlyRankingCache(redisClient)
	monthlyRankingCache := infraRedis.NewMonthlyRankingCache(redisClient)
	weeklyRankingCache := infraRedis.NewWeeklyRankingCache(redisClient)

	entryService := usecaseEntry.NewService(entryRepo, dayEntriesCache, tagEntriesCache, log)
	tagService := usecaseTag.NewService(tagRepo, tagsListCache)
	searchService := usecaseSearch.NewService(entryRepo, searchHistoryRepo, searchCache, log)
	archiveService := usecaseArchive.NewService(entryRepo, archiveCache)
	rankingService := usecaseRanking.NewService(entryRepo, yearlyRankingCache, monthlyRankingCache, weeklyRankingCache)

	if _, err := tagService.List(ctx, 50, 0); err != nil {
		return fmt.Errorf("warm tags list: %w", err)
	}

	for _, date := range dateList {
		if _, err := entryService.ListNewEntries(ctx, usecaseEntry.DayListParams{
			Date:             date,
			MinBookmarkCount: 0,
			Limit:            1,
			Offset:           0,
		}); err != nil {
			return fmt.Errorf("warm day entries: %s: %w", date, err)
		}
	}

	for _, tagName := range splitCSV(*tags) {
		if _, err := entryService.ListTagEntries(ctx, tagName, usecaseEntry.TagListParams{
			MinBookmarkCount: 0,
			Limit:            1,
			Offset:           0,
		}); err != nil {
			return fmt.Errorf("warm tag entries: %s: %w", tagName, err)
		}
	}

	for _, mu := range splitCSVInts(*minUsers) {
		if _, err := archiveService.List(ctx, mu); err != nil {
			return fmt.Errorf("warm archive: min_users=%d: %w", mu, err)
		}
	}

	for _, year := range splitCSVInts(*yearly) {
		for _, mu := range splitCSVInts(*minUsers) {
			if _, err := rankingService.Yearly(ctx, year, 1000, mu); err != nil {
				return fmt.Errorf("warm yearly ranking: year=%d min_users=%d: %w", year, mu, err)
			}
		}
	}
	for _, ym := range splitCSV(*monthly) {
		year, month, err := parseYearMonth(ym)
		if err != nil {
			return err
		}
		for _, mu := range splitCSVInts(*minUsers) {
			if _, err := rankingService.Monthly(ctx, year, month, 100, mu); err != nil {
				return fmt.Errorf("warm monthly ranking: %s min_users=%d: %w", ym, mu, err)
			}
		}
	}
	for _, yw := range splitCSV(*weekly) {
		year, week, err := parseYearWeek(yw)
		if err != nil {
			return err
		}
		for _, mu := range splitCSVInts(*minUsers) {
			if _, err := rankingService.Weekly(ctx, year, week, 100, mu); err != nil {
				return fmt.Errorf("warm weekly ranking: %s min_users=%d: %w", yw, mu, err)
			}
		}
	}

	for _, q := range splitCSV(*searchQueries) {
		if _, err := searchService.Search(ctx, q, usecaseSearch.Params{
			MinBookmarkCount: 5,
			Limit:            25,
			Offset:           0,
		}); err != nil {
			return fmt.Errorf("warm search: %q: %w", q, err)
		}
	}

	log.Info("cache warmup completed",
		"dates", len(dateList),
		"tags", len(splitCSV(*tags)),
		"yearly", len(splitCSV(*yearly)),
		"monthly", len(splitCSV(*monthly)),
		"weekly", len(splitCSV(*weekly)),
		"search", len(splitCSV(*searchQueries)),
	)
	return nil
}

func rebuildArchiveCounts(ctx context.Context, pool *pgxpool.Pool) error {
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

	if _, err = tx.Exec(ctx, "TRUNCATE TABLE archive_counts"); err != nil {
		return err
	}
	const insertQuery = `
INSERT INTO archive_counts (day, bookmark_count, count)
SELECT DATE(posted_at) AS day, bookmark_count, COUNT(1)
FROM entries
GROUP BY day, bookmark_count`
	if _, err = tx.Exec(ctx, insertQuery); err != nil {
		return err
	}

	err = tx.Commit(ctx)
	return err
}

func connect(ctx context.Context) (*config.Config, *slog.Logger, *cache.Cache, func(), bool, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, nil, func() {}, false, fmt.Errorf("load config: %w", err)
	}
	if err := timeutil.SetLocation(cfg.App.TimeZone); err != nil {
		return nil, nil, nil, func() {}, false, fmt.Errorf("load timezone: %w", err)
	}

	sentryEnabled, err := telemetry.InitSentry(cfg.Sentry)
	if err != nil {
		return nil, nil, nil, func() {}, false, fmt.Errorf("init sentry: %w", err)
	}

	log := logger.New(logger.Config{
		Level:  logger.Level(cfg.App.LogLevel),
		Format: logger.Format(cfg.App.LogFormat),
	})
	if sentryEnabled {
		log = logger.WrapWithSentry(log)
	}
	logger.SetDefault(log)

	redisClient, err := cache.New(cache.Config{
		Address:      cfg.Redis.Address(),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		MaxRetries:   cfg.Redis.MaxRetries,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
	}, log)
	if err != nil {
		return nil, nil, nil, func() {}, sentryEnabled, fmt.Errorf("connect redis: %w", err)
	}
	closeAll := func() {
		if sentryEnabled {
			telemetry.Flush(2 * time.Second)
		}
		_ = redisClient.Close()
	}
	return cfg, log, redisClient, closeAll, sentryEnabled, nil
}

func splitCSV(value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func splitCSVInts(value string) []int {
	parts := splitCSV(value)
	if len(parts) == 0 {
		return nil
	}
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}

func parseYearMonth(value string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid year-month: %s", value)
	}
	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid year: %s", value)
	}
	month, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid month: %s", value)
	}
	return year, month, nil
}

func parseYearWeek(value string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid year-week: %s", value)
	}
	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid year: %s", value)
	}
	week, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid week: %s", value)
	}
	return year, week, nil
}
