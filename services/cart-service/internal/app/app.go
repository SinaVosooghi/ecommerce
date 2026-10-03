package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/config"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/logging"
)

// Application is the main application container that holds all dependencies.
type Application struct {
	Config *config.Config
	Logger *logging.Logger

	// Core dependencies
	Repository CartRepository
	Publisher  EventPublisher
	Metrics    MetricsCollector
	Features   FeatureFlags
	Secrets    SecretsManager

	// Resilience
	CircuitBreakers map[string]CircuitBreaker

	// Lifecycle management
	shutdownFuncs []func(context.Context) error
	mu            sync.Mutex
}

// New creates a new Application instance with the provided options.
func New(ctx context.Context, opts ...Option) (*Application, error) {
	app := &Application{
		CircuitBreakers: make(map[string]CircuitBreaker),
		shutdownFuncs:   make([]func(context.Context) error, 0),
	}

	// Apply all options
	for _, opt := range opts {
		if err := opt(app); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Validate required dependencies
	if app.Config == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	if app.Logger == nil {
		// Create default logger if not provided
		app.Logger = logging.New(logging.Config{
			Level:       app.Config.LogLevel,
			ServiceName: app.Config.ServiceName,
			Environment: app.Config.Environment,
		})
	}

	app.Logger.Info("Application initialized successfully")
	return app, nil
}

// RegisterShutdown registers a function to be called during graceful shutdown.
func (a *Application) RegisterShutdown(fn func(context.Context) error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.shutdownFuncs = append(a.shutdownFuncs, fn)
}

// Shutdown gracefully shuts down the application.
func (a *Application) Shutdown(ctx context.Context) error {
	a.Logger.Info("Starting graceful shutdown...")

	a.mu.Lock()
	funcs := make([]func(context.Context) error, len(a.shutdownFuncs))
	copy(funcs, a.shutdownFuncs)
	a.mu.Unlock()

	var firstErr error
	// Execute shutdown functions in reverse order (LIFO)
	for i := len(funcs) - 1; i >= 0; i-- {
		if err := funcs[i](ctx); err != nil {
			a.Logger.WithError(err).Error("Shutdown function failed")
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	if firstErr != nil {
		a.Logger.Error("Shutdown completed with errors")
		return firstErr
	}

	a.Logger.Info("Shutdown completed successfully")
	return nil
}

// GetCircuitBreaker returns a circuit breaker by name.
func (a *Application) GetCircuitBreaker(name string) (CircuitBreaker, bool) {
	cb, ok := a.CircuitBreakers[name]
	return cb, ok
}

// RegisterCircuitBreaker registers a circuit breaker.
func (a *Application) RegisterCircuitBreaker(name string, cb CircuitBreaker) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.CircuitBreakers[name] = cb
}
