package middleware

import (
	"encoding/json"
	"net/http"
	"runtime/debug"

	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/errors"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/logging"
)

// Recovery is a middleware that recovers from panics.
func Recovery(logger *logging.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					// Log the panic with stack trace
					logger.WithContext(r.Context()).
						WithField("panic", rec).
						WithField("stack", string(debug.Stack())).
						Error("Panic recovered")

					// Return internal error response
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"code":    errors.CodeInternalError,
						"message": "An internal error occurred",
					})
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
