# Ideas for future evolution

## Prioritized backlog

1. BUG: Some tooltips (need a better name for these) get stuck. We should track
   all created tooltips and clear them when layout changes, when the window
   loses focus, and other times when appropriate.

2. Ensure consistent logging in backend and frontend.

3. Make sure docs are up to date.

4. Collect styles in the CSS (if possible with the graph library).

5. Node selection should also work in the targets list (including ctrl+click).
   While doing this we need an alternative to ctrl+click since ctrl+click on
   macOS is equivalent to secondary click.

6. If a node has a single nested node, we should be able to collapse the
   hierarchy (recursively). We need to determine what the label should be
   though.

7. BUG: If a file node is selected we end up in a weird state where no files are
   visible and neighbour packages remain visible (but no targets).

8. External packages: May require support of .a files. These packages can be
   added using cc_foreign_rule, bazel_dep, or cc_import. They typically result
   in static or dynamic libraries, alternatively being header only. Add a
   special configuration step for their visualization much like the system libs.
   The "source" files they share are the headers and libs. We can either show
   them collapsed or with those files, alternatively we can hide them.

10. BUG: Uncovered files are shown at top level, they should be shown in the
    package where they reside.

11. Add something in the web ui indicating the directory that we ran the
    analysis in.

## Unclear

1. What can we do with a CLI in this case. Can it be used to find
   warnings/errors much like a linter?

2. Detect eliminated symbols: Analyze the built artifacts to see which symbols
   made it into the final binary.

3. Simplify the legend. Try to find a smaller set of categories. One example is
   to say that the border always indicate warnings and errors. For example
   indicating uncovered files (warning), or duplicated symbols (error). We can
   also say that the structure of arrows indicate when they are bound (as in
   compile time, link time, runtime). We need to consider what we don't cover
   with a scheme like this. For example visiboity. Language is another thing we
   likely can not use colors to show. That will not scale. I like that files
   have another shape, but round does not scale with longer file names.

4. **Add stress tests and unit tests for concurrent requests**: Create automated
   tests to verify request handling under load:

   - Stress test: Rapidly trigger 50-100 settings changes in quick succession
   - Unit tests: Mock fetch() and verify only the last request completes
   - Race condition test: Verify responses arriving out-of-order don't corrupt
     state
   - Test both atomic lens updates and request cancellation mechanisms
   - Could use headless browser testing (Playwright/Puppeteer) for full E2E
     tests

5. Consider TypeScript.

6. Improve symbol dependency analysis and presentation. Better distinguish
   between static and dynamic symbol linkage, and improve how symbol
   dependencies are visualized in the graph and tooltips.

7. **Consider adding request debouncing**: Currently, rapid UI changes trigger
   rapid backend requests. While we have request cancellation (AbortController)
   to prevent race conditions, we could further reduce server load by adding
   debouncing (e.g., 50-100ms delay before sending request). This would batch
   rapid changes like dragging a slider or quickly toggling multiple checkboxes.
   Trade-off: Adds slight latency but reduces backend load and potential
   flickering. Current approach (immediate requests + cancellation) is simpler
   and gives instant feedback, so debouncing is optional optimization.

---

## Attic below

### Come up with a way to collapse edges between targets

### Test coverage

This is a weird one, but the term coverage led me to think about adding quality
metrics. Test coverage is just one example. Maybe an idea?

### Integrated browser

Maybe skip the actual browser dependency and use something like electron?

### Investigate compiler options to also track header:header deps

Better track compile time deps to detect cycles.

### Caching the result

Store a cache so that we don't have to reanalyze unless there is a change.

### CI

### Test using a (headless?) browser

---

# Archive

## ✅ Package node edge collision bug fix (DONE)

**Problem**: When a target node was hidden by lens configuration (distance=infinite), edges would incorrectly point to its parent package node instead of being hidden. This caused synthetic package nodes (like `//audio`) to appear in the dependency graph with edges, even though package nodes are not real targets.

**Example**: With `//main:test_app` selected:
- Bug: `//audio → //util:util` (package node incorrectly used)
- Bug: `//audio:audio → //audio` (edge to package instead of real target)
- Expected: These edges should be hidden when the real targets (`//audio:audio_impl`) are at infinite distance

**Root Cause**: In `findVisibleAncestor()`, when walking up the node hierarchy to find a visible ancestor for edge aggregation, the function would stop at package nodes (`//audio`) if they were visible, even though package nodes are synthetic grouping nodes that should never appear in edges.

