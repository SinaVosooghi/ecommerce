package middleware

import (
	stderrors "errors"
	"net/http"
	"runtime/debug"

	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/errors"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/logging"
)

// Recovery is a middleware that recovers from panics.
func Recovery(logger *logging.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer recoverPanic(w, r, logger)
			next.ServeHTTP(w, r)
		})
	}
}

// recoverPanic must be called directly by defer for recover to take effect.
func recoverPanic(w http.ResponseWriter, r *http.Request, logger *logging.Logger) {
	rec := recover()
	if rec == nil {
		return
	}
	// http.ErrAbortHandler aborts a response deliberately; let net/http handle it.
	if err, ok := rec.(error); ok && stderrors.Is(err, http.ErrAbortHandler) {
		panic(rec)
	}

	// Log the panic with stack trace
	logger.WithContext(r.Context()).
		WithField("panic", rec).
		WithField("stack", string(debug.Stack())).
		Error("Panic recovered")

	// Return internal error response
	writeJSONError(w, http.StatusInternalServerError, errors.CodeInternalError, "An internal error occurred")
}
