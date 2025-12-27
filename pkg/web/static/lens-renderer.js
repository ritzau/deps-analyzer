/**
 * Lens-Based Graph Renderer
 *
 * Transforms raw graph data according to lens configuration rules.
 * Implements the rendering pipeline: distances → lens assignment → rule application → filtering
 */

class LensRenderer {
  /**
   * Main rendering function
   * Applies lens rules to transform raw graph data into the view to display
   *
   * @param {Object} state - Current view state (from ViewStateManager)
   * @param {Object} rawGraphData - Raw graph from API with nodes and edges
   * @returns {Object} Filtered and transformed graph ready for Cytoscape
   */
  renderGraph(state, rawGraphData) {
    if (!rawGraphData || !rawGraphData.nodes) {
      console.warn('LensRenderer: Invalid graph data');
      return { nodes: [], edges: [] };
    }

    console.log('[LensRenderer] Rendering graph with', rawGraphData.nodes.length, 'nodes');
    console.log('[LensRenderer] Focused nodes:', Array.from(state.focusedNodes));

    // 1. Compute distances from focused nodes using BFS
    const distances = this.computeDistances(rawGraphData, state.focusedNodes);

    // 2. Assign which lens controls each node (default or focus)
    const nodeLensMap = this.assignLensesToNodes(state, distances);

    const focusLensCount = Array.from(nodeLensMap.values()).filter(v => v === 'focus').length;
    console.log('[LensRenderer] Nodes using focus lens:', focusLensCount);

    // 3. Apply lens rules to determine visibility and collapse state
    const nodeStates = this.applyLensRules(
      rawGraphData,
      nodeLensMap,
      distances,
      state.defaultLens,
      state.focusLens,
      state.manualOverrides
    );

    // 4. Filter to only visible nodes
    const visibleNodes = this.filterVisibleNodes(rawGraphData, nodeStates);

    // 5. Process edges based on node visibility and edge rules
    const visibleEdges = this.processEdges(rawGraphData, nodeStates, state);

    console.log('[LensRenderer] Visible nodes:', visibleNodes.length, 'Visible edges:', visibleEdges.length);

    return {
      nodes: visibleNodes,
      edges: visibleEdges,
      nodeStates: nodeStates  // Include for debugging/tooltips
    };
  }

  /**
   * Compute shortest distance from each node to nearest focused node using BFS
   *
   * @param {Object} graph - Graph with nodes and edges
   * @param {Set<string>} focusedNodes - Set of focused node IDs
   * @returns {Map<string, number|'infinite'>} Map of node ID to distance
   */
  computeDistances(graph, focusedNodes) {
    if (!focusedNodes || focusedNodes.size === 0) {
      // No focus: all distances are infinite
      const distances = new Map();
      graph.nodes.forEach(node => distances.set(node.id || node.label, 'infinite'));
      return distances;
    }

    const distances = new Map();
    const queue = [];
    const adjacency = this.buildAdjacencyList(graph);

    // Initialize BFS with focused nodes at distance 0
    focusedNodes.forEach(nodeId => {
      distances.set(nodeId, 0);
      queue.push({ nodeId, dist: 0 });
    });

    // BFS to compute distances
    while (queue.length > 0) {
      const { nodeId, dist } = queue.shift();

      const neighbors = adjacency.get(nodeId) || [];
      for (const neighbor of neighbors) {
        if (!distances.has(neighbor)) {
          distances.set(neighbor, dist + 1);
          queue.push({ nodeId: neighbor, dist: dist + 1 });
        }
      }
    }

    // Nodes not reached are at infinite distance
    graph.nodes.forEach(node => {
      const nodeId = node.id || node.label;
      if (!distances.has(nodeId)) {
        distances.set(nodeId, 'infinite');
      }
    });

    return distances;
  }

  /**
   * Build adjacency list from graph edges (undirected for distance computation)
   *
   * @param {Object} graph - Graph with nodes and edges
   * @returns {Map<string, string[]>} Map of node ID to list of neighbor IDs
   */
  buildAdjacencyList(graph) {
    const adjacency = new Map();

    // Initialize all nodes
    graph.nodes.forEach(node => {
      const nodeId = node.id || node.label;
      adjacency.set(nodeId, []);
    });

    // Add edges (treat as undirected for distance calculation)
    graph.edges.forEach(edge => {
      const source = edge.source;
      const target = edge.target;

      if (!adjacency.has(source)) adjacency.set(source, []);
      if (!adjacency.has(target)) adjacency.set(target, []);

      adjacency.get(source).push(target);
      adjacency.get(target).push(source);  // Undirected
    });

    return adjacency;
  }

  /**
   * Determine which lens applies to each node
   * Nodes at distance 0 or 1 from focus use focus lens, others use default
   *
   * @param {Object} state - View state
   * @param {Map} distances - Distance map
   * @returns {Map<string, 'default'|'focus'>} Map of node ID to lens type
   */
  assignLensesToNodes(state, distances) {
    const nodeLensMap = new Map();

    if (!state.focusedNodes || state.focusedNodes.size === 0) {
      return nodeLensMap;  // Empty map = all use default lens
    }

    // Nodes at distance 0 or 1 use focus lens, rest use default
    distances.forEach((distance, nodeId) => {
      if (distance === 0 || distance === 1) {
        nodeLensMap.set(nodeId, 'focus');
      }
    });

    return nodeLensMap;
  }

