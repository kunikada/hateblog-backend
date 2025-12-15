package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	appEntry "hateblog/internal/app/entry"
	"hateblog/internal/infra/handler"
	infraPostgres "hateblog/internal/infra/postgres"
	infraRedis "hateblog/internal/infra/redis"
	"hateblog/internal/platform/cache"
	"hateblog/internal/platform/config"
	"hateblog/internal/platform/database"
	"hateblog/internal/platform/logger"
	"hateblog/internal/platform/server"
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
	entryCache := infraRedis.NewEntryListCache(redisClient, cfg.App.CacheTTL)
	entryService := appEntry.NewService(entryRepo, entryCache, log)

	entryHandler := handler.NewEntryHandler(entryService)
	healthHandler := &handler.HealthHandler{
		DB:    db,
		Cache: redisClient,
	}
	router := handler.NewRouter(entryHandler, healthHandler)

	srv := server.New(server.Config{
		Address:      cfg.Server.Address(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}, router, log)

	return srv.ListenAndServeWithGracefulShutdown()
}
