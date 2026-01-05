// Use structured logger (loaded from logger.js)
const appLogger = new Logger();

// ===== Cytoscape Graph Style Configuration =====
// Centralized color palette and style definitions to reduce duplication

const GRAPH_COLORS = {
  // Node colors
  library: '#4fc1ff',
  binary: '#ff8c00',
  sharedLib: '#c586c0',
  systemLib: '#d7ba7d',
  source: '#89d185',
  header: '#4fc1ff',
  uncovered: '#ff6b6b',
  external: '#6a6a6a',
  package: '#4a4a4e',
  targetGroup: '#2d2d30',

  // Edge colors
  teal: '#4ec9b0',
  blue: '#4fc1ff',
  purple: '#c586c0',
  gold: '#d7ba7d',
  lightBlue: '#9cdcfe',
  gray: '#6a6a6a',

  // State colors
  selected: '#ff8c00',
  overlap: '#ff4444',
  publicVis: '#ffd700',

  // Text colors
  textLight: '#1e1e1e',
  textDark: '#cccccc',
  textWhite: 'white',
  textGray: '#969696',

  // Border colors
  borderDark: '#3e3e42',
  borderGray: '#666666',
  borderMedium: '#696969',
  borderTargetGroup: '#4a4a4e',
};

// Helper to create edge style with common properties
const edgeStyle = (color, width = 2, lineStyle = 'solid') => ({
  'line-color': color,
  'target-arrow-color': color,
  width: width,
  'line-style': lineStyle,
});

// Helper to create basic node style
const nodeStyle = (bgColor, textColor, borderColor = null) => ({
  'background-color': bgColor,
  color: textColor,
  'border-color': borderColor || GRAPH_COLORS.borderDark,
});

// Helper for ellipse file nodes
const fileNodeStyle = (bgColor, borderColor) => ({
  ...nodeStyle(bgColor, GRAPH_COLORS.textLight, borderColor),
  shape: 'ellipse',
  width: '60px',
  height: '60px',
});

// Global reference to the info popup element (singleton)
let infoPopup = null;

// Clear/hide the info popup (with optional fade animation)
function clearInfoPopup(fade = false) {
  if (infoPopup) {
    if (fade) {
      infoPopup.style.opacity = '0';
      infoPopup.style.transition = 'opacity 0.2s ease-out';
      setTimeout(() => {
        infoPopup.style.display = 'none';
        infoPopup.style.opacity = '1'; // Reset for next show
      }, 200);
    } else {
      infoPopup.style.display = 'none';
    }
  }
}

// Update loading checklist progress
// completedStep: the step that just finished (will show ✓)
// activeStep: the step that is now running (will show spinner), or null if all done
function updateLoadingProgress(completedStep, activeStep = null) {
  // Mark the completed step
  if (completedStep) {
    const item = document.querySelector(`.loading-checklist-item[data-step="${completedStep}"]`);
    if (item) {
      item.classList.remove('active');
      item.classList.add('completed');
    }
  }

  // Mark the active step (if any)
  if (activeStep) {
    const item = document.querySelector(`.loading-checklist-item[data-step="${activeStep}"]`);
    if (item) {
      item.classList.add('active');
    }
  }
}

// Hide loading overlay
function hideLoadingOverlay() {
  const overlay = document.getElementById('loadingOverlay');
  if (overlay) {
    overlay.style.display = 'none';
  }
}

// Simplify C++ symbol names by reducing template verbosity
function simplifySymbol(symbol) {
  if (!symbol) return symbol;

  // Replace std::__1:: with std:: for brevity
  let simplified = symbol.replace(/std::__1::/g, 'std::');

  // Simplify common std templates
  simplified = simplified.replace(
    /std::basic_string<char,\s*std::char_traits<char>,\s*std::allocator<char>\s*>/g,
    'std::string'
  );
  simplified = simplified.replace(
    /std::basic_string<char,\s*std::char_traits<char>>/g,
    'std::string'
  );

  // Simplify allocator types in templates
  simplified = simplified.replace(/,\s*std::allocator<[^>]+>\s*>/g, '>');

  // Simplify char_traits in templates
  simplified = simplified.replace(/,\s*std::char_traits<char>/g, '');

  // Remove extra spaces after commas in templates
  simplified = simplified.replace(/,\s+/g, ', ');

  // Collapse multiple spaces
  simplified = simplified.replace(/\s+/g, ' ');

  return simplified.trim();
}

// Simplify Bazel target labels by removing redundant target names
// e.g., //foo:foo -> //foo, //bar/baz:baz -> //bar/baz
function simplifyLabel(label) {
  if (!label) return label;

  // Match pattern //package:target or //package/subpackage:target
  const match = label.match(/^(\/\/[^:]+):([^:]+)$/);
  if (!match) return label;

  const packagePath = match[1]; // e.g., "//foo" or "//bar/baz"
  const targetName = match[2]; // e.g., "foo" or "baz"

  // Get the last component of the package path
  const packageParts = packagePath.split('/');
  const lastPackage = packageParts[packageParts.length - 1];

  // If target name matches the last package component, simplify
  if (targetName === lastPackage) {
    return packagePath;
  }

  return label;
}

// Track last update time for watching indicator
let lastUpdateTime = null;
let watchingUpdateInterval = null;

// Update watching indicator
function updateWatchingIndicator(watching) {
  const statusBar = document.querySelector('.status-bar');
  let indicator = document.getElementById('watchingIndicator');

  if (watching && !indicator) {
    lastUpdateTime = Date.now();
    indicator = document.createElement('div');
    indicator.id = 'watchingIndicator';
    indicator.className = 'watching-badge';
    statusBar.appendChild(indicator);

    // Update immediately
    updateWatchingText();

    // Start interval to update "time ago" text every 10 seconds
    if (watchingUpdateInterval) {
      clearInterval(watchingUpdateInterval);
    }
    watchingUpdateInterval = setInterval(updateWatchingText, 10000);
  } else if (!watching && indicator) {
    indicator.remove();
    if (watchingUpdateInterval) {
      clearInterval(watchingUpdateInterval);
      watchingUpdateInterval = null;
    }
  }
}

// Update the watching indicator text with time since last update
function updateWatchingText() {
  const indicator = document.getElementById('watchingIndicator');
  if (!indicator || !lastUpdateTime) return;

  const secondsAgo = Math.floor((Date.now() - lastUpdateTime) / 1000);
  let timeText;

  if (secondsAgo < 10) {
    timeText = 'just now';
  } else if (secondsAgo < 60) {
    timeText = `${secondsAgo}s ago`;
  } else if (secondsAgo < 3600) {
    const minutes = Math.floor(secondsAgo / 60);
    timeText = `${minutes}m ago`;
  } else {
    const hours = Math.floor(secondsAgo / 3600);
    timeText = `${hours}h ago`;
  }

  indicator.innerHTML = `<span style="opacity: 0.6;">●</span> Watching · Updated ${timeText}`;
}

// Update last update time when analysis completes
function markAnalysisUpdate() {
  lastUpdateTime = Date.now();
  updateWatchingText();
}

// Update the subtitle with the module/workspace name and path
function updateModuleName(name, workspacePath) {
  const subtitle = document.querySelector('.subtitle');
  if (subtitle) {
    if (name && workspacePath) {
      subtitle.textContent = `${name} • ${workspacePath}`;
    } else if (name) {
      subtitle.textContent = name;
    } else if (workspacePath) {
      subtitle.textContent = workspacePath;
    }
  }
}

// Show notification
function showNotification(message, duration = 3000) {
  const notif = document.createElement('div');
  notif.className = 'notification';
  notif.textContent = message;
  document.body.appendChild(notif);

  setTimeout(() => {
    notif.classList.add('fade-out');
    setTimeout(() => notif.remove(), 300);
  }, duration);
}

// Position caching is no longer needed - we do incremental Cytoscape updates
// Nodes that already exist keep their positions automatically when we only add/remove changed elements

