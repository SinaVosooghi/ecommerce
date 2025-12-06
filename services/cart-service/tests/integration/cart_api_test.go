// Package integration provides integration tests for the cart service.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/api/v1/handlers"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/core/cart"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/logging"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/persistence/inmemory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRouter() (*chi.Mux, *cart.Service) {
	repo := inmemory.NewRepository()
	logger := logging.New(logging.Config{
		Level:       "debug",
		ServiceName: "cart-service-test",
		Environment: "test",
	})

	service := cart.NewService(repo, nil, cart.ServiceConfig{
		PublishEvents: false,
	})

	handler := handlers.NewCartHandler(service, logger)

	r := chi.NewRouter()
	r.Route("/v1/cart/{userID}", func(r chi.Router) {
		r.Get("/", handler.GetCart)
		r.Delete("/", handler.ClearCart)
		r.Post("/items", handler.AddItem)
		r.Patch("/items/{itemID}", handler.UpdateItem)
		r.Delete("/items/{itemID}", handler.RemoveItem)
	})

	return r, service
}

func TestCartAPI_AddItem(t *testing.T) {
	router, _ := setupTestRouter()

	tests := []struct {
		name       string
		userID     string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name:   "add valid item",
			userID: "user-123",
			body: map[string]interface{}{
				"product_id": "product-1",
				"quantity":   2,
				"unit_price": 1999,
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:   "add item with invalid quantity",
			userID: "user-123",
			body: map[string]interface{}{
				"product_id": "product-1",
				"quantity":   0,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "add item with missing product_id",
			userID: "user-123",
			body: map[string]interface{}{
				"quantity": 1,
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/v1/cart/"+tt.userID+"/items", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestCartAPI_GetCart(t *testing.T) {
	router, service := setupTestRouter()
	ctx := context.Background()

	// Add an item first
	_, err := service.AddItem(ctx, "user-123", cart.AddItemRequest{
		ProductID: "product-1",
		Quantity:  2,
		UnitPrice: 1999,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/v1/cart/user-123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response handlers.CartResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "user-123", response.UserID)
	assert.Len(t, response.Items, 1)
	assert.Equal(t, "product-1", response.Items[0].ProductID)
	assert.Equal(t, 2, response.Items[0].Quantity)
}

func TestCartAPI_UpdateItem(t *testing.T) {
	router, service := setupTestRouter()
	ctx := context.Background()

	// Add an item first
	c, err := service.AddItem(ctx, "user-123", cart.AddItemRequest{
		ProductID: "product-1",
		Quantity:  2,
		UnitPrice: 1999,
	})
	require.NoError(t, err)

	itemID := c.Items[0].ItemID

	// Update quantity
	body, _ := json.Marshal(map[string]interface{}{
		"quantity": 5,
	})
	req := httptest.NewRequest(http.MethodPatch, "/v1/cart/user-123/items/"+itemID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response handlers.CartResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 5, response.Items[0].Quantity)
}

func TestCartAPI_RemoveItem(t *testing.T) {
	router, service := setupTestRouter()
	ctx := context.Background()

	// Add an item first
	c, err := service.AddItem(ctx, "user-123", cart.AddItemRequest{
		ProductID: "product-1",
		Quantity:  2,
		UnitPrice: 1999,
	})
	require.NoError(t, err)

	itemID := c.Items[0].ItemID

	// Remove item
	req := httptest.NewRequest(http.MethodDelete, "/v1/cart/user-123/items/"+itemID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response handlers.CartResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response.Items, 0)
}

func TestCartAPI_ClearCart(t *testing.T) {
	router, service := setupTestRouter()
	ctx := context.Background()

	// Add items first
	_, err := service.AddItem(ctx, "user-123", cart.AddItemRequest{
		ProductID: "product-1",
		Quantity:  2,
		UnitPrice: 1999,
	})
	require.NoError(t, err)

	_, err = service.AddItem(ctx, "user-123", cart.AddItemRequest{
		ProductID: "product-2",
		Quantity:  1,
		UnitPrice: 999,
	})
	require.NoError(t, err)

	// Clear cart
	req := httptest.NewRequest(http.MethodDelete, "/v1/cart/user-123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify cart is empty
	c, err := service.GetCart(ctx, "user-123")
	require.NoError(t, err)
	assert.Len(t, c.Items, 0)
}

func TestCartAPI_NotFound(t *testing.T) {
	router, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/cart/nonexistent-user", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCartAPI_InvalidUserID(t *testing.T) {
	router, _ := setupTestRouter()

	// Test with invalid user ID format (contains special chars that are URL-safe but invalid for user ID)
	req := httptest.NewRequest(http.MethodGet, "/v1/cart/invalid$$user$$id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
