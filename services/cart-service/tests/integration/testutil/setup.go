// Package testutil provides test utilities for integration tests.
package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/api/middleware"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/api/v1/handlers"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/app"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/config"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/core/cart"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/health"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/idempotency"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/logging"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/persistence/inmemory"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/server"
	"github.com/stretchr/testify/require"
)

// TestJWTSecret signs tokens in tests.
const TestJWTSecret = "integration-test-secret-0123456789abcdef"

// TestEnv holds a fully wired server backed by the in-memory repository.
type TestEnv struct {
	Ctx     context.Context
	Config  *config.Config
	Logger  *logging.Logger
	Repo    *inmemory.Repository
	Service *cart.Service
	Router  http.Handler
}

// Option customises the test configuration.
type Option func(*config.Config)

// NewTestEnv builds the production router (server.New) with in-memory dependencies.
func NewTestEnv(t *testing.T, opts ...Option) *TestEnv {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cfg := &config.Config{
		Port:               8080,
		Environment:        "dev",
		ServiceName:        "cart-service-test",
		LogLevel:           "error",
		AWSRegion:          "us-east-1",
		DynamoDBTable:      "test-carts",
		RateLimitRPS:       1000,
		RateLimitBurst:     1000,
		MaxRequestSize:     1 << 20,
		IdempotencyEnabled: true,
		IdempotencyTTL:     5 * time.Minute,
		AuthEnabled:        true,
		JWTSecretKey:       TestJWTSecret,
		CORSAllowedOrigins: []string{"*"},
		CORSAllowedMethods: []string{"GET", "POST", "PATCH", "DELETE"},
		CORSAllowedHeaders: []string{"Authorization", "Content-Type", "Idempotency-Key"},
	}
	for _, opt := range opts {
		opt(cfg)
	}

	logger := logging.New(logging.Config{
		Level:       cfg.LogLevel,
		ServiceName: cfg.ServiceName,
		Environment: cfg.Environment,
		Output:      io.Discard,
	})

	repo := inmemory.NewRepository()
	service := cart.NewService(repo, nil, cart.ServiceConfig{})

	application, err := app.New(ctx,
		app.WithConfig(cfg),
		app.WithLogger(logger),
		app.WithRepository(repo),
	)
	require.NoError(t, err)

	healthHandler := health.NewHandler()
	healthHandler.RegisterChecker(health.NewRepositoryChecker("repository", repo.HealthCheck))

	srv, err := server.New(server.Config{Port: cfg.Port}, application, server.Deps{
		Cart:        handlers.NewCartHandler(service, logger),
		Health:      healthHandler,
		RateLimiter: middleware.NewRateLimiter(ctx, cfg.RateLimitRPS, cfg.RateLimitBurst),
		Idempotency: idempotency.NewMemoryStore(ctx),
	})
	require.NoError(t, err)

	return &TestEnv{
		Ctx:     ctx,
		Config:  cfg,
		Logger:  logger,
		Repo:    repo,
		Service: service,
		Router:  srv.Router(),
	}
}

// Token returns a valid HS256 bearer token for userID.
func Token(t *testing.T, userID string) string {
	t.Helper()
	return SignToken(t, jwt.SigningMethodHS256, []byte(TestJWTSecret), jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
}

// SignToken signs arbitrary claims, for negative auth tests.
func SignToken(t *testing.T, method jwt.SigningMethod, key interface{}, claims jwt.Claims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString(key)
	require.NoError(t, err)
	return token
}

// Request describes an HTTP request sent through the router.
type Request struct {
	Method string
	Path   string
	// Token is sent as a bearer token when non-empty.
	Token   string
	Body    interface{}
	Headers map[string]string
}

// Do sends req through the router and returns the recorded response.
func (e *TestEnv) Do(t *testing.T, req Request) *httptest.ResponseRecorder {
	t.Helper()

	var body io.Reader
	if req.Body != nil {
		data, err := json.Marshal(req.Body)
		require.NoError(t, err)
		body = bytes.NewReader(data)
	}

	r := httptest.NewRequest(req.Method, req.Path, body)
	if req.Body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	if req.Token != "" {
		r.Header.Set("Authorization", "Bearer "+req.Token)
	}
	for k, v := range req.Headers {
		r.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	e.Router.ServeHTTP(w, r)
	return w
}

// Cleanup cleans up test resources.
func (e *TestEnv) Cleanup() {
	e.Repo.Clear()
}