function displayDependencyGraph(graphData) {
  appLogger.debug('displayDependencyGraph called with', graphData.nodes?.length, 'nodes');

  // Clear any visible info popups when graph is re-rendered
  clearInfoPopup();

  // Show the graph section (in case it's hidden)
  const graphSection = document.getElementById('graphSection');
  if (graphSection) {
    graphSection.style.display = 'flex';
  }

  // Hide the graph loading spinner
  const graphLoading = document.getElementById('graphLoading');
  if (graphLoading) {
    graphLoading.style.display = 'none';
  }

  const isInitialLoad = !cy;

  // Create elements array
  const elements = [
    // Nodes
    ...graphData.nodes.map((node) => {
      const nodeData = {
        id: node.id,
        label: simplifyLabel(node.label),
        type: node.type,
        parent: node.parent, // For compound nodes (grouping)
      };

      // Mark selected nodes
      const selectedNodes = viewStateManager.getState().selectedNodes;
      if (selectedNodes.has(node.id) || selectedNodes.has(node.label)) {
        nodeData.selected = true;
      }

      // Only set hasOverlap if it's true (don't set it at all if false)
      if (node.hasOverlap === true) {
        nodeData.hasOverlap = true;
      }

      // Only set isPublic if it's true (don't set it at all if false)
      if (node.isPublic === true) {
        nodeData.isPublic = true;
      }

      // Add overlapping metadata for tooltips
      if (node.overlappingTargets && node.overlappingTargets.length > 0) {
        nodeData.overlappingTargets = node.overlappingTargets;
      }
      if (node.overlappingWith && node.overlappingWith.length > 0) {
        nodeData.overlappingWith = node.overlappingWith;
      }

      return { data: nodeData };
    }),
    // Edges
    ...graphData.edges.map((edge) => {
      const edgeData = {
        source: edge.source,
        target: edge.target,
        type: edge.type,
        linkage: edge.linkage,
        symbols: edge.symbols || [],
        sourceLabel: edge.sourceLabel,
        targetLabel: edge.targetLabel,
        fileDetails: edge.fileDetails || {},
      };
      // Only set isOverlapping if it's true (don't set it at all if false)
      if (edge.isOverlapping === true) {
        edgeData.isOverlapping = true;
      }
      return { data: edgeData };
    }),
  ];

  // Debug: Log overlapping flags
  const overlappingNodes = elements.filter((e) => e.data.hasOverlap === true);
  const overlappingEdges = elements.filter((e) => e.data.isOverlapping === true);
  appLogger.debug(
    'Nodes with hasOverlap=true:',
    overlappingNodes.map((n) => n.data.id)
  );
  appLogger.debug(
    'Edges with isOverlapping=true:',
    overlappingEdges.map((e) => `${e.data.source} -> ${e.data.target}`)
  );

  // Cytoscape stylesheet (shared between initial and incremental updates)
  const cytoscapeStylesheet = [
    {
      selector: 'node',
      style: {
        shape: 'roundrectangle',
        'background-color': '#4fc1ff',
        label: 'data(label)',
        color: '#1e1e1e',
        'text-valign': 'center',
        'text-halign': 'center',
        'font-size': '12px',
        'font-family': '-apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
        'font-weight': '600',
        'text-wrap': 'wrap',
        'text-max-width': '140px',
        width: 'label',
        height: 'label',
        padding: '14px',
        'border-width': '2px',
        'border-color': '#3e3e42',
      },
    },
    {
      selector: ':parent',
      style: {
        'text-valign': 'top',
        'text-halign': 'center',
        'padding-top': '30px',
        'padding-left': '20px',
        'padding-right': '20px',
        'padding-bottom': '20px',
        'background-opacity': 0.1,
        'border-width': '2px',
        'border-color': '#666666',
        'border-opacity': 0.5,
        'compound-sizing-wrt-labels': 'include',
        'min-width': '100px',
        'min-height': '100px',
      },
    },
    // Target node types
    {
      selector: 'node[type = "cc_binary"]',
      style: nodeStyle(GRAPH_COLORS.binary, GRAPH_COLORS.textWhite, '#cc7000'),
    },
    {
      selector: 'node[type = "cc_shared_library"]',
      style: nodeStyle(GRAPH_COLORS.sharedLib, GRAPH_COLORS.textWhite, '#9d6b99'),
    },
    {
      selector: 'node[type = "system_library"]',
      style: {
        ...nodeStyle(GRAPH_COLORS.systemLib, GRAPH_COLORS.textLight, '#b89b5d'),
        shape: 'hexagon',
      },
    },
    // File nodes
    {
      selector: 'node[type = "source"], node[type ^= "source"]',
      style: fileNodeStyle(GRAPH_COLORS.source, '#6fb06b'),
    },
    {
      selector: 'node[type = "header"], node[type ^= "header"]',
      style: fileNodeStyle(GRAPH_COLORS.header, '#3fa0d9'),
    },
    // Uncovered file nodes (shared style)
    {
      selector: 'node[type = "uncovered_source"], node[type = "uncovered_header"]',
      style: {
        ...fileNodeStyle(GRAPH_COLORS.uncovered, GRAPH_COLORS.overlap),
        'border-width': '3px',
        'border-style': 'double',
        opacity: 0.7,
        color: GRAPH_COLORS.textWhite,
        'font-size': '10px',
      },
    },
    // Other node types
    {
      selector: 'node[type = "external"]',
      style: nodeStyle(GRAPH_COLORS.external, GRAPH_COLORS.textDark, '#505050'),
    },
    {
      selector: 'node[type = "package"]',
      style: {
        ...nodeStyle(GRAPH_COLORS.package, GRAPH_COLORS.textDark, GRAPH_COLORS.borderMedium),
        shape: 'roundrectangle',
        'font-weight': 'bold',
        'font-size': '14px',
        padding: '18px',
      },
    },
    // Node state modifiers
    {
      selector: 'node[type $= "_selected"]',
      style: {
        ...nodeStyle(GRAPH_COLORS.selected, GRAPH_COLORS.textWhite, GRAPH_COLORS.selected),
        'border-width': '4px',
        'font-weight': 'bold',
      },
    },
    {
      selector: 'node[type $= "_incoming"]',
      style: nodeStyle(GRAPH_COLORS.teal, GRAPH_COLORS.textLight, '#3da889'),
    },
    {
      selector: 'node[type $= "_outgoing"]',
      style: nodeStyle(GRAPH_COLORS.sharedLib, GRAPH_COLORS.textWhite, '#9d6b99'),
    },
    // Target groups
    {
      selector: 'node[type = "target-group"]',
      style: {
        shape: 'roundrectangle',
        'background-color': GRAPH_COLORS.targetGroup,
        'background-opacity': 0.3,
        'border-width': '2px',
        'border-color': GRAPH_COLORS.borderTargetGroup,
        'border-style': 'solid',
        label: 'data(label)',
        color: GRAPH_COLORS.textGray,
        'text-valign': 'top',
        'text-halign': 'center',
        'font-size': '14px',
        'font-weight': 'bold',
        padding: '30px',
      },
    },
    {
      selector: 'node[type = "target-group"][selected]',
      style: {
        'border-width': '4px',
        'border-color': GRAPH_COLORS.selected,
        'background-opacity': 0.4,
        color: GRAPH_COLORS.selected,
      },
    },
    // ===== Edge Styles =====
    // Base edge style
    {
      selector: 'edge',
      style: {
        ...edgeStyle(GRAPH_COLORS.gray, 2),
        'target-arrow-shape': 'triangle',
        'curve-style': 'bezier',
        'arrow-scale': 1.5,
      },
    },
    // File edges (use default gray)
    {
      selector: 'edge[type = "file"]',
      style: edgeStyle(GRAPH_COLORS.gray, 2),
    },
    // Symbol edges (base style: dashed, thin)
    {
      selector: 'edge[type = "symbol"]',
      style: {
        'line-style': 'dashed',
        width: 1.5,
      },
    },
    // Symbol edges by linkage type
    {
      selector: 'edge[type = "symbol"][linkage = "static"]',
      style: edgeStyle(GRAPH_COLORS.teal),
    },
    {
      selector: 'edge[type = "symbol"][linkage = "dynamic"]',
      style: edgeStyle(GRAPH_COLORS.purple),
    },
    {
      selector: 'edge[type = "symbol"][linkage = "cross"]',
      style: edgeStyle(GRAPH_COLORS.gold),
    },
    // Dependency edge types
    {
      selector: 'edge[type = "static"]',
      style: edgeStyle(GRAPH_COLORS.teal, 2, 'solid'),
    },
    {
      selector: 'edge[type = "dynamic"]',
      style: edgeStyle(GRAPH_COLORS.teal, 2, 'dashed'),
    },
    {
      selector: 'edge[type = "dynamic_link"]',
      style: edgeStyle(GRAPH_COLORS.purple, 3, 'solid'),
    },
    {
      selector: 'edge[type = "data"]',
      style: edgeStyle(GRAPH_COLORS.teal, 2, 'dotted'),
    },
    {
      selector: 'edge[type = "system_link"]',
      style: edgeStyle(GRAPH_COLORS.teal, 2, 'dashed'),
    },
    {
      selector: 'edge[type = "compile"]',
      style: edgeStyle(GRAPH_COLORS.blue, 2, 'solid'),
    },
    {
      selector: 'edge[type = "multi"]',
      style: edgeStyle(GRAPH_COLORS.lightBlue, 3, 'solid'),
    },
    // ===== State Overlays (must come after base styles) =====
    // Selection indicators
    {
      selector: 'node:selected',
      style: {
        'border-width': '3px',
        'border-color': GRAPH_COLORS.selected,
      },
    },
    // Public visibility indicator (solid gold border)
    {
      selector: 'node[isPublic]',
      style: {
        'border-style': 'solid',
        'border-width': '3px',
        'border-color': GRAPH_COLORS.publicVis,
      },
    },
    // Selected node styling - MUST come after isPublic to override it
    {
      selector: 'node[selected]',
      style: {
        'border-width': '4px',
        'border-color': GRAPH_COLORS.selected,
        'border-style': 'solid',
      },
    },
    // Overlapping dependencies - MUST come after other border styles to be visible
    {
      selector:
        'node[hasOverlap][type = "cc_binary"], node[hasOverlap][type = "cc_shared_library"], node[hasOverlap][type = "cc_library"]',
      style: {
        'border-width': '8px',
        'border-color': GRAPH_COLORS.overlap,
        'border-style': 'double',
      },
    },
    {
      selector: 'edge[isOverlapping]',
      style: edgeStyle(GRAPH_COLORS.overlap, 4, 'solid'),
    },
  ];

  if (isInitialLoad) {
    // Initial creation - create new cytoscape instance
    appLogger.debug('Creating new cytoscape instance with', elements.length, 'elements');

    cy = cytoscape({
      container: document.getElementById('cy'),
      elements: elements,
      style: cytoscapeStylesheet,
      layout: {
        name: 'preset', // Use preset to avoid initial layout, we'll run it manually
      },
    });

    // Setup event handlers (only on initial load)
    setupEventHandlers();

    // Run stable layout without animation on initial load
    runStableDagreLayout(false, true);
  } else {
    // Incremental update - update only changed elements
    appLogger.debug('Incrementally updating cytoscape with', elements.length, 'elements');

    cy.startBatch();

    // Get current element IDs
    const currentNodeIds = new Set(cy.nodes().map((n) => n.id()));
    const currentEdgeIds = new Set(
      cy.edges().map((e) => `${e.data('source')}|${e.data('target')}|${e.data('type')}`)
    );

    // Build sets of new element IDs
    const newNodeIds = new Set(elements.filter((e) => !e.data.source).map((e) => e.data.id));
    const newEdgeIds = new Set(
      elements
        .filter((e) => e.data.source)
        .map((e) => `${e.data.source}|${e.data.target}|${e.data.type}`)
    );

    // Remove elements that no longer exist
    const nodesToRemove = Array.from(currentNodeIds).filter((id) => !newNodeIds.has(id));
    const edgesToRemove = Array.from(currentEdgeIds).filter((id) => !newEdgeIds.has(id));

    appLogger.debug(
      `[Cytoscape] Removing ${nodesToRemove.length} nodes, ${edgesToRemove.length} edges`
    );
    nodesToRemove.forEach((id) => cy.getElementById(id).remove());
    edgesToRemove.forEach((id) => {
      const [source, target, type] = id.split('|');
      cy.edges(`[source = "${source}"][target = "${target}"][type = "${type}"]`).remove();
    });

    // Update existing nodes and add new elements
    const elementsToAdd = [];
    let updatedNodes = 0;

    elements.forEach((e) => {
      if (e.data.source) {
        // Edge
        const id = `${e.data.source}|${e.data.target}|${e.data.type}`;
        if (!currentEdgeIds.has(id)) {
          elementsToAdd.push(e);
        }
      } else {
        // Node
        if (currentNodeIds.has(e.data.id)) {
          // Update existing node data (all fields, not just label)
          const existingNode = cy.getElementById(e.data.id);
          // Update all data fields from the new element
          Object.keys(e.data).forEach((key) => {
            if (existingNode.data(key) !== e.data[key]) {
              existingNode.data(key, e.data[key]);
            }
          });
          updatedNodes++;
        } else {
          // Add new node
          elementsToAdd.push(e);
        }
      }
    });

    appLogger.debug(
      `[Cytoscape] Adding ${elementsToAdd.filter((e) => !e.data.source).length} nodes, ${elementsToAdd.filter((e) => e.data.source).length} edges, updated ${updatedNodes} node labels`
    );
    if (elementsToAdd.length > 0) {
      cy.add(elementsToAdd);

      // Position new nodes at their parent's location for smooth expand animation
      const newNodes = elementsToAdd.filter((e) => !e.data.source);
      newNodes.forEach((nodeData) => {
        const node = cy.getElementById(nodeData.data.id);
        const parent = node.parent();

        if (parent && parent.length > 0) {
          // Position at parent's current location
          const parentPos = parent.position();
          node.position({ x: parentPos.x, y: parentPos.y });
          appLogger.debug(`[Cytoscape] Positioned new node ${nodeData.data.id} at parent location`);
        }
      });
    }

    cy.endBatch();

    // Only run layout if nodes were added/removed
    // If only edges changed, nodes keep their positions (no movement!)
    const nodesChanged = nodesToRemove.length > 0 || elementsToAdd.some((e) => !e.data.source);

    if (nodesChanged) {
      appLogger.debug('[Cytoscape] Running layout because nodes changed');
      runStableDagreLayout(true, false); // Animate, don't reset viewport
    } else {
      appLogger.debug('[Cytoscape] Skipping layout - only edges changed, nodes stay in place');
    }
  }
}

