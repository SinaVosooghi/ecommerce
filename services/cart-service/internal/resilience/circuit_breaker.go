// Package resilience provides resilience patterns for the cart service.
package resilience

import (
	"context"
	"time"

	"github.com/sony/gobreaker"
)

// CircuitBreakerConfig holds circuit breaker configuration.
type CircuitBreakerConfig struct {
	Name              string
	MaxRequests       uint32        // Max requests allowed in half-open state
	Interval          time.Duration // Cyclic period for clearing counts
	Timeout           time.Duration // Time to wait before transitioning to half-open
	FailureThreshold  uint32        // Failures before opening
	SuccessThreshold  uint32        // Successes needed to close
	FailureRatio      float64       // Ratio of failures to total requests
}

// DefaultCircuitBreakerConfig returns default configuration.
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:             name,
		MaxRequests:      3,
		Interval:         10 * time.Second,
		Timeout:          30 * time.Second,
		FailureThreshold: 5,
		SuccessThreshold: 3,
		FailureRatio:     0.6,
	}
}

// CircuitBreaker wraps gobreaker with a simpler interface.
type CircuitBreaker struct {
	breaker *gobreaker.CircuitBreaker
	name    string
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	settings := gobreaker.Settings{
		Name:        cfg.Name,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Trip if we've hit the failure threshold
			if counts.ConsecutiveFailures >= cfg.FailureThreshold {
				return true
			}
			// Also trip if failure ratio is too high
			if counts.Requests >= 10 {
				failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
				return failureRatio >= cfg.FailureRatio
			}
			return false
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			// This could log or emit metrics
		},
	}

	return &CircuitBreaker{
		breaker: gobreaker.NewCircuitBreaker(settings),
		name:    cfg.Name,
	}
}

// Execute runs a function through the circuit breaker.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	_, err := cb.breaker.Execute(func() (interface{}, error) {
		return nil, fn()
	})
	return err
}

// ExecuteWithResult runs a function that returns a result through the circuit breaker.
func (cb *CircuitBreaker) ExecuteWithResult(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	return cb.breaker.Execute(fn)
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() string {
	state := cb.breaker.State()
	switch state {
	case gobreaker.StateClosed:
		return "closed"
	case gobreaker.StateHalfOpen:
		return "half-open"
	case gobreaker.StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// Name returns the circuit breaker name.
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// Counts returns the current counts.
func (cb *CircuitBreaker) Counts() gobreaker.Counts {
	return cb.breaker.Counts()
}

// IsOpen returns true if the circuit is open.
func (cb *CircuitBreaker) IsOpen() bool {
	return cb.breaker.State() == gobreaker.StateOpen
}
