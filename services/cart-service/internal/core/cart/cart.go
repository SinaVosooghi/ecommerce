// Package cart provides the domain logic for shopping cart operations.
package cart

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/errors"
)

// Cart limits - business rules
const (
	MaxItemsPerCart    = 100
	MaxQuantityPerItem = 99
	MinQuantityPerItem = 1
	CartExpirationDays = 7
)

// Cart represents a shopping cart.
type Cart struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Items     []CartItem `json:"items"`
	Version   int64      `json:"version"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ExpiresAt time.Time  `json:"expires_at"`
}

// CartItem represents an item in the cart.
type CartItem struct {
	ItemID    string    `json:"item_id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	UnitPrice int64     `json:"unit_price"` // In cents
	AddedAt   time.Time `json:"added_at"`
}

// NewCart creates a new cart for a user.
func NewCart(userID string) *Cart {
	now := time.Now().UTC()
	return &Cart{
		ID:        uuid.New().String(),
		UserID:    userID,
		Items:     make([]CartItem, 0),
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(CartExpirationDays * 24 * time.Hour),
	}
}

// NewCartItem creates a new cart item.
func NewCartItem(productID string, quantity int, unitPrice int64) *CartItem {
	return &CartItem{
		ItemID:    uuid.New().String(),
		ProductID: productID,
		Quantity:  quantity,
		UnitPrice: unitPrice,
		AddedAt:   time.Now().UTC(),
	}
}

// IsExpired checks if the cart has expired.
func (c *Cart) IsExpired() bool {
	return time.Now().UTC().After(c.ExpiresAt)
}

// ItemCount returns the number of items in the cart.
func (c *Cart) ItemCount() int {
	return len(c.Items)
}

// TotalQuantity returns the total quantity of all items.
func (c *Cart) TotalQuantity() int {
	total := 0
	for _, item := range c.Items {
		total += item.Quantity
	}
	return total
}

// TotalPrice returns the total price in cents.
func (c *Cart) TotalPrice() int64 {
	var total int64
	for _, item := range c.Items {
		total += item.UnitPrice * int64(item.Quantity)
	}
	return total
}

// FindItem finds an item by its ID.
func (c *Cart) FindItem(itemID string) (*CartItem, int) {
	for i, item := range c.Items {
		if item.ItemID == itemID {
			return &c.Items[i], i
		}
	}
	return nil, -1
}

// FindItemByProductID finds an item by product ID.
func (c *Cart) FindItemByProductID(productID string) (*CartItem, int) {
	for i, item := range c.Items {
		if item.ProductID == productID {
			return &c.Items[i], i
		}
	}
	return nil, -1
}

// AddItem adds an item to the cart or updates quantity if product already exists.
func (c *Cart) AddItem(item *CartItem) error {
	// Validate quantity
	if err := ValidateQuantity(item.Quantity); err != nil {
		return err
	}

	// Check if product already exists in cart
	if existing, idx := c.FindItemByProductID(item.ProductID); existing != nil {
		// Update quantity
		newQuantity := existing.Quantity + item.Quantity
		if newQuantity > MaxQuantityPerItem {
			return errors.ErrQuantityLimitExceeded(newQuantity, MaxQuantityPerItem)
		}
		c.Items[idx].Quantity = newQuantity
		c.Items[idx].UnitPrice = item.UnitPrice // Update price
		c.UpdatedAt = time.Now().UTC()
		return nil
	}

	// Check cart item limit
	if len(c.Items) >= MaxItemsPerCart {
		return errors.ErrCartLimitExceeded(len(c.Items), MaxItemsPerCart)
	}

	// Add new item
	c.Items = append(c.Items, *item)
	c.UpdatedAt = time.Now().UTC()
	return nil
}

