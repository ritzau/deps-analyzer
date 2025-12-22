package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ritzau/deps-analyzer/pkg/analysis"
	"github.com/ritzau/deps-analyzer/pkg/cycles"
	"github.com/ritzau/deps-analyzer/pkg/graph"
	"github.com/ritzau/deps-analyzer/pkg/symbols"
)

//go:embed static/*
var staticFiles embed.FS

// GraphNode represents a node in the dependency graph
type GraphNode struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Type   string `json:"type"`   // "cc_library", "cc_binary", "source", "header", "external"
	Parent string `json:"parent"` // Parent node ID for grouping (optional)
}

// GraphEdge represents an edge in the dependency graph
type GraphEdge struct {
	Source  string   `json:"source"`
	Target  string   `json:"target"`
	Type    string   `json:"type"`    // "file" (from .d files) or "symbol" (from nm)
	Linkage string   `json:"linkage"` // For symbol edges: "static", "dynamic", or "cross"
	Symbols []string `json:"symbols"` // For symbol edges: list of symbol names
}

// GraphData holds the dependency graph for visualization
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// AnalysisData holds all the analysis results to send to the frontend
type AnalysisData struct {
	Workspace         string                      `json:"workspace"`
	TotalFiles        int                         `json:"totalFiles"`
	CoveredFiles      int                         `json:"coveredFiles"`
	UncoveredFiles    []analysis.UncoveredFile    `json:"uncoveredFiles"`
	CoveragePercent   float64                     `json:"coveragePercent"`
	Graph             *GraphData                  `json:"graph,omitempty"`
	CrossPackageDeps  []analysis.CrossPackageDep  `json:"crossPackageDeps,omitempty"`
	FileCycles        []cycles.FileCycle          `json:"fileCycles,omitempty"`
	AnalysisStep      int                         `json:"analysisStep"` // 1-4, which step is complete
}

// Server represents the web server
type Server struct {
	router           *mux.Router
	analysisData     *AnalysisData
	fileGraph        *graph.FileGraph
	crossPackageDeps []analysis.CrossPackageDep
	symbolDeps       []symbols.SymbolDependency
}

// NewServer creates a new web server
func NewServer() *Server {
	s := &Server{
		router: mux.NewRouter(),
	}
	s.setupRoutes()
	return s
}

// SetAnalysisData updates the analysis data
func (s *Server) SetAnalysisData(data *AnalysisData) {
	s.analysisData = data
}

// SetFileGraph stores the file graph for target detail queries
func (s *Server) SetFileGraph(fg *graph.FileGraph) {
	s.fileGraph = fg
}

// SetCrossPackageDeps stores cross-package dependencies for target detail queries
func (s *Server) SetCrossPackageDeps(deps []analysis.CrossPackageDep) {
	s.crossPackageDeps = deps
}

// SetSymbolDeps stores symbol-level dependencies for target detail queries
func (s *Server) SetSymbolDeps(deps []symbols.SymbolDependency) {
	s.symbolDeps = deps
}

func (s *Server) setupRoutes() {
	// API routes - more specific routes must come first
	s.router.HandleFunc("/api/analysis", s.handleAnalysis).Methods("GET")
	s.router.HandleFunc("/api/target/{label:.*}/graph", s.handleTargetGraph).Methods("GET")
	s.router.HandleFunc("/api/target/{label:.*}", s.handleTargetDetails).Methods("GET")

	// Serve static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}
	s.router.PathPrefix("/").Handler(http.FileServer(http.FS(staticFS)))
}

func (s *Server) handleAnalysis(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.analysisData == nil {
		http.Error(w, "No analysis data available", http.StatusServiceUnavailable)
		return
	}

	json.NewEncoder(w).Encode(s.analysisData)
}

func (s *Server) handleTargetDetails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	targetLabel := vars["label"]

	// HTTP path normalization strips leading //, so restore it if missing
	if len(targetLabel) > 0 && targetLabel[0] != '/' {
		targetLabel = "//" + targetLabel
	}

	if s.fileGraph == nil {
		http.Error(w, "File graph not available", http.StatusServiceUnavailable)
		return
	}

	// Get target file details
	details := analysis.GetTargetFileDetails(targetLabel, s.fileGraph, s.crossPackageDeps)

	json.NewEncoder(w).Encode(details)
}

