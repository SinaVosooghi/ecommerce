package cart

import (
	"context"
	"time"

	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/errors"
)

// Repository defines the interface for cart persistence.
type Repository interface {
	GetCart(ctx context.Context, userID string) (*Cart, error)
	SaveCart(ctx context.Context, cart *Cart) error
	SaveCartWithVersion(ctx context.Context, cart *Cart, expectedVersion int64) error
	DeleteCart(ctx context.Context, userID string) error
}

// EventPublisher defines the interface for publishing cart events.
type EventPublisher interface {
	PublishCartCreated(ctx context.Context, cart *Cart) error
	PublishItemAdded(ctx context.Context, cart *Cart, item *CartItem) error
	PublishItemRemoved(ctx context.Context, cart *Cart, itemID string) error
	PublishItemUpdated(ctx context.Context, cart *Cart, item *CartItem) error
	PublishCartCleared(ctx context.Context, cart *Cart) error
}

// ServiceConfig holds configuration for the cart service.
type ServiceConfig struct {
	PublishEvents bool
}

// Service provides cart business operations.
type Service struct {
	repo      Repository
	publisher EventPublisher
	config    ServiceConfig
}

// NewService creates a new cart service.
func NewService(repo Repository, publisher EventPublisher, config ServiceConfig) *Service {
	return &Service{
		repo:      repo,
		publisher: publisher,
		config:    config,
	}
}

// GetCart retrieves a cart for a user.
func (s *Service) GetCart(ctx context.Context, userID string) (*Cart, error) {
	cart, err := s.repo.GetCart(ctx, userID)
	if err != nil {
		if errors.IsCode(err, errors.CodeCartNotFound) {
			return nil, err
		}
		return nil, errors.Wrap(errors.CodePersistenceError, "failed to get cart", err)
	}

	if cart.IsExpired() {
		return nil, errors.ErrCartExpired(userID)
	}

	return cart, nil
}

// GetOrCreateCart retrieves a cart or creates a new one if it doesn't exist.
func (s *Service) GetOrCreateCart(ctx context.Context, userID string) (*Cart, bool, error) {
	cart, err := s.repo.GetCart(ctx, userID)
	if err != nil {
		if errors.IsCode(err, errors.CodeCartNotFound) {
			// Create new cart
			newCart := NewCart(userID)
			if err := s.repo.SaveCart(ctx, newCart); err != nil {
				return nil, false, errors.Wrap(errors.CodePersistenceError, "failed to create cart", err)
			}

			// Publish event
			if s.config.PublishEvents && s.publisher != nil {
				_ = s.publisher.PublishCartCreated(ctx, newCart)
			}

			return newCart, true, nil
		}
		return nil, false, errors.Wrap(errors.CodePersistenceError, "failed to get cart", err)
	}

	if cart.IsExpired() {
		// Create new cart for expired cart
		newCart := NewCart(userID)
		if err := s.repo.SaveCart(ctx, newCart); err != nil {
			return nil, false, errors.Wrap(errors.CodePersistenceError, "failed to create cart", err)
		}

		if s.config.PublishEvents && s.publisher != nil {
			_ = s.publisher.PublishCartCreated(ctx, newCart)
		}

		return newCart, true, nil
	}

	return cart, false, nil
}

// AddItemRequest represents a request to add an item to the cart.
type AddItemRequest struct {
	ProductID string
	Quantity  int
	UnitPrice int64
}

// AddItem adds an item to a user's cart.
func (s *Service) AddItem(ctx context.Context, userID string, req AddItemRequest) (*Cart, error) {
	// Get or create cart
	cart, _, err := s.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Create cart item
	item := NewCartItem(req.ProductID, req.Quantity, req.UnitPrice)

	// Add item to cart (domain logic handles validation)
	if err := cart.AddItem(item); err != nil {
		return nil, err
	}

	// Increment version and save
	cart.IncrementVersion()
	if err := s.repo.SaveCart(ctx, cart); err != nil {
		return nil, errors.Wrap(errors.CodePersistenceError, "failed to save cart", err)
	}

	// Publish event
	if s.config.PublishEvents && s.publisher != nil {
		_ = s.publisher.PublishItemAdded(ctx, cart, item)
	}

	return cart, nil
}

// UpdateItemRequest represents a request to update an item quantity.
type UpdateItemRequest struct {
	ItemID          string
	Quantity        int
	ExpectedVersion int64
}

