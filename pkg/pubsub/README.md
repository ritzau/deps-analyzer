# PubSub Package

Event-driven publish/subscribe system for real-time state updates using Server-Sent Events (SSE).

## Architecture

### Server-Driven State Machine
- Server controls state transitions
- Clients subscribe and react to events
- No race conditions - server is source of truth
- Events published as state changes occur

### Configurable Event Buffering
Each topic can be configured independently:
- **Buffer Size**: Number of recent events to keep
- **Replay Strategy**: Send all buffered events or only the last one

## Usage

### Basic Setup

```go
import "github.com/ritzau/deps-analyzer/pkg/pubsub"

// Create publisher
publisher := pubsub.NewSSEPublisher()
defer publisher.Close()

// Configure topics
publisher.ConfigureTopic("workspace_status", pubsub.TopicConfig{
    BufferSize: 10,    // Keep last 10 events
    ReplayAll:  false, // Only replay last event to new subscribers
})

publisher.ConfigureTopic("build_log", pubsub.TopicConfig{
    BufferSize: 100,   // Keep last 100 log entries
    ReplayAll:  true,  // Replay entire log history to new subscribers
})
```

### Publishing Events

```go
// Publish workspace state changes
status := pubsub.WorkspaceStatus{
    State:   "analyzing",
    Message: "Analyzing dependencies...",
    Step:    2,
    Total:   5,
}
publisher.Publish("workspace_status", "analyzing", status)
```

### Subscribing (Server-Side)

```go
// In HTTP handler
func handleSSE(w http.ResponseWriter, r *http.Request) {
    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    // Create subscription
    sub, err := publisher.Subscribe(r.Context(), "workspace_status")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer sub.Close()

    // Stream events
    flusher, _ := w.(http.Flusher)
    flusher.Flush()

    for event := range sub.Events() {
        pubsub.WriteSSE(w, event)
        flusher.Flush()
    }
}
```

### Subscribing (Client-Side)

```javascript
// JavaScript EventSource API
const eventSource = new EventSource('/api/subscribe/workspace_status');

eventSource.onmessage = function(event) {
    const sseEvent = JSON.parse(event.data);
    const status = JSON.parse(sseEvent.data);

    console.log('State:', status.state);
    console.log('Message:', status.message);
    console.log('Progress:', status.step, '/', status.total);
};

eventSource.onerror = function(error) {
    console.error('SSE error:', error);
    eventSource.close();
};
```

## Configuration Strategies

### Current State Only (Default)
Best for status updates where only the latest state matters:

```go
publisher.ConfigureTopic("workspace_status", pubsub.TopicConfig{
    BufferSize: 1,     // Keep only current state
    ReplayAll:  false, // Send only current state
})
```

**Use cases:**
- Workspace analysis state
- Build status
- Current progress indicators

### Recent History
Keep recent history for debugging or showing recent activity:

```go
publisher.ConfigureTopic("notifications", pubsub.TopicConfig{
    BufferSize: 20,    // Keep last 20 notifications
    ReplayAll:  false, // Only show latest notification
})
```

**Use cases:**
- Notification feeds
- Recent errors/warnings
- Activity logs

### Full Event Log
Replay entire history for comprehensive state reconstruction:

```go
publisher.ConfigureTopic("build_events", pubsub.TopicConfig{
    BufferSize: 1000,  // Keep up to 1000 events
    ReplayAll:  true,  // Replay full history
})
```

**Use cases:**
- Build logs
- Audit trails
- Debug event sequences

### No Buffering
Only receive events published after subscription:

```go
publisher.ConfigureTopic("realtime_metrics", pubsub.TopicConfig{
    BufferSize: 0,     // No buffering
    ReplayAll:  false,
})
```

**Use cases:**
- Real-time metrics (don't care about history)
- Live monitoring
- Ephemeral notifications

## Benefits

✅ **No Race Conditions**: Server controls when events are published
✅ **Reliable State Sync**: Late subscribers get current state instantly
✅ **Flexible History**: Per-topic buffer configuration
✅ **Browser Native**: Uses EventSource API (no external dependencies)
✅ **Automatic Cleanup**: Buffers are size-limited, old events discarded
✅ **Reconnection Friendly**: Refresh page and get current state

## Event Format

Events are JSON-encoded with metadata:

```json
{
    "topic": "workspace_status",
    "type": "analyzing",
    "data": {
        "state": "analyzing",
        "message": "Analyzing dependencies...",
        "step": 2,
        "total": 5
    },
    "version": 42
}
```

- **topic**: Subscription topic
- **type**: Event type (semantic label)
- **data**: Event payload (topic-specific structure)
- **version**: Monotonically increasing version per topic

## Thread Safety

All methods are thread-safe:
- Multiple goroutines can publish concurrently
- Subscriptions can be created/closed from any goroutine
- Buffer operations are protected by mutex

## Performance

- **Non-blocking publish**: Full subscriber channels drop events (with warning)
- **Buffered channels**: 100-event buffer per subscription prevents blocking
- **Efficient replay**: Events copied once, then streamed
- **Memory bounded**: Buffer size limits memory per topic

## Testing

See [sse_test.go](sse_test.go) for comprehensive test coverage:
- Event buffering with size limits
- Replay all vs replay last
- No buffering behavior
- Concurrent subscribers
