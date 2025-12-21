# Bazel C++ Analyzer - Development Guide

## Overview

This project creates a comprehensive analyzer for C++ Bazel codebases that:

1. Detects files not covered by any Bazel target
2. Identifies circular dependencies (both target and file level)
3. Analyzes package complexity and suggests refactoring
4. Compares Bazel dependency graph with actual file dependencies from .d files
5. Helps migrate from .so-heavy architecture to cleaner static linking

## Test Workspace

A complete test workspace is provided in `/home/claude/example/` with
intentional problems:

- Uncovered files (orphaned.cc not in any BUILD)
- Monolithic packages (util with mixed responsibilities)
- Cross-package dependencies
- Shared library (.so) complexity
- Internal package coupling

See `example/README.md` for details on all the test cases.

## Architecture Decisions from Discussion

### Technology Stack

- **Language:** Go (chosen over Python for performance, static binary, GC)
- **Graph Analysis:** Use gonum/graph library
- **Web UI:**
  - Backend: Go HTTP server with WebSocket for real-time updates
  - Frontend: Cytoscape.js for interactive dependency graph visualization
  - No React needed - vanilla JS/HTML/CSS

### Core Features

1. **File Watching:**

   - Monitor BUILD files and source files for changes
   - Trigger re-analysis automatically
   - Debounce to avoid excessive re-scans

2. **Bazel Integration:**

   - Query targets: `bazel query 'kind("cc_.* rule", //...)'`
   - Get dependencies: `bazel query 'deps(//path:target)'`
   - Parse XML output for file lists

3. **Dependency File Parsing:**

   - Parse .d files from bazel-out/
   - Extract file-level include dependencies
   - Compare with Bazel target dependencies

4. **Graph Analysis:**

   - Build both target-level and file-level dependency graphs
   - Use Tarjan's algorithm (via gonum) for cycle detection
   - Identify strongly connected components

5. **Package Analysis:**

   - Detect monolithic packages (many unrelated files)
   - Suggest splits based on:
     - Co-inclusion patterns from .d files
     - Independent usage patterns
     - Internal coupling
   - Suggest merges for tightly coupled small packages

6. **Web UI:**
   - Real-time graph visualization with Cytoscape.js
   - Interactive: zoom, pan, click nodes for details
   - Color coding:
     - Red: cycles
     - Orange: high complexity
     - Yellow: refactoring candidates
     - Green: clean structure
   - Sidebar with:
     - Uncovered files list
     - Package complexity scores
     - Refactoring suggestions
     - Analysis timestamp

### Project Structure

```
bazel_analyzer/
├── cmd/
│   └── analyzer/
│       └── main.go           # Entry point
├── pkg/
│   ├── bazel/
│   │   ├── query.go          # Bazel query interface
│   │   ├── parser.go         # Parse Bazel XML output
│   │   └── targets.go        # Target data structures
│   ├── deps/
│   │   ├── dfile.go          # Parse .d files
│   │   └── graph.go          # Dependency graph building
│   ├── analysis/
│   │   ├── coverage.go       # File coverage analysis
│   │   ├── cycles.go         # Cycle detection
│   │   ├── complexity.go     # Package complexity scoring
│   │   └── suggestions.go    # Refactoring suggestions
│   ├── watcher/
│   │   └── watcher.go        # File system watching
│   └── web/
│       ├── server.go         # HTTP/WebSocket server
│       ├── handlers.go       # API handlers
│       └── static/
│           ├── index.html
│           ├── app.js        # Frontend logic
│           └── styles.css
├── go.mod
├── go.sum
└── README.md
```

## Key Algorithms

### 1. Coverage Analysis

```
for each .cc/.h file in workspace:
    if file not in any BUILD file:
        report as uncovered
```

### 2. Dependency Graph Construction

**From Bazel:**

```
targets = bazel query 'kind("cc_.* rule", //...)'
for each target:
    deps = bazel query 'deps(target)'
    graph.AddEdges(target -> deps)
```

**From .d files:**

```
for each .d file in bazel-out:
    parse target.o: dep1.h dep2.cc ...
    for each dep:
        graph.AddEdge(target_file -> dep_file)
```

### 3. Cycle Detection

```
sccs = tarjan(graph)  // Strongly connected components
for each scc:
    if len(scc) > 1:
        report as cycle
```

### 4. Package Complexity Scoring

```
score = 0
score += file_count * 0.1
score += internal_dependency_count * 2
score += fan_out_count * 0.5  // How many other packages depend on this
score += diversity_metric  // How unrelated are the files?

if score > threshold:
    suggest split
```

### 5. Split Suggestions

```
// Cluster files by co-inclusion pattern
clusters = kmeans(files, by=cooccurrence_in_d_files)

for each cluster:
    suggest new package with cluster files
    show which consumers would use this cluster
```

## Implementation Steps

### Phase 1: Core Analysis (Priority)

1. Bazel query integration
2. .d file parsing
3. Dependency graph construction
4. Cycle detection
5. Coverage analysis

### Phase 2: Advanced Analysis

1. Package complexity scoring
2. Split/merge suggestions
3. Inter-package vs intra-package dependency analysis

### Phase 3: Web UI

1. Basic HTTP server
2. REST API for analysis results
3. WebSocket for real-time updates
4. Cytoscape.js visualization
5. Interactive features (filtering, search, details)

### Phase 4: Continuous Monitoring

1. File watcher integration
2. Incremental re-analysis
3. Change detection and notification

## Getting Started with Claude Code

1. **Initialize the project:**

   ```bash
   mkdir bazel_analyzer
   cd bazel_analyzer
   go mod init github.com/yourusername/bazel_analyzer
   ```

2. **Install dependencies:**

   ```bash
   go get gonum.org/v1/gonum/graph
   go get github.com/fsnotify/fsnotify  # For file watching
   go get github.com/gorilla/websocket  # For WebSocket
   ```

3. **Start with basic structure:**

   - Create the project structure above
   - Implement Bazel query first
   - Test against test_bazel_workspace

4. **Iterative development:**
   - Build incrementally
   - Test each component against test workspace
   - Add features one at a time

## Testing

```bash
# Run analyzer against test workspace
go run cmd/analyzer/main.go \
    --workspace=/home/claude/test_bazel_workspace \
    --port=8080

# Expected outputs:
# - Detect orphaned.cc as uncovered
# - Identify util as high complexity
# - Suggest splitting util into 4 packages
# - Show cross-package dependencies from .d files
# - Flag plugin/main duplicate dependencies
```

## Next Steps

Once basic functionality works:

1. Add caching for analysis results
2. Implement incremental analysis (only re-analyze changed files)
3. Add export features (JSON, GraphML, DOT)
4. Create CLI commands for specific queries
5. Add configuration file support (.bazel_analyzer.yaml)
6. Implement suggestions for BUILD file changes
7. Generate migration plans

## Resources

- Bazel query documentation: https://bazel.build/query/language
- gonum/graph: https://pkg.go.dev/gonum.org/v1/gonum/graph
- Cytoscape.js: https://js.cytoscape.org/
- Tarjan's algorithm:
  https://en.wikipedia.org/wiki/Tarjan%27s_strongly_connected_components_algorithm
