package analysis

import (
	"context"
	"fmt"
	"sync"

	"github.com/ritzau/deps-analyzer/pkg/analysis/api"
	"github.com/ritzau/deps-analyzer/pkg/binaries"
	"github.com/ritzau/deps-analyzer/pkg/config"
	"github.com/ritzau/deps-analyzer/pkg/deps"
	"github.com/ritzau/deps-analyzer/pkg/logging"
	"github.com/ritzau/deps-analyzer/pkg/model"
	"github.com/ritzau/deps-analyzer/pkg/symbols"
	"github.com/ritzau/deps-analyzer/pkg/web"
)

// AnalysisRunner orchestrates the analysis process
type AnalysisRunner struct {
	workspace string
	server    *web.Server
	mu        sync.Mutex   // Prevent concurrent analysis runs
	Sources   []api.Source // Registered sources
	Config    *config.Config
	Graph     *model.Graph

	// Dependency Injection functions to break import cycles
	// These placeholders allow main.go to inject implementations from pkg/bazel
	// without this package depending on pkg/bazel.
	FnQueryWorkspace        func(workspace string) (*model.Module, error)
	FnAddCompileDeps        func(module *model.Module, workspace string) error
	FnNormalizeSourcePath   func(path string) string
	FnDiscoverSourceFiles   func(workspace string) (map[string]bool, error)
	FnFindUncoveredFiles    func(discovered map[string]bool, fileToTarget map[string]string) []string
	FnAddSymbolDependencies func(module *model.Module, workspace string) error
}

// AnalysisOptions configures which analysis phases to run
type AnalysisOptions struct {
	FullAnalysis    bool
	SkipBazelQuery  bool
	SkipCompileDeps bool
	SkipSymbolDeps  bool
	SkipBinaryDeriv bool
	Reason          string // e.g., "initial analysis", "BUILD changed"
}

// NewAnalysisRunner creates a new analysis runner
func NewAnalysisRunner(workspace string, server *web.Server, cfg *config.Config) *AnalysisRunner {
	return &AnalysisRunner{
		workspace: workspace,
		server:    server,
		Config:    cfg,
		Graph:     model.NewGraph(),
		Sources:   make([]api.Source, 0),
	}
}

// RegisterSource adds a source to the runner
func (ar *AnalysisRunner) RegisterSource(s api.Source) {
	ar.Sources = append(ar.Sources, s)
}

// Run executes the analysis with the given options
func (ar *AnalysisRunner) Run(ctx context.Context, opts AnalysisOptions) error {
	// Lock to prevent concurrent analysis
	ar.mu.Lock()
	defer ar.mu.Unlock()

	logging.Info("starting analysis", "reason", opts.Reason)

	// Run registered sources
	ar.runRegisteredSources(ctx, opts.Reason)

	// Phase 1: Bazel Query
	module, err := ar.runBazelQueryPhase(opts)
	if err != nil {
		return err
	}

	// Phase 2: Compile Dependencies
	ar.runCompileDepsPhase(opts, module)

	// Phase 3: Symbol Dependencies
	ar.runSymbolDepsPhase(opts, module)

	// Phase 4: Binary Derivation
	ar.runBinaryDerivationPhase(opts, module)

	// Publish final ready state
	_ = ar.server.PublishWorkspaceStatus("ready", "Analysis complete", 6, 6)

	logging.Info("analysis complete", "reason", opts.Reason)
	return nil
}

func (ar *AnalysisRunner) runRegisteredSources(ctx context.Context, reason string) {
	for _, src := range ar.Sources {
		logging.Info("running source", "name", src.Name())
		graph, err := src.Run(ctx, ar.Config)
		if err != nil {
			logging.Error("source failed", "name", src.Name(), "error", err)
			continue
		}
		ar.Graph.Merge(graph)
		logging.Info("source complete", "name", src.Name())
	}
}

func (ar *AnalysisRunner) runBazelQueryPhase(opts AnalysisOptions) (*model.Module, error) {
	module := ar.server.GetModule()
	if !opts.SkipBazelQuery {
		if ar.FnQueryWorkspace != nil {
			_ = ar.server.PublishWorkspaceStatus("bazel_querying", "Querying Bazel workspace...", 1, 6)
			logging.Info("querying bazel module")

			var err error
			module, err = ar.FnQueryWorkspace(ar.workspace)
			if err != nil {
				logging.Error("bazel query failed", "error", err)
				_ = ar.server.PublishWorkspaceStatus("error", fmt.Sprintf("Error querying workspace: %v", err), 1, 6)
				return nil, fmt.Errorf("bazel query failed: %w", err)
			}

			logging.Info("bazel query complete", "targets", len(module.Targets), "dependencies", len(module.Dependencies))
			ar.server.SetModule(module)
			_ = ar.server.PublishTargetGraph("partial_data", false)
		} else {
			logging.Warn("FnQueryWorkspace not set, skipping bazel query")
		}
	}
	return module, nil
}