/**
 * Setup all event handlers for the cytoscape instance
 * Only called once during initial graph creation
 */
function setupEventHandlers() {
  if (!cy) return;

  // Create info popup element if it doesn't exist (singleton pattern)
  if (!infoPopup) {
    infoPopup = document.createElement('div');
    infoPopup.id = 'edge-tooltip';
    infoPopup.style.position = 'absolute';
    infoPopup.style.display = 'none';
    infoPopup.style.backgroundColor = 'rgba(0, 0, 0, 0.85)';
    infoPopup.style.color = 'white';
    infoPopup.style.padding = '8px 12px';
    infoPopup.style.borderRadius = '4px';
    infoPopup.style.fontSize = '12px';
    infoPopup.style.maxWidth = '400px';
    infoPopup.style.zIndex = '10000';
    infoPopup.style.pointerEvents = 'none';
    infoPopup.style.whiteSpace = 'pre-wrap';
    infoPopup.style.fontFamily = 'monospace';
    infoPopup.style.opacity = '1'; // For fade animations
    document.body.appendChild(infoPopup);
  }

  // Use local reference for convenience
  const tooltip = infoPopup;

  // Tooltip hover delay
  let tooltipTimeout = null;

  // Show tooltip on edge hover with delay
  cy.on('mouseover', 'edge', (evt) => {
    // Clear any existing timeout
    if (tooltipTimeout) {
      clearTimeout(tooltipTimeout);
    }

    // Set timeout for 500ms delay
    tooltipTimeout = setTimeout(() => {
      const edge = evt.target;
      const edgeType = edge.data('type');
      const linkage = edge.data('linkage');
      const symbols = edge.data('symbols');
      const fileDetails = edge.data('fileDetails');

      // Get source and target node labels (prefer edge labels if available)
      const sourceLabel =
        edge.data('sourceLabel') ||
        (() => {
          const sourceNode = cy.getElementById(edge.data('source'));
          return sourceNode.data('label') || edge.data('source');
        })();
      const targetLabel =
        edge.data('targetLabel') ||
        (() => {
          const targetNode = cy.getElementById(edge.data('target'));
          return targetNode.data('label') || edge.data('target');
        })();

      let tooltipText = '';

      // Add description based on edge type with directional information
      if (edgeType === 'static') {
        tooltipText = `📦 Static Linkage\n\n${sourceLabel}\n  depends on (statically links)\n${targetLabel}\n\nCode from ${targetLabel} is included in ${sourceLabel} at link time.`;

        // Add file-level compile details if available
        if (fileDetails && Object.keys(fileDetails).length > 0) {
          tooltipText += '\n\nHeader Includes:';
          const entries = Object.entries(fileDetails).slice(0, 10);
          for (const [sourceFile, targetFiles] of entries) {
            tooltipText += `\n  ${sourceFile} → ${targetFiles}`;
          }
          if (Object.keys(fileDetails).length > 10) {
            tooltipText += `\n  ... and ${Object.keys(fileDetails).length - 10} more files`;
          }
        }

        // Add symbols if available
        if (symbols && symbols.length > 0) {
          const simplifiedSymbols = symbols.map((s) => simplifySymbol(s));
          const symbolList = simplifiedSymbols.slice(0, 10).join(', ');
          const more = symbols.length > 10 ? ` ... +${symbols.length - 10} more` : '';
          tooltipText += `\n\nSymbols (${symbols.length}): ${symbolList}${more}`;
        }
      } else if (edgeType === 'dynamic') {
        tooltipText = `🔗 Dynamic Linkage\n\n${sourceLabel}\n  depends on (dynamically links)\n${targetLabel}\n\nShared library ${targetLabel} is loaded at runtime.`;

        // Add symbols if available
        if (symbols && symbols.length > 0) {
          const simplifiedSymbols = symbols.map((s) => simplifySymbol(s));
          const symbolList = simplifiedSymbols.slice(0, 10).join(', ');
          const more = symbols.length > 10 ? ` ... +${symbols.length - 10} more` : '';
          tooltipText += `\n\nSymbols (${symbols.length}): ${symbolList}${more}`;
        }
      } else if (edgeType === 'data') {
        tooltipText = `📄 Data Dependency\n\n${sourceLabel}\n  needs at runtime\n${targetLabel}\n\nSpecified in 'data' attribute.`;
      } else if (edgeType === 'compile') {
        tooltipText = `📝 Compile Dependency\n\n${sourceLabel}\n  #includes header\n${targetLabel}\n\nDetected from .d files (compiler dependency output).`;
      } else if (edgeType === 'system_link') {
        tooltipText = `⚙️ System Library Link\n\n${sourceLabel}\n  links against system library\n${targetLabel}\n\nSpecified in linkopts (-l${targetLabel}).`;
      } else if (edgeType === 'symbol') {
        const linkageDesc =
          linkage === 'static'
            ? 'statically linked'
            : linkage === 'dynamic'
              ? 'dynamically linked'
              : linkage === 'cross'
                ? 'cross-binary'
                : linkage;
        tooltipText = `🔧 Symbol Dependency (${linkageDesc})\n\n${sourceLabel}\n  uses symbols from\n${targetLabel}`;

        // Add symbol list for symbol edges
        if (symbols && symbols.length > 0) {
          const simplifiedSymbols = symbols.map((s) => simplifySymbol(s));
          const symbolList = simplifiedSymbols.slice(0, 15).join('\n  ');
          const more = symbols.length > 15 ? `\n  ... and ${symbols.length - 15} more` : '';
          tooltipText += `\n\nSymbols used (${symbols.length}):\n  ${symbolList}${more}`;
        }
      } else {
        tooltipText = `Dependency: ${sourceLabel} → ${targetLabel}\nType: ${edgeType || 'unknown'}`;
      }

      tooltip.textContent = tooltipText;
      // Fade in animation
      tooltip.style.display = 'block';
      tooltip.style.opacity = '0';
      // Force reflow to ensure transition works
      tooltip.offsetHeight;
      tooltip.style.transition = 'opacity 0.2s ease-in';
      tooltip.style.opacity = '1';
    }, 500); // 500ms delay
  });

  cy.on('mousemove', 'edge', (evt) => {
    tooltip.style.left = `${evt.originalEvent.pageX + 10}px`;
    tooltip.style.top = `${evt.originalEvent.pageY + 10}px`;
  });

  cy.on('mouseout', 'edge', (_evt) => {
    // Clear timeout and hide tooltip with fade
    if (tooltipTimeout) {
      clearTimeout(tooltipTimeout);
      tooltipTimeout = null;
    }
    clearInfoPopup(true); // Fade out
  });

  // Tooltip for nodes with delay
  cy.on('mouseover', 'node', (evt) => {
    // Clear any existing timeout
    if (tooltipTimeout) {
      clearTimeout(tooltipTimeout);
    }

    // Set timeout for 500ms delay
    tooltipTimeout = setTimeout(() => {
      const node = evt.target;
      const nodeType = node.data('type');
      const nodeLabel = node.data('label');
      const overlappingTargets = node.data('overlappingTargets');
      const overlappingWith = node.data('overlappingWith');

      let tooltipText = '';

      // Special handling for overlapping shared library warning
      if (overlappingTargets && overlappingTargets.length > 0) {
        const targetList = overlappingTargets.join('\n  ');
        tooltipText = `⚠️ DUPLICATE SYMBOLS!\n\nBoth this binary and ${nodeLabel} statically link:\n  ${targetList}\n\nThis can cause symbol conflicts at runtime!`;
      }
      // Special handling for overlapping library targets
      else if (overlappingWith && overlappingWith.length > 0) {
        const sharedLibList = overlappingWith.join('\n  ');
        tooltipText = `⚠️ DUPLICATE SYMBOLS!\n\n${nodeLabel} is statically linked by both:\n  • This binary\n  • Shared libraries:\n    ${sharedLibList}\n\nThis can cause symbol conflicts at runtime!`;
      }
      // Show type information for regular nodes
      else {
        if (nodeType === 'cc_binary') {
          tooltipText =
            '📦 Binary (cc_binary)\nExecutable program.\nLinks dependencies into final executable.';
        } else if (nodeType === 'cc_library') {
          tooltipText =
            '📚 Library (cc_library)\nStatic library.\nCompiled code reused by other targets.';
        } else if (nodeType === 'cc_shared_library') {
          tooltipText =
            '🔗 Shared Library (cc_shared_library)\nDynamic library (.so/.dylib).\nLoaded at runtime, shared between processes.';
        } else if (nodeType === 'system_library') {
          tooltipText =
            '⚙️ System Library\nExternal library from the system.\nProvided by OS or installed separately.';
        } else if (nodeType === 'target-group') {
          tooltipText =
            '📁 Target Container\nGroups files within a target.\nClick to focus on this target.';
        } else if (nodeType?.startsWith('source')) {
          tooltipText =
            '📄 Source File (.cc/.cpp)\nImplementation file.\nCompiled into object code.';
        } else if (nodeType?.startsWith('header')) {
          tooltipText =
            '📋 Header File (.h/.hpp)\nInterface definitions.\nIncluded by source files.';
        } else if (nodeType) {
          tooltipText = `Type: ${nodeType}\n${nodeLabel}`;
        } else {
          tooltipText = nodeLabel;
        }
      }

      tooltip.textContent = tooltipText;
      // Fade in animation
      tooltip.style.display = 'block';
      tooltip.style.opacity = '0';
      // Force reflow to ensure transition works
      tooltip.offsetHeight;
      tooltip.style.transition = 'opacity 0.2s ease-in';
      tooltip.style.opacity = '1';
    }, 500); // 500ms delay
  });

  cy.on('mousemove', 'node', (evt) => {
    tooltip.style.left = `${evt.originalEvent.pageX + 10}px`;
    tooltip.style.top = `${evt.originalEvent.pageY + 10}px`;
  });

  cy.on('mouseout', 'node', (_evt) => {
    // Clear timeout and hide tooltip with fade
    if (tooltipTimeout) {
      clearTimeout(tooltipTimeout);
      tooltipTimeout = null;
    }
    clearInfoPopup(true); // Fade out
  });

  // Hide tooltip when clicking anywhere (fixes tooltip staying visible after click)
  cy.on('tap', (_evt) => {
    // Clear timeout and hide tooltip
    if (tooltipTimeout) {
      clearTimeout(tooltipTimeout);
      tooltipTimeout = null;
    }
    clearInfoPopup(); // Immediate hide on click
  });

  // Click on graph background to clear selection
  cy.on('tap', (evt) => {
    // Check if we clicked on the background (not a node or edge)
    if (evt.target === cy) {
      appLogger.info('Background clicked - clearing selection');
      viewStateManager.clearSelection();
    }
  });

  // Click on nodes to select them
  // Simple click: select single node (clear other selections)
  // Ctrl+Click: toggle node in selection
  cy.on('tap', 'node', (evt) => {
    const node = evt.target;
    let nodeId = node.data('id');
    const nodeType = node.data('type');

    // If clicking a file node, select its parent target instead
    // Files don't have dependencies - their parent targets do
    const isFileNode =
      nodeType === 'source_file' ||
      nodeType === 'header_file' ||
      nodeType === 'uncovered_source' ||
      nodeType === 'uncovered_header';

    if (isFileNode) {
      const parentId = node.data('parent');
      if (parentId) {
        appLogger.info('File node clicked - redirecting to parent target:', {
          file: nodeId,
          parent: parentId,
        });
        nodeId = parentId;
      } else {
        appLogger.warn('File node has no parent - selecting anyway:', nodeId);
      }
    }

    if (evt.originalEvent.ctrlKey || evt.originalEvent.metaKey) {
      // Ctrl/Cmd+Click: Toggle selection
      appLogger.info('Node ctrl+clicked (toggle selection):', nodeId);
      viewStateManager.toggleSelection(nodeId);
    } else {
      // Simple click: Replace selection with this node only
      appLogger.info('Node clicked (select):', nodeId);
      viewStateManager.setSelection([nodeId]);
    }
  });

  // Set explicit dimensions based on flex container size
  updateCytoscapeSize();

  // Center and fit the graph after layout completes and canvas is ready
  cy.one('layoutstop', () => {
    // Small delay to ensure canvas has final dimensions
    setTimeout(() => {
      // Notify cytoscape that container dimensions are finalized
      cy.resize();

      // Center on all elements (the entire graph)
      cy.center(cy.elements());
      // Then fit to viewport with padding
      cy.fit(cy.elements(), 50);
    }, 10);
  });
}

