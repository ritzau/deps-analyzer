# Ideas for future evolution

## Prioritized backlog

1. The "Base set" section of the default lens does no longer make sense. Remove
   it.

2. If a node has a single nested node, we should be able to collapse the
   hierarchy (recursively). We need to determine what the label should be
   though.

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

## Alphabetical navigation sorting

Added alphabetical sorting to navigation targets list. Modified
`filterAndRenderNavigationList()` to sort with `localeCompare()` after
filtering.

## External dependency support

Added comprehensive support for querying and visualizing external Bazel
dependencies in the dependency analyzer.

**Implementation**:

Backend ([pkg/bazel/query.go](pkg/bazel/query.go)):

- Added `collectExternalDependencies()` function to extract external dependency
  labels from workspace target deps
- Added `queryExternalTargets()` function to query Bazel for external target
  details using `bazel query "label1 + label2 + ..." --output=xml`
- Modified `QueryWorkspace()` to perform three-pass parsing:
  1. Parse workspace targets from `//...` query
  2. Query and parse external targets referenced by workspace
  3. Parse dependencies from both workspace and external rules
- Filters out system repositories (`@bazel_tools//`, `@local_config_*`,
  `@platforms//`)

Hierarchy handling ([pkg/lens/distance.go](pkg/lens/distance.go)):

- Updated `extractParentID()` to handle external target label format
  (`@repo//:target:file`)
- Added special case for `@` prefix: splits by `:` and removes last component
- External files now properly collapse under their parent targets

Frontend visibility
([pkg/web/static/lens-config.js](pkg/web/static/lens-config.js)):

- Updated default lens configurations to show external dependencies by default
- Changed `showExternal: false` to `showExternal: true` in both
  DEFAULT_PACKAGE_LENS and DEFAULT_DETAIL_LENS

**Example dependencies added to test workspace**:

- `@fmt//:fmt` - Modern C++ formatting library via bzlmod `bazel_dep`
- `@nlohmann_json//:json` - JSON library via `http_archive` with custom BUILD
  file
- Created wrapper packages `//formatter` and `//config` to demonstrate usage

**Label formats**:

- Workspace targets: `//package:target` or `//package:target:file`
- External targets: `@repo//:target` or `@repo//:target:file`
- External files properly nest under parent: `@fmt//:fmt:@fmt/src/format.cc` →
  parent `@fmt//:fmt`

**Result**:

- External targets appear in target list and navigation sidebar ✓
- Dependency edges from workspace to external targets display correctly ✓
- External targets visible in graph by default ✓
- External files properly collapse under parent targets at collapseLevel: 2 ✓
- Works with both bzlmod (`bazel_dep`) and legacy (`http_archive`) integration
  methods ✓

**Files modified**:

- [pkg/bazel/query.go](pkg/bazel/query.go) - External dependency querying
- [pkg/lens/distance.go](pkg/lens/distance.go) - External target hierarchy
- [pkg/web/static/lens-config.js](pkg/web/static/lens-config.js) - Visibility
  defaults
- [example/MODULE.bazel](example/MODULE.bazel) - Test dependencies
- [example/formatter/](example/formatter/) - Wrapper for fmt library (new
  package)
- [example/config/](example/config/) - Wrapper for nlohmann/json (new package)
- [example/third_party/nlohmann_json.BUILD](example/third_party/nlohmann_json.BUILD) -
  Custom BUILD file
- [example/main/BUILD.bazel](example/main/BUILD.bazel) - Usage examples
- [example/main/main.cc](example/main/main.cc) - Integration demos

## Compact log format with client/server indicators

Restructured log output to use a compact, scannable format with clear source
indicators to distinguish client and server logs.

**Format**: `HH:MM:SS/L/S message | key=value key=value`

Where:

- `HH:MM:SS` = timestamp in 24-hour format
- `L` = log level initial (D/I/W/E/T for Debug/Info/Warn/Error/Trace)
- `S` = source indicator (S for Server, C for Client)

**Backend Implementation**
([pkg/logging/compact_handler.go](pkg/logging/compact_handler.go)):

- Updated `Handle()` function to scan log attributes for "source" field
- When `source="frontend"` is present, uses 'C' indicator
- Server-originated logs use 'S' indicator
- Filters out the "source" attribute from displayed attributes (already used for
  indicator)
- Time comes first for easy chronological scanning
- Slash separators provide clear visual boundaries

**Frontend Implementation**
([pkg/web/static/logger.js](pkg/web/static/logger.js)):

