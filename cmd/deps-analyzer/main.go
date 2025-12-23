package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/ritzau/deps-analyzer/pkg/bazel"
	"github.com/ritzau/deps-analyzer/pkg/binaries"
	"github.com/ritzau/deps-analyzer/pkg/deps"
	"github.com/ritzau/deps-analyzer/pkg/symbols"
	"github.com/ritzau/deps-analyzer/pkg/web"
)

func main() {
	// Parse command-line flags
	workspace := flag.String("workspace", ".", "Path to the Bazel workspace root")
	webMode := flag.Bool("web", false, "Start web server instead of printing to console")
	port := flag.Int("port", 8080, "Port for web server (only used with --web)")
	flag.Parse()

	if *webMode {
		// Start web server and run streamlined analysis
		startWebServerAsync(*workspace, *port)
	} else {
		// TODO: Add CLI mode back with Module-based output
		// - Show targets, dependencies by type, packages
		// - Show dependency issues/warnings
		// - Optional: coverage analysis (files not in any target)
		fmt.Fprintf(os.Stderr, "CLI mode not yet implemented. Use --web flag to start web server.\n")
		os.Exit(1)
	}
}

func startWebServerAsync(workspace string, port int) {
	// Create server
	server := web.NewServer()

	url := fmt.Sprintf("http://localhost:%d", port)
	fmt.Printf("Starting web server on %s\n", url)

	// Start server in background
	go func() {
		if err := server.Start(port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait a moment for server to start
	time.Sleep(500 * time.Millisecond)

	// Open browser
	fmt.Println("Opening browser...")
	openBrowser(url)

	// Run streamlined analysis in background
	go func() {
		// Publish initial state
		server.PublishWorkspaceStatus("initializing", "Starting analysis...", 0, 5)

		// Query the Module model with all dependency types
		log.Println("[1/2] Querying Bazel module...")
		server.PublishWorkspaceStatus("bazel_querying", "Querying Bazel workspace...", 1, 5)

		module, err := bazel.QueryWorkspace(workspace)
		if err != nil {
			log.Printf("[1/2] Error: Could not query module: %v", err)
			server.PublishWorkspaceStatus("error", fmt.Sprintf("Error querying workspace: %v", err), 1, 5)
			return
		}

		log.Printf("[1/2] Found %d targets, %d dependencies", len(module.Targets), len(module.Dependencies))
		server.SetModule(module)
		server.PublishTargetGraph("partial_data", false)

		// Add compile dependencies from .d files
		log.Println("[1/2] Adding compile dependencies from .d files...")
		server.PublishWorkspaceStatus("analyzing_deps", "Adding compile dependencies...", 2, 5)

		// Parse file-level dependencies and store them
		fileDeps, err := deps.ParseAllDFiles(workspace)
		if err != nil {
			log.Printf("[1/2] Warning: Could not parse .d files: %v", err)
		} else {
			log.Printf("[1/2] Parsed %d file dependencies from .d files", len(fileDeps))
			server.SetFileDependencies(fileDeps)
		}

		// Add target-level compile dependencies
		if err := bazel.AddCompileDependencies(module, workspace); err != nil {
			log.Printf("[1/2] Warning: Could not add compile dependencies: %v", err)
		} else {
			log.Printf("[1/2] Added compile dependencies, now have %d total dependencies", len(module.Dependencies))
		}
		server.PublishTargetGraph("partial_data", false)

		// Add symbol dependencies from nm
		log.Println("[1/2] Adding symbol dependencies from nm analysis...")
		server.PublishWorkspaceStatus("analyzing_symbols", "Adding symbol dependencies...", 3, 5)

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
		server.SetFileToTargetMap(fileToTarget)

		// Build symbol graph and store file-level symbol dependencies
		symbolDeps, err := symbols.BuildSymbolGraph(workspace, fileToTarget, targetToKind)
		if err != nil {
			log.Printf("[1/2] Warning: Could not build symbol graph: %v", err)
		} else {
			log.Printf("[1/2] Found %d symbol dependencies between files", len(symbolDeps))
			server.SetSymbolDependencies(symbolDeps)
		}

		// Add target-level symbol dependencies
		if err := bazel.AddSymbolDependencies(module, workspace); err != nil {
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
		server.SetModule(module)
		server.PublishWorkspaceStatus("targets_ready", "Target analysis complete", 4, 5)
		server.PublishTargetGraph("complete", true)

		// Derive binary-level information from the Module (fast, no additional queries)
		log.Println("[2/2] Deriving binary information from module...")
		server.PublishWorkspaceStatus("analyzing_binaries", "Deriving binary info...", 5, 5)

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
			if len(bin.OverlappingDeps) > 0 {
				log.Printf("[2/2]     ⚠️  Overlapping deps (potential duplicate symbols):")
				for sharedLib, targets := range bin.OverlappingDeps {
					log.Printf("[2/2]       %s shares: %v", sharedLib, targets)
				}
			}
		}
		server.SetBinaries(binaryInfos)

		log.Printf("[2/2] Analysis complete! Module has %d targets, %d dependencies (%d packages)",
			len(module.Targets), len(module.Dependencies), module.GetPackageCount())
		log.Println("View results at", url)

		// Publish final ready state
		server.PublishWorkspaceStatus("ready", "Analysis complete", 5, 5)
	}()

	// Block forever (server runs in goroutine)
	select {}
}

func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		log.Printf("Cannot open browser on platform: %s", runtime.GOOS)
		return
	}

	if err := exec.Command(cmd, args...).Start(); err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}
