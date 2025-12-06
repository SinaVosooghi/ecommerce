package cart

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCart(t *testing.T) {
	userID := "user-123"
	cart := NewCart(userID)

	assert.NotEmpty(t, cart.ID)
	assert.Equal(t, userID, cart.UserID)
	assert.Empty(t, cart.Items)
	assert.Equal(t, int64(1), cart.Version)
	assert.False(t, cart.IsExpired())
}

func TestNewCartItem(t *testing.T) {
	item := NewCartItem("product-123", 2, 1999)

	assert.NotEmpty(t, item.ItemID)
	assert.Equal(t, "product-123", item.ProductID)
	assert.Equal(t, 2, item.Quantity)
	assert.Equal(t, int64(1999), item.UnitPrice)
}

func TestCart_AddItem(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Cart)
		item      *CartItem
		wantErr   bool
		wantCount int
	}{
		{
			name:      "add first item",
			setup:     func(c *Cart) {},
			item:      NewCartItem("product-1", 1, 1000),
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "add item with existing product increases quantity",
			setup: func(c *Cart) {
				c.AddItem(NewCartItem("product-1", 2, 1000))
			},
			item:      NewCartItem("product-1", 3, 1000),
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:      "add item with invalid quantity",
			setup:     func(c *Cart) {},
			item:      NewCartItem("product-1", 0, 1000),
			wantErr:   true,
			wantCount: 0,
		},
		{
			name:      "add item exceeds quantity limit",
			setup:     func(c *Cart) {},
			item:      NewCartItem("product-1", 100, 1000),
			wantErr:   true,
			wantCount: 0,
		},
		{
			name: "add item to full cart",
			setup: func(c *Cart) {
				for i := 0; i < MaxItemsPerCart; i++ {
					c.Items = append(c.Items, CartItem{
						ItemID:    "item-" + string(rune(i)),
						ProductID: "product-" + string(rune(i)),
						Quantity:  1,
						UnitPrice: 100,
					})
				}
			},
			item:      NewCartItem("new-product", 1, 1000),
			wantErr:   true,
			wantCount: MaxItemsPerCart,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cart := NewCart("user-123")
			tt.setup(cart)

			err := cart.AddItem(tt.item)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantCount, cart.ItemCount())
		})
	}
}

func TestCart_AddItem_UpdatesQuantityForExistingProduct(t *testing.T) {
	cart := NewCart("user-123")
	
	err := cart.AddItem(NewCartItem("product-1", 2, 1000))
	require.NoError(t, err)
	
	err = cart.AddItem(NewCartItem("product-1", 3, 1000))
	require.NoError(t, err)

	assert.Equal(t, 1, cart.ItemCount())
	item, _ := cart.FindItemByProductID("product-1")
	assert.Equal(t, 5, item.Quantity)
}

func TestCart_RemoveItem(t *testing.T) {
	cart := NewCart("user-123")
	item := NewCartItem("product-1", 1, 1000)
	cart.AddItem(item)

	err := cart.RemoveItem(item.ItemID)
	assert.NoError(t, err)
	assert.Equal(t, 0, cart.ItemCount())

	// Try to remove non-existent item
	err = cart.RemoveItem("non-existent")
	assert.Error(t, err)
}

func TestCart_UpdateItemQuantity(t *testing.T) {
	tests := []struct {
		name         string
		quantity     int
		wantErr      bool
		wantQuantity int
	}{
		{
			name:         "valid quantity update",
			quantity:     5,
			wantErr:      false,
			wantQuantity: 5,
		},
		{
			name:         "quantity below minimum",
			quantity:     0,
			wantErr:      true,
			wantQuantity: 1,
		},
		{
			name:         "quantity above maximum",
			quantity:     100,
			wantErr:      true,
			wantQuantity: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cart := NewCart("user-123")
			item := NewCartItem("product-1", 1, 1000)
			cart.AddItem(item)

			err := cart.UpdateItemQuantity(item.ItemID, tt.quantity)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			foundItem, _ := cart.FindItem(item.ItemID)
			assert.Equal(t, tt.wantQuantity, foundItem.Quantity)
		})
	}
}

func TestCart_UpdateItemQuantity_NotFound(t *testing.T) {
	cart := NewCart("user-123")
	err := cart.UpdateItemQuantity("non-existent", 5)
	assert.Error(t, err)
}

func TestCart_Clear(t *testing.T) {
	cart := NewCart("user-123")
	cart.AddItem(NewCartItem("product-1", 1, 1000))
	cart.AddItem(NewCartItem("product-2", 2, 2000))

	assert.Equal(t, 2, cart.ItemCount())
	
	cart.Clear()
	
	assert.Equal(t, 0, cart.ItemCount())
}

func TestCart_TotalPrice(t *testing.T) {
	cart := NewCart("user-123")
	cart.AddItem(NewCartItem("product-1", 2, 1000)) // 2 x 1000 = 2000
	cart.AddItem(NewCartItem("product-2", 3, 500))  // 3 x 500 = 1500

	assert.Equal(t, int64(3500), cart.TotalPrice())
}

func TestCart_TotalQuantity(t *testing.T) {
	cart := NewCart("user-123")
	cart.AddItem(NewCartItem("product-1", 2, 1000))
	cart.AddItem(NewCartItem("product-2", 3, 500))

	assert.Equal(t, 5, cart.TotalQuantity())
}

func TestCart_IsExpired(t *testing.T) {
	cart := NewCart("user-123")
	assert.False(t, cart.IsExpired())

	// Set expiration to past
	cart.ExpiresAt = time.Now().UTC().Add(-1 * time.Hour)
	assert.True(t, cart.IsExpired())
}

