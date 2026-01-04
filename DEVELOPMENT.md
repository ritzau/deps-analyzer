# Development Guide

This document provides architecture context, development guidelines, and implementation details for contributors.

## Project Overview

Bazel C++ Dependency Analyzer helps analyze and visualize C++ dependencies in Bazel projects, with special focus on:

1. Detecting files not covered by any Bazel target
2. Identifying circular dependencies (both target and file level)
3. Analyzing package complexity and suggesting refactoring
4. Comparing Bazel dependency graph with actual file dependencies from .d files
5. Analyzing shared library (.so) complexity and overlapping dependencies

## Architecture

### Technology Stack

- **Language:** Go 1.21+ (chosen for performance, static binary, built-in concurrency)
- **Graph Analysis:** Custom BFS/traversal algorithms
- **Web UI:**
  - Backend: Go HTTP server with Server-Sent Events (SSE) for real-time updates
  - Frontend: Vanilla JavaScript with Cytoscape.js for interactive graph visualization
  - No heavy frontend framework needed

### Core Components

#### 1. Bazel Integration (`pkg/bazel/`)

- Query targets: `bazel query 'kind("cc_.* rule", //...)'`
- Get dependencies: `bazel query 'deps(//path:target)'`
- Parse XML output for target metadata
- Extract workspace name from `bazel mod graph`

#### 2. Dependency Analysis (`pkg/deps/`, `pkg/symbols/`)

- Parse `.d` files from `bazel-out/` for compile dependencies
- Use `nm` to analyze symbols from object files
- Extract file-level include dependencies
- Track which symbols are actually used between targets

#### 3. Binary Analysis (`pkg/binaries/`)

- Analyze binaries and shared libraries
- Detect overlapping dependencies (same code linked into multiple binaries)
- Track system library dependencies
- Identify data dependencies

#### 4. Lens Rendering System (`pkg/lens/`)

The lens system provides sophisticated graph filtering and visualization:

- **Distance Computation**: BFS-based distance from selected nodes
- **Visibility Rules**: Control which nodes/edges appear based on type, distance, and filters
- **Hierarchy Management**: Package → Target → File three-level hierarchy
- **Collapse Levels**: Dynamically hide children of collapsed nodes
- **Edge Aggregation**: Combine edges when child nodes are hidden
- **Diff-based Updates**: Compute incremental changes for efficient UI updates

#### 5. PubSub System (`pkg/pubsub/`)

Event-driven publish/subscribe using Server-Sent Events (SSE):

```go
// Configure topic with buffering
publisher.ConfigureTopic("workspace_status", pubsub.TopicConfig{
    BufferSize: 1,     // Keep only current state
    ReplayAll:  false, // Send only current state to new subscribers
})

// Publish events
publisher.Publish("workspace_status", "analyzing", status)

// Subscribe (HTTP handler)
sub, _ := publisher.Subscribe(r.Context(), "workspace_status")
for event := range sub.Events() {
    pubsub.WriteSSE(w, event)
    flusher.Flush()
}
```

**Benefits:**

- Server controls state transitions (no race conditions)
- Late subscribers get current state instantly
- Configurable buffering per topic
- Browser-native EventSource API

#### 6. File Watching (`pkg/watcher/`)

- Monitor BUILD files and bazel-out/ for changes
- Smart debouncing (1.5s quiet period, 10s max wait)
- Intelligent change detection for incremental updates
- Triggers appropriate re-analysis based on what changed

#### 7. Structured Logging (`pkg/logging/`)

- Wraps Go's `log/slog` with custom compact handler
- Request ID tracking for HTTP requests
- Context-aware logging
- Format: `[LEVEL] HH:MM:SS message | key=value key=value`

### Project Structure

