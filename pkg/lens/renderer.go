package lens

import (
	"fmt"
	"log"
	"sort"
	"strings"
)

// RenderGraph applies lens transformations to raw graph data
// This is the main entry point for the lens rendering pipeline
func RenderGraph(rawGraph *GraphData, defaultLens, focusLens *LensConfig, focusedNodes []string, manualOverrides map[string]ManualOverride) (*GraphData, error) {
	log.Printf("[LensRenderer] Rendering graph with %d nodes", len(rawGraph.Nodes))
	log.Printf("[LensRenderer] Focused nodes: %v", focusedNodes)

	// 1. Compute distances from focused nodes using BFS
	distances := ComputeDistances(rawGraph, focusedNodes)

	// 2. Assign which lens controls each node (default or focus)
	nodeLensMap := assignLensesToNodes(distances, focusedNodes)

	focusCount := 0
	for _, lensType := range nodeLensMap {
		if lensType == "focus" {
			focusCount++
		}
	}
	log.Printf("[LensRenderer] Nodes using focus lens: %d", focusCount)

	// 3. Apply lens rules to determine visibility and collapse state
	nodeStates := applyLensRules(rawGraph, nodeLensMap, distances, defaultLens, focusLens, manualOverrides)

	// 4. Extract and create synthetic package nodes from ALL targets
	allPackageNodes := extractPackageNodes(rawGraph)

	// Add states for synthetic package nodes
	for _, pkgNode := range allPackageNodes {
		if _, exists := nodeStates[pkgNode.ID]; !exists {
			lensType := nodeLensMap[pkgNode.ID]
			if lensType == "" {
				lensType = "default"
			}

			var lens *LensConfig
			if lensType == "focus" {
				lens = focusLens
			} else {
				lens = defaultLens
			}

			// For packages, use distance 0 if assigned focus lens (contains focused target)
			var distance interface{} = "infinite"
			if lensType == "focus" {
				distance = 0
			}

			rule := findDistanceRule(lens, distance)
			collapsed := shouldNodeBeCollapsed(pkgNode, rule, manualOverrides)

			nodeStates[pkgNode.ID] = &NodeState{
				Visible:     true,
				Collapsed:   collapsed,
				Distance:    distance,
				AppliedLens: lensType,
				Rule:        rule,
			}
		}
	}

	// 5. Combine raw nodes with package nodes for visibility filtering
	allNodes := append([]GraphNode{}, rawGraph.Nodes...)
	allNodes = append(allNodes, allPackageNodes...)

	// 6. Filter to only visible nodes
	visibleNodes := filterVisibleNodes(allNodes, nodeStates)

	log.Printf("[LensRenderer] Visible nodes after filtering: %d", len(visibleNodes))

	// 7. Build hierarchy relationships for visible nodes
	hierarchicalNodes := buildHierarchy(visibleNodes, nodeStates)

	// 8. Filter out children of collapsed nodes
	expandedNodes := filterCollapsedChildren(hierarchicalNodes, nodeStates)

	log.Printf("[LensRenderer] Nodes after collapse filtering: %d", len(expandedNodes))

	// 9. Rebuild hierarchy with filtered nodes
	finalNodes := buildHierarchy(expandedNodes, nodeStates)

	// 10. Build child->parent map for edge aggregation
	childToParentMap := buildChildToParentMap(allNodes, nodeStates)

	// 11. Create set of included node IDs for edge filtering
	includedNodeIds := make(map[string]bool)
	for _, node := range finalNodes {
		includedNodeIds[node.ID] = true
	}

	// 12. Aggregate edges for collapsed nodes
	visibleEdges := aggregateEdgesForCollapsedNodes(rawGraph, nodeStates, defaultLens, focusLens, nodeLensMap, includedNodeIds, childToParentMap)

	// 13. Sort nodes for deterministic ordering (Dagre layout stability)
	sort.Slice(finalNodes, func(i, j int) bool {
		return finalNodes[i].ID < finalNodes[j].ID
	})

	log.Printf("[LensRenderer] Final result: %d nodes, %d edges", len(finalNodes), len(visibleEdges))

	return &GraphData{
		Nodes: finalNodes,
		Edges: visibleEdges,
	}, nil
}

// assignLensesToNodes determines which lens applies to each node
// When focused nodes exist, ALL nodes use focus lens (allowing distance rules to control visibility)
// When no focused nodes exist, all nodes use default lens
func assignLensesToNodes(distances map[string]interface{}, focusedNodes []string) map[string]string {
	nodeLensMap := make(map[string]string)

	if len(focusedNodes) == 0 {
		return nodeLensMap // Empty map = all use default lens
	}

	// When we have focused nodes, ALL nodes use the focus lens
	// This allows the focus lens's distance rules (0, 1, infinite) to properly control visibility
	// Nodes at distance 0: shown with files (per focus lens distance 0 rule)
	// Nodes at distance 1: shown without files (per focus lens distance 1 rule)
	// Nodes at distance 2+: hidden (per focus lens infinite distance rule with targetTypes: [])
	for nodeID := range distances {
		nodeLensMap[nodeID] = "focus"
	}

	return nodeLensMap
}

