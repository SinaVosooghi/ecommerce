// Package eventbridge provides an EventBridge implementation of the event publisher.
package eventbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/google/uuid"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/core/cart"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/events"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/events/models"
	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/logging"
)

// PublisherConfig holds configuration for the EventBridge publisher.
type PublisherConfig struct {
	Region   string
	BusName  string
	Source   string
	Endpoint string // Optional, for local testing
}

// Publisher is an EventBridge implementation of the event publisher.
type Publisher struct {
	client  *eventbridge.Client
	busName string
	source  string
	logger  *logging.Logger
}

// NewPublisher creates a new EventBridge publisher.
func NewPublisher(ctx context.Context, cfg PublisherConfig, logger *logging.Logger) (*Publisher, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var client *eventbridge.Client
	if cfg.Endpoint != "" {
		client = eventbridge.NewFromConfig(awsCfg, func(o *eventbridge.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	} else {
		client = eventbridge.NewFromConfig(awsCfg)
	}

	return &Publisher{
		client:  client,
		busName: cfg.BusName,
		source:  cfg.Source,
		logger:  logger,
	}, nil
}

// Publish publishes a single event to EventBridge.
func (p *Publisher) Publish(ctx context.Context, event events.Event) error {
	detail, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	entry := types.PutEventsRequestEntry{
		EventBusName: aws.String(p.busName),
		Source:       aws.String(p.source),
		DetailType:   aws.String(event.Type),
		Detail:       aws.String(string(detail)),
		Time:         aws.Time(time.Now().UTC()),
	}

	// Add trace ID if present
	if event.Metadata.TraceID != "" {
		entry.TraceHeader = aws.String(event.Metadata.TraceID)
	}

	_, err = p.client.PutEvents(ctx, &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{entry},
	})
	if err != nil {
		p.logger.WithContext(ctx).WithError(err).Error("Failed to publish event")
		return fmt.Errorf("failed to publish event: %w", err)
	}

	p.logger.WithContext(ctx).
		WithField("event_type", event.Type).
		WithField("event_id", event.ID).
		Debug("Event published")

	return nil
}

// PublishBatch publishes multiple events to EventBridge.
func (p *Publisher) PublishBatch(ctx context.Context, eventList []events.Event) error {
	if len(eventList) == 0 {
		return nil
	}

	entries := make([]types.PutEventsRequestEntry, 0, len(eventList))

	for _, event := range eventList {
		detail, err := json.Marshal(event)
		if err != nil {
			p.logger.WithContext(ctx).WithError(err).Error("Failed to marshal event")
			continue
		}

		entry := types.PutEventsRequestEntry{
			EventBusName: aws.String(p.busName),
			Source:       aws.String(p.source),
			DetailType:   aws.String(event.Type),
			Detail:       aws.String(string(detail)),
			Time:         aws.Time(time.Now().UTC()),
		}

		if event.Metadata.TraceID != "" {
			entry.TraceHeader = aws.String(event.Metadata.TraceID)
		}

		entries = append(entries, entry)
	}

	// EventBridge allows max 10 entries per batch
	for i := 0; i < len(entries); i += 10 {
		end := i + 10
		if end > len(entries) {
			end = len(entries)
		}

		batch := entries[i:end]
		result, err := p.client.PutEvents(ctx, &eventbridge.PutEventsInput{
			Entries: batch,
		})
		if err != nil {
			p.logger.WithContext(ctx).WithError(err).Error("Failed to publish event batch")
			return fmt.Errorf("failed to publish event batch: %w", err)
		}

		if result.FailedEntryCount > 0 {
			p.logger.WithContext(ctx).
				WithField("failed_count", result.FailedEntryCount).
				Warn("Some events failed to publish")
		}
	}

	return nil
}

// Close closes the publisher (no-op for EventBridge).
func (p *Publisher) Close() error {
	return nil
}

// CartEventPublisher wraps the publisher with cart-specific methods.
type CartEventPublisher struct {
	publisher *Publisher
	source    string
}

// NewCartEventPublisher creates a new cart event publisher.
func NewCartEventPublisher(publisher *Publisher) *CartEventPublisher {
	return &CartEventPublisher{
		publisher: publisher,
		source:    publisher.source,
	}
}

// PublishCartCreated publishes a cart.created event.
func (p *CartEventPublisher) PublishCartCreated(ctx context.Context, c *cart.Cart) error {
	event := p.createEvent(ctx, events.EventTypeCartCreated, models.CartCreatedData{
		CartID:    c.ID,
		UserID:    c.UserID,
		CreatedAt: c.CreatedAt,
		ExpiresAt: c.ExpiresAt,
	})
	return p.publisher.Publish(ctx, event)
}

// PublishItemAdded publishes a cart.item_added event.
func (p *CartEventPublisher) PublishItemAdded(ctx context.Context, c *cart.Cart, item *cart.CartItem) error {
	event := p.createEvent(ctx, events.EventTypeItemAdded, models.ItemAddedData{
		CartID: c.ID,
		UserID: c.UserID,
		Item: models.CartItemDTO{
			ItemID:    item.ItemID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			Subtotal:  item.UnitPrice * int64(item.Quantity),
			AddedAt:   item.AddedAt,
		},
		CartTotal: c.TotalPrice(),
		ItemCount: c.ItemCount(),
	})
	return p.publisher.Publish(ctx, event)
}

// PublishItemRemoved publishes a cart.item_removed event.
func (p *CartEventPublisher) PublishItemRemoved(ctx context.Context, c *cart.Cart, itemID string) error {
	event := p.createEvent(ctx, events.EventTypeItemRemoved, models.ItemRemovedData{
		CartID:    c.ID,
		UserID:    c.UserID,
		ItemID:    itemID,
		CartTotal: c.TotalPrice(),
		ItemCount: c.ItemCount(),
	})
	return p.publisher.Publish(ctx, event)
}

// PublishItemUpdated publishes a cart.item_updated event.
func (p *CartEventPublisher) PublishItemUpdated(ctx context.Context, c *cart.Cart, item *cart.CartItem) error {
	event := p.createEvent(ctx, events.EventTypeItemUpdated, models.ItemUpdatedData{
		CartID: c.ID,
		UserID: c.UserID,
		Item: models.CartItemDTO{
			ItemID:    item.ItemID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			Subtotal:  item.UnitPrice * int64(item.Quantity),
			AddedAt:   item.AddedAt,
		},
		CartTotal: c.TotalPrice(),
	})
	return p.publisher.Publish(ctx, event)
}

// PublishCartCleared publishes a cart.cleared event.
func (p *CartEventPublisher) PublishCartCleared(ctx context.Context, c *cart.Cart) error {
	event := p.createEvent(ctx, events.EventTypeCartCleared, models.CartClearedData{
		CartID: c.ID,
		UserID: c.UserID,
	})
	return p.publisher.Publish(ctx, event)
}

func (p *CartEventPublisher) createEvent(ctx context.Context, eventType string, data interface{}) events.Event {
	return events.Event{
		ID:          uuid.New().String(),
		Source:      p.source,
		Type:        eventType,
		Time:        time.Now().UTC().Format(time.RFC3339),
		Data:        data,
		DataVersion: "1.0",
		Metadata: events.EventMetadata{
			TraceID:       logging.TraceIDFromContext(ctx),
			CorrelationID: logging.RequestIDFromContext(ctx),
			UserID:        logging.UserIDFromContext(ctx),
		},
	}
}
