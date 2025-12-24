# Ideas for future evolution

## ✅ Binary/so-level (DONE)

✅ Start at the level of the generated artefacts. Clicking let's you see what is
inside. You may also be able to see what is accessed from other artefacts, and
what is used in external ones.

**Status**: Implemented! Click binaries in sidebar to see binary-focused view with overlapping dependencies highlighted.

## ✅ Message bus (DONE)

✅ Communicate from the server to the UI using a pub/sub
✅ UI subscribes to get ui state messages
✅ Great for start up
✅ Awesome also to do live updates
- Can also be fun to develop a terminal client (not done)

**Status**: SSE (Server-Sent Events) implemented with `/api/subscribe/workspace_status` and `/api/subscribe/target_graph`

## ✅ Tooltips on all edges (DONE)

✅ Store in the edge (type and text)
✅ Maybe for each node?
✅ Maybe some more info when clicking on edges?

**Status**: Comprehensive tooltips with hover delay, directional info, file details, and symbol lists

## ✅ Color scheme (DONE)

✅ Make a good looking color scheme and support dark/light (auto detected by
default). Maybe dive deeper and also choose a font that we can enjoy some :)

**Status**: VS Code dark theme colors, good font stack, professional UI

## ✅ Optimize analysis (PARTIALLY DONE)

✅ Fewer queries should be possible for many phases
✅ Can we use bazel to generate and cache this info?
- Concurrent operations? (could still be improved)

**Status**: Reduced from 4 analysis passes to 2, eliminated redundant work

---

## Test coverage

This is a weird one, but the term coverage led me to think about adding quality
metrics. Test coverage is just one example. Maybe an idea?

## Integrated browser

Maybe skip the actual browser dependency and use something like electron?

## Live updates

Watch the project files for changes and update continously. At least make it
possible to trigger restarting the analysis.

## Investigate compiler options to also track header:header deps

Better track compile time deps to detect cycles.

## Caching the result

Store a cache so that we don't have to reanalyze unless there is a change.

## CI

## Test using a (headless?) browser

## Real decent logging

## External packages

- May require support of .a files

## Real so file test cases

## Detect eliminated symbols

- Analyze the built artifacts to see which symbols made it
- Use ldd to see which libraries are dynamically linked

## BUG: . as project name

- If we use the current dir, at least check to see what its name is

## Show visibility in graph

## ✅ BUG: Splash screen says that we load the workspace, should be module(?) (FIXED)

**Status**: Loading checklist now shows correct terminology
