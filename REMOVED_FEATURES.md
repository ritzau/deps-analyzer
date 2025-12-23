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
**Status**: Removed
**Location**: `pkg/web/server.go` (buildFileGraphData function)
**Endpoints removed**:
- `GET /api/target/{label}` - Target file details
- `GET /api/target/{label}/graph` - File-level dependency graph
**What it did**: Showed individual files within a target and their compile dependencies to files in other targets
**TODO**: Bring back using Module compile dependencies
- Location: `pkg/web/server.go:76-78`
- Can build from Module.Dependencies where Type == DependencyCompile
- Would show which specific files in target A depend on files in target B

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
✅ **Package-level Dependencies**: Aggregated dependencies between packages
✅ **Binary Analysis**: Binary metadata, system libraries, overlapping dependencies
✅ **Issue Detection**: Duplicate linkage warnings (static+dynamic to same target)
✅ **Web Server**: API endpoints for Module, graph, packages, binaries

### API Endpoints Still Available
- `GET /api/module` - Full module JSON
- `GET /api/module/graph` - Target-level dependency graph
- `GET /api/module/packages` - Package-to-package dependencies
- `GET /api/binaries` - Binary metadata
- `GET /api/binaries/graph` - Binary dependency graph

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

1. **Decide on CLI mode**: Do we want file coverage analysis back?
2. **File-level visualization**: Implement using Module compile dependencies if desired
3. **Clean up unused packages**: Remove or document obsolete code
4. **Update frontend**: Web UI may reference removed endpoints (/api/analysis, /api/target/*)
