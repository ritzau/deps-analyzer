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
    simplified = simplified.replace(/std::basic_string<char,\s*std::char_traits<char>,\s*std::allocator<char>\s*>/g, 'std::string');
    simplified = simplified.replace(/std::basic_string<char,\s*std::char_traits<char>>/g, 'std::string');

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

    const packagePath = match[1];  // e.g., "//foo" or "//bar/baz"
    const targetName = match[2];   // e.g., "foo" or "baz"

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

// Update the subtitle with the module/workspace name
function updateModuleName(name) {
    const subtitle = document.querySelector('.subtitle');
    if (subtitle && name) {
        subtitle.textContent = name;
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

function displayUncoveredFiles(files) {
    const listEl = document.getElementById('uncoveredList');
    listEl.innerHTML = '';

    files.forEach(file => {
        const fileDiv = document.createElement('div');
        fileDiv.className = 'file-item';

        const pathDiv = document.createElement('div');
        pathDiv.className = 'file-path';
        pathDiv.textContent = file.Path;

        const packageDiv = document.createElement('div');
        packageDiv.className = 'file-package';
        packageDiv.textContent = `Package: ${file.Package}`;

        const suggestionDiv = document.createElement('div');
        suggestionDiv.className = 'file-suggestion';
        suggestionDiv.textContent = '💡 Add to BUILD.bazel or remove if unused';

        fileDiv.appendChild(pathDiv);
        fileDiv.appendChild(packageDiv);
        fileDiv.appendChild(suggestionDiv);

        listEl.appendChild(fileDiv);
    });
}

// Position caching is no longer needed - we do incremental Cytoscape updates
// Nodes that already exist keep their positions automatically when we only add/remove changed elements

function displayDependencyGraph(graphData) {
    console.log('displayDependencyGraph called with', graphData.nodes?.length, 'nodes');

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
        ...graphData.nodes.map(node => {
            const nodeData = {
                id: node.id,
                label: simplifyLabel(node.label),
                type: node.type,
                parent: node.parent // For compound nodes (grouping)
            };

            // Mark focused nodes
            const focusedNodes = viewStateManager.getState().focusedNodes;
            if (focusedNodes.has(node.id) || focusedNodes.has(node.label)) {
                nodeData.focused = true;
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
        ...graphData.edges.map(edge => {
            const edgeData = {
                source: edge.source,
                target: edge.target,
                type: edge.type,
                linkage: edge.linkage,
                symbols: edge.symbols || [],
                sourceLabel: edge.sourceLabel,
                targetLabel: edge.targetLabel,
                fileDetails: edge.fileDetails || {}
            };
            // Only set isOverlapping if it's true (don't set it at all if false)
            if (edge.isOverlapping === true) {
                edgeData.isOverlapping = true;
            }
            return { data: edgeData };
        })
    ];

    // Debug: Log overlapping flags
    const overlappingNodes = elements.filter(e => e.data.hasOverlap === true);
    const overlappingEdges = elements.filter(e => e.data.isOverlapping === true);
    console.log('Nodes with hasOverlap=true:', overlappingNodes.map(n => n.data.label || n.data.id));
    console.log('Edges with isOverlapping=true:', overlappingEdges.map(e => `${e.data.source} -> ${e.data.target}`));

    // Cytoscape stylesheet (shared between initial and incremental updates)
    const cytoscapeStylesheet = [
            {
                selector: 'node',
                style: {
                    'shape': 'roundrectangle',
                    'background-color': '#4fc1ff',
                    'label': 'data(label)',
                    'color': '#1e1e1e',
                    'text-valign': 'center',
                    'text-halign': 'center',
                    'font-size': '12px',
                    'font-family': '-apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
                    'font-weight': '600',
                    'text-wrap': 'wrap',
                    'text-max-width': '140px',
                    'width': 'label',
                    'height': 'label',
                    'padding': '14px',
                    'border-width': '2px',
                    'border-color': '#3e3e42'
                }
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
                    'min-height': '100px'
                }
            },
            {
                selector: 'node[type = "cc_binary"]',
                style: {
                    'background-color': '#ff8c00',
                    'color': 'white',
                    'border-color': '#cc7000'
                }
            },
            {
                selector: 'node[type = "cc_shared_library"]',
                style: {
                    'background-color': '#c586c0',
                    'color': 'white',
                    'border-color': '#9d6b99'
                }
            },
            {
                selector: 'node[type = "system_library"]',
                style: {
                    'background-color': '#d7ba7d',
                    'color': '#1e1e1e',
                    'border-color': '#b89b5d',
                    'shape': 'hexagon'
                }
            },
            {
                selector: 'node[type = "source"], node[type ^= "source"]',
                style: {
                    'background-color': '#89d185',
                    'color': '#1e1e1e',
                    'border-color': '#6fb06b',
                    'shape': 'ellipse',
                    'width': '60px',
                    'height': '60px'
                }
            },
            {
                selector: 'node[type = "header"], node[type ^= "header"]',
                style: {
                    'background-color': '#4fc1ff',
                    'color': '#1e1e1e',
                    'border-color': '#3fa0d9',
                    'shape': 'ellipse',
                    'width': '60px',
                    'height': '60px'
                }
            },
            {
                selector: 'node[type = "uncovered_source"]',
                style: {
                    'background-color': '#ff6b6b',
                    'border-width': '3px',
                    'border-color': '#ff4444',
                    'border-style': 'double',
                    'opacity': 0.7,
                    'color': '#ffffff',
                    'shape': 'ellipse',
                    'width': '60px',
                    'height': '60px',
                    'font-size': '10px'
                }
            },
            {
                selector: 'node[type = "uncovered_header"]',
                style: {
                    'background-color': '#ff6b6b',
                    'border-width': '3px',
                    'border-color': '#ff4444',
                    'border-style': 'double',
                    'opacity': 0.7,
                    'color': '#ffffff',
                    'shape': 'ellipse',
                    'width': '60px',
                    'height': '60px',
                    'font-size': '10px'
                }
            },
            {
                selector: 'node[type = "external"]',
                style: {
                    'background-color': '#6a6a6a',
                    'color': '#cccccc',
                    'border-color': '#505050'
                }
            },
            {
                selector: 'node[type = "package"]',
                style: {
                    'background-color': '#4a4a4e',
                    'color': '#cccccc',
                    'border-color': '#696969',
                    'shape': 'roundrectangle',
                    'font-weight': 'bold',
                    'font-size': '14px',
                    'padding': '18px'
                }
            },
            {
                selector: 'node[type $= "_focused"]',
                style: {
                    'background-color': '#ff8c00',
                    'color': 'white',
                    'border-color': '#ff8c00',
                    'border-width': '4px',
                    'font-weight': 'bold'
                }
            },
            {
                selector: 'node[type $= "_incoming"]',
                style: {
                    'background-color': '#4ec9b0',
                    'color': '#1e1e1e',
                    'border-color': '#3da889'
                }
            },
            {
                selector: 'node[type $= "_outgoing"]',
                style: {
                    'background-color': '#c586c0',
                    'color': 'white',
                    'border-color': '#9d6b99'
                }
            },
            {
                selector: 'node[type = "target-group"]',
                style: {
                    'shape': 'roundrectangle',
                    'background-color': '#2d2d30',
                    'background-opacity': 0.3,
                    'border-width': '2px',
                    'border-color': '#4a4a4e',
                    'border-style': 'solid',
                    'label': 'data(label)',
                    'color': '#969696',
                    'text-valign': 'top',
                    'text-halign': 'center',
                    'font-size': '14px',
                    'font-weight': 'bold',
                    'padding': '30px'
                }
            },
            {
                selector: 'node[type = "target-group"][focused]',
                style: {
                    'border-width': '4px',
                    'border-color': '#ff8c00',
                    'background-opacity': 0.4,
                    'color': '#ff8c00'
                }
            },
            {
                selector: 'edge',
                style: {
                    'width': 2,
                    'line-color': '#6a6a6a',
                    'target-arrow-color': '#6a6a6a',
                    'target-arrow-shape': 'triangle',
                    'curve-style': 'bezier',
                    'arrow-scale': 1.5
                }
            },
            {
                selector: 'edge[type = "file"]',
                style: {
                    'line-color': '#6a6a6a',
                    'target-arrow-color': '#6a6a6a'
                }
            },
            {
                selector: 'edge[type = "symbol"]',
                style: {
                    'line-style': 'dashed',
                    'width': 1.5
                }
            },
            {
                selector: 'edge[type = "symbol"][linkage = "static"]',
                style: {
                    'line-color': '#4ec9b0',
                    'target-arrow-color': '#4ec9b0'
                }
            },
            {
                selector: 'edge[type = "symbol"][linkage = "dynamic"]',
                style: {
                    'line-color': '#c586c0',
                    'target-arrow-color': '#c586c0'
                }
            },
            {
                selector: 'edge[type = "symbol"][linkage = "cross"]',
                style: {
                    'line-color': '#d7ba7d',
                    'target-arrow-color': '#d7ba7d'
                }
            },
            {
                selector: 'edge[type = "dynamic_link"]',
                style: {
                    'line-color': '#c586c0',
                    'target-arrow-color': '#c586c0',
                    'width': 3,
                    'line-style': 'solid'
                }
            },
            {
                selector: 'edge[type = "data"]',
                style: {
                    'line-color': '#4ec9b0',
                    'target-arrow-color': '#4ec9b0',
                    'width': 2,
                    'line-style': 'dotted'
                }
            },
            {
                selector: 'edge[type = "system_link"]',
                style: {
                    'line-color': '#4ec9b0',
                    'target-arrow-color': '#4ec9b0',
                    'width': 2,
                    'line-style': 'dashed'
                }
            },
            {
                selector: 'edge[type = "compile"]',
                style: {
                    'line-color': '#4fc1ff',
                    'target-arrow-color': '#4fc1ff',
                    'width': 2,
                    'line-style': 'solid'
                }
            },
            {
                selector: 'edge[type = "static"]',
                style: {
                    'line-color': '#4ec9b0',
                    'target-arrow-color': '#4ec9b0',
                    'width': 2,
                    'line-style': 'solid'
                }
            },
            {
                selector: 'edge[type = "dynamic"]',
                style: {
                    'line-color': '#4ec9b0',
                    'target-arrow-color': '#4ec9b0',
                    'width': 2,
                    'line-style': 'dashed'
                }
            },
            // Overlapping dependencies - MUST be after type-specific selectors to override
            // Note: Only nodes/edges with hasOverlap/isOverlapping set to true will have this attribute
            {
                selector: 'node[hasOverlap][type = "cc_binary"], node[hasOverlap][type = "cc_shared_library"], node[hasOverlap][type = "cc_library"]',
                style: {
                    'border-width': '8px',
                    'border-color': '#ff4444',
                    'border-style': 'double'
                }
            },
            {
                selector: 'edge[isOverlapping]',
                style: {
                    'line-color': '#ff4444',
                    'target-arrow-color': '#ff4444',
                    'width': 4,
                    'line-style': 'solid'
                }
            },
            {
                selector: 'node:selected',
                style: {
                    'border-width': '3px',
                    'border-color': '#ff8c00'
                }
            },
            // Public visibility indicator (solid gold border)
            {
                selector: 'node[isPublic]',
                style: {
                    'border-style': 'solid',
                    'border-width': '3px',
                    'border-color': '#ffd700'
                }
            },
            // Focused node styling - MUST come after isPublic to override it
            {
                selector: 'node[focused]',
                style: {
                    'border-width': '4px',
                    'border-color': '#ff8c00',
                    'border-style': 'solid'
                }
            }
    ];

    if (isInitialLoad) {
        // Initial creation - create new cytoscape instance
        console.log('Creating new cytoscape instance with', elements.length, 'elements');

        cy = cytoscape({
            container: document.getElementById('cy'),
            elements: elements,
            style: cytoscapeStylesheet,
            layout: {
                name: 'dagre',
                rankDir: 'TB',
                ranker: 'network-simplex',
                nodeSep: 80,
                edgeSep: 20,
                rankSep: 120,
                animate: false,
                padding: 50
            }
        });

        // Setup event handlers (only on initial load)
        setupEventHandlers();
    } else {
        // Incremental update - update only changed elements
        console.log('Incrementally updating cytoscape with', elements.length, 'elements');

        cy.startBatch();

        // Get current element IDs
        const currentNodeIds = new Set(cy.nodes().map(n => n.id()));
        const currentEdgeIds = new Set(cy.edges().map(e => `${e.data('source')}|${e.data('target')}|${e.data('type')}`));

        // Build sets of new element IDs
        const newNodeIds = new Set(elements.filter(e => !e.data.source).map(e => e.data.id));
        const newEdgeIds = new Set(elements.filter(e => e.data.source).map(e => `${e.data.source}|${e.data.target}|${e.data.type}`));

        // Remove elements that no longer exist
        const nodesToRemove = Array.from(currentNodeIds).filter(id => !newNodeIds.has(id));
        const edgesToRemove = Array.from(currentEdgeIds).filter(id => !newEdgeIds.has(id));

        console.log(`[Cytoscape] Removing ${nodesToRemove.length} nodes, ${edgesToRemove.length} edges`);
        nodesToRemove.forEach(id => cy.getElementById(id).remove());
        edgesToRemove.forEach(id => {
            const [source, target, type] = id.split('|');
            cy.edges(`[source = "${source}"][target = "${target}"][type = "${type}"]`).remove();
        });

        // Add new elements
        const elementsToAdd = elements.filter(e => {
            if (e.data.source) {
                // Edge
                const id = `${e.data.source}|${e.data.target}|${e.data.type}`;
                return !currentEdgeIds.has(id);
            } else {
                // Node
                return !currentNodeIds.has(e.data.id);
            }
        });

        console.log(`[Cytoscape] Adding ${elementsToAdd.filter(e => !e.data.source).length} nodes, ${elementsToAdd.filter(e => e.data.source).length} edges`);
        if (elementsToAdd.length > 0) {
            cy.add(elementsToAdd);
        }

        cy.endBatch();

        // Run layout with animation and no viewport reset
        cy.layout({
            name: 'dagre',
            rankDir: 'TB',
            ranker: 'network-simplex',
            nodeSep: 80,
            edgeSep: 20,
            rankSep: 120,
            fit: false,          // Don't reset viewport/zoom
            animate: true,       // Smooth transitions
            animationDuration: 1000,  // Slow animation for debugging
            animationEasing: 'ease-out',
            padding: 50
        }).run();
    }
}

/**
 * Setup all event handlers for the cytoscape instance
 * Only called once during initial graph creation
 */
function setupEventHandlers() {
    if (!cy) return;

    // Add tooltip for edges and nodes
    const tooltip = document.createElement('div');
    tooltip.id = 'edge-tooltip';
    tooltip.style.position = 'absolute';
    tooltip.style.display = 'none';
    tooltip.style.backgroundColor = 'rgba(0, 0, 0, 0.85)';
    tooltip.style.color = 'white';
    tooltip.style.padding = '8px 12px';
    tooltip.style.borderRadius = '4px';
    tooltip.style.fontSize = '12px';
    tooltip.style.maxWidth = '400px';
    tooltip.style.zIndex = '10000';
    tooltip.style.pointerEvents = 'none';
    tooltip.style.whiteSpace = 'pre-wrap';
    tooltip.style.fontFamily = 'monospace';
    document.body.appendChild(tooltip);

    // Tooltip hover delay
    let tooltipTimeout = null;

    // Show tooltip on edge hover with delay
    cy.on('mouseover', 'edge', function(evt) {
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
        const sourceLabel = edge.data('sourceLabel') || (() => {
            const sourceNode = cy.getElementById(edge.data('source'));
            return sourceNode.data('label') || edge.data('source');
        })();
        const targetLabel = edge.data('targetLabel') || (() => {
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
                const simplifiedSymbols = symbols.map(s => simplifySymbol(s));
                const symbolList = simplifiedSymbols.slice(0, 10).join(', ');
                const more = symbols.length > 10 ? ` ... +${symbols.length - 10} more` : '';
                tooltipText += `\n\nSymbols (${symbols.length}): ${symbolList}${more}`;
            }
        } else if (edgeType === 'dynamic') {
            tooltipText = `🔗 Dynamic Linkage\n\n${sourceLabel}\n  depends on (dynamically links)\n${targetLabel}\n\nShared library ${targetLabel} is loaded at runtime.`;

            // Add symbols if available
            if (symbols && symbols.length > 0) {
                const simplifiedSymbols = symbols.map(s => simplifySymbol(s));
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
            const linkageDesc = linkage === 'static' ? 'statically linked' :
                              linkage === 'dynamic' ? 'dynamically linked' :
                              linkage === 'cross' ? 'cross-binary' : linkage;
            tooltipText = `🔧 Symbol Dependency (${linkageDesc})\n\n${sourceLabel}\n  uses symbols from\n${targetLabel}`;

            // Add symbol list for symbol edges
            if (symbols && symbols.length > 0) {
                const simplifiedSymbols = symbols.map(s => simplifySymbol(s));
                const symbolList = simplifiedSymbols.slice(0, 15).join('\n  ');
                const more = symbols.length > 15 ? `\n  ... and ${symbols.length - 15} more` : '';
                tooltipText += `\n\nSymbols used (${symbols.length}):\n  ${symbolList}${more}`;
            }
        } else {
            tooltipText = `Dependency: ${sourceLabel} → ${targetLabel}\nType: ${edgeType || 'unknown'}`;
        }

        tooltip.textContent = tooltipText;
        tooltip.style.display = 'block';
        }, 500); // 500ms delay
    });

    cy.on('mousemove', 'edge', function(evt) {
        tooltip.style.left = (evt.originalEvent.pageX + 10) + 'px';
        tooltip.style.top = (evt.originalEvent.pageY + 10) + 'px';
    });

    cy.on('mouseout', 'edge', function(evt) {
        // Clear timeout and hide tooltip
        if (tooltipTimeout) {
            clearTimeout(tooltipTimeout);
            tooltipTimeout = null;
        }
        tooltip.style.display = 'none';
    });

    // Tooltip for nodes with delay
    cy.on('mouseover', 'node', function(evt) {
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
                tooltipText = '📦 Binary (cc_binary)\nExecutable program.\nLinks dependencies into final executable.';
            } else if (nodeType === 'cc_library') {
                tooltipText = '📚 Library (cc_library)\nStatic library.\nCompiled code reused by other targets.';
            } else if (nodeType === 'cc_shared_library') {
                tooltipText = '🔗 Shared Library (cc_shared_library)\nDynamic library (.so/.dylib).\nLoaded at runtime, shared between processes.';
            } else if (nodeType === 'system_library') {
                tooltipText = '⚙️ System Library\nExternal library from the system.\nProvided by OS or installed separately.';
            } else if (nodeType === 'target-group') {
                tooltipText = '📁 Target Container\nGroups files within a target.\nClick to focus on this target.';
            } else if (nodeType && nodeType.startsWith('source')) {
                tooltipText = '📄 Source File (.cc/.cpp)\nImplementation file.\nCompiled into object code.';
            } else if (nodeType && nodeType.startsWith('header')) {
                tooltipText = '📋 Header File (.h/.hpp)\nInterface definitions.\nIncluded by source files.';
            } else if (nodeType) {
                tooltipText = `Type: ${nodeType}\n${nodeLabel}`;
            } else {
                tooltipText = nodeLabel;
            }
        }

        tooltip.textContent = tooltipText;
        tooltip.style.display = 'block';
        }, 500); // 500ms delay
    });

    cy.on('mousemove', 'node', function(evt) {
        tooltip.style.left = (evt.originalEvent.pageX + 10) + 'px';
        tooltip.style.top = (evt.originalEvent.pageY + 10) + 'px';
    });

    cy.on('mouseout', 'node', function(evt) {
        // Clear timeout and hide tooltip
        if (tooltipTimeout) {
            clearTimeout(tooltipTimeout);
            tooltipTimeout = null;
        }
        tooltip.style.display = 'none';
    });

    // Hide tooltip when clicking anywhere (fixes tooltip staying visible after click)
    cy.on('tap', function(evt) {
        // Clear timeout and hide tooltip
        if (tooltipTimeout) {
            clearTimeout(tooltipTimeout);
            tooltipTimeout = null;
        }
        tooltip.style.display = 'none';
    });

    // Click on graph background to clear focus
    cy.on('tap', function(evt) {
        // Check if we clicked on the background (not a node or edge)
        if (evt.target === cy) {
            console.log('Background clicked - clearing focus');
            viewStateManager.clearFocus();
        }
    });

    // Track click timing to distinguish single from double clicks
    let clickTimeout = null;

    // Single-click on nodes to focus them (with delay to check for double-click)
    cy.on('tap', 'node', function(evt) {
        const node = evt.target;
        const nodeId = node.data('id');

        // Clear any existing timeout
        if (clickTimeout) {
            clearTimeout(clickTimeout);
            clickTimeout = null;
        }

        // Set a timeout to handle single click
        clickTimeout = setTimeout(function() {
            console.log('Node clicked (focus):', nodeId);
            viewStateManager.updateFocus(nodeId);
        }, 250); // 250ms delay to check for double-click
    });

    // Double-click on nodes to toggle manual collapse
    cy.on('dbltap', 'node', function(evt) {
        const node = evt.target;
        const nodeId = node.data('id');
        const nodeType = node.data('type');

        // Cancel the single-click timeout
        if (clickTimeout) {
            clearTimeout(clickTimeout);
            clickTimeout = null;
        }

        console.log('Node double-clicked (toggle collapse):', nodeId, 'type:', nodeType);

        const currentState = viewStateManager.getState();
        const manual = currentState.manualOverrides.get(nodeId);

        // Determine current collapse state:
        // 1. If there's a manual override, use that
        // 2. Otherwise, use the default from lens collapseLevel
        let currentCollapsed;
        if (manual !== undefined && manual.collapsed !== null && manual.collapsed !== undefined) {
            currentCollapsed = manual.collapsed;
        } else {
            // Use the lens's collapseLevel to determine default state
            const collapseLevel = currentState.defaultLens.distanceRules[0]?.collapseLevel || 3;

            // Determine node level
            let nodeLevel = 0;
            if (nodeType === 'package') {
                nodeLevel = 1;
            } else if (nodeType === 'cc_binary' || nodeType === 'cc_library' || nodeType === 'cc_shared_library') {
                nodeLevel = 2;
            } else if (nodeType === 'source_file' || nodeType === 'header_file') {
                nodeLevel = 3;
            }

            // Check if this node should be collapsed based on collapseLevel
            if (nodeLevel === 2 && collapseLevel < 3) {
                currentCollapsed = true; // Collapse targets when we don't want to see files
            } else if (nodeLevel === 1 && collapseLevel < 2) {
                currentCollapsed = true; // Collapse packages when we don't want to see targets
            } else {
                currentCollapsed = false;
            }
        }

        console.log('Current collapse state:', currentCollapsed, 'Manual override:', manual?.collapsed);

        // Toggle: flip the current state
        const newCollapsed = !currentCollapsed;
        console.log('Setting new collapse state:', newCollapsed);

        viewStateManager.setManualOverride(nodeId, newCollapsed);
    });

    // Set explicit dimensions based on flex container size
    updateCytoscapeSize();

    // Center and fit the graph after layout completes and canvas is ready
    cy.one('layoutstop', function() {
        // Small delay to ensure canvas has final dimensions
        setTimeout(function() {
            // Notify cytoscape that container dimensions are finalized
            cy.resize();

            // Center on all elements (the entire graph)
            cy.center(cy.elements());
            // Then fit to viewport with padding
            cy.fit(cy.elements(), 50);
        }, 10);
    });
}

