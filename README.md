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

### Prerequisites

- Go 1.21 or later
- Bazel 7.0 or later
- A Bazel workspace with C++ targets

### Build from Source

```bash
go build -o deps-analyzer cmd/deps-analyzer/main.go
```

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

- **Navigation Panel**: Browse binaries and targets
- **Dependency Graph**: Interactive visualization with Cytoscape.js
  - Click nodes to focus on specific targets
  - Hover for tooltips with dependency details
  - Color-coded by target type (binary, library, shared library)
  - Warnings for overlapping dependencies
- **Real-time Status**: SSE-based updates during analysis
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
  model/              Graph data model
  pubsub/             SSE event publishing
  symbols/            Symbol dependency analysis (nm)
  watcher/            File system watching and debouncing
  web/                HTTP server and static files
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

## License

[Your License Here]

## Contributing

Contributions welcome! Please open an issue or pull request.
