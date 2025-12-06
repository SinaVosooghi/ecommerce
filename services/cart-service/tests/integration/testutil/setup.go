// Package testutil provides test utilities for integration tests.
package testutil

import (
	"context"
	"testing"

	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/config"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/logging"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/persistence/inmemory"
)

// TestEnv holds test environment dependencies.
type TestEnv struct {
	Ctx    context.Context
	Config *config.Config
	Logger *logging.Logger
	Repo   *inmemory.Repository
}

// NewTestEnv creates a new test environment.
func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	ctx := context.Background()

	cfg := &config.Config{
		Port:                  8080,
		Environment:          "test",
		ServiceName:          "cart-service-test",
		LogLevel:             "debug",
		AWSRegion:            "us-east-1",
		DynamoDBTable:        "test-carts",
		RateLimitRPS:         100,
		RateLimitBurst:       200,
		MaxRequestSize:       1048576,
		IdempotencyEnabled:   true,
		IdempotencyTTL:       300,
		CircuitBreakerEnabled: true,
	}

	logger := logging.New(logging.Config{
		Level:       cfg.LogLevel,
		ServiceName: cfg.ServiceName,
		Environment: cfg.Environment,
	})

	repo := inmemory.NewRepository()

	return &TestEnv{
		Ctx:    ctx,
		Config: cfg,
		Logger: logger,
		Repo:   repo,
	}
}

// Cleanup cleans up test resources.
func (e *TestEnv) Cleanup() {
	e.Repo.Clear()
}
