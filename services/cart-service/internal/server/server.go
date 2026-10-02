// Package server provides HTTP server setup and lifecycle management.
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/api/middleware"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/api/v1/handlers"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/app"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/health"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/idempotency"
)

// requestTimeout bounds handler execution. It must stay below Config.WriteTimeout so the
// timeout response can still be written.
const requestTimeout = 10 * time.Second

// Config holds server configuration.
type Config struct {
	Port           int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	MaxHeaderBytes int
}

// Deps holds the collaborators the HTTP layer needs.
type Deps struct {
	Cart   *handlers.CartHandler
	Health *health.Handler
	// Optional: nil disables the corresponding middleware.
	Metrics     middleware.MetricsCollector
	RateLimiter *middleware.RateLimiter
	Idempotency idempotency.Store
}

// Server wraps the HTTP server with application context.
type Server struct {
	httpServer *http.Server
	app        *app.Application
	router     *chi.Mux
}

// New creates a new Server instance.
func New(cfg Config, application *app.Application, deps Deps) (*Server, error) {
	if application == nil || application.Config == nil || application.Logger == nil {
		return nil, errors.New("server: application with config and logger is required")
	}
	if deps.Cart == nil || deps.Health == nil {
		return nil, errors.New("server: cart and health handlers are required")
	}

	appCfg := application.Config
	router := chi.NewRouter()

	// Base middleware stack. Logging is outermost so every log line, including panics
	// caught by Recovery, carries the request ID.
	router.Use(middleware.Logger(application.Logger))
	if deps.Metrics != nil {
		router.Use(middleware.Metrics(deps.Metrics))
	}
	router.Use(middleware.Recovery(application.Logger))
	router.Use(middleware.SecurityHeaders)
	if len(appCfg.CORSAllowedOrigins) > 0 {
		router.Use(cors.Handler(cors.Options{
			AllowedOrigins: appCfg.CORSAllowedOrigins,
			AllowedMethods: appCfg.CORSAllowedMethods,
			AllowedHeaders: appCfg.CORSAllowedHeaders,
			ExposedHeaders: []string{"X-Request-ID", "X-Idempotent-Replayed"},
			// Auth uses bearer tokens, not cookies. Never combine credentials with a wildcard origin.
			AllowCredentials: !slices.Contains(appCfg.CORSAllowedOrigins, "*"),
			MaxAge:           300,
		}))
	}
	router.Use(middleware.RequestSizeLimit(appCfg.MaxRequestSize))
	router.Use(chimw.Timeout(requestTimeout))

	// Health check endpoints (no auth required)
	router.Get("/health", deps.Health.LivenessHandler)
	router.Get("/ready", deps.Health.ReadinessHandler)

	// API v1 routes
	router.Route("/v1", func(r chi.Router) {
		if appCfg.AuthEnabled {
			r.Use(middleware.JWTAuth(middleware.AuthConfig{
				JWTSecretKey: appCfg.JWTSecretKey,
				JWTIssuer:    appCfg.JWTIssuer,
				JWTAudience:  appCfg.JWTAudience,
			}))
		}
		// After auth, so authenticated callers are limited per user rather than per IP.
		if deps.RateLimiter != nil {
			r.Use(deps.RateLimiter.Middleware)
		}
		r.Use(middleware.ContentType("application/json"))

		r.Route("/cart/{userID}", func(r chi.Router) {
			if appCfg.AuthEnabled {
				r.Use(middleware.RequireOwner("userID"))
			}
			// After the ownership check, so a replay can never bypass it.
			r.Use(middleware.Idempotency(middleware.IdempotencyConfig{
				Enabled: appCfg.IdempotencyEnabled,
				TTL:     appCfg.IdempotencyTTL,
				Store:   deps.Idempotency,
			}))

			h := deps.Cart
			r.Get("/", h.GetCart)
			r.Delete("/", h.ClearCart)
			r.Post("/items", h.AddItem)
			r.Patch("/items/{itemID}", h.UpdateItem)
			r.Delete("/items/{itemID}", h.RemoveItem)
			r.Post("/merge", h.MergeCart)
		})
	})

	return &Server{
		httpServer: &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Port),
			Handler:           router,
			ReadTimeout:       cfg.ReadTimeout,
			ReadHeaderTimeout: cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
			MaxHeaderBytes:    cfg.MaxHeaderBytes,
		},
		app:    application,
		router: router,
	}, nil
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Close forcefully closes the server.
func (s *Server) Close() error {
	return s.httpServer.Close()
}

// Router returns the chi router for testing purposes.
func (s *Server) Router() *chi.Mux {
	return s.router
}