// Update Cytoscape canvas size based on actual container dimensions
function updateCytoscapeSize() {
  const container = document.getElementById('cy');
  if (!container) return;

  // Clear any explicit dimensions to allow flex to work
  container.style.width = '';
  container.style.height = '';

  // Force a reflow by reading offsetHeight
  void container.offsetHeight;

  // Get the actual computed size of the flex container
  const rect = container.getBoundingClientRect();

  // Set explicit pixel dimensions based on current size
  container.style.width = `${rect.width}px`;
  container.style.height = `${rect.height}px`;

  // Tell Cytoscape to update its canvas size
  if (cy) {
    cy.resize();
  }
}

// SSE subscriptions
let workspaceStatusSource = null;
let targetGraphSource = null;

// State tracking for UI updates
let graphDataLoaded = false;
let analysisComplete = false;

// Connection monitoring
let connectionLost = false;
let lastSuccessfulRequest = Date.now();

// Handle connection lost
function handleConnectionLost(source) {
  if (connectionLost) {
    return; // Already handling connection loss
  }

  connectionLost = true;
  appLogger.error('Connection lost to backend server (source:', source, ')');

  // Show connection lost modal
  showConnectionLostModal();
}

// Show connection lost modal
function showConnectionLostModal() {
  const modal = document.getElementById('connectionLostModal');
  const messageEl = document.getElementById('connectionErrorMessage');

  messageEl.textContent = 'The backend server is not running. Restart the server to continue.';
  modal.style.display = 'flex';
}

