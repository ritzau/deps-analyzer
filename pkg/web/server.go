package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/ritzau/deps-analyzer/pkg/binaries"
	"github.com/ritzau/deps-analyzer/pkg/model"
	"github.com/ritzau/deps-analyzer/pkg/pubsub"
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

// Server represents the web server
type Server struct {
	router    *mux.Router
	binaries  []*binaries.BinaryInfo
	module    *model.Module
	publisher pubsub.Publisher
}

// NewServer creates a new web server
func NewServer() *Server {
	ssePublisher := pubsub.NewSSEPublisher()

	// Configure topic buffering
	// workspace_status: buffer last 10 events, replay only last event to new subscribers
	ssePublisher.ConfigureTopic("workspace_status", pubsub.TopicConfig{
		BufferSize: 10,
		ReplayAll:  false, // Only send current state
	})

	// target_graph: buffer last 5 events, replay only last event
	ssePublisher.ConfigureTopic("target_graph", pubsub.TopicConfig{
		BufferSize: 5,
		ReplayAll:  false, // Only send current state
	})

	s := &Server{
		router:    mux.NewRouter(),
		publisher: ssePublisher,
	}
	s.setupRoutes()
	return s
}

// SetBinaries stores binary-level information
func (s *Server) SetBinaries(bins []*binaries.BinaryInfo) {
	s.binaries = bins
}

// SetModule stores the new Module data model
func (s *Server) SetModule(m *model.Module) {
	s.module = m
}

// PublishWorkspaceStatus publishes a workspace status event
func (s *Server) PublishWorkspaceStatus(state, message string, step, total int) error {
	status := pubsub.WorkspaceStatus{
		State:   state,
		Message: message,
		Step:    step,
		Total:   total,
	}
	return s.publisher.Publish("workspace_status", state, status)
}

// PublishTargetGraph publishes a target graph event
func (s *Server) PublishTargetGraph(eventType string, complete bool) error {
	var targetsCount, depsCount int
	if s.module != nil {
		targetsCount = len(s.module.Targets)
		depsCount = len(s.module.Dependencies)
	}

	data := pubsub.TargetGraphData{
		TargetsCount:      targetsCount,
		DependenciesCount: depsCount,
		Complete:          complete,
	}
	return s.publisher.Publish("target_graph", eventType, data)
}

func (s *Server) setupRoutes() {
	// SSE subscription endpoints
	s.router.HandleFunc("/api/subscribe/workspace_status", s.handleSubscribeWorkspaceStatus).Methods("GET")
	s.router.HandleFunc("/api/subscribe/target_graph", s.handleSubscribeTargetGraph).Methods("GET")

	// API routes - more specific routes must come first
	s.router.HandleFunc("/api/analysis", s.handleAnalysis).Methods("GET") // Legacy endpoint for UI polling
	s.router.HandleFunc("/api/module", s.handleModule).Methods("GET")
	s.router.HandleFunc("/api/module/graph", s.handleModuleGraph).Methods("GET")
	s.router.HandleFunc("/api/module/packages", s.handlePackages).Methods("GET")
	s.router.HandleFunc("/api/binaries", s.handleBinaries).Methods("GET")
	s.router.HandleFunc("/api/binaries/graph", s.handleBinaryGraph).Methods("GET")
	s.router.HandleFunc("/api/target/{label}/focused", s.handleTargetFocused).Methods("GET")

	// Serve static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}
	s.router.PathPrefix("/").Handler(http.FileServer(http.FS(staticFS)))
}

func (s *Server) handleSubscribeWorkspaceStatus(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*") // CORS support

	// Send initial comment to establish connection (Safari compatibility)
	fmt.Fprintf(w, ": connected\n\n")
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Create subscription
	sub, err := s.publisher.Subscribe(r.Context(), "workspace_status")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer sub.Close()

	// Stream events
	for event := range sub.Events() {
		if err := pubsub.WriteSSE(w, event); err != nil {
			log.Printf("Error writing SSE event: %v", err)
			return
		}
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}
}

func (s *Server) handleSubscribeTargetGraph(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*") // CORS support

	// Send initial comment to establish connection (Safari compatibility)
	fmt.Fprintf(w, ": connected\n\n")
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Create subscription
	sub, err := s.publisher.Subscribe(r.Context(), "target_graph")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer sub.Close()

	// Stream events
	for event := range sub.Events() {
		if err := pubsub.WriteSSE(w, event); err != nil {
			log.Printf("Error writing SSE event: %v", err)
			return
		}
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}
}

func (s *Server) handleBinaries(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.binaries == nil {
		json.NewEncoder(w).Encode([]*binaries.BinaryInfo{})
		return
	}

	json.NewEncoder(w).Encode(s.binaries)
}