```
deps-analyzer/
├── cmd/
│   └── deps-analyzer/
│       └── main.go              # Entry point
├── pkg/
│   ├── analysis/
│   │   └── runner.go            # Analysis orchestration
│   ├── bazel/
│   │   ├── query.go             # Bazel query interface
│   │   └── workspace.go         # Workspace name extraction
│   ├── binaries/
│   │   └── analyzer.go          # Binary analysis
│   ├── deps/
│   │   └── parser.go            # .d file parsing
│   ├── lens/
│   │   ├── lens.go              # Lens configuration
│   │   ├── distance.go          # BFS distance computation
│   │   ├── renderer.go          # Graph rendering pipeline
│   │   └── diff.go              # Diff computation
│   ├── logging/
│   │   ├── logger.go            # Structured logger
│   │   ├── compact_handler.go   # Compact console format
│   │   └── middleware.go        # HTTP logging middleware
│   ├── model/
│   │   └── graph.go             # Graph data structures
│   ├── pubsub/
│   │   └── sse.go               # SSE pub/sub system
│   ├── symbols/
│   │   └── analyzer.go          # Symbol dependency analysis
│   ├── watcher/
│   │   ├── watcher.go           # File system watching
│   │   └── debouncer.go         # Change debouncing
│   └── web/
│       ├── server.go            # HTTP server
│       └── static/
│           ├── index.html       # Main UI
│           ├── app.js           # Graph visualization
│           ├── logger.js        # Frontend structured logging
│           ├── view-state.js    # State management
│           ├── lens-config.js   # Lens configurations
│           └── lens-controls.js # Lens UI controls
├── example/                     # Test workspace
├── DEVELOPMENT.md               # This file
├── README.md                    # User documentation
└── TODO.md                      # Feature ideas and backlog
```

## Development Workflow

### Setup

```bash
git clone https://github.com/ritzau/deps-analyzer.git
cd deps-analyzer
go mod download
```

### Building

```bash
go build -o deps-analyzer cmd/deps-analyzer/main.go
```

### Testing

```bash
# Run all tests
go test ./...

# Test against example workspace
cd example
bazel build //...
cd ..
./deps-analyzer --web --watch --workspace=./example
```

### Frontend Development

Frontend files are served from `pkg/web/static/`. The server embeds these files, so you need to rebuild after changes:

```bash
# Make frontend changes
vim pkg/web/static/app.js

# Rebuild
go build -o deps-analyzer cmd/deps-analyzer/main.go

# Or use file watching during development
# (changes require browser refresh but no rebuild)
```

### Logging

**Backend:**

```go
import "github.com/ritzau/deps-analyzer/pkg/logging"

logging.Info("message", "key", value)
logging.Debug("internal detail", "data", obj)
logging.Error("problem occurred", "error", err)
```

**Frontend:**

```javascript
appLogger.info("User clicked node", { nodeId: id });
appLogger.debug("Internal state", { state: viewState });
appLogger.error("Request failed", { error: err.message });

// Enable DEBUG logs in browser console:
appLogger.setLevel(LogLevel.DEBUG);
```

## Key Algorithms

### Distance Computation (BFS)

The lens system uses BFS to compute distances from selected nodes:

1. Start with selected nodes at distance 0
2. **Package expansion**: When processing a package node, add all its child targets to the queue
3. For each node, visit all neighbors (following edges in both directions)
4. Track minimum distance to each node
5. Nodes never reached have distance "infinite"

This enables "show neighbors within N hops" filtering.

### Lens Rendering Pipeline

1. **Compute distances** from selected nodes using BFS
2. **Assign lenses** to nodes (default vs detail based on selection)
3. **Apply visibility rules** from distance rules (node types, file types, visibility flags)
4. **Build hierarchy** (package → target → file parent relationships)
5. **Filter collapsed children** (hide children of collapsed nodes)
6. **Aggregate edges** (combine edges when targets are collapsed)
7. **Sort deterministically** (for stable Dagre layouts)

### Diff Computation

For efficient UI updates:

