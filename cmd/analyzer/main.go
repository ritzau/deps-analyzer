package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ritzau/deps-analyzer/pkg/analysis"
	"github.com/ritzau/deps-analyzer/pkg/bazel"
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

	// Run analysis
	allFiles, _, uncovered, err := runAnalysis(*workspace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	coveredCount := len(allFiles) - len(uncovered)

	if *webMode {
		// Start web server
		startWebServer(*workspace, len(allFiles), coveredCount, uncovered, *port)
	} else {
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

func startWebServer(workspace string, totalFiles, coveredFiles int, uncovered []analysis.UncoveredFile, port int) {
	percentage := 100.0
	if totalFiles > 0 {
		percentage = float64(coveredFiles) / float64(totalFiles) * 100.0
	}

	// Build dependency graph
	fmt.Println("Building dependency graph...")
	targetGraph, err := graph.BuildTargetGraph(workspace)
	var graphData *web.GraphData
	if err != nil {
		fmt.Printf("Warning: Could not build dependency graph: %v\n", err)
	} else {
		// Convert to web format
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

		fmt.Printf("Graph: %d nodes, %d edges\n", len(graphData.Nodes), len(graphData.Edges))
	}

	// Create analysis data
	data := &web.AnalysisData{
		Workspace:       workspace,
		TotalFiles:      totalFiles,
		CoveredFiles:    coveredFiles,
		UncoveredFiles:  uncovered,
		CoveragePercent: percentage,
		Graph:           graphData,
	}

	// Create and start server
	server := web.NewServer()
	server.SetAnalysisData(data)

	fmt.Printf("Starting web server on http://localhost:%d\n", port)
	fmt.Printf("Coverage: %.0f%% (%d/%d files)\n", percentage, coveredFiles, totalFiles)

	if err := server.Start(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