func (s *Server) handleAnalysis(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Legacy endpoint for UI polling - returns Module data in expected format
	// Since we now do analysis in one go, we always report as complete (step 4)
	response := map[string]interface{}{
		"analysisStep":     4, // Always complete
		"graph":            &GraphData{Nodes: []GraphNode{}, Edges: []GraphEdge{}},
		"crossPackageDeps": []interface{}{},
		"fileCycles":       []interface{}{},
	}

	if s.module != nil {
		// Convert Module to graph format
		response["graph"] = buildModuleGraphData(s.module)
		// Convert package dependencies
		response["crossPackageDeps"] = s.module.GetAllPackageDependencies()
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleBinaryGraph(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.binaries == nil {
		json.NewEncoder(w).Encode(&GraphData{
			Nodes: []GraphNode{},
			Edges: []GraphEdge{},
		})
		return
	}

	// Build binary-level graph
	graphData := buildBinaryGraphData(s.binaries)
	json.NewEncoder(w).Encode(graphData)
}

func (s *Server) handleModule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.module == nil {
		http.Error(w, "Module data not available", http.StatusServiceUnavailable)
		return
	}

	json.NewEncoder(w).Encode(s.module)
}

func (s *Server) handleModuleGraph(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.module == nil {
		json.NewEncoder(w).Encode(&GraphData{
			Nodes: []GraphNode{},
			Edges: []GraphEdge{},
		})
		return
	}

	// Build target-level graph from module
	graphData := buildModuleGraphData(s.module)
	json.NewEncoder(w).Encode(graphData)
}

func (s *Server) handlePackages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.module == nil {
		json.NewEncoder(w).Encode([]model.PackageDependency{})
		return
	}

	// Get all package dependencies
	pkgDeps := s.module.GetAllPackageDependencies()
	json.NewEncoder(w).Encode(pkgDeps)
}

func (s *Server) handleTargetFocused(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.module == nil {
		http.Error(w, "Module data not available", http.StatusServiceUnavailable)
		return
	}

	// Get target label from URL path
	vars := mux.Vars(r)
	targetLabel := vars["label"]
	if targetLabel == "" {
		http.Error(w, "Target label required", http.StatusBadRequest)
		return
	}

	// Ensure label starts with //
	if !strings.HasPrefix(targetLabel, "//") {
		targetLabel = "//" + targetLabel
	}

	// Find the target
	target, exists := s.module.Targets[targetLabel]
	if !exists {
		http.Error(w, fmt.Sprintf("Target not found: %s", targetLabel), http.StatusNotFound)
		return
	}

	// Build focused graph data
	graphData := buildTargetFocusedGraph(s.module, target)
	json.NewEncoder(w).Encode(graphData)
}

// buildBinaryGraphData creates a graph visualization for binaries and their shared library dependencies
func buildBinaryGraphData(binaryInfos []*binaries.BinaryInfo) *GraphData {
	graphData := &GraphData{
		Nodes: make([]GraphNode, 0),
		Edges: make([]GraphEdge, 0),
	}

	// Create nodes for all binaries
	for _, bin := range binaryInfos {
		nodeType := bin.Kind
		// Use specific type for binaries vs shared libraries
		if bin.Kind == "cc_binary" {
			nodeType = "cc_binary"
		} else if bin.Kind == "cc_shared_library" {
			nodeType = "cc_shared_library"
		}

		graphData.Nodes = append(graphData.Nodes, GraphNode{
			ID:    bin.Label,
			Label: bin.Label,
			Type:  nodeType,
		})
	}

	// Create nodes for system libraries
	systemLibs := make(map[string]bool)
	for _, bin := range binaryInfos {
		for _, sysLib := range bin.SystemLibraries {
			if !systemLibs[sysLib] {
				systemLibs[sysLib] = true
				graphData.Nodes = append(graphData.Nodes, GraphNode{
					ID:    "system:" + sysLib,
					Label: sysLib,
					Type:  "system_library",
				})
			}
		}
	}

	// Create edges for dependencies
	for _, bin := range binaryInfos {
		// Dynamic deps (linked shared libraries)
		for _, dep := range bin.DynamicDeps {
			graphData.Edges = append(graphData.Edges, GraphEdge{
				Source:  bin.Label,
				Target:  dep,
				Type:    "dynamic_link",
				Symbols: []string{},
			})
		}

		// Data deps (runtime-loaded shared libraries)
		for _, dep := range bin.DataDeps {
			graphData.Edges = append(graphData.Edges, GraphEdge{
				Source:  bin.Label,
				Target:  dep,
				Type:    "data_dependency",
				Symbols: []string{},
			})
		}

		// System library dependencies
		for _, sysLib := range bin.SystemLibraries {
			graphData.Edges = append(graphData.Edges, GraphEdge{
				Source:  bin.Label,
				Target:  "system:" + sysLib,
				Type:    "system_link",
				Symbols: []string{},
			})
		}
	}

	return graphData
}

