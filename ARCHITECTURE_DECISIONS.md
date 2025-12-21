# Architecture Decisions & Context

This document captures the key architectural discussions and decisions made during planning.

## Problem Context

### Current State
- C++ codebase with Bazel build system
- Many .so files (plugins + shared libraries)
- Originally a static binary, recently split into .so files for "versioning"
- But all plugins use "latest" - no actual versioning happening
- Result: complexity without benefits
- Symbol export issues, duplicate initialization, unclear API boundaries

### Key Issues
1. **Monolithic packages:** `util` and `core` contain unrelated functionality
2. **Uncovered files:** Some .cc/.h files not in any BUILD target
3. **Dependency mess:** Unclear what depends on what
4. **Plugin complexity:** .so files causing more problems than they solve
5. **Duplicate dependencies:** Both main binary and plugins link the same code

### Migration Goals
1. **Platform as single binary:** Static linking, one versioned unit
2. **Clients in separate repos:** Depend on platform version
3. **Clean API boundaries:** Clear separation between platform and clients
4. **Refactor util/core:** Split into cohesive packages

## Technology Choices

### Why Go over Python?
- **Performance:** Native compilation, faster execution
- **Static binary:** Single executable, easy deployment
- **Concurrency:** Built-in goroutines for file watching + web server
- **GC:** Automatic memory management without performance penalty
- **Bazel ecosystem:** Good Go support (Gazelle)

### Why not C++?
- Would need external libraries for web server, JSON, etc.
- Overkill for a build tool
- Slower iteration during development

### Graph Libraries
- **gonum/graph:** Has everything we need
  - Tarjan's algorithm for SCC
  - Cycle detection
  - Topological sort
  - Good performance for our use case

## IPC Architecture Discussion

### Original Plugin Design
- Plugins as .so files for "flexibility"
- Reality: all use latest, no real versioning
- Creates complexity: symbol export, initialization, dependencies

### Considered: Scripting Engine with .so Platform
- Load platform as .so from script
- Rejected: too complex, same problems

### Final Direction: IPC-based
**Platform:** Standalone binary
**Clients:** Connect via IPC (gRPC or ZeroMQ)