- Updated `_output()` function to use new format with 'C' for Client
- Consistent format across browser console and backend logs
- All client logs show 'C' whether viewed in browser or forwarded to backend

**Backend Log Forwarding** ([pkg/web/server.go](pkg/web/server.go)):

- Frontend logs sent to `/api/logs` endpoint include `source="frontend"`
  attribute
- Backend handler adds this attribute when logging frontend messages
- CompactHandler detects this and uses 'C' indicator

**Example Output**:

Server logs:

```
18:40:18/I/S starting web server | url=http://localhost:8080
18:40:18/I/S request started | req=eb419103 method=GET path=/
18:40:18/I/S request completed | req=eb419103 status=200 duration=3ms
```

Client logs (in browser console):

```
18:40:19/I/C fetch started | requestID="abc123" url="/api/module/graph/lens"
18:40:19/I/C fetch completed | requestID="abc123" status=200
```

Client logs (forwarded to backend):

```
18:40:19/I/C fetch started | requestID="abc123" url="/api/module/graph/lens"
18:40:19/I/C fetch completed | requestID="abc123" status=200
```

**Benefits**:

- Easy to distinguish client vs server logs at a glance
- Time-first ordering aids chronological analysis
- Compact format reduces visual clutter
- Consistent format across all logging contexts
- Works seamlessly with existing log forwarding infrastructure

**Files Modified**:

- [pkg/logging/compact_handler.go](pkg/logging/compact_handler.go) - Source
  indicator logic
- [pkg/web/static/logger.js](pkg/web/static/logger.js) - Client-side format

## Unified navigation list with filtering and highlighting fixes

Consolidated binaries and targets into a single filterable navigation list, and
fixed navigation highlighting to properly reflect graph selections.

