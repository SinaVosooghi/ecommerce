package middleware

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_EvictsIdleClients(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rl := NewRateLimiter(ctx, 10, 10)
	now := time.Now()
	rl.now = func() time.Time { return now }

	rl.getLimiter("ip:1.1.1.1")
	rl.getLimiter("ip:2.2.2.2")

	now = now.Add(limiterIdleTTL / 2)
	rl.getLimiter("ip:2.2.2.2") // still active

	now = now.Add(limiterIdleTTL/2 + time.Second)
	rl.sweep()

	assert.Equal(t, 1, rl.Len())
}

func TestGetClientKey_IgnoresSpoofableHeaders(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:12345"
	r.Header.Set("X-Forwarded-For", "6.6.6.6")
	r.Header.Set("X-User-ID", "victim")

	assert.Equal(t, "ip:10.0.0.1", getClientKey(r))

	claims := &UserClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "user-1"}}
	r = withUser(r, claims)
	assert.Equal(t, "user:user-1", getClientKey(r))
}
