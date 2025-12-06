// Package server provides HTTP server setup and lifecycle management.
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/app"
)

// Config holds server configuration.
type Config struct {
	Port           int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	MaxHeaderBytes int
}

// Server wraps the HTTP server with application context.
type Server struct {
	httpServer *http.Server
	app        *app.Application
	router     *chi.Mux
}

// New creates a new Server instance.
func New(cfg Config, application *app.Application) (*Server, error) {
	router := chi.NewRouter()

	// Base middleware stack
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))

	// CORS configuration
	if application.Config != nil {
		router.Use(cors.Handler(cors.Options{
			AllowedOrigins:   application.Config.CORSAllowedOrigins,
			AllowedMethods:   application.Config.CORSAllowedMethods,
			AllowedHeaders:   application.Config.CORSAllowedHeaders,
			ExposedHeaders:   []string{"Link", "X-Request-ID"},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	}

	srv := &Server{
		httpServer: &http.Server{
			Addr:           fmt.Sprintf(":%d", cfg.Port),
			Handler:        router,
			ReadTimeout:    cfg.ReadTimeout,
			WriteTimeout:   cfg.WriteTimeout,
			IdleTimeout:    cfg.IdleTimeout,
			MaxHeaderBytes: cfg.MaxHeaderBytes,
		},
		app:    application,
		router: router,
	}

	// Register routes
	srv.registerRoutes()

	return srv, nil
}

// registerRoutes sets up all HTTP routes.
func (s *Server) registerRoutes() {
	// Health check endpoints (no auth required)
	s.router.Get("/health", s.handleHealth)
	s.router.Get("/ready", s.handleReady)

	// API v1 routes
	s.router.Route("/v1", func(r chi.Router) {
		// Cart routes
		r.Route("/cart/{userID}", func(r chi.Router) {
			r.Get("/", s.handleGetCart)
			r.Delete("/", s.handleClearCart)
			r.Post("/items", s.handleAddItem)
			r.Patch("/items/{itemID}", s.handleUpdateItem)
			r.Delete("/items/{itemID}", s.handleRemoveItem)
		})
	})
}

// handleHealth is the liveness probe endpoint.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// handleReady is the readiness probe endpoint.
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if err := s.app.ReadinessCheck(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(fmt.Sprintf(`{"status":"not ready","error":"%s"}`, err.Error())))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}

// Placeholder handlers - will be implemented in Phase 4
func (s *Server) handleGetCart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(`{"error":"not implemented"}`))
}

func (s *Server) handleClearCart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(`{"error":"not implemented"}`))
}

func (s *Server) handleAddItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(`{"error":"not implemented"}`))
}

func (s *Server) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(`{"error":"not implemented"}`))
}

func (s *Server) handleRemoveItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(`{"error":"not implemented"}`))
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
