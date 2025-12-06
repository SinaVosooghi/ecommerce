package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/errors"
)

// SecurityHeaders adds security headers to all responses.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add security headers
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}

// RequestSizeLimit limits the size of request bodies.
func RequestSizeLimit(maxBytes int64) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"code":    errors.CodeInvalidRequest,
					"message": "Request body too large",
					"details": map[string]interface{}{
						"max_bytes": maxBytes,
					},
				})
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// ContentType validates the Content-Type header for requests with bodies.
func ContentType(contentTypes ...string) func(next http.Handler) http.Handler {
	allowedTypes := make(map[string]bool)
	for _, ct := range contentTypes {
		allowedTypes[strings.ToLower(ct)] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only check for methods that typically have bodies
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				ct := r.Header.Get("Content-Type")
				if ct == "" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnsupportedMediaType)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"code":    errors.CodeInvalidRequest,
						"message": "Content-Type header is required",
					})
					return
				}

				// Parse content type (ignore parameters like charset)
				mediaType := strings.ToLower(strings.Split(ct, ";")[0])
				if !allowedTypes[mediaType] {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnsupportedMediaType)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"code":    errors.CodeInvalidRequest,
						"message": "Unsupported Content-Type",
						"details": map[string]interface{}{
							"allowed": contentTypes,
						},
					})
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// NoCache adds headers to prevent caching.
func NoCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		next.ServeHTTP(w, r)
	})
}

// sanitizedBody wraps an io.ReadCloser to sanitize input.
type sanitizedBody struct {
	original io.ReadCloser
	reader   io.Reader
}

func (s *sanitizedBody) Read(p []byte) (int, error) {
	return s.reader.Read(p)
}

func (s *sanitizedBody) Close() error {
	return s.original.Close()
}
