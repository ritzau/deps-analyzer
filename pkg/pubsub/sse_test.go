package pubsub

import (
	"context"
	"testing"
	"time"
)

func TestEventBuffer(t *testing.T) {
	pub := NewSSEPublisher()
	defer pub.Close()

	// Configure topic with buffer size 3, replay all
	pub.ConfigureTopic("test", TopicConfig{
		BufferSize: 3,
		ReplayAll:  true,
	})

	// Publish 5 events
	for i := 1; i <= 5; i++ {
		err := pub.Publish("test", "event", map[string]int{"num": i})
		if err != nil {
			t.Fatalf("Failed to publish event %d: %v", i, err)
		}
	}

	// Subscribe and verify we get last 3 events
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	sub, err := pub.Subscribe(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer sub.Close()

	// Should receive last 3 events (3, 4, 5)
	receivedCount := 0
	for receivedCount < 3 {
		select {
		case event := <-sub.Events():
			receivedCount++
			t.Logf("Received replayed event version %d", event.Version)
			// Events should be 3, 4, 5 (last 3 of 5)
			expectedVersion := receivedCount + 2
			if event.Version != expectedVersion {
				t.Errorf("Expected version %d, got %d", expectedVersion, event.Version)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Timeout waiting for event %d", receivedCount+1)
		}
	}

	if receivedCount != 3 {
		t.Errorf("Expected 3 replayed events, got %d", receivedCount)
	}
}

func TestReplayLastOnly(t *testing.T) {
	pub := NewSSEPublisher()
	defer pub.Close()

	// Configure topic with buffer size 5, replay only last
	pub.ConfigureTopic("test", TopicConfig{
		BufferSize: 5,
		ReplayAll:  false,
	})

	// Publish 3 events
	for i := 1; i <= 3; i++ {
		err := pub.Publish("test", "event", map[string]int{"num": i})
		if err != nil {
			t.Fatalf("Failed to publish event %d: %v", i, err)
		}
	}

	// Subscribe and verify we get only last event
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	sub, err := pub.Subscribe(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer sub.Close()

	// Should receive only last event (version 3)
	select {
	case event := <-sub.Events():
		if event.Version != 3 {
			t.Errorf("Expected version 3, got %d", event.Version)
		}
		t.Logf("Received last event version %d", event.Version)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for event")
	}

	// Verify no more events are sent
	select {
	case event := <-sub.Events():
		t.Errorf("Received unexpected extra event version %d", event.Version)
	case <-time.After(50 * time.Millisecond):
		// Good, no extra events
	}
}

func TestNoBuffer(t *testing.T) {
	pub := NewSSEPublisher()
	defer pub.Close()

	// Configure topic with no buffer
	pub.ConfigureTopic("test", TopicConfig{
		BufferSize: 0,
		ReplayAll:  false,
	})

	// Publish events before subscribing
	for i := 1; i <= 3; i++ {
		err := pub.Publish("test", "event", map[string]int{"num": i})
		if err != nil {
			t.Fatalf("Failed to publish event %d: %v", i, err)
		}
	}

	// Subscribe - should not receive any replayed events
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	sub, err := pub.Subscribe(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer sub.Close()

	// Verify no events are received (because none were buffered)
	select {
	case event := <-sub.Events():
		t.Errorf("Received unexpected replayed event version %d", event.Version)
	case <-time.After(50 * time.Millisecond):
		// Good, no events replayed
		t.Log("Correctly received no events (buffer disabled)")
	}

	// Now publish a new event - subscriber should receive it
	err = pub.Publish("test", "event", map[string]int{"num": 4})
	if err != nil {
		t.Fatalf("Failed to publish new event: %v", err)
	}

	select {
	case event := <-sub.Events():
		if event.Version != 4 {
			t.Errorf("Expected version 4, got %d", event.Version)
		}
		t.Logf("Received new event version %d", event.Version)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for new event")
	}
}
