// Package models provides event model definitions.
package models

import "time"

// CartCreatedData represents data for cart.created event.
type CartCreatedData struct {
	CartID    string    `json:"cart_id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ItemAddedData represents data for cart.item_added event.
type ItemAddedData struct {
	CartID    string      `json:"cart_id"`
	UserID    string      `json:"user_id"`
	Item      CartItemDTO `json:"item"`
	CartTotal int64       `json:"cart_total"`
	ItemCount int         `json:"item_count"`
}

// ItemRemovedData represents data for cart.item_removed event.
type ItemRemovedData struct {
	CartID    string `json:"cart_id"`
	UserID    string `json:"user_id"`
	ItemID    string `json:"item_id"`
	ProductID string `json:"product_id"`
	CartTotal int64  `json:"cart_total"`
	ItemCount int    `json:"item_count"`
}

// ItemUpdatedData represents data for cart.item_updated event.
type ItemUpdatedData struct {
	CartID       string      `json:"cart_id"`
	UserID       string      `json:"user_id"`
	Item         CartItemDTO `json:"item"`
	PrevQuantity int         `json:"prev_quantity"`
	CartTotal    int64       `json:"cart_total"`
}

// CartClearedData represents data for cart.cleared event.
type CartClearedData struct {
	CartID         string `json:"cart_id"`
	UserID         string `json:"user_id"`
	ItemsRemoved   int    `json:"items_removed"`
	PreviousTotal  int64  `json:"previous_total"`
}

// CartAbandonedData represents data for cart.abandoned event.
type CartAbandonedData struct {
	CartID      string    `json:"cart_id"`
	UserID      string    `json:"user_id"`
	ItemCount   int       `json:"item_count"`
	CartTotal   int64     `json:"cart_total"`
	LastUpdated time.Time `json:"last_updated"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// CartItemDTO represents a cart item in events.
type CartItemDTO struct {
	ItemID    string    `json:"item_id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	UnitPrice int64     `json:"unit_price"`
	Subtotal  int64     `json:"subtotal"`
	AddedAt   time.Time `json:"added_at"`
}
