package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/core/cart"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/logging"
)

// CartHandler handles cart-related HTTP requests.
type CartHandler struct {
	service *cart.Service
	logger  *logging.Logger
}

// NewCartHandler creates a new cart handler.
func NewCartHandler(service *cart.Service, logger *logging.Logger) *CartHandler {
	return &CartHandler{
		service: service,
		logger:  logger,
	}
}

// GetCart handles GET /v1/cart/{userID}
func (h *CartHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "userID")

	// Validate user ID
	if err := ValidateUserID(userID); err != nil {
		writeError(w, err)
		return
	}

	// Get cart
	c, err := h.service.GetCart(ctx, userID)
	if err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("Failed to get cart")
		writeError(w, err)
		return
	}

	writeSuccess(w, NewCartResponse(c))
}

// AddItem handles POST /v1/cart/{userID}/items
func (h *CartHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "userID")

	// Validate user ID
	if err := ValidateUserID(userID); err != nil {
		writeError(w, err)
		return
	}

	// Decode request
	var req AddItemRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		writeError(w, err)
		return
	}

	// Add item
	c, err := h.service.AddItem(ctx, userID, cart.AddItemRequest{
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		UnitPrice: req.UnitPrice,
	})
	if err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("Failed to add item")
		writeError(w, err)
		return
	}

	writeCreated(w, NewCartResponse(c))
}

// UpdateItem handles PATCH /v1/cart/{userID}/items/{itemID}
func (h *CartHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "userID")
	itemID := chi.URLParam(r, "itemID")

	// Validate IDs
	if err := ValidateUserID(userID); err != nil {
		writeError(w, err)
		return
	}
	if err := ValidateItemID(itemID); err != nil {
		writeError(w, err)
		return
	}

	// Decode request
	var req UpdateQuantityRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		writeError(w, err)
		return
	}

	// Update item
	c, err := h.service.UpdateItemQuantity(ctx, userID, cart.UpdateItemRequest{
		ItemID:          itemID,
		Quantity:        req.Quantity,
		ExpectedVersion: req.Version,
	})
	if err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("Failed to update item")
		writeError(w, err)
		return
	}

	writeSuccess(w, NewCartResponse(c))
}

// RemoveItem handles DELETE /v1/cart/{userID}/items/{itemID}
func (h *CartHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "userID")
	itemID := chi.URLParam(r, "itemID")

	// Validate IDs
	if err := ValidateUserID(userID); err != nil {
		writeError(w, err)
		return
	}
	if err := ValidateItemID(itemID); err != nil {
		writeError(w, err)
		return
	}

	// Remove item
	c, err := h.service.RemoveItem(ctx, userID, itemID)
	if err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("Failed to remove item")
		writeError(w, err)
		return
	}

	writeSuccess(w, NewCartResponse(c))
}

// ClearCart handles DELETE /v1/cart/{userID}
func (h *CartHandler) ClearCart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "userID")

	// Validate user ID
	if err := ValidateUserID(userID); err != nil {
		writeError(w, err)
		return
	}

	// Clear cart
	if err := h.service.ClearCart(ctx, userID); err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("Failed to clear cart")
		writeError(w, err)
		return
	}

	writeNoContent(w)
}

// MergeCart handles POST /v1/cart/{userID}/merge
func (h *CartHandler) MergeCart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "userID")

	// Validate user ID
	if err := ValidateUserID(userID); err != nil {
		writeError(w, err)
		return
	}

	// Decode request
	var req MergeCartRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	// Merge carts
	c, err := h.service.MergeGuestCart(ctx, userID, req.GuestID)
	if err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("Failed to merge cart")
		writeError(w, err)
		return
	}

	writeSuccess(w, NewCartResponse(c))
}
