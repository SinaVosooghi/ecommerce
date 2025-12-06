// Package errors provides standardized error handling for the cart service.
package errors

// Error codes for cart service operations.
const (
	// Client errors (4xx)
	CodeCartNotFound        = "CART_NOT_FOUND"
	CodeItemNotFound        = "ITEM_NOT_FOUND"
	CodeCartLimitExceeded   = "CART_LIMIT_EXCEEDED"
	CodeQuantityLimit       = "QUANTITY_LIMIT_EXCEEDED"
	CodeInvalidQuantity     = "INVALID_QUANTITY"
	CodeCartExpired         = "CART_EXPIRED"
	CodeValidationError     = "VALIDATION_ERROR"
	CodeConflict            = "CONFLICT"
	CodeRateLimited         = "RATE_LIMITED"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeForbidden           = "FORBIDDEN"
	CodeInvalidRequest      = "INVALID_REQUEST"
	CodeIdempotencyConflict = "IDEMPOTENCY_CONFLICT"

	// Server errors (5xx)
	CodeInternalError       = "INTERNAL_ERROR"
	CodeServiceUnavailable  = "SERVICE_UNAVAILABLE"
	CodePersistenceError    = "PERSISTENCE_ERROR"
	CodeEventPublishError   = "EVENT_PUBLISH_ERROR"
	CodeInventoryError      = "INVENTORY_ERROR"
	CodeInventoryInsufficient = "INVENTORY_INSUFFICIENT"
)

// HTTP status codes mapped to error codes.
var httpStatusCodes = map[string]int{
	CodeCartNotFound:          404,
	CodeItemNotFound:          404,
	CodeCartLimitExceeded:     400,
	CodeQuantityLimit:         400,
	CodeInvalidQuantity:       400,
	CodeCartExpired:           410,
	CodeValidationError:       400,
	CodeConflict:              409,
	CodeRateLimited:           429,
	CodeUnauthorized:          401,
	CodeForbidden:             403,
	CodeInvalidRequest:        400,
	CodeIdempotencyConflict:   409,
	CodeInternalError:         500,
	CodeServiceUnavailable:    503,
	CodePersistenceError:      500,
	CodeEventPublishError:     500,
	CodeInventoryError:        500,
	CodeInventoryInsufficient: 409,
}

// HTTPStatusForCode returns the HTTP status code for a given error code.
func HTTPStatusForCode(code string) int {
	if status, ok := httpStatusCodes[code]; ok {
		return status
	}
	return 500
}
