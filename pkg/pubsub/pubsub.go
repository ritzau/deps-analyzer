package pubsub

import (
	"context"
	"encoding/json"
)

// Event represents a pub/sub event
type Event struct {
	Topic   string          `json:"topic"`   // Subscription topic (e.g., "workspace_status", "target_graph")
	Type    string          `json:"type"`    // Event type (e.g., "initializing", "bazel_querying", "loading", "partial_data")
	Data    json.RawMessage `json:"data"`    // Event payload
	Version int             `json:"version"` // Version number for ordering
}

// Subscription represents a client subscription to a topic
type Subscription interface {
	// Topic returns the subscription topic
	Topic() string

	// Events returns a channel for receiving events
	Events() <-chan Event

	// Close closes the subscription
	Close() error
}

// Publisher manages pub/sub subscriptions and event publishing
type Publisher interface {
	// Subscribe creates a new subscription to a topic
	// Context cancellation will close the subscription
	Subscribe(ctx context.Context, topic string) (Subscription, error)

	// Publish sends an event to all subscribers of a topic
	Publish(topic string, eventType string, data interface{}) error

	// Close shuts down the publisher and all subscriptions
	Close() error
}

// WorkspaceStatus represents workspace analysis state
type WorkspaceStatus struct {
	State   string `json:"state"`   // initializing, bazel_querying, binaries_ready, targets_ready, ready
	Message string `json:"message"` // Human-readable status message
	Step    int    `json:"step"`    // Current step number (1-based)
	Total   int    `json:"total"`   // Total number of steps
}

// TargetGraphData represents partial or complete graph data
type TargetGraphData struct {
	TargetsCount      int  `json:"targets_count"`
	DependenciesCount int  `json:"dependencies_count"`
	Complete          bool `json:"complete"` // True when all data is loaded
}
