# Removed Features & TODOs

This document tracks features that were removed during the Module model migration and potential future enhancements.

## Removed Features (Not Planned for Restoration)

### 1. CLI Mode
**Status**: Removed
**What it did**: Printed file coverage report to console (files not in any Bazel target)
**Reason**: Focus shifted to web UI for better visualization and interactivity
**Note**: Can be restored from git history if needed for automation/CI use cases

### 2. File Coverage Analysis
**Status**: Removed
**What it did**: Found all source files and compared to Bazel targets to find uncovered files
**Reason**: Not part of core dependency analysis mission
**Note**: Could be added back as optional CLI/web feature if needed

## Current Feature Set

### Core Functionality
- **Module Model**: Complete dependency graph with 5 types (static, dynamic, data, compile, symbol)
- **Target-level Graph**: Interactive visualization of all targets and their dependencies
- **File-level Graph**: Click any target to see files and their compile/symbol dependencies
- **Package-level Dependencies**: Aggregated dependencies between packages
- **Binary Analysis**: Binary metadata, system libraries, overlapping dependencies detection
- **Overlapping Dependency Visualization**: Red highlighting for duplicate symbols
- **Issue Detection**: Warnings for duplicate linkage (static+dynamic to same target)
- **Visibility Visualization**: Public targets marked with dashed gold border
- **Responsive Web UI**: Full-screen layout with navigation sidebar and interactive graph

### API Endpoints
- `GET /api/module` - Full module JSON
- `GET /api/module/graph` - Target-level dependency graph
- `GET /api/module/packages` - Package-to-package dependencies
- `GET /api/target/{label}/focused` - File-level dependency graph for a target
- `GET /api/binaries` - Binary metadata with overlapping dependencies
- `GET /api/binaries/graph` - Binary dependency graph
- `SSE /api/subscribe/workspace_status` - Real-time analysis progress
- `SSE /api/subscribe/target_graph` - Streaming graph updates

### Active Packages
- `pkg/bazel` - Bazel query and workspace interaction
- `pkg/binaries` - Binary analysis and overlapping dependency detection
- `pkg/deps` - Compile dependency parsing (.d files)
- `pkg/graph` - File graph construction
- `pkg/model` - Core Module data model
- `pkg/pubsub` - SSE event streaming
- `pkg/symbols` - Symbol dependency analysis (nm)
- `pkg/web` - Web server and API endpoints

## Potential Future Enhancements
- **Cycle Detection**: Detect and visualize circular dependencies in the dependency graph
- **Package-level View**: Show overlapping dependencies at package granularity
- **Export Reports**: Export dependency analysis and issues to various formats (JSON, CSV, HTML)
- **Search & Filter**: Filter graph by target type, package, visibility, or dependency type
- **Performance Metrics**: Show compilation impact and dependency weight analysis
