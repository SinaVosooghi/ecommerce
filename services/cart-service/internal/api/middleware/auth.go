package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/errors"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/logging"
)

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	JWTSecretKey string
	JWTIssuer    string
	JWTAudience  string
	SkipPaths    []string // Paths to skip authentication
}

// UserClaims represents the claims in a JWT token. The user ID is the standard "sub" claim.
type UserClaims struct {
	jwt.RegisteredClaims
	Email    string   `json:"email,omitempty"`
	TenantID string   `json:"tenant_id,omitempty"`
	Groups   []string `json:"cognito:groups,omitempty"`
}

// contextKey is a custom type for context keys.
type contextKey string

const (
	userContextKey contextKey = "user"
)

// newTokenParser returns a parser that only accepts HS256 tokens with an expiry and,
// when configured, the expected issuer and audience.
func newTokenParser(config AuthConfig) *jwt.Parser {
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	}
	if config.JWTIssuer != "" {
		opts = append(opts, jwt.WithIssuer(config.JWTIssuer))
	}
	if config.JWTAudience != "" {
		opts = append(opts, jwt.WithAudience(config.JWTAudience))
	}
	return jwt.NewParser(opts...)
}

// parseBearer validates the bearer token in the Authorization header and returns its claims.
func parseBearer(parser *jwt.Parser, key []byte, r *http.Request) (*UserClaims, string) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, "Authorization header is required"
	}

	scheme, tokenString, ok := strings.Cut(authHeader, " ")
	if !ok || !strings.EqualFold(scheme, "bearer") || tokenString == "" {
		return nil, "Invalid authorization header format"
	}

	claims := &UserClaims{}
	_, err := parser.ParseWithClaims(tokenString, claims, func(*jwt.Token) (interface{}, error) {
		return key, nil
	})
	if err != nil {
		return nil, "Invalid token"
	}
	if claims.Subject == "" {
		return nil, "Token has no subject"
	}
	return claims, ""
}

func withUser(r *http.Request, claims *UserClaims) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, claims)
	ctx = logging.ContextWithUserID(ctx, claims.Subject)
	return r.WithContext(ctx)
}

// JWTAuth provides JWT authentication middleware.
func JWTAuth(config AuthConfig) func(next http.Handler) http.Handler {
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}
	parser := newTokenParser(config)
	key := []byte(config.JWTSecretKey)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for certain paths
			if skipPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			claims, msg := parseBearer(parser, key, r)
			if claims == nil {
				writeAuthError(w, msg)
				return
			}

			next.ServeHTTP(w, withUser(r, claims))
		})
	}
}

// OptionalJWTAuth provides optional JWT authentication.
// It will set user context if token is present and valid, but won't reject if missing.
func OptionalJWTAuth(config AuthConfig) func(next http.Handler) http.Handler {
	parser := newTokenParser(config)
	key := []byte(config.JWTSecretKey)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if claims, _ := parseBearer(parser, key, r); claims != nil {
				r = withUser(r, claims)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireOwner rejects requests whose URL parameter (e.g. {userID}) does not match the
// authenticated subject, so callers can only access their own resources.
// It must run after JWTAuth.
func RequireOwner(param string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUserFromContext(r.Context())
			if claims == nil || claims.Subject != chi.URLParam(r, param) {
				writeJSONError(w, http.StatusForbidden, errors.CodeForbidden, "Access to this resource is not allowed")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// APIKeyAuth provides API key authentication for service-to-service calls.
func APIKeyAuth(validKeys map[string]string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				writeAuthError(w, "API key is required")
				return
			}

			serviceName, valid := validKeys[apiKey]
			if !valid {
				writeAuthError(w, "Invalid API key")
				return
			}

			// Set service name in header
			r.Header.Set("X-Service-Name", serviceName)
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserFromContext retrieves user claims from the context.
func GetUserFromContext(ctx context.Context) *UserClaims {
	if claims, ok := ctx.Value(userContextKey).(*UserClaims); ok {
		return claims
	}
	return nil
}

func writeAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	writeJSONError(w, http.StatusUnauthorized, errors.CodeUnauthorized, message)
}

// writeJSONError writes a standard error body. Write errors are ignored because the
// client has already gone away if the response cannot be written.
func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    code,
		"message": message,
	})
}
