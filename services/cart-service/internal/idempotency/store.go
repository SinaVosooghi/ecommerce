// Package idempotency defines storage for Idempotency-Key request deduplication.
package idempotency

import (
	"context"
	"sync"
	"time"
)

// Record is the stored state for one idempotency key.
type Record struct {
	// Fingerprint identifies the original request (method, path and body hash).
	Fingerprint string
	// Completed is false while the original request is still being processed.
	Completed   bool
	StatusCode  int
	ContentType string
	Body        []byte
	ExpiresAt   time.Time
}

// Store persists idempotency records.
type Store interface {
	// Reserve atomically claims key for a new request. It returns nil if the caller now owns
	// the key, or the existing unexpired record if the key is already claimed.
	Reserve(ctx context.Context, key, fingerprint string, lockTTL time.Duration) (*Record, error)
	// Complete stores the final response for a key previously reserved by the caller.
	Complete(ctx context.Context, key string, record *Record, ttl time.Duration) error
	// Release removes a reservation so the request can be retried.
	Release(ctx context.Context, key string) error
}

// MemoryStore is an in-process Store for tests and local development. It is not shared
// between instances, so it does not deduplicate across multiple replicas.
type MemoryStore struct {
	records map[string]Record
	mu      sync.Mutex
	now     func() time.Time
}

// NewMemoryStore creates a MemoryStore that evicts expired records until ctx is cancelled.
func NewMemoryStore(ctx context.Context) *MemoryStore {
	s := &MemoryStore{
		records: make(map[string]Record),
		now:     time.Now,
	}
	go s.cleanupLoop(ctx)
	return s
}

// Reserve implements Store.
func (s *MemoryStore) Reserve(_ context.Context, key, fingerprint string, lockTTL time.Duration) (*Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if existing, ok := s.records[key]; ok && now.Before(existing.ExpiresAt) {
		return &existing, nil
	}

	s.records[key] = Record{Fingerprint: fingerprint, ExpiresAt: now.Add(lockTTL)}
	return nil, nil
}

// Complete implements Store.
func (s *MemoryStore) Complete(_ context.Context, key string, record *Record, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored := *record
	stored.Completed = true
	stored.ExpiresAt = s.now().Add(ttl)
	s.records[key] = stored
	return nil
}

// Release implements Store.
func (s *MemoryStore) Release(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, key)
	return nil
}

func (s *MemoryStore) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			now := s.now()
			for key, record := range s.records {
				if !now.Before(record.ExpiresAt) {
					delete(s.records, key)
				}
			}
			s.mu.Unlock()
		}
	}
}
