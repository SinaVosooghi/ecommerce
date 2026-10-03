// Package main is the entry point for the cart service.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/api/middleware"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/api/v1/handlers"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/app"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/config"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/core/cart"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/events/eventbridge"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/health"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/logging"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/metrics"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/persistence/dynamodb"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/server"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	healthCheck := flag.Bool("health-check", false, "probe the local /health endpoint and exit 0 if healthy (for container health checks)")
	flag.Parse()

	if *healthCheck {
		port := os.Getenv("APP_PORT")
		if port == "" {
			port = "8080"
		}
		os.Exit(probeHealth("http://127.0.0.1:" + port + "/health"))
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// probeHealth returns 0 if url answers 200 OK within the timeout, and 1 otherwise.
// The runtime image has no shell or wget, so the binary checks itself.
func probeHealth(url string) int {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url) //nolint:gosec // url is the local health endpoint built from APP_PORT
	if err != nil {
		fmt.Fprintf(os.Stderr, "health check failed: %v\n", err)
		return 1
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "health check failed: status %d\n", resp.StatusCode)
		return 1
	}
	return 0
}

func run() error {
	// Cancelled on SIGINT/SIGTERM; background workers stop with it.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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

	logger.Infof("Starting cart service %s", version)
	logger.Infof("Environment: %s, Port: %d", cfg.Environment, cfg.Port)
	if !cfg.AuthEnabled {
		logger.Warn("Authentication is DISABLED; any caller can access any cart (dev only)")
	}

	// Initialize DynamoDB client
	dbClient, err := dynamodb.NewClient(ctx, dynamodb.ClientConfig{
		Region:    cfg.AWSRegion,
		Endpoint:  cfg.DynamoDBEndpoint,
		TableName: cfg.DynamoDBTable,
	})
	if err != nil {
		return fmt.Errorf("failed to create DynamoDB client: %w", err)
	}
	logger.Infof("Using DynamoDB table: %s", cfg.DynamoDBTable)

	repo := dynamodb.NewRepository(dbClient)

	// Event publishing. Leave the interface nil when disabled so the service skips it.
	var publisher cart.EventPublisher
	if cfg.EventBridgeEnabled {
		ebPublisher, err := eventbridge.NewPublisher(ctx, eventbridge.PublisherConfig{
			Region:  cfg.AWSRegion,
			BusName: cfg.EventBridgeBusName,
			Source:  cfg.EventBridgeSource,
		}, logger)
		if err != nil {
			return fmt.Errorf("failed to create EventBridge publisher: %w", err)
		}
		publisher = eventbridge.NewCartEventPublisher(ebPublisher)
	}

	// No product catalog exists yet, so prices come from the client.
	logger.Warn("No price validator configured; client-supplied unit_price is trusted")
	cartService := cart.NewService(repo, publisher, cart.ServiceConfig{
		PublishEvents: cfg.EventBridgeEnabled,
	})

	application, err := app.New(ctx,
		app.WithConfig(cfg),
		app.WithLogger(logger),
		app.WithRepository(repo),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	healthHandler := health.NewHandler()
	healthHandler.RegisterChecker(health.NewRepositoryChecker("dynamodb", repo.HealthCheck))

	deps := server.Deps{
		Cart:        handlers.NewCartHandler(cartService, logger),
		Health:      healthHandler,
		RateLimiter: middleware.NewRateLimiter(ctx, cfg.RateLimitRPS, cfg.RateLimitBurst),
	}
	if cfg.IdempotencyEnabled {
		deps.Idempotency = dynamodb.NewIdempotencyStore(dbClient)
	}
	if cfg.MetricsEnabled {
		deps.Metrics = metrics.NewCloudWatchCollector(metrics.CloudWatchConfig{
			Namespace:   "Ecommerce/CartService",
			ServiceName: cfg.ServiceName,
			Environment: cfg.Environment,
		})
	}

	srv, err := server.New(server.Config{
		Port:           cfg.Port,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}, application, deps)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Infof("Server listening on port %d", cfg.Port)
		serverErrors <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server error: %w", err)
		}
	case <-ctx.Done():
		logger.Info("Shutdown signal received, initiating graceful shutdown")
		// Restore default signal handling so a second signal terminates immediately.
		stop()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
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
