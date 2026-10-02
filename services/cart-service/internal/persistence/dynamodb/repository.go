package dynamodb

import (
	"context"
	stderrors "errors"
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
	PK        string           `dynamodbav:"PK"`
	SK        string           `dynamodbav:"SK"`
	Type      string           `dynamodbav:"type"`
	ID        string           `dynamodbav:"id"`
	UserID    string           `dynamodbav:"user_id"`
	Items     []cartItemRecord `dynamodbav:"items"`
	Version   int64            `dynamodbav:"version"`
	CreatedAt string           `dynamodbav:"created_at"`
	UpdatedAt string           `dynamodbav:"updated_at"`
	ExpiresAt string           `dynamodbav:"expires_at"`
	TTL       int64            `dynamodbav:"ttl"`
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

// SaveCartWithVersion saves a cart with optimistic locking. An expectedVersion of 0 means
// the cart must not exist yet; otherwise the stored version must equal expectedVersion.
func (r *Repository) SaveCartWithVersion(ctx context.Context, c *cart.Cart, expectedVersion int64) error {
	record := cartToRecord(c)

	item, err := attributevalue.MarshalMap(record)
	if err != nil {
		return errors.Wrap(errors.CodePersistenceError, "failed to marshal cart", err)
	}

	input := &dynamodb.PutItemInput{
		TableName:                           aws.String(r.client.tableName),
		Item:                                item,
		ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailureAllOld,
	}
	if expectedVersion == 0 {
		input.ConditionExpression = aws.String("attribute_not_exists(PK)")
	} else {
		input.ConditionExpression = aws.String("version = :expected_version")
		input.ExpressionAttributeValues = map[string]types.AttributeValue{
			":expected_version": &types.AttributeValueMemberN{Value: strconv.FormatInt(expectedVersion, 10)},
		}
	}

	if _, err = r.client.db.PutItem(ctx, input); err != nil {
		var condErr *types.ConditionalCheckFailedException
		if stderrors.As(err, &condErr) {
			return errors.ErrConflict(expectedVersion, storedVersion(condErr.Item))
		}
		return errors.Wrap(errors.CodePersistenceError, "failed to save cart", err)
	}

	return nil
}

// storedVersion extracts the version from the item returned with a failed condition check.
// It returns 0 when the item does not exist.
func storedVersion(item map[string]types.AttributeValue) int64 {
	if v, ok := item["version"].(*types.AttributeValueMemberN); ok {
		if n, err := strconv.ParseInt(v.Value, 10, 64); err == nil {
			return n
		}
	}
	return 0
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
		if stderrors.As(err, &condErr) {
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
			AddedAt:   item.AddedAt.Format(time.RFC3339Nano),
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
		CreatedAt: c.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt: c.UpdatedAt.Format(time.RFC3339Nano),
		ExpiresAt: c.ExpiresAt.Format(time.RFC3339Nano),
		TTL:       c.ExpiresAt.Unix(),
	}
}

// recordToCart converts a stored record to a cart. time.RFC3339 parsing also accepts
// the fractional seconds written by RFC3339Nano, so older rows still load.
func recordToCart(r *cartRecord) (*cart.Cart, error) {
	items := make([]cart.CartItem, len(r.Items))
	for i, item := range r.Items {
		addedAt, err := time.Parse(time.RFC3339, item.AddedAt)
		if err != nil {
			return nil, errors.Wrap(errors.CodePersistenceError, "invalid item added_at", err)
		}
		items[i] = cart.CartItem{
			ItemID:    item.ItemID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			AddedAt:   addedAt,
		}
	}

	var times [3]time.Time
	for i, value := range []string{r.CreatedAt, r.UpdatedAt, r.ExpiresAt} {
		t, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, errors.Wrap(errors.CodePersistenceError, "invalid cart timestamp", err)
		}
		times[i] = t
	}

	return &cart.Cart{
		ID:        r.ID,
		UserID:    r.UserID,
		Items:     items,
		Version:   r.Version,
		CreatedAt: times[0],
		UpdatedAt: times[1],
		ExpiresAt: times[2],
	}, nil
}