// UpdateItemQuantity updates the quantity of an item in the cart.
func (s *Service) UpdateItemQuantity(ctx context.Context, userID string, req UpdateItemRequest) (*Cart, error) {
	cart, err := s.GetCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Check version for optimistic locking
	if req.ExpectedVersion > 0 && cart.Version != req.ExpectedVersion {
		return nil, errors.ErrConflict(req.ExpectedVersion, cart.Version)
	}

	// Update quantity (domain logic handles validation)
	if err := cart.UpdateItemQuantity(req.ItemID, req.Quantity); err != nil {
		return nil, err
	}

	// Get the updated item for event
	item, _ := cart.FindItem(req.ItemID)

	// Increment version and save with optimistic locking
	expectedVersion := cart.Version
	cart.IncrementVersion()

	if err := s.repo.SaveCartWithVersion(ctx, cart, expectedVersion); err != nil {
		if errors.IsCode(err, errors.CodeConflict) {
			return nil, err
		}
		return nil, errors.Wrap(errors.CodePersistenceError, "failed to save cart", err)
	}

	// Publish event
	if s.config.PublishEvents && s.publisher != nil && item != nil {
		_ = s.publisher.PublishItemUpdated(ctx, cart, item)
	}

	return cart, nil
}

// RemoveItem removes an item from the cart.
func (s *Service) RemoveItem(ctx context.Context, userID, itemID string) (*Cart, error) {
	cart, err := s.GetCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Remove item (domain logic handles validation)
	if err := cart.RemoveItem(itemID); err != nil {
		return nil, err
	}

	// Save cart
	cart.IncrementVersion()
	if err := s.repo.SaveCart(ctx, cart); err != nil {
		return nil, errors.Wrap(errors.CodePersistenceError, "failed to save cart", err)
	}

	// Publish event
	if s.config.PublishEvents && s.publisher != nil {
		_ = s.publisher.PublishItemRemoved(ctx, cart, itemID)
	}

	return cart, nil
}

// ClearCart removes all items from the cart.
func (s *Service) ClearCart(ctx context.Context, userID string) error {
	cart, err := s.GetCart(ctx, userID)
	if err != nil {
		if errors.IsCode(err, errors.CodeCartNotFound) {
			return nil // Cart doesn't exist, nothing to clear
		}
		return err
	}

	cart.Clear()
	cart.IncrementVersion()

	if err := s.repo.SaveCart(ctx, cart); err != nil {
		return errors.Wrap(errors.CodePersistenceError, "failed to save cart", err)
	}

	// Publish event
	if s.config.PublishEvents && s.publisher != nil {
		_ = s.publisher.PublishCartCleared(ctx, cart)
	}

	return nil
}

// DeleteCart deletes a cart entirely.
func (s *Service) DeleteCart(ctx context.Context, userID string) error {
	if err := s.repo.DeleteCart(ctx, userID); err != nil {
		if errors.IsCode(err, errors.CodeCartNotFound) {
			return nil
		}
		return errors.Wrap(errors.CodePersistenceError, "failed to delete cart", err)
	}
	return nil
}

// MergeGuestCart merges a guest cart into a user's cart.
func (s *Service) MergeGuestCart(ctx context.Context, userID, guestID string) (*Cart, error) {
	// Get user cart (or create new one)
	userCart, _, err := s.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get guest cart
	guestCart, err := s.repo.GetCart(ctx, guestID)
	if err != nil {
		if errors.IsCode(err, errors.CodeCartNotFound) {
			// No guest cart to merge
			return userCart, nil
		}
		return nil, errors.Wrap(errors.CodePersistenceError, "failed to get guest cart", err)
	}

	// Merge carts
	mergedCart := MergeCarts(userCart, guestCart)
	mergedCart.IncrementVersion()

	// Save merged cart
	if err := s.repo.SaveCart(ctx, mergedCart); err != nil {
		return nil, errors.Wrap(errors.CodePersistenceError, "failed to save merged cart", err)
	}

	// Delete guest cart
	_ = s.repo.DeleteCart(ctx, guestID)

	return mergedCart, nil
}

// TouchCart extends the expiration of a cart.
func (s *Service) TouchCart(ctx context.Context, userID string) error {
	cart, err := s.GetCart(ctx, userID)
	if err != nil {
		return err
	}

	cart.ExtendExpiration()
	return s.repo.SaveCart(ctx, cart)
}

// GetCartSummary returns a summary of the cart.
func (s *Service) GetCartSummary(ctx context.Context, userID string) (*CartSummary, error) {
	cart, err := s.GetCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	summary := cart.Summary()
	return &summary, nil
}

// AbandonedCartCriteria defines criteria for finding abandoned carts.
type AbandonedCartCriteria struct {
	InactiveSince time.Time
	Limit         int
}
