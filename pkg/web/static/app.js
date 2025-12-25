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

    // If we already have a cytoscape instance, destroy it first
    if (cy) {
        console.log('Destroying existing cytoscape instance');
        cy.destroy();
        cy = null;
    }

    // Create elements array
    const elements = [
        // Nodes
        ...graphData.nodes.map(node => {
            const nodeData = {
                id: node.id,
                label: node.label,
                type: node.type,
                parent: node.parent // For compound nodes (grouping)
            };

            // Mark the currently focused target-group with a special attribute
            if (node.type === 'target-group') {
                const targetLabel = node.id.replace('parent-', '');
                if ((currentView === 'file' || currentView === 'focused') && targetLabel === currentTarget) {
                    nodeData.focused = true;
                }
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

    console.log('Creating new cytoscape instance with', elements.length, 'elements');

    // Debug: Log overlapping flags
    const overlappingNodes = elements.filter(e => e.data.hasOverlap === true);
    const overlappingEdges = elements.filter(e => e.data.isOverlapping === true);
    console.log('Nodes with hasOverlap=true:', overlappingNodes.map(n => n.data.label || n.data.id));
    console.log('Edges with isOverlapping=true:', overlappingEdges.map(e => `${e.data.source} -> ${e.data.target}`));

    cy = cytoscape({
        container: document.getElementById('cy'),
        elements: elements,

        style: [
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
                selector: 'edge[type = "data_dependency"]',
                style: {
                    'line-color': '#89d185',
                    'target-arrow-color': '#89d185',
                    'width': 2,
                    'line-style': 'dashed'
                }
            },
            {
                selector: 'edge[type = "system_link"]',
                style: {
                    'line-color': '#d7ba7d',
                    'target-arrow-color': '#d7ba7d',
                    'width': 2,
                    'line-style': 'dotted'
                }
            },
            {
                selector: 'edge[type = "compile"]',
                style: {
                    'line-color': '#4fc1ff',
                    'target-arrow-color': '#4fc1ff',
                    'width': 1.5,
                    'line-style': 'dotted'
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
                    'line-color': '#c586c0',
                    'target-arrow-color': '#c586c0',
                    'width': 2.5,
                    'line-style': 'solid'
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
            // Public visibility indicator (star badge)
            {
                selector: 'node[isPublic]',
                style: {
                    'border-style': 'dashed',
                    'border-width': '3px',
                    'border-color': '#ffd700'
                }
            }
        ],

        layout: {
            name: 'dagre',
            rankDir: 'TB',           // Top-to-bottom layout (use 'LR' for left-to-right)
            ranker: 'network-simplex', // Algorithm: 'network-simplex', 'tight-tree', or 'longest-path'
            nodeSep: 80,             // Horizontal spacing between nodes
            edgeSep: 20,             // Spacing between edges
            rankSep: 120,            // Vertical spacing between ranks/layers
            animate: false,
            padding: 50
        }
    });

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
                const symbolList = symbols.slice(0, 10).join(', ');
                const more = symbols.length > 10 ? ` ... +${symbols.length - 10} more` : '';
                tooltipText += `\n\nSymbols (${symbols.length}): ${symbolList}${more}`;
            }
        } else if (edgeType === 'dynamic') {
            tooltipText = `🔗 Dynamic Linkage\n\n${sourceLabel}\n  depends on (dynamically links)\n${targetLabel}\n\nShared library ${targetLabel} is loaded at runtime.`;

            // Add symbols if available
            if (symbols && symbols.length > 0) {
                const symbolList = symbols.slice(0, 10).join(', ');
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
                const symbolList = symbols.slice(0, 15).join('\n  ');
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

    // Click on graph background to zoom out one level
    cy.on('tap', function(evt) {
        // Check if we clicked on the background (not a node or edge)
        if (evt.target === cy) {
            console.log('Background clicked - zooming out');
            zoomOutOneLevel();
        }
    });

    // Click on nodes to select them in the tree or navigate
    cy.on('tap', 'node', function(evt) {
        const node = evt.target;
        const nodeId = node.data('id');
        const nodeType = node.data('type');

        console.log('Node clicked:', nodeId, 'Type:', nodeType);

        if (currentView === 'package') {
            // At package level, clicking a target should go into focused mode
            // Check if this is a target node (not a package)
            if (!nodeType.includes('package') && !nodeType.includes('system')) {
                console.log('Target node clicked in package view, entering focused mode');
                showFocusedTargetView(nodeId);
            }
        } else if (currentView === 'focused') {
            // In focused mode, clicking a target-group or different target switches focus
            if (nodeType === 'target-group') {
                // Clicked a target group container - extract target label
                const targetLabel = nodeId.replace('parent-', '');
                console.log('Target group clicked, switching focus to:', targetLabel);
                showFocusedTargetView(targetLabel);
            } else if (nodeType.includes('_incoming') || nodeType.includes('_outgoing')) {
                // Clicked a file within an incoming/outgoing target - switch to that target
                // File node ID format: "//target:name:file:..." - extract the target part
                const targetLabel = nodeId.split(':file:')[0];
                console.log('File clicked, switching focus to parent target:', targetLabel);
                showFocusedTargetView(targetLabel);
            }
        } else if (currentView === 'file') {
            // At file level, check what type of node was clicked
            if (nodeType === 'target-group') {
                // Clicked a target group - switch to that target's file view
                // Extract target label from parent node ID (format: "parent-//target:name")
                const targetLabel = nodeId.replace('parent-', '');

                // Check if we're already viewing this target - if so, do nothing
                if (targetLabel === currentTarget) {
                    console.log('Already viewing target:', targetLabel);
                    return;
                }

                console.log('Target group clicked, switching to:', targetLabel);

                // Find and click the corresponding target in the tree
                const targetNodes = document.querySelectorAll('.tree-node[data-type="target"]');
                for (const treeNode of targetNodes) {
                    if (treeNode.dataset.id === targetLabel) {
                        console.log('Match found! Clicking tree node for:', targetLabel);
                        const label = treeNode.querySelector('.tree-label');
                        if (label) {
                            label.click();
                            return;
                        }
                    }
                }
                console.log('No matching tree node found for:', targetLabel);
            } else {
                // Clicked a file node - could highlight it in the tree
                console.log('File node clicked:', nodeId);
            }
        }
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

// Update Cytoscape canvas size based on actual container dimensions
function updateCytoscapeSize() {
    const container = document.getElementById('cy');
    if (!container) return;

    // Get the actual computed size of the flex container
    const rect = container.getBoundingClientRect();

    // Set explicit pixel dimensions
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
    // Check every 5 seconds if we haven't had a successful request in 10 seconds
    healthCheckInterval = setInterval(() => {
        const timeSinceLastSuccess = Date.now() - lastSuccessfulRequest;

        // Only check if analysis is complete and it's been a while
        if (analysisComplete && timeSinceLastSuccess > 10000 && !connectionLost) {
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
    }, 5000);
}

// Stop health check
function stopHealthCheck() {
    if (healthCheckInterval) {
        clearInterval(healthCheckInterval);
        healthCheckInterval = null;
    }
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
            } else if (status.state === 'ready') {
                updateLoadingProgress(5, null); // Mark step 5 complete
                analysisComplete = true;

                hideLoadingOverlay();

                // Load graph data if we haven't already
                if (!graphDataLoaded) {
                    loadGraphData();
                    graphDataLoaded = true;
                }

                // Close SSE connections when done
                if (workspaceStatusSource) {
                    workspaceStatusSource.close();
                    workspaceStatusSource = null;
                }
                if (targetGraphSource) {
                    targetGraphSource.close();
                    targetGraphSource = null;
                }

                // Start health check to detect backend failures
                startHealthCheck();
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

            // Display the graph
            if (packageGraph && packageGraph.nodes && packageGraph.nodes.length > 0) {
                console.log('Calling displayDependencyGraph with', packageGraph.nodes.length, 'nodes');
                displayDependencyGraph(packageGraph);
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
                // Redisplay the graph with overlapping info
                displayDependencyGraph(packageGraph);
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

            // Populate tree browser
            if (packageGraph && packageGraph.nodes) {
                populateTreeBrowser(analysisData);
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
    labelEl.textContent = details.targetLabel;
    
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
let selectedNode = null;
let analysisData = null; // Store full analysis data
let packageGraph = null; // Store the original package-level graph
let binaryGraph = null; // Store the binary-level graph
let binaryData = null; // Store binary information
let cy = null; // Store the Cytoscape instance
let currentView = 'package'; // Track current view: 'package', 'file', 'binary', or 'focused'
let currentTarget = null; // Track which target we're viewing at file or focused level
let currentBinary = null; // Track which binary we're viewing at binary level

// Build tree structure from analysis data
function buildTreeData(data) {
    const tree = {
        binaries: [],
        targets: [],
        uncoveredFiles: data.uncoveredFiles || []
    };

    // Build binaries section if available
    if (binaryData && binaryData.length > 0) {
        for (const binary of binaryData) {
            tree.binaries.push({
                label: binary.label,
                type: 'binary',
                kind: binary.kind,
                id: binary.label
            });
        }
    }

    // Build targets section with their files
    if (data.graph && data.graph.nodes) {
        for (const node of data.graph.nodes) {
            tree.targets.push({
                label: node.label,
                type: 'target',
                id: node.id,
                files: [] // Will be populated when fetched
            });
        }
    }

    return tree;
}

// Create a tree node element
function createTreeNode(item, type) {
    const node = document.createElement('div');
    node.className = 'tree-node';
    node.dataset.type = type;
    node.dataset.id = item.label || item.path || item;

    const content = document.createElement('div');
    content.className = 'tree-node-content';

    // Toggle arrow for expandable nodes
    const toggle = document.createElement('span');
    toggle.className = 'tree-toggle';
    if (type === 'binary' || type === 'target') {
        toggle.textContent = '▶';
        node.classList.add('collapsed');
    } else {
        toggle.classList.add('empty');
    }
    content.appendChild(toggle);

    // Icon and label
    const label = document.createElement('span');
    label.className = 'tree-label';

    if (type === 'binary') {
        const icon = item.kind === 'cc_binary' ? '🔧' : '📚';
        label.textContent = `${icon} ${item.label}`;
    } else if (type === 'target') {
        label.textContent = `📦 ${item.label}`;
    } else if (type === 'file') {
        const fileName = item.path ? item.path.split('/').pop() : item.split('/').pop();
        const icon = item.path && item.path.endsWith('.h') ? '📄' : '📝';
        label.textContent = `${icon} ${fileName}`;
        label.title = item.path || item; // Show full path on hover
    } else if (type === 'uncovered') {
        const filePath = item.Path || item;
        const fileName = filePath.split('/').pop();
        label.textContent = `⚠️ ${fileName}`;
        label.title = filePath; // Show full path on hover
    }

    content.appendChild(label);
    node.appendChild(content);

    // Add children container for binaries and targets
    if (type === 'binary' || type === 'target') {
        const children = document.createElement('div');
        children.className = 'tree-children';
        node.appendChild(children);

        // Click on arrow to toggle expansion only
        toggle.addEventListener('click', async (e) => {
            console.log('Toggle arrow clicked');
            e.stopPropagation();
            try {
                await toggleExpansion(node, item, type);
            } catch (error) {
                console.error('Error in toggleExpansion:', error);
            }
        });

        // Click on label to always select/zoom (regardless of expansion)
        label.addEventListener('click', async (e) => {
            console.log(`${type} label clicked:`, item.label);
            e.stopPropagation();
            try {
                await selectTreeNode(node, item, type);
            } catch (error) {
                console.error('Error in selectTreeNode:', error);
            }
        });
    } else {
        // Click handler for files
        content.addEventListener('click', (e) => {
            e.stopPropagation();
            selectTreeNode(node, item, type);
        });
    }

    return node;
}

// Toggle tree node expansion (without selecting)
async function toggleExpansion(node, item, type) {
    const toggle = node.querySelector('.tree-toggle');
    const children = node.querySelector('.tree-children');

    if (node.classList.contains('collapsed')) {
        // Expand: load children if not already loaded
        node.classList.remove('collapsed');
        toggle.textContent = '▼';

        if (children.children.length === 0 && item.label) {
            if (type === 'binary') {
                // For binaries, show dependency info (not expandable to children for now)
                // We'll just show a message - the graph will show the dependencies
                const infoMsg = document.createElement('div');
                infoMsg.className = 'tree-label';
                infoMsg.style.color = '#999';
                infoMsg.style.fontStyle = 'italic';
                infoMsg.style.marginLeft = '20px';
                infoMsg.textContent = 'Click to view dependencies in graph';
                children.appendChild(infoMsg);
            } else if (type === 'target') {
                // Fetch target details
                try {
                    const encodedLabel = encodeURIComponent(item.label);
                    const response = await monitoredFetch(`/api/target/${encodedLabel}`);
                    if (response.ok) {
                        const details = await response.json();
                        if (details.files && details.files.length > 0) {
                            details.files.forEach(file => {
                                const fileNode = createTreeNode(file, 'file');
                                children.appendChild(fileNode);
                            });
                        } else {
                            const emptyMsg = document.createElement('div');
                            emptyMsg.className = 'tree-label';
                            emptyMsg.style.color = '#999';
                            emptyMsg.style.fontStyle = 'italic';
                            emptyMsg.textContent = 'No files found';
                            children.appendChild(emptyMsg);
                        }
                    }
                } catch (error) {
                    console.error('Failed to load target files:', error);
                }
            }
        }
    } else {
        // Collapse
        node.classList.add('collapsed');
        toggle.textContent = '▶';
    }
}

// Select a tree node and update the view
async function selectTreeNode(node, item, type) {
    // Remove previous selection
    document.querySelectorAll('.tree-node.selected').forEach(n => {
        n.classList.remove('selected');
    });

    // Add selection to this node
    node.classList.add('selected');
    selectedNode = { node, item, type };

    // Update the main view based on selection
    if (type === 'project') {
        // Show package-level graph (the default view)
        console.log('Selected project - showing package graph');
        currentView = 'package';
        currentTarget = null;
        currentBinary = null;
        if (packageGraph) {
            displayDependencyGraph(packageGraph);
        }
    } else if (type === 'binary') {
        // Show binary-level graph focused on this binary
        console.log('Selected binary:', item.label);
        currentView = 'binary';
        currentBinary = item.label;
        currentTarget = null;
        await showBinaryGraphFocused(item.label);
    } else if (type === 'target') {
        // Show file-level graph for this target
        console.log('Selected target:', item.label);
        currentView = 'file';
        currentTarget = item.label;
        currentBinary = null;
        await showFileGraphForTarget(item.label);
    } else if (type === 'file') {
        // Keep showing the current target's file graph
        console.log('Selected file:', item.path);
        // File is already shown in the current graph
    } else if (type === 'uncovered') {
        // Show uncovered file (maybe highlight in uncovered section?)
        console.log('Selected uncovered file:', item);
    }
}

// Show binary-level graph focused on a specific binary
async function showBinaryGraphFocused(binaryLabel) {
    try {
        console.log('Fetching binary graph focused on:', binaryLabel);

        // For now, show the full binary graph (in the future we could filter by focused binary)
        if (!binaryGraph) {
            const response = await monitoredFetch('/api/binaries/graph');
            if (!response.ok) {
                console.error('Failed to fetch binary graph:', response.status);
                return;
            }
            binaryGraph = await response.json();
        }

        console.log('Received binary graph data:', binaryGraph);

        // Display the binary-level graph
        if (binaryGraph && binaryGraph.nodes) {
            // TODO: In the future, we could filter/highlight the focused binary
            displayDependencyGraph(binaryGraph);
        } else {
            console.error('Invalid binary graph data received:', binaryGraph);
        }
    } catch (error) {
        console.error('Error fetching binary graph:', error);
    }
}

// Zoom out one level in the graph hierarchy
function zoomOutOneLevel() {
    if (currentView === 'focused') {
        // We're in focused view, zoom out to package level
        console.log('Zooming out from focused view to package view');
        currentView = 'package';
        currentTarget = null;
        currentBinary = null;

        // Clear list selection
        document.querySelectorAll('.nav-item.selected').forEach(el => {
            el.classList.remove('selected');
        });

        if (packageGraph) {
            displayDependencyGraph(packageGraph);
        }
    } else if (currentView === 'file') {
        // We're at file level, zoom out to package level
        console.log('Zooming out from file view to package view');

        // Find and click the project node in the tree
        const projectNode = document.querySelector('.tree-node.project-node .tree-node-content');
        if (projectNode) {
            projectNode.click();
        } else {
            // Fallback: directly show package graph
            currentView = 'package';
            currentTarget = null;
            currentBinary = null;
            if (packageGraph) {
                displayDependencyGraph(packageGraph);
            }
        }
    } else if (currentView === 'binary') {
        // We're at binary level, zoom out to package level
        console.log('Zooming out from binary view to package view');

        const projectNode = document.querySelector('.tree-node.project-node .tree-node-content');
        if (projectNode) {
            projectNode.click();
        } else {
            // Fallback: directly show package graph
            currentView = 'package';
            currentTarget = null;
            currentBinary = null;
            if (packageGraph) {
                displayDependencyGraph(packageGraph);
            }
        }
    } else {
        // Already at package level, can't zoom out further
        console.log('Already at top level (package view)');
    }
}

// Show focused view for a target (all dependencies touching this target)
async function showFocusedTargetView(targetLabel) {
    try {
        console.log('Showing focused view for target:', targetLabel);

        // Update state
        currentView = 'focused';
        currentTarget = targetLabel;
        currentBinary = null;

        // Sync selection in the list
        syncListSelection(targetLabel);

        // Fetch focused graph data from API
        const encodedLabel = encodeURIComponent(targetLabel);
        const url = `/api/target/${encodedLabel}/focused`;
        console.log('Fetching from URL:', url);

        const response = await monitoredFetch(url);
        if (!response.ok) {
            console.error('Failed to fetch focused graph:', response.status, response.statusText);
            return;
        }

        const graphData = await response.json();
        console.log('Received focused graph data:', graphData);
        console.log('Nodes:', graphData.nodes?.length, 'Edges:', graphData.edges?.length);

        // Display the focused graph
        if (graphData && graphData.nodes) {
            displayDependencyGraph(graphData);
        } else {
            console.error('Invalid graph data received:', graphData);
        }
    } catch (error) {
        console.error('Error showing focused target view:', error);
    }
}

// Show file-level graph for a target
async function showFileGraphForTarget(targetLabel) {
    try {
        console.log('Fetching file graph for target:', targetLabel);

        // Encode the target label for URL (strip leading // for the path)
        const encodedLabel = targetLabel.startsWith('//') ? targetLabel.substring(2) : targetLabel;
        const url = `/api/target/${encodedLabel}/graph`;
        console.log('Fetching from URL:', url);

        const response = await monitoredFetch(url);

        if (!response.ok) {
            console.error('Failed to fetch target graph:', response.status, response.statusText);
            const text = await response.text();
            console.error('Response body:', text);
            return;
        }

        const graphData = await response.json();
        console.log('Received file graph data:', graphData);
        console.log('Nodes:', graphData.nodes?.length, 'Edges:', graphData.edges?.length);

        // Display the file-level graph
        if (graphData && graphData.nodes) {
            displayDependencyGraph(graphData);
        } else {
            console.error('Invalid graph data received:', graphData);
        }
    } catch (error) {
        console.error('Error fetching target graph:', error);
    }
}

// Select a binary and show its focused graph
function selectBinary(binaryLabel, itemElement) {
    console.log('Selected binary:', binaryLabel);

    // Update UI selection
    document.querySelectorAll('.nav-item').forEach(el => el.classList.remove('selected'));
    if (itemElement) itemElement.classList.add('selected');

    // Update state
    currentView = 'binary';
    currentBinary = binaryLabel;
    currentTarget = null;

    // Show binary-focused graph
    showBinaryFocusedGraph(binaryLabel);
}

// Sync list selection to match the given target label
function syncListSelection(targetLabel) {
    // Clear all selections
    document.querySelectorAll('.nav-item').forEach(el => el.classList.remove('selected'));

    // Find and select the matching item in the targets list
    const targetsItems = document.getElementById('targetsItems');
    if (targetsItems) {
        const items = targetsItems.querySelectorAll('.nav-item');
        for (const item of items) {
            if (item.textContent.includes(targetLabel)) {
                item.classList.add('selected');
                break;
            }
        }
    }
}

// Select a target and show its focused graph
function selectTarget(targetLabel, itemElement) {
    console.log('Selected target:', targetLabel);

    // Update UI selection
    document.querySelectorAll('.nav-item').forEach(el => el.classList.remove('selected'));
    if (itemElement) itemElement.classList.add('selected');

    // Update state
    currentView = 'focused';
    currentTarget = targetLabel;
    currentBinary = null;

    // Show focused view for this target
    showFocusedTargetView(targetLabel);
}

// Show binary-focused graph: internal targets/packages + external binaries
async function showBinaryFocusedGraph(binaryLabel) {
    try {
        console.log('Building binary-focused graph for:', binaryLabel);

        // Fetch full binary graph if not already loaded
        if (!binaryGraph) {
            const response = await monitoredFetch('/api/binaries/graph');
            if (!response.ok) {
                console.error('Failed to fetch binary graph');
                return;
            }
            binaryGraph = await response.json();
        }

        // Find the focused binary's info
        const focusedBinary = binaryData.find(b => b.label === binaryLabel);
        if (!focusedBinary) {
            console.error('Binary not found:', binaryLabel);
            return;
        }

        // Build graph showing:
        // 1. Internal targets/packages (from packageGraph)
        // 2. External binaries (as binary-level nodes)
        // 3. System libraries

        const graphData = buildBinaryFocusedGraphData(focusedBinary);
        displayDependencyGraph(graphData);

    } catch (error) {
        console.error('Error showing binary-focused graph:', error);
    }
}

// Build graph data for a focused binary
function buildBinaryFocusedGraphData(focusedBinary) {
    const graphData = {
        nodes: [],
        edges: []
    };

    console.log('Building focused graph for binary:', focusedBinary.label);
    console.log('Dynamic deps:', focusedBinary.dynamicDeps);
    console.log('Data deps:', focusedBinary.dataDeps);
    console.log('System libs:', focusedBinary.systemLibraries);

    // Add compound parent node for internal (static) dependencies
    graphData.nodes.push({
        id: 'internal-group',
        label: focusedBinary.label,
        type: 'target-group'
    });

    // Add the binary itself as a node inside the internal group
    graphData.nodes.push({
        id: focusedBinary.label,
        label: focusedBinary.label,
        type: focusedBinary.kind,
        parent: 'internal-group'
    });

    // Filter to only include cc_library targets this binary depends on (static deps)
    const internalTargetSet = new Set(focusedBinary.internalTargets || []);
    console.log('Internal targets for', focusedBinary.label, ':', Array.from(internalTargetSet));

    if (packagesCollapsed && packageGraph) {
        // Show as packages - but only packages that contain our internal targets
        const collapsedPkg = buildCollapsedPackageGraph(packageGraph);

        // Find which packages contain our internal targets
        const relevantPackages = new Set();
        internalTargetSet.forEach(targetLabel => {
            const packageName = targetLabel.split(':')[0];
            relevantPackages.add(packageName);
        });

        console.log('Relevant packages:', Array.from(relevantPackages));

        collapsedPkg.nodes.forEach(node => {
            if (relevantPackages.has(node.id)) {
                graphData.nodes.push({
                    ...node,
                    parent: 'internal-group'
                });
            }
        });

        // Add edges only between our relevant packages
        collapsedPkg.edges.forEach(edge => {
            if (relevantPackages.has(edge.source) && relevantPackages.has(edge.target)) {
                graphData.edges.push(edge);
            }
        });

        // Add edges from the binary to packages containing its direct dependencies
        if (focusedBinary.regularDeps) {
            focusedBinary.regularDeps.forEach(depLabel => {
                const packageName = depLabel.split(':')[0];
                if (relevantPackages.has(packageName)) {
                    graphData.edges.push({
                        source: focusedBinary.label,
                        target: packageName,
                        type: 'static',
                        symbols: []
                    });
                }
            });
        }
    } else {
        // Show individual targets - but only our internal (static) targets
        const allTargets = packageGraph ? packageGraph.nodes : [];

        // Collect all overlapping target labels
        const overlappingTargetSet = new Set();
        if (focusedBinary.overlappingDeps) {
            console.log('Overlapping deps for', focusedBinary.label, ':', focusedBinary.overlappingDeps);
            Object.values(focusedBinary.overlappingDeps).forEach(targets => {
                targets.forEach(target => overlappingTargetSet.add(target));
            });
            console.log('Overlapping target set:', Array.from(overlappingTargetSet));
        }

        allTargets.forEach(target => {
            const targetLabel = target.label || target.id;
            if (internalTargetSet.has(targetLabel)) {
                const isOverlapping = overlappingTargetSet.has(targetLabel);
                graphData.nodes.push({
                    ...target,
                    parent: 'internal-group',
                    hasOverlap: isOverlapping,
                    overlappingWith: isOverlapping ?
                        Object.keys(focusedBinary.overlappingDeps || {}).filter(sharedLib =>
                            focusedBinary.overlappingDeps[sharedLib].includes(targetLabel)
                        ) : []
                });
            }
        });

        // Add edges only between our internal targets
        if (packageGraph && packageGraph.edges) {
            packageGraph.edges.forEach(edge => {
                if (internalTargetSet.has(edge.source) && internalTargetSet.has(edge.target)) {
                    graphData.edges.push(edge);
                }
            });
        }

        // Add edges from the binary to its direct dependencies
        if (focusedBinary.regularDeps) {
            focusedBinary.regularDeps.forEach(depLabel => {
                if (internalTargetSet.has(depLabel)) {
                    const isOverlapping = overlappingTargetSet.has(depLabel);
                    graphData.edges.push({
                        source: focusedBinary.label,
                        target: depLabel,
                        type: 'static',
                        symbols: [],
                        isOverlapping: isOverlapping
                    });
                }
            });
        }
    }

    // Add external binaries this binary depends on
    if (focusedBinary.dynamicDeps) {
        focusedBinary.dynamicDeps.forEach(depLabel => {
            const depBinary = binaryData.find(b => b.label === depLabel);
            if (depBinary) {
                // Check if this dependency has overlapping symbols
                const hasOverlap = focusedBinary.overlappingDeps &&
                                   focusedBinary.overlappingDeps[depLabel] &&
                                   focusedBinary.overlappingDeps[depLabel].length > 0;

                graphData.nodes.push({
                    id: depLabel,
                    label: depLabel,
                    type: depBinary.kind, // cc_binary or cc_shared_library
                    external: true,
                    hasOverlap: hasOverlap,
                    overlappingTargets: hasOverlap ? focusedBinary.overlappingDeps[depLabel] : []
                });
            }
        });
    }

    if (focusedBinary.dataDeps) {
        focusedBinary.dataDeps.forEach(depLabel => {
            const depBinary = binaryData.find(b => b.label === depLabel);
            if (depBinary) {
                graphData.nodes.push({
                    id: depLabel,
                    label: depLabel,
                    type: depBinary.kind,
                    external: true
                });
            }
        });
    }

    // Add system libraries
    if (focusedBinary.systemLibraries) {
        focusedBinary.systemLibraries.forEach(sysLib => {
            graphData.nodes.push({
                id: 'system:' + sysLib,
                label: sysLib,
                type: 'system_library',
                external: true
            });
        });
    }

    // Add edges connecting to external dependencies
    // We need to find which internal target actually uses these external binaries
    // For now, create edges from main:test_app (the focused binary's main target)

    const mainTarget = focusedBinary.label; // e.g., "//main:test_app"

    // Check if this target exists in the graph
    const hasMainTarget = graphData.nodes.some(n => n.id === mainTarget || n.label === mainTarget);

    if (hasMainTarget) {
        // Add edges for dynamic dependencies
        if (focusedBinary.dynamicDeps) {
            focusedBinary.dynamicDeps.forEach(depLabel => {
                graphData.edges.push({
                    source: mainTarget,
                    target: depLabel,
                    type: 'dynamic_link',
                    linkage: 'dynamic',
                    symbols: []
                });
            });
        }

        // Add edges for data dependencies
        if (focusedBinary.dataDeps) {
            focusedBinary.dataDeps.forEach(depLabel => {
                graphData.edges.push({
                    source: mainTarget,
                    target: depLabel,
                    type: 'data_dependency',
                    linkage: 'data',
                    symbols: []
                });
            });
        }

        // Add edges for system libraries
        if (focusedBinary.systemLibraries) {
            focusedBinary.systemLibraries.forEach(sysLib => {
                graphData.edges.push({
                    source: mainTarget,
                    target: 'system:' + sysLib,
                    type: 'system_link',
                    linkage: 'system',
                    symbols: []
                });
            });
        }
    }

    return graphData;
}

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
            item.textContent = `${icon} ${binary.label}`;
            item.onclick = () => selectBinary(binary.label, item);
            binariesItems.appendChild(item);
        });
    }

    // Populate targets list
    const targetsItems = document.getElementById('targetsItems');
    if (targetsItems && data.graph && data.graph.nodes) {
        targetsItems.innerHTML = '';
        data.graph.nodes.forEach(node => {
            const item = document.createElement('div');
            item.className = 'nav-item';
            item.textContent = `📦 ${node.label}`;
            item.onclick = () => selectTarget(node.label, item);
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