// Wrapper for fetch that detects connection failures
function monitoredFetch(url, options) {
  return fetch(url, options)
    .then((response) => {
      if (response.ok) {
        lastSuccessfulRequest = Date.now();
      }
      return response;
    })
    .catch((error) => {
      appLogger.error('Fetch failed:', url, error);
      // Network error - backend likely down
      handleConnectionLost('fetch');
      throw error;
    });
}

// Check connection on user activity
function checkConnectionOnActivity() {
  if (!analysisComplete || connectionLost) {
    return;
  }

  // Check if SSE connections are closed
  if (workspaceStatusSource && workspaceStatusSource.readyState === 2) {
    handleConnectionLost('activity_check');
    return;
  }

  if (targetGraphSource && targetGraphSource.readyState === 2) {
    handleConnectionLost('activity_check');
    return;
  }

  // If it's been a while since last successful request, do a quick health check
  const timeSinceLastSuccess = Date.now() - lastSuccessfulRequest;
  if (timeSinceLastSuccess > 3000) {
    fetch('/api/module', { method: 'HEAD' })
      .then((response) => {
        if (response.ok) {
          lastSuccessfulRequest = Date.now();
        } else {
          handleConnectionLost('activity_check');
        }
      })
      .catch(() => {
        handleConnectionLost('activity_check');
      });
  }
}

// Set up activity listeners - will be called after page loads
let activityCheckTimeout = null;
function setupActivityListeners() {
  document.addEventListener('visibilitychange', () => {
    if (!document.hidden) {
      checkConnectionOnActivity();
    }
  });

  function scheduleActivityCheck() {
    if (!activityCheckTimeout) {
      activityCheckTimeout = setTimeout(() => {
        checkConnectionOnActivity();
        activityCheckTimeout = null;
      }, 1000);
    }
  }

  document.addEventListener('mousemove', scheduleActivityCheck);
  document.addEventListener('keydown', scheduleActivityCheck);
  document.addEventListener('click', scheduleActivityCheck);

  appLogger.info('Activity-based connection monitoring enabled');
}

// Subscribe to workspace status events
function subscribeToWorkspaceStatus() {
  appLogger.debug('Creating EventSource for workspace_status...');
  workspaceStatusSource = new EventSource('/api/subscribe/workspace_status');

  workspaceStatusSource.onopen = () => {
    appLogger.debug(
      'workspace_status EventSource connected, readyState:',
      workspaceStatusSource.readyState
    );
  };

  workspaceStatusSource.onmessage = (event) => {
    appLogger.debug('Raw workspace_status event data:', event.data);
    try {
      const sseEvent = JSON.parse(event.data);
      appLogger.debug('Parsed SSE event:', sseEvent);

      // sseEvent.data is json.RawMessage (already a JSON string), parse it
      let status;
      if (typeof sseEvent.data === 'string') {
        status = JSON.parse(sseEvent.data);
      } else {
        status = sseEvent.data; // Already an object (shouldn't happen with json.RawMessage)
      }
      appLogger.debug('Parsed status:', status);

      appLogger.info('Workspace status:', status.state, '-', status.message);

      // Update watching indicator
      updateWatchingIndicator(status.watching);

      // Show re-analysis notification
      if (status.reason && status.reason !== 'initial analysis') {
        showNotification(`Re-analyzing: ${status.reason}`);
      }

      // Update loading progress based on state
      if (status.state === 'bazel_querying') {
        updateLoadingProgress(null, 1);
        document.getElementById('graphSection').style.display = 'flex';
      } else if (status.state === 'analyzing_deps') {
        updateLoadingProgress(1, 2);
      } else if (status.state === 'analyzing_symbols') {
        updateLoadingProgress(2, 3);
      } else if (status.state === 'discovering_files') {
        updateLoadingProgress(3, 4);
      } else if (status.state === 'targets_ready') {
        updateLoadingProgress(4, 5);
      } else if (status.state === 'analyzing_binaries') {
        updateLoadingProgress(5, 5); // Keep step 5 active during binary analysis
      } else if (status.state === 'ready' || status.state === 'watching') {
        updateLoadingProgress(5, null); // Mark step 5 complete
        analysisComplete = true;

        hideLoadingOverlay();

        // Load/reload graph data
        // For initial analysis or re-analysis, always reload to get latest data
        loadGraphData();
        graphDataLoaded = true;

        // Update the "last updated" timestamp in watching indicator
        if (status.watching) {
          markAnalysisUpdate();
        }

        // Don't close SSE connections if we're in watch mode
        // Keep them open to receive re-analysis notifications
        if (!status.watching) {
          // Close SSE connections when done (non-watch mode)
          if (workspaceStatusSource) {
            workspaceStatusSource.close();
            workspaceStatusSource = null;
          }
          if (targetGraphSource) {
            targetGraphSource.close();
            targetGraphSource = null;
          }
        }
      }
    } catch (e) {
      appLogger.error('Error processing workspace status:', e, 'Raw data:', event.data);
    }
  };

  workspaceStatusSource.onerror = (error) => {
    appLogger.error(
      'Workspace status SSE error:',
      error,
      'readyState:',
      workspaceStatusSource.readyState
    );

    // EventSource readyState: 0 = CONNECTING, 1 = OPEN, 2 = CLOSED
    if (workspaceStatusSource.readyState === 2) {
      handleConnectionLost('workspace_status');
    }
  };
}

