// Package main is the entry point for the cart service.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/app"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/config"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/logging"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/server"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Create base context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger
	logger := logging.New(logging.Config{
		Level:       cfg.LogLevel,
		ServiceName: cfg.ServiceName,
		Environment: cfg.Environment,
	})

	logger.Info("Starting cart service...")
	logger.Infof("Environment: %s, Port: %d", cfg.Environment, cfg.Port)

	// Initialize application container
	application, err := app.New(ctx,
		app.WithConfig(cfg),
		app.WithLogger(logger),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	// Initialize server
	srv, err := server.New(server.Config{
		Port:           cfg.Port,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}, application)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Infof("Server listening on port %d", cfg.Port)
		serverErrors <- srv.ListenAndServe()
	}()

	// Wait for shutdown signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	case sig := <-shutdown:
		logger.Infof("Received signal: %v, initiating graceful shutdown", sig)

		// Create shutdown context with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
		defer shutdownCancel()

		// Shutdown server
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.WithError(err).Error("Server shutdown error")
			// Force close if graceful shutdown fails
			if closeErr := srv.Close(); closeErr != nil {
				logger.WithError(closeErr).Error("Server close error")
			}
		}

		// Shutdown application dependencies
		if err := application.Shutdown(shutdownCtx); err != nil {
			logger.WithError(err).Error("Application shutdown error")
			return fmt.Errorf("application shutdown error: %w", err)
		}
	}

	logger.Info("Cart service stopped")
	return nil
}
