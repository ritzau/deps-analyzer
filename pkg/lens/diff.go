package lens

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

// GraphDiff represents the difference between two graph states
type GraphDiff struct {
	AddedNodes    []GraphNode `json:"addedNodes"`
	RemovedNodes  []string    `json:"removedNodes"`  // Node IDs
	ModifiedNodes []GraphNode `json:"modifiedNodes"` // Nodes with changed properties
	AddedEdges    []GraphEdge `json:"addedEdges"`
	RemovedEdges  []string    `json:"removedEdges"` // Edge IDs (source|target|type)
	FullGraph     bool        `json:"fullGraph"`    // True if this is a full graph, not a diff
}

// GraphSnapshot represents a cached graph state for diffing
type GraphSnapshot struct {
	Hash  string
	Nodes map[string]GraphNode // nodeID -> node
	Edges map[string]GraphEdge // edgeKey -> edge
}

// ComputeHash generates a hash for the request to identify cache entries
func ComputeHash(defaultLens, detailLens *LensConfig, selectedNodes []string) string {
	// Serialize the request to JSON for hashing
	data := struct {
		DefaultLens   *LensConfig
		DetailLens    *LensConfig
		SelectedNodes []string
	}{
		DefaultLens:   defaultLens,
		DetailLens:    detailLens,
		SelectedNodes: selectedNodes,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hash)
}

// CreateSnapshot creates a snapshot from graph data for diffing
func CreateSnapshot(graph *GraphData) *GraphSnapshot {
	snapshot := &GraphSnapshot{
		Nodes: make(map[string]GraphNode),
		Edges: make(map[string]GraphEdge),
	}

	// Index nodes by ID
	for _, node := range graph.Nodes {
		snapshot.Nodes[node.ID] = node
	}

	// Index edges by key (source|target|type)
	for _, edge := range graph.Edges {
		key := edgeKey(edge.Source, edge.Target, edge.Type)
		snapshot.Edges[key] = edge
	}

	// Compute hash of the graph
	jsonData, _ := json.Marshal(graph)
	hash := sha256.Sum256(jsonData)
	snapshot.Hash = fmt.Sprintf("%x", hash)

	return snapshot
}

// ComputeDiff computes the difference between two graph snapshots
func ComputeDiff(oldSnapshot *GraphSnapshot, newGraph *GraphData) *GraphDiff {
	// If no old snapshot, return full graph
	if oldSnapshot == nil {
		return &GraphDiff{
			AddedNodes: newGraph.Nodes,
			AddedEdges: newGraph.Edges,
			FullGraph:  true,
		}
	}

	diff := &GraphDiff{
		AddedNodes:    make([]GraphNode, 0),
		RemovedNodes:  make([]string, 0),
		ModifiedNodes: make([]GraphNode, 0),
		AddedEdges:    make([]GraphEdge, 0),
		RemovedEdges:  make([]string, 0),
		FullGraph:     false,
	}

	// Create index of new nodes and edges
	newNodes := make(map[string]GraphNode)
	newEdges := make(map[string]GraphEdge)

	for _, node := range newGraph.Nodes {
		newNodes[node.ID] = node
	}

	for _, edge := range newGraph.Edges {
		key := edgeKey(edge.Source, edge.Target, edge.Type)
		newEdges[key] = edge
	}

	// Find added and modified nodes
	for id, newNode := range newNodes {
		if oldNode, exists := oldSnapshot.Nodes[id]; exists {
			// Node exists - check if modified
			if !nodesEqual(oldNode, newNode) {
				diff.ModifiedNodes = append(diff.ModifiedNodes, newNode)
			}
		} else {
			// New node
			diff.AddedNodes = append(diff.AddedNodes, newNode)
		}
	}

	// Find removed nodes
	for id := range oldSnapshot.Nodes {
		if _, exists := newNodes[id]; !exists {
			diff.RemovedNodes = append(diff.RemovedNodes, id)
		}
	}

	// Find added edges
	for key, newEdge := range newEdges {
		if _, exists := oldSnapshot.Edges[key]; !exists {
			diff.AddedEdges = append(diff.AddedEdges, newEdge)
		}
	}

	// Find removed edges
	for key := range oldSnapshot.Edges {
		if _, exists := newEdges[key]; !exists {
			diff.RemovedEdges = append(diff.RemovedEdges, key)
		}
	}

	return diff
}

// edgeKey creates a unique key for an edge
func edgeKey(source, target, edgeType string) string {
	return fmt.Sprintf("%s|%s|%s", source, target, edgeType)
}

// nodesEqual checks if two nodes are equal (excluding position)
func nodesEqual(a, b GraphNode) bool {
	// Compare all fields except position (which changes during layout)
	return a.ID == b.ID &&
		a.Label == b.Label &&
		a.Type == b.Type &&
		a.Parent == b.Parent
	// Note: We don't compare metadata fields that don't affect structure
}

