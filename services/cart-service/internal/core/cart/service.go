package cart

import (
	"context"
	"time"

	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/errors"
)

// maxSaveAttempts bounds how often a read-modify-write is retried after a concurrent update.
const maxSaveAttempts = 3

// Repository defines the interface for cart persistence.
type Repository interface {
	GetCart(ctx context.Context, userID string) (*Cart, error)
	SaveCart(ctx context.Context, cart *Cart) error
	// SaveCartWithVersion saves the cart only if the stored version equals expectedVersion
	// (0 means the cart must not exist yet). It returns a CodeConflict error otherwise.
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
	// Prices is optional. When set, AddItem uses the catalog price instead of the
	// client-supplied unit price. Without it the client price is trusted, which is only
	// acceptable until a catalog service exists.
	Prices PriceValidator
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

func (s *Service) publishing() bool {
	return s.config.PublishEvents && s.publisher != nil
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

// load returns the cart to modify and the version it is stored with (0 if not stored).
// When create is true, a missing or expired cart is replaced by a new empty cart.
func (s *Service) load(ctx context.Context, userID string, create bool) (cart *Cart, storedVersion int64, created bool, err error) {
	existing, err := s.repo.GetCart(ctx, userID)
	switch {
	case err == nil && !existing.IsExpired():
		return existing, existing.Version, false, nil
	case err == nil:
		if !create {
			return nil, 0, false, errors.ErrCartExpired(userID)
		}
		fresh := NewCart(userID)
		fresh.Version = existing.Version // keep versions monotonic across replacement
		return fresh, existing.Version, true, nil
	case errors.IsCode(err, errors.CodeCartNotFound):
		if !create {
			return nil, 0, false, err
		}
		fresh := NewCart(userID)
		fresh.Version = 0
		return fresh, 0, true, nil
	default:
		return nil, 0, false, errors.Wrap(errors.CodePersistenceError, "failed to get cart", err)
	}
}

// mutate applies fn to the user's cart and saves it with optimistic locking. If another
// request changed the cart in the meantime, it reloads the cart and re-applies fn so no
// update is lost. Errors returned by fn abort without saving.
func (s *Service) mutate(ctx context.Context, userID string, create bool, fn func(*Cart) error) (*Cart, bool, error) {
	for attempt := 1; ; attempt++ {
		cart, storedVersion, created, err := s.load(ctx, userID, create)
		if err != nil {
			return nil, false, err
		}

		if err := fn(cart); err != nil {
			return nil, false, err
		}

		cart.IncrementVersion()
		err = s.repo.SaveCartWithVersion(ctx, cart, storedVersion)
		if err == nil {
			return cart, created, nil
		}
		if !errors.IsCode(err, errors.CodeConflict) {
			return nil, false, errors.Wrap(errors.CodePersistenceError, "failed to save cart", err)
		}
		if attempt == maxSaveAttempts {
			return nil, false, err
		}
	}
}

// GetOrCreateCart retrieves a cart or creates a new one if it doesn't exist.
func (s *Service) GetOrCreateCart(ctx context.Context, userID string) (*Cart, bool, error) {
	cart, err := s.GetCart(ctx, userID)
	if err == nil {
		return cart, false, nil
	}
	if !errors.IsCode(err, errors.CodeCartNotFound) && !errors.IsCode(err, errors.CodeCartExpired) {
		return nil, false, err
	}

	cart, created, err := s.mutate(ctx, userID, true, func(*Cart) error { return nil })
	if err != nil {
		return nil, false, err
	}
	if created && s.publishing() {
		_ = s.publisher.PublishCartCreated(ctx, cart)
	}
	return cart, created, nil
}

// AddItemRequest represents a request to add an item to the cart.
type AddItemRequest struct {
	ProductID string
	Quantity  int
	UnitPrice int64
}

// AddItem adds an item to a user's cart, creating the cart if needed.
func (s *Service) AddItem(ctx context.Context, userID string, req AddItemRequest) (*Cart, error) {
	unitPrice := req.UnitPrice
	if s.config.Prices != nil {
		price, err := s.config.Prices.GetCurrentPrice(ctx, req.ProductID)
		if err != nil {
			return nil, errors.Wrap(errors.CodeServiceUnavailable, "failed to look up product price", err)
		}
		unitPrice = price
	}

	item := NewCartItem(req.ProductID, req.Quantity, unitPrice)

	cart, created, err := s.mutate(ctx, userID, true, func(c *Cart) error {
		// Domain logic handles validation
		return c.AddItem(item)
	})
	if err != nil {
		return nil, err
	}

	if s.publishing() {
		if created {
			_ = s.publisher.PublishCartCreated(ctx, cart)
		}
		_ = s.publisher.PublishItemAdded(ctx, cart, item)
	}

	return cart, nil
}

// UpdateItemRequest represents a request to update an item quantity.
type UpdateItemRequest struct {
	ItemID   string
	Quantity int
	// ExpectedVersion, when non-zero, is the cart version the client last saw. A mismatch
	// returns a conflict instead of overwriting newer changes.
	ExpectedVersion int64
}

// UpdateItemQuantity updates the quantity of an item in the cart.
func (s *Service) UpdateItemQuantity(ctx context.Context, userID string, req UpdateItemRequest) (*Cart, error) {
	cart, _, err := s.mutate(ctx, userID, false, func(c *Cart) error {
		if req.ExpectedVersion > 0 && c.Version != req.ExpectedVersion {
			return errors.ErrConflict(req.ExpectedVersion, c.Version)
		}
		// Domain logic handles validation
		return c.UpdateItemQuantity(req.ItemID, req.Quantity)
	})
	if err != nil {
		return nil, err
	}

	if s.publishing() {
		if item, _ := cart.FindItem(req.ItemID); item != nil {
			_ = s.publisher.PublishItemUpdated(ctx, cart, item)
		}
	}

	return cart, nil
}

// RemoveItem removes an item from the cart.
func (s *Service) RemoveItem(ctx context.Context, userID, itemID string) (*Cart, error) {
	cart, _, err := s.mutate(ctx, userID, false, func(c *Cart) error {
		// Domain logic handles validation
		return c.RemoveItem(itemID)
	})
	if err != nil {
		return nil, err
	}

	if s.publishing() {
		_ = s.publisher.PublishItemRemoved(ctx, cart, itemID)
	}

	return cart, nil
}

// ClearCart removes all items from the cart.
func (s *Service) ClearCart(ctx context.Context, userID string) error {
	cart, _, err := s.mutate(ctx, userID, false, func(c *Cart) error {
		c.Clear()
		return nil
	})
	if err != nil {
		if errors.IsCode(err, errors.CodeCartNotFound) {
			return nil // Cart doesn't exist, nothing to clear
		}
		return err
	}

	if s.publishing() {
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

// MergeGuestCart merges a guest cart into a user's cart and deletes the guest cart.
func (s *Service) MergeGuestCart(ctx context.Context, userID, guestID string) (*Cart, error) {
	if guestID == userID {
		return nil, errors.ErrValidation("guest_id must differ from the user ID", nil)
	}

	guestCart, err := s.repo.GetCart(ctx, guestID)
	switch {
	case errors.IsCode(err, errors.CodeCartNotFound):
		guestCart = nil
	case err != nil:
		return nil, errors.Wrap(errors.CodePersistenceError, "failed to get guest cart", err)
	case guestCart.IsExpired():
		guestCart = nil
	}

	mergedCart, created, err := s.mutate(ctx, userID, true, func(c *Cart) error {
		if guestCart != nil {
			MergeCarts(c, guestCart)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if created && s.publishing() {
		_ = s.publisher.PublishCartCreated(ctx, mergedCart)
	}

	// Delete guest cart (best effort; it expires on its own otherwise)
	if guestCart != nil {
		_ = s.repo.DeleteCart(ctx, guestID)
	}

	return mergedCart, nil
}

// TouchCart extends the expiration of a cart.
func (s *Service) TouchCart(ctx context.Context, userID string) error {
	_, _, err := s.mutate(ctx, userID, false, func(c *Cart) error {
		c.ExtendExpiration()
		return nil
	})
	return err
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
