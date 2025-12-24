# Removed Features & TODOs

This document tracks features that were removed during the Module model migration and TODOs for bringing them back.

## Removed in Streamlined Analysis (2025-12-23)

### 1. CLI Mode
**Status**: Removed
**Location**: `cmd/deps-analyzer/main.go:28-33`
**What it did**: Printed file coverage report to console (files not in any Bazel target)
**TODO**: Add CLI mode back with Module-based output:
- Show targets, dependencies by type, packages
- Show dependency issues/warnings
- Optional: coverage analysis (files not in any target)

### 2. File Coverage Analysis
**Status**: Removed
**Dependencies**: `pkg/finder`, `pkg/analysis.FindUncoveredFiles`, `pkg/output`
**What it did**: Found all source files and compared to Bazel targets to find uncovered files
**TODO**: Can be brought back as optional feature in CLI or web UI

### 3. Old AnalysisData Structure
**Status**: Removed
**Location**: `pkg/web/server.go` (was lines 42-53)
**What it stored**: Coverage data, cross-package deps, file cycles
**Replaced by**: Module model

### 4. File-Level Graph Visualization
**Status**: ✅ **RESTORED** (2025-12-24)
**Location**: `pkg/web/server.go` (buildTargetFocusedGraph function)
**Endpoints**:
- `GET /api/target/{label}/focused` - Target-focused graph with file-level dependencies
**What it does**: Shows individual files within a target and their compile/symbol dependencies to files in other targets
**Implementation**: Uses Module compile dependencies (from .d files) and symbol dependencies (from nm analysis) to show file-level edges between targets

### 5. File Cycles Detection
**Status**: Removed
**Dependencies**: `pkg/cycles`
**What it did**: Detected circular file-level dependencies
**Note**: Module model now has compile and symbol dependencies at target level, which captures the important circular dependency information

### 6. Cross-Package File Dependencies
**Status**: Removed
**Dependencies**: `pkg/analysis.CrossPackageDep`
**What it did**: Tracked file-to-file dependencies across package boundaries
**Replaced by**: Module compile dependencies (target-level)

### 7. FileGraph Structure
**Status**: Removed
**Dependencies**: `pkg/graph.FileGraph`
**What it did**: Graph of file-to-file compile dependencies from .d files
**Replaced by**: Module compile dependencies (aggregated to target level)

### 8. Separate Symbol Dependency Storage
**Status**: Removed (integrated into Module)
**Was**: `Server.symbolDeps []symbols.SymbolDependency`
**Now**: Part of `Module.Dependencies` with `Type == DependencySymbol`

## What's Still Working

### Core Functionality
✅ **Module Model**: Complete dependency graph with 5 types (static, dynamic, data, compile, symbol)
✅ **Target-level Graph**: Visualization of all targets and their dependencies
✅ **File-level Graph**: Click on any target to see files and their dependencies (compile & symbol edges)
✅ **Package-level Dependencies**: Aggregated dependencies between packages
✅ **Binary Analysis**: Binary metadata, system libraries, overlapping dependencies
✅ **Overlapping Dependency Visualization**: Red highlighting for duplicate symbols in module overview and binary-focused views
✅ **Issue Detection**: Duplicate linkage warnings (static+dynamic to same target)
✅ **Web Server**: API endpoints for Module, graph, packages, binaries

### API Endpoints Still Available
- `GET /api/module` - Full module JSON
- `GET /api/module/graph` - Target-level dependency graph
- `GET /api/module/packages` - Package-to-package dependencies
- `GET /api/target/{label}/focused` - File-level dependency graph for a target
- `GET /api/binaries` - Binary metadata with overlapping dependencies
- `GET /api/binaries/graph` - Binary dependency graph
- `SSE /api/subscribe/workspace_status` - Real-time analysis progress
- `SSE /api/subscribe/target_graph` - Streaming graph updates

## Performance Improvements

### Before (4 steps):
1. File coverage analysis (finder, analysis)
2. Build target graph (duplicate of Module)
3. Parse .d files again (duplicate work)
4. Run nm analysis again (duplicate work)

### After (2 steps):
1. Query Module + AddCompileDependencies + AddSymbolDependencies
2. Binary analysis (GetAllBinariesInfo)

**Result**: Eliminated 3 redundant analysis passes, significantly faster initialization.

## Packages That Could Be Removed

These packages are no longer used and could be deleted:
- ~~`pkg/cycles`~~ - File cycle detection
- ~~`pkg/graph/target_graph.go`~~ - Old target graph (already renamed to .old)
- Potentially `pkg/finder` and `pkg/output` if CLI coverage analysis not brought back
- Potentially `pkg/analysis` (CrossPackageDep, UncoveredFile) if those features not brought back

## Next Steps

1. **CLI Mode (Optional)**: Decide if file coverage analysis should be restored
   - Would show files not in any Bazel target
   - Requires `pkg/finder` and `pkg/analysis` packages

2. **File Cycles Detection (Optional)**: Consider bringing back cycle detection
   - Original `pkg/cycles` detected circular file dependencies
   - Could be useful for identifying problematic include chains

3. **Clean up unused packages**:
   - `pkg/finder` - Only needed if CLI coverage analysis is restored
   - `pkg/analysis` - CrossPackageDep and UncoveredFile types
   - `pkg/output` - CLI output formatting

4. **Potential Enhancements**:
   - Add legend entry for overlapping dependency visualization
   - Package-level view for overlapping dependencies
   - Export overlapping dependency report
   - Cycle detection in dependency graph
