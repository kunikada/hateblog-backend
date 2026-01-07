package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	infraGoogle "hateblog/internal/infra/external/google"
	"hateblog/internal/infra/handler"
	infraPostgres "hateblog/internal/infra/postgres"
	infraRedis "hateblog/internal/infra/redis"
	"hateblog/internal/platform/cache"
	"hateblog/internal/platform/config"
	"hateblog/internal/platform/database"
	"hateblog/internal/platform/logger"
	"hateblog/internal/platform/metrics"
	"hateblog/internal/platform/migration"
	"hateblog/internal/platform/server"
	usecaseAPIKey "hateblog/internal/usecase/api_key"
	usecaseArchive "hateblog/internal/usecase/archive"
	usecaseEntry "hateblog/internal/usecase/entry"
	usecaseFavicon "hateblog/internal/usecase/favicon"
	usecaseMetrics "hateblog/internal/usecase/metrics"
	usecaseRanking "hateblog/internal/usecase/ranking"
	usecaseSearch "hateblog/internal/usecase/search"
	usecaseTag "hateblog/internal/usecase/tag"
)

// runMigrate handles migration subcommands.
func runMigrate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: migrate <up|down|version|force <version>>")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.New(logger.Config{
		Level:  logger.Level(cfg.App.LogLevel),
		Format: logger.Format(cfg.App.LogFormat),
	})

	migrator, err := migration.New(migration.Config{
		DatabaseURL:    cfg.Database.ConnectionString(),
		MigrationsPath: "file://migrations",
		Logger:         log,
	})
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer func() {
		if err := migrator.Close(); err != nil {
			log.Error("failed to close migrator", "error", err)
		}
	}()

	switch args[0] {
	case "up":
		log.Info("running migrations up")
		if err := migrator.Up(); err != nil {
			return fmt.Errorf("migration up failed: %w", err)
		}
		log.Info("migrations completed successfully")

	case "down":
		log.Info("rolling back one migration")
		if err := migrator.Down(); err != nil {
			return fmt.Errorf("migration down failed: %w", err)
		}
		log.Info("migration rolled back successfully")

	case "version":
		version, dirty, err := migrator.Version()
		if err != nil {
			return fmt.Errorf("failed to get version: %w", err)
		}
		if dirty {
			log.Info("current migration version", "version", version, "dirty", true)
		} else {
			log.Info("current migration version", "version", version)
		}

	case "force":
		if len(args) < 2 {
			return fmt.Errorf("usage: migrate force <version>")
		}
		var version int
		if _, err := fmt.Sscanf(args[1], "%d", &version); err != nil {
			return fmt.Errorf("invalid version number: %w", err)
		}
		log.Info("forcing migration version", "version", version)
		if err := migrator.Force(version); err != nil {
			return fmt.Errorf("migration force failed: %w", err)
		}
		log.Info("migration version forced successfully")

	default:
		return fmt.Errorf("unknown command: %s (available: up, down, version, force)", args[0])
	}

	return nil
}

// healthcheck performs a health check by calling the /health endpoint.
func healthcheck() error {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	apiBasePath := strings.TrimSpace(os.Getenv("APP_API_BASE_PATH"))
	if apiBasePath == "" {
		apiBasePath = "/api/v1"
	}
	if !strings.HasPrefix(apiBasePath, "/") {
		apiBasePath = "/" + apiBasePath
	}
	if apiBasePath != "/" {
		apiBasePath = strings.TrimRight(apiBasePath, "/")
	}

	healthPath := apiBasePath + "/health"
	url := fmt.Sprintf("http://localhost:%s%s", port, healthPath)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to call health endpoint: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close healthcheck response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unhealthy status code: %d", resp.StatusCode)
	}

	return nil
}

