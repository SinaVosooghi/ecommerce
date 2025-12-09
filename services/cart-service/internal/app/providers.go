// Package app provides the application container and dependency injection.
package app

import (
	"context"

	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/config"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/core/cart"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/logging"
)

// Option is a functional option for configuring the Application.
type Option func(*Application) error

// WithConfig sets the configuration.
func WithConfig(cfg *config.Config) Option {
	return func(a *Application) error {
		a.Config = cfg
		return nil
	}
}

// WithLogger sets the logger.
func WithLogger(logger *logging.Logger) Option {
	return func(a *Application) error {
		a.Logger = logger
		return nil
	}
}

// WithRepository sets the cart repository.
func WithRepository(repo CartRepository) Option {
	return func(a *Application) error {
		a.Repository = repo
		return nil
	}
}

// WithEventPublisher sets the event publisher.
func WithEventPublisher(pub EventPublisher) Option {
	return func(a *Application) error {
		a.Publisher = pub
		return nil
	}
}

// WithMetrics sets the metrics collector.
func WithMetrics(m MetricsCollector) Option {
	return func(a *Application) error {
		a.Metrics = m
		return nil
	}
}

// WithFeatureFlags sets the feature flags service.
func WithFeatureFlags(f FeatureFlags) Option {
	return func(a *Application) error {
		a.Features = f
		return nil
	}
}

// WithSecrets sets the secrets manager.
func WithSecrets(s SecretsManager) Option {
	return func(a *Application) error {
		a.Secrets = s
		return nil
	}
}

// CartRepository interface for cart persistence.
type CartRepository interface {
	GetCart(ctx context.Context, userID string) (*cart.Cart, error)
	SaveCart(ctx context.Context, c *cart.Cart) error
	SaveCartWithVersion(ctx context.Context, c *cart.Cart, expectedVersion int64) error
	DeleteCart(ctx context.Context, userID string) error
	HealthCheck(ctx context.Context) error
}

// EventPublisher interface for event publishing.
type EventPublisher interface {
	Publish(ctx context.Context, event interface{}) error
	PublishBatch(ctx context.Context, events []interface{}) error
}

// MetricsCollector interface for metrics collection.
type MetricsCollector interface {
	IncrementCounter(name string, labels map[string]string)
	ObserveHistogram(name string, value float64, labels map[string]string)
	SetGauge(name string, value float64, labels map[string]string)
}

// FeatureFlags interface for feature flag evaluation.
type FeatureFlags interface {
	IsEnabled(ctx context.Context, flag string, userID string) bool
	GetVariant(ctx context.Context, flag string, userID string) string
}

// SecretsManager interface for secrets retrieval.
type SecretsManager interface {
	GetSecret(ctx context.Context, key string) (string, error)
	GetSecretJSON(ctx context.Context, key string, target interface{}) error
}

// CircuitBreaker interface for circuit breaker pattern.
type CircuitBreaker interface {
	Execute(ctx context.Context, fn func() error) error
	State() string
}
