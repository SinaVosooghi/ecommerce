// Package persistence provides data storage abstractions for the cart service.
package persistence

import (
	"context"

	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/core/cart"
)

// CartRepository defines the interface for cart persistence operations.
type CartRepository interface {
	// GetCart retrieves a cart by user ID.
	GetCart(ctx context.Context, userID string) (*cart.Cart, error)

	// SaveCart saves a cart (creates or updates).
	SaveCart(ctx context.Context, c *cart.Cart) error

	// SaveCartWithVersion saves a cart with optimistic locking.
	// Returns an error if the expected version doesn't match.
	SaveCartWithVersion(ctx context.Context, c *cart.Cart, expectedVersion int64) error

	// DeleteCart deletes a cart by user ID.
	DeleteCart(ctx context.Context, userID string) error

	// HealthCheck verifies repository connectivity.
	HealthCheck(ctx context.Context) error
}