func TestCart_ExtendExpiration(t *testing.T) {
	cart := NewCart("user-123")
	originalExpiry := cart.ExpiresAt

	time.Sleep(10 * time.Millisecond)
	cart.ExtendExpiration()

	assert.True(t, cart.ExpiresAt.After(originalExpiry))
}

func TestCart_IncrementVersion(t *testing.T) {
	cart := NewCart("user-123")
	assert.Equal(t, int64(1), cart.Version)

	cart.IncrementVersion()
	assert.Equal(t, int64(2), cart.Version)

	cart.IncrementVersion()
	assert.Equal(t, int64(3), cart.Version)
}

func TestCart_FindItem(t *testing.T) {
	cart := NewCart("user-123")
	item := NewCartItem("product-1", 1, 1000)
	cart.AddItem(item)

	found, idx := cart.FindItem(item.ItemID)
	assert.NotNil(t, found)
	assert.Equal(t, 0, idx)
	assert.Equal(t, item.ItemID, found.ItemID)

	notFound, idx := cart.FindItem("non-existent")
	assert.Nil(t, notFound)
	assert.Equal(t, -1, idx)
}

func TestCart_FindItemByProductID(t *testing.T) {
	cart := NewCart("user-123")
	item := NewCartItem("product-1", 1, 1000)
	cart.AddItem(item)

	found, idx := cart.FindItemByProductID("product-1")
	assert.NotNil(t, found)
	assert.Equal(t, 0, idx)

	notFound, idx := cart.FindItemByProductID("non-existent")
	assert.Nil(t, notFound)
	assert.Equal(t, -1, idx)
}

func TestCart_Summary(t *testing.T) {
	cart := NewCart("user-123")
	cart.AddItem(NewCartItem("product-1", 2, 1000))
	cart.AddItem(NewCartItem("product-2", 3, 500))

	summary := cart.Summary()

	assert.Equal(t, cart.ID, summary.ID)
	assert.Equal(t, cart.UserID, summary.UserID)
	assert.Equal(t, 2, summary.ItemCount)
	assert.Equal(t, 5, summary.TotalQuantity)
	assert.Equal(t, int64(3500), summary.TotalPrice)
	assert.Equal(t, cart.Version, summary.Version)
}

func TestMergeCarts(t *testing.T) {
	tests := []struct {
		name           string
		setupUserCart  func() *Cart
		setupGuestCart func() *Cart
		wantItemCount  int
		wantQuantity   map[string]int // product_id -> expected quantity
	}{
		{
			name: "nil user cart returns guest cart",
			setupUserCart: func() *Cart {
				return nil
			},
			setupGuestCart: func() *Cart {
				cart := NewCart("guest-123")
				cart.AddItem(NewCartItem("product-1", 2, 1000))
				return cart
			},
			wantItemCount: 1,
			wantQuantity:  map[string]int{"product-1": 2},
		},
		{
			name: "nil guest cart returns user cart",
			setupUserCart: func() *Cart {
				cart := NewCart("user-123")
				cart.AddItem(NewCartItem("product-1", 2, 1000))
				return cart
			},
			setupGuestCart: func() *Cart {
				return nil
			},
			wantItemCount: 1,
			wantQuantity:  map[string]int{"product-1": 2},
		},
		{
			name: "merge keeps higher quantity for duplicates",
			setupUserCart: func() *Cart {
				cart := NewCart("user-123")
				cart.AddItem(NewCartItem("product-1", 2, 1000))
				return cart
			},
			setupGuestCart: func() *Cart {
				cart := NewCart("guest-123")
				cart.AddItem(NewCartItem("product-1", 5, 1000))
				return cart
			},
			wantItemCount: 1,
			wantQuantity:  map[string]int{"product-1": 5},
		},
		{
			name: "merge adds new items from guest cart",
			setupUserCart: func() *Cart {
				cart := NewCart("user-123")
				cart.AddItem(NewCartItem("product-1", 2, 1000))
				return cart
			},
			setupGuestCart: func() *Cart {
				cart := NewCart("guest-123")
				cart.AddItem(NewCartItem("product-2", 3, 500))
				return cart
			},
			wantItemCount: 2,
			wantQuantity:  map[string]int{"product-1": 2, "product-2": 3},
		},
		{
			name: "merge combines duplicate and new items",
			setupUserCart: func() *Cart {
				cart := NewCart("user-123")
				cart.AddItem(NewCartItem("product-1", 2, 1000))
				return cart
			},
			setupGuestCart: func() *Cart {
				cart := NewCart("guest-123")
				cart.AddItem(NewCartItem("product-1", 5, 1000))
				cart.AddItem(NewCartItem("product-2", 3, 500))
				return cart
			},
			wantItemCount: 2,
			wantQuantity:  map[string]int{"product-1": 5, "product-2": 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userCart := tt.setupUserCart()
			guestCart := tt.setupGuestCart()

			result := MergeCarts(userCart, guestCart)

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			assert.Equal(t, tt.wantItemCount, result.ItemCount())

			for productID, expectedQty := range tt.wantQuantity {
				item, _ := result.FindItemByProductID(productID)
				require.NotNil(t, item, "expected to find product %s", productID)
				assert.Equal(t, expectedQty, item.Quantity)
			}
		})
	}
}

func TestValidateQuantity(t *testing.T) {
	tests := []struct {
		name     string
		quantity int
		wantErr  bool
	}{
		{"valid min quantity", 1, false},
		{"valid max quantity", 99, false},
		{"valid middle quantity", 50, false},
		{"zero quantity", 0, true},
		{"negative quantity", -1, true},
		{"exceeds max quantity", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateQuantity(tt.quantity)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
