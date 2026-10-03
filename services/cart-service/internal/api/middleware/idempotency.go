package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/errors"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/idempotency"
)

const (
	// maxIdempotencyKeyLength bounds the client-supplied key.
	maxIdempotencyKeyLength = 255
	// idempotencyLockTTL bounds how long an in-flight reservation blocks retries if the
	// instance processing it dies before completing.
	idempotencyLockTTL = time.Minute
)

// IdempotencyConfig holds configuration for idempotency middleware.
type IdempotencyConfig struct {
	Enabled bool
	TTL     time.Duration
	Store   idempotency.Store
}

// Idempotency deduplicates POST and PATCH requests that carry an Idempotency-Key header.
// Keys are scoped to the authenticated user. A repeated key replays the stored response;
// a key reused with a different request returns 422, and a key whose original request is
// still running returns 409. Only 2xx responses are stored, so failed requests can be retried.
func Idempotency(config IdempotencyConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only apply to methods that modify state
			if r.Method != http.MethodPost && r.Method != http.MethodPatch {
				next.ServeHTTP(w, r)
				return
			}

			if !config.Enabled || config.Store == nil {
				next.ServeHTTP(w, r)
				return
			}

			idempotencyKey := r.Header.Get("Idempotency-Key")
			if idempotencyKey == "" {
				next.ServeHTTP(w, r)
				return
			}
			if len(idempotencyKey) > maxIdempotencyKeyLength {
				writeJSONError(w, http.StatusBadRequest, errors.CodeInvalidRequest, "Idempotency-Key is too long")
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				var tooLarge *http.MaxBytesError
				if stderrors.As(err, &tooLarge) {
					writeJSONError(w, http.StatusRequestEntityTooLarge, errors.CodeInvalidRequest, "Request body too large")
					return
				}
				writeJSONError(w, http.StatusBadRequest, errors.CodeInvalidRequest, "Failed to read request body")
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			scopedKey := scopeKey(r, idempotencyKey)
			fingerprint := requestFingerprint(r, body)

			existing, err := config.Store.Reserve(r.Context(), scopedKey, fingerprint, idempotencyLockTTL)
			if err != nil {
				writeJSONError(w, http.StatusServiceUnavailable, errors.CodeServiceUnavailable, "Idempotency store unavailable")
				return
			}
			if existing != nil {
				replay(w, existing, fingerprint)
				return
			}

			rw := &responseCapture{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				body:           &bytes.Buffer{},
			}
			next.ServeHTTP(rw, r)

			// Finish bookkeeping even if the client disconnected.
			ctx := context.WithoutCancel(r.Context())
			if rw.statusCode >= 200 && rw.statusCode < 300 {
				_ = config.Store.Complete(ctx, scopedKey, &idempotency.Record{
					Fingerprint: fingerprint,
					StatusCode:  rw.statusCode,
					ContentType: rw.Header().Get("Content-Type"),
					Body:        rw.body.Bytes(),
				}, config.TTL)
			} else {
				_ = config.Store.Release(ctx, scopedKey)
			}
		})
	}
}

func replay(w http.ResponseWriter, record *idempotency.Record, fingerprint string) {
	switch {
	case record.Fingerprint != fingerprint:
		writeJSONError(w, http.StatusUnprocessableEntity, errors.CodeIdempotencyConflict,
			"Idempotency-Key was already used for a different request")
	case !record.Completed:
		writeJSONError(w, http.StatusConflict, errors.CodeIdempotencyConflict,
			"A request with this Idempotency-Key is still being processed")
	default:
		if record.ContentType != "" {
			w.Header().Set("Content-Type", record.ContentType)
		}
		w.Header().Set("X-Idempotent-Replayed", "true")
		w.WriteHeader(record.StatusCode)
		_, _ = w.Write(record.Body)
	}
}

// scopeKey namespaces the client key by the authenticated user. The length prefix keeps
// the encoding unambiguous for any user or key contents.
func scopeKey(r *http.Request, key string) string {
	user := "anonymous"
	if claims := GetUserFromContext(r.Context()); claims != nil {
		user = claims.Subject
	}
	return fmt.Sprintf("%d:%s:%s", len(user), user, key)
}

// requestFingerprint identifies a request by method, path and body.
func requestFingerprint(r *http.Request, body []byte) string {
	h := sha256.New()
	h.Write([]byte(r.Method + " " + r.URL.Path + "\n"))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

// responseCapture captures the response for idempotency storage.
type responseCapture struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (r *responseCapture) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseCapture) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// IdempotencyKeyRequired is middleware that requires an idempotency key for certain methods.
func IdempotencyKeyRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPatch {
			if r.Header.Get("Idempotency-Key") == "" {
				writeJSONError(w, http.StatusBadRequest, errors.CodeInvalidRequest,
					"Idempotency-Key header is required for this request")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
