package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/ritzau/deps-analyzer/pkg/analysis"
	"github.com/ritzau/deps-analyzer/pkg/logging"
	"github.com/ritzau/deps-analyzer/pkg/watcher"
	"github.com/ritzau/deps-analyzer/pkg/web"
)

func main() {
	// Parse command-line flags
	// Long and short options following Unix conventions
	workspace := flag.String("workspace", ".", "Path to the Bazel workspace root")
	workspaceShort := flag.String("w", ".", "Path to the Bazel workspace root (shorthand)")

	webMode := flag.Bool("web", false, "Start web server instead of printing to console")

	port := flag.Int("port", 8080, "Port for web server (only used with --web)")
	portShort := flag.Int("p", 8080, "Port for web server (shorthand)")

	watch := flag.Bool("watch", false, "Watch for file changes and re-analyze automatically")

	open := flag.Bool("open", true, "Automatically open browser when starting web server")
	flag.Parse()

	// Resolve short vs long options (short takes precedence if different from default)
	workspaceVal := *workspace
	if *workspaceShort != "." {
		workspaceVal = *workspaceShort
	}

	portVal := *port
	if *portShort != 8080 {
		portVal = *portShort
	}

	if *webMode {
		// Start web server and run streamlined analysis
		startWebServerAsync(workspaceVal, portVal, *watch, *open)
	} else {
		// TODO: Add CLI mode back with Module-based output
		// - Show targets, dependencies by type, packages
		// - Show dependency issues/warnings
		// - Optional: coverage analysis (files not in any target)
		fmt.Fprintf(os.Stderr, "CLI mode not yet implemented. Use --web flag to start web server.\n")
		os.Exit(1)
	}
}

func startWebServerAsync(workspace string, port int, watch bool, open bool) {
	// Create server
	server := web.NewServer()

	url := fmt.Sprintf("http://localhost:%d", port)
	fmt.Printf("Starting web server on %s\n", url)

	// Start server in background
	go func() {
		if err := server.Start(port); err != nil {
			logging.Fatal("failed to start server", "error", err)
		}
	}()

	// Wait a moment for server to start
	time.Sleep(500 * time.Millisecond)

	// Open browser if requested
	if open {
		fmt.Println("Opening browser...")
		openBrowser(url)
	} else {
		fmt.Printf("Server ready at %s (use --open to auto-open browser)\n", url)
	}

	// Create analysis runner
	runner := analysis.NewAnalysisRunner(workspace, server)
	ctx := context.Background()

	// Run initial analysis in background
	go func() {
		err := runner.Run(ctx, analysis.AnalysisOptions{
			FullAnalysis: true,
			Reason:       "initial analysis",
		})
		if err != nil {
			logging.Error("initial analysis failed", "error", err)
			return
		}

		// Start file watcher if requested
		if watch {
			startFileWatcher(ctx, workspace, runner, server)
		}
	}()

	// Block forever (server runs in goroutine)
	select {}
}

func startFileWatcher(ctx context.Context, workspace string, runner *analysis.AnalysisRunner, server *web.Server) {
	logging.Info("starting file watcher", "workspace", workspace)

	// Notify UI that watching is active
	server.SetWatching(true)
	server.PublishWorkspaceStatus("watching", "Watching for changes...", 6, 6)

	// Create watcher
	fw, err := watcher.NewFileWatcher(workspace)
	if err != nil {
		logging.Error("failed to create file watcher", "error", err)
		return
	}

	// Start watcher
	if err := fw.Start(ctx); err != nil {
		logging.Error("failed to start file watcher", "error", err)
		return
	}

	// Create debouncer
	debouncer := watcher.NewDebouncer(
		fw.Events(),
		1500*time.Millisecond, // quietPeriod
		10*time.Second,         // maxWait
	)
	debouncer.Start(ctx)

	logging.Info("file watcher ready - monitoring for changes")

	// Process debounced events
	go func() {
		for event := range debouncer.Output() {
			logging.Info("file changes detected", "filesChanged", len(event.Paths))

			// Analyze what changed
			changeAnalysis := watcher.AnalyzeChanges(event, workspace)

			// Determine reason for re-analysis
			reason := formatReason(event)
			logging.Info("triggering re-analysis", "reason", reason)

			// Build analysis options
			opts := analysis.AnalysisOptions{
				FullAnalysis:    changeAnalysis.NeedFullAnalysis,
				SkipBazelQuery:  !changeAnalysis.NeedFullAnalysis,
				SkipCompileDeps: !changeAnalysis.NeedCompileDeps,
				SkipSymbolDeps:  !changeAnalysis.NeedSymbolDeps,
				SkipBinaryDeriv: !changeAnalysis.NeedBinaryDeriv,
				Reason:          reason,
			}

			// Run re-analysis
			err := runner.Run(ctx, opts)
			if err != nil {
				logging.Error("re-analysis failed", "error", err)
				// Don't crash - just log and continue watching
			}

			// Restore watching state
			server.PublishWorkspaceStatus("watching", "Watching for changes...", 6, 6)
		}
	}()
}

func formatReason(event watcher.ChangeEvent) string {
	switch event.Type {
	case watcher.ChangeTypeBuildFile:
		return "BUILD files changed"
	case watcher.ChangeTypeDFile:
		return "Compile dependencies changed"
	case watcher.ChangeTypeOFile:
		return "Symbol dependencies changed"
	default:
		return "Files changed"
	}
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
		logging.Warn("cannot open browser on unsupported platform", "platform", runtime.GOOS)
		return
	}

	if err := exec.Command(cmd, args...).Start(); err != nil {
		logging.Warn("failed to open browser", "error", err)
	}
}
