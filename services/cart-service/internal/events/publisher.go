// Package events provides event publishing functionality for the cart service.
package events

import (
	"context"
)

// Publisher defines the interface for publishing events.
type Publisher interface {
	// Publish publishes a single event.
	Publish(ctx context.Context, event Event) error

	// PublishBatch publishes multiple events.
	PublishBatch(ctx context.Context, events []Event) error

	// Close closes the publisher.
	Close() error
}

// Event represents a domain event.
type Event struct {
	ID            string                 `json:"id"`
	Source        string                 `json:"source"`
	Type          string                 `json:"type"`
	Time          string                 `json:"time"`
	Data          interface{}            `json:"data"`
	Metadata      EventMetadata          `json:"metadata"`
	DataVersion   string                 `json:"data_version"`
}

// EventMetadata contains event metadata.
type EventMetadata struct {
	TraceID       string `json:"trace_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	UserID        string `json:"user_id,omitempty"`
}

// Event types
const (
	EventTypeCartCreated    = "cart.created"
	EventTypeItemAdded      = "cart.item_added"
	EventTypeItemRemoved    = "cart.item_removed"
	EventTypeItemUpdated    = "cart.item_updated"
	EventTypeCartCleared    = "cart.cleared"
	EventTypeCartAbandoned  = "cart.abandoned"
)
