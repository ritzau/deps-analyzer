package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/ritzau/deps-analyzer/pkg/analysis"
	"github.com/ritzau/deps-analyzer/pkg/logging"
	"github.com/ritzau/deps-analyzer/pkg/watcher"
	"github.com/ritzau/deps-analyzer/pkg/web"
	"github.com/spf13/pflag"
)

func main() {
	// Parse command-line flags using pflag for POSIX/GNU-style flags
	workspace := pflag.StringP("workspace", "w", ".", "path to Bazel workspace")
	webMode := pflag.Bool("web", false, "start web server")
	port := pflag.IntP("port", "p", 8080, "web server port")
	watch := pflag.Bool("watch", false, "watch for file changes and re-analyze")
	open := pflag.Bool("open", true, "auto-open browser when starting server")
	licenses := pflag.Bool("licenses", false, "list all third-party licenses")

	// Verbosity flags
	verboseCount := pflag.CountP("verbose", "v", "increase verbosity (can be repeated: -v, -vv, -vvv)")
	verbosity := pflag.String("verbosity", "", "set log level explicitly: T(race), D(ebug), I(nfo), W(arn), E(rror)")

	pflag.Parse()

	// Configure logging level based on verbosity flags
	configureLogging(*verboseCount, *verbosity)

	if *licenses {
		printLicenses()
		return
	}

	if *webMode {
		// Start web server and run streamlined analysis
		startWebServerAsync(*workspace, *port, *watch, *open)
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

// configureLogging sets the log level based on verbosity flags
func configureLogging(verboseCount int, verbosityFlag string) {
	var level slog.Level

	// Explicit verbosity flag takes precedence
	if verbosityFlag != "" {
		switch strings.ToUpper(verbosityFlag)[0] {
		case 'T':
			level = slog.LevelDebug - 4 // Trace
		case 'D':
			level = slog.LevelDebug
		case 'I':
			level = slog.LevelInfo
		case 'W':
			level = slog.LevelWarn
		case 'E':
			level = slog.LevelError
		default:
			fmt.Fprintf(os.Stderr, "Invalid verbosity level: %s (use T, D, I, W, or E)\n", verbosityFlag)
			os.Exit(1)
		}
	} else {
		// Use -v count to determine level
		// Default (0): Info
		// -v (1): Debug
		// -vv (2): Trace
		// -vvv+ (3+): Trace
		switch verboseCount {
		case 0:
			level = slog.LevelInfo
		case 1:
			level = slog.LevelDebug
		default: // 2 or more
			level = slog.LevelDebug - 4 // Trace
		}
	}

	logging.SetLevel(level)
}

// printLicenses outputs all third-party licenses used by this project
func printLicenses() {
	fmt.Println("Third-Party Licenses")
	fmt.Println("====================")
	fmt.Println()

	licenses := []struct {
		name    string
		author  string
		license string
		url     string
	}{
		// Go dependencies
		{
			name:    "pflag",
			author:  "The Go Authors and spf13 contributors",
			license: "BSD-3-Clause",
			url:     "https://github.com/spf13/pflag",
		},
		{
			name:    "fsnotify",
			author:  "fsnotify contributors",
			license: "BSD-3-Clause",
			url:     "https://github.com/fsnotify/fsnotify",
		},
		{
			name:    "uuid",
			author:  "Google Inc.",
			license: "BSD-3-Clause",
			url:     "https://github.com/google/uuid",
		},
		{
			name:    "gorilla/mux",
			author:  "Gorilla web toolkit contributors",
			license: "BSD-3-Clause",
			url:     "https://github.com/gorilla/mux",
		},
		{
			name:    "gonum",
			author:  "The Gonum Authors",
			license: "BSD-3-Clause",
			url:     "https://gonum.org/v1/gonum",
		},

		// Frontend JavaScript libraries
		{
			name:    "Cytoscape.js",
			author:  "The Cytoscape Consortium",
			license: "MIT",
			url:     "https://js.cytoscape.org/",
		},
		{
			name:    "dagre",
			author:  "Chris Pettitt",
			license: "MIT",
			url:     "https://github.com/dagrejs/dagre",
		},
		{
			name:    "cytoscape-dagre",
			author:  "The Cytoscape Consortium",
			license: "MIT",
			url:     "https://github.com/cytoscape/cytoscape.js-dagre",
		},

		// C++ libraries (used in example workspace, may be analyzed)
		{
			name:    "fmt",
			author:  "Victor Zverovich and contributors",
			license: "MIT",
			url:     "https://github.com/fmtlib/fmt",
		},
		{
			name:    "nlohmann/json",
			author:  "Niels Lohmann",
			license: "MIT",
			url:     "https://github.com/nlohmann/json",
		},

		// Assets
		{
			name:    "Filter icon",
			author:  "Kiranshastry",
			license: "Flaticon License",
			url:     "https://www.flaticon.com/free-icons/filter",
		},

		// Build tools (development dependencies)
		{
			name:    "esbuild",
			author:  "Evan Wallace",
			license: "MIT",
			url:     "https://github.com/evanw/esbuild",
		},
		{
			name:    "goimports",
			author:  "The Go Authors",
			license: "BSD-3-Clause",
			url:     "https://golang.org/x/tools/cmd/goimports",
		},
		{
			name:    "buildifier",
			author:  "The Bazel Authors",
			license: "Apache-2.0",
			url:     "https://github.com/bazelbuild/buildtools",
		},
		{
			name:    "lefthook",
			author:  "Evil Martians",
			license: "MIT",
			url:     "https://github.com/evilmartians/lefthook",
		},
	}

	for _, lib := range licenses {
		fmt.Printf("%s by %s — %s — %s\n", lib.name, lib.author, lib.license, lib.url)
	}
}