// Subscribe to target graph events
function subscribeToTargetGraph() {
  targetGraphSource = new EventSource('/api/subscribe/target_graph');

  targetGraphSource.onmessage = (event) => {
    try {
      const sseEvent = JSON.parse(event.data);

      // sseEvent.data is json.RawMessage (already a JSON string), parse it
      let graphData;
      if (typeof sseEvent.data === 'string') {
        graphData = JSON.parse(sseEvent.data);
      } else {
        graphData = sseEvent.data;
      }

      appLogger.debug('Target graph update:', sseEvent.type, 'complete:', graphData.complete);

      // Load full graph data when available
      if (sseEvent.type === 'complete' && !graphDataLoaded) {
        loadGraphData();
        graphDataLoaded = true;
      }
    } catch (e) {
      appLogger.error('Error processing target graph event:', e);
    }
  };

  targetGraphSource.onerror = (error) => {
    appLogger.error('Target graph SSE error:', error);

    // EventSource readyState: 0 = CONNECTING, 1 = OPEN, 2 = CLOSED
    if (targetGraphSource.readyState === 2) {
      handleConnectionLost('target_graph');
    }
  };
}

// Enrich graph nodes with overlapping dependency information from binaries
function enrichGraphWithOverlappingInfo(graph, binaries) {
  // Collect all overlapping targets across all binaries
  const allOverlappingTargets = new Map(); // target -> Set of shared libraries causing overlap

  binaries.forEach((binary) => {
    if (binary.overlappingDeps) {
      Object.entries(binary.overlappingDeps).forEach(([sharedLib, targets]) => {
        targets.forEach((target) => {
          if (!allOverlappingTargets.has(target)) {
            allOverlappingTargets.set(target, new Set());
          }
          allOverlappingTargets.get(target).add(sharedLib);
        });
      });
    }
  });

  appLogger.debug('Overlapping targets found:', Array.from(allOverlappingTargets.keys()));

  // Mark nodes that have overlapping dependencies
  // Use node.id for matching because node.label may have distance annotations like "(d=1)"
  graph.nodes.forEach((node) => {
    if (allOverlappingTargets.has(node.id)) {
      node.hasOverlap = true;
      node.overlappingWith = Array.from(allOverlappingTargets.get(node.id));
      appLogger.debug('Marked node as overlapping:', node.id);
    }
  });

  // Mark shared library nodes that have overlapping deps
  // Use node.id for matching because node.label may be simplified
  binaries.forEach((binary) => {
    if (binary.kind === 'cc_shared_library' && binary.overlappingDeps) {
      const overlappingCount = Object.keys(binary.overlappingDeps).length;
      if (overlappingCount > 0) {
        const node = graph.nodes.find((n) => n.id === binary.label);
        if (node) {
          node.hasOverlap = true;
          node.overlappingTargets = Object.values(binary.overlappingDeps).flat();
          appLogger.debug('Marked shared library as overlapping:', binary.label);
        }
      }
    }
  });
}

// Load full graph data from API
async function loadGraphData() {
  appLogger.debug('loadGraphData() called');
  try {
    // Fetch module graph
    const graphResponse = await monitoredFetch('/api/module/graph');
    appLogger.info('Module graph response status:', graphResponse.status);
    if (graphResponse.ok) {
      packageGraph = await graphResponse.json();
      appLogger.info('Loaded graph with', packageGraph.nodes?.length, 'nodes');

      // Render through backend lens API to ensure proper hierarchy and collapse states
      if (packageGraph?.nodes && packageGraph.nodes.length > 0) {
        appLogger.info('Graph loaded, rendering through backend lens API');
        try {
          const currentState = viewStateManager.getState();
          const renderedGraph = await fetchRenderedGraphFromBackend(currentState);
          // Note: binaryData not loaded yet at this point, will be enriched in second render
          displayDependencyGraph(renderedGraph);
        } catch (error) {
          appLogger.error('Error rendering graph via backend:', error);
        }
      } else {
        appLogger.warn('Package graph has no nodes or is invalid:', packageGraph);
      }
    } else {
      appLogger.error(
        'Failed to fetch module graph:',
        graphResponse.status,
        graphResponse.statusText
      );
    }

    // Fetch module data for tree browser
    const moduleResponse = await monitoredFetch('/api/module');
    if (moduleResponse.ok) {
      const moduleData = await moduleResponse.json();
      analysisData = {
        graph: packageGraph,
        module: moduleData,
      };

      // Update subtitle with module name and workspace path
      updateModuleName(moduleData.name, moduleData.workspacePath);

      // Populate tree browser
      if (packageGraph?.nodes) {
        populateTreeBrowser(analysisData);
      }

      // Fetch binaries data for overlapping dependency detection
      try {
        const binariesResponse = await monitoredFetch('/api/binaries');
        if (binariesResponse.ok) {
          binaryData = await binariesResponse.json();
          appLogger.info('Loaded binary data:', binaryData.length, 'binaries');
        }
      } catch (error) {
        appLogger.warn('Failed to load binaries data:', error);
      }

      // Initialize lens controls (after DOM is ready and data is loaded)
      initializeLensControls();

      // Trigger initial render with backend lens API
      try {
        const renderedGraph = await fetchRenderedGraphFromBackend(viewStateManager.getState());
        // Enrich with overlapping dependency information if we have binary data
        if (binaryData && packageGraph) {
          enrichGraphWithOverlappingInfo(renderedGraph, binaryData);
        }
        displayDependencyGraph(renderedGraph);
      } catch (error) {
        appLogger.error('Error rendering graph via backend:', error);
      }
    }
  } catch (e) {
    appLogger.error('Error loading graph data:', e);
  }
}

// Clean up connections when page is being unloaded
window.addEventListener('beforeunload', () => {
  appLogger.info('Page unloading, closing SSE connections');
  if (workspaceStatusSource) {
    workspaceStatusSource.close();
    workspaceStatusSource = null;
  }
  if (targetGraphSource) {
    targetGraphSource.close();
    targetGraphSource = null;
  }
  if (cy) {
    cy.destroy();
    cy = null;
  }
});

// ===== Navigation Filter Setup =====
// Timeout for debouncing search input
let searchTimeout = null;

/**
 * Set up event handlers for navigation filter controls
 */
function setupNavigationFilters() {
  appLogger.debug('[App] Setting up navigation filters');

  // Dropdown toggle
  const dropdownBtn = document.getElementById('ruleTypeDropdown');
  const dropdownMenu = document.getElementById('ruleTypeMenu');

  if (!dropdownBtn || !dropdownMenu) {
    appLogger.warn('[App] Filter controls not found in DOM');
    return;
  }

  dropdownBtn.onclick = (e) => {
    const isVisible = dropdownMenu.style.display !== 'none';
    dropdownMenu.style.display = isVisible ? 'none' : 'block';
    e.stopPropagation();
    appLogger.trace('[App] Toggled filter dropdown', { isVisible: !isVisible });
  };

  // Close dropdown when clicking outside
  document.addEventListener('click', () => {
    if (dropdownMenu.style.display !== 'none') {
      dropdownMenu.style.display = 'none';
      appLogger.trace('[App] Closed filter dropdown (click outside)');
    }
  });

  // Checkbox changes trigger filter update
  document.querySelectorAll('#ruleTypeMenu input[type="checkbox"]').forEach((checkbox) => {
    checkbox.onchange = () => {
      const selectedTypes = new Set(
        Array.from(document.querySelectorAll('#ruleTypeMenu input:checked')).map((cb) => cb.value)
      );
      const searchText = document.getElementById('targetSearch').value;

      appLogger.debug('[App] Rule type filter changed', {
        selectedTypes: Array.from(selectedTypes),
        searchText,
      });

      viewStateManager.updateNavigationFilters(selectedTypes, searchText);
    };
  });

  // Search input with debounce
  const searchInput = document.getElementById('targetSearch');
  if (searchInput) {
    searchInput.oninput = () => {
      clearTimeout(searchTimeout);
      const searchText = searchInput.value;

      searchTimeout = setTimeout(() => {
        const selectedTypes = new Set(
          Array.from(document.querySelectorAll('#ruleTypeMenu input:checked')).map((cb) => cb.value)
        );

        appLogger.debug('[App] Search filter changed', {
          selectedTypes: Array.from(selectedTypes),
          searchText,
        });

        viewStateManager.updateNavigationFilters(selectedTypes, searchText);
      }, 300); // 300ms debounce
    };
  }

  appLogger.info('[App] Navigation filters initialized');
}