/**
 * Apply collapse states to parent nodes in the graph
 * Collapsed parent nodes will have their children hidden
 */
function applyCollapseStates(nodeStates) {
    if (!cy || !nodeStates) return;

    console.log('[CollapseStates] Applying collapse states to graph');

    // First, show all nodes (reset state)
    cy.nodes().show();

    // Then hide descendants of collapsed parent nodes
    cy.nodes(':parent').forEach(parentNode => {
        const parentId = parentNode.data('id');
        const state = nodeStates.get(parentId);

        if (state && state.collapsed) {
            // Hide all descendants (children and their children recursively)
            const descendants = parentNode.descendants();
            descendants.hide();
            console.log(`[CollapseStates] Collapsed ${parentId}, hiding ${descendants.length} descendants`);
        }
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
    container.style.width = rect.width + 'px';
    container.style.height = rect.height + 'px';

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
let reconnectAttempts = 0;
const MAX_RECONNECT_ATTEMPTS = 3;
let lastSuccessfulRequest = Date.now();
let healthCheckInterval = null;

// Map workspace status state to loading step
const statusToStep = {
    'initializing': 1,
    'bazel_querying': 1,
    'analyzing_deps': 2,
    'analyzing_symbols': 3,
    'discovering_files': 4,
    'targets_ready': 5,
    'analyzing_binaries': 5,
    'ready': 6
};

// Handle connection lost
function handleConnectionLost(source) {
    if (connectionLost) {
        return; // Already handling connection loss
    }

    connectionLost = true;
    console.error('Connection lost to backend server (source:', source, ')');

    // Show connection lost modal
    showConnectionLostModal();
}

// Show connection lost modal
function showConnectionLostModal() {
    const modal = document.getElementById('connectionLostModal');
    const messageEl = document.getElementById('connectionErrorMessage');

    if (reconnectAttempts > 0) {
        messageEl.textContent = `Reconnection attempt ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS} failed.`;
    } else {
        messageEl.textContent = 'Please check that the backend server is running.';
    }

    modal.style.display = 'flex';

    // Set up button handlers
    document.getElementById('retryConnection').onclick = attemptReconnect;
    document.getElementById('reloadPage').onclick = () => window.location.reload();
}

// Hide connection lost modal
function hideConnectionLostModal() {
    document.getElementById('connectionLostModal').style.display = 'none';
}

// Attempt to reconnect
function attemptReconnect() {
    if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
        const messageEl = document.getElementById('connectionErrorMessage');
        messageEl.textContent = 'Maximum reconnection attempts reached. Please reload the page.';
        document.getElementById('retryConnection').disabled = true;
        return;
    }

    reconnectAttempts++;
    hideConnectionLostModal();

    // Try to reconnect by testing the API
    fetch('/api/module')
        .then(response => {
            if (response.ok) {
                // Connection restored
                console.log('Connection restored, reloading page...');
                window.location.reload();
            } else {
                throw new Error('Server not ready');
            }
        })
        .catch(error => {
            console.error('Reconnection failed:', error);
            // Wait a bit before showing modal again
            setTimeout(() => {
                if (connectionLost) {
                    showConnectionLostModal();
                }
            }, 1000);
        });
}

// Wrapper for fetch that detects connection failures
function monitoredFetch(url, options) {
    return fetch(url, options)
        .then(response => {
            if (response.ok) {
                lastSuccessfulRequest = Date.now();
            }
            return response;
        })
        .catch(error => {
            console.error('Fetch failed:', url, error);
            // Network error - backend likely down
            handleConnectionLost('fetch');
            throw error;
        });
}

// Start periodic health check
function startHealthCheck() {
    // Check every 2 seconds if we haven't had a successful request in 3 seconds
    healthCheckInterval = setInterval(() => {
        const timeSinceLastSuccess = Date.now() - lastSuccessfulRequest;

        // Only check if analysis is complete and it's been a while
        if (analysisComplete && timeSinceLastSuccess > 3000 && !connectionLost) {
            console.log('Performing health check...');
            fetch('/api/module', { method: 'HEAD' })
                .then(response => {
                    if (response.ok) {
                        lastSuccessfulRequest = Date.now();
                    } else {
                        handleConnectionLost('health_check');
                    }
                })
                .catch(() => {
                    handleConnectionLost('health_check');
                });
        }
    }, 2000);
}

// Stop health check
function stopHealthCheck() {
    if (healthCheckInterval) {
        clearInterval(healthCheckInterval);
        healthCheckInterval = null;
    }
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
            .then(response => {
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

    console.log('Activity-based connection monitoring enabled');
}

// Subscribe to workspace status events
function subscribeToWorkspaceStatus() {
    console.log('Creating EventSource for workspace_status...');
    workspaceStatusSource = new EventSource('/api/subscribe/workspace_status');

    workspaceStatusSource.onopen = function() {
        console.log('workspace_status EventSource connected, readyState:', workspaceStatusSource.readyState);
    };

    workspaceStatusSource.onmessage = function(event) {
        console.log('Raw workspace_status event data:', event.data);
        try {
            const sseEvent = JSON.parse(event.data);
            console.log('Parsed SSE event:', sseEvent);

            // sseEvent.data is json.RawMessage (already a JSON string), parse it
            let status;
            if (typeof sseEvent.data === 'string') {
                status = JSON.parse(sseEvent.data);
            } else {
                status = sseEvent.data; // Already an object (shouldn't happen with json.RawMessage)
            }
            console.log('Parsed status:', status);

            console.log('Workspace status:', status.state, '-', status.message);

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

                // Start health check to detect backend failures
                if (!healthCheckInterval) {
                    startHealthCheck();
                }
            }
        } catch (e) {
            console.error('Error processing workspace status:', e, 'Raw data:', event.data);
        }
    };

    workspaceStatusSource.onerror = function(error) {
        console.error('Workspace status SSE error:', error, 'readyState:', workspaceStatusSource.readyState);

        // EventSource readyState: 0 = CONNECTING, 1 = OPEN, 2 = CLOSED
        if (workspaceStatusSource.readyState === 2) {
            handleConnectionLost('workspace_status');
        }
    };
}

// Subscribe to target graph events
function subscribeToTargetGraph() {
    targetGraphSource = new EventSource('/api/subscribe/target_graph');

    targetGraphSource.onmessage = function(event) {
        try {
            const sseEvent = JSON.parse(event.data);

            // sseEvent.data is json.RawMessage (already a JSON string), parse it
            let graphData;
            if (typeof sseEvent.data === 'string') {
                graphData = JSON.parse(sseEvent.data);
            } else {
                graphData = sseEvent.data;
            }

            console.log('Target graph update:', sseEvent.type, 'complete:', graphData.complete);

            // Load full graph data when available
            if (sseEvent.type === 'complete' && !graphDataLoaded) {
                loadGraphData();
                graphDataLoaded = true;
            }
        } catch (e) {
            console.error('Error processing target graph event:', e);
        }
    };

    targetGraphSource.onerror = function(error) {
        console.error('Target graph SSE error:', error);

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

    binaries.forEach(binary => {
        if (binary.overlappingDeps) {
            Object.entries(binary.overlappingDeps).forEach(([sharedLib, targets]) => {
                targets.forEach(target => {
                    if (!allOverlappingTargets.has(target)) {
                        allOverlappingTargets.set(target, new Set());
                    }
                    allOverlappingTargets.get(target).add(sharedLib);
                });
            });
        }
    });

    console.log('Overlapping targets found:', Array.from(allOverlappingTargets.keys()));

    // Mark nodes that have overlapping dependencies
    graph.nodes.forEach(node => {
        const nodeLabel = node.label || node.id;
        if (allOverlappingTargets.has(nodeLabel)) {
            node.hasOverlap = true;
            node.overlappingWith = Array.from(allOverlappingTargets.get(nodeLabel));
            console.log('Marked node as overlapping:', nodeLabel, 'with:', node.overlappingWith);
        }
    });

    // Mark shared library nodes that have overlapping deps
    binaries.forEach(binary => {
        if (binary.kind === 'cc_shared_library' && binary.overlappingDeps) {
            const overlappingCount = Object.keys(binary.overlappingDeps).length;
            if (overlappingCount > 0) {
                const node = graph.nodes.find(n => (n.label || n.id) === binary.label);
                if (node) {
                    node.hasOverlap = true;
                    node.overlappingTargets = Object.values(binary.overlappingDeps).flat();
                    console.log('Marked shared library as overlapping:', binary.label);
                }
            }
        }
    });
}

// Load full graph data from API
async function loadGraphData() {
    console.log('loadGraphData() called');
    try {
        // Fetch module graph
        const graphResponse = await monitoredFetch('/api/module/graph');
        console.log('Module graph response status:', graphResponse.status);
        if (graphResponse.ok) {
            packageGraph = await graphResponse.json();
            console.log('Loaded graph with', packageGraph.nodes?.length, 'nodes');

            // Render through backend lens API to ensure proper hierarchy and collapse states
            if (packageGraph && packageGraph.nodes && packageGraph.nodes.length > 0) {
                console.log('Graph loaded, rendering through backend lens API');
                try {
                    const currentState = viewStateManager.getState();
                    const renderedGraph = await fetchRenderedGraphFromBackend(currentState);
                    displayDependencyGraph(renderedGraph);
                } catch (error) {
                    console.error('Error rendering graph via backend:', error);
                }
            } else {
                console.warn('Package graph has no nodes or is invalid:', packageGraph);
            }
        } else {
            console.error('Failed to fetch module graph:', graphResponse.status, graphResponse.statusText);
        }

        // Fetch binary data
        const binariesResponse = await monitoredFetch('/api/binaries');
        if (binariesResponse.ok) {
            binaryData = await binariesResponse.json();
            console.log('Loaded binary data:', binaryData);

            // Enrich the displayed graph with overlapping dependency information
            if (packageGraph && packageGraph.nodes) {
                enrichGraphWithOverlappingInfo(packageGraph, binaryData);
                // Redisplay the graph with overlapping info through backend lens API
                try {
                    const currentState = viewStateManager.getState();
                    const renderedGraph = await fetchRenderedGraphFromBackend(currentState);
                    displayDependencyGraph(renderedGraph);
                } catch (error) {
                    console.error('Error rendering graph via backend:', error);
                }
            }
        }

        // Fetch module data for tree browser
        const moduleResponse = await monitoredFetch('/api/module');
        if (moduleResponse.ok) {
            const moduleData = await moduleResponse.json();
            analysisData = {
                graph: packageGraph,
                module: moduleData
            };

            // Update subtitle with module name
            updateModuleName(moduleData.name);

            // Populate tree browser
            if (packageGraph && packageGraph.nodes) {
                populateTreeBrowser(analysisData);
            }

            // Initialize lens controls (after DOM is ready and data is loaded)
            initializeLensControls();

            // Populate binary selector for lens controls
            if (binaryData && binaryData.length > 0) {
                populateBinarySelector(binaryData);
            }

            // Trigger initial render with backend lens API
            try {
                const renderedGraph = await fetchRenderedGraphFromBackend(viewStateManager.getState());
                displayDependencyGraph(renderedGraph);
            } catch (error) {
                console.error('Error rendering graph via backend:', error);
            }
        }
    } catch (e) {
        console.error('Error loading graph data:', e);
    }
}

// Clean up connections when page is being unloaded
window.addEventListener('beforeunload', function() {
    console.log('Page unloading, closing SSE connections');
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

// Initialize subscriptions when page loads
document.addEventListener('DOMContentLoaded', function() {
    console.log('Starting SSE subscriptions...');

    // Close any existing connections first (in case of reload)
    if (workspaceStatusSource) {
        console.log('Closing existing workspace_status connection');
        workspaceStatusSource.close();
        workspaceStatusSource = null;
    }
    if (targetGraphSource) {
        console.log('Closing existing target_graph connection');
        targetGraphSource.close();
        targetGraphSource = null;
    }

    // Destroy any existing Cytoscape instance
    if (cy) {
        console.log('Destroying existing Cytoscape instance');
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

// Modal functions for showing target details
async function showTargetDetails(targetLabel) {
    try {
        // Encode the target label for URL
        const encodedLabel = encodeURIComponent(targetLabel);
        const response = await monitoredFetch(`/api/target/${encodedLabel}`);
        
        if (!response.ok) {
            console.error('Failed to fetch target details');
            return;
        }
        
        const details = await response.json();
        displayTargetModal(details);
    } catch (error) {
        console.error('Error fetching target details:', error);
    }
}

function displayTargetModal(details) {
    const modal = document.getElementById('targetModal');
    const labelEl = document.getElementById('modalTargetLabel');
    const filesEl = document.getElementById('modalFiles');
    const incomingEl = document.getElementById('modalIncoming');
    const outgoingEl = document.getElementById('modalOutgoing');
    
    // Set title
    labelEl.textContent = simplifyLabel(details.targetLabel);
    
    // Display files in target
    filesEl.innerHTML = '';
    if (details.files && details.files.length > 0) {
        details.files.forEach(file => {
            const fileDiv = document.createElement('div');
            fileDiv.className = `modal-file-item ${file.type}`;
            fileDiv.textContent = file.path;
            filesEl.appendChild(fileDiv);
        });
    } else {
        filesEl.innerHTML = '<div class="modal-empty">No files found (may need .d files)</div>';
    }
    
    // Display incoming dependencies
    incomingEl.innerHTML = '';
    if (details.incomingFileDeps && details.incomingFileDeps.length > 0) {
        details.incomingFileDeps.forEach(dep => {
            const depDiv = document.createElement('div');
            depDiv.className = 'modal-dep-item';
            
            const fileDiv = document.createElement('div');
            fileDiv.className = 'modal-dep-file';
            fileDiv.textContent = `${dep.sourceFile} → ${dep.targetFile}`;
            
            const targetDiv = document.createElement('div');
            targetDiv.className = 'modal-dep-target';
            targetDiv.textContent = `From: ${dep.sourceTarget}`;
            
            depDiv.appendChild(fileDiv);
            depDiv.appendChild(targetDiv);
            incomingEl.appendChild(depDiv);
        });
    } else {
        incomingEl.innerHTML = '<div class="modal-empty">No incoming dependencies</div>';
    }
    
    // Display outgoing dependencies
    outgoingEl.innerHTML = '';
    if (details.outgoingFileDeps && details.outgoingFileDeps.length > 0) {
        details.outgoingFileDeps.forEach(dep => {
            const depDiv = document.createElement('div');
            depDiv.className = 'modal-dep-item';
            
            const fileDiv = document.createElement('div');
            fileDiv.className = 'modal-dep-file';
            fileDiv.textContent = `${dep.sourceFile} → ${dep.targetFile}`;
            
            const targetDiv = document.createElement('div');
            targetDiv.className = 'modal-dep-target';
            targetDiv.textContent = `To: ${dep.targetTarget}`;
            
            depDiv.appendChild(fileDiv);
            depDiv.appendChild(targetDiv);
            outgoingEl.appendChild(depDiv);
        });
    } else {
        outgoingEl.innerHTML = '<div class="modal-empty">No outgoing dependencies</div>';
    }
    
    // Show modal
    modal.style.display = 'block';
}

// Close modal handlers
document.addEventListener('DOMContentLoaded', function() {
    const modal = document.getElementById('targetModal');
    const closeBtn = document.querySelector('.modal-close');

    // Close when clicking X
    closeBtn.onclick = function() {
        modal.style.display = 'none';
    };

    // Close when clicking outside modal
    window.onclick = function(event) {
        if (event.target == modal) {
            modal.style.display = 'none';
        }
    };
});

// ============================================================================
// Tree Browser Implementation
// ============================================================================

let treeData = null;
let analysisData = null; // Store full analysis data
let packageGraph = null; // Store the original package-level graph
let binaryGraph = null; // Store the binary-level graph
let binaryData = null; // Store binary information
let cy = null; // Store the Cytoscape instance
// ===== Lens-Based Visualization System =====
// Replaced currentView, currentTarget, currentBinary with lens system
// All lens rendering now happens server-side via /api/module/graph/lens

// Current graph state hash (for diff-based updates)
let currentGraphHash = null;

// Current graph data (for applying diffs)
let currentGraphData = null;

/**
 * Fetch rendered graph from backend lens API
 * @param {Object} viewState - Current view state with lens configurations
 * @returns {Promise<Object>} Rendered graph from backend
 */
async function fetchRenderedGraphFromBackend(viewState) {
  console.log('[App] Fetching rendered graph from backend lens API');

  // Convert edgeRules.types from Set to Array for JSON serialization
  const serializeLens = (lens) => ({
    ...lens,
    edgeRules: {
      ...lens.edgeRules,
      types: Array.from(lens.edgeRules.types)
    }
  });

  const requestBody = {
    defaultLens: serializeLens(viewState.defaultLens),
    focusLens: serializeLens(viewState.focusLens),
    focusedNodes: Array.from(viewState.focusedNodes),
    manualOverrides: Object.fromEntries(viewState.manualOverrides),
    previousHash: currentGraphHash  // Send previous hash for diff-based updates
  };

  console.log('[App] Request body:', JSON.stringify(requestBody, null, 2));

  const response = await fetch('/api/module/graph/lens', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(requestBody)
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
    console.log('[App] Received full graph from backend:', responseData.fullGraph.nodes?.length, 'nodes,', responseData.fullGraph.edges?.length, 'edges');
    renderedGraph = responseData.fullGraph;
    currentGraphData = renderedGraph;
  } else if (responseData.diff) {
    console.log('[App] Received diff from backend:',
      responseData.diff.addedNodes?.length || 0, 'added nodes,',
      responseData.diff.removedNodes?.length || 0, 'removed nodes,',
      responseData.diff.addedEdges?.length || 0, 'added edges,',
      responseData.diff.removedEdges?.length || 0, 'removed edges');
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
  const nodeMap = new Map(currentGraph.nodes.map(n => [n.id, n]));
  const edgeMap = new Map(currentGraph.edges.map(e => [`${e.source}|${e.target}|${e.type}`, e]));

  // Apply node changes
  if (diff.removedNodes) {
    diff.removedNodes.forEach(nodeId => nodeMap.delete(nodeId));
  }

  if (diff.addedNodes) {
    diff.addedNodes.forEach(node => nodeMap.set(node.id, node));
  }

  if (diff.modifiedNodes) {
    diff.modifiedNodes.forEach(node => nodeMap.set(node.id, node));
  }

  // Apply edge changes
  if (diff.removedEdges) {
    diff.removedEdges.forEach(edgeKey => edgeMap.delete(edgeKey));
  }

  if (diff.addedEdges) {
    diff.addedEdges.forEach(edge => {
      const key = `${edge.source}|${edge.target}|${edge.type}`;
      edgeMap.set(key, edge);
    });
  }

  // Convert maps back to arrays
  return {
    nodes: Array.from(nodeMap.values()),
    edges: Array.from(edgeMap.values())
  };
}

// Track previous state for detecting when full re-layout is needed
let previousViewState = null;

// Listen for state changes and re-render graph
viewStateManager.addListener(async (newState) => {
  if (!packageGraph) return; // Wait for initial load

  console.log('[App] State changed, rendering with backend API');
  console.log('[App] BaseSet:', newState.defaultLens.baseSet);
  console.log('[App] Focused nodes:', Array.from(newState.focusedNodes));

  // Deep clone the state to avoid reference issues
  // (Maps need special handling)
  previousViewState = {
    ...newState,
    focusedNodes: new Set(newState.focusedNodes),
    manualOverrides: new Map(newState.manualOverrides),
    defaultLens: JSON.parse(JSON.stringify(newState.defaultLens)),
    focusLens: JSON.parse(JSON.stringify(newState.focusLens))
  };

  try {
    // Fetch rendered graph from backend
    const renderedGraph = await fetchRenderedGraphFromBackend(newState);

    // Display the pre-rendered graph from backend
    displayDependencyGraph(renderedGraph);
  } catch (error) {
    console.error('[App] Error fetching rendered graph from backend:', error);
    console.error('[App] Backend lens rendering failed - this is a fatal error');
  }
});

// NOTE: Old tree-building functions removed
// buildTreeData(), createTreeNode(), toggleExpansion() are no longer needed
// The navigation now uses simple flat lists populated by populateTreeBrowser()

// NOTE: Old view-switching functions removed - now handled by lens system
// The following functions have been replaced by the lens-based visualization:
// - selectTreeNode() -> use viewStateManager.updateFocus()
// - showBinaryGraphFocused() -> handled by lens renderer
// - zoomOutOneLevel() -> use viewStateManager.clearFocus()
// - showFocusedTargetView() -> handled by lens renderer
// - showFileGraphForTarget() -> handled by lens renderer
// - selectBinary() -> use viewStateManager.updateFocus()
// - selectTarget() -> use viewStateManager.updateFocus()
// - showBinaryFocusedGraph() -> handled by lens renderer
// - buildBinaryFocusedGraphData() -> handled by lens renderer

// Populate the tree browser
function populateTreeBrowser(data) {
    console.log('Populating navigation with data:', data);

    // Populate binaries list
    const binariesItems = document.getElementById('binariesItems');
    if (binariesItems && binaryData && binaryData.length > 0) {
        binariesItems.innerHTML = '';
        binaryData.forEach(binary => {
            const item = document.createElement('div');
            item.className = 'nav-item';
            const icon = binary.kind === 'cc_binary' ? '🔧' : '📚';
            item.textContent = `${icon} ${simplifyLabel(binary.label)}`;

            // Click focuses on this binary
            item.onclick = () => {
                viewStateManager.updateFocus(binary.label);
                // Highlight in tree
                document.querySelectorAll('.nav-item').forEach(i => i.classList.remove('selected'));
                item.classList.add('selected');
            };

            binariesItems.appendChild(item);
        });
    }

    // Populate targets list (exclude system libraries)
    const targetsItems = document.getElementById('targetsItems');
    if (targetsItems && data.graph && data.graph.nodes) {
        targetsItems.innerHTML = '';
        data.graph.nodes
            .filter(node => node.type !== 'system_library')  // Exclude system libraries
            .forEach(node => {
                const item = document.createElement('div');
                item.className = 'nav-item';
                item.textContent = `📦 ${simplifyLabel(node.label)}`;

                // Click focuses on this target
                item.onclick = () => {
                    viewStateManager.updateFocus(node.label);
                    // Highlight in tree
                    document.querySelectorAll('.nav-item').forEach(i => i.classList.remove('selected'));
                    item.classList.add('selected');
                };

                targetsItems.appendChild(item);
            });
    }
}

// Handle window resize to update Cytoscape canvas size
let resizeTimeout;
window.addEventListener('resize', function() {
    // Debounce resize events to avoid excessive updates
    clearTimeout(resizeTimeout);
    resizeTimeout = setTimeout(function() {
        if (cy) {
            updateCytoscapeSize();
            cy.fit(undefined, 50); // Refit with padding after resize
        }
    }, 150);
});

// Handle horizontal resize of sidebar
(function() {
    const resizeHandle = document.getElementById('resizeHandle');
    const treeBrowser = document.getElementById('treeBrowser');
    let isResizing = false;
    let startX = 0;
    let startWidth = 0;

    if (!resizeHandle || !treeBrowser) return;

    resizeHandle.addEventListener('mousedown', function(e) {
        isResizing = true;
        startX = e.clientX;
        startWidth = treeBrowser.offsetWidth;
        resizeHandle.classList.add('active');

        // Prevent text selection during drag
        e.preventDefault();
        document.body.style.userSelect = 'none';
        document.body.style.cursor = 'ew-resize';
    });

    document.addEventListener('mousemove', function(e) {
        if (!isResizing) return;

        const delta = e.clientX - startX;
        const newWidth = startWidth + delta;

        // Respect min and max width constraints
        const minWidth = 200;
        const maxWidth = 600;
        const constrainedWidth = Math.max(minWidth, Math.min(maxWidth, newWidth));

        treeBrowser.style.width = constrainedWidth + 'px';

        // Update Cytoscape canvas size if it exists
        if (cy) {
            updateCytoscapeSize();
        }
    });

    document.addEventListener('mouseup', function() {
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
})();
