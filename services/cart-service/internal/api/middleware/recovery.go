package middleware

import (
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
				rec := recover()
				if rec == nil {
					return
				}
				// http.ErrAbortHandler is used to abort a response deliberately; let net/http handle it.
				if rec == http.ErrAbortHandler {
					panic(rec)
				}

				// Log the panic with stack trace
				logger.WithContext(r.Context()).
					WithField("panic", rec).
					WithField("stack", string(debug.Stack())).
					Error("Panic recovered")

				// Return internal error response
				writeJSONError(w, http.StatusInternalServerError, errors.CodeInternalError, "An internal error occurred")
			}()

			next.ServeHTTP(w, r)
		})
	}
}
