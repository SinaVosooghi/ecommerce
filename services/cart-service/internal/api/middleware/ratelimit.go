package middleware

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/errors"
	"golang.org/x/time/rate"
)

// RateLimiter provides rate limiting middleware.
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rps      rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(rps int, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

// getLimiter returns a rate limiter for the given key.
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[key]
	rl.mu.RUnlock()

	if exists {
		return limiter
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists = rl.limiters[key]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(rl.rps, rl.burst)
	rl.limiters[key] = limiter
	return limiter
}

// Middleware returns the rate limiting middleware.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client identifier (IP address or user ID)
		key := getClientKey(r)

		limiter := rl.getLimiter(key)
		if !limiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code":    errors.CodeRateLimited,
				"message": "Too many requests, please try again later",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getClientKey extracts the client identifier from the request.
func getClientKey(r *http.Request) string {
	// Try to get user ID from context first (set by auth middleware)
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		return "user:" + userID
	}

	// Fall back to IP address
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	return "ip:" + ip
}

// RateLimit creates a simple rate limit middleware with default settings.
func RateLimit(rps int, burst int) func(next http.Handler) http.Handler {
	limiter := NewRateLimiter(rps, burst)
	return limiter.Middleware
}
