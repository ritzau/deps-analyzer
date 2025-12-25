# Ideas for future evolution

## Prioritized backlog

1. BUG: Symbol dependencies within the same target (e.g., math.cc → strings.cc in //util)
   are not being displayed in the file-level graph. The compile dependency (math.cc → strings.h)
   shows up, but not the symbol dependency from the actual function call.

2. Simplify targets so that //foo:foo is presented as //foo

3. Detect eliminated symbols: Analyze the built artifacts to see which symbols
   made it into the final binary.

4. BUG: Project name shows "." when using current directory. Should detect and
   use the actual directory name.

5. Ensure consistent logging in backend and frontend.

6. Make sure docs are up to date.

7. External packages: May require support of .a files.

8. Collect styles in the CSS (if possible with the graph library).

---

## Attic below

## Test coverage

This is a weird one, but the term coverage led me to think about adding quality
metrics. Test coverage is just one example. Maybe an idea?

## Integrated browser

Maybe skip the actual browser dependency and use something like electron?

## Investigate compiler options to also track header:header deps

Better track compile time deps to detect cycles.

## Caching the result

Store a cache so that we don't have to reanalyze unless there is a change.

## CI

## Test using a (headless?) browser

---

# Archive

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

**Status**: Implemented with fsnotify-based file watcher, smart debouncing
(1.5s quiet, 10s max), intelligent change detection for incremental updates,
and discrete UI status indicator. Run with `--watch` flag.
