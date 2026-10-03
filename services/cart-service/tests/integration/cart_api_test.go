// Package integration provides integration tests for the cart service. They exercise the
// production router from server.New, including auth and the rest of the middleware stack.
package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/api/v1/handlers"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/config"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/core/cart"
	"github.com/sinavosooghi/ecommerce/services/cart-service/tests/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testUser = "user-123"

func decodeCart(t *testing.T, body []byte) handlers.CartResponse {
	t.Helper()
	var response handlers.CartResponse
	require.NoError(t, json.Unmarshal(body, &response))
	return response
}

func addItem(t *testing.T, env *testutil.TestEnv, productID string, quantity int) *cart.Cart {
	t.Helper()
	c, err := env.Service.AddItem(context.Background(), testUser, cart.AddItemRequest{
		ProductID: productID,
		Quantity:  quantity,
		UnitPrice: 1999,
	})
	require.NoError(t, err)
	return c
}

func TestCartAPI_AddItem(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token := testutil.Token(t, testUser)

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name: "add valid item",
			body: map[string]interface{}{
				"product_id": "product-1",
				"quantity":   2,
				"unit_price": 1999,
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "add item with invalid quantity",
			body: map[string]interface{}{
				"product_id": "product-1",
				"quantity":   0,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "add item with missing product_id",
			body: map[string]interface{}{
				"quantity": 1,
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := env.Do(t, testutil.Request{
				Method: http.MethodPost,
				Path:   "/v1/cart/" + testUser + "/items",
				Token:  token,
				Body:   tt.body,
			})
			assert.Equal(t, tt.wantStatus, w.Code, w.Body.String())
		})
	}
}

func TestCartAPI_GetCart(t *testing.T) {
	env := testutil.NewTestEnv(t)
	addItem(t, env, "product-1", 2)

	w := env.Do(t, testutil.Request{Method: http.MethodGet, Path: "/v1/cart/" + testUser, Token: testutil.Token(t, testUser)})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	response := decodeCart(t, w.Body.Bytes())
	assert.Equal(t, testUser, response.UserID)
	require.Len(t, response.Items, 1)
	assert.Equal(t, "product-1", response.Items[0].ProductID)
	assert.Equal(t, 2, response.Items[0].Quantity)
}