// applyLensRules applies lens rules to determine visibility and collapse state for each node
func applyLensRules(graph *GraphData, nodeLensMap map[string]string, distances map[string]interface{}, defaultLens, focusLens *LensConfig, manualOverrides map[string]ManualOverride) map[string]*NodeState {
	nodeStates := make(map[string]*NodeState)

	for _, node := range graph.Nodes {
		lensType := nodeLensMap[node.ID]
		if lensType == "" {
			lensType = "default"
		}

		var lens *LensConfig
		if lensType == "focus" {
			lens = focusLens
		} else {
			lens = defaultLens
		}

		distance := distances[node.ID]
		if distance == nil {
			distance = "infinite"
		}

		// Find the appropriate distance rule
		rule := findDistanceRule(lens, distance)

		// Check visibility
		visible := isNodeVisibleByRule(&node, rule, lens)

		// Check collapse state
		collapsed := shouldNodeBeCollapsed(node, rule, manualOverrides)

		nodeStates[node.ID] = &NodeState{
			Visible:     visible,
			Collapsed:   collapsed,
			Distance:    distance,
			AppliedLens: lensType,
			Rule:        rule,
		}
	}

	return nodeStates
}

// findDistanceRule finds the matching distance rule for a given distance
func findDistanceRule(lens *LensConfig, distance interface{}) *DistanceRule {
	if lens == nil || len(lens.DistanceRules) == 0 {
		return nil
	}

	// Try to find exact match
	for i := range lens.DistanceRules {
		rule := &lens.DistanceRules[i]
		if compareDistance(rule.Distance, distance) {
			return rule
		}
	}

	// Fall back to "infinite" rule if it exists
	for i := range lens.DistanceRules {
		rule := &lens.DistanceRules[i]
		if rule.Distance == "infinite" {
			return rule
		}
	}

	// No matching rule found
	return nil
}

// compareDistance compares two distance values for equality
func compareDistance(a, b interface{}) bool {
	// Handle string comparison (e.g., "infinite")
	aStr, aIsStr := a.(string)
	bStr, bIsStr := b.(string)
	if aIsStr && bIsStr {
		return aStr == bStr
	}

	// Handle int comparison
	aInt, aIsInt := a.(int)
	bInt, bIsInt := b.(int)
	if aIsInt && bIsInt {
		return aInt == bInt
	}

	// Handle float64 (JSON unmarshaling often produces float64)
	aFloat, aIsFloat := a.(float64)
	bFloat, bIsFloat := b.(float64)
	if aIsFloat && bIsFloat {
		return aFloat == bFloat
	}

	// Mixed int/float comparison
	if aIsInt && bIsFloat {
		return float64(aInt) == bFloat
	}
	if aIsFloat && bIsInt {
		return aFloat == float64(bInt)
	}

	return false
}

// isNodeVisibleByRule determines if a node is visible according to the lens rule
func isNodeVisibleByRule(node *GraphNode, rule *DistanceRule, lens *LensConfig) bool {
	if rule == nil {
		return false
	}

	vis := rule.NodeVisibility

	// Check global filters first
	if lens.GlobalFilters.HideExternal && (node.Type == "external" || strings.Contains(node.ID, "@")) {
		return false
	}
	if lens.GlobalFilters.HideUncovered && (node.Type == "uncovered_source" || node.Type == "uncovered_header") {
		return false
	}
	if lens.GlobalFilters.HideSystemLibs && node.Type == "system_library" {
		return false
	}

	// Check target types
	if isTargetType(node.Type) {
		if !contains(vis.TargetTypes, node.Type) {
			return false
		}
	}

	// Check file types
	if isFileType(node.Type) {
		if !contains(vis.FileTypes, "all") && !contains(vis.FileTypes, node.Type) {
			// Special handling for "none" - hide all files
			if contains(vis.FileTypes, "none") {
				return false
			}
		}
	}

	// Check package visibility - packages should be hidden if no target types are visible
	// Package nodes have type "package" or empty string and ID like "//foo"
	if node.Type == "package" || (node.Type == "" && strings.HasPrefix(node.ID, "//") && !strings.Contains(node.ID, ":")) {
		// If targetTypes is empty, hide the package (since all its children would be hidden)
		if len(vis.TargetTypes) == 0 {
			return false
		}
	}

	// Check specific visibility flags
	if node.Type == "uncovered_source" || node.Type == "uncovered_header" {
		if !vis.ShowUncovered {
			return false
		}
	}

	if node.Type == "external" || strings.Contains(node.ID, "@") {
		if !vis.ShowExternal {
			return false
		}
	}

	if node.Type == "system_library" {
		if !vis.ShowSystemLibraries {
			return false
		}
	}

	return true
}