  /**
   * Apply lens rules to determine visibility and collapse state for each node
   *
   * @param {Object} graph - Raw graph data
   * @param {Map} nodeLensMap - Map of node ID to lens type
   * @param {Map} distances - Distance map
   * @param {Object} defaultLens - Default lens configuration
   * @param {Object} focusLens - Focus lens configuration
   * @param {Map} manualOverrides - Manual collapse overrides
   * @returns {Map<string, Object>} Map of node ID to node state
   */
  applyLensRules(graph, nodeLensMap, distances, defaultLens, focusLens, manualOverrides) {
    const nodeStates = new Map();

    graph.nodes.forEach(node => {
      const nodeId = node.id || node.label;
      const lensType = nodeLensMap.get(nodeId) || 'default';
      const lens = lensType === 'focus' ? focusLens : defaultLens;
      const distance = distances.get(nodeId) || 'infinite';

      // Find matching distance rule
      const rule = this.findDistanceRule(lens, distance);

      // Determine visibility based on lens rules
      const visible = this.isNodeVisible(node, rule, lens.globalFilters);

      // Determine collapse state (lens rules + manual overrides)
      const collapsed = this.shouldNodeBeCollapsed(node, rule, manualOverrides);

      nodeStates.set(nodeId, {
        visible,
        collapsed,
        distance,
        appliedLens: lensType,
        rule
      });
    });

    return nodeStates;
  }

  /**
   * Find the distance rule that applies to a given distance
   *
   * @param {Object} lens - Lens configuration
   * @param {number|'infinite'} distance - Distance from focus
   * @returns {Object} Matching distance rule
   */
  findDistanceRule(lens, distance) {
    // Try to find exact match
    for (const rule of lens.distanceRules) {
      if (rule.distance === distance) {
        return rule;
      }
    }

    // Fall back to infinite rule
    for (const rule of lens.distanceRules) {
      if (rule.distance === 'infinite') {
        return rule;
      }
    }

    // Shouldn't happen, but return first rule as fallback
    return lens.distanceRules[0];
  }

  /**
   * Check if a node should be visible based on lens rules
   *
   * @param {Object} node - Node data
   * @param {Object} rule - Distance rule
   * @param {Object} globalFilters - Global filters
   * @returns {boolean} Whether node should be visible
   */
  isNodeVisible(node, rule, globalFilters) {
    // Apply global filters first
    if (globalFilters.hideExternal && node.external) return false;
    if (globalFilters.hideUncovered && node.uncovered) return false;
    if (globalFilters.hideSystemLibs && node.type === 'system_library') return false;

    // System libraries have special visibility rules
    if (node.type === 'system_library') {
      return rule.nodeVisibility.showSystemLibraries;
    }

    // Check if this target type should be shown
    const targetTypes = rule.nodeVisibility.targetTypes || [];
    if (!targetTypes.includes(node.type)) {
      return false;
    }

    // Check file-level visibility
    if (node.fileType) {
      const fileTypes = rule.nodeVisibility.fileTypes || [];
      if (fileTypes.includes('none')) return false;
      if (fileTypes.includes('all')) return true;
      return fileTypes.includes(node.fileType);
    }

    return true;
  }

  /**
   * Determine if a node should be collapsed
   * Checks manual overrides first, then falls back to lens rules
   *
   * @param {Object} node - Node data
   * @param {Object} rule - Distance rule
   * @param {Map} manualOverrides - Manual collapse state overrides
   * @returns {boolean} Whether node should be collapsed
   */
  shouldNodeBeCollapsed(node, rule, manualOverrides) {
    const nodeId = node.id || node.label;

    // Check for manual override first (Layer 3)
    if (manualOverrides && manualOverrides.has(nodeId)) {
      const manual = manualOverrides.get(nodeId);
      if (manual && manual.collapsed !== null) {
        return manual.collapsed;
      }
    }

    // Fall back to lens rules (Layer 1 or 2)
    // For now, always return false (expanded)
    // TODO: Implement hierarchy-based collapsing using collapseLevel
    return false;
  }

  /**
   * Filter nodes to only those that should be visible
   *
   * @param {Object} graph - Raw graph data
   * @param {Map} nodeStates - Node state map
   * @returns {Array} Array of visible nodes
   */
  filterVisibleNodes(graph, nodeStates) {
    return graph.nodes.filter(node => {
      const nodeId = node.id || node.label;
      const state = nodeStates.get(nodeId);
      return state && state.visible;
    });
  }

  /**
   * Process edges based on node visibility and edge rules
   *
   * @param {Object} graph - Raw graph data
   * @param {Map} nodeStates - Node state map
   * @param {Object} viewState - Current view state
   * @returns {Array} Array of visible edges
   */
  processEdges(graph, nodeStates, viewState) {
    return graph.edges.filter(edge => {
      const sourceState = nodeStates.get(edge.source);
      const targetState = nodeStates.get(edge.target);

      // Only show edge if both nodes are visible
      if (!sourceState || !targetState) return false;
      if (!sourceState.visible || !targetState.visible) return false;

      // Determine which lens controls this edge (based on source node)
      const sourceLens = sourceState.appliedLens === 'focus' ?
        viewState.focusLens : viewState.defaultLens;

      // Check if edge type is enabled in lens rules
      if (!sourceLens.edgeRules.types.has(edge.type)) {
        return false;
      }

      return true;
    });
  }
}
