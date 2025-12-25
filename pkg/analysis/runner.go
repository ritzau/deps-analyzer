package analysis

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/ritzau/deps-analyzer/pkg/bazel"
	"github.com/ritzau/deps-analyzer/pkg/binaries"
	"github.com/ritzau/deps-analyzer/pkg/deps"
	"github.com/ritzau/deps-analyzer/pkg/symbols"
	"github.com/ritzau/deps-analyzer/pkg/web"
)

// AnalysisRunner orchestrates the analysis process
type AnalysisRunner struct {
	workspace string
	server    *web.Server
	mu        sync.Mutex // Prevent concurrent analysis runs
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
func NewAnalysisRunner(workspace string, server *web.Server) *AnalysisRunner {
	return &AnalysisRunner{
		workspace: workspace,
		server:    server,
	}
}

// Run executes the analysis with the given options
func (ar *AnalysisRunner) Run(ctx context.Context, opts AnalysisOptions) error {
	// Lock to prevent concurrent analysis
	ar.mu.Lock()
	defer ar.mu.Unlock()

	log.Printf("[ANALYSIS] Starting analysis: %s", opts.Reason)

	// Phase 1: Bazel Query
	module := ar.server.GetModule()
	if !opts.SkipBazelQuery {
		ar.server.PublishWorkspaceStatus("bazel_querying", "Querying Bazel workspace...", 1, 6)
		log.Println("[1/2] Querying Bazel module...")

		var err error
		module, err = bazel.QueryWorkspace(ar.workspace)
		if err != nil {
			log.Printf("[1/2] Error: Could not query module: %v", err)
			ar.server.PublishWorkspaceStatus("error", fmt.Sprintf("Error querying workspace: %v", err), 1, 6)
			return fmt.Errorf("bazel query failed: %w", err)
		}

		log.Printf("[1/2] Found %d targets, %d dependencies", len(module.Targets), len(module.Dependencies))
		ar.server.SetModule(module)
		ar.server.PublishTargetGraph("partial_data", false)
	}

	// Phase 2: Compile Dependencies
	if !opts.SkipCompileDeps {
		ar.server.PublishWorkspaceStatus("analyzing_deps", "Adding compile dependencies...", 2, 6)
		log.Println("[1/2] Adding compile dependencies from .d files...")

		// Parse file-level dependencies and store them
		fileDeps, err := deps.ParseAllDFiles(ar.workspace)
		if err != nil {
			log.Printf("[1/2] Warning: Could not parse .d files: %v", err)
		} else {
			log.Printf("[1/2] Parsed %d file dependencies from .d files", len(fileDeps))
			ar.server.SetFileDependencies(fileDeps)
		}

		// Add target-level compile dependencies
		if err := bazel.AddCompileDependencies(module, ar.workspace); err != nil {
			log.Printf("[1/2] Warning: Could not add compile dependencies: %v", err)
		} else {
			log.Printf("[1/2] Added compile dependencies, now have %d total dependencies", len(module.Dependencies))
		}
		ar.server.PublishTargetGraph("partial_data", false)
	}

	// Phase 3: Symbol Dependencies
	if !opts.SkipSymbolDeps {
		ar.server.PublishWorkspaceStatus("analyzing_symbols", "Adding symbol dependencies...", 3, 6)
		log.Println("[1/2] Adding symbol dependencies from nm analysis...")

		// Build file-to-target map for symbol analysis and file dependencies
		fileToTarget := make(map[string]string)
		targetToKind := make(map[string]string)
		for _, target := range module.Targets {
			targetToKind[target.Label] = string(target.Kind)
			// Map source files
			for _, src := range target.Sources {
				filePath := bazel.NormalizeSourcePath(src)
				fileToTarget[filePath] = target.Label
			}
			// Map header files
			for _, hdr := range target.Headers {
				filePath := bazel.NormalizeSourcePath(hdr)
				fileToTarget[filePath] = target.Label
			}
		}
		ar.server.SetFileToTargetMap(fileToTarget)

		// Discover source files in workspace
		log.Println("[1/2] Discovering source files in workspace...")
		ar.server.PublishWorkspaceStatus("discovering_files", "Discovering source files...", 4, 6)

		discovered, err := bazel.DiscoverSourceFiles(ar.workspace)
		if err != nil {
			log.Printf("[1/2] Warning: Failed to discover source files: %v", err)
			discovered = make(map[string]bool)
		}

		// Find uncovered files
		uncoveredFiles := bazel.FindUncoveredFiles(discovered, fileToTarget)
		if len(uncoveredFiles) > 0 {
			log.Printf("[1/2] Found %d uncovered files not included in any target:", len(uncoveredFiles))
			for _, file := range uncoveredFiles {
				log.Printf("[1/2]   - %s", file)
			}
		} else {
			log.Println("[1/2] All source files are covered by targets")
		}

		// Store for web API
		ar.server.SetUncoveredFiles(uncoveredFiles)

		// Build symbol graph and store file-level symbol dependencies
		symbolDeps, err := symbols.BuildSymbolGraph(ar.workspace, fileToTarget, targetToKind)
		if err != nil {
			log.Printf("[1/2] Warning: Could not build symbol graph: %v", err)
		} else {
			log.Printf("[1/2] Found %d symbol dependencies between files", len(symbolDeps))
			ar.server.SetSymbolDependencies(symbolDeps)
		}

		// Add target-level symbol dependencies
		if err := bazel.AddSymbolDependencies(module, ar.workspace); err != nil {
			log.Printf("[1/2] Warning: Could not add symbol dependencies: %v", err)
		} else {
			log.Printf("[1/2] Module has %d total dependencies", len(module.Dependencies))
			if len(module.Issues) > 0 {
				log.Printf("[1/2] ⚠️  Found %d dependency issues", len(module.Issues))
				for _, issue := range module.Issues {
					log.Printf("[1/2]   %s: %s -> %s (%v)", issue.Severity, issue.From, issue.To, issue.Types)
				}
			}
		}

		// Store module in server and publish targets ready
		ar.server.SetModule(module)
		ar.server.PublishWorkspaceStatus("targets_ready", "Target analysis complete", 5, 6)
		ar.server.PublishTargetGraph("complete", true)
	}

	// Phase 4: Binary Derivation
	if !opts.SkipBinaryDeriv {
		ar.server.PublishWorkspaceStatus("analyzing_binaries", "Deriving binary info...", 6, 6)
		log.Println("[2/2] Deriving binary information from module...")

		binaryInfos := binaries.DeriveBinaryInfoFromModule(module)
		log.Printf("[2/2] Found %d binaries", len(binaryInfos))
		for _, bin := range binaryInfos {
			log.Printf("[2/2]   %s (%s)", bin.Label, bin.Kind)
			if len(bin.DynamicDeps) > 0 {
				log.Printf("[2/2]     Dynamic deps: %v", bin.DynamicDeps)
			}
			if len(bin.DataDeps) > 0 {
				log.Printf("[2/2]     Data deps: %v", bin.DataDeps)
			}
			if len(bin.SystemLibraries) > 0 {
				log.Printf("[2/2]     System libs: %v", bin.SystemLibraries)
			}
		}
		ar.server.SetBinaries(binaryInfos)

		log.Printf("[2/2] Analysis complete! Module has %d targets, %d dependencies (%d packages)",
			len(module.Targets), len(module.Dependencies), module.GetPackageCount())
	}

	// Publish final ready state
	ar.server.PublishWorkspaceStatus("ready", "Analysis complete", 6, 6)

	log.Printf("[ANALYSIS] Complete: %s", opts.Reason)
	return nil
}