// Initialize subscriptions when page loads
document.addEventListener('DOMContentLoaded', () => {
  appLogger.info('Starting SSE subscriptions...');

  // Set up navigation filter event handlers
  setupNavigationFilters();

  // Close any existing connections first (in case of reload)
  if (workspaceStatusSource) {
    appLogger.info('Closing existing workspace_status connection');
    workspaceStatusSource.close();
    workspaceStatusSource = null;
  }
  if (targetGraphSource) {
    appLogger.info('Closing existing target_graph connection');
    targetGraphSource.close();
    targetGraphSource = null;
  }

  // Destroy any existing Cytoscape instance
  if (cy) {
    appLogger.info('Destroying existing Cytoscape instance');
    cy.destroy();
    cy = null;
  }

  // Reset state flags
  graphDataLoaded = false;
  analysisComplete = false;

  // Set up activity-based connection monitoring
  setupActivityListeners();

  // Subscribe to both event streams
  subscribeToWorkspaceStatus();
  subscribeToTargetGraph();
});

// Close modal handlers
document.addEventListener('DOMContentLoaded', () => {
  const modal = document.getElementById('targetModal');
  const closeBtn = document.querySelector('.modal-close');

  // Close when clicking X
  closeBtn.onclick = () => {
    modal.style.display = 'none';
  };

  // Close when clicking outside modal
  window.onclick = (event) => {
    if (event.target === modal) {
      modal.style.display = 'none';
    }
  };
});

// ============================================================================
// Tree Browser Implementation
// ============================================================================

let analysisData = null; // Store full analysis data
let packageGraph = null; // Store the original package-level graph
let binaryData = null; // Store binary analysis data for overlapping dependency detection
let cy = null; // Store the Cytoscape instance
let allTargetNodes = []; // Store all target nodes for client-side filtering

// Global configuration object
const GRAPH_CONFIG = {
  animation: {
    duration: 500, // Animation duration in milliseconds
    easing: 'ease-out', // Easing function
  },
  layout: {
    nodeSep: 80,
    edgeSep: 20,
    rankSep: 120,
    padding: 50,
  },
};

// ===== Lens-Based Visualization System =====
// Replaced currentView, currentTarget, currentBinary with lens system
// All lens rendering now happens server-side via /api/module/graph/lens

// Current graph state hash (for diff-based updates)
let currentGraphHash = null;

// Current graph data (for applying diffs)
let currentGraphData = null;

// Track pending request for cancellation
let pendingRequestController = null;

/**
 * Run Dagre layout with stable, deterministic ordering
 * Uses centralized configuration for consistent layout parameters
 * Note: Edge ordering is determined by the backend (renderer.go sorts edges)
 * and preserved when adding to Cytoscape
 * @param {boolean} animate - Whether to animate the layout
 * @param {boolean} fit - Whether to fit the viewport to the graph
 */
function runStableDagreLayout(animate = true, fit = false) {
  if (!cy) return;

  // Run layout with configuration
  cy.layout({
    name: 'dagre',
    rankDir: 'TB', // Top to bottom - arrows go down
    ranker: 'network-simplex', // Most deterministic ranker
    nodeSep: GRAPH_CONFIG.layout.nodeSep,
    edgeSep: GRAPH_CONFIG.layout.edgeSep,
    rankSep: GRAPH_CONFIG.layout.rankSep,
    padding: GRAPH_CONFIG.layout.padding,
    fit: fit,
    animate: animate,
    animationDuration: GRAPH_CONFIG.animation.duration,
    animationEasing: GRAPH_CONFIG.animation.easing,
  }).run();
}

/**
 * Fetch rendered graph from backend lens API
 * @param {Object} viewState - Current view state with lens configurations
 * @returns {Promise<Object>} Rendered graph from backend
 */
async function fetchRenderedGraphFromBackend(viewState) {
  // Cancel any pending request
  if (pendingRequestController) {
    appLogger.info('[App] Cancelling previous request');
    pendingRequestController.abort();
  }

  // Create new AbortController for this request
  pendingRequestController = new AbortController();
  const signal = pendingRequestController.signal;

  appLogger.info('[App] Fetching rendered graph from backend lens API');

  // Convert edgeRules.types from Set to Array for JSON serialization
  const serializeLens = (lens) => ({
    ...lens,
    edgeRules: {
      ...lens.edgeRules,
      types: Array.from(lens.edgeRules.types),
    },
  });

  const requestBody = {
    defaultLens: serializeLens(viewState.defaultLens),
    detailLens: serializeLens(viewState.detailLens),
    selectedNodes: Array.from(viewState.selectedNodes),
    previousHash: currentGraphHash, // Send previous hash for diff-based updates
  };

  appLogger.debug('[App] Request body:', JSON.stringify(requestBody, null, 2));

  const response = await fetch('/api/module/graph/lens', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(requestBody),
    signal: signal, // Attach abort signal
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`Backend lens rendering failed: ${response.statusText} - ${errorText}`);
  }

  const responseData = await response.json();

  // Update current hash
  currentGraphHash = responseData.hash;

  // Handle diff vs full graph response
  let renderedGraph;
  if (responseData.fullGraph) {
    appLogger.info(
      '[App] Received full graph from backend:',
      responseData.fullGraph.nodes?.length,
      'nodes,',
      responseData.fullGraph.edges?.length,
      'edges'
    );
    renderedGraph = responseData.fullGraph;
    currentGraphData = renderedGraph;
  } else if (responseData.diff) {
    appLogger.info(
      '[App] Received diff from backend:',
      responseData.diff.addedNodes?.length || 0,
      'added nodes,',
      responseData.diff.removedNodes?.length || 0,
      'removed nodes,',
      responseData.diff.addedEdges?.length || 0,
      'added edges,',
      responseData.diff.removedEdges?.length || 0,
      'removed edges'
    );
    renderedGraph = applyGraphDiff(currentGraphData, responseData.diff);
    currentGraphData = renderedGraph;
  } else {
    throw new Error('Invalid response: neither fullGraph nor diff provided');
  }

  return renderedGraph;
}

/**
 * Apply a graph diff to the current graph data
 * @param {Object} currentGraph - Current graph data
 * @param {Object} diff - Diff to apply
 * @returns {Object} Updated graph data
 */
function applyGraphDiff(currentGraph, diff) {
  if (!currentGraph) {
    throw new Error('Cannot apply diff: no current graph');
  }

  // Create maps for efficient lookup
  const nodeMap = new Map(currentGraph.nodes.map((n) => [n.id, n]));
  const edgeMap = new Map(currentGraph.edges.map((e) => [`${e.source}|${e.target}|${e.type}`, e]));

  // Apply node changes
  if (diff.removedNodes) {
    diff.removedNodes.forEach((nodeId) => nodeMap.delete(nodeId));
  }

  if (diff.addedNodes) {
    diff.addedNodes.forEach((node) => nodeMap.set(node.id, node));
  }

  if (diff.modifiedNodes) {
    diff.modifiedNodes.forEach((node) => nodeMap.set(node.id, node));
  }

  // Apply edge changes
  if (diff.removedEdges) {
    diff.removedEdges.forEach((edgeKey) => edgeMap.delete(edgeKey));
  }

  if (diff.addedEdges) {
    diff.addedEdges.forEach((edge) => {
      const key = `${edge.source}|${edge.target}|${edge.type}`;
      edgeMap.set(key, edge);
    });
  }

  // Convert maps back to arrays
  return {
    nodes: Array.from(nodeMap.values()),
    edges: Array.from(edgeMap.values()),
  };
}