func (ar *AnalysisRunner) runCompileDepsPhase(opts AnalysisOptions, module *model.Module) {
	if !opts.SkipCompileDeps {
		_ = ar.server.PublishWorkspaceStatus("analyzing_deps", "Adding compile dependencies...", 2, 6)
		logging.Info("adding compile dependencies from .d files")

		// Parse file-level dependencies and store them
		fileDeps, err := deps.ParseAllDFiles(ar.workspace)
		if err != nil {
			logging.Warn("could not parse .d files", "error", err)
		} else {
			logging.Info("parsed file dependencies", "count", len(fileDeps))
			ar.server.SetFileDependencies(fileDeps)
		}

		// Add target-level compile dependencies
		if ar.FnAddCompileDeps != nil {
			if err := ar.FnAddCompileDeps(module, ar.workspace); err != nil {
				logging.Warn("could not add compile dependencies", "error", err)
			} else {
				logging.Info("added compile dependencies", "totalDependencies", len(module.Dependencies))
			}
		}
		_ = ar.server.PublishTargetGraph("partial_data", false)
	}
}

func (ar *AnalysisRunner) runSymbolDepsPhase(opts AnalysisOptions, module *model.Module) {
	if !opts.SkipSymbolDeps {
		_ = ar.server.PublishWorkspaceStatus("analyzing_symbols", "Adding symbol dependencies...", 3, 6)
		logging.Info("adding symbol dependencies from nm analysis")

		// Build file-to-target map for symbol analysis and file dependencies
		fileToTarget := make(map[string]string)
		targetToKind := make(map[string]string)

		// We need normalization function
		normalize := func(p string) string { return p }
		if ar.FnNormalizeSourcePath != nil {
			normalize = ar.FnNormalizeSourcePath
		}

		for _, target := range module.Targets {
			targetToKind[target.Label] = string(target.Kind)
			// Map source files
			for _, src := range target.Sources {
				filePath := normalize(src)
				fileToTarget[filePath] = target.Label
			}
			// Map header files
			for _, hdr := range target.Headers {
				filePath := normalize(hdr)
				fileToTarget[filePath] = target.Label
			}
		}
		ar.server.SetFileToTargetMap(fileToTarget)

		// Discover source files in workspace
		if ar.FnDiscoverSourceFiles != nil && ar.FnFindUncoveredFiles != nil {
			logging.Info("discovering source files in workspace")
			_ = ar.server.PublishWorkspaceStatus("discovering_files", "Discovering source files...", 4, 6)

			discovered, err := ar.FnDiscoverSourceFiles(ar.workspace)
			if err != nil {
				logging.Warn("failed to discover source files", "error", err)
				discovered = make(map[string]bool)
			}

			// Find uncovered files
			uncoveredFiles := ar.FnFindUncoveredFiles(discovered, fileToTarget)
			if len(uncoveredFiles) > 0 {
				logging.Info("found uncovered files", "count", len(uncoveredFiles))
				for _, file := range uncoveredFiles {
					logging.Debug("uncovered file", "path", file)
				}
			} else {
				logging.Info("all source files are covered by targets")
			}

			// Store for web API
			ar.server.SetUncoveredFiles(uncoveredFiles)
		}

		// Build symbol graph and store file-level symbol dependencies
		symbolDeps, err := symbols.BuildSymbolGraph(ar.workspace, fileToTarget, targetToKind)
		if err != nil {
			logging.Warn("could not build symbol graph", "error", err)
		} else {
			logging.Info("found symbol dependencies", "count", len(symbolDeps))
			ar.server.SetSymbolDependencies(symbolDeps)
		}

		// Add target-level symbol dependencies
		if ar.FnAddSymbolDependencies != nil {
			if err := ar.FnAddSymbolDependencies(module, ar.workspace); err != nil {
				logging.Warn("could not add symbol dependencies", "error", err)
			} else {
				logging.Info("module analysis complete", "totalDependencies", len(module.Dependencies))
				if len(module.Issues) > 0 {
					logging.Warn("found dependency issues", "count", len(module.Issues))
					for _, issue := range module.Issues {
						logging.Debug("dependency issue detail", "severity", issue.Severity, "from", issue.From, "to", issue.To, "types", issue.Types)
					}
				}
			}
		}

		// Store module in server and publish targets ready
		ar.server.SetModule(module)
		_ = ar.server.PublishWorkspaceStatus("targets_ready", "Target analysis complete", 5, 6)
		_ = ar.server.PublishTargetGraph("complete", true)
	}
}

func (ar *AnalysisRunner) runBinaryDerivationPhase(opts AnalysisOptions, module *model.Module) {
	if !opts.SkipBinaryDeriv {
		_ = ar.server.PublishWorkspaceStatus("analyzing_binaries", "Deriving binary info...", 6, 6)
		logging.Info("deriving binary information from module")

		binaryInfos := binaries.DeriveBinaryInfoFromModule(module)
		logging.Info("found binaries", "count", len(binaryInfos))
		for _, bin := range binaryInfos {
			logging.Debug("binary", "label", bin.Label, "kind", bin.Kind)
			if len(bin.DynamicDeps) > 0 {
				logging.Debug("binary dynamic dependencies", "label", bin.Label, "deps", bin.DynamicDeps)
			}
			if len(bin.DataDeps) > 0 {
				logging.Debug("binary data dependencies", "label", bin.Label, "deps", bin.DataDeps)
			}
			if len(bin.SystemLibraries) > 0 {
				logging.Debug("binary system libraries", "label", bin.Label, "libs", bin.SystemLibraries)
			}
		}
		ar.server.SetBinaries(binaryInfos)

		logging.Info("analysis complete",
			"targets", len(module.Targets), "dependencies", len(module.Dependencies), "packages", module.GetPackageCount())
	}
}

// GetGraph returns the current unified graph
func (ar *AnalysisRunner) GetGraph() *model.Graph {
	return ar.Graph
}