1. **Hash** the lens configuration (SHA256)
2. **Check cache** for previously rendered graph with same hash
3. If cache hit:
   - Compute diff (added/removed/modified nodes and edges)
   - Send diff if < 50% of graph changed
   - Otherwise send full graph
4. Frontend applies diff to current graph for smooth transitions

## Architecture Decisions

### Why Go over Python?

- **Performance:** Native compilation, ~10x faster for graph algorithms
- **Static binary:** Single executable, no dependencies
- **Concurrency:** Built-in goroutines for file watching + web server
- **Deployment:** Cross-compile for any platform

### Why Server-Side Lens Rendering?

Original implementation had client-side lens rendering in JavaScript. Migrated to Go for:

- **10x+ performance improvement** for large graphs (1000+ nodes)
- **Single source of truth** for rendering logic
- **Better scalability** for complex lens configurations
- **Diff-based updates** reduce bandwidth

### Why SSE over WebSockets?

- **Simpler**: Unidirectional server→client fits pub/sub model
- **Browser native**: EventSource API, no library needed
- **Automatic reconnection**: Built into browser
- **HTTP/2 friendly**: Multiplexed over single connection

### Why Cytoscape.js?

- **Mature**: Battle-tested graph visualization library
- **Feature-rich**: Layouts, styling, events, extensions
- **Performance**: Handles 1000+ nodes smoothly
- **No framework lock-in**: Works with vanilla JS

## Testing Strategy

### Unit Tests

- Graph algorithms (distance, cycles)
- Diff computation
- Parser edge cases

### Integration Tests

- Full analysis pipeline with example workspace
- SSE pub/sub system
- File watcher debouncing

### Manual Testing

The `example/` workspace has intentional problems:

- Uncovered files (orphaned.cc)
- Monolithic packages (util with 4 unrelated files)
- Cross-package dependencies
- Shared library complexity
- Internal coupling (math.cc → strings.h)

See [example/README.md](example/README.md) for full test cases.

## Common Development Tasks

### Adding a New Analysis Phase

1. Create analyzer in appropriate `pkg/` directory
2. Add to `pkg/analysis/runner.go` pipeline
3. Publish events via `pubsub` for UI updates
4. Add UI display in `pkg/web/static/`

### Adding a New Lens Feature

1. Update `pkg/lens/lens.go` structs (LensConfig, DistanceRule, etc.)
2. Modify `pkg/lens/renderer.go` rendering pipeline
3. Update frontend `lens-config.js` default configurations
4. Add UI controls in `lens-controls.js`
5. Update `index.html` if adding new UI elements

### Debugging Tips

**Enable DEBUG logging:**

```bash
# Backend: modify pkg/logging/logger.go init() to use slog.LevelDebug
# Frontend: in browser console
appLogger.setLevel(LogLevel.DEBUG)
```

**Inspect lens rendering:**

- Backend logs distance computation and visibility decisions at DEBUG level
- Check browser Network tab for `/api/module/graph/lens` request/response
- Use `previousHash` parameter to test diff computation

**Test file watching:**

```bash
# Run with watch
./deps-analyzer --web --watch --workspace=./example

# In another terminal
cd example
touch BUILD  # Should trigger re-analysis
```

## Performance Considerations

- **Large graphs (1000+ nodes):** Server-side rendering essential
- **Frequent lens changes:** Diff-based updates reduce bandwidth
- **File watching:** Debouncing prevents excessive re-analysis
- **SSE buffering:** Prevents memory leaks with late subscribers

## Future Improvements

See [TODO.md](TODO.md) for planned features and ideas.

## References

- [Bazel Query Language](https://bazel.build/query/language)
- [Cytoscape.js Documentation](https://js.cytoscape.org/)
- [Server-Sent Events Specification](https://html.spec.whatwg.org/multipage/server-sent-events.html)
- [Go slog Package](https://pkg.go.dev/log/slog)
