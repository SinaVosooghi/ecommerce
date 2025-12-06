package resilience

import (
	"context"
	"fmt"
	"time"
)

// TimeoutConfig holds timeout configuration for different operations.
type TimeoutConfig struct {
	Default        time.Duration
	Read           time.Duration
	Write          time.Duration
	Connect        time.Duration
	ExternalAPI    time.Duration
}

// DefaultTimeoutConfig returns default timeout configuration.
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Default:     5 * time.Second,
		Read:        500 * time.Millisecond,
		Write:       1 * time.Second,
		Connect:     2 * time.Second,
		ExternalAPI: 10 * time.Second,
	}
}

// WithTimeout wraps a context with a timeout.
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// ExecuteWithTimeout executes a function with a timeout.
func ExecuteWithTimeout(ctx context.Context, timeout time.Duration, fn func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- fn(ctx)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("operation timed out after %v: %w", timeout, ctx.Err())
	}
}

// ExecuteWithTimeoutResult executes a function that returns a result with a timeout.
func ExecuteWithTimeoutResult[T any](ctx context.Context, timeout time.Duration, fn func(context.Context) (T, error)) (T, error) {
	var zero T
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type result struct {
		value T
		err   error
	}

	done := make(chan result, 1)
	go func() {
		v, err := fn(ctx)
		done <- result{value: v, err: err}
	}()

	select {
	case r := <-done:
		return r.value, r.err
	case <-ctx.Done():
		return zero, fmt.Errorf("operation timed out after %v: %w", timeout, ctx.Err())
	}
}

// Deadline returns the deadline from context if set, otherwise returns the default timeout.
func Deadline(ctx context.Context, defaultTimeout time.Duration) time.Time {
	if deadline, ok := ctx.Deadline(); ok {
		return deadline
	}
	return time.Now().Add(defaultTimeout)
}

// RemainingTime returns the remaining time until context deadline.
func RemainingTime(ctx context.Context) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		return time.Until(deadline)
	}
	return 0
}