func main() {
	// Handle subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "healthcheck":
			if err := healthcheck(); err != nil {
				slog.Error("healthcheck failed", "error", err)
				os.Exit(1)
			}
			os.Exit(0)
		case "migrate":
			if err := runMigrate(os.Args[2:]); err != nil {
				slog.Error("migration command failed", "error", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}

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

	// Run database migrations
	log.Info("running database migrations")
	migrator, err := migration.New(migration.Config{
		DatabaseURL:    cfg.Database.ConnectionString(),
		MigrationsPath: "file://migrations",
		Logger:         log,
	})
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer func() {
		if err := migrator.Close(); err != nil {
			log.Error("failed to close migrator", "error", err)
		}
	}()

	if err := migrator.Up(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

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

	var (
		dayEntriesCache     *infraRedis.DayEntriesCache
		tagEntriesCache     *infraRedis.TagEntriesCache
		searchCache         *infraRedis.SearchCache
		tagsListCache       *infraRedis.TagsListCache
		archiveCache        *infraRedis.ArchiveCache
		yearlyRankingCache  *infraRedis.YearlyRankingCache
		monthlyRankingCache *infraRedis.MonthlyRankingCache
		weeklyRankingCache  *infraRedis.WeeklyRankingCache
		faviconCache        usecaseFavicon.Cache
	)

	if cfg.App.CacheEnabled {
		dayEntriesCache = infraRedis.NewDayEntriesCache(redisClient)
		tagEntriesCache = infraRedis.NewTagEntriesCache(redisClient)
		searchCache = infraRedis.NewSearchCache(redisClient)
		tagsListCache = infraRedis.NewTagsListCache(redisClient)
		archiveCache = infraRedis.NewArchiveCache(redisClient)
		yearlyRankingCache = infraRedis.NewYearlyRankingCache(redisClient)
		monthlyRankingCache = infraRedis.NewMonthlyRankingCache(redisClient)
		weeklyRankingCache = infraRedis.NewWeeklyRankingCache(redisClient)
		faviconCache = infraRedis.NewFaviconCache(redisClient, cfg.App.FaviconCacheTTL)
	}

	entryService := usecaseEntry.NewService(entryRepo, dayEntriesCache, tagEntriesCache, log)
	archiveService := usecaseArchive.NewService(entryRepo, archiveCache)
	rankingService := usecaseRanking.NewService(entryRepo, yearlyRankingCache, monthlyRankingCache, weeklyRankingCache)
	tagService := usecaseTag.NewService(tagRepo, tagsListCache)
	searchService := usecaseSearch.NewService(entryRepo, searchHistoryRepo, searchCache, log)
	metricsService := usecaseMetrics.NewService(entryRepo, clickMetricsRepo)

	// API Key service
	apiKeyRepo := infraRedis.NewAPIKeyRepository(redisClient)
	apiKeyService := usecaseAPIKey.NewService(apiKeyRepo, cfg.App.APIKeyPrefix)

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
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyService, cfg.App.APIKeyTTL)
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
			promHandler = server.DynamicAPIKeyAuth(apiKeyRepo, log)(promHandler)
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
	if cfg.App.APIKeyRequired {
		healthPath := apiBasePath + "/health"
		if apiBasePath == "/" {
			healthPath = "/health"
		}
		apiKeysPath := apiBasePath + "/api-keys"
		if apiBasePath == "/" {
			apiKeysPath = "/api-keys"
		}
		middlewares = append(middlewares, func(next http.Handler) http.Handler {
			dynamicAuth := server.DynamicAPIKeyAuth(apiKeyRepo, log)
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Skip authentication for health and api-keys generation endpoints
				if r.URL.Path == healthPath || r.URL.Path == apiKeysPath {
					next.ServeHTTP(w, r)
					return
				}
				dynamicAuth(next).ServeHTTP(w, r)
			})
		})
	}

	router := handler.NewRouter(handler.RouterConfig{
		EntryHandler:      entryHandler,
		ArchiveHandler:    archiveHandler,
		RankingHandler:    rankingHandler,
		TagHandler:        tagHandler,
		SearchHandler:     searchHandler,
		MetricsHandler:    metricsHandler,
		APIKeyHandler:     apiKeyHandler,
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
