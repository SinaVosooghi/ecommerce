package resilience

import (
	"context"
	"math/rand"
	"time"
)

// RetryConfig holds retry configuration.
type RetryConfig struct {
	MaxAttempts   int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	Multiplier    float64
	Jitter        bool
	RetryableFunc func(error) bool // Function to determine if error is retryable
}

// DefaultRetryConfig returns default configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
		RetryableFunc: func(err error) bool {
			return err != nil // Retry all errors by default
		},
	}
}

// Retry executes a function with retry logic.
func Retry(ctx context.Context, cfg RetryConfig, fn func() error) error {
	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		// Check context before each attempt
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Execute function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if cfg.RetryableFunc != nil && !cfg.RetryableFunc(err) {
			return err
		}

		// Don't wait after last attempt
		if attempt == cfg.MaxAttempts-1 {
			break
		}

		// Calculate next delay with exponential backoff
		waitTime := delay
		if cfg.Jitter {
			// Add jitter: 50% to 150% of delay
			jitterRange := float64(delay) * 0.5
			jitter := time.Duration(rand.Float64()*jitterRange*2 - jitterRange)
			waitTime = delay + jitter
		}

		// Wait with context
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
		}

		// Increase delay for next iteration
		delay = time.Duration(float64(delay) * cfg.Multiplier)
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}

	return lastErr
}

// RetryWithResult executes a function that returns a result with retry logic.
func RetryWithResult[T any](ctx context.Context, cfg RetryConfig, fn func() (T, error)) (T, error) {
	var result T
	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		result, lastErr = fn()
		if lastErr == nil {
			return result, nil
		}

		if cfg.RetryableFunc != nil && !cfg.RetryableFunc(lastErr) {
			return result, lastErr
		}

		if attempt == cfg.MaxAttempts-1 {
			break
		}

		waitTime := delay
		if cfg.Jitter {
			jitterRange := float64(delay) * 0.5
			jitter := time.Duration(rand.Float64()*jitterRange*2 - jitterRange)
			waitTime = delay + jitter
		}

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(waitTime):
		}

		delay = time.Duration(float64(delay) * cfg.Multiplier)
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}

	return result, lastErr
}

// WithRetry is a convenience function for simple retry scenarios.
func WithRetry(ctx context.Context, maxAttempts int, fn func() error) error {
	cfg := DefaultRetryConfig()
	cfg.MaxAttempts = maxAttempts
	return Retry(ctx, cfg, fn)
}
