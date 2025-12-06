// Package health provides health and readiness check endpoints.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Checker defines the interface for health checks.
type Checker interface {
	Name() string
	Check(ctx context.Context) error
}

// Handler provides health and readiness endpoints.
type Handler struct {
	checkers []Checker
	mu       sync.RWMutex
}

// NewHandler creates a new health handler.
func NewHandler() *Handler {
	return &Handler{
		checkers: make([]Checker, 0),
	}
}

// RegisterChecker registers a health checker.
func (h *Handler) RegisterChecker(checker Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers = append(h.checkers, checker)
}

// HealthResponse represents the response from health endpoints.
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks,omitempty"`
}

// CheckResult represents the result of a single health check.
type CheckResult struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// LivenessHandler handles GET /health - always returns 200 OK.
func (h *Handler) LivenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
	})
}

// ReadinessHandler handles GET /ready - checks all dependencies.
func (h *Handler) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	h.mu.RLock()
	checkers := make([]Checker, len(h.checkers))
	copy(checkers, h.checkers)
	h.mu.RUnlock()

	checks := make(map[string]CheckResult)
	allHealthy := true

	// Run all checks
	for _, checker := range checkers {
		start := time.Now()
		err := checker.Check(ctx)
		latency := time.Since(start)

		result := CheckResult{
			Status:  "ok",
			Latency: latency.String(),
		}

		if err != nil {
			result.Status = "error"
			result.Message = err.Error()
			allHealthy = false
		}

		checks[checker.Name()] = result
	}

	response := HealthResponse{
		Timestamp: time.Now().UTC(),
		Checks:    checks,
	}

	w.Header().Set("Content-Type", "application/json")

	if allHealthy {
		response.Status = "ready"
		w.WriteHeader(http.StatusOK)
	} else {
		response.Status = "not ready"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}

// RepositoryChecker checks repository connectivity.
type RepositoryChecker struct {
	name      string
	checkFunc func(ctx context.Context) error
}

// NewRepositoryChecker creates a new repository checker.
func NewRepositoryChecker(name string, checkFunc func(ctx context.Context) error) *RepositoryChecker {
	return &RepositoryChecker{
		name:      name,
		checkFunc: checkFunc,
	}
}

// Name returns the checker name.
func (c *RepositoryChecker) Name() string {
	return c.name
}

// Check performs the health check.
func (c *RepositoryChecker) Check(ctx context.Context) error {
	return c.checkFunc(ctx)
}
