package middleware

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/errors"
	"golang.org/x/time/rate"
)

const (
	// limiterIdleTTL is how long an unused per-client limiter is kept before eviction.
	limiterIdleTTL = 10 * time.Minute
	// limiterSweepInterval is how often idle limiters are evicted.
	limiterSweepInterval = time.Minute
)

// RateLimiter provides per-client token-bucket rate limiting.
type RateLimiter struct {
	limiters map[string]*clientLimiter
	mu       sync.Mutex
	rps      rate.Limit
	burst    int
	now      func() time.Time
}

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates a new rate limiter. Idle client entries are evicted in the
// background until ctx is cancelled.
func NewRateLimiter(ctx context.Context, rps int, burst int) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*clientLimiter),
		rps:      rate.Limit(rps),
		burst:    burst,
		now:      time.Now,
	}
	go rl.sweepLoop(ctx)
	return rl
}

// getLimiter returns a rate limiter for the given key.
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cl, ok := rl.limiters[key]
	if !ok {
		cl = &clientLimiter{limiter: rate.NewLimiter(rl.rps, rl.burst)}
		rl.limiters[key] = cl
	}
	cl.lastSeen = rl.now()
	return cl.limiter
}

func (rl *RateLimiter) sweepLoop(ctx context.Context) {
	ticker := time.NewTicker(limiterSweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.sweep()
		}
	}
}

// sweep evicts limiters that have been idle longer than limiterIdleTTL.
func (rl *RateLimiter) sweep() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := rl.now().Add(-limiterIdleTTL)
	for key, cl := range rl.limiters {
		if cl.lastSeen.Before(cutoff) {
			delete(rl.limiters, key)
		}
	}
}

// Len returns the number of tracked clients.
func (rl *RateLimiter) Len() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return len(rl.limiters)
}

// Middleware returns the rate limiting middleware.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.getLimiter(getClientKey(r)).Allow() {
			w.Header().Set("Retry-After", "1")
			writeJSONError(w, http.StatusTooManyRequests, errors.CodeRateLimited, "Too many requests, please try again later")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getClientKey identifies the caller. Authenticated requests are keyed by the verified
// token subject; otherwise the TCP peer address is used. Client-supplied headers such as
// X-Forwarded-For are deliberately ignored because they can be spoofed to evade limits.
func getClientKey(r *http.Request) string {
	if claims := GetUserFromContext(r.Context()); claims != nil {
		return "user:" + claims.Subject
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return "ip:" + host
}