func (s *Server) handleTargetGraph(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	targetLabel := vars["label"]

	// HTTP path normalization strips leading //, so restore it if missing
	if len(targetLabel) > 0 && targetLabel[0] != '/' {
		targetLabel = "//" + targetLabel
	}

	if s.fileGraph == nil {
		http.Error(w, "File graph not available", http.StatusServiceUnavailable)
		return
	}

	// Get target file details first
	details := analysis.GetTargetFileDetails(targetLabel, s.fileGraph, s.crossPackageDeps)

	// Build file-level graph for this target
	graphData := buildFileGraphData(targetLabel, details, s.fileGraph, s.symbolDeps)

	json.NewEncoder(w).Encode(graphData)
}

// buildFileGraphData creates a graph visualization for files within and connected to a target
func buildFileGraphData(targetLabel string, details *analysis.TargetFileDetails, fileGraph *graph.FileGraph, symbolDeps []symbols.SymbolDependency) *GraphData {
	graphData := &GraphData{
		Nodes: make([]GraphNode, 0),
		Edges: make([]GraphEdge, 0),
	}

	// Track which targets we've seen for creating parent nodes
	targetParents := make(map[string]bool)

	// Create parent node for the current target
	currentTargetParent := "parent-" + targetLabel
	targetParents[currentTargetParent] = true
	graphData.Nodes = append(graphData.Nodes, GraphNode{
		ID:    currentTargetParent,
		Label: targetLabel,
		Type:  "target-group",
	})

	// Create nodes for all files in this target
	filesInTarget := make(map[string]bool)
	for _, file := range details.Files {
		filesInTarget[file.Path] = true
		graphData.Nodes = append(graphData.Nodes, GraphNode{
			ID:     file.Path,
			Label:  getFileName(file.Path),
			Type:   file.Type, // "source" or "header"
			Parent: currentTargetParent,
		})
	}

	// Create nodes for external files (files from other targets that this target depends on)
	externalFiles := make(map[string]bool)

	// Helper function to ensure parent node exists for a target
	ensureParentNode := func(targetLabel string) string {
		parentID := "parent-" + targetLabel
		if !targetParents[parentID] {
			targetParents[parentID] = true
			graphData.Nodes = append(graphData.Nodes, GraphNode{
				ID:    parentID,
				Label: targetLabel,
				Type:  "target-group",
			})
		}
		return parentID
	}

	// Add outgoing dependency files (files this target depends on)
	for _, dep := range details.OutgoingFileDeps {
		if !filesInTarget[dep.TargetFile] && !externalFiles[dep.TargetFile] {
			externalFiles[dep.TargetFile] = true
			parentID := ensureParentNode(dep.TargetTarget)
			graphData.Nodes = append(graphData.Nodes, GraphNode{
				ID:     dep.TargetFile,
				Label:  getFileName(dep.TargetFile),
				Type:   getFileType(dep.TargetFile),
				Parent: parentID,
			})
		}
	}

	// Add incoming dependency files (files from other targets that depend on this target)
	for _, dep := range details.IncomingFileDeps {
		if !filesInTarget[dep.SourceFile] && !externalFiles[dep.SourceFile] {
			externalFiles[dep.SourceFile] = true
			parentID := ensureParentNode(dep.SourceTarget)
			graphData.Nodes = append(graphData.Nodes, GraphNode{
				ID:     dep.SourceFile,
				Label:  getFileName(dep.SourceFile),
				Type:   getFileType(dep.SourceFile),
				Parent: parentID,
			})
		}
	}

	// Add internal edges (dependencies within this target from .d files)
	for _, sourceNode := range fileGraph.Nodes() {
		if !filesInTarget[sourceNode.Path] {
			continue
		}

		// Get all dependencies from this file
		deps := fileGraph.GetDependencies(sourceNode.Path)
		for _, targetPath := range deps {
			if filesInTarget[targetPath] {
				// Internal dependency
				graphData.Edges = append(graphData.Edges, GraphEdge{
					Source: sourceNode.Path,
					Target: targetPath,
					Type:   "file",
				})
			}
		}
	}

	// Add cross-target edges (outgoing) from .d files
	for _, dep := range details.OutgoingFileDeps {
		graphData.Edges = append(graphData.Edges, GraphEdge{
			Source: dep.SourceFile,
			Target: dep.TargetFile,
			Type:   "file",
		})
	}

	// Add cross-target edges (incoming) from .d files
	for _, dep := range details.IncomingFileDeps {
		graphData.Edges = append(graphData.Edges, GraphEdge{
			Source: dep.SourceFile,
			Target: dep.TargetFile,
			Type:   "file",
		})
	}

	// Add symbol-level edges and nodes for symbol-referenced files
	if symbolDeps != nil {
		// First pass: add any missing nodes for source files referenced by symbols
		for _, symDep := range symbolDeps {
			// Check if this symbol dependency involves files in the current target
			sourceInTarget := filesInTarget[symDep.SourceFile]
			targetInTarget := filesInTarget[symDep.TargetFile]

			// If source is in target and target is not, add target as external node
			if sourceInTarget && !targetInTarget && !externalFiles[symDep.TargetFile] {
				externalFiles[symDep.TargetFile] = true
				parentID := ensureParentNode(symDep.TargetTarget)
				graphData.Nodes = append(graphData.Nodes, GraphNode{
					ID:     symDep.TargetFile,
					Label:  getFileName(symDep.TargetFile),
					Type:   getFileType(symDep.TargetFile),
					Parent: parentID,
				})
			}

			// If target is in target and source is not, add source as external node
			if targetInTarget && !sourceInTarget && !externalFiles[symDep.SourceFile] {
				externalFiles[symDep.SourceFile] = true
				parentID := ensureParentNode(symDep.SourceTarget)
				graphData.Nodes = append(graphData.Nodes, GraphNode{
					ID:     symDep.SourceFile,
					Label:  getFileName(symDep.SourceFile),
					Type:   getFileType(symDep.SourceFile),
					Parent: parentID,
				})
			}
		}

		// Second pass: deduplicate and add symbol edges
		// Map of (source, target, linkage) -> set of unique symbols
		type edgeKey struct {
			source  string
			target  string
			linkage string
		}
		symbolEdgeMap := make(map[edgeKey]map[string]bool)

		for _, symDep := range symbolDeps {
			// Include symbol edges that involve files in this target
			sourceInTarget := filesInTarget[symDep.SourceFile] || externalFiles[symDep.SourceFile]
			targetInTarget := filesInTarget[symDep.TargetFile] || externalFiles[symDep.TargetFile]

			if sourceInTarget && targetInTarget {
				key := edgeKey{
					source:  symDep.SourceFile,
					target:  symDep.TargetFile,
					linkage: string(symDep.Linkage),
				}
				// Initialize the symbol set if needed
				if symbolEdgeMap[key] == nil {
					symbolEdgeMap[key] = make(map[string]bool)
				}
				// Add symbol to the set (automatically deduplicates)
				symbolEdgeMap[key][symDep.Symbol] = true
			}
		}

		// Create deduplicated edges with combined symbol lists
		for key, symbolSet := range symbolEdgeMap {
			// Convert set to sorted list for consistent display
			symbols := make([]string, 0, len(symbolSet))
			for symbol := range symbolSet {
				symbols = append(symbols, symbol)
			}

			graphData.Edges = append(graphData.Edges, GraphEdge{
				Source:  key.source,
				Target:  key.target,
				Type:    "symbol",
				Linkage: key.linkage,
				Symbols: symbols,
			})
		}
	}

	return graphData
}

// getFileName extracts just the filename from a path
func getFileName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

// getFileType determines if a file is a source or header file
func getFileType(path string) string {
	// Check file extension
	if len(path) > 2 {
		ext := path[len(path)-2:]
		if ext == ".h" {
			return "header"
		}
	}
	if len(path) > 3 {
		ext := path[len(path)-3:]
		if ext == ".cc" || ext == ".cpp" {
			return "source"
		}
	}
	if len(path) > 4 {
		ext := path[len(path)-4:]
		if ext == ".hpp" {
			return "header"
		}
	}
	// Default to source for external files
	return "source"
}

// Start starts the web server on the specified port
func (s *Server) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting web server on http://localhost%s", addr)
	return http.ListenAndServe(addr, s.router)
}
