package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	infraGoogle "hateblog/internal/infra/external/google"
	"hateblog/internal/infra/handler"
	infraPostgres "hateblog/internal/infra/postgres"
	infraRedis "hateblog/internal/infra/redis"
	"hateblog/internal/platform/cache"
	"hateblog/internal/platform/config"
	"hateblog/internal/platform/database"
	"hateblog/internal/platform/logger"
	"hateblog/internal/platform/metrics"
	"hateblog/internal/platform/server"
	usecaseArchive "hateblog/internal/usecase/archive"
	usecaseEntry "hateblog/internal/usecase/entry"
	usecaseFavicon "hateblog/internal/usecase/favicon"
	usecaseMetrics "hateblog/internal/usecase/metrics"
	usecaseRanking "hateblog/internal/usecase/ranking"
	usecaseSearch "hateblog/internal/usecase/search"
	usecaseTag "hateblog/internal/usecase/tag"
)

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.New(logger.Config{
		Level:  logger.Level(cfg.App.LogLevel),
		Format: logger.Format(cfg.App.LogFormat),
	})
	logger.SetDefault(log)

	db, err := database.New(ctx, database.Config{
		ConnectionString: cfg.Database.ConnectionString(),
		MaxConns:         cfg.Database.MaxConns,
		MinConns:         cfg.Database.MinConns,
		MaxConnLifetime:  cfg.Database.MaxConnLifetime,
		MaxConnIdleTime:  cfg.Database.MaxConnIdleTime,
		ConnectTimeout:   cfg.Database.ConnectTimeout,
	}, log)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

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
		return fmt.Errorf("connect redis: %w", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Error("failed to close redis", "error", err)
		}
	}()

	entryRepo := infraPostgres.NewEntryRepository(db.Pool)
	tagRepo := infraPostgres.NewTagRepository(db.Pool)
	searchHistoryRepo := infraPostgres.NewSearchHistoryRepository(db.Pool)
	clickMetricsRepo := infraPostgres.NewClickMetricsRepository(db.Pool)

	dayEntriesCache := infraRedis.NewDayEntriesCache(redisClient)
	tagEntriesCache := infraRedis.NewTagEntriesCache(redisClient)
	searchCache := infraRedis.NewSearchCache(redisClient)
	tagsListCache := infraRedis.NewTagsListCache(redisClient)
	archiveCache := infraRedis.NewArchiveCache(redisClient)
	yearlyRankingCache := infraRedis.NewYearlyRankingCache(redisClient)
	monthlyRankingCache := infraRedis.NewMonthlyRankingCache(redisClient)
	weeklyRankingCache := infraRedis.NewWeeklyRankingCache(redisClient)

	entryService := usecaseEntry.NewService(entryRepo, dayEntriesCache, tagEntriesCache, log)
	archiveService := usecaseArchive.NewService(entryRepo, archiveCache)
	rankingService := usecaseRanking.NewService(entryRepo, yearlyRankingCache, monthlyRankingCache, weeklyRankingCache)
	tagService := usecaseTag.NewService(tagRepo, tagsListCache)
	searchService := usecaseSearch.NewService(entryRepo, searchHistoryRepo, searchCache, log)
	metricsService := usecaseMetrics.NewService(entryRepo, clickMetricsRepo)

	apiBasePath := strings.TrimSpace(cfg.App.APIBasePath)
	if apiBasePath == "" {
		apiBasePath = "/"
	} else {
		if !strings.HasPrefix(apiBasePath, "/") {
			apiBasePath = "/" + apiBasePath
		}
		if apiBasePath != "/" {
			apiBasePath = strings.TrimRight(apiBasePath, "/")
		}
	}

	faviconCache := infraRedis.NewFaviconCache(redisClient, cfg.App.FaviconCacheTTL)
	faviconLimiter := infraRedis.NewFaviconRateLimiter(redisClient, cfg.External.FaviconRateLimit)
	googleClient := infraGoogle.NewClient(infraGoogle.Config{
		HTTPClient: &http.Client{
			Timeout: cfg.External.FaviconAPITimeout,
		},
		UserAgent: "hateblog-favicon-proxy",
	})
	faviconService := usecaseFavicon.NewService(googleClient, faviconCache, faviconLimiter, log)

	entryHandler := handler.NewEntryHandler(entryService, apiBasePath)
	archiveHandler := handler.NewArchiveHandler(archiveService)
	rankingHandler := handler.NewRankingHandler(rankingService, apiBasePath)
	tagHandler := handler.NewTagHandler(tagService, entryService, apiBasePath)
	searchHandler := handler.NewSearchHandler(searchService, apiBasePath)
	metricsHandler := handler.NewMetricsHandler(metricsService)
	faviconHandler := handler.NewFaviconHandler(faviconService)
	healthHandler := &handler.HealthHandler{
		DB:    db,
		Cache: redisClient,
	}

	var middlewares []func(http.Handler) http.Handler
	var promHandler http.Handler
	if cfg.App.EnableMetrics {
		httpMetrics := metrics.NewHTTPMetrics()
		middlewares = append(middlewares, httpMetrics.Middleware)
		promHandler = httpMetrics.Handler()
		if cfg.App.APIKeyRequired {
			promHandler = server.APIKeyAuth(cfg.App.MasterAPIKey, log)(promHandler)
		}
	}
	if cfg.App.RateLimitEnabled {
		healthPath := apiBasePath + "/health"
		if apiBasePath == "/" {
			healthPath = "/health"
		}
		middlewares = append(middlewares, server.RateLimit(server.RateLimitConfig{
			Cache:  redisClient,
			Limit:  cfg.App.RateLimitMaxRequests,
			Window: cfg.App.RateLimitWindow,
			Logger: log,
			Prefix: "http",
			Skip: func(r *http.Request) bool {
				switch r.URL.Path {
				case healthPath:
					return true
				default:
					return false
				}
			},
		}))
	}

	router := handler.NewRouter(handler.RouterConfig{
		EntryHandler:      entryHandler,
		ArchiveHandler:    archiveHandler,
		RankingHandler:    rankingHandler,
		TagHandler:        tagHandler,
		SearchHandler:     searchHandler,
		MetricsHandler:    metricsHandler,
		FaviconHandler:    faviconHandler,
		HealthHandler:     healthHandler,
		APIBasePath:       apiBasePath,
		Middlewares:       middlewares,
		PrometheusHandler: promHandler,
	})

	srv := server.New(server.Config{
		Address:      cfg.Server.Address(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}, router, log)

	return srv.ListenAndServeWithGracefulShutdown()
}
