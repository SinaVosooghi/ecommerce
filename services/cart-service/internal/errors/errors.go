package errors

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AppError represents a structured application error.
type AppError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	HTTPStatus int                    `json:"-"`
	Cause      error                  `json:"-"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause of the error.
func (e *AppError) Unwrap() error {
	return e.Cause
}

// MarshalJSON implements json.Marshaler for API responses.
func (e *AppError) MarshalJSON() ([]byte, error) {
	type errorResponse struct {
		Code    string                 `json:"code"`
		Message string                 `json:"message"`
		Details map[string]interface{} `json:"details,omitempty"`
	}
	return json.Marshal(errorResponse{
		Code:    e.Code,
		Message: e.Message,
		Details: e.Details,
	})
}

// New creates a new AppError with the given code and message.
func New(code, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: HTTPStatusForCode(code),
	}
}

// Newf creates a new AppError with a formatted message.
func Newf(code, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       code,
		Message:    fmt.Sprintf(format, args...),
		HTTPStatus: HTTPStatusForCode(code),
	}
}

// Wrap wraps an existing error with an AppError.
func Wrap(code string, message string, cause error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: HTTPStatusForCode(code),
		Cause:      cause,
	}
}

// WithDetails adds details to an AppError and returns it.
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	e.Details = details
	return e
}

// WithDetail adds a single detail to an AppError and returns it.
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// IsAppError checks if an error is an AppError and returns it.
func IsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// IsCode checks if an error has a specific error code.
func IsCode(err error, code string) bool {
	if appErr, ok := IsAppError(err); ok {
		return appErr.Code == code
	}
	return false
}

// Common error constructors

// ErrCartNotFound creates a cart not found error.
func ErrCartNotFound(userID string) *AppError {
	return New(CodeCartNotFound, "Cart not found").
		WithDetail("user_id", userID)
}

// ErrItemNotFound creates an item not found error.
func ErrItemNotFound(userID, itemID string) *AppError {
	return New(CodeItemNotFound, "Item not found in cart").
		WithDetails(map[string]interface{}{
			"user_id": userID,
			"item_id": itemID,
		})
}

// ErrCartLimitExceeded creates a cart limit exceeded error.
func ErrCartLimitExceeded(currentCount, maxAllowed int) *AppError {
	return New(CodeCartLimitExceeded, "Cart cannot contain more items").
		WithDetails(map[string]interface{}{
			"current_count": currentCount,
			"max_allowed":   maxAllowed,
		})
}

// ErrQuantityLimitExceeded creates a quantity limit exceeded error.
func ErrQuantityLimitExceeded(quantity, maxAllowed int) *AppError {
	return New(CodeQuantityLimit, "Quantity exceeds maximum allowed").
		WithDetails(map[string]interface{}{
			"requested_quantity": quantity,
			"max_allowed":        maxAllowed,
		})
}

// ErrInvalidQuantity creates an invalid quantity error.
func ErrInvalidQuantity(quantity int) *AppError {
	return New(CodeInvalidQuantity, "Quantity must be at least 1").
		WithDetail("quantity", quantity)
}

// ErrCartExpired creates a cart expired error.
func ErrCartExpired(userID string) *AppError {
	return New(CodeCartExpired, "Cart has expired").
		WithDetail("user_id", userID)
}

// ErrValidation creates a validation error.
func ErrValidation(message string, details map[string]interface{}) *AppError {
	return New(CodeValidationError, message).WithDetails(details)
}

// ErrConflict creates a conflict error for optimistic locking failures.
func ErrConflict(expectedVersion, currentVersion int64) *AppError {
	return New(CodeConflict, "Cart was modified by another request").
		WithDetails(map[string]interface{}{
			"expected_version": expectedVersion,
			"current_version":  currentVersion,
		})
}

// ErrRateLimited creates a rate limited error.
func ErrRateLimited() *AppError {
	return New(CodeRateLimited, "Too many requests, please try again later")
}

// ErrUnauthorized creates an unauthorized error.
func ErrUnauthorized(message string) *AppError {
	return New(CodeUnauthorized, message)
}

// ErrForbidden creates a forbidden error.
func ErrForbidden(message string) *AppError {
	return New(CodeForbidden, message)
}

// ErrInternal creates an internal error without exposing details.
func ErrInternal(cause error) *AppError {
	return Wrap(CodeInternalError, "An internal error occurred", cause)
}

// ErrServiceUnavailable creates a service unavailable error.
func ErrServiceUnavailable(service string) *AppError {
	return New(CodeServiceUnavailable, "Service temporarily unavailable").
		WithDetail("service", service)
}

// ErrPersistence creates a persistence error.
func ErrPersistence(operation string, cause error) *AppError {
	return Wrap(CodePersistenceError, fmt.Sprintf("Persistence operation failed: %s", operation), cause)
}

// ErrInventoryInsufficient creates an insufficient inventory error.
func ErrInventoryInsufficient(productID string, requested, available int) *AppError {
	return New(CodeInventoryInsufficient, "Insufficient inventory").
		WithDetails(map[string]interface{}{
			"product_id": productID,
			"requested":  requested,
			"available":  available,
		})
}
