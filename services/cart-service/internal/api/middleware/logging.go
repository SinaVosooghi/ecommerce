// Package middleware provides HTTP middleware for the cart service.
package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/logging"
)

// Logger is a middleware that logs HTTP requests.
func Logger(logger *logging.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Generate or extract request ID
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}

			// Extract trace ID if present
			traceID := r.Header.Get("X-Amzn-Trace-Id")
			if traceID == "" {
				traceID = requestID
			}

			// Add IDs to context
			ctx := r.Context()
			ctx = logging.ContextWithRequestID(ctx, requestID)
			ctx = logging.ContextWithTraceID(ctx, traceID)
			r = r.WithContext(ctx)

			// Set response headers
			w.Header().Set("X-Request-ID", requestID)

			// Wrap response writer to capture status code
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Process request
			next.ServeHTTP(ww, r)

			// Log request completion
			duration := time.Since(start)
			logger.LogRequest(ctx, r.Method, r.URL.Path, ww.Status(), duration, r.RemoteAddr)
		})
	}
}

// RequestID extracts or generates a request ID.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		ctx := logging.ContextWithRequestID(r.Context(), requestID)
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
