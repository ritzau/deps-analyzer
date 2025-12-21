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
)

//go:embed static/*
var staticFiles embed.FS

// GraphNode represents a node in the dependency graph
type GraphNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"` // "cc_library", "cc_binary", etc.
}

// GraphEdge represents an edge in the dependency graph
type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
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

func (s *Server) setupRoutes() {
	// API routes
	s.router.HandleFunc("/api/analysis", s.handleAnalysis).Methods("GET")
	s.router.HandleFunc("/api/target/{label:.*}", s.handleTargetDetails).Methods("GET")
	s.router.HandleFunc("/api/target/{label:.*}/graph", s.handleTargetGraph).Methods("GET")

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
	graphData := buildFileGraphData(targetLabel, details, s.fileGraph)

	json.NewEncoder(w).Encode(graphData)
}

// buildFileGraphData creates a graph visualization for files within and connected to a target
func buildFileGraphData(targetLabel string, details *analysis.TargetFileDetails, fileGraph *graph.FileGraph) *GraphData {
	graphData := &GraphData{
		Nodes: make([]GraphNode, 0),
		Edges: make([]GraphEdge, 0),
	}

	// Create nodes for all files in this target
	filesInTarget := make(map[string]bool)
	for _, file := range details.Files {
		filesInTarget[file.Path] = true
		graphData.Nodes = append(graphData.Nodes, GraphNode{
			ID:    file.Path,
			Label: getFileName(file.Path),
			Type:  file.Type, // "source" or "header"
		})
	}

	// Create nodes for external files (files from other targets that this target depends on)
	externalFiles := make(map[string]bool)

	// Add outgoing dependency files (files this target depends on)
	for _, dep := range details.OutgoingFileDeps {
		if !filesInTarget[dep.TargetFile] && !externalFiles[dep.TargetFile] {
			externalFiles[dep.TargetFile] = true
			graphData.Nodes = append(graphData.Nodes, GraphNode{
				ID:    dep.TargetFile,
				Label: getFileName(dep.TargetFile),
				Type:  "external",
			})
		}
	}

	// Add incoming dependency files (files from other targets that depend on this target)
	for _, dep := range details.IncomingFileDeps {
		if !filesInTarget[dep.SourceFile] && !externalFiles[dep.SourceFile] {
			externalFiles[dep.SourceFile] = true
			graphData.Nodes = append(graphData.Nodes, GraphNode{
				ID:    dep.SourceFile,
				Label: getFileName(dep.SourceFile),
				Type:  "external",
			})
		}
	}

	// Add internal edges (dependencies within this target)
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
				})
			}
		}
	}

	// Add cross-target edges (outgoing)
	for _, dep := range details.OutgoingFileDeps {
		graphData.Edges = append(graphData.Edges, GraphEdge{
			Source: dep.SourceFile,
			Target: dep.TargetFile,
		})
	}

	// Add cross-target edges (incoming)
	for _, dep := range details.IncomingFileDeps {
		graphData.Edges = append(graphData.Edges, GraphEdge{
			Source: dep.SourceFile,
			Target: dep.TargetFile,
		})
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

// Start starts the web server on the specified port
func (s *Server) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting web server on http://localhost%s", addr)
	return http.ListenAndServe(addr, s.router)
}
