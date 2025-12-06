// Package handlers provides HTTP handlers for the cart API v1.
package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/go-playground/validator/v10"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/errors"
)

var (
	validate    = validator.New()
	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	alphanumPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// AddItemRequest represents a request to add an item to the cart.
type AddItemRequest struct {
	ProductID string `json:"product_id" validate:"required,max=64"`
	Quantity  int    `json:"quantity" validate:"required,min=1,max=99"`
	UnitPrice int64  `json:"unit_price" validate:"min=0,max=999999999"`
}

// UpdateQuantityRequest represents a request to update item quantity.
type UpdateQuantityRequest struct {
	Quantity int   `json:"quantity" validate:"required,min=1,max=99"`
	Version  int64 `json:"version" validate:"min=0"`
}

// MergeCartRequest represents a request to merge guest cart.
type MergeCartRequest struct {
	GuestID string `json:"guest_id" validate:"required,max=64"`
}

// Validate validates the request and returns an error if invalid.
func (r *AddItemRequest) Validate() error {
	if err := validate.Struct(r); err != nil {
		return errors.ErrValidation("Invalid request", validationErrors(err))
	}
	if !alphanumPattern.MatchString(r.ProductID) {
		return errors.ErrValidation("Invalid product_id format", map[string]interface{}{
			"product_id": "must be alphanumeric with underscores and hyphens only",
		})
	}
	return nil
}

// Validate validates the request and returns an error if invalid.
func (r *UpdateQuantityRequest) Validate() error {
	if err := validate.Struct(r); err != nil {
		return errors.ErrValidation("Invalid request", validationErrors(err))
	}
	return nil
}

// ValidateUserID validates a user ID.
func ValidateUserID(userID string) error {
	if userID == "" {
		return errors.ErrValidation("user_id is required", nil)
	}
	if len(userID) > 64 {
		return errors.ErrValidation("user_id too long", map[string]interface{}{
			"max_length": 64,
		})
	}
	// Allow UUIDs or alphanumeric IDs
	if !uuidPattern.MatchString(userID) && !alphanumPattern.MatchString(userID) {
		return errors.ErrValidation("Invalid user_id format", nil)
	}
	return nil
}

// ValidateItemID validates an item ID.
func ValidateItemID(itemID string) error {
	if itemID == "" {
		return errors.ErrValidation("item_id is required", nil)
	}
	if len(itemID) > 64 {
		return errors.ErrValidation("item_id too long", nil)
	}
	if !uuidPattern.MatchString(itemID) && !alphanumPattern.MatchString(itemID) {
		return errors.ErrValidation("Invalid item_id format", nil)
	}
	return nil
}

// decodeJSON decodes JSON from request body.
func decodeJSON(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return errors.ErrValidation("Request body is required", nil)
	}
	
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	
	if err := decoder.Decode(v); err != nil {
		return errors.ErrValidation("Invalid JSON", map[string]interface{}{
			"error": err.Error(),
		})
	}
	return nil
}

// validationErrors converts validator errors to a map.
func validationErrors(err error) map[string]interface{} {
	if err == nil {
		return nil
	}
	
	errs := make(map[string]interface{})
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrs {
			errs[e.Field()] = e.Tag()
		}
	}
	return errs
}
