// Package inmemory provides an in-memory implementation of the cart repository for testing.
package inmemory

import (
	"context"
	"sync"

	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/core/cart"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/errors"
)

// Repository is an in-memory implementation of the cart repository.
type Repository struct {
	carts     map[string]*cart.Cart
	healthErr error
	mu        sync.RWMutex
}

// NewRepository creates a new in-memory repository.
func NewRepository() *Repository {
	return &Repository{
		carts: make(map[string]*cart.Cart),
	}
}

// GetCart retrieves a cart by user ID.
func (r *Repository) GetCart(ctx context.Context, userID string) (*cart.Cart, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, ok := r.carts[userID]
	if !ok {
		return nil, errors.ErrCartNotFound(userID)
	}

	// Return a copy to prevent external modification
	return copyCart(c), nil
}

// SaveCart saves a cart.
func (r *Repository) SaveCart(ctx context.Context, c *cart.Cart) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.carts[c.UserID] = copyCart(c)
	return nil
}

// SaveCartWithVersion saves a cart with optimistic locking. An expectedVersion of 0 means
// the cart must not exist yet; otherwise the stored version must equal expectedVersion.
func (r *Repository) SaveCartWithVersion(ctx context.Context, c *cart.Cart, expectedVersion int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.carts[c.UserID]
	switch {
	case !ok && expectedVersion != 0:
		return errors.ErrConflict(expectedVersion, 0)
	case ok && existing.Version != expectedVersion:
		return errors.ErrConflict(expectedVersion, existing.Version)
	}

	r.carts[c.UserID] = copyCart(c)
	return nil
}

// DeleteCart deletes a cart by user ID.
func (r *Repository) DeleteCart(ctx context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.carts[userID]; !ok {
		return errors.ErrCartNotFound(userID)
	}

	delete(r.carts, userID)
	return nil
}

// HealthCheck verifies repository is healthy. It returns the error set with SetHealthError.
func (r *Repository) HealthCheck(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.healthErr
}

// SetHealthError makes HealthCheck fail with err (useful for testing readiness).
func (r *Repository) SetHealthError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.healthErr = err
}

// Clear removes all carts (useful for testing).
func (r *Repository) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.carts = make(map[string]*cart.Cart)
}

// Count returns the number of carts (useful for testing).
func (r *Repository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.carts)
}

// copyCart creates a deep copy of a cart.
func copyCart(c *cart.Cart) *cart.Cart {
	if c == nil {
		return nil
	}

	items := make([]cart.CartItem, len(c.Items))
	copy(items, c.Items)

	return &cart.Cart{
		ID:        c.ID,
		UserID:    c.UserID,
		Items:     items,
		Version:   c.Version,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		ExpiresAt: c.ExpiresAt,
	}
}