// Update navigation item highlighting based on selection
function updateNavigationHighlighting(selectedNodes) {
  const navItems = document.querySelectorAll('.nav-item');
  navItems.forEach((item) => {
    const nodeId = item.dataset.nodeId;
    if (!nodeId) return;

    // Check if this item's node ID is directly in the selection
    let isSelected = selectedNodes.has(nodeId);

    // If not directly selected, check if its parent package is selected
    // Navigation shows targets (//package:target), selection might contain packages (//package)
    if (!isSelected) {
      // Extract package from target ID (//package:target -> //package)
      const colonIndex = nodeId.indexOf(':');
      if (colonIndex !== -1) {
        const packageId = nodeId.substring(0, colonIndex);
        isSelected = selectedNodes.has(packageId);
      }
    }

    if (isSelected) {
      item.classList.add('selected');
    } else {
      item.classList.remove('selected');
    }
  });
}

// Listen for state changes and re-render graph
viewStateManager.addListener(async (newState) => {
  if (!packageGraph) return; // Wait for initial load

  appLogger.debug('[App] State changed, rendering with backend API');
  appLogger.debug('[App] BaseSet:', newState.defaultLens.baseSet);
  appLogger.debug('[App] Selected nodes:', Array.from(newState.selectedNodes));

  // Update navigation highlighting
  updateNavigationHighlighting(newState.selectedNodes);

  // Deep clone the state to avoid reference issues
  previousViewState = {
    ...newState,
    selectedNodes: new Set(newState.selectedNodes),
    defaultLens: JSON.parse(JSON.stringify(newState.defaultLens)),
    detailLens: JSON.parse(JSON.stringify(newState.detailLens)),
  };

  try {
    // Fetch rendered graph from backend
    const renderedGraph = await fetchRenderedGraphFromBackend(newState);

    // Enrich with overlapping dependency information if we have binary data
    if (binaryData) {
      enrichGraphWithOverlappingInfo(renderedGraph, binaryData);
    }

    // Display the pre-rendered graph from backend
    displayDependencyGraph(renderedGraph);
  } catch (error) {
    // Ignore AbortError - this happens when a new request cancels the previous one
    if (error.name === 'AbortError') {
      appLogger.info('[App] Request cancelled (new request started)');
      return;
    }

    appLogger.error('[App] Error fetching rendered graph from backend:', error);
    appLogger.error('[App] Backend lens rendering failed - this is a fatal error');
  }
});

// NOTE: Old tree-building functions removed
// buildTreeData(), createTreeNode(), toggleExpansion() are no longer needed
// The navigation now uses simple flat lists populated by populateTreeBrowser()

// NOTE: Old view-switching functions removed - now handled by lens system
// The following functions have been replaced by the lens-based visualization:
// - selectTreeNode() -> use viewStateManager.setSelection()
// - showBinaryGraphFocused() -> handled by lens renderer
// - zoomOutOneLevel() -> use viewStateManager.clearSelection()
// - showFocusedTargetView() -> handled by lens renderer
// - showFileGraphForTarget() -> handled by lens renderer
// - selectBinary() -> use viewStateManager.setSelection()
// - selectTarget() -> use viewStateManager.setSelection()
// - showBinaryFocusedGraph() -> handled by lens renderer
// - buildBinaryFocusedGraphData() -> handled by lens renderer

// Populate the tree browser (stores all targets and renders with current filters)
function populateTreeBrowser(data) {
  appLogger.debug('Populating navigation with data:', {
    nodeCount: data.graph?.nodes?.length,
    edgeCount: data.graph?.edges?.length,
  });

  if (!data.graph || !data.graph.nodes) return;

  // Store all target nodes for client-side filtering
  allTargetNodes = data.graph.nodes.filter((node) => {
    const allowedTypes = ['cc_binary', 'cc_library', 'cc_shared_library'];
    return allowedTypes.includes(node.type);
  });

  appLogger.debug('Stored target nodes:', { count: allTargetNodes.length });

  // Render the filtered list
  filterAndRenderNavigationList();
}

// Filter and render navigation list based on current filter state (exposed globally for view-state.js)
window.filterAndRenderNavigationList = function filterAndRenderNavigationList() {
  const targetsItems = document.getElementById('targetsItems');
  if (!targetsItems) return;

  targetsItems.innerHTML = '';

  // Get current filters from state
  const filters = viewStateManager.state.navigationFilters;
  const ruleTypes = filters.ruleTypes || new Set(['cc_binary', 'cc_library', 'cc_shared_library']);
  const searchText = (filters.searchText || '').toLowerCase();

  // Apply client-side filtering
  const filteredNodes = allTargetNodes.filter((node) => {
    // Filter by rule type
    if (!ruleTypes.has(node.type)) {
      return false;
    }

    // Filter by search text
    if (searchText && !node.label.toLowerCase().includes(searchText)) {
      return false;
    }

    return true;
  });

  appLogger.debug('Filtered navigation list:', {
    total: allTargetNodes.length,
    filtered: filteredNodes.length,
    ruleTypes: Array.from(ruleTypes),
    searchText,
  });

  if (filteredNodes.length === 0) {
    const msg = document.createElement('div');
    msg.className = 'nav-empty-message';
    msg.textContent = 'No targets match filters';
    targetsItems.appendChild(msg);
    return;
  }

  // Sort alphabetically by label
  filteredNodes.sort((a, b) => a.label.localeCompare(b.label));

  filteredNodes.forEach((node) => {
    const item = document.createElement('div');
    item.className = 'nav-item';
    item.dataset.nodeId = node.label; // Store full node ID for selection matching

    item.textContent = simplifyLabel(node.label);

    // Click selects this target (Cmd/Ctrl+click for multi-select)
    item.onclick = (e) => {
      if (e.ctrlKey || e.metaKey) {
        // Multi-select: toggle this item
        viewStateManager.toggleSelection(node.label);
      } else {
        // Single select: replace selection
        viewStateManager.setSelection([node.label]);
      }
    };

    targetsItems.appendChild(item);
  });

  // Update highlighting to match current selection
  updateNavigationHighlighting(viewStateManager.state.selectedNodes);
};

// Handle window resize to update Cytoscape canvas size
let resizeTimeout;
window.addEventListener('resize', () => {
  // Debounce resize events to avoid excessive updates
  clearTimeout(resizeTimeout);
  resizeTimeout = setTimeout(() => {
    if (cy) {
      updateCytoscapeSize();
      cy.fit(undefined, 50); // Refit with padding after resize
    }
  }, 150);
});

// Handle horizontal resize of sidebar
(() => {
  const resizeHandle = document.getElementById('resizeHandle');
  const treeBrowser = document.getElementById('treeBrowser');
  let isResizing = false;
  let startX = 0;
  let startWidth = 0;

  if (!resizeHandle || !treeBrowser) return;

  resizeHandle.addEventListener('mousedown', (e) => {
    isResizing = true;
    startX = e.clientX;
    startWidth = treeBrowser.offsetWidth;
    resizeHandle.classList.add('active');

    // Prevent text selection during drag
    e.preventDefault();
    document.body.style.userSelect = 'none';
    document.body.style.cursor = 'ew-resize';
  });

  document.addEventListener('mousemove', (e) => {
    if (!isResizing) return;

    const delta = e.clientX - startX;
    const newWidth = startWidth + delta;

    // Respect min and max width constraints
    const minWidth = 200;
    const maxWidth = 600;
    const constrainedWidth = Math.max(minWidth, Math.min(maxWidth, newWidth));

    treeBrowser.style.width = `${constrainedWidth}px`;

    // Update Cytoscape canvas size if it exists
    if (cy) {
      updateCytoscapeSize();
    }
  });

  document.addEventListener('mouseup', () => {
    if (isResizing) {
      isResizing = false;
      resizeHandle.classList.remove('active');
      document.body.style.userSelect = '';
      document.body.style.cursor = '';

      // Final update to Cytoscape
      if (cy) {
        updateCytoscapeSize();
        cy.fit(undefined, 50);
      }
    }
  });

  // Clear info popups when window loses focus
  window.addEventListener('blur', () => {
    clearInfoPopup(true); // Use fade animation
  });
})();
