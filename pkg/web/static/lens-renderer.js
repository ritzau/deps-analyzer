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

    // 3.5. Apply manual override visibility (Layer 3 overrides)
    // When a parent is manually expanded, make its children visible
    this.applyManualVisibilityOverrides(rawGraphData, nodeStates, state.manualOverrides);

    // 4. Extract and create synthetic package nodes from ALL targets (before visibility filtering)
    // This ensures package nodes exist even when targets are hidden (e.g., collapseLevel=1)
    const allPackageNodes = this.extractPackageNodes(rawGraphData);

    // Add states for synthetic package nodes
    const rule = state.defaultLens.distanceRules[0];
    allPackageNodes.forEach(node => {
      if (!nodeStates.has(node.id)) {
        nodeStates.set(node.id, {
          visible: true,
          collapsed: this.shouldNodeBeCollapsed(node, rule, state.manualOverrides),
          distance: 'infinite',
          appliedLens: 'default',
          rule: rule
        });
      }
    });

    // 5. Filter to only visible nodes (now including package nodes)
    const visibleNodes = this.filterVisibleNodes(
      { nodes: [...rawGraphData.nodes, ...allPackageNodes], edges: rawGraphData.edges },
      nodeStates
    );

    // 6. Build hierarchy relationships for visible nodes
    const hierarchicalData = this.buildHierarchy(visibleNodes, nodeStates);

    // 7. Filter out children of collapsed nodes
    const expandedNodes = this.filterCollapsedChildren(hierarchicalData.nodes, nodeStates);

    // 8. Rebuild hierarchy with filtered nodes
    const hierarchicalNodes = this.buildHierarchy(expandedNodes, nodeStates).nodes;

    // Create a set of included node IDs for edge filtering
    const includedNodeIds = new Set(hierarchicalNodes.map(n => n.id || n.label));

    // 9. Build a map of child->parent for edge aggregation
    // Include both raw nodes and synthetic package nodes
    const allNodes = [...rawGraphData.nodes, ...allPackageNodes];
    const childToParentMap = this.buildChildToParentMap(allNodes, nodeStates);

    // 10. Process edges based on node visibility and edge rules, aggregating for collapsed nodes
    const visibleEdges = this.aggregateEdgesForCollapsedNodes(
      rawGraphData,
      nodeStates,
      state,
      includedNodeIds,
      childToParentMap
    );

    console.log('[LensRenderer] Visible nodes:', visibleNodes.length, 'After collapse filter:', expandedNodes.length, 'Hierarchical nodes:', hierarchicalNodes.length, 'Visible edges:', visibleEdges.length);

    return {
      nodes: hierarchicalNodes,
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

      // Determine visibility based on lens rules (Layers 1 & 2)
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
   * Apply manual visibility overrides (Layer 3)
   * When a parent node is manually expanded, make its children visible
   *
   * @param {Object} graph - Raw graph data
   * @param {Map} nodeStates - Node state map to update
   * @param {Map} manualOverrides - Manual override states
   */
  applyManualVisibilityOverrides(graph, nodeStates, manualOverrides) {
    if (!manualOverrides || manualOverrides.size === 0) return;

    // Find all manually expanded nodes (collapsed=false)
    const expandedNodes = new Set();
    manualOverrides.forEach((override, nodeId) => {
      if (override.collapsed === false) {
        expandedNodes.add(nodeId);
      }
    });

    if (expandedNodes.size === 0) return;

    console.log('[ManualVisibility] Manually expanded nodes:', Array.from(expandedNodes));

    // Debug: count node types in raw graph and show sample IDs
    const nodeTypes = {};
    const sampleIds = { file: null, target: null };
    graph.nodes.forEach(n => {
      nodeTypes[n.type] = (nodeTypes[n.type] || 0) + 1;
      if ((n.type === 'source_file' || n.type === 'header_file') && !sampleIds.file) {
        sampleIds.file = { id: n.id, label: n.label, parent: n.parent };
      }
      if ((n.type === 'cc_library' || n.type === 'cc_binary') && !sampleIds.target) {
        sampleIds.target = { id: n.id, label: n.label };
      }
    });
    console.log('[ManualVisibility] Node types in raw graph:', nodeTypes);
    console.log('[ManualVisibility] Sample file node:', sampleIds.file);
    console.log('[ManualVisibility] Sample target node:', sampleIds.target);

    let targetsShown = 0;
    let filesShown = 0;

    // For each node, check if its parent is manually expanded
    graph.nodes.forEach(node => {
      const nodeId = node.id || node.label;
      // Use node.id for hierarchical parsing (contains full path for files)
      // node.label is just the display name (e.g., "engine.cc" for files)
      const hierarchicalId = node.id || node.label;

      // Check if this is a target node with a package parent
      if (hierarchicalId.startsWith('//') && hierarchicalId.includes(':')) {
        const parts = hierarchicalId.substring(2).split(':');

        if (parts.length === 2) {
          // This is a target: //package:target
          const packageLabel = '//' + parts[0];

          // If parent package is manually expanded, make this target visible
          if (expandedNodes.has(packageLabel)) {
            const state = nodeStates.get(nodeId);
            if (state) {
              state.visible = true;
              targetsShown++;
            }
          }
        } else if (parts.length >= 3) {
          // This is a file: //package:target:filepath
          const targetLabel = parts[0] + ':' + parts[1];
          const fullTargetLabel = '//' + targetLabel;

          // If parent target is manually expanded, make this file visible
          if (expandedNodes.has(fullTargetLabel)) {
            const state = nodeStates.get(nodeId);
            if (state) {
              state.visible = true;
              filesShown++;
              console.log('[ManualVisibility] Made file visible:', nodeId, 'parent:', fullTargetLabel);
            }
          }
        }
      }
    });

    if (targetsShown > 0 || filesShown > 0) {
      console.log(`[ManualVisibility] Made ${targetsShown} targets and ${filesShown} files visible due to manual expansion`);
    }
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
   * Check if a node should be visible based on lens rules (Layers 1 & 2)
   * Note: Manual overrides (Layer 3) are applied separately in applyManualVisibilityOverrides
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

    // Check file node visibility (source_file, header_file)
    if (node.type === 'source_file' || node.type === 'header_file') {
      const fileTypes = rule.nodeVisibility.fileTypes || [];
      if (fileTypes.includes('none')) return false;
      if (fileTypes.includes('all')) return true;
      // Could add more granular filtering here (source vs header)
      return false;
    }

    // Check collapseLevel to determine if targets should be visible
    // collapseLevel: 1 = packages only (hide targets by default)
    // collapseLevel: 2 = targets visible (hide files)
    // collapseLevel: 3 = files visible
    const collapseLevel = rule.collapseLevel || 3;
    if (node.type === 'cc_binary' || node.type === 'cc_library' || node.type === 'cc_shared_library') {
      // Targets should be hidden when collapseLevel < 2
      // Manual overrides will make them visible if parent is expanded
      if (collapseLevel < 2) {
        return false;
      }
    }

    // Check if this target type should be shown
    const targetTypes = rule.nodeVisibility.targetTypes || [];
    if (!targetTypes.includes(node.type)) {
      return false;
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
    // Use collapseLevel from the rule to determine default collapse state
    // collapseLevel: 1 = packages expanded, targets collapsed
    // collapseLevel: 2 = targets expanded, files collapsed (hidden)
    // collapseLevel: 3 = files expanded (visible)

    if (!rule || !rule.collapseLevel) {
      // No rule specified, use conservative default (collapse targets)
      if (node.type === 'cc_binary' || node.type === 'cc_library' || node.type === 'cc_shared_library') {
        return true;
      }
      return false;
    }

    const collapseLevel = rule.collapseLevel;

    // Determine node level
    // Level 1 = package nodes
    // Level 2 = target nodes
    // Level 3 = file nodes
    let nodeLevel = 0;
    if (node.type === 'package') {
      nodeLevel = 1;
    } else if (node.type === 'cc_binary' || node.type === 'cc_library' || node.type === 'cc_shared_library') {
      nodeLevel = 2;
    } else if (node.type === 'source_file' || node.type === 'header_file') {
      nodeLevel = 3;
    }

    // Collapse nodes at levels deeper than collapseLevel
    // E.g., if collapseLevel is 2, collapse level 3 nodes (files)
    // This is handled by parent collapse - targets (level 2) should be collapsed if collapseLevel < 3
    if (nodeLevel === 2 && collapseLevel < 3) {
      return true; // Collapse targets when we don't want to see files
    }
    if (nodeLevel === 1 && collapseLevel < 2) {
      return true; // Collapse packages when we don't want to see targets
    }

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
   * Filter out children of collapsed parent nodes
   * File nodes whose parent target is collapsed should not be rendered
   *
   * @param {Array} nodes - Array of visible nodes
   * @param {Map} nodeStates - Node state map
   * @returns {Array} Array of nodes with collapsed children removed
   */
  filterCollapsedChildren(nodes, nodeStates) {
    // Build a map of which nodes are children of which parents
    const parentMap = new Map();

    nodes.forEach(node => {
      const nodeId = node.id || node.label;
      const label = node.label;

      // File nodes: parent is the target they belong to
      if (node.type === 'source_file' || node.type === 'header_file') {
        // Extract parent from node ID (format: "//target:label:filepath")
        const parts = nodeId.split(':');
        if (parts.length >= 3) {
          const parentLabel = parts[0] + ':' + parts[1]; // //package:target
          parentMap.set(nodeId, parentLabel);
        }
      }
      // Target nodes: parent is the package (if it has one)
      else if (label.startsWith('//') && label.includes(':')) {
        const parts = label.substring(2).split(':');
        if (parts.length > 1) {
          const packageLabel = '//' + parts[0];
          parentMap.set(nodeId, packageLabel);
        }
      }
    });

    // Filter out nodes whose parent (or any ancestor) is collapsed
    const result = nodes.filter(node => {
      const nodeId = node.id || node.label;
      let currentId = nodeId;

      // Walk up the parent chain
      while (true) {
        const parentId = parentMap.get(currentId);
        if (!parentId) {
          // No more parents, node is not under a collapsed parent
          return true;
        }

        const parentState = nodeStates.get(parentId);
        if (parentState) {
          // Found a parent in nodeStates - check if it's collapsed
          if (parentState.collapsed) {
            return false; // Parent is collapsed, filter out this node
          }
          // Parent exists and is not collapsed, continue up the chain
          currentId = parentId;
        } else {
          // Parent doesn't exist in nodeStates (was filtered out)
          // This means the parent was filtered for some reason (likely collapsed grandparent)
          // Continue walking up to find the ancestor that exists
          currentId = parentId;
        }
      }
    });

    return result;
  }

  /**
   * Build hierarchical relationships between nodes
   * Creates parent-child relationships for compound graph display
   *
   * This function:
   * 1. Parses Bazel labels to extract package paths and target names
   * 2. Creates intermediate package/directory nodes as needed
   * 3. Only creates package wrappers when there are multiple children
   * 4. Sets parent property on child nodes
   *
   * @param {Array} visibleNodes - Array of visible nodes
   * @param {Map} nodeStates - Node state map
   * @returns {Array} Array with both original nodes and generated package nodes
   */
  buildHierarchy(visibleNodes, nodeStates) {
    console.log('[LensRenderer] Building hierarchy for', visibleNodes.length, 'nodes');

    // Map to track all nodes by their ID
    const allNodes = new Map();

    // Map to track children of each potential parent
    const childrenMap = new Map();

    // First pass: Parse labels and track potential parent-child relationships
    visibleNodes.forEach(node => {
      const nodeId = node.id || node.label;
      allNodes.set(nodeId, { ...node });

      // Parse Bazel label: //package/path:target or //package/path
      const label = node.label;

      if (label.startsWith('//')) {
        const parts = label.substring(2).split(':');
        const packagePath = parts[0];
        const targetName = parts.length > 1 ? parts[1] : null;

        if (targetName) {
          // This is a target (//foo/bar:baz)
          // Its parent should be the package (//foo/bar)
          const packageLabel = '//' + packagePath;

          if (!childrenMap.has(packageLabel)) {
            childrenMap.set(packageLabel, []);
          }
          childrenMap.get(packageLabel).push(nodeId);

          // Also track directory hierarchy for the package path
          const pathParts = packagePath.split('/').filter(p => p.length > 0);
          for (let i = 1; i <= pathParts.length; i++) {
            const parentPath = '//' + pathParts.slice(0, i).join('/');
            const childPath = i === pathParts.length ?
              packageLabel :
              '//' + pathParts.slice(0, i + 1).join('/');

            if (i < pathParts.length) {
              if (!childrenMap.has(parentPath)) {
                childrenMap.set(parentPath, []);
              }
              if (!childrenMap.get(parentPath).includes(childPath)) {
                childrenMap.get(parentPath).push(childPath);
              }
            }
          }
        } else {
          // This is a package node (//foo/bar)
          // Track directory hierarchy
          const pathParts = packagePath.split('/').filter(p => p.length > 0);
          for (let i = 1; i < pathParts.length; i++) {
            const parentPath = '//' + pathParts.slice(0, i).join('/');
            const childPath = '//' + pathParts.slice(0, i + 1).join('/');

            if (!childrenMap.has(parentPath)) {
              childrenMap.set(parentPath, []);
            }
            if (!childrenMap.get(parentPath).includes(childPath)) {
              childrenMap.get(parentPath).push(childPath);
            }
          }
        }
      }
    });

    // Second pass: Create package/directory nodes only where needed
    const packageNodes = new Map();

    childrenMap.forEach((children, parentLabel) => {
      // Only create package node if:
      // 1. It has multiple children, OR
      // 2. It doesn't already exist as a real node
      if (children.length > 1 || !allNodes.has(parentLabel)) {
        if (!allNodes.has(parentLabel)) {
          // Create synthetic package node
          packageNodes.set(parentLabel, {
            id: parentLabel,
            label: parentLabel,
            type: 'package',
            synthetic: true  // Mark as generated
          });
        }
      }
    });

    // Third pass: Set parent relationships
    const result = [];

    // Add all original nodes with parent set
    allNodes.forEach((node, nodeId) => {
      const label = node.label;
      let parent = null;

      if (label.startsWith('//')) {
        const parts = label.substring(2).split(':');
        const packagePath = parts[0];
        const targetName = parts.length > 1 ? parts[1] : null;

        if (targetName) {
          // Target node: parent is package
          const packageLabel = '//' + packagePath;
          const siblings = childrenMap.get(packageLabel) || [];

          // Only set parent if package has multiple children or package node exists
          if (siblings.length > 1 || packageNodes.has(packageLabel) || allNodes.has(packageLabel)) {
            parent = packageLabel;
          }
        } else {
          // Package node: parent is parent directory
          const pathParts = packagePath.split('/').filter(p => p.length > 0);
          if (pathParts.length > 1) {
            const parentPath = '//' + pathParts.slice(0, -1).join('/');
            const siblings = childrenMap.get(parentPath) || [];

            // Only set parent if parent directory has multiple children
            if (siblings.length > 1 || packageNodes.has(parentPath)) {
              parent = parentPath;
            }
          }
        }
      }

      if (parent) {
        result.push({ ...node, parent });
      } else {
        result.push(node);
      }
    });

    // Add synthetic package nodes with their parents set
    packageNodes.forEach((node, nodeId) => {
      const label = node.label;
      let parent = null;

      if (label.startsWith('//')) {
        const packagePath = label.substring(2);
        const pathParts = packagePath.split('/').filter(p => p.length > 0);

        if (pathParts.length > 1) {
          const parentPath = '//' + pathParts.slice(0, -1).join('/');
          const siblings = childrenMap.get(parentPath) || [];

          // Only set parent if parent directory has multiple children
          if (siblings.length > 1 || packageNodes.has(parentPath)) {
            parent = parentPath;
          }
        }
      }

      if (parent) {
        result.push({ ...node, parent });
      } else {
        result.push(node);
      }
    });

    console.log('[LensRenderer] Hierarchy built:', result.length, 'total nodes (',
                packageNodes.size, 'synthetic package nodes)');

    return {
      nodes: result,
      syntheticNodes: Array.from(packageNodes.values())
    };
  }

  /**
   * Extract package nodes from target labels
   * Creates synthetic package nodes for all packages that contain targets
   * This is done before visibility filtering to ensure packages exist even when targets are hidden
   *
   * @param {Object} graph - Raw graph data
   * @returns {Array} Array of synthetic package nodes
   */
  extractPackageNodes(graph) {
    const packageSet = new Set();

    // Extract package paths from all target labels
    graph.nodes.forEach(node => {
      const label = node.label;
      if (label && label.startsWith('//') && label.includes(':')) {
        // This is a target (//package/path:target)
        const parts = label.substring(2).split(':');
        const packagePath = parts[0];
        const packageLabel = '//' + packagePath;

        packageSet.add(packageLabel);
      }
    });

    // Create synthetic package nodes
    const packageNodes = Array.from(packageSet).map(packageLabel => ({
      id: packageLabel,
      label: packageLabel,
      type: 'package',
      synthetic: true
    }));

    console.log('[LensRenderer] Extracted', packageNodes.length, 'package nodes from targets');
    return packageNodes;
  }

  /**
   * Build a map of child node ID to parent node ID
   * Used for aggregating edges when nodes are collapsed
   *
   * @param {Array} nodes - All nodes
   * @param {Map} nodeStates - Node state map
   * @returns {Map<string, string>} Map of child ID to parent ID
   */
  buildChildToParentMap(nodes, nodeStates) {
    const childToParent = new Map();

    nodes.forEach(node => {
      const nodeId = node.id || node.label;
      const label = node.label;

      // File nodes: parent is the target
      if (node.type === 'source_file' || node.type === 'header_file') {
        const parts = nodeId.split(':');
        if (parts.length >= 3) {
          const parentLabel = parts[0] + ':' + parts[1];
          childToParent.set(nodeId, parentLabel);
        }
      }
      // Target nodes: parent is the package
      else if (label.startsWith('//') && label.includes(':')) {
        const parts = label.substring(2).split(':');
        if (parts.length > 1) {
          const packageLabel = '//' + parts[0];
          childToParent.set(nodeId, packageLabel);
        }
      }
    });

    return childToParent;
  }

  /**
   * Aggregate edges for collapsed nodes
   * When a node is collapsed, edges to/from its children are redirected to the parent
   *
   * @param {Object} graph - Raw graph data
   * @param {Map} nodeStates - Node state map
   * @param {Object} viewState - Current view state
   * @param {Set} includedNodeIds - Set of node IDs actually rendered
   * @param {Map} childToParentMap - Map of child to parent IDs
   * @returns {Array} Array of aggregated edges
   */
  aggregateEdgesForCollapsedNodes(graph, nodeStates, viewState, includedNodeIds, childToParentMap) {
    // First, get all edges (skip node visibility check since we'll redirect to visible parents)
    const baseEdges = this.processEdges(graph, nodeStates, viewState, null, true);

    // Map to track aggregated edges (key: "source->target->type", value: edge data)
    const aggregatedEdges = new Map();
    let redirectedCount = 0;

    baseEdges.forEach(edge => {
      let source = edge.source;
      let target = edge.target;
      const originalSource = source;
      const originalTarget = target;

      // If source is not in included nodes (it's hidden by collapse), redirect to parent
      while (!includedNodeIds.has(source) && childToParentMap.has(source)) {
        source = childToParentMap.get(source);
      }

      // If target is not in included nodes (it's hidden by collapse), redirect to parent
      while (!includedNodeIds.has(target) && childToParentMap.has(target)) {
        target = childToParentMap.get(target);
      }

      // Track redirections
      if (source !== originalSource || target !== originalTarget) {
        redirectedCount++;
      }

      // Skip edges where we couldn't find valid endpoints
      if (!includedNodeIds.has(source) || !includedNodeIds.has(target)) {
        return;
      }

      // Skip self-loops (edges from a node to itself)
      if (source === target) {
        return;
      }

      // Create aggregated edge key
      const edgeKey = `${source}->${target}->${edge.type}`;

      // Store or merge the edge
      if (!aggregatedEdges.has(edgeKey)) {
        aggregatedEdges.set(edgeKey, {
          source,
          target,
          type: edge.type,
          aggregated: source !== edge.source || target !== edge.target
        });
      }
    });

    if (redirectedCount > 0) {
      console.log(`[EdgeAggregation] Redirected ${redirectedCount} edges, resulted in ${aggregatedEdges.size} aggregated edges`);
    }

    return Array.from(aggregatedEdges.values());
  }

  /**
   * Process edges based on node visibility and edge rules
   *
   * @param {Object} graph - Raw graph data
   * @param {Map} nodeStates - Node state map
   * @param {Object} viewState - Current view state
   * @param {Set} includedNodeIds - Set of node IDs that are actually included in the render (null to skip this check)
   * @param {boolean} skipNodeVisibilityCheck - If true, skip checking if source/target nodes are visible (for edge aggregation)
   * @returns {Array} Array of visible edges
   */
  processEdges(graph, nodeStates, viewState, includedNodeIds, skipNodeVisibilityCheck = false) {
    return graph.edges.filter(edge => {
      // First check if both source and target nodes are actually included in the render
      // (they might be filtered out due to collapse)
      if (includedNodeIds && (!includedNodeIds.has(edge.source) || !includedNodeIds.has(edge.target))) {
        return false;
      }

      const sourceState = nodeStates.get(edge.source);
      const targetState = nodeStates.get(edge.target);

      // Only show edge if both nodes exist in nodeStates
      if (!sourceState || !targetState) return false;

      // Check node visibility unless we're aggregating edges (where invisible nodes will be redirected to parents)
      if (!skipNodeVisibilityCheck) {
        if (!sourceState.visible || !targetState.visible) return false;
      }

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
