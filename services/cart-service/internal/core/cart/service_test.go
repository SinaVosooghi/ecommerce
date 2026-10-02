package cart

import (
	"context"
	"testing"

	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memRepo is a minimal Repository with optimistic locking and injectable save conflicts.
type memRepo struct {
	carts           map[string]Cart
	conflictsToFake int
}

func newMemRepo() *memRepo { return &memRepo{carts: map[string]Cart{}} }

func (r *memRepo) GetCart(_ context.Context, userID string) (*Cart, error) {
	c, ok := r.carts[userID]
	if !ok {
		return nil, errors.ErrCartNotFound(userID)
	}
	c.Items = append([]CartItem(nil), c.Items...)
	return &c, nil
}

func (r *memRepo) SaveCart(_ context.Context, c *Cart) error {
	r.carts[c.UserID] = *c
	return nil
}

func (r *memRepo) SaveCartWithVersion(_ context.Context, c *Cart, expected int64) error {
	if r.conflictsToFake > 0 {
		r.conflictsToFake--
		return errors.ErrConflict(expected, expected+1)
	}
	current, ok := r.carts[c.UserID]
	if (ok && current.Version != expected) || (!ok && expected != 0) {
		return errors.ErrConflict(expected, current.Version)
	}
	r.carts[c.UserID] = *c
	return nil
}

func (r *memRepo) DeleteCart(_ context.Context, userID string) error {
	delete(r.carts, userID)
	return nil
}

type fixedPrices map[string]int64

func (p fixedPrices) ValidatePrice(_ context.Context, productID string, price int64) (bool, error) {
	return p[productID] == price, nil
}

func (p fixedPrices) GetCurrentPrice(_ context.Context, productID string) (int64, error) {
	return p[productID], nil
}

func TestService_AddItem_CreatesCartWithVersionOne(t *testing.T) {
	svc := NewService(newMemRepo(), nil, ServiceConfig{})

	c, err := svc.AddItem(context.Background(), "user-1", AddItemRequest{ProductID: "p1", Quantity: 1, UnitPrice: 100})
	require.NoError(t, err)
	assert.Equal(t, int64(1), c.Version)
}

func TestService_RetriesOnConflict(t *testing.T) {
	repo := newMemRepo()
	repo.conflictsToFake = maxSaveAttempts - 1
	svc := NewService(repo, nil, ServiceConfig{})

	_, err := svc.AddItem(context.Background(), "user-1", AddItemRequest{ProductID: "p1", Quantity: 1, UnitPrice: 100})
	require.NoError(t, err)
}

func TestService_GivesUpAfterMaxAttempts(t *testing.T) {
	repo := newMemRepo()
	repo.conflictsToFake = maxSaveAttempts
	svc := NewService(repo, nil, ServiceConfig{})

	_, err := svc.AddItem(context.Background(), "user-1", AddItemRequest{ProductID: "p1", Quantity: 1, UnitPrice: 100})
	assert.True(t, errors.IsCode(err, errors.CodeConflict))
}

func TestService_UpdateItemQuantity_StaleClientVersion(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newMemRepo(), nil, ServiceConfig{})
	c, err := svc.AddItem(ctx, "user-1", AddItemRequest{ProductID: "p1", Quantity: 1, UnitPrice: 100})
	require.NoError(t, err)

	_, err = svc.UpdateItemQuantity(ctx, "user-1", UpdateItemRequest{
		ItemID: c.Items[0].ItemID, Quantity: 3, ExpectedVersion: c.Version + 5,
	})
	assert.True(t, errors.IsCode(err, errors.CodeConflict))
}

func TestService_AddItem_UsesCatalogPriceWhenConfigured(t *testing.T) {
	svc := NewService(newMemRepo(), nil, ServiceConfig{Prices: fixedPrices{"p1": 4200}})

	c, err := svc.AddItem(context.Background(), "user-1", AddItemRequest{ProductID: "p1", Quantity: 1, UnitPrice: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(4200), c.Items[0].UnitPrice, "client price must be ignored")
}

func TestService_MergeGuestCart_RejectsSelfMerge(t *testing.T) {
	svc := NewService(newMemRepo(), nil, ServiceConfig{})

	_, err := svc.MergeGuestCart(context.Background(), "user-1", "user-1")
	assert.True(t, errors.IsCode(err, errors.CodeValidationError))
}
