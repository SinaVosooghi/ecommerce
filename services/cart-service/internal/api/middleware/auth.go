package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

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

// UserClaims represents the claims in a JWT token.
type UserClaims struct {
	jwt.RegisteredClaims
	UserID   string   `json:"sub"`
	Email    string   `json:"email,omitempty"`
	TenantID string   `json:"tenant_id,omitempty"`
	Groups   []string `json:"cognito:groups,omitempty"`
}

// contextKey is a custom type for context keys.
type contextKey string

const (
	userContextKey contextKey = "user"
)

// JWTAuth provides JWT authentication middleware.
func JWTAuth(config AuthConfig) func(next http.Handler) http.Handler {
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for certain paths
			if skipPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeAuthError(w, "Authorization header is required")
				return
			}

			// Check Bearer prefix
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				writeAuthError(w, "Invalid authorization header format")
				return
			}

			tokenString := parts[1]

			// Parse and validate token
			claims := &UserClaims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				// Validate signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.ErrUnauthorized("Invalid signing method")
				}
				return []byte(config.JWTSecretKey), nil
			})

			if err != nil {
				writeAuthError(w, "Invalid token")
				return
			}

			if !token.Valid {
				writeAuthError(w, "Token is invalid")
				return
			}

			// Validate issuer if configured
			if config.JWTIssuer != "" {
				iss, _ := claims.GetIssuer()
				if iss != config.JWTIssuer {
					writeAuthError(w, "Invalid token issuer")
					return
				}
			}

			// Validate audience if configured
			if config.JWTAudience != "" {
				aud, _ := claims.GetAudience()
				found := false
				for _, a := range aud {
					if a == config.JWTAudience {
						found = true
						break
					}
				}
				if !found {
					writeAuthError(w, "Invalid token audience")
					return
				}
			}

			// Add user to context
			ctx := context.WithValue(r.Context(), userContextKey, claims)
			ctx = logging.ContextWithUserID(ctx, claims.UserID)
			
			// Set user ID header for downstream use
			r.Header.Set("X-User-ID", claims.UserID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalJWTAuth provides optional JWT authentication.
// It will set user context if token is present and valid, but won't reject if missing.
func OptionalJWTAuth(config AuthConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				next.ServeHTTP(w, r)
				return
			}

			tokenString := parts[1]
			claims := &UserClaims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.ErrUnauthorized("Invalid signing method")
				}
				return []byte(config.JWTSecretKey), nil
			})

			if err == nil && token.Valid {
				ctx := context.WithValue(r.Context(), userContextKey, claims)
				ctx = logging.ContextWithUserID(ctx, claims.UserID)
				r.Header.Set("X-User-ID", claims.UserID)
				r = r.WithContext(ctx)
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
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    errors.CodeUnauthorized,
		"message": message,
	})
}
