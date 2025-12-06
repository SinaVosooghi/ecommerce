package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/core/cart"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/errors"
)

// CartResponse represents the API response for a cart.
type CartResponse struct {
	ID            string             `json:"id"`
	UserID        string             `json:"user_id"`
	Items         []CartItemResponse `json:"items"`
	ItemCount     int                `json:"item_count"`
	TotalQuantity int                `json:"total_quantity"`
	TotalPrice    int64              `json:"total_price"`
	Version       int64              `json:"version"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	ExpiresAt     time.Time          `json:"expires_at"`
}

// CartItemResponse represents the API response for a cart item.
type CartItemResponse struct {
	ItemID    string    `json:"item_id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	UnitPrice int64     `json:"unit_price"`
	Subtotal  int64     `json:"subtotal"`
	AddedAt   time.Time `json:"added_at"`
}

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// NewCartResponse creates a CartResponse from a cart domain object.
func NewCartResponse(c *cart.Cart) *CartResponse {
	items := make([]CartItemResponse, len(c.Items))
	for i, item := range c.Items {
		items[i] = CartItemResponse{
			ItemID:    item.ItemID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			Subtotal:  item.UnitPrice * int64(item.Quantity),
			AddedAt:   item.AddedAt,
		}
	}

	return &CartResponse{
		ID:            c.ID,
		UserID:        c.UserID,
		Items:         items,
		ItemCount:     c.ItemCount(),
		TotalQuantity: c.TotalQuantity(),
		TotalPrice:    c.TotalPrice(),
		Version:       c.Version,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
		ExpiresAt:     c.ExpiresAt,
	}
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, err error) {
	appErr, ok := errors.IsAppError(err)
	if !ok {
		// Unknown error - return internal error
		appErr = errors.ErrInternal(err)
	}

	resp := ErrorResponse{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
	}

	writeJSON(w, appErr.HTTPStatus, resp)
}

// writeSuccess writes a success response with optional data.
func writeSuccess(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, data)
}

// writeCreated writes a created response with optional data.
func writeCreated(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusCreated, data)
}

// writeNoContent writes a no content response.
func writeNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