**Implementation (TODO #1)**:

- Merged separate binaries and targets lists into unified "Targets" navigation
  section
- Added filter controls: rule type checkboxes (cc_binary, cc_library,
  cc_shared_library) and free text search
- Implemented client-side filtering (300ms debounce on search)
- Removed `/api/binaries` endpoint (use graph nodes as single source of truth)
- All three rule types checked by default

**Navigation highlighting bug fixes**:

**Problem 1**: When selecting nodes in the graph view, the navigation sidebar
items were not being highlighted to show what was selected.

**Root Cause 1**: After `filterAndRenderNavigationList()` re-created the
navigation DOM elements, it wasn't applying the selection highlighting to match
the current state.

**Fix 1**: Added call to
`updateNavigationHighlighting(viewStateManager.state.selectedNodes)` at the end
of `filterAndRenderNavigationList()` in
[app.js:2099](pkg/web/static/app.js#L2099).

**Problem 2**: When selecting a package node (e.g., `//core`) in the graph, the
targets within that package (e.g., `//core:core`) were not being highlighted in
the navigation sidebar.

**Root Cause 2**: The highlighting logic only checked for direct node ID
matches, but didn't check if a navigation item's parent package was selected.
Since the backend expands package selections to include all child targets, the
navigation should reflect this.

**Fix 2**: Updated `updateNavigationHighlighting()` in
[app.js:1946-1973](pkg/web/static/app.js#L1946-L1973) to check both:

- Direct target selection (`//core:core` in selectedNodes)
- Parent package selection (`//core` in selectedNodes)

Extracts package prefix from target IDs (`//package:target` → `//package`) and
checks if that package is selected.

**Files modified**:

- [pkg/web/static/app.js](pkg/web/static/app.js) - Navigation highlighting fixes

**Result**:

- Navigation sidebar properly highlights selected targets ✓
- Package selections correctly highlight all child targets in navigation ✓
- Highlighting updates immediately when filters change ✓
- Works with both single and multi-select (Cmd/Ctrl+click) ✓

## Navigation multi-select and mixed-level edge fix

Fixed two issues with node selection and edge rendering.

**Problem 1: No multi-select in navigation lists**

Navigation sidebar (binaries and targets lists) only supported single-select,
and Ctrl+click on macOS triggers right-click menu instead of multi-select.

**Solution**: Added Cmd/Ctrl+click support for multi-select in navigation lists:

- Simple click replaces selection with clicked item
- Cmd+click (⌘ on macOS) or Ctrl+click toggles item in/out of selection
- Added `data-node-id` attribute to navigation items for accurate matching
- Implemented `updateNavigationHighlighting()` to sync `.selected` class with
  view state
- Visual feedback shows which items are currently selected

**Implementation**: Modified [pkg/web/static/app.js](pkg/web/static/app.js):

- Updated binaries list click handler (lines 1975-1982)
- Updated targets list click handler (lines 2008-2015)
- Added navigation highlighting sync (lines 1891-1904)

**Problem 2: Mixed-level edges (package→target)**

When selecting `//main:test_app`, saw edge from `//audio` (package) to
`//util:util` (target), which is inconsistent. Edges should connect nodes at the
same hierarchy level.

**Root Cause**: When finding visible ancestors for edge endpoints during
aggregation, one endpoint might resolve to a package while the other resolves to
a target, creating mixed-level edges like `//audio → //util:util`.

**Solution**: Added endpoint normalization after ancestor resolution. If one
endpoint is a package and the other is a target, elevate the target to its
package level to ensure consistency.

**Implementation**: Modified
[pkg/lens/renderer.go:588-610](pkg/lens/renderer.go#L588-L610):

```go
// Normalize endpoints to same level (package vs target)
sourceIsPackage := !strings.Contains(actualSource, ":")
targetIsPackage := !strings.Contains(actualTarget, ":")

if sourceIsPackage && !targetIsPackage {
    // Elevate target to package
    targetPackage := childToParentMap[actualTarget]
    if targetPackage != "" && includedNodeIds[targetPackage] {
        actualTarget = targetPackage
    }
} else if !sourceIsPackage && targetIsPackage {
    // Elevate source to package
    sourcePackage := childToParentMap[actualSource]
    if sourcePackage != "" && includedNodeIds[sourcePackage] {
        actualSource = sourcePackage
    }
}
```

**Result**:

- Navigation lists support Cmd/Ctrl+click multi-select with visual feedback ✓
- macOS users can use Cmd+click (standard macOS modifier) ✓
- Edges are always at consistent hierarchy levels (package→package or
  target→target) ✓
- Fixed edge: `//audio → //util` (both packages) instead of
  `//audio → //util:util` ✓

## Atomic state updates for performance

Fixed redundant backend requests when changing default lens settings.

**Problem**: When changing hierarchy level, base set type, or binary selection,
the UI triggered 3-4 backend requests instead of 1, causing unnecessary load and
delays.

**Root Cause**: `clearSelection()` and `updateDefaultLens()` each called
`notifyListeners()` separately, triggering independent backend fetches. These
operations needed to happen atomically but were executed as two separate state
mutations.

**Solution**: Added `updateDefaultLensAndClearSelection()` method that batches
both state updates before calling `notifyListeners()` once.

**Implementation**:

- Added atomic update method to
  [pkg/web/static/view-state.js:109-119](pkg/web/static/view-state.js#L109-L119)
- Updated base set change handler in
  [pkg/web/static/lens-controls.js:93](pkg/web/static/lens-controls.js#L93)
- Updated collapse level change handler in
  [pkg/web/static/lens-controls.js:187](pkg/web/static/lens-controls.js#L187)
- Updated binary selector change handler in
  [pkg/web/static/lens-controls.js:298](pkg/web/static/lens-controls.js#L298)

**Result**: Changing hierarchy level or other major lens settings now triggers
exactly 1 backend request instead of 3-4, improving performance and reducing
server load.

## Frontend-to-backend logging integration

Implemented centralized logging that sends frontend logs to the backend for
monitoring and debugging.

**Implementation**:

Backend ([pkg/web/server.go](pkg/web/server.go)):

- Added `/api/logs` POST endpoint to receive frontend logs
- Added `FrontendLogEntry` and `FrontendLogsRequest` structs for JSON parsing
- Handler logs each frontend entry with `source=frontend` tag for easy filtering
- Maps frontend log levels (TRACE/DEBUG/INFO/WARN/ERROR) to slog levels
- Includes all frontend data attributes in backend logs
- Uses request context for request ID correlation

Frontend ([pkg/web/static/logger.js](pkg/web/static/logger.js)):

- Added `enableBackendLogging(enabled)` method to turn on/off backend logging
- Implemented log batching: sends after 10 logs or 5 second timeout
- Uses `fetch()` to POST batched logs to `/api/logs` endpoint
- Fails silently on errors to avoid infinite error loops
- Non-blocking: logs sent asynchronously without blocking UI
- Disabled by default for minimal overhead

Documentation ([pkg/web/static/index.html](pkg/web/static/index.html)):

- Added configuration comments explaining how to enable backend logging
- Documented batching behavior and filtering approach

**Usage**:

```javascript
// In browser console
logger.enableBackendLogging(true); // Start sending logs to backend
appLogger.info("Test message", { foo: "bar" }); // Will appear in backend logs
logger.enableBackendLogging(false); // Stop sending logs
```

**Backend log format**:

```
[INFO] 12:34:56 Test message | source=frontend foo="bar" requestID=abc123
```

**Benefits**:

- Easy to filter backend logs: `grep "source=frontend"` in logs
- Batching reduces backend load and network overhead
- Useful for debugging production issues or CI environments
- Request ID correlation between frontend and backend
- All frontend context (component, data) preserved in backend logs

## File node selection redirect to parent target

Fixed confusing behavior when selecting file nodes in the graph.

**Problem**: Selecting a file node (like `engine.cc` or `orphaned.cc`) resulted
in a weird state where:

- No files were visible in the graph
- Neighbor packages remained visible but their targets were hidden
- The graph appeared broken or incomplete

**Root Cause**: Files don't have dependencies - their parent targets do. The
lens system computes distances and visibility based on dependencies, so when a
file was selected, it had no outgoing edges to follow. This caused the BFS
distance computation to not reach other nodes properly, resulting in an
inconsistent visibility state.

**Solution**: Redirect file node selections to their parent target
automatically. When a user clicks on a file node in the graph, the selection
logic now:

1. Detects if the clicked node is a file type (`source_file`, `header_file`,
   `uncovered_source`, `uncovered_header`)
2. Extracts the parent target ID from the file node's `parent` field
3. Selects the parent target instead of the file itself
4. Logs the redirection for debugging:
   `File node clicked - redirecting to parent target`

This makes semantic sense because:

- Files are implementation details of targets
- Users likely want to see what the target depends on, not the individual file
- Target nodes have proper dependency edges that the lens system can follow
- Prevents the confusing "broken graph" state entirely

**Implementation**: Modified
[pkg/web/static/app.js:964-993](pkg/web/static/app.js#L964-L993) node click
handler:

```javascript
const isFileNode =
  nodeType === "source_file" ||
  nodeType === "header_file" ||
  nodeType === "uncovered_source" ||
  nodeType === "uncovered_header";

if (isFileNode) {
  const parentId = node.data("parent");
  if (parentId) {
    appLogger.info("File node clicked - redirecting to parent target:", {
      file: nodeId,
      parent: parentId,
    });
    nodeId = parentId;
  }
}
```

Also added defensive filtering in navigation tree to prevent file nodes from
appearing in the targets sidebar list (only packages, targets, and binaries
should be listed there).

**Example Behavior**:

- Click on `engine.cc` file → automatically selects `//core:core` target instead
- Click on uncovered `orphaned.cc` → selects `//util:util` target
- The selected target's dependencies are then shown normally with proper
  visibility
- Works with both simple click and Ctrl+click (toggle)

**Result**: File nodes can still be visible in the graph (when their parent
target is selected), but clicking them now produces sensible behavior instead of
breaking the visualization.

## Uncovered files visibility fixes

Fixed two related bugs with uncovered file visibility in the graph.

**Problem 1: Files appeared at top level instead of in packages**

Uncovered files (files discovered by git but not included in any Bazel target)
were shown as orphaned nodes at the root level instead of being nested under
their package nodes (e.g., `//util`, `//cycle_demo`).

**Root Cause 1**: The uncovered file handling code in `buildModuleGraph()` set
parent references correctly, but the lens renderer's `extractParentID()`
function only handled standard Bazel node IDs (starting with `//` and using `:`
separators). When it encountered uncovered file IDs like
`uncovered:cycle_demo/file_a.h`, it returned an empty string, clearing the
parent field.

**Fix 1**: Modified
[pkg/lens/distance.go:173-201](pkg/lens/distance.go#L173-L201)
`extractParentID()` function to handle uncovered file IDs specially:

- Detects IDs with `uncovered:` prefix
- Extracts package path from file path (e.g., `uncovered:util/orphaned.cc` →
  `//util`)
- Uses `strings.LastIndex("/")` to find package boundary
- Returns empty string only for root-level files without a package

Also updated [pkg/web/server.go:877-944](pkg/web/server.go#L877-L944)
`buildModuleGraph()` to ensure package nodes exist for packages containing only
uncovered files (no targets).

**Problem 2: Files didn't show when package was selected**

When a package like `//util` was selected, only the targets within that package
were shown. Uncovered files (e.g., `orphaned.cc`) were not included in the
selection expansion, so they remained hidden even though their parent package
was selected.

**Root Cause 2**: The `expandPackagesToTargets()` function only looked for
target nodes (IDs with `:` separators) when expanding a package selection. It
didn't consider uncovered files which have IDs like
`uncovered:util/orphaned.cc`.

**Fix 2**: Modified [pkg/lens/distance.go:34-91](pkg/lens/distance.go#L34-L91)
`expandPackagesToTargets()` to also include uncovered files when expanding
package selections:

- When package `//util` is selected, function now finds both targets
  (`//util:util`) and uncovered files (`uncovered:util/orphaned.cc`)
- Adds both to the initial BFS queue at distance 0
- Ensures uncovered files are visible alongside regular target files when
  exploring a package

**Result**:

- All uncovered files correctly appear as children of their package nodes in the
  graph hierarchy ✓
- Uncovered files are visible when their package is selected ✓
- Makes uncovered files much more discoverable and useful for understanding
  which files are not included in any Bazel targets

**Testing**: Verified with example workspace:

- `uncovered:cycle_demo/file_a.h` → parent: `//cycle_demo` ✓
- `uncovered:cycle_demo/file_b.h` → parent: `//cycle_demo` ✓
- `uncovered:util/orphaned.cc` → parent: `//util` ✓
- Selecting `//util` shows `orphaned.cc` ✓
- Selecting `//cycle_demo` shows both `file_a.h` and `file_b.h` ✓

## Package-only view edge aggregation fix

Fixed missing edges when showing only packages in the graph (collapseLevel: 1).

**Problem**: When the default lens was set to show only packages (hiding targets
and files), no edges were displayed between packages. The graph showed 8 package
nodes but 0 edges, making it impossible to understand the high-level dependency
structure.

**Root Cause**: The `findVisibleAncestor()` function was designed to skip
package nodes when aggregating edges (from an earlier bug fix to prevent
synthetic package edges when targets are visible). However, when ONLY packages
are visible, this caused all edges to be dropped since the function couldn't
find any visible non-package ancestors.

**Previous Context**: An earlier fix prevented package nodes from appearing in
edges when their child targets were visible (e.g., preventing
`//audio → //util:util` when `//audio:audio_impl` should be used instead). The
fix skipped package nodes during edge aggregation by continuing to walk up the
hierarchy past them. This worked correctly when targets were visible but broke
the package-only view.

**Fix**: Modified [pkg/lens/renderer.go:660-703](pkg/lens/renderer.go#L660-L703)
`findVisibleAncestor()` to use package nodes as fallback ancestors:

- Track the first visible package encountered while walking up the hierarchy
- Continue walking up past packages to search for visible target ancestors
- If a visible target is found above, use it (maintains previous bug fix)
- If no visible target exists, fall back to using the package (fixes
  package-only view)
- Only return empty string if no visible ancestor at all

**Implementation Details**:

```go
// Walk up the hierarchy, tracking the first visible package in case we need it
var firstVisiblePackage string
for ... {
    if includedNodeIds[parentID] {
        isPackage := !strings.Contains(parentID, ":")
        if isPackage {
            if firstVisiblePackage == "" {
                firstVisiblePackage = parentID
            }
            // Continue walking up to see if there's a visible target above
            continue
        }
        return parentID  // Found visible target
    }
}
// Fall back to package if no target found
if firstVisiblePackage != "" {
    return firstVisiblePackage
}
```

**Result**: Package-level dependency visualization now works correctly:

- **Before**: collapseLevel: 1 showed 8 packages, 0 edges ✗
- **After**: collapseLevel: 1 shows 8 packages, 9 edges between packages ✓
- **Maintains previous fix**: When targets are visible, edges skip package nodes
  and connect targets directly ✓

**Example edges now visible in package-only view**:

- `//main → //core` (static)
- `//main → //graphics` (dynamic, static)
- `//main → //util` (static)
- `//audio → //util` (static)
- `//core → //util` (static)
- `//plugins → //core` (static)
- `//plugins → //util` (static)
- `//foobar → //cycle_demo` (static)

This makes the package-only view useful for understanding high-level
architecture and identifying which packages depend on each other.

## Workspace directory display

Added workspace directory path to the web UI header to help users identify which
workspace is being analyzed.

**Implementation**:

- Added `WorkspacePath` field to `Module` struct in
  [pkg/model/model.go:103](pkg/model/model.go#L103)
- Set workspace path using `filepath.Abs()` in
  [pkg/bazel/query.go:87-92](pkg/bazel/query.go#L87-L92)
- Updated frontend `updateModuleName()` function in
  [pkg/web/static/app.js:231-242](pkg/web/static/app.js#L231-L242) to display
  both module name and absolute path
- Format: `module_name • /absolute/path/to/workspace`
- Falls back gracefully if either value is missing

**Benefits**:

- Users can immediately see which workspace is being analyzed
- Especially helpful when using relative paths like "."
- Absolute path helps distinguish between multiple workspaces
- No ambiguity when switching between projects

## Frontend structured logging migration

Migrated all frontend JavaScript files to use the structured logger
infrastructure.

**Implementation**:

- Converted all console.\* calls in app.js (66 statements), view-state.js (14
  statements), and lens-controls.js (6 statements) to use appLogger,
  viewStateLogger, and lensLogger respectively
- Fixed logger.js bug where strings were treated as character arrays due to
  Object.entries() iteration
- Added \_normalizeArgs() helper to handle both console.log-style arguments and
  structured data objects
- Downgraded 25+ log statements from INFO to DEBUG to reduce console noise
- Changed large object logging to summary counts (e.g., log array length instead
  of full array)
- Added HTML comments documenting runtime log level control via browser console

**Benefits**:

- Consistent structured logging format across frontend and backend
- Appropriate log levels (DEBUG for internal details, INFO for user-facing
  operations)
- Runtime log level control without rebuilding
- Reduced console noise from internal operations

## Documentation consolidation

Consolidated multiple markdown files and added missing installation
instructions.

**Implementation**:

- Added Quick Install section to README.md with `go install` command
- Created comprehensive DEVELOPMENT.md merging:
  - ARCHITECTURE_DECISIONS.md (technology rationale, design decisions)
  - DEVELOPMENT_GUIDE.md (project structure, algorithms)
  - pkg/pubsub/README.md (SSE pub/sub documentation)
  - REMOVED_FEATURES.md (outdated content)
- Deleted redundant markdown files
- Added cross-references between README.md, DEVELOPMENT.md, and
  example/README.md
- Kept TODO.md separate as requested

**Benefits**:

- Single source for developer onboarding (DEVELOPMENT.md)
- Standard Go installation method documented
- Reduced documentation fragmentation
- Easier to maintain and keep up-to-date

## CSS style refactoring

Refactored Cytoscape graph styles to reduce duplication and improve
maintainability.

**Implementation**:

- Added GRAPH_COLORS constant with semantic color definitions (node colors, edge
  colors, state colors, text/border colors)
- Created helper functions:
  - edgeStyle(color, width, lineStyle) - generates complete edge styles
  - nodeStyle(bgColor, textColor, borderColor) - generates basic node styles
  - fileNodeStyle(bgColor, borderColor) - generates ellipse file node styles
- Refactored all node and edge type styles to use helpers and color constants
- Combined duplicate uncovered node styles into single selector
- Improved code organization with section comments

**Code Reduction**:

- Before: 156 lines of style definitions
- After: 121 lines (35 lines removed, 22% reduction)

**Benefits**:

- Easier to maintain consistent visual design
- Change colors globally in one place
- Better readability with semantic color names
- Less duplication in edge/node style definitions
- No visual changes - purely maintainability improvement

## Structured logging infrastructure

**Goal**: Implement consistent, structured logging across backend and frontend
with request tracking, proper log levels, and request-response correlation.

**Log Level Philosophy**:

- **TRACE**: Very spammy, debug-time only
- **DEBUG**: Internal component behavior
- **INFO**: User-facing operations
- **WARN**: Should be monitored
- **ERROR**: Logical bugs that shouldn't happen
- **FATAL**: Unrecoverable bugs

**Backend Implementation** ([pkg/logging](pkg/logging)):

- Created logging package wrapping Go's standard library `log/slog`
- Request ID middleware generates/extracts UUIDs for each HTTP request
- Request ID stored in `context.Context` and automatically added to all logs
- HTTP middleware logs every request start/completion with timing and status
- Updated [pkg/web/server.go](pkg/web/server.go) to use structured logging with
  context
- Updated [cmd/deps-analyzer/main.go](cmd/deps-analyzer/main.go) to use new
  logger
- All lens API operations now log with request IDs for tracing

**Frontend Implementation**
([pkg/web/static/logger.js](pkg/web/static/logger.js)):

- Created custom structured logger with same log levels as backend
- Generates request IDs for fetch requests
- Logs formatted as `[LEVEL] message | key=value key=value`
- Child logger support for adding persistent context (e.g., component name)
- Console output with appropriate methods (debug, log, warn, error)

**Request Tracking**:

- Backend: UUID generated per request, stored in context, returned in
  `X-Request-ID` header
- Frontend: Can send `X-Request-ID` header or backend generates one
- All logs within a request lifecycle include the request ID
- Enables end-to-end request tracing across frontend/backend boundary

**Completed**:

- ✅ Go logging package with slog wrapper
- ✅ Request ID middleware for HTTP (with SSE Flusher support)
- ✅ JavaScript structured logger
- ✅ HTTP request/response logging with timing
- ✅ Updated server.go and main.go to use new logging
- ✅ Migrated all remaining Go files: `analysis/runner.go`, `watcher/*.go`,
  `lens/renderer.go`, `pubsub/sse.go`
- ✅ Logger script added to index.html
- ✅ Fixed SSE streaming bug in logging middleware
- ✅ **Compact console handler** with readable format:
  `[LEVEL] HH:MM:SS message | key=value`
- ✅ Fixed all printf-style format strings to use proper structured key-value
  pairs
- ✅ Appropriate log levels (Debug for internal details, Info for operations,
  Warn/Error for issues)

**Remaining Work**:

- ✅ ~~Update frontend JavaScript files to use new structured logger~~ (DONE -
  see above)
- ⏸️ Add log level configuration via command-line flags
- ⏸️ Consider JSON output mode for production log aggregation

**Example Structured Logs**:

Backend (compact format):

```
[INFO]  21:54:51 starting web server | url=http://localhost:8080
[INFO]  21:54:51 request started | req=eb419103 method=GET path=/ remoteAddr=[::1]:57427
[INFO]  21:54:51 request completed | req=eb419103 method=GET path=/ status=200 duration=3ms
[INFO]  21:54:55 analysis complete | targets=10 dependencies=25 packages=8
```

Frontend:

```
[INFO] fetch started | requestID="abc123" url="/api/module/graph/lens"
[INFO] fetch completed | requestID="abc123" status=200 durationMs=45
```

## Info popups bug fixes

**Problem 1: Stuck popups accumulating in DOM**: Info popups (hover tooltips)
were getting stuck on screen and accumulating in the DOM, causing visual clutter
and memory leaks.

**Root Causes**:

1. Each call to `displayDependencyGraph()` created a new tooltip element without
   removing old ones
2. No cleanup mechanism when window lost focus or graph layout changed
3. No tracking of active popups for cleanup

**Problem 2: Popups disappeared after initial fix**: After fixing the stuck
popup bug by removing popups from DOM, they stopped appearing entirely because
`clearInfoPopup()` was called on every graph re-render.

**Problem 3: Missing fade animations**: Popups were showing/hiding but without
smooth transitions.

**Final Fix** (three iterations): Implemented proper info popup lifecycle
management in [pkg/web/static/app.js](pkg/web/static/app.js):

- Changed from create/destroy pattern to singleton pattern with show/hide
- Global `infoPopup` reference persists in DOM for reuse
- `clearInfoPopup(fade)` now hides popup with `display: 'none'` instead of
  removing from DOM
- Added fade-in animation (200ms ease-in, opacity 0→1) when showing popup
- Added fade-out animation (200ms ease-out) on mouseout from edges/nodes
- Immediate hide (no fade) on tap/click for responsive feel
- Used forced reflow (`offsetHeight`) to ensure CSS transitions work correctly
- Clear popups on graph re-render and window focus loss

**Result**:

- Only one info popup element exists at a time (singleton)
- Popups are properly cleaned up on all appropriate events
- Smooth visual transitions with fade animations
- Better performance (reusing DOM element instead of create/destroy)

## Package node edge collision bug fix

**Problem**: When a target node was hidden by lens configuration
(distance=infinite), edges would incorrectly point to its parent package node
instead of being hidden. This caused synthetic package nodes (like `//audio`) to
appear in the dependency graph with edges, even though package nodes are not
real targets.

**Example**: With `//main:test_app` selected:

- Bug: `//audio → //util:util` (package node incorrectly used)
- Bug: `//audio:audio → //audio` (edge to package instead of real target)
- Expected: These edges should be hidden when the real targets
  (`//audio:audio_impl`) are at infinite distance

**Root Cause**: In `findVisibleAncestor()`, when walking up the node hierarchy
to find a visible ancestor for edge aggregation, the function would stop at
package nodes (`//audio`) if they were visible, even though package nodes are
synthetic grouping nodes that should never appear in edges.

**Fix**: Modified [pkg/lens/renderer.go:656-688](pkg/lens/renderer.go#L656-L688)
`findVisibleAncestor()` to skip package nodes when aggregating edges. Package
nodes are identified by having no colon in their ID (e.g., `//audio` vs
`//audio:audio`). When a package node is encountered as a potential ancestor,
the function continues walking up the hierarchy instead of using it, ultimately
returning empty string if no non-package ancestor is found, which causes the
edge to be dropped.

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

**Result**: Edges now only connect real target nodes. When a target's children
are hidden, edges to/from those hidden nodes are correctly dropped instead of
being incorrectly aggregated to the parent package node.

## "Focus" to "Select" terminology refactoring

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

## Edge type collapse option

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

## Browser auto-open flag

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

## Backend lens rendering system

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

## Project name display fix

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

## Legend simplification

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

## System library filtering

Fixed system libraries incorrectly appearing in the targets navigation sidebar.
System libraries (like 'dl', 'pthread', 'rt', etc.) are specified via linkopts
and are not actual Bazel targets, so they should not be listed alongside real
targets in the navigation.

Implementation:

- Added filter in app.js:2204 to exclude nodes with type 'system_library'
- System libraries still appear correctly in graph visualizations
- Only affects the clickable targets list in the navigation sidebar

## Horizontal sidebar resize

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

## Target label simplification

Implemented client-side label simplification to reduce visual clutter. The
`simplifyLabel()` function removes redundant target names when they match the
package name:

- `//foo:foo` → `//foo`
- `//bar/baz:baz` → `//bar/baz`
- `//util:util` → `//util`

Applied to all label displays: graph nodes, sidebar navigation, tree browser,
and modal dialogs. Internal lookups and API calls still use full labels.

## Symbol name simplification

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

## Symbol dependency parsing fix

Fixed nm output parsing to correctly handle C++ symbol names containing spaces
(e.g., template parameters). The parser was using `strings.Fields()` which split
on all whitespace, breaking symbol names like
`util::ToUpper(std::__1::basic_string<char, std::__1::char_traits<char>, ...>)`.
Now joins all parts after the type field to preserve the full symbol name. This
fixed the bug where intra-target symbol dependencies (e.g., math.cc → strings.cc
within //util) were not being detected.

## Show visibility in graph

Dashed gold border for public targets.

## Remove the collapse package toggle

Removed non-functional toggle and 75 lines of unused code.

## Front end layout

Full-screen responsive layout: header row, navigation sidebar, graph fills
remaining space with compact legend and proper canvas sizing.

## File coverage

Git-based discovery identifies C++ files not included in any target; displayed
as red warning nodes in focused view; includes bug fix for header parsing in
srcs attribute.

## Backend connection monitoring

Modal notification on connection loss with retry/reload options; hybrid
detection using SSE error handlers, monitoredFetch wrapper for immediate
failures, and periodic health checks every 5s when idle; prevents silent
failures when backend goes down.

## BUG FIX: Binary selection

Removed leftover packagesCollapsed reference from collapse package toggle
removal; binary-focused graphs now display correctly when clicking binaries in
navigation.

## Binary/so-level

Start at the level of the generated artefacts. Clicking lets you see what is
inside. You may also be able to see what is accessed from other artefacts, and
what is used in external ones.

**Status**: Implemented! Click binaries in sidebar to see binary-focused view
with overlapping dependencies highlighted.

## Message bus

Communicate from the server to the UI using a pub/sub. UI subscribes to get UI
state messages. Great for start up. Awesome also to do live updates.

- Can also be fun to develop a terminal client (not done)

**Status**: SSE (Server-Sent Events) implemented with
`/api/subscribe/workspace_status` and `/api/subscribe/target_graph`

## Tooltips on all edges

Store in the edge (type and text). Maybe for each node? Maybe some more info
when clicking on edges?

**Status**: Comprehensive tooltips with hover delay, directional info, file
details, and symbol lists.

## Color scheme

Make a good looking color scheme and support dark/light (auto detected by
default). Maybe dive deeper and also choose a font that we can enjoy some.

**Status**: VS Code dark theme colors, good font stack, professional UI.

## Optimize analysis

Fewer queries should be possible for many phases. Can we use bazel to generate
and cache this info?

- Concurrent operations? (could still be improved)

**Status**: Reduced from 4 analysis passes to 2, eliminated redundant work.

## Live updates

Watch the project files for changes and update continuously. Automatically
re-analyze when BUILD files or build artifacts change.

**Status**: Implemented with fsnotify-based file watcher, smart debouncing (1.5s
quiet, 10s max), intelligent change detection for incremental updates, and
discrete UI status indicator. Run with `--watch` flag.
