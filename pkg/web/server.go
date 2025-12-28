package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/ritzau/deps-analyzer/pkg/binaries"
	"github.com/ritzau/deps-analyzer/pkg/deps"
	"github.com/ritzau/deps-analyzer/pkg/lens"
	"github.com/ritzau/deps-analyzer/pkg/model"
	"github.com/ritzau/deps-analyzer/pkg/pubsub"
	"github.com/ritzau/deps-analyzer/pkg/symbols"
)

//go:embed static/*
var staticFiles embed.FS

// GraphNode represents a node in the dependency graph
type GraphNode struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Type     string `json:"type"`     // "cc_library", "cc_binary", "source", "header", "external"
	Parent   string `json:"parent"`   // Parent node ID for grouping (optional)
	IsPublic bool   `json:"isPublic"` // Whether target has public visibility
}

// GraphEdge represents an edge in the dependency graph
type GraphEdge struct {
	Source      string            `json:"source"`
	Target      string            `json:"target"`
	Type        string            `json:"type"`        // "file" (from .d files) or "symbol" (from nm)
	Linkage     string            `json:"linkage"`     // For symbol edges: "static", "dynamic", or "cross"
	Symbols     []string          `json:"symbols"`     // For symbol edges: list of symbol names
	SourceLabel string            `json:"sourceLabel"` // Human-readable label for source node
	TargetLabel string            `json:"targetLabel"` // Human-readable label for target node
	FileDetails map[string]string `json:"fileDetails"` // File-level details: source file -> target file(s)
}

// GraphData holds the dependency graph for visualization
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// Server represents the web server
type Server struct {
	router         *mux.Router
	binaries       []*binaries.BinaryInfo
	module         *model.Module
	publisher      pubsub.Publisher
	fileDeps       []*deps.FileDependency       // Compile-time file dependencies from .d files
	symbolDeps     []symbols.SymbolDependency   // Link-time symbol dependencies from nm
	fileToTarget   map[string]string            // Maps file paths to target labels
	uncoveredFiles []string                     // Files not included in any target
	watching       bool                         // File watching active
	mu             sync.RWMutex                 // Protect all state from concurrent access
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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.binaries = bins
}

// SetModule stores the new Module data model
func (s *Server) SetModule(m *model.Module) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.module = m
}

// GetModule retrieves the current Module data model
func (s *Server) GetModule() *model.Module {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.module
}

// SetFileDependencies stores file-level compile dependencies from .d files
func (s *Server) SetFileDependencies(fileDeps []*deps.FileDependency) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fileDeps = fileDeps
}

// SetSymbolDependencies stores file-level symbol dependencies from nm analysis
func (s *Server) SetSymbolDependencies(symbolDeps []symbols.SymbolDependency) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.symbolDeps = symbolDeps
}

// SetFileToTargetMap stores the mapping from file paths to target labels
func (s *Server) SetFileToTargetMap(fileToTarget map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fileToTarget = fileToTarget
}

// SetUncoveredFiles stores files that are not included in any target
func (s *Server) SetUncoveredFiles(files []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.uncoveredFiles = files
}

// SetWatching sets the file watching state
func (s *Server) SetWatching(watching bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.watching = watching
}

// PublishWorkspaceStatus publishes a workspace status event
func (s *Server) PublishWorkspaceStatus(state, message string, step, total int) error {
	s.mu.RLock()
	watching := s.watching
	s.mu.RUnlock()

	status := pubsub.WorkspaceStatus{
		State:    state,
		Message:  message,
		Step:     step,
		Total:    total,
		Watching: watching,
		Reason:   "",
	}
	return s.publisher.Publish("workspace_status", state, status)
}

