// Package secrets provides secure credential retrieval for the cart service.
package secrets

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// Manager defines the interface for secrets management.
type Manager interface {
	GetSecret(ctx context.Context, key string) (string, error)
	GetSecretJSON(ctx context.Context, key string, target interface{}) error
}

// CachedManager wraps a Manager with caching.
type CachedManager struct {
	manager Manager
	cache   map[string]*cachedSecret
	ttl     time.Duration
	mu      sync.RWMutex
}

type cachedSecret struct {
	value     string
	expiresAt time.Time
}

// NewCachedManager creates a new cached secrets manager.
func NewCachedManager(manager Manager, ttl time.Duration) *CachedManager {
	return &CachedManager{
		manager: manager,
		cache:   make(map[string]*cachedSecret),
		ttl:     ttl,
	}
}

// GetSecret retrieves a secret, using cache if available.
func (c *CachedManager) GetSecret(ctx context.Context, key string) (string, error) {
	// Check cache
	c.mu.RLock()
	cached, ok := c.cache[key]
	c.mu.RUnlock()

	if ok && time.Now().Before(cached.expiresAt) {
		return cached.value, nil
	}

	// Fetch from underlying manager
	value, err := c.manager.GetSecret(ctx, key)
	if err != nil {
		return "", err
	}

	// Update cache
	c.mu.Lock()
	c.cache[key] = &cachedSecret{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return value, nil
}

// GetSecretJSON retrieves a secret and unmarshals it as JSON.
func (c *CachedManager) GetSecretJSON(ctx context.Context, key string, target interface{}) error {
	value, err := c.GetSecret(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(value), target)
}

// InvalidateCache clears the cache for a specific key.
func (c *CachedManager) InvalidateCache(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, key)
}

// InvalidateAll clears the entire cache.
func (c *CachedManager) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*cachedSecret)
}

// InMemoryManager is an in-memory implementation for testing.
type InMemoryManager struct {
	secrets map[string]string
	mu      sync.RWMutex
}

// NewInMemoryManager creates a new in-memory secrets manager.
func NewInMemoryManager() *InMemoryManager {
	return &InMemoryManager{
		secrets: make(map[string]string),
	}
}

// GetSecret retrieves a secret from memory.
func (m *InMemoryManager) GetSecret(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	value, ok := m.secrets[key]
	if !ok {
		return "", &SecretNotFoundError{Key: key}
	}
	return value, nil
}

// GetSecretJSON retrieves and unmarshals a secret.
func (m *InMemoryManager) GetSecretJSON(ctx context.Context, key string, target interface{}) error {
	value, err := m.GetSecret(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(value), target)
}

// SetSecret sets a secret (for testing).
func (m *InMemoryManager) SetSecret(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.secrets[key] = value
}

// SecretNotFoundError indicates a secret was not found.
type SecretNotFoundError struct {
	Key string
}

func (e *SecretNotFoundError) Error() string {
	return "secret not found: " + e.Key
}
