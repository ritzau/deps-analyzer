package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/ritzau/deps-analyzer/pkg/analysis"
	"github.com/ritzau/deps-analyzer/pkg/bazel"
	"github.com/ritzau/deps-analyzer/pkg/cycles"
	"github.com/ritzau/deps-analyzer/pkg/deps"
	"github.com/ritzau/deps-analyzer/pkg/finder"
	"github.com/ritzau/deps-analyzer/pkg/graph"
	"github.com/ritzau/deps-analyzer/pkg/output"
	"github.com/ritzau/deps-analyzer/pkg/web"
)

func main() {
	// Parse command-line flags
	workspace := flag.String("workspace", ".", "Path to the Bazel workspace root")
	webMode := flag.Bool("web", false, "Start web server instead of printing to console")
	port := flag.Int("port", 8080, "Port for web server (only used with --web)")
	flag.Parse()

	if *webMode {
		// Start web server first, then run analysis in background
		startWebServerAsync(*workspace, *port)
	} else {
		// Run analysis synchronously for CLI mode
		allFiles, _, uncovered, err := runAnalysis(*workspace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		coveredCount := len(allFiles) - len(uncovered)
		// Print colorized coverage report to console
		output.PrintCoverageReport(*workspace, len(allFiles), coveredCount, uncovered)
	}
}

func runAnalysis(workspace string) ([]string, []string, []analysis.UncoveredFile, error) {
	// Find all source files in the workspace
	allFiles, err := finder.FindSourceFiles(workspace)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("finding source files: %w", err)
	}

	// Query Bazel for covered files
	coveredFiles, err := bazel.QueryAllSourceFiles(workspace)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("querying Bazel for covered files: %w", err)
	}

	// Find uncovered files
	uncovered := analysis.FindUncoveredFiles(allFiles, coveredFiles)

	return allFiles, coveredFiles, uncovered, nil
}

func startWebServerAsync(workspace string, port int) {
	// Create server with initial empty data
	server := web.NewServer()

	// Set initial "loading" state
	initialData := &web.AnalysisData{
		Workspace:       workspace,
		TotalFiles:      0,
		CoveredFiles:    0,
		UncoveredFiles:  []analysis.UncoveredFile{},
		CoveragePercent: 0,
	}
	server.SetAnalysisData(initialData)

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

	// Run analysis in background and update server data progressively
	go func() {
		log.Println("[1/4] Finding source files...")
		allFiles, _, uncovered, err := runAnalysis(workspace)
		if err != nil {
			log.Printf("Error during analysis: %v", err)
			return
		}

		coveredCount := len(allFiles) - len(uncovered)
		percentage := 100.0
		if len(allFiles) > 0 {
			percentage = float64(coveredCount) / float64(len(allFiles)) * 100.0
		}

		log.Printf("[1/4] Complete: Found %d source files, %d covered (%.0f%%)", len(allFiles), coveredCount, percentage)

		// Update with coverage data
		data := &web.AnalysisData{
			Workspace:       workspace,
			TotalFiles:      len(allFiles),
			CoveredFiles:    coveredCount,
			UncoveredFiles:  uncovered,
			CoveragePercent: percentage,
			AnalysisStep:    1,
		}
		server.SetAnalysisData(data)

		// Build dependency graph
		log.Println("[2/4] Building target dependency graph...")
		targetGraph, err := graph.BuildTargetGraph(workspace)
		var graphData *web.GraphData
		if err != nil {
			log.Printf("[2/4] Warning: Could not build dependency graph: %v", err)
		} else {
			graphData = &web.GraphData{
				Nodes: make([]web.GraphNode, 0),
				Edges: make([]web.GraphEdge, 0),
			}

			for _, node := range targetGraph.Nodes() {
				graphData.Nodes = append(graphData.Nodes, web.GraphNode{
					ID:    node.Label,
					Label: node.Label,
					Type:  node.Kind,
				})
			}

			for _, edge := range targetGraph.Edges() {
				graphData.Edges = append(graphData.Edges, web.GraphEdge{
					Source: edge[0],
					Target: edge[1],
				})
			}

			log.Printf("[2/4] Complete: Graph has %d nodes, %d edges", len(graphData.Nodes), len(graphData.Edges))
		}

		// Update with graph data
		data.Graph = graphData
		data.AnalysisStep = 2
		server.SetAnalysisData(data)

		// Build file dependency graph and detect cycles
		log.Println("[3/4] Parsing .d files for file-level dependencies...")
		var crossPackageDeps []analysis.CrossPackageDep
		var fileCycles []cycles.FileCycle
		var fileGraph *graph.FileGraph
		fileDeps, err := deps.ParseAllDFiles(workspace)
		if err != nil {
			log.Printf("[3/4] Warning: Could not parse .d files: %v", err)
		} else if len(fileDeps) == 0 {
			log.Println("[3/4] Note: No .d files found. Run 'bazel build //... --save_temps' to generate them.")
		} else {
			log.Printf("[3/4] Found %d .d files", len(fileDeps))
			fileGraph = graph.BuildFileGraph(fileDeps)

			// Build file-to-target mapping for accurate target labels
			log.Println("[3/4] Building file-to-target mapping...")
			fileToTarget, err := bazel.BuildFileToTargetMap(workspace)
			if err != nil {
				log.Printf("[3/4] Warning: Could not build file-to-target map: %v", err)
				crossPackageDeps = analysis.FindCrossPackageDeps(fileGraph)
			} else {
				log.Printf("[3/4] Mapped %d files to targets", len(fileToTarget))
				crossPackageDeps = analysis.FindCrossPackageDepsWithTargets(fileGraph, fileToTarget)
			}

			fileCycles = cycles.FindFileCycles(fileGraph)
			log.Printf("[3/4] Complete: Found %d cross-package file dependencies, %d circular dependencies", len(crossPackageDeps), len(fileCycles))

			// Store file graph and cross-package deps in server for target detail queries
			server.SetFileGraph(fileGraph)
			server.SetCrossPackageDeps(crossPackageDeps)
		}

		// Update with cross-package deps and cycles
		data.CrossPackageDeps = crossPackageDeps
		data.FileCycles = fileCycles
		data.AnalysisStep = 4
		server.SetAnalysisData(data)

		log.Println("[4/4] Analysis complete! View results at", url)
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
