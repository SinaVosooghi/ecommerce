package resilience

import (
	"context"
	"fmt"
	"sync"
)

// BulkheadConfig holds bulkhead configuration.
type BulkheadConfig struct {
	Name          string
	MaxConcurrent int
	MaxWaiting    int
}

// DefaultBulkheadConfig returns default configuration.
func DefaultBulkheadConfig(name string) BulkheadConfig {
	return BulkheadConfig{
		Name:          name,
		MaxConcurrent: 10,
		MaxWaiting:    100,
	}
}

// Bulkhead implements the bulkhead pattern for isolating concurrent operations.
type Bulkhead struct {
	name          string
	semaphore     chan struct{}
	maxConcurrent int
	maxWaiting    int
	waiting       int
	mu            sync.Mutex
}

// NewBulkhead creates a new bulkhead.
func NewBulkhead(cfg BulkheadConfig) *Bulkhead {
	return &Bulkhead{
		name:          cfg.Name,
		semaphore:     make(chan struct{}, cfg.MaxConcurrent),
		maxConcurrent: cfg.MaxConcurrent,
		maxWaiting:    cfg.MaxWaiting,
	}
}

// Execute runs a function within the bulkhead limits.
func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error {
	// Check if we can accept more waiting requests
	b.mu.Lock()
	if b.waiting >= b.maxWaiting {
		b.mu.Unlock()
		return fmt.Errorf("bulkhead %s: max waiting requests exceeded", b.name)
	}
	b.waiting++
	b.mu.Unlock()

	// Decrement waiting count when done
	defer func() {
		b.mu.Lock()
		b.waiting--
		b.mu.Unlock()
	}()

	// Try to acquire semaphore
	select {
	case b.semaphore <- struct{}{}:
		// Acquired, release when done
		defer func() { <-b.semaphore }()
		return fn()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ExecuteWithResult runs a function that returns a result within the bulkhead limits.
func (b *Bulkhead) ExecuteWithResult(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	b.mu.Lock()
	if b.waiting >= b.maxWaiting {
		b.mu.Unlock()
		return nil, fmt.Errorf("bulkhead %s: max waiting requests exceeded", b.name)
	}
	b.waiting++
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		b.waiting--
		b.mu.Unlock()
	}()

	select {
	case b.semaphore <- struct{}{}:
		defer func() { <-b.semaphore }()
		return fn()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Stats returns current bulkhead statistics.
func (b *Bulkhead) Stats() BulkheadStats {
	b.mu.Lock()
	defer b.mu.Unlock()
	return BulkheadStats{
		Name:          b.name,
		Active:        len(b.semaphore),
		MaxConcurrent: b.maxConcurrent,
		Waiting:       b.waiting,
		MaxWaiting:    b.maxWaiting,
	}
}

// BulkheadStats contains bulkhead statistics.
type BulkheadStats struct {
	Name          string
	Active        int
	MaxConcurrent int
	Waiting       int
	MaxWaiting    int
}

// BulkheadManager manages multiple bulkheads.
type BulkheadManager struct {
	bulkheads map[string]*Bulkhead
	mu        sync.RWMutex
}

// NewBulkheadManager creates a new bulkhead manager.
func NewBulkheadManager() *BulkheadManager {
	return &BulkheadManager{
		bulkheads: make(map[string]*Bulkhead),
	}
}

// Get returns a bulkhead by name, creating it if it doesn't exist.
func (m *BulkheadManager) Get(name string, cfg BulkheadConfig) *Bulkhead {
	m.mu.RLock()
	if b, ok := m.bulkheads[name]; ok {
		m.mu.RUnlock()
		return b
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if b, ok := m.bulkheads[name]; ok {
		return b
	}

	cfg.Name = name
	b := NewBulkhead(cfg)
	m.bulkheads[name] = b
	return b
}

// AllStats returns stats for all bulkheads.
func (m *BulkheadManager) AllStats() map[string]BulkheadStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]BulkheadStats)
	for name, b := range m.bulkheads {
		stats[name] = b.Stats()
	}
	return stats
}
