# Ideas for future evolution

## Prioritized backlog

1. Improve symbol dependency analysis and presentation. Better distinguish
   between static and dynamic symbol linkage, and improve how symbol
   dependencies are visualized in the graph and tooltips.

2. Add collapsible external dependencies in focused view. Give users control
   over detail level:
   - Level 1: Hide external dependencies completely (only show files within
     focused target)
   - Level 2: Show external targets as collapsed nodes (hide individual files)
   - Level 3: Show all files in external targets (current behavior)

3. Detect eliminated symbols: Analyze the built artifacts to see which symbols
   made it into the final binary.

4. Ensure consistent logging in backend and frontend.

5. Make sure docs are up to date.

6. External packages: May require support of .a files.

7. Collect styles in the CSS (if possible with the graph library).

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

## ✅ Project name display fix (DONE)

Fixed the bug where the project name showed "." when using the current directory.

**Implementation**:
- Created `pkg/bazel/workspace.go` with `GetWorkspaceName()` function
- Extracts module name from `bazel mod graph` output (for bzlmod workspaces)
- Falls back to directory name if bazel command fails
- Added `Name` field to `model.Module` struct
- Updated UI to display module name in subtitle via `updateModuleName()` function
- Module name replaces the static "Coverage Analysis" text

**Example**:
- Before: Subtitle showed "Coverage Analysis"
- After: Subtitle shows "bazel_test_workspace" (from MODULE.bazel)

## ✅ Legend simplification (DONE)

Simplified the dependency types legend for better clarity and visual consistency:

**Dependencies section**:
- Unified color scheme: single teal color (#4ec9b0) for most dependencies
- Differentiated by line style:
  - Solid: Static dependencies (deps)
  - Dashed: Dynamic dependencies (shared libs)
  - Dotted: Data dependencies (runtime files)
- Compile dependencies: blue (#4fc1ff) solid line to distinguish from runtime deps
- Shorter, clearer labels

**Visibility section**:
- Changed public visibility indicator from dashed to solid gold border
- Updated both graph visualization and legend
- Simplified labels from "Public visibility" to "Public"

Implementation:
- Updated legend HTML in [index.html:134-152](pkg/web/static/index.html#L134-L152)
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
