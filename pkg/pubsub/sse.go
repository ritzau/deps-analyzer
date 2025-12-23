package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
)

// TopicConfig configures buffering behavior for a topic
type TopicConfig struct {
	BufferSize int  // Number of events to buffer (0 = no buffering)
	ReplayAll  bool // If true, replay all buffered events; if false, only replay last event
}

// SSEPublisher implements Publisher using Server-Sent Events
type SSEPublisher struct {
	mu            sync.RWMutex
	subscriptions map[string]map[*sseSubscription]bool // topic -> set of subscriptions
	version       map[string]int                       // topic -> version counter
	eventBuffer   map[string][]Event                   // topic -> ring buffer of events
	topicConfig   map[string]TopicConfig               // topic -> configuration
	closed        bool
}

// NewSSEPublisher creates a new SSE-based publisher
func NewSSEPublisher() *SSEPublisher {
	return &SSEPublisher{
		subscriptions: make(map[string]map[*sseSubscription]bool),
		version:       make(map[string]int),
		eventBuffer:   make(map[string][]Event),
		topicConfig:   make(map[string]TopicConfig),
	}
}

// ConfigureTopic sets buffering configuration for a topic
func (p *SSEPublisher) ConfigureTopic(topic string, config TopicConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.topicConfig[topic] = config
}

// Subscribe creates a new subscription to a topic
func (p *SSEPublisher) Subscribe(ctx context.Context, topic string) (Subscription, error) {
	p.mu.Lock()

	if p.closed {
		p.mu.Unlock()
		return nil, fmt.Errorf("publisher is closed")
	}

	// Create subscription
	sub := &sseSubscription{
		topic:     topic,
		events:    make(chan Event, 100), // Buffered to prevent blocking publishers
		publisher: p,
	}

	// Register subscription
	if p.subscriptions[topic] == nil {
		p.subscriptions[topic] = make(map[*sseSubscription]bool)
	}
	p.subscriptions[topic][sub] = true

	// Get buffered events to replay (copy while holding lock)
	config := p.topicConfig[topic]
	bufferedEvents := make([]Event, len(p.eventBuffer[topic]))
	copy(bufferedEvents, p.eventBuffer[topic])

	p.mu.Unlock()

	// Replay events to new subscriber based on topic configuration
	if len(bufferedEvents) > 0 {
		eventsToReplay := bufferedEvents
		if !config.ReplayAll && len(bufferedEvents) > 0 {
			// Only replay last event
			eventsToReplay = bufferedEvents[len(bufferedEvents)-1:]
		}

		for _, event := range eventsToReplay {
			select {
			case sub.events <- event:
				// Event sent successfully
			default:
				log.Printf("Warning: could not replay event to new subscriber for topic %s", topic)
			}
		}
		log.Printf("Replayed %d event(s) for topic %s to new subscriber", len(eventsToReplay), topic)
	}

	// Handle context cancellation
	go func() {
		<-ctx.Done()
		sub.Close()
	}()

	return sub, nil
}

// Publish sends an event to all subscribers of a topic
func (p *SSEPublisher) Publish(topic string, eventType string, data interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("publisher is closed")
	}

	// Increment version for this topic
	p.version[topic]++
	version := p.version[topic]

	// Marshal data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	// Create event
	event := Event{
		Topic:   topic,
		Type:    eventType,
		Data:    jsonData,
		Version: version,
	}

	// Add to buffer if configured
	config := p.topicConfig[topic]
	if config.BufferSize > 0 {
		buffer := p.eventBuffer[topic]
		buffer = append(buffer, event)

		// Trim buffer to configured size (keep most recent events)
		if len(buffer) > config.BufferSize {
			buffer = buffer[len(buffer)-config.BufferSize:]
		}
		p.eventBuffer[topic] = buffer
	}

	// Send to all subscribers (non-blocking)
	subs := p.subscriptions[topic]
	for sub := range subs {
		select {
		case sub.events <- event:
			// Event sent successfully
		default:
			// Channel full, log warning but don't block
			log.Printf("Warning: subscription channel full for topic %s, dropping event", topic)
		}
	}

	return nil
}

// Close shuts down the publisher and all subscriptions
func (p *SSEPublisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	// Close all subscriptions
	for _, subs := range p.subscriptions {
		for sub := range subs {
			close(sub.events)
		}
	}

	// Clear subscriptions
	p.subscriptions = make(map[string]map[*sseSubscription]bool)

	return nil
}

// unsubscribe removes a subscription (called by subscription.Close())
func (p *SSEPublisher) unsubscribe(sub *sseSubscription) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if subs := p.subscriptions[sub.topic]; subs != nil {
		delete(subs, sub)
		if len(subs) == 0 {
			delete(p.subscriptions, sub.topic)
		}
	}
}

// sseSubscription implements Subscription
type sseSubscription struct {
	topic     string
	events    chan Event
	publisher *SSEPublisher
	closed    bool
	mu        sync.Mutex
}

// Topic returns the subscription topic
func (s *sseSubscription) Topic() string {
	return s.topic
}

// Events returns a channel for receiving events
func (s *sseSubscription) Events() <-chan Event {
	return s.events
}

// Close closes the subscription
func (s *sseSubscription) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	s.publisher.unsubscribe(s)

	return nil
}

// WriteSSE writes an event to an SSE response writer
// Format: "data: {json}\n\n"
func WriteSSE(w io.Writer, event Event) error {
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = fmt.Fprintf(w, "data: %s\n\n", jsonData)
	return err
}