**Benefits:**
- Language agnostic (any client language)
- Process isolation (crashes don't propagate)
- Clear protocol versioning
- Easier debugging
- No more .so complexity

### IPC Latency Considerations

**gRPC over Unix Domain Sockets:**
- ~10-50 microseconds per call
- Fine for 95% of use cases (UI, events, commands)

**When NOT to use IPC:**
- Sub-millisecond requirements → keep in-process (C++ plugins)
- Tight loops with high call frequency
- Performance-critical callbacks

**Architecture principle:**
> Platform never blocks waiting for scripts

**Patterns:**
- **Async/pub-sub:** For most communication
- **Request/reply:** For queries (when needed)
- **Shared memory:** For bulk data (if needed)

### Plugin Architecture (Final)

**C++ Plugins:** (when performance matters)
- Loaded via dlopen in dev (hot reload)
- Statically linked in production
- For performance-critical extensions
- Direct function calls (<1μs)

**Script/IPC Clients:** (for flexibility)
- Connected via gRPC/ZeroMQ
- For orchestration, tools, automation
- Async only - can't block platform
- ~50μs latency acceptable

## Dependency Analysis Strategy

### Two Sources of Dependency Information

**1. Bazel Targets (Intended Dependencies):**
```
bazel query 'deps(//package:target)'
```
- Shows declared/intended dependencies
- Package-level granularity
- What the build system knows about

**2. .d Files (Actual Dependencies):**
```
target.o: source.cc header1.h header2.h
```
- Shows actual #include dependencies  
- File-level granularity
- What the code actually uses

### Key Insight: Compare Both!

**Mismatches reveal problems:**
- File includes something but BUILD doesn't declare it → missing dep
- File-level deps show finer coupling than Bazel targets
- Can see inter-package dependencies at file granularity
- Helps identify what should be split/merged

### Inter-Package Dependencies from .d Files

**Example:**
```
util/math.cc includes util/strings.h
core/engine.cc includes util/strings.h
core/engine.cc includes util/time.h
```

**This shows:**
- `util` has internal coupling (math → strings)
- `core` needs specific parts of `util`, not all of it
- These parts could be separate packages

### Split vs Merge Analysis

**Split candidates:**
- Large packages with diverse functionality
- Files clustered by usage pattern (from .d files)
- Example: util → util_strings, util_io, util_time, util_math

**Merge candidates:**
- Small packages always used together
- Circular dependencies at package level
- Tight coupling shown in .d files

**Header-only vs Implementation:**
- Track which packages include only .h files
- Vs which need both .h and .cc (shouldn't happen, but does)
- Identify implementation detail leakage

## Analysis Types

### 1. Coverage Analysis
- Find all .cc/.h files in workspace
- Check if each is in a BUILD file
- Report uncovered files

### 2. Cycle Detection
- Build dependency graph from Bazel
- Run Tarjan's algorithm
- Find strongly connected components with >1 node
- Do same for file-level graph from .d files

### 3. Package Complexity
**Metrics:**
- Number of files
- Number of internal dependencies
- Fan-out (how many packages depend on it)
- Functional diversity (do files relate to each other?)

**Scoring heuristic:**
```
score = files * 0.1 
      + internal_deps * 2 
      + fan_out * 0.5
      + diversity_metric
```

### 4. Split Suggestions
**Clustering approach:**
- Group files by co-inclusion patterns from .d files
- If util/strings.h and util/time.h never included together → separate
- If always included together → could merge

**Output:**
- Suggested new package structure
- Which consumers would use each package
- Migration impact (which BUILD files change)

### 5. .so Analysis
**Detect:**
- cc_binary with linkshared=True
- Dependencies shared between main and plugin
- Circular dependencies between .so files

**Suggest:**
- Static linking for internal code
- Keep only system libs as .so
- Simplified architecture

## Web UI Design

### Technology
- **Backend:** Go HTTP server + WebSocket
- **Frontend:** Vanilla JS + Cytoscape.js (no React needed)

### Features
- Interactive dependency graph (zoom, pan, click)
- Color coding by health (red=cycles, orange=complex, green=good)
- Sidebar with analysis results
- Real-time updates via WebSocket when files change
- Search and filter capabilities
- Detail panels for packages/files

### Layout
```
+------------------+------------------------+
|                  |                        |
|   File Tree      |   Dependency Graph     |
|   & Issues       |   (Cytoscape.js)       |
|                  |                        |
|  - Uncovered     |   Interactive:         |
|  - Cycles        |   - Zoom/Pan           |
|  - Complex       |   - Click for details  |
|                  |   - Highlight paths    |
+------------------+------------------------+
|                                           |
|   Analysis Details / Suggestions          |
|                                           |
+-------------------------------------------+
```

## Implementation Priority

**Phase 1:** Basic analysis (gets us 80% of value)
1. Bazel query integration
2. .d file parsing  
3. Dependency graph construction
4. Cycle detection
5. Coverage analysis

**Phase 2:** Advanced analysis
1. Package complexity scoring
2. Split/merge suggestions
3. Comparison of Bazel vs .d file dependencies

**Phase 3:** User interface
1. Web server + REST API
2. Cytoscape.js visualization
3. Real-time updates

**Phase 4:** Continuous monitoring
1. File watching
2. Incremental re-analysis
3. Change notifications

## Key Design Principles

1. **Correctness over speed:** Get the analysis right first
2. **Incremental:** Build piece by piece, test each part
3. **Observable:** Good logging, clear error messages
4. **Actionable:** Don't just report problems, suggest solutions
5. **Visual:** Graph visualization makes complex deps understandable
6. **Real-time:** Continuous monitoring catches regressions early

## Open Questions for Implementation

1. How to handle generated code in bazel-bin/?
2. Should we parse C++ AST for perfect accuracy, or is .d file good enough?
3. Caching strategy for large codebases?
4. How to handle Bazel external dependencies?
5. Configuration file format for custom rules?

These can be addressed during implementation in Claude Code.
