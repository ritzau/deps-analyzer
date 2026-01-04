# Test Bazel C++ Workspace

This is a test workspace designed to demonstrate common problems in C++ Bazel projects that the analyzer should detect.

## Structure

```
test_bazel_workspace/
├── util/           - Utility package (intentionally monolithic)
├── core/           - Core application package
├── plugins/        - Plugin as shared library (.so)
├── main/           - Main application binary
└── bazel-out/      - Simulated build outputs with .d files
```

## Intentional Problems to Detect

### 1. Uncovered Files
- `util/orphaned.cc` - Not included in any BUILD target
- The analyzer should flag this as an uncovered file

### 2. Monolithic Packages
- `//util:util` - Single large library containing unrelated functionality:
  - String utilities (strings.h/cc)
  - File I/O (file_io.h/cc)
  - Time utilities (time.h/cc)
  - Math utilities (math.h/cc)

**Expected analysis:**
- Suggest splitting into separate targets: util_strings, util_io, util_time, util_math
- Show that different files are used by different consumers

### 3. Internal Package Dependencies
- Within util: math.cc depends on strings.h
- This creates coupling that prevents clean separation
- Analyzer should detect and highlight these internal dependencies

### 4. Cross-Package Dependencies
From .d files, we can see:
- core/engine.cc → util/strings.h, util/time.h
- core/state.cc → util/file_io.h
- plugins/renderer.cc → util/strings.h, core/engine.h

**Expected analysis:**
- Show which parts of util are actually used by which consumers
- Suggest that different consumers need different parts of util
- Map file-level dependencies vs Bazel target-level dependencies

### 5. Shared Library (.so) Issues
- `//plugins:renderer_plugin` is built as a shared library
- It depends on `//util:util` and `//core:core`
- Main binary also depends on the same libraries
- This can cause duplicate symbol issues

**Expected analysis:**
- Flag that both main and plugin depend on the same libraries
- Suggest either:
  - Static linking everything (simpler)
  - Main exports symbols with --export-dynamic (current setup)
  - Move shared dependencies to main only

### 6. Dependency Patterns
From the .d files, the analyzer can determine:

**util package file-level dependencies:**
- strings.cc → strings.h (standalone)
- file_io.cc → file_io.h (standalone)
- time.cc → time.h (standalone)
- math.cc → math.h, strings.h (depends on strings!)

**This shows util can be split into:**
- util_strings (no internal deps)
- util_io (no internal deps)
- util_time (no internal deps)
- util_math (depends on util_strings)

**core package dependencies:**
- engine.cc needs: util/strings.h, util/time.h
- state.cc needs: util/file_io.h

**This shows core doesn't need monolithic util, just specific parts**

### 7. Potential Circular Dependencies
Currently there are none, but the structure makes them likely:
- If util/strings.cc started using core/state.h for configuration
- We'd have core → util → core (bad!)

## Expected Analyzer Output

The analyzer should provide:

1. **Coverage Report:**
   - List of files not in any BUILD target
   - `util/orphaned.cc` should be flagged

2. **Package Complexity Analysis:**
   - util: HIGH complexity
     - 4 distinct functional areas
     - Should be split into 4 libraries
     - Internal dependency: math → strings

3. **Refactoring Suggestions:**
   ```
   Split //util:util into:
   - //util:strings (strings.h, strings.cc)
   - //util:file_io (file_io.h, file_io.cc)
   - //util:time (time.h, time.cc)
   - //util:math (math.h, math.cc, deps=[":strings"])
   ```

4. **Dependency Graph:**
   - Visualize package-level dependencies
   - Visualize file-level dependencies
   - Highlight cross-package includes

5. **Shared Library Analysis:**
   - Flag duplicate dependencies between main and plugin
   - Suggest migration to static linking

## Building the Project

```bash
# Build everything
bazel build //...

# Build just main
bazel build //main:test_app

# Build plugin
bazel build //plugins:renderer_plugin

# Run main
bazel run //main:test_app
```

## Using with the Analyzer

```bash
# Run analyzer on this workspace
../deps-analyzer --web --workspace=. --port=8080

# Or with live file watching
../deps-analyzer --web --watch --workspace=. --port=8080

# View results in browser (opens automatically)
# http://localhost:8080
```

The analyzer should detect all the issues listed above and provide actionable refactoring suggestions.