// PublishWorkspaceStatusWithReason publishes a workspace status event with a reason
func (s *Server) PublishWorkspaceStatusWithReason(state, message, reason string, step, total int) error {
	s.mu.RLock()
	watching := s.watching
	s.mu.RUnlock()

	status := pubsub.WorkspaceStatus{
		State:    state,
		Message:  message,
		Step:     step,
		Total:    total,
		Watching: watching,
		Reason:   reason,
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
	s.router.HandleFunc("/api/module", s.handleModule).Methods("GET", "HEAD") // HEAD for health checks
	s.router.HandleFunc("/api/module/graph", s.handleModuleGraph).Methods("GET")
	s.router.HandleFunc("/api/module/graph/lens", s.handleModuleGraphWithLens).Methods("POST")
	s.router.HandleFunc("/api/binaries", s.handleBinaries).Methods("GET")
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

	// Build target-level graph from module with file-level details
	graphData := buildModuleGraphData(s.module, s.fileDeps, s.symbolDeps, s.fileToTarget, s.uncoveredFiles)
	json.NewEncoder(w).Encode(graphData)
}

// LensRenderRequest represents the request body for lens rendering
type LensRenderRequest struct {
	DefaultLens    *lens.LensConfig          `json:"defaultLens"`
	FocusLens      *lens.LensConfig          `json:"focusLens"`
	FocusedNodes   []string                  `json:"focusedNodes"`
	ManualOverrides map[string]lens.ManualOverride `json:"manualOverrides"`
}

func (s *Server) handleModuleGraphWithLens(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.module == nil {
		json.NewEncoder(w).Encode(&GraphData{
			Nodes: []GraphNode{},
			Edges: []GraphEdge{},
		})
		return
	}

	// Parse lens configuration from request body
	var req LensRenderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate that we have lens configurations
	if req.DefaultLens == nil || req.FocusLens == nil {
		http.Error(w, "Missing required lens configurations", http.StatusBadRequest)
		return
	}

	// Build raw graph data
	rawGraphData := buildModuleGraphData(s.module, s.fileDeps, s.symbolDeps, s.fileToTarget, s.uncoveredFiles)

	// Convert web.GraphData to lens.GraphData
	lensGraphData := convertToLensGraphData(rawGraphData)

	// Apply lens rendering
	manualOverrides := req.ManualOverrides
	if manualOverrides == nil {
		manualOverrides = make(map[string]lens.ManualOverride)
	}

	renderedGraph, err := lens.RenderGraph(lensGraphData, req.DefaultLens, req.FocusLens, req.FocusedNodes, manualOverrides)
	if err != nil {
		http.Error(w, fmt.Sprintf("Lens rendering failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert lens.GraphData back to web.GraphData
	resultGraphData := convertFromLensGraphData(renderedGraph, rawGraphData)

	json.NewEncoder(w).Encode(resultGraphData)
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

	// Build focused graph data with file-level dependencies
	graphData := buildTargetFocusedGraph(s.module, target, s.fileDeps, s.symbolDeps, s.fileToTarget, s.uncoveredFiles)
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
func buildModuleGraphData(module *model.Module, fileDeps []*deps.FileDependency, symbolDeps []symbols.SymbolDependency, fileToTarget map[string]string, uncoveredFiles []string) *GraphData {
	graphData := &GraphData{
		Nodes: make([]GraphNode, 0),
		Edges: make([]GraphEdge, 0),
	}

	// Create nodes for all targets
	for _, target := range module.Targets {
		graphData.Nodes = append(graphData.Nodes, GraphNode{
			ID:       target.Label,
			Label:    target.Label,
			Type:     string(target.Kind),
			IsPublic: target.IsPublic(),
		})

		// Create file nodes as children of this target
		// Build a set of all files from this target (for later edge matching)
		allFiles := make(map[string]bool)
		for _, source := range target.Sources {
			allFiles[source] = true
		}
		for _, header := range target.Headers {
			allFiles[header] = true
		}
	}

	// Create file nodes using the file-to-target mapping to ensure consistent IDs
	// This ensures file node IDs match what's used in edges
	createdFileNodes := make(map[string]bool)
	if fileToTarget != nil {
		for filePath, targetLabel := range fileToTarget {
			fileID := targetLabel + ":" + filePath

			// Skip if already created
			if createdFileNodes[fileID] {
				continue
			}
			createdFileNodes[fileID] = true

			// Extract just the filename for display
			filename := filePath
			// Remove package prefix if present (e.g., "graphics:" from "graphics:renderer.cc")
			if idx := strings.LastIndex(filename, ":"); idx >= 0 {
				filename = filename[idx+1:]
			}
			// Remove directory path
			if idx := strings.LastIndex(filename, "/"); idx >= 0 {
				filename = filename[idx+1:]
			}

			// Determine file type
			fileType := "source_file"
			if strings.HasSuffix(filePath, ".h") || strings.HasSuffix(filePath, ".hpp") {
				fileType = "header_file"
			}

			graphData.Nodes = append(graphData.Nodes, GraphNode{
				ID:     fileID,
				Label:  filename,
				Type:   fileType,
				Parent: targetLabel,
			})
		}
	}

	// Create file-to-file edges for compile dependencies (header includes)
	if fileDeps != nil && fileToTarget != nil {
		for _, fileDep := range fileDeps {
			sourceTarget, sourceOK := fileToTarget[fileDep.SourceFile]
			if !sourceOK {
				continue
			}

			sourceFileID := sourceTarget + ":" + fileDep.SourceFile
			sourceFileName := getFileName(fileDep.SourceFile)

			for _, depFile := range fileDep.Dependencies {
				targetTarget, targetOK := fileToTarget[depFile]
				if !targetOK {
					continue
				}

				targetFileID := targetTarget + ":" + depFile
				targetFileName := getFileName(depFile)

				// Create edge from source file to dependency file
				graphData.Edges = append(graphData.Edges, GraphEdge{
					Source: sourceFileID,
					Target: targetFileID,
					Type:   string(model.DependencyCompile),
					FileDetails: map[string]string{
						sourceFileName: targetFileName,
					},
				})
			}
		}
	}

	// Create file-to-file edges for symbol dependencies
	if symbolDeps != nil {
		// Group symbol deps by file pair
		type fileEdgeKey struct {
			sourceFile string
			targetFile string
		}
		symbolsByFilePair := make(map[fileEdgeKey][]string)

		for _, symDep := range symbolDeps {
			key := fileEdgeKey{
				sourceFile: symDep.SourceTarget + ":" + symDep.SourceFile,
				targetFile: symDep.TargetTarget + ":" + symDep.TargetFile,
			}
			symbolsByFilePair[key] = append(symbolsByFilePair[key], symDep.Symbol)
		}

		// Create edges with aggregated symbols
		for key, symbols := range symbolsByFilePair {
			graphData.Edges = append(graphData.Edges, GraphEdge{
				Source:  key.sourceFile,
				Target:  key.targetFile,
				Type:    string(model.DependencySymbol),
				Symbols: symbols,
			})
		}
	}

	// Track system libraries to avoid duplicates
	systemLibs := make(map[string]bool)

	// Add system library nodes and edges from linkopts
	for _, target := range module.Targets {
		if len(target.Linkopts) > 0 {
			for _, linkopt := range target.Linkopts {
				if strings.HasPrefix(linkopt, "-l") {
					libName := strings.TrimPrefix(linkopt, "-l")
					if libName != "" && !systemLibs[libName] {
						systemLibs[libName] = true
						// Add system library node
						graphData.Nodes = append(graphData.Nodes, GraphNode{
							ID:    "system:" + libName,
							Label: libName,
							Type:  "system_library",
						})
					}
				}
			}
		}
	}

	// Build a map to track file-level and symbol details for each target-level edge
	type edgeKey struct {
		from string
		to   string
	}
	edgeDetails := make(map[edgeKey]map[string][]string) // edgeKey -> (sourceFile -> []targetFiles)
	edgeSymbols := make(map[edgeKey]map[string]bool)    // edgeKey -> set of symbols

	// Aggregate compile dependencies (file-level header includes)
	if fileDeps != nil && fileToTarget != nil {
		for _, fileDep := range fileDeps {
			sourceTarget, sourceOK := fileToTarget[fileDep.SourceFile]
			if !sourceOK {
				continue
			}

			for _, depFile := range fileDep.Dependencies {
				targetTarget, targetOK := fileToTarget[depFile]
				if !targetOK || sourceTarget == targetTarget {
					continue // Skip if same target or unknown
				}

				key := edgeKey{from: sourceTarget, to: targetTarget}
				if edgeDetails[key] == nil {
					edgeDetails[key] = make(map[string][]string)
				}
				sourceFileName := getFileName(fileDep.SourceFile)
				targetFileName := getFileName(depFile)
				edgeDetails[key][sourceFileName] = append(edgeDetails[key][sourceFileName], targetFileName)
			}
		}
	}

	// Aggregate symbol dependencies
	if symbolDeps != nil {
		for _, symDep := range symbolDeps {
			if symDep.SourceTarget == symDep.TargetTarget {
				continue // Skip intra-target symbols
			}

			key := edgeKey{from: symDep.SourceTarget, to: symDep.TargetTarget}
			if edgeSymbols[key] == nil {
				edgeSymbols[key] = make(map[string]bool)
			}
			edgeSymbols[key][symDep.Symbol] = true
		}
	}

	// Create edges for all dependencies, colored by type
	for _, dep := range module.Dependencies {
		key := edgeKey{from: dep.From, to: dep.To}

		// Collect file details for this edge
		fileDetailsMap := make(map[string]string)
		if details, exists := edgeDetails[key]; exists {
			for sourceFile, targetFiles := range details {
				// Store as "source.cc" -> "header1.h, header2.h"
				fileDetailsMap[sourceFile] = strings.Join(targetFiles, ", ")
			}
		}

		// Collect symbols for this edge
		var symbols []string
		if symMap, exists := edgeSymbols[key]; exists {
			for sym := range symMap {
				symbols = append(symbols, sym)
			}
		}

		graphData.Edges = append(graphData.Edges, GraphEdge{
			Source:      dep.From,
			Target:      dep.To,
			Type:        string(dep.Type),
			Symbols:     symbols,
			SourceLabel: dep.From, // Use full label for module graph
			TargetLabel: dep.To,
			FileDetails: fileDetailsMap,
		})
	}

	// Add edges from targets to their system libraries
	for _, target := range module.Targets {
		if len(target.Linkopts) > 0 {
			for _, linkopt := range target.Linkopts {
				if strings.HasPrefix(linkopt, "-l") {
					libName := strings.TrimPrefix(linkopt, "-l")
					if libName != "" {
						graphData.Edges = append(graphData.Edges, GraphEdge{
							Source:      target.Label,
							Target:      "system:" + libName,
							Type:        "system_link",
							Linkage:     "system",
							Symbols:     []string{},
							SourceLabel: target.Label,
							TargetLabel: libName, // Just the library name for display
						})
					}
				}
			}
		}
	}

	// Add uncovered files as nodes (files not in any target)
	if uncoveredFiles != nil && len(uncoveredFiles) > 0 {
		for _, uncoveredFile := range uncoveredFiles {
			// Determine if source or header
			nodeType := "uncovered_source"
			if strings.HasSuffix(uncoveredFile, ".h") || strings.HasSuffix(uncoveredFile, ".hpp") {
				nodeType = "uncovered_header"
			}

			// Extract package from file path (e.g., "core/engine.cc" -> "core")
			packagePath := ""
			if idx := strings.LastIndex(uncoveredFile, "/"); idx >= 0 {
				packagePath = uncoveredFile[:idx]
			}

			// Create node ID and determine parent package
			fileID := "uncovered:" + uncoveredFile
			parentPackage := ""
			if packagePath != "" {
				parentPackage = "//" + packagePath
			}

			graphData.Nodes = append(graphData.Nodes, GraphNode{
				ID:     fileID,
				Label:  getFileName(uncoveredFile),
				Type:   nodeType,
				Parent: parentPackage, // Parent is the package, not a target
			})
		}
	}

	return graphData
}

// buildTargetFocusedGraph creates a focused view of a target showing:
// - The focused target with all its files (sources and headers)
// - Incoming dependencies (targets that depend on this one) with their files
// - Outgoing dependencies (targets this one depends on) with their files
// - All compile-time and link-time dependencies between files and targets
// - Uncovered files in the focused target's package
func buildTargetFocusedGraph(module *model.Module, focusedTarget *model.Target, fileDeps []*deps.FileDependency, symbolDeps []symbols.SymbolDependency, fileToTarget map[string]string, uncoveredFiles []string) *GraphData {
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

	// First, add parent nodes for all relevant targets (we'll add file nodes later after we know which have edges)
	addTargetParent := func(target *model.Target) {
		parentID := "parent-" + target.Label
		graphData.Nodes = append(graphData.Nodes, GraphNode{
			ID:    parentID,
			Label: target.Label,
			Type:  "target-group",
		})
	}

	// Add parent nodes for all relevant targets
	addTargetParent(focusedTarget)
	for targetLabel := range incomingDeps {
		if target, exists := module.Targets[targetLabel]; exists {
			addTargetParent(target)
		}
	}
	for targetLabel := range outgoingDeps {
		if target, exists := module.Targets[targetLabel]; exists {
			addTargetParent(target)
		}
	}

	// Track which files have edges (so we only show files that are connected)
	filesWithEdges := make(map[string]bool)

	// Add target-level edges - only those that connect to/from the focused target
	// Edges connect to the parent node IDs (with "parent-" prefix)
	for _, dep := range module.Dependencies {
		// Include edge if it connects to or from the focused target
		if dep.From == focusedTarget.Label || dep.To == focusedTarget.Label {
			// Use parent- prefix for compound node IDs
			sourceID := "parent-" + dep.From
			targetID := "parent-" + dep.To

			graphData.Edges = append(graphData.Edges, GraphEdge{
				Source:      sourceID,
				Target:      targetID,
				Type:        string(dep.Type),
				Linkage:     string(dep.Type),
				Symbols:     []string{},
				SourceLabel: dep.From,
				TargetLabel: dep.To,
			})
		}
	}

	// Add system library nodes and edges for the focused target
	if len(focusedTarget.Linkopts) > 0 {
		for _, linkopt := range focusedTarget.Linkopts {
			if strings.HasPrefix(linkopt, "-l") {
				libName := strings.TrimPrefix(linkopt, "-l")
				if libName != "" {
					// Add system library node
					libNodeID := "system:" + libName
					graphData.Nodes = append(graphData.Nodes, GraphNode{
						ID:    libNodeID,
						Label: libName,
						Type:  "system_library",
					})

					// Add edge from focused target to system library
					graphData.Edges = append(graphData.Edges, GraphEdge{
						Source:      "parent-" + focusedTarget.Label,
						Target:      libNodeID,
						Type:        "system_link",
						Linkage:     "system",
						Symbols:     []string{},
						SourceLabel: focusedTarget.Label,
						TargetLabel: libName,
					})
				}
			}
		}
	}

	// Add file-to-file edges from compile dependencies (.d files)
	// Build a reverse map from normalized paths to original source paths
	normalizedToOriginal := make(map[string]string)
	if module != nil {
		for _, target := range module.Targets {
			for _, src := range target.Sources {
				normalized := strings.ReplaceAll(strings.TrimPrefix(src, "//"), ":", "/")
				normalizedToOriginal[normalized] = src
			}
			for _, hdr := range target.Headers {
				normalized := strings.ReplaceAll(strings.TrimPrefix(hdr, "//"), ":", "/")
				normalizedToOriginal[normalized] = hdr
			}
		}
	}

	if fileDeps != nil && fileToTarget != nil {
		for _, fileDep := range fileDeps {
			// Find which target owns the source file
			sourceTarget, sourceOK := fileToTarget[fileDep.SourceFile]
			if !sourceOK || !relevantTargets[sourceTarget] {
				continue // Skip if source is not in a relevant target
			}

			// Get the original Bazel format for the source file
			sourceOriginal, ok := normalizedToOriginal[fileDep.SourceFile]
			if !ok {
				continue // Skip if we can't find the original format
			}

			// Process each header dependency
			for _, depFile := range fileDep.Dependencies {
				// Find which target owns the dependency file
				targetTarget, targetOK := fileToTarget[depFile]
				if !targetOK || !relevantTargets[targetTarget] {
					continue // Skip if target is not in a relevant target
				}

				// Only show edges where at least one end is in the focused target
				if sourceTarget != focusedTarget.Label && targetTarget != focusedTarget.Label {
					continue
				}

				// Get the original Bazel format for the dependency file
				depOriginal, ok := normalizedToOriginal[depFile]
				if !ok {
					continue // Skip if we can't find the original format
				}

				// Create file node IDs using original Bazel format
				// Source file ID format: targetLabel:file:bazelPath
				sourceFileID := sourceTarget + ":file:" + sourceOriginal
				targetFileID := targetTarget + ":file:" + depOriginal

				// Track that these files have edges
				filesWithEdges[sourceFileID] = true
				filesWithEdges[targetFileID] = true

				// Add compile dependency edge between files
				graphData.Edges = append(graphData.Edges, GraphEdge{
					Source:      sourceFileID,
					Target:      targetFileID,
					Type:        "compile",
					Linkage:     "compile",
					Symbols:     []string{},
					SourceLabel: getFileName(sourceOriginal),
					TargetLabel: getFileName(depOriginal),
				})
			}
		}
	}

	// Add file-to-file edges from symbol dependencies (nm analysis)
	// Use a map to deduplicate and aggregate symbols for the same edge
	type edgeKey struct {
		source  string
		target  string
		linkage string
	}
	symbolEdges := make(map[edgeKey]*GraphEdge)

	if symbolDeps != nil {
		for _, symDep := range symbolDeps {
			// Only include if both targets are relevant
			if !relevantTargets[symDep.SourceTarget] || !relevantTargets[symDep.TargetTarget] {
				continue
			}

			// Only show edges where at least one end is in the focused target
			if symDep.SourceTarget != focusedTarget.Label && symDep.TargetTarget != focusedTarget.Label {
				continue
			}

			// Get the original Bazel format for source and target files
			sourceOriginal, sourceOK := normalizedToOriginal[symDep.SourceFile]
			targetOriginal, targetOK := normalizedToOriginal[symDep.TargetFile]
			if !sourceOK || !targetOK {
				continue // Skip if we can't find the original format
			}

			// Create file node IDs using original Bazel format
			sourceFileID := symDep.SourceTarget + ":file:" + sourceOriginal
			targetFileID := symDep.TargetTarget + ":file:" + targetOriginal

			// Track that these files have edges
			filesWithEdges[sourceFileID] = true
			filesWithEdges[targetFileID] = true

			// Create edge key for deduplication
			key := edgeKey{
				source:  sourceFileID,
				target:  targetFileID,
				linkage: string(symDep.Linkage),
			}

			// Get or create edge
			edge, exists := symbolEdges[key]
			if !exists {
				edge = &GraphEdge{
					Source:      sourceFileID,
					Target:      targetFileID,
					Type:        "symbol",
					Linkage:     string(symDep.Linkage),
					Symbols:     []string{},
					SourceLabel: getFileName(sourceOriginal),
					TargetLabel: getFileName(targetOriginal),
				}
				symbolEdges[key] = edge
			}

			// Add symbol to the edge (avoiding duplicates)
			symbolExists := false
			for _, existingSym := range edge.Symbols {
				if existingSym == symDep.Symbol {
					symbolExists = true
					break
				}
			}
			if !symbolExists {
				edge.Symbols = append(edge.Symbols, symDep.Symbol)
			}
		}
	}

	// Add deduplicated symbol edges to graph
	for _, edge := range symbolEdges {
		graphData.Edges = append(graphData.Edges, *edge)
	}

	// Now add file nodes - only for files that have edges OR are in the focused target
	addFileNodes := func(target *model.Target, typeSuffix string) {
		parentID := "parent-" + target.Label
		isFocused := target.Label == focusedTarget.Label

		// Add source file nodes
		for _, source := range target.Sources {
			fileID := target.Label + ":file:" + source
			// Only add if file has edges OR is in focused target
			if isFocused || filesWithEdges[fileID] {
				graphData.Nodes = append(graphData.Nodes, GraphNode{
					ID:     fileID,
					Label:  getFileName(source),
					Type:   "source" + typeSuffix,
					Parent: parentID,
				})
			}
		}

		// Add header file nodes
		for _, header := range target.Headers {
			fileID := target.Label + ":file:" + header
			// Only add if file has edges OR is in focused target
			if isFocused || filesWithEdges[fileID] {
				graphData.Nodes = append(graphData.Nodes, GraphNode{
					ID:     fileID,
					Label:  getFileName(header),
					Type:   "header" + typeSuffix,
					Parent: parentID,
				})
			}
		}
	}

	// Add file nodes for focused target
	addFileNodes(focusedTarget, "_focused")

	// Add uncovered files in the focused target's package
	focusedPackage := focusedTarget.Package
	parentID := "parent-" + focusedTarget.Label
	for _, uncoveredFile := range uncoveredFiles {
		// Check if file is in the focused package
		filePath := uncoveredFile
		if strings.HasPrefix(filePath, strings.TrimPrefix(focusedPackage, "//")+"/") {
			// Determine if source or header
			nodeType := "uncovered_source"
			if strings.HasSuffix(filePath, ".h") || strings.HasSuffix(filePath, ".hpp") {
				nodeType = "uncovered_header"
			}

			// Create node with warning styling
			graphData.Nodes = append(graphData.Nodes, GraphNode{
				ID:     "uncovered:" + uncoveredFile,
				Label:  getFileName(uncoveredFile),
				Type:   nodeType,
				Parent: parentID, // Group under focused target
			})
		}
	}

	// Add file nodes for incoming dependency targets
	for targetLabel := range incomingDeps {
		if target, exists := module.Targets[targetLabel]; exists {
			addFileNodes(target, "_incoming")
		}
	}

	// Add file nodes for outgoing dependency targets
	for targetLabel := range outgoingDeps {
		if target, exists := module.Targets[targetLabel]; exists {
			addFileNodes(target, "_outgoing")
		}
	}

	return graphData
}

// getFileName extracts the file name from a full path or Bazel label
func getFileName(path string) string {
	// Handle Bazel label format: //package:file.cc
	if strings.Contains(path, ":") {
		parts := strings.Split(path, ":")
		if len(parts) > 1 {
			path = parts[len(parts)-1]
		}
	}

	// Extract just the filename from path
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}

// convertToLensGraphData converts web.GraphData to lens.GraphData
func convertToLensGraphData(webGraph *GraphData) *lens.GraphData {
	lensNodes := make([]lens.GraphNode, len(webGraph.Nodes))
	for i, node := range webGraph.Nodes {
		lensNodes[i] = lens.GraphNode{
			ID:     node.ID,
			Label:  node.Label,
			Type:   node.Type,
			Parent: node.Parent,
		}
	}

	lensEdges := make([]lens.GraphEdge, len(webGraph.Edges))
	for i, edge := range webGraph.Edges {
		lensEdges[i] = lens.GraphEdge{
			Source: edge.Source,
			Target: edge.Target,
			Type:   edge.Type,
		}
	}

	return &lens.GraphData{
		Nodes: lensNodes,
		Edges: lensEdges,
	}
}

// convertFromLensGraphData converts lens.GraphData back to web.GraphData
// It enriches the lens-rendered graph with metadata from the original raw graph
func convertFromLensGraphData(lensGraph *lens.GraphData, rawGraph *GraphData) *GraphData {
	// Create lookup map for raw graph nodes to get additional metadata
	rawNodeMap := make(map[string]GraphNode)
	for _, node := range rawGraph.Nodes {
		rawNodeMap[node.ID] = node
	}

	// Create lookup map for raw graph edges to get additional metadata
	type edgeKey struct {
		source string
		target string
		edgeType string
	}
	rawEdgeMap := make(map[edgeKey]GraphEdge)
	for _, edge := range rawGraph.Edges {
		key := edgeKey{edge.Source, edge.Target, edge.Type}
		rawEdgeMap[key] = edge
	}

	// Convert nodes, preserving metadata
	webNodes := make([]GraphNode, len(lensGraph.Nodes))
	for i, node := range lensGraph.Nodes {
		webNodes[i] = GraphNode{
			ID:     node.ID,
			Label:  node.Label,
			Type:   node.Type,
			Parent: node.Parent,
		}

		// Copy additional metadata from raw graph if available
		if rawNode, exists := rawNodeMap[node.ID]; exists {
			webNodes[i].IsPublic = rawNode.IsPublic
		}
	}

	// Convert edges, preserving metadata
	webEdges := make([]GraphEdge, len(lensGraph.Edges))
	for i, edge := range lensGraph.Edges {
		webEdges[i] = GraphEdge{
			Source: edge.Source,
			Target: edge.Target,
			Type:   edge.Type,
		}

		// Copy additional metadata from raw graph if available
		key := edgeKey{edge.Source, edge.Target, edge.Type}
		if rawEdge, exists := rawEdgeMap[key]; exists {
			webEdges[i].Linkage = rawEdge.Linkage
			webEdges[i].Symbols = rawEdge.Symbols
			webEdges[i].SourceLabel = rawEdge.SourceLabel
			webEdges[i].TargetLabel = rawEdge.TargetLabel
			webEdges[i].FileDetails = rawEdge.FileDetails
		}
	}

	return &GraphData{
		Nodes: webNodes,
		Edges: webEdges,
	}
}

// Start starts the web server on the specified port
func (s *Server) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting web server on http://localhost%s", addr)
	return http.ListenAndServe(addr, s.router)
}