// RemoveItem removes an item from the cart by item ID.
func (c *Cart) RemoveItem(itemID string) error {
	_, idx := c.FindItem(itemID)
	if idx == -1 {
		return errors.ErrItemNotFound(c.UserID, itemID)
	}

	// Remove item by swapping with last and truncating
	c.Items[idx] = c.Items[len(c.Items)-1]
	c.Items = c.Items[:len(c.Items)-1]
	c.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateItemQuantity updates the quantity of an item.
func (c *Cart) UpdateItemQuantity(itemID string, quantity int) error {
	if err := ValidateQuantity(quantity); err != nil {
		return err
	}

	item, _ := c.FindItem(itemID)
	if item == nil {
		return errors.ErrItemNotFound(c.UserID, itemID)
	}

	item.Quantity = quantity
	c.UpdatedAt = time.Now().UTC()
	return nil
}

// Clear removes all items from the cart.
func (c *Cart) Clear() {
	c.Items = make([]CartItem, 0)
	c.UpdatedAt = time.Now().UTC()
}

// IncrementVersion increments the cart version for optimistic locking.
func (c *Cart) IncrementVersion() {
	c.Version++
	c.UpdatedAt = time.Now().UTC()
}

// ExtendExpiration extends the cart expiration time.
func (c *Cart) ExtendExpiration() {
	c.ExpiresAt = time.Now().UTC().Add(CartExpirationDays * 24 * time.Hour)
	c.UpdatedAt = time.Now().UTC()
}

// ValidateQuantity validates that quantity is within allowed limits.
func ValidateQuantity(quantity int) error {
	if quantity < MinQuantityPerItem {
		return errors.ErrInvalidQuantity(quantity)
	}
	if quantity > MaxQuantityPerItem {
		return errors.ErrQuantityLimitExceeded(quantity, MaxQuantityPerItem)
	}
	return nil
}

// MergeCarts merges a guest cart into a user cart.
// For duplicate products, keeps the higher quantity.
func MergeCarts(userCart, guestCart *Cart) *Cart {
	if userCart == nil {
		if guestCart != nil {
			guestCart.UpdatedAt = time.Now().UTC()
		}
		return guestCart
	}

	if guestCart == nil {
		return userCart
	}

	for _, guestItem := range guestCart.Items {
		if existing, _ := userCart.FindItemByProductID(guestItem.ProductID); existing != nil {
			// Keep higher quantity
			if guestItem.Quantity > existing.Quantity {
				existing.Quantity = guestItem.Quantity
			}
		} else {
			// Add new item if cart isn't full
			if len(userCart.Items) < MaxItemsPerCart {
				userCart.Items = append(userCart.Items, guestItem)
			}
		}
	}

	userCart.UpdatedAt = time.Now().UTC()
	return userCart
}

// PriceValidator interface for validating prices with product catalog.
type PriceValidator interface {
	ValidatePrice(ctx context.Context, productID string, price int64) (bool, error)
	GetCurrentPrice(ctx context.Context, productID string) (int64, error)
}

// InventoryChecker interface for checking stock availability.
type InventoryChecker interface {
	CheckAvailability(ctx context.Context, productID string, quantity int) (bool, error)
	ReserveStock(ctx context.Context, productID string, quantity int) (reservationID string, err error)
	ReleaseReservation(ctx context.Context, reservationID string) error
}

// CartSummary provides a summary of the cart for API responses.
type CartSummary struct {
	ID            string `json:"id"`
	UserID        string `json:"user_id"`
	ItemCount     int    `json:"item_count"`
	TotalQuantity int    `json:"total_quantity"`
	TotalPrice    int64  `json:"total_price"`
	Version       int64  `json:"version"`
}

// Summary returns a summary of the cart.
func (c *Cart) Summary() CartSummary {
	return CartSummary{
		ID:            c.ID,
		UserID:        c.UserID,
		ItemCount:     c.ItemCount(),
		TotalQuantity: c.TotalQuantity(),
		TotalPrice:    c.TotalPrice(),
		Version:       c.Version,
	}
}
