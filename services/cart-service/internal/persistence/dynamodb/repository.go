package dynamodb

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/core/cart"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/errors"
)

// Key prefixes for single-table design
const (
	UserKeyPrefix = "USER#"
	CartKeyPrefix = "CART#"
)

// Repository is a DynamoDB implementation of the cart repository.
type Repository struct {
	client *Client
}

// NewRepository creates a new DynamoDB repository.
func NewRepository(client *Client) *Repository {
	return &Repository{
		client: client,
	}
}

// cartRecord represents a cart stored in DynamoDB.
type cartRecord struct {
	PK        string          `dynamodbav:"PK"`
	SK        string          `dynamodbav:"SK"`
	Type      string          `dynamodbav:"type"`
	ID        string          `dynamodbav:"id"`
	UserID    string          `dynamodbav:"user_id"`
	Items     []cartItemRecord `dynamodbav:"items"`
	Version   int64           `dynamodbav:"version"`
	CreatedAt string          `dynamodbav:"created_at"`
	UpdatedAt string          `dynamodbav:"updated_at"`
	ExpiresAt string          `dynamodbav:"expires_at"`
	TTL       int64           `dynamodbav:"ttl"`
}

// cartItemRecord represents a cart item stored in DynamoDB.
type cartItemRecord struct {
	ItemID    string `dynamodbav:"item_id"`
	ProductID string `dynamodbav:"product_id"`
	Quantity  int    `dynamodbav:"quantity"`
	UnitPrice int64  `dynamodbav:"unit_price"`
	AddedAt   string `dynamodbav:"added_at"`
}

// GetCart retrieves a cart by user ID.
func (r *Repository) GetCart(ctx context.Context, userID string) (*cart.Cart, error) {
	pk := UserKeyPrefix + userID
	sk := CartKeyPrefix + userID

	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return nil, errors.Wrap(errors.CodePersistenceError, "failed to get cart", err)
	}

	if result.Item == nil {
		return nil, errors.ErrCartNotFound(userID)
	}

	var record cartRecord
	if err := attributevalue.UnmarshalMap(result.Item, &record); err != nil {
		return nil, errors.Wrap(errors.CodePersistenceError, "failed to unmarshal cart", err)
	}

	return recordToCart(&record)
}

// SaveCart saves a cart.
func (r *Repository) SaveCart(ctx context.Context, c *cart.Cart) error {
	record := cartToRecord(c)

	item, err := attributevalue.MarshalMap(record)
	if err != nil {
		return errors.Wrap(errors.CodePersistenceError, "failed to marshal cart", err)
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.client.tableName),
		Item:      item,
	})
	if err != nil {
		return errors.Wrap(errors.CodePersistenceError, "failed to save cart", err)
	}

	return nil
}

// SaveCartWithVersion saves a cart with optimistic locking.
func (r *Repository) SaveCartWithVersion(ctx context.Context, c *cart.Cart, expectedVersion int64) error {
	record := cartToRecord(c)

	item, err := attributevalue.MarshalMap(record)
	if err != nil {
		return errors.Wrap(errors.CodePersistenceError, "failed to marshal cart", err)
	}

	// Use conditional expression for optimistic locking
	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(PK) OR version = :expected_version"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":expected_version": &types.AttributeValueMemberN{Value: strconv.FormatInt(expectedVersion, 10)},
		},
	})
	if err != nil {
		// Check if it's a conditional check failed exception
		var condErr *types.ConditionalCheckFailedException
		if ok := isConditionalCheckFailedException(err, &condErr); ok {
			// Get current version for error reporting
			currentCart, getErr := r.GetCart(ctx, c.UserID)
			if getErr != nil {
				return errors.ErrConflict(expectedVersion, 0)
			}
			return errors.ErrConflict(expectedVersion, currentCart.Version)
		}
		return errors.Wrap(errors.CodePersistenceError, "failed to save cart", err)
	}

	return nil
}

// DeleteCart deletes a cart by user ID.
func (r *Repository) DeleteCart(ctx context.Context, userID string) error {
	pk := UserKeyPrefix + userID
	sk := CartKeyPrefix + userID

	_, err := r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if ok := isConditionalCheckFailedException(err, &condErr); ok {
			return errors.ErrCartNotFound(userID)
		}
		return errors.Wrap(errors.CodePersistenceError, "failed to delete cart", err)
	}

	return nil
}

// HealthCheck verifies repository connectivity.
func (r *Repository) HealthCheck(ctx context.Context) error {
	return r.client.HealthCheck(ctx)
}

// Helper functions

func cartToRecord(c *cart.Cart) *cartRecord {
	items := make([]cartItemRecord, len(c.Items))
	for i, item := range c.Items {
		items[i] = cartItemRecord{
			ItemID:    item.ItemID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			AddedAt:   item.AddedAt.Format(time.RFC3339),
		}
	}

	return &cartRecord{
		PK:        UserKeyPrefix + c.UserID,
		SK:        CartKeyPrefix + c.UserID,
		Type:      "CART",
		ID:        c.ID,
		UserID:    c.UserID,
		Items:     items,
		Version:   c.Version,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
		UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
		ExpiresAt: c.ExpiresAt.Format(time.RFC3339),
		TTL:       c.ExpiresAt.Unix(),
	}
}

func recordToCart(r *cartRecord) (*cart.Cart, error) {
	items := make([]cart.CartItem, len(r.Items))
	for i, item := range r.Items {
		addedAt, err := time.Parse(time.RFC3339, item.AddedAt)
		if err != nil {
			addedAt = time.Now().UTC()
		}
		items[i] = cart.CartItem{
			ItemID:    item.ItemID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			AddedAt:   addedAt,
		}
	}

	createdAt, err := time.Parse(time.RFC3339, r.CreatedAt)
	if err != nil {
		createdAt = time.Now().UTC()
	}

	updatedAt, err := time.Parse(time.RFC3339, r.UpdatedAt)
	if err != nil {
		updatedAt = time.Now().UTC()
	}

	expiresAt, err := time.Parse(time.RFC3339, r.ExpiresAt)
	if err != nil {
		expiresAt = time.Now().UTC().Add(7 * 24 * time.Hour)
	}

	return &cart.Cart{
		ID:        r.ID,
		UserID:    r.UserID,
		Items:     items,
		Version:   r.Version,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		ExpiresAt: expiresAt,
	}, nil
}

func isConditionalCheckFailedException(err error, target **types.ConditionalCheckFailedException) bool {
	if err == nil {
		return false
	}
	// Simple string check since errors.As might not work with AWS SDK errors
	return fmt.Sprintf("%T", err) == "*types.ConditionalCheckFailedException" ||
		contains(err.Error(), "ConditionalCheckFailed")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
