package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Config holds HTTP server configuration
type Config struct {
	Address      string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// Server wraps http.Server with graceful shutdown support
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

// New creates a new HTTP server
func New(cfg Config, handler http.Handler, logger *slog.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         cfg.Address,
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
		logger: logger,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("starting HTTP server", "address", s.httpServer.Addr)

	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down HTTP server")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	s.logger.Info("HTTP server stopped")
	return nil
}

// ListenAndServeWithGracefulShutdown starts the server and handles graceful shutdown
func (s *Server) ListenAndServeWithGracefulShutdown() error {
	// Channel to listen for errors from the server
	serverErrors := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		serverErrors <- s.Start()
	}()

	// Channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Block until we receive a signal or an error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		s.logger.Info("received shutdown signal", "signal", sig.String())

		// Create a context with timeout for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := s.Shutdown(ctx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
	}

	return nil
}

// Address returns the server address
func (s *Server) Address() string {
	return s.httpServer.Addr
}