func TestCartAPI_UpdateItem(t *testing.T) {
	env := testutil.NewTestEnv(t)
	c := addItem(t, env, "product-1", 2)

	w := env.Do(t, testutil.Request{
		Method: http.MethodPatch,
		Path:   "/v1/cart/" + testUser + "/items/" + c.Items[0].ItemID,
		Token:  testutil.Token(t, testUser),
		Body:   map[string]interface{}{"quantity": 5, "version": c.Version},
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	response := decodeCart(t, w.Body.Bytes())
	assert.Equal(t, 5, response.Items[0].Quantity)
	assert.Equal(t, c.Version+1, response.Version)
}

func TestCartAPI_UpdateItem_StaleVersionConflicts(t *testing.T) {
	env := testutil.NewTestEnv(t)
	c := addItem(t, env, "product-1", 2)
	addItem(t, env, "product-2", 1) // bumps the version past what the client saw

	w := env.Do(t, testutil.Request{
		Method: http.MethodPatch,
		Path:   "/v1/cart/" + testUser + "/items/" + c.Items[0].ItemID,
		Token:  testutil.Token(t, testUser),
		Body:   map[string]interface{}{"quantity": 5, "version": c.Version},
	})
	assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
}

func TestCartAPI_RemoveItem(t *testing.T) {
	env := testutil.NewTestEnv(t)
	c := addItem(t, env, "product-1", 2)

	w := env.Do(t, testutil.Request{
		Method: http.MethodDelete,
		Path:   "/v1/cart/" + testUser + "/items/" + c.Items[0].ItemID,
		Token:  testutil.Token(t, testUser),
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Empty(t, decodeCart(t, w.Body.Bytes()).Items)
}

func TestCartAPI_ClearCart(t *testing.T) {
	env := testutil.NewTestEnv(t)
	addItem(t, env, "product-1", 2)
	addItem(t, env, "product-2", 1)

	w := env.Do(t, testutil.Request{Method: http.MethodDelete, Path: "/v1/cart/" + testUser, Token: testutil.Token(t, testUser)})
	assert.Equal(t, http.StatusNoContent, w.Code)

	c, err := env.Service.GetCart(context.Background(), testUser)
	require.NoError(t, err)
	assert.Empty(t, c.Items)
}

func TestCartAPI_MergeCart(t *testing.T) {
	env := testutil.NewTestEnv(t)
	ctx := context.Background()
	_, err := env.Service.AddItem(ctx, "guest-1", cart.AddItemRequest{ProductID: "product-9", Quantity: 1, UnitPrice: 500})
	require.NoError(t, err)

	w := env.Do(t, testutil.Request{
		Method: http.MethodPost,
		Path:   "/v1/cart/" + testUser + "/merge",
		Token:  testutil.Token(t, testUser),
		Body:   map[string]interface{}{"guest_id": "guest-1"},
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Len(t, decodeCart(t, w.Body.Bytes()).Items, 1)

	_, err = env.Service.GetCart(ctx, "guest-1")
	assert.Error(t, err, "guest cart should be deleted after merge")

	t.Run("guest id equal to user is rejected", func(t *testing.T) {
		w := env.Do(t, testutil.Request{
			Method: http.MethodPost,
			Path:   "/v1/cart/" + testUser + "/merge",
			Token:  testutil.Token(t, testUser),
			Body:   map[string]interface{}{"guest_id": testUser},
		})
		assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	})
}

func TestCartAPI_NotFound(t *testing.T) {
	env := testutil.NewTestEnv(t)

	w := env.Do(t, testutil.Request{Method: http.MethodGet, Path: "/v1/cart/nonexistent-user", Token: testutil.Token(t, "nonexistent-user")})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCartAPI_InvalidUserID(t *testing.T) {
	env := testutil.NewTestEnv(t)

	// URL-safe characters that are not valid in a user ID
	userID := "invalid$$user$$id"
	w := env.Do(t, testutil.Request{Method: http.MethodGet, Path: "/v1/cart/" + userID, Token: testutil.Token(t, userID)})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuth(t *testing.T) {
	env := testutil.NewTestEnv(t)
	path := "/v1/cart/" + testUser

	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "missing token", token: "", wantStatus: http.StatusUnauthorized},
		{name: "garbage token", token: "not-a-jwt", wantStatus: http.StatusUnauthorized},
		{
			name: "wrong key",
			token: testutil.SignToken(t, jwt.SigningMethodHS256, []byte("some-other-secret-0123456789abcdef"), jwt.RegisteredClaims{
				Subject: testUser, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			}),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "wrong algorithm",
			token: testutil.SignToken(t, jwt.SigningMethodHS512, []byte(testutil.TestJWTSecret), jwt.RegisteredClaims{
				Subject: testUser, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			}),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "expired",
			token: testutil.SignToken(t, jwt.SigningMethodHS256, []byte(testutil.TestJWTSecret), jwt.RegisteredClaims{
				Subject: testUser, ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			}),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "no expiry",
			token: testutil.SignToken(t, jwt.SigningMethodHS256, []byte(testutil.TestJWTSecret), jwt.RegisteredClaims{
				Subject: testUser,
			}),
			wantStatus: http.StatusUnauthorized,
		},
		{name: "another user's cart", token: testutil.Token(t, "someone-else"), wantStatus: http.StatusForbidden},
		{name: "own cart", token: testutil.Token(t, testUser), wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := env.Do(t, testutil.Request{Method: http.MethodGet, Path: path, Token: tt.token})
			assert.Equal(t, tt.wantStatus, w.Code, w.Body.String())
		})
	}
}

func TestAuth_DisabledInDev(t *testing.T) {
	env := testutil.NewTestEnv(t, func(c *config.Config) { c.AuthEnabled = false })

	w := env.Do(t, testutil.Request{Method: http.MethodGet, Path: "/v1/cart/" + testUser})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestConcurrentAddItem_NoLostUpdates(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token := testutil.Token(t, testUser)

	const workers = 20
	statuses := make([]int, workers)
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := env.Do(t, testutil.Request{
				Method: http.MethodPost,
				Path:   "/v1/cart/" + testUser + "/items",
				Token:  token,
				Body:   map[string]interface{}{"product_id": fmt.Sprintf("product-%d", i), "quantity": 1, "unit_price": 100},
			})
			statuses[i] = w.Code
		}()
	}
	wg.Wait()

	succeeded := 0
	for _, status := range statuses {
		switch status {
		case http.StatusCreated:
			succeeded++
		case http.StatusConflict:
			// Acceptable under heavy contention: the client is told to retry.
		default:
			t.Fatalf("unexpected status %d", status)
		}
	}

	c, err := env.Service.GetCart(context.Background(), testUser)
	require.NoError(t, err)
	assert.Positive(t, succeeded)
	assert.Len(t, c.Items, succeeded, "every successful add must be persisted")
}

func TestIdempotency(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token := testutil.Token(t, testUser)
	request := testutil.Request{
		Method:  http.MethodPost,
		Path:    "/v1/cart/" + testUser + "/items",
		Token:   token,
		Body:    map[string]interface{}{"product_id": "product-1", "quantity": 2, "unit_price": 100},
		Headers: map[string]string{"Idempotency-Key": "key-1"},
	}

	first := env.Do(t, request)
	require.Equal(t, http.StatusCreated, first.Code, first.Body.String())

	second := env.Do(t, request)
	require.Equal(t, http.StatusCreated, second.Code)
	assert.Equal(t, "true", second.Header().Get("X-Idempotent-Replayed"))
	assert.JSONEq(t, first.Body.String(), second.Body.String())

	c, err := env.Service.GetCart(context.Background(), testUser)
	require.NoError(t, err)
	assert.Equal(t, 2, c.Items[0].Quantity, "replay must not apply the change twice")

	t.Run("same key with different body is rejected", func(t *testing.T) {
		changed := request
		changed.Body = map[string]interface{}{"product_id": "product-2", "quantity": 1, "unit_price": 100}
		w := env.Do(t, changed)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
	})

	t.Run("keys are scoped per user", func(t *testing.T) {
		other := request
		other.Path = "/v1/cart/user-456/items"
		other.Token = testutil.Token(t, "user-456")
		w := env.Do(t, other)
		assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		assert.Empty(t, w.Header().Get("X-Idempotent-Replayed"))
	})
}

func TestRateLimit(t *testing.T) {
	env := testutil.NewTestEnv(t, func(c *config.Config) {
		c.RateLimitRPS = 1
		c.RateLimitBurst = 2
	})
	token := testutil.Token(t, testUser)

	var codes []int
	for range 3 {
		w := env.Do(t, testutil.Request{Method: http.MethodGet, Path: "/v1/cart/" + testUser, Token: token})
		codes = append(codes, w.Code)
	}
	assert.Equal(t, []int{http.StatusNotFound, http.StatusNotFound, http.StatusTooManyRequests}, codes)
}

func TestHealthAndReadiness(t *testing.T) {
	env := testutil.NewTestEnv(t)

	assert.Equal(t, http.StatusOK, env.Do(t, testutil.Request{Method: http.MethodGet, Path: "/health"}).Code)
	assert.Equal(t, http.StatusOK, env.Do(t, testutil.Request{Method: http.MethodGet, Path: "/ready"}).Code)

	env.Repo.SetHealthError(errors.New("connection refused"))
	assert.Equal(t, http.StatusOK, env.Do(t, testutil.Request{Method: http.MethodGet, Path: "/health"}).Code,
		"liveness must not depend on downstream dependencies")
	assert.Equal(t, http.StatusServiceUnavailable, env.Do(t, testutil.Request{Method: http.MethodGet, Path: "/ready"}).Code)
}

func TestCORS_NoCredentialsWithWildcard(t *testing.T) {
	env := testutil.NewTestEnv(t)

	w := env.Do(t, testutil.Request{
		Method: http.MethodOptions,
		Path:   "/v1/cart/" + testUser,
		Headers: map[string]string{
			"Origin":                        "https://evil.example",
			"Access-Control-Request-Method": http.MethodGet,
		},
	})
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
}