// shouldNodeBeCollapsed determines if a node should be collapsed
func shouldNodeBeCollapsed(node GraphNode, rule *DistanceRule, manualOverrides map[string]ManualOverride) bool {
	// Manual overrides take precedence (Layer 3)
	if override, exists := manualOverrides[node.ID]; exists {
		return override.Collapsed
	}

	// Use lens rule (Layer 1 or 2)
	if rule == nil {
		return false
	}

	// Collapse level determines how deep in the hierarchy we show nodes
	// 0 = hide everything (collapse all packages)
	// 1 = show only packages (collapse all targets)
	// 2 = show packages and targets (collapse all files)
	// 3 = show everything (no collapse)
	//
	// A node should be "collapsed" if we want to hide its children.
	// For example, with collapseLevel = 2:
	//   - Packages (level 1): NOT collapsed (show their children = targets)
	//   - Targets (level 2): YES collapsed (hide their children = files)
	//   - Files (level 3): NOT collapsed (they have no children)

	nodeLevel := getNodeHierarchyLevel(node.ID, node.Type)

	// A node is collapsed if its level equals the collapse level
	// (meaning we show nodes at this level, but hide their children)
	return nodeLevel == rule.CollapseLevel
}

// getNodeHierarchyLevel returns the hierarchy level of a node
// 1 = package, 2 = target, 3 = file
func getNodeHierarchyLevel(nodeID, nodeType string) int {
	// Package nodes (synthetic) have no colons or only package prefix
	if !strings.Contains(nodeID, ":") || strings.HasSuffix(nodeID, ":") {
		return 1 // Package level
	}

	// Count colons to determine depth
	colonCount := strings.Count(nodeID, ":")

	if colonCount == 1 {
		return 2 // Target level (//package:target)
	}

	return 3 // File level (//package:target:file)
}

// extractPackageNodes creates synthetic package nodes from target nodes
func extractPackageNodes(graph *GraphData) []GraphNode {
	packages := make(map[string]bool)
	var packageNodes []GraphNode

	// Extract unique packages from target nodes
	for _, node := range graph.Nodes {
		if isTargetType(node.Type) {
			pkgID := extractPackageID(node.ID)
			if pkgID != "" && !packages[pkgID] {
				packages[pkgID] = true
				packageNodes = append(packageNodes, GraphNode{
					ID:     pkgID,
					Label:  pkgID,
					Type:   "package",
					Parent: "",
				})
			}
		}
	}

	return packageNodes
}

// extractPackageID extracts the package ID from a target or file ID
// Examples: //util:util -> //util, //foo/bar:baz -> //foo/bar
func extractPackageID(nodeID string) string {
	if !strings.HasPrefix(nodeID, "//") {
		return ""
	}

	// Find the first colon
	colonIdx := strings.Index(nodeID, ":")
	if colonIdx == -1 {
		return nodeID // Already a package ID
	}

	return nodeID[:colonIdx]
}

// filterVisibleNodes filters nodes based on visibility state
func filterVisibleNodes(nodes []GraphNode, nodeStates map[string]*NodeState) []GraphNode {
	var visible []GraphNode

	for _, node := range nodes {
		state := nodeStates[node.ID]
		if state != nil && state.Visible {
			visible = append(visible, node)
		}
	}

	return visible
}

// buildHierarchy builds parent-child relationships for nodes
func buildHierarchy(nodes []GraphNode, nodeStates map[string]*NodeState) []GraphNode {
	result := make([]GraphNode, len(nodes))

	for i, node := range nodes {
		// Copy node
		result[i] = node

		// Determine parent based on ID structure
		// //package:target:file -> parent is //package:target
		// //package:target -> parent is //package
		parent := extractParentID(node.ID)
		if parent != "" && parent != node.ID {
			result[i].Parent = parent
		} else {
			result[i].Parent = ""
		}
	}

	return result
}

// filterCollapsedChildren filters out children of collapsed parent nodes
func filterCollapsedChildren(nodes []GraphNode, nodeStates map[string]*NodeState) []GraphNode {
	var result []GraphNode
	filtered := 0

	for _, node := range nodes {
		// Check if any ancestor is collapsed
		if !hasCollapsedAncestor(node.ID, nodeStates) {
			result = append(result, node)
		} else {
			filtered++
		}
	}

	if filtered > 0 {
		log.Printf("[Lens] Filtered out %d nodes with collapsed ancestors (kept %d nodes)", filtered, len(result))
	}

	return result
}

