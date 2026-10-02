package dynamodb

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/core/cart"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCartRecordRoundTrip_PreservesSubSecondTimestamps(t *testing.T) {
	c := cart.NewCart("user-1")
	require.NoError(t, c.AddItem(cart.NewCartItem("product-1", 2, 500)))

	got, err := recordToCart(cartToRecord(c))
	require.NoError(t, err)

	assert.True(t, c.UpdatedAt.Equal(got.UpdatedAt), "UpdatedAt lost precision")
	assert.True(t, c.Items[0].AddedAt.Equal(got.Items[0].AddedAt), "AddedAt lost precision")
	assert.Equal(t, c.Version, got.Version)
}

func TestRecordToCart_ReadsLegacyRFC3339(t *testing.T) {
	legacy := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC).Format(time.RFC3339)
	got, err := recordToCart(&cartRecord{
		UserID: "user-1", CreatedAt: legacy, UpdatedAt: legacy, ExpiresAt: legacy,
		Items: []cartItemRecord{{ItemID: "i", ProductID: "p", Quantity: 1, AddedAt: legacy}},
	})
	require.NoError(t, err)
	assert.Equal(t, 2025, got.CreatedAt.Year())
}

func TestRecordToCart_RejectsCorruptTimestamps(t *testing.T) {
	_, err := recordToCart(&cartRecord{CreatedAt: "garbage"})
	assert.Error(t, err)
}

func TestStoredVersion(t *testing.T) {
	assert.Equal(t, int64(7), storedVersion(map[string]types.AttributeValue{
		"version": &types.AttributeValueMemberN{Value: "7"},
	}))
	assert.Equal(t, int64(0), storedVersion(nil))
}
