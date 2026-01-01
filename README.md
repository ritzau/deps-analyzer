# Bazel C++ Dependency Analyzer

A tool for analyzing and visualizing C++ dependencies in Bazel projects, with support for live updates and file watching.

## Features

- **Interactive Dependency Graph**: Visualize target, package, and file-level dependencies
- **Binary Analysis**: Analyze binaries, shared libraries, and their dependencies
- **Symbol Dependency Tracking**: Discover actual symbol usage via nm analysis
- **Compile Dependency Detection**: Parse .d files for header dependencies
- **Uncovered File Detection**: Find source files not included in any target
- **Live Updates**: Automatically re-analyze when BUILD files or artifacts change
- **Web Interface**: Modern, responsive UI with real-time status updates

## Installation

### Quick Install (Recommended)

```bash
go install github.com/ritzau/deps-analyzer/cmd/deps-analyzer@latest
```

This installs the `deps-analyzer` binary to `$GOPATH/bin` (usually `~/go/bin`). Make sure `$GOPATH/bin` is in your `$PATH`.

### Build from Source

```bash
git clone https://github.com/ritzau/deps-analyzer.git
cd deps-analyzer
go build -o deps-analyzer cmd/deps-analyzer/main.go
```

### Prerequisites

- Go 1.21 or later
- Bazel 7.0 or later
- A Bazel workspace with C++ targets

## Usage

### Basic Usage

Start the web server and open the UI:

```bash
./deps-analyzer --web --workspace=/path/to/bazel/workspace
```

The tool will:
1. Query Bazel for all targets and dependencies
2. Parse .d files for compile dependencies
3. Analyze symbols from object files
4. Generate interactive dependency graphs
5. Open your browser to http://localhost:8080

### Live Updates

Enable automatic re-analysis when files change:

```bash
./deps-analyzer --web --watch --workspace=/path/to/bazel/workspace
```

The analyzer will monitor:
- **BUILD and BUILD.bazel files** → triggers full re-analysis
- **bazel-out/**/*.d files** (compile dependencies) → triggers partial re-analysis
- **bazel-out/**/*.o files** (symbol info) → triggers symbol dependency re-analysis

Changes are debounced (1.5s quiet period, 10s max wait) to avoid excessive re-analysis.

**Note**: You must run `bazel build` manually. The tool only detects the resulting artifact changes, it does not trigger builds.

The UI displays "👁️ Watching for changes..." when active, and shows notifications when re-analysis is triggered.

### Command-Line Options

- `--web`: Start web server mode (required)
- `--watch`: Enable file watching for live updates
- `--workspace PATH`: Path to Bazel workspace (default: current directory)
- `--port PORT`: HTTP server port (default: 8080)

### Logging

The tool uses structured logging with a compact, readable console format:

```
[INFO]  21:54:51 starting web server | url=http://localhost:8080
[INFO]  21:54:51 request started | req=eb419103 method=GET path=/
[INFO]  21:54:51 request completed | req=eb419103 status=200 duration=3ms
[INFO]  21:54:55 analysis complete | targets=10 dependencies=25 packages=8
```

Features:
- **Compact timestamps**: HH:MM:SS instead of full RFC3339
- **Shortened request IDs**: First 8 characters for readability
- **Structured key-value pairs**: Easy to grep and parse
- **Request tracking**: Each HTTP request gets a unique ID for end-to-end tracing
- **Log levels**: DEBUG (internal details), INFO (operations), WARN (issues), ERROR (bugs)

## How It Works

### Analysis Phases

1. **Bazel Query**: Queries `bazel query` to discover all targets and their declared dependencies
2. **Compile Dependencies**: Parses `.d` files (compiler dependency output) to find actual header includes
3. **Symbol Dependencies**: Uses `nm` to analyze object files and discover which symbols are used between targets
4. **Binary Derivation**: Analyzes binaries and shared libraries to find dynamic dependencies, data dependencies, and system libraries
5. **Uncovered Files**: Walks the workspace to find source files not included in any target

### Incremental Re-analysis

When `--watch` is enabled, the tool intelligently determines which analysis phases to re-run:

| Changed Files | Phases Re-run | Why |
|--------------|---------------|-----|
| BUILD files | Full analysis | Target definitions may have changed |
| .d files only | Compile deps → Symbols → Binaries | Header dependencies changed |
| .o files only | Symbols → Binaries | Symbol information changed |

This minimizes re-analysis time for common changes.

## Web Interface

The web UI provides:

- **Navigation Panel**: Browse binaries and targets, with tabbed configuration
- **Lens-based Visualization**: Advanced graph filtering and focus system
  - **Default Lens**: Controls the base graph view (hierarchy level, filters, edge types)
  - **Detail Lens**: Automatically applied to selected nodes and their neighbors
  - **Distance-based Rules**: Show/hide nodes based on distance from selection
  - **Configurable Collapse**: Package-level, target-level, or file-level detail
- **Dependency Graph**: Interactive visualization with Cytoscape.js
  - Click nodes to select/focus on specific targets
  - Ctrl+Click to toggle multiple selections
  - Hover for tooltips with dependency details
  - Color-coded by target type (binary, library, shared library, system library)
  - Edge types: Static deps, dynamic deps, compile deps (#include), data deps
  - Warnings for overlapping dependencies
- **Real-time Status**: SSE-based updates during analysis with progress checklist
- **Live Updates**: Automatic refresh when files change (with `--watch`)

## Development

### Project Structure

```
cmd/deps-analyzer/    Main entry point
pkg/
  analysis/           Analysis orchestration and runner
  bazel/              Bazel query interface
  binaries/           Binary and shared library analysis
  deps/               Compile dependency parser (.d files)
  lens/               Lens-based graph filtering and rendering
  logging/            Structured logging with compact console output
  model/              Graph data model
  pubsub/             SSE event publishing for real-time updates
  symbols/            Symbol dependency analysis (nm)
  watcher/            File system watching and debouncing
  web/                HTTP server and static files
    static/           Frontend HTML, CSS, and JavaScript
```

### Running Tests

```bash
go test ./...
```

### Example Workspace

The `example/` directory contains a small Bazel C++ project for testing:

```bash
cd example
bazel build //...
cd ..
./deps-analyzer --web --watch --workspace=./example
```

See [example/README.md](example/README.md) for details on the test cases and intentional problems.

## Contributing

Contributions welcome! Please open an issue or pull request.

For development documentation, architecture decisions, and implementation details, see [DEVELOPMENT.md](DEVELOPMENT.md).

## License

[Your License Here]
