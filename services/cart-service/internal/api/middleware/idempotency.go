package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/errors"
)

// IdempotencyStore defines the interface for storing idempotency records.
type IdempotencyStore interface {
	Get(ctx context.Context, key string) (*IdempotencyRecord, error)
	Set(ctx context.Context, key string, record *IdempotencyRecord, ttl time.Duration) error
}

// IdempotencyRecord represents a stored idempotency response.
type IdempotencyRecord struct {
	StatusCode int       `json:"status_code"`
	Body       []byte    `json:"body"`
	Headers    http.Header `json:"headers"`
	CreatedAt  time.Time `json:"created_at"`
}

// IdempotencyConfig holds configuration for idempotency middleware.
type IdempotencyConfig struct {
	Enabled bool
	TTL     time.Duration
	Store   IdempotencyStore
}

// Idempotency provides idempotency middleware for safe retries.
func Idempotency(config IdempotencyConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only apply to methods that modify state
			if r.Method != http.MethodPost && r.Method != http.MethodPatch {
				next.ServeHTTP(w, r)
				return
			}

			if !config.Enabled || config.Store == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Get idempotency key from header
			idempotencyKey := r.Header.Get("Idempotency-Key")
			if idempotencyKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Get user ID for key scoping
			userID := r.Header.Get("X-User-ID")
			if userID == "" {
				userID = "anonymous"
			}

			// Create scoped key
			scopedKey := userID + ":" + idempotencyKey

			// Check for existing record
			record, err := config.Store.Get(r.Context(), scopedKey)
			if err == nil && record != nil {
				// Return cached response
				for key, values := range record.Headers {
					for _, value := range values {
						w.Header().Add(key, value)
					}
				}
				w.Header().Set("X-Idempotent-Replayed", "true")
				w.WriteHeader(record.StatusCode)
				w.Write(record.Body)
				return
			}

			// Capture response
			rw := &responseCapture{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				body:           &bytes.Buffer{},
			}

			next.ServeHTTP(rw, r)

			// Only cache successful responses
			if rw.statusCode >= 200 && rw.statusCode < 300 {
				newRecord := &IdempotencyRecord{
					StatusCode: rw.statusCode,
					Body:       rw.body.Bytes(),
					Headers:    rw.Header().Clone(),
					CreatedAt:  time.Now().UTC(),
				}
				config.Store.Set(r.Context(), scopedKey, newRecord, config.TTL)
			}
		})
	}
}

// responseCapture captures the response for idempotency storage.
type responseCapture struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (r *responseCapture) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseCapture) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// InMemoryIdempotencyStore provides an in-memory implementation of IdempotencyStore.
type InMemoryIdempotencyStore struct {
	records map[string]*storedRecord
	mu      sync.RWMutex
}

type storedRecord struct {
	record    *IdempotencyRecord
	expiresAt time.Time
}

// NewInMemoryIdempotencyStore creates a new in-memory idempotency store.
func NewInMemoryIdempotencyStore() *InMemoryIdempotencyStore {
	store := &InMemoryIdempotencyStore{
		records: make(map[string]*storedRecord),
	}
	// Start cleanup goroutine
	go store.cleanup()
	return store
}

// Get retrieves an idempotency record by key.
func (s *InMemoryIdempotencyStore) Get(ctx context.Context, key string) (*IdempotencyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stored, ok := s.records[key]
	if !ok {
		return nil, errors.New(errors.CodeCartNotFound, "Record not found")
	}

	if time.Now().After(stored.expiresAt) {
		return nil, errors.New(errors.CodeCartNotFound, "Record expired")
	}

	return stored.record, nil
}

// Set stores an idempotency record.
func (s *InMemoryIdempotencyStore) Set(ctx context.Context, key string, record *IdempotencyRecord, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records[key] = &storedRecord{
		record:    record,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

// cleanup periodically removes expired records.
func (s *InMemoryIdempotencyStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, stored := range s.records {
			if now.After(stored.expiresAt) {
				delete(s.records, key)
			}
		}
		s.mu.Unlock()
	}
}

// IdempotencyKeyRequired is middleware that requires an idempotency key for certain methods.
func IdempotencyKeyRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPatch {
			if r.Header.Get("Idempotency-Key") == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"code":    errors.CodeInvalidRequest,
					"message": "Idempotency-Key header is required for this request",
				})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// drainBody reads and returns the body, allowing it to be read again.
func drainBody(body io.ReadCloser) ([]byte, io.ReadCloser, error) {
	if body == nil {
		return nil, nil, nil
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, body, err
	}
	return data, io.NopCloser(bytes.NewReader(data)), nil
}
