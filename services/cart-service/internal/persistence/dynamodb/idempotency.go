package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/idempotency"
)

// Key prefix for idempotency records, stored in the cart table.
const idempotencyKeyPrefix = "IDEMP#"

var _ idempotency.Store = (*IdempotencyStore)(nil)

// IdempotencyStore is a DynamoDB-backed idempotency.Store shared by all service replicas.
// Records reuse the table's "ttl" attribute for automatic cleanup; because TTL deletion is
// delayed, expiry is also enforced on read via expires_at.
type IdempotencyStore struct {
	client *Client
	now    func() time.Time
}

// NewIdempotencyStore creates a DynamoDB idempotency store.
func NewIdempotencyStore(client *Client) *IdempotencyStore {
	return &IdempotencyStore{client: client, now: time.Now}
}

type idempotencyRecord struct {
	PK          string `dynamodbav:"PK"`
	SK          string `dynamodbav:"SK"`
	Type        string `dynamodbav:"type"`
	Fingerprint string `dynamodbav:"fingerprint"`
	Completed   bool   `dynamodbav:"completed"`
	StatusCode  int    `dynamodbav:"status_code,omitempty"`
	ContentType string `dynamodbav:"content_type,omitempty"`
	Body        []byte `dynamodbav:"body,omitempty"`
	ExpiresAt   int64  `dynamodbav:"expires_at"`
	TTL         int64  `dynamodbav:"ttl"`
}

func idempotencyKey(key string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: idempotencyKeyPrefix + key},
		"SK": &types.AttributeValueMemberS{Value: idempotencyKeyPrefix},
	}
}

func (s *IdempotencyStore) put(ctx context.Context, key string, rec idempotencyRecord, conditional bool) error {
	rec.PK = idempotencyKeyPrefix + key
	rec.SK = idempotencyKeyPrefix
	rec.Type = "IDEMPOTENCY"
	rec.TTL = rec.ExpiresAt

	item, err := attributevalue.MarshalMap(rec)
	if err != nil {
		return fmt.Errorf("marshal idempotency record: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(s.client.tableName),
		Item:      item,
	}
	if conditional {
		input.ConditionExpression = aws.String("attribute_not_exists(PK) OR expires_at <= :now")
		input.ExpressionAttributeValues = map[string]types.AttributeValue{
			":now": &types.AttributeValueMemberN{Value: strconv.FormatInt(s.now().Unix(), 10)},
		}
		input.ReturnValuesOnConditionCheckFailure = types.ReturnValuesOnConditionCheckFailureAllOld
	}

	_, err = s.client.db.PutItem(ctx, input)
	return err
}

// Reserve implements idempotency.Store.
func (s *IdempotencyStore) Reserve(ctx context.Context, key, fingerprint string, lockTTL time.Duration) (*idempotency.Record, error) {
	err := s.put(ctx, key, idempotencyRecord{
		Fingerprint: fingerprint,
		ExpiresAt:   s.now().Add(lockTTL).Unix(),
	}, true)
	if err == nil {
		return nil, nil
	}

	var condErr *types.ConditionalCheckFailedException
	if !errors.As(err, &condErr) {
		return nil, fmt.Errorf("reserve idempotency key: %w", err)
	}

	var existing idempotencyRecord
	if err := attributevalue.UnmarshalMap(condErr.Item, &existing); err != nil {
		return nil, fmt.Errorf("unmarshal idempotency record: %w", err)
	}
	return &idempotency.Record{
		Fingerprint: existing.Fingerprint,
		Completed:   existing.Completed,
		StatusCode:  existing.StatusCode,
		ContentType: existing.ContentType,
		Body:        existing.Body,
		ExpiresAt:   time.Unix(existing.ExpiresAt, 0),
	}, nil
}

// Complete implements idempotency.Store.
func (s *IdempotencyStore) Complete(ctx context.Context, key string, record *idempotency.Record, ttl time.Duration) error {
	err := s.put(ctx, key, idempotencyRecord{
		Fingerprint: record.Fingerprint,
		Completed:   true,
		StatusCode:  record.StatusCode,
		ContentType: record.ContentType,
		Body:        record.Body,
		ExpiresAt:   s.now().Add(ttl).Unix(),
	}, false)
	if err != nil {
		return fmt.Errorf("complete idempotency key: %w", err)
	}
	return nil
}

// Release implements idempotency.Store.
func (s *IdempotencyStore) Release(ctx context.Context, key string) error {
	_, err := s.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.client.tableName),
		Key:       idempotencyKey(key),
	})
	if err != nil {
		return fmt.Errorf("release idempotency key: %w", err)
	}
	return nil
}
