package mdk

import (
	"context"
	"time"
)

// Event is an immutable record of something that happened.
type Event struct {
	ID         string
	Namespace  string         // e.g. "commerce.order"
	Type       string         // e.g. "created", "cancelled"
	Payload    map[string]any
	OccurredAt time.Time
	TraceID    string
}

// EventHandler is a function that processes an event.
type EventHandler func(ctx context.Context, e Event) error

// SubscriptionInfo represents metadata about an active event subscription.
type SubscriptionInfo struct {
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
	Handler   string `json:"handler"`
}

// EventBus is the pub/sub interface modules use to emit and react to events.
type EventBus interface {
	// Publish emits an event to all subscribers.
	Publish(ctx context.Context, e Event) error

	// Subscribe registers a handler for a namespace+type combination.
	// Returns an unsubscribe function.
	Subscribe(namespace, eventType string, handler EventHandler) (unsubscribe func(), err error)

	// Subscribers returns a list of all active event subscriptions.
	Subscribers() []SubscriptionInfo
}