// hasCollapsedAncestor checks if any ancestor of a node is collapsed
func hasCollapsedAncestor(nodeID string, nodeStates map[string]*NodeState) bool {
	parentID := extractParentID(nodeID)

	for parentID != "" && parentID != nodeID {
		state := nodeStates[parentID]
		if state != nil && state.Collapsed {
			return true
		}
		nodeID = parentID
		parentID = extractParentID(nodeID)
	}

	return false
}

// buildChildToParentMap builds a map from child node ID to parent node ID
func buildChildToParentMap(nodes []GraphNode, nodeStates map[string]*NodeState) map[string]string {
	childToParent := make(map[string]string)

	for _, node := range nodes {
		parentID := extractParentID(node.ID)
		if parentID != "" && parentID != node.ID {
			childToParent[node.ID] = parentID
		}
	}

	return childToParent
}

// aggregateEdgesForCollapsedNodes aggregates edges based on node collapse state
func aggregateEdgesForCollapsedNodes(rawGraph *GraphData, nodeStates map[string]*NodeState, defaultLens, focusLens *LensConfig, nodeLensMap map[string]string, includedNodeIds map[string]bool, childToParentMap map[string]string) []GraphEdge {
	var visibleEdges []GraphEdge
	edgeMap := make(map[string]*GraphEdge) // Key: "source|target|type"

	for _, edge := range rawGraph.Edges {
		// Find the actual source and target nodes (may be aggregated to parent)
		actualSource := findVisibleAncestor(edge.Source, includedNodeIds, childToParentMap)
		actualTarget := findVisibleAncestor(edge.Target, includedNodeIds, childToParentMap)

		// Skip edges where source or target is not visible
		if actualSource == "" || actualTarget == "" {
			continue
		}

		// Skip self-edges
		if actualSource == actualTarget {
			continue
		}

		// Check if edge type is allowed by lens rules
		sourceState := nodeStates[actualSource]
		if sourceState == nil {
			continue
		}

		lensType := sourceState.AppliedLens
		var lens *LensConfig
		if lensType == "focus" {
			lens = focusLens
		} else {
			lens = defaultLens
		}

		// Check edge type filter
		if !contains(lens.EdgeRules.Types, edge.Type) {
			continue
		}

		// Create edge key for aggregation
		edgeKey := fmt.Sprintf("%s|%s|%s", actualSource, actualTarget, edge.Type)

		// Aggregate edges (for collapsed nodes, multiple edges may map to same aggregated edge)
		if _, exists := edgeMap[edgeKey]; !exists {
			// Create new aggregated edge (just the key fields - metadata will be added by web layer)
			edgeMap[edgeKey] = &GraphEdge{
				Source: actualSource,
				Target: actualTarget,
				Type:   edge.Type,
			}
		}
		// Note: Multiple edges with same source/target/type are aggregated into one
		// The web layer will restore metadata (symbols, file details) from the raw graph
	}

	// Convert map to slice and sort for deterministic order
	// This is critical for Dagre layout stability - if edges arrive in different
	// orders, Dagre may place nodes differently even with the same graph structure
	for _, edge := range edgeMap {
		visibleEdges = append(visibleEdges, *edge)
	}

	// Sort edges by source, then target, then type for canonical ordering
	sort.Slice(visibleEdges, func(i, j int) bool {
		if visibleEdges[i].Source != visibleEdges[j].Source {
			return visibleEdges[i].Source < visibleEdges[j].Source
		}
		if visibleEdges[i].Target != visibleEdges[j].Target {
			return visibleEdges[i].Target < visibleEdges[j].Target
		}
		return visibleEdges[i].Type < visibleEdges[j].Type
	})

	return visibleEdges
}

// findVisibleAncestor finds the nearest visible ancestor of a node
func findVisibleAncestor(nodeID string, includedNodeIds map[string]bool, childToParentMap map[string]string) string {
	// Check if node itself is visible
	if includedNodeIds[nodeID] {
		return nodeID
	}

	// Walk up the hierarchy
	currentID := nodeID
	for {
		parentID := childToParentMap[currentID]
		if parentID == "" {
			break
		}

		if includedNodeIds[parentID] {
			return parentID
		}

		currentID = parentID
	}

	return ""
}

// Helper functions

func isTargetType(nodeType string) bool {
	return nodeType == "cc_library" || nodeType == "cc_binary" || nodeType == "cc_shared_library"
}

func isFileType(nodeType string) bool {
	return nodeType == "source" || nodeType == "header" || nodeType == "uncovered_source" || nodeType == "uncovered_header"
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