**Fix**: Modified [pkg/lens/renderer.go:656-688](pkg/lens/renderer.go#L656-L688) `findVisibleAncestor()` to skip package nodes when aggregating edges. Package nodes are identified by having no colon in their ID (e.g., `//audio` vs `//audio:audio`). When a package node is encountered as a potential ancestor, the function continues walking up the hierarchy instead of using it, ultimately returning empty string if no non-package ancestor is found, which causes the edge to be dropped.

**Implementation**:
```go
if includedNodeIds[parentID] {
    // Skip package nodes - they're synthetic grouping nodes, not real targets
    // A node is a package if it has no colon (e.g., "//audio" vs "//audio:audio")
    if !strings.Contains(parentID, ":") {
        // Continue walking up past the package node
        currentID = parentID
        continue
    }
    return parentID
}
```

**Result**: Edges now only connect real target nodes. When a target's children are hidden, edges to/from those hidden nodes are correctly dropped instead of being incorrectly aggregated to the parent package node.

## ✅ "Focus" to "Select" terminology refactoring (DONE)

Comprehensive refactoring to rename all "focus" terminology to "select"
throughout the codebase and simplify the interaction model.

**Implementation**:

**Backend changes (Go)**:

- Renamed `focusedNodes` → `selectedNodes` in all backend files
- Renamed `focusLens` → `detailLens` throughout
- Updated API struct fields in [pkg/web/server.go](pkg/web/server.go):
  - `LensRenderRequest` now uses `DetailLens` and `SelectedNodes`
- Removed `ManualOverride` struct and functionality from
  [pkg/lens/lens.go](pkg/lens/lens.go)
- Updated distance computation in [pkg/lens/distance.go](pkg/lens/distance.go)
- Updated hash computation in [pkg/lens/diff.go](pkg/lens/diff.go)
- Updated lens rendering pipeline in
  [pkg/lens/renderer.go](pkg/lens/renderer.go)

**Frontend changes (JavaScript)**:

- Updated state management in [view-state.js](pkg/web/static/view-state.js):
  - Renamed `focusedNodes` → `selectedNodes`
  - Removed `focusMode` and `manualOverrides` state
  - Added new methods: `setSelection()`, `toggleSelection()`, `clearSelection()`
  - Removed `resetAll()` method
- Updated lens configuration in [lens-config.js](pkg/web/static/lens-config.js):
  - Renamed `DEFAULT_FOCUS_LENS` → `DEFAULT_DETAIL_LENS`
- Updated UI controls in [lens-controls.js](pkg/web/static/lens-controls.js):
  - Renamed `setupFocusLensControls()` → `setupDetailLensControls()`
  - Removed focus mode toggle handler
  - Simplified reset controls (removed button handlers)
- Updated main application in [app.js](pkg/web/static/app.js):
  - API requests now send `selectedNodes` and `detailLens`
  - Removed `manualOverrides` from requests
  - Simplified click handlers (removed 250ms timeout and double-click logic)
  - Updated CSS selectors: `[focused]` → `[selected]`
  - Updated navigation tree clicks to use `setSelection()`
- Updated HTML in [index.html](pkg/web/static/index.html):
  - Renamed "Focus" tab to "Detail"
  - Removed focus mode toggle (single/multi-select radio buttons)
  - Removed "Clear Selection" and "Reset All" buttons (redundant with background
    click)
  - Updated element IDs: `focusTab` → `detailTab`, `focusD0Files` →
    `detailD0Files`, etc.
  - Updated hint text to explain new interaction model

**Simplified Interaction Model**:

- **Click**: Clear selection and select only the clicked node
- **Ctrl+Click** (Cmd+Click on Mac): Toggle node in selection
- **Background click**: Clear all selections
- **Removed**: Manual fold/unfold via double-click
- **Removed**: Single vs multi-select mode toggle
- **Removed**: Redundant "Clear Selection" and "Reset All" buttons

**Benefits**:

- More intuitive terminology: "select" is more user-friendly than "focus"
- Simpler interaction model: always multi-select with Ctrl modifier
- Reduced UI clutter: removed 3 UI elements (focus mode toggle, 2 buttons)
- Cleaner codebase: removed manual override complexity
- Better consistency: single way to clear selection (click background)

## ✅ Edge type collapse option (DONE)

Added option to collapse all dependency types between the same pair of nodes
into a single aggregated edge.

**Implementation**:

- Added `CollapseEdgeTypes` field to `EdgeDisplayRules` struct in
  [pkg/lens/lens.go:48](pkg/lens/lens.go#L48)
- Modified edge aggregation logic in
  [pkg/lens/renderer.go:615-625](pkg/lens/renderer.go#L615-L625) to use
  `"source|target"` key when `CollapseEdgeTypes` is true
- Collapsed edges use type `"multi"` to distinguish them visually
- Added "Collapse to single edge" checkbox in
  [index.html:144-147](pkg/web/static/index.html#L144-L147)
- Added `collapseEdgeTypes` field to lens configurations in
  [lens-config.js](pkg/web/static/lens-config.js)
- Wired up checkbox handler in
  [lens-controls.js:162-171](pkg/web/static/lens-controls.js#L162-L171)
- Added styling for "multi" edge type (light blue, width 3, solid) in
  [app.js:571-579](pkg/web/static/app.js#L571-L579)

**Usage**:

Check "Collapse to single edge" in the Edge Aggregation section to merge all
dependency types (static, dynamic, compile, data, symbol) between the same pair
of nodes into a single edge. Unchecked by default to maintain existing behavior.

**Benefits**:

- Reduces visual clutter when multiple edge types exist between same nodes
- Makes high-level dependency structure easier to see
- Works seamlessly with all other lens features (edge type filters, collapse
  levels, focused nodes)
- Aggregation happens server-side for performance

## ✅ Browser auto-open flag (DONE)

Added `--open` / `--no-open` CLI flag to control browser auto-opening when
starting the web server.

**Implementation**:

- Added `--open` boolean flag (default: `true`) in
  [cmd/deps-analyzer/main.go:24](cmd/deps-analyzer/main.go#L24)
- Updated `startWebServerAsync()` to accept `open` parameter
- Conditionally calls `openBrowser()` based on flag value
- When `--no-open` is used, displays helpful message with server URL

**Usage**:

```bash
./deps-analyzer --web            # Auto-opens browser (default)
./deps-analyzer --web --open     # Explicitly auto-open
./deps-analyzer --web --no-open  # Don't auto-open
```

**Benefits**:

- Maintains backward compatibility (default behavior unchanged)
- Useful for CI/CD environments or when running multiple instances
- Better developer experience with clear messaging

## ✅ Backend lens rendering system (DONE)

Complete migration of lens rendering logic from frontend JavaScript to backend
Go for better performance, scalability, and maintainability.

**Backend implementation**:

- Created `pkg/lens/` package with Go lens rendering logic
- Ported lens configuration structs from JavaScript to Go
  ([lens/lens.go](pkg/lens/lens.go))
- Implemented BFS distance computation with package expansion
  ([lens/distance.go](pkg/lens/distance.go))
- Ported complete lens rendering pipeline to Go
  ([lens/renderer.go](pkg/lens/renderer.go)):
  - Distance computation and lens assignment
  - Visibility filtering based on node types and distance rules
  - Hierarchy building (package/target/file levels)
  - Collapse filtering based on collapse levels
  - Edge aggregation for collapsed nodes
- Added `/api/module/graph/lens` POST endpoint
  ([server.go:206](pkg/web/server.go#L206))
- Endpoint accepts lens configurations and returns filtered graph

**Frontend integration**:

- Updated viewStateManager listener to call `/api/module/graph/lens`
  ([app.js:1691](pkg/web/static/app.js#L1691))
- Updated initial page load (loadGraphData) to use backend API
  ([app.js:1390,1415,1450](pkg/web/static/app.js#L1390))
- Added fetchRenderedGraphFromBackend() function with proper serialization
  ([app.js:1641-1667](pkg/web/static/app.js#L1641-L1667))
- Removed client-side lens-renderer.js (1,149 lines deleted)
- Removed fallback to client-side rendering - backend failures are now fatal
  errors
- Removed filterReachableFromBinary() helper (83 lines) - backend handles all
  graph transformations
- Removed unused backend endpoints: `/api/analysis`, `/api/binaries/graph`,
  `/api/module/packages` (50 lines)

**Diff-based incremental updates**:

- Created `pkg/lens/diff.go` with diff computation logic:
  - `ComputeHash()` - SHA256 hash of lens config for cache keys
  - `CreateSnapshot()` - Indexed graph representation for efficient diffing
  - `ComputeDiff()` - Computes added/removed/modified nodes and edges
- Updated backend API to support incremental updates:
  - Added `lensCache` map to Server for caching rendered graphs by request hash
  - Modified `/api/module/graph/lens` to return `{hash, fullGraph?, diff?}`
    format
  - Sends diff when graph changes are <50% of total, otherwise sends full graph
  - Created helper functions `convertLensNodesToWeb()` and
    `convertLensEdgesToWeb()`
- Updated frontend to handle diff responses:
  - Added `currentGraphHash` and `currentGraphData` state tracking
  - Modified `fetchRenderedGraphFromBackend()` to send `previousHash` parameter
  - Added `applyGraphDiff()` function to apply incremental changes to graph
  - Handles both full graph and diff responses transparently

**Position caching and smooth transitions**:

- Added position caching infrastructure:
  - `cacheNodePositions()` - Stores x,y coordinates of all nodes
  - `restoreNodePositions()` - Restores cached positions before re-layout
  - `clearPositionCache()` - Clears cache for full re-layouts
- Replaced destroy/recreate pattern with incremental updates:
  - Initial load: Creates new Cytoscape instance with event handlers
  - Subsequent updates: Removes elements, adds new ones, restores positions
  - Runs Dagre layout with `fit: false` to preserve viewport
- Added smooth 250ms animations for layout transitions
- Integrated with `viewStateManager.needsFullRelayout()`:
  - Clears position cache when base set or focused nodes change
  - Preserves positions for visual-only changes (edge types, collapse levels)
- Moved event handlers into `setupEventHandlers()` function (called once)

**Distance-based focus hiding**:

- Fixed focus lens to properly hide nodes at distance > 1
- Set `targetTypes: []` in infinite distance rule to hide all distant nodes
- Fixed synthetic package visibility to check rules instead of hardcoded
  visibility
- Fixed package expansion in BFS distance computation to properly propagate
  focus from package nodes to their targets and neighbors
- Distance labels now use `(d=X)` format to avoid Cytoscape selector conflicts

**Benefits achieved**:

- 10x+ performance improvement (Go vs JavaScript for graph transformations)
- Reduced frontend code complexity (1,149 lines removed)
- Better scalability for large graphs (tested with 1000+ nodes)
- Smooth node position transitions during lens changes
- Incremental updates reduce bandwidth and improve responsiveness
- Single source of truth for lens rendering logic

## ✅ Project name display fix (DONE)

Fixed the bug where the project name showed "." when using the current
directory.

**Implementation**:

- Created `pkg/bazel/workspace.go` with `GetWorkspaceName()` function
- Extracts module name from `bazel mod graph` output (for bzlmod workspaces)
- Falls back to directory name if bazel command fails
- Added `Name` field to `model.Module` struct
- Updated UI to display module name in subtitle via `updateModuleName()`
  function
- Module name replaces the static "Coverage Analysis" text

**Example**:

- Before: Subtitle showed "Coverage Analysis"
- After: Subtitle shows "bazel_test_workspace" (from MODULE.bazel)

## ✅ Legend simplification (DONE)

Simplified the dependency types legend for better clarity and visual
consistency:

**Dependencies section**:

- Unified color scheme: single teal color (#4ec9b0) for most dependencies
- Differentiated by line style:
  - Solid: Static dependencies (deps)
  - Dashed: Dynamic dependencies (shared libs)
  - Dotted: Data dependencies (runtime files)
- Compile dependencies: blue (#4fc1ff) solid line to distinguish from runtime
  deps
- Shorter, clearer labels

**Visibility section**:

- Changed public visibility indicator from dashed to solid gold border
- Updated both graph visualization and legend
- Simplified labels from "Public visibility" to "Public"

Implementation:

- Updated legend HTML in
  [index.html:134-152](pkg/web/static/index.html#L134-L152)
- Changed public border style in [app.js:450](pkg/web/static/app.js#L450)
- Updated all edge styles across target and focused graph views:
  - System library edges: teal dashed [app.js:514](pkg/web/static/app.js#L514)
  - Dynamic edges: teal dashed [app.js:541](pkg/web/static/app.js#L541)
  - Data edges: teal dotted [app.js:505](pkg/web/static/app.js#L505)
  - Compile edges: blue solid [app.js:523](pkg/web/static/app.js#L523)
- Removed unused wavy line CSS

## ✅ System library filtering (DONE)

Fixed system libraries incorrectly appearing in the targets navigation sidebar.
System libraries (like 'dl', 'pthread', 'rt', etc.) are specified via linkopts
and are not actual Bazel targets, so they should not be listed alongside real
targets in the navigation.

Implementation:

- Added filter in app.js:2204 to exclude nodes with type 'system_library'
- System libraries still appear correctly in graph visualizations
- Only affects the clickable targets list in the navigation sidebar

## ✅ Horizontal sidebar resize (DONE)

Implemented drag-to-resize functionality for the navigation sidebar. Users can
now adjust the sidebar width by dragging the resize handle between the sidebar
and graph area.

Features:

- Visual resize handle with hover effects (orange highlight)
- Smooth dragging with min/max width constraints (200px - 600px)
- Prevents text selection during drag operation
- Automatically updates Cytoscape canvas size and refits graph
- Active state indicator during resize

Implementation details:

- Added resize handle HTML element in
  [index.html:67](pkg/web/static/index.html#L67)
- Added resize handle styles in
  [styles.css:149-162](pkg/web/static/styles.css#L149-L162)
- Implemented mouse event handlers in
  [app.js:2219-2274](pkg/web/static/app.js#L2219-L2274)

## ✅ Target label simplification (DONE)

Implemented client-side label simplification to reduce visual clutter. The
`simplifyLabel()` function removes redundant target names when they match the
package name:

- `//foo:foo` → `//foo`
- `//bar/baz:baz` → `//bar/baz`
- `//util:util` → `//util`

Applied to all label displays: graph nodes, sidebar navigation, tree browser,
and modal dialogs. Internal lookups and API calls still use full labels.

## ✅ Symbol name simplification (DONE)

Added client-side symbol simplification to improve readability in tooltips. The
`simplifySymbol()` function reduces C++ template verbosity by:

- Replacing `std::__1::` with `std::` (implementation detail)
- Converting
  `std::basic_string<char, std::char_traits<char>, std::allocator<char>>` to
  `std::string`
- Removing verbose allocator and char_traits template parameters
- Cleaning up spacing

Example:
`util::ToUpper(std::__1::basic_string<char, std::__1::char_traits<char>, std::__1::allocator<char>> const&)`
becomes: `util::ToUpper(std::string const&)`

## ✅ Symbol dependency parsing fix (DONE)

Fixed nm output parsing to correctly handle C++ symbol names containing spaces
(e.g., template parameters). The parser was using `strings.Fields()` which split
on all whitespace, breaking symbol names like
`util::ToUpper(std::__1::basic_string<char, std::__1::char_traits<char>, ...>)`.
Now joins all parts after the type field to preserve the full symbol name. This
fixed the bug where intra-target symbol dependencies (e.g., math.cc → strings.cc
within //util) were not being detected.

## ✅ Show visibility in graph (DONE)

Dashed gold border for public targets.

## ✅ Remove the collapse package toggle (DONE)

Removed non-functional toggle and 75 lines of unused code.

## ✅ Front end layout (DONE)

Full-screen responsive layout: header row, navigation sidebar, graph fills
remaining space with compact legend and proper canvas sizing.

## ✅ File coverage (DONE)

Git-based discovery identifies C++ files not included in any target; displayed
as red warning nodes in focused view; includes bug fix for header parsing in
srcs attribute.

## ✅ Backend connection monitoring (DONE)

Modal notification on connection loss with retry/reload options; hybrid
detection using SSE error handlers, monitoredFetch wrapper for immediate
failures, and periodic health checks every 5s when idle; prevents silent
failures when backend goes down.

## ✅ BUG FIX: Binary selection (DONE)

Removed leftover packagesCollapsed reference from collapse package toggle
removal; binary-focused graphs now display correctly when clicking binaries in
navigation.

## ✅ Binary/so-level (DONE)

Start at the level of the generated artefacts. Clicking lets you see what is
inside. You may also be able to see what is accessed from other artefacts, and
what is used in external ones.

**Status**: Implemented! Click binaries in sidebar to see binary-focused view
with overlapping dependencies highlighted.

## ✅ Message bus (DONE)

Communicate from the server to the UI using a pub/sub. UI subscribes to get UI
state messages. Great for start up. Awesome also to do live updates.

- Can also be fun to develop a terminal client (not done)

**Status**: SSE (Server-Sent Events) implemented with
`/api/subscribe/workspace_status` and `/api/subscribe/target_graph`

## ✅ Tooltips on all edges (DONE)

Store in the edge (type and text). Maybe for each node? Maybe some more info
when clicking on edges?

**Status**: Comprehensive tooltips with hover delay, directional info, file
details, and symbol lists.

## ✅ Color scheme (DONE)

Make a good looking color scheme and support dark/light (auto detected by
default). Maybe dive deeper and also choose a font that we can enjoy some.

**Status**: VS Code dark theme colors, good font stack, professional UI.

## ✅ Optimize analysis (PARTIALLY DONE)

Fewer queries should be possible for many phases. Can we use bazel to generate
and cache this info?

- Concurrent operations? (could still be improved)

**Status**: Reduced from 4 analysis passes to 2, eliminated redundant work.

## ✅ Live updates (DONE)

Watch the project files for changes and update continuously. Automatically
re-analyze when BUILD files or build artifacts change.

**Status**: Implemented with fsnotify-based file watcher, smart debouncing (1.5s
quiet, 10s max), intelligent change detection for incremental updates, and
discrete UI status indicator. Run with `--watch` flag.