// TODO: Bring back file-level graph visualization using Module compile dependencies
// This would show files within a target and their compile-time dependencies to other targets

// buildModuleGraphData creates a graph visualization from the Module model
func buildModuleGraphData(module *model.Module) *GraphData {
	graphData := &GraphData{
		Nodes: make([]GraphNode, 0),
		Edges: make([]GraphEdge, 0),
	}

	// Create nodes for all targets
	for _, target := range module.Targets {
		graphData.Nodes = append(graphData.Nodes, GraphNode{
			ID:    target.Label,
			Label: target.Label,
			Type:  string(target.Kind),
		})
	}

	// Create edges for all dependencies, colored by type
	for _, dep := range module.Dependencies {
		graphData.Edges = append(graphData.Edges, GraphEdge{
			Source:  dep.From,
			Target:  dep.To,
			Type:    string(dep.Type),
			Symbols: []string{},
		})
	}

	return graphData
}

// buildTargetFocusedGraph creates a focused view of a target showing:
// - The focused target with all its files (sources and headers)
// - Incoming dependencies (targets that depend on this one) with their files
// - Outgoing dependencies (targets this one depends on) with their files
// - All compile-time and link-time dependencies between files and targets
func buildTargetFocusedGraph(module *model.Module, focusedTarget *model.Target) *GraphData {
	graphData := &GraphData{
		Nodes: make([]GraphNode, 0),
		Edges: make([]GraphEdge, 0),
	}

	// Track which targets are relevant (connect to/from focused target)
	relevantTargets := make(map[string]bool)
	relevantTargets[focusedTarget.Label] = true

	// Find all incoming dependencies (targets that depend on focused target)
	incomingDeps := make(map[string]bool)
	for _, dep := range module.Dependencies {
		if dep.To == focusedTarget.Label {
			incomingDeps[dep.From] = true
			relevantTargets[dep.From] = true
		}
	}

	// Find all outgoing dependencies (targets that focused target depends on)
	outgoingDeps := make(map[string]bool)
	for _, dep := range module.Dependencies {
		if dep.From == focusedTarget.Label {
			outgoingDeps[dep.To] = true
			relevantTargets[dep.To] = true
		}
	}

	// Helper function to add target with its files as a compound node
	addTargetWithFiles := func(target *model.Target, typeSuffix string) {
		// Add parent/container node for the target
		parentID := "parent-" + target.Label
		graphData.Nodes = append(graphData.Nodes, GraphNode{
			ID:    parentID,
			Label: target.Label,
			Type:  "target-group",
		})

		// Add file nodes (sources and headers) as children
		for _, source := range target.Sources {
			fileID := target.Label + ":file:" + source
			graphData.Nodes = append(graphData.Nodes, GraphNode{
				ID:     fileID,
				Label:  getFileName(source),
				Type:   "source" + typeSuffix,
				Parent: parentID,
			})
		}
		for _, header := range target.Headers {
			fileID := target.Label + ":file:" + header
			graphData.Nodes = append(graphData.Nodes, GraphNode{
				ID:     fileID,
				Label:  getFileName(header),
				Type:   "header" + typeSuffix,
				Parent: parentID,
			})
		}
	}

	// Add the focused target with its files
	addTargetWithFiles(focusedTarget, "_focused")

	// Add incoming dependency targets with their files
	for targetLabel := range incomingDeps {
		if target, exists := module.Targets[targetLabel]; exists {
			addTargetWithFiles(target, "_incoming")
		}
	}

	// Add outgoing dependency targets with their files
	for targetLabel := range outgoingDeps {
		if target, exists := module.Targets[targetLabel]; exists {
			addTargetWithFiles(target, "_outgoing")
		}
	}

	// Add edges - only those that connect to/from the focused target
	// Edges now connect to the parent node IDs (with "parent-" prefix)
	for _, dep := range module.Dependencies {
		// Include edge if it connects to or from the focused target
		if dep.From == focusedTarget.Label || dep.To == focusedTarget.Label {
			// Use parent- prefix for compound node IDs
			sourceID := "parent-" + dep.From
			targetID := "parent-" + dep.To

			graphData.Edges = append(graphData.Edges, GraphEdge{
				Source:  sourceID,
				Target:  targetID,
				Type:    string(dep.Type),
				Linkage: string(dep.Type),
				Symbols: []string{},
			})
		}
	}

	return graphData
}

// getFileName extracts the file name from a full path
func getFileName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}

// Start starts the web server on the specified port
func (s *Server) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting web server on http://localhost%s", addr)
	return http.ListenAndServe(addr, s.router)
}
