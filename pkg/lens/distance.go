package lens

import (
	"strings"
)

// GraphNode represents a node in the dependency graph (temporary, mirrors web.GraphNode)
type GraphNode struct {
	ID     string
	Label  string
	Type   string
	Parent string
}

// GraphEdge represents an edge in the dependency graph (temporary, mirrors web.GraphEdge)
type GraphEdge struct {
	Source string
	Target string
	Type   string
}

// GraphData holds the dependency graph for visualization (temporary, mirrors web.GraphData)
type GraphData struct {
	Nodes []GraphNode
	Edges []GraphEdge
}

// distanceQueueNode represents a node in the BFS queue
type distanceQueueNode struct {
	nodeID   string
	distance int
}

// expandPackagesToTargets expands package IDs into all their target IDs and uncovered files
// For example, "//main" becomes ["//main:test_app", "//main:other_target", "uncovered:main/file.cc", ...]
// This allows selecting a package to select all targets and uncovered files within it
func expandPackagesToTargets(selectedNodes []string, graph *GraphData) []string {
	expanded := make(map[string]bool)

	for _, nodeID := range selectedNodes {
		// Check if this is a package ID (no colons, like "//main")
		if !strings.Contains(nodeID, ":") {
			// Find all targets in this package
			foundTargets := false
			for _, node := range graph.Nodes {
				if strings.HasPrefix(node.ID, nodeID+":") {
					// This is a target or file in the focused package
					// Extract the target ID (first two parts: //package:target)
					parts := strings.SplitN(node.ID, ":", 3)
					if len(parts) >= 2 {
						targetID := parts[0] + ":" + parts[1]
						expanded[targetID] = true
						foundTargets = true
					}
				}
			}

			// Also find uncovered files in this package
			// Extract package path from package label (e.g., "//util" -> "util")
			packagePath := strings.TrimPrefix(nodeID, "//")
			for _, node := range graph.Nodes {
				if strings.HasPrefix(node.ID, "uncovered:") {
					filePath := strings.TrimPrefix(node.ID, "uncovered:")
					// Check if this file belongs to the package
					// E.g., "util/orphaned.cc" belongs to package "util"
					if strings.HasPrefix(filePath, packagePath+"/") {
						expanded[node.ID] = true
						foundTargets = true // Mark that we found something in this package
					}
				}
			}

			// If we found targets or uncovered files, don't add the package itself
			// If nothing found, add the package (in case it's a valid node)
			if !foundTargets {
				expanded[nodeID] = true
			}
		} else {
			// Not a package, just add it directly
			expanded[nodeID] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(expanded))
	for nodeID := range expanded {
		result = append(result, nodeID)
	}

	return result
}

// ComputeDistances calculates shortest distance from each node to nearest selected node
// Returns a map of nodeID -> distance (int or "infinite")
func ComputeDistances(graph *GraphData, selectedNodes []string) map[string]interface{} {
	distances := make(map[string]interface{})

	// If no selected nodes, all distances are infinite
	if len(selectedNodes) == 0 {
		for _, node := range graph.Nodes {
			distances[node.ID] = "infinite"
		}
		return distances
	}

	// Build adjacency list (undirected graph for distance computation)
	adjacency := buildAdjacencyList(graph)

	// Expand selected nodes: if a package is selected (e.g., "//main"), include all its targets
	// This ensures that clicking on a package selects all targets within it
	expandedSelectedNodes := expandPackagesToTargets(selectedNodes, graph)

	// Initialize BFS queue with selected nodes at distance 0
	queue := []distanceQueueNode{}
	for _, nodeID := range expandedSelectedNodes {
		distances[nodeID] = 0
		queue = append(queue, distanceQueueNode{nodeID: nodeID, distance: 0})
	}

	// BFS traversal
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, neighbor := range adjacency[current.nodeID] {
			if _, exists := distances[neighbor]; !exists {
				newDistance := current.distance + 1
				distances[neighbor] = newDistance
				queue = append(queue, distanceQueueNode{nodeID: neighbor, distance: newDistance})
			}
		}
	}

	// Handle nodes not reached by BFS - inherit from parent or mark as infinite
	for _, node := range graph.Nodes {
		if _, exists := distances[node.ID]; !exists {
			distances[node.ID] = getInheritedDistance(node.ID, node.Parent, distances)
		}
	}

	return distances
}

// buildAdjacencyList creates an undirected adjacency list from graph edges
func buildAdjacencyList(graph *GraphData) map[string][]string {
	adjacency := make(map[string][]string)

	for _, edge := range graph.Edges {
		// Add both directions (undirected for distance computation)
		adjacency[edge.Source] = append(adjacency[edge.Source], edge.Target)
		adjacency[edge.Target] = append(adjacency[edge.Target], edge.Source)
	}

	return adjacency
}

// getInheritedDistance recursively inherits distance from parent nodes
// This handles cases where child nodes (like files) should inherit the distance of their parent (target/package)
func getInheritedDistance(nodeID string, parentID string, distances map[string]interface{}) interface{} {
	// If we have a parent, inherit its distance
	if parentID != "" {
		if dist, exists := distances[parentID]; exists {
			return dist
		}
		// Parent doesn't have a distance yet - shouldn't happen in well-formed graph
		return "infinite"
	}

	// No parent - check if this is a child node by parsing the ID
	// Node IDs follow pattern: //package:target:file or //package:target
	implicitParent := extractParentID(nodeID)
	if implicitParent != "" && implicitParent != nodeID {
		if dist, exists := distances[implicitParent]; exists {
			return dist
		}
		// Recurse up the hierarchy
		return getInheritedDistance(implicitParent, "", distances)
	}

	// Top-level node with no distance - mark as infinite
	return "infinite"
}

// extractParentID extracts the parent node ID from a hierarchical node ID
// Examples:
//
//	//package:target:file -> //package:target
//	//package:target -> //package
//	//package -> ""
func extractParentID(nodeID string) string {
	// Handle uncovered files specially: uncovered:path/file.cc -> //path
	if strings.HasPrefix(nodeID, "uncovered:") {
		filePath := strings.TrimPrefix(nodeID, "uncovered:")
		if idx := strings.LastIndex(filePath, "/"); idx >= 0 {
			packagePath := filePath[:idx]
			return "//" + packagePath
		}
		// No slash means file at root, no parent
		return ""
	}

	// Handle external targets: @repo//:target:file
	if strings.HasPrefix(nodeID, "@") {
		// Split by ':'
		parts := strings.Split(nodeID, ":")
		if len(parts) <= 1 {
			// No colons, no parent (e.g., just "@repo")
			return ""
		}

		// Remove the last component
		parentParts := parts[:len(parts)-1]
		return strings.Join(parentParts, ":")
	}

	// Handle workspace targets: //package:target:file
	if !strings.HasPrefix(nodeID, "//") {
		return ""
	}
	nodeID = nodeID[2:]

	// Split by ':'
	parts := strings.Split(nodeID, ":")
	if len(parts) <= 1 {
		// Already at package level (no ':' separators)
		return ""
	}

	// Remove the last component
	parentParts := parts[:len(parts)-1]
	return "//" + strings.Join(parentParts, ":")
}

