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
            if (node.type === 'target-group' && currentView === 'file') {
                const targetLabel = node.id.replace('parent-', '');
                if (targetLabel === currentTarget) {
                    nodeData.focused = true;
                }
            }

            return { data: nodeData };
        }),
        // Edges
        ...graphData.edges.map(edge => ({
            data: {
                source: edge.source,
                target: edge.target,
                type: edge.type,
                linkage: edge.linkage,
                symbols: edge.symbols || []
            }
        }))
    ];

    console.log('Creating new cytoscape instance with', elements.length, 'elements');

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
                selector: 'node[type = "source"]',
                style: {
                    'background-color': '#89d185',
                    'color': '#1e1e1e',
                    'border-color': '#6fb06b'
                }
            },
            {
                selector: 'node[type = "header"]',
                style: {
                    'background-color': '#4fc1ff',
                    'color': '#1e1e1e',
                    'border-color': '#3fa0d9'
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
                selector: 'node[hasOverlap]',
                style: {
                    'border-width': 3,
                    'border-color': '#f48771',
                    'border-style': 'dashed'
                }
            },
            {
                selector: 'node[type = "target-group"]',
                style: {
                    'shape': 'roundrectangle',
                    'background-color': '#2d2d30',
                    'background-opacity': 0.5,
                    'border-width': '2px',
                    'border-color': '#4a4a4e',
                    'border-style': 'dashed',
                    'label': 'data(label)',
                    'color': '#969696',
                    'text-valign': 'top',
                    'text-halign': 'center',
                    'font-size': '14px',
                    'font-weight': 'bold',
                    'padding': '20px'
                }
            },
            {
                selector: 'node[type = "target-group"][focused]',
                style: {
                    'border-width': '3px',
                    'border-color': '#ff8c00',
                    'border-style': 'dashed'
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
                selector: 'node:selected',
                style: {
                    'border-width': '3px',
                    'border-color': '#ff8c00'
                }
            }
        ],

        layout: {
            name: 'cose',
            directed: true,
            padding: 50,
            animate: false,
            nodeRepulsion: 8000,
            idealEdgeLength: 100,
            edgeElasticity: 100,
            nestingFactor: 1.2,
            gravity: 1,
            numIter: 1000,
            initialTemp: 200,
            coolingFactor: 0.95,
            minTemp: 1.0
        }
    });

    // Add tooltip for symbol edges
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

    // Show tooltip on edge hover
    cy.on('mouseover', 'edge[type = "symbol"]', function(evt) {
        const edge = evt.target;
        const symbols = edge.data('symbols');
        if (symbols && symbols.length > 0) {
            const symbolList = symbols.join('\n');
            tooltip.textContent = `Symbols (${symbols.length}):\n${symbolList}`;
            tooltip.style.display = 'block';
        }
    });

    cy.on('mousemove', 'edge[type = "symbol"]', function(evt) {
        tooltip.style.left = (evt.originalEvent.pageX + 10) + 'px';
        tooltip.style.top = (evt.originalEvent.pageY + 10) + 'px';
    });

    cy.on('mouseout', 'edge[type = "symbol"]', function(evt) {
        tooltip.style.display = 'none';
    });

    // Tooltip for nodes with overlapping dependencies (duplicate symbols warning)
    cy.on('mouseover', 'node[hasOverlap]', function(evt) {
        const node = evt.target;
        const overlappingTargets = node.data('overlappingTargets');
        if (overlappingTargets && overlappingTargets.length > 0) {
            const targetList = overlappingTargets.join('\n  ');
            tooltip.textContent = `⚠️ Duplicate symbols!\n\nBoth this binary and ${node.data('label')} link:\n  ${targetList}`;
            tooltip.style.display = 'block';
        }
    });

    cy.on('mousemove', 'node[hasOverlap]', function(evt) {
        tooltip.style.left = (evt.originalEvent.pageX + 10) + 'px';
        tooltip.style.top = (evt.originalEvent.pageY + 10) + 'px';
    });

    cy.on('mouseout', 'node[hasOverlap]', function(evt) {
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
            // At package level, clicking a target should zoom into it
            // Find and click the corresponding target in the tree
            const targetNodes = document.querySelectorAll('.tree-node[data-type="target"]');
            console.log('Looking for tree node with id:', nodeId);
            console.log('Found', targetNodes.length, 'target nodes in tree');

            for (const treeNode of targetNodes) {
                console.log('Checking tree node id:', treeNode.dataset.id);
                if (treeNode.dataset.id === nodeId) {
                    console.log('Match found! Clicking tree node');
                    const label = treeNode.querySelector('.tree-label');
                    if (label) {
                        label.click();
                        return;
                    }
                }
            }
            console.log('No matching tree node found for:', nodeId);
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

function displayCrossPackageDeps(deps) {
    const listEl = document.getElementById('crossPackageList');
    listEl.innerHTML = '';

    // Group by source package
    const grouped = {};
    deps.forEach(dep => {
        if (!grouped[dep.sourcePackage]) {
            grouped[dep.sourcePackage] = [];
        }
        grouped[dep.sourcePackage].push(dep);
    });

    // Display grouped dependencies
    Object.keys(grouped).sort().forEach(sourcePackage => {
        const packageDeps = grouped[sourcePackage];

        const packageDiv = document.createElement('div');
        packageDiv.style.marginBottom = '15px';

        const headerDiv = document.createElement('div');
        headerDiv.style.fontWeight = 'bold';
        headerDiv.style.marginBottom = '5px';
        headerDiv.style.color = '#0d6efd';
        headerDiv.textContent = `${sourcePackage} dependencies:`;
        packageDiv.appendChild(headerDiv);

        packageDeps.forEach(dep => {
            const depDiv = document.createElement('div');
            depDiv.className = 'file-item';
            depDiv.style.marginLeft = '20px';

            const pathDiv = document.createElement('div');
            pathDiv.className = 'file-path';
            pathDiv.textContent = `${dep.sourceFile} → ${dep.targetFile}`;

            const targetDiv = document.createElement('div');
            targetDiv.className = 'file-package';
            targetDiv.textContent = `Depends on: ${dep.targetPackage}`;

            depDiv.appendChild(pathDiv);
            depDiv.appendChild(targetDiv);
            packageDiv.appendChild(depDiv);
        });

        listEl.appendChild(packageDiv);
    });
}

function displayFileCycles(cycles) {
    const listEl = document.getElementById('cyclesList');
    listEl.innerHTML = '';

    cycles.forEach((cycle, index) => {
        const cycleDiv = document.createElement('div');
        cycleDiv.style.marginBottom = '20px';

        const headerDiv = document.createElement('div');
        headerDiv.style.fontWeight = 'bold';
        headerDiv.style.marginBottom = '10px';
        headerDiv.style.color = '#dc3545';
        headerDiv.textContent = `Cycle ${index + 1} (${cycle.files.length} files):`;
        cycleDiv.appendChild(headerDiv);

        const cycleItemDiv = document.createElement('div');
        cycleItemDiv.className = 'file-item';
        cycleItemDiv.style.background = '#ffe6e6';
        cycleItemDiv.style.borderLeftColor = '#dc3545';

        // Display the cycle as a chain
        const pathDiv = document.createElement('div');
        pathDiv.className = 'file-path';
        pathDiv.style.fontFamily = 'Courier New, monospace';
        pathDiv.style.fontSize = '0.9em';
        pathDiv.style.lineHeight = '1.6';

        // Show cycle as: file1 → file2 → file3 → file1
        const cycleChain = cycle.files.join(' → ') + ' → ' + cycle.files[0];
        pathDiv.textContent = cycleChain;

        cycleItemDiv.appendChild(pathDiv);
        cycleDiv.appendChild(cycleItemDiv);
        listEl.appendChild(cycleDiv);
    });
}

let refreshInterval;
let lastDataHash = '';

// Hash the data to detect changes
function hashData(data) {
    return JSON.stringify({
        total: data.totalFiles,
        covered: data.coveredFiles,
        uncoveredCount: data.uncoveredFiles ? data.uncoveredFiles.length : 0,
        hasGraph: !!data.graph,
        hasCrossDeps: !!data.crossPackageDeps && data.crossPackageDeps.length > 0,
        hasCycles: !!data.fileCycles && data.fileCycles.length > 0
    });
}

let hasShownGraph = false;
let hasShownCrossDeps = false;
let hasShownCycles = false;
let hasShownCoverageResult = false;
let graphSectionShown = false;

// Load data and check if analysis is complete
async function loadAndCheckComplete() {
    try {
        const response = await fetch('/api/analysis');
        if (!response.ok) return;

        const data = await response.json();
        const currentHash = hashData(data);

        // Only update if data changed
        if (currentHash !== lastDataHash) {
            lastDataHash = currentHash;

            // Use backend's analysisStep to drive UI progress
            const step = data.analysisStep || 0;
            console.log('Analysis step:', step, 'Flags:', {graphSectionShown, hasShownGraph, hasShownCrossDeps});

            // Step 1: Coverage complete
            if (step >= 1 && !graphSectionShown) {
                console.log('Step 1 -> 2');
                updateLoadingProgress(1, 2);
                document.getElementById('graphSection').style.display = 'block';
                graphSectionShown = true;
            }

            // Step 2: Graph complete
            if (step >= 2 && data.graph && !hasShownGraph) {
                console.log('Step 2 -> 3');

                // Store the analysis data and package graph
                analysisData = data;
                packageGraph = data.graph;

                // Fetch binary data
                try {
                    const binariesResponse = await fetch('/api/binaries');
                    if (binariesResponse.ok) {
                        binaryData = await binariesResponse.json();
                        console.log('Loaded binary data:', binaryData);
                    }
                } catch (e) {
                    console.error('Failed to fetch binary data:', e);
                }

                displayDependencyGraph(data.graph);
                const graphLoading = document.getElementById('graphLoading');
                if (graphLoading) {
                    graphLoading.style.display = 'none';
                }
                updateLoadingProgress(2, 3);
                hasShownGraph = true;

                // Populate tree browser (after binary data is loaded)
                if (data.graph.nodes) {
                    populateTreeBrowser(data);
                }

                console.log('Binary list populated, binaryData count:', binaryData ? binaryData.length : 0);
            }

            // Step 4: All complete
            if (step >= 4 && !hasShownCrossDeps) {
                console.log('Step 4 - completing all');
                // Display cross-package deps if any
                if (data.crossPackageDeps && data.crossPackageDeps.length > 0) {
                    displayCrossPackageDeps(data.crossPackageDeps);
                    document.getElementById('crossPackageSection').style.display = 'block';
                }

                // Display cycles if any
                if (data.fileCycles && data.fileCycles.length > 0) {
                    displayFileCycles(data.fileCycles);
                    document.getElementById('cyclesSection').style.display = 'block';
                }

                // Complete steps 3 and 4, then hide overlay
                updateLoadingProgress(3, 4);
                updateLoadingProgress(4, null);

                setTimeout(() => {
                    console.log('Hiding overlay');
                    hideLoadingOverlay();
                    if (refreshInterval) {
                        clearInterval(refreshInterval);
                        refreshInterval = null;
                    }
                }, 1000);

                hasShownCrossDeps = true;
                hasShownCycles = true;
            }
        }
    } catch (e) {
        console.error('Error checking for updates:', e);
    }
}

// Load data when page loads
document.addEventListener('DOMContentLoaded', function() {
    // Activate step 1 immediately
    updateLoadingProgress(null, 1);

    // Start polling immediately
    loadAndCheckComplete();

    // Poll every 1 second for updates during analysis
    refreshInterval = setInterval(loadAndCheckComplete, 1000);
});

// Modal functions for showing target details
async function showTargetDetails(targetLabel) {
    try {
        // Encode the target label for URL
        const encodedLabel = encodeURIComponent(targetLabel);
        const response = await fetch(`/api/target/${encodedLabel}`);
        
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

    // Package collapse toggle handler
    const packageCollapseToggle = document.getElementById('packageCollapseToggle');
    if (packageCollapseToggle) {
        packageCollapseToggle.addEventListener('change', function() {
            packagesCollapsed = this.checked;
            console.log('Package collapse toggled:', packagesCollapsed);

            // Rebuild the current view based on mode
            if (currentView === 'package' && packageGraph) {
                // Package level view
                if (packagesCollapsed) {
                    displayDependencyGraph(buildCollapsedPackageGraph(packageGraph));
                } else {
                    displayDependencyGraph(packageGraph);
                }
            } else if (currentView === 'binary' && currentBinary && binaryData) {
                // Binary-focused view - rebuild with new collapse state
                const focusedBinary = binaryData.find(b => b.label === currentBinary);
                if (focusedBinary) {
                    const graphData = buildBinaryFocusedGraphData(focusedBinary);
                    displayDependencyGraph(graphData);
                }
            }
        });
    }
});

// Build a collapsed version of the package graph where targets are grouped by package
function buildCollapsedPackageGraph(graph) {
    const packages = new Map(); // packageName -> { targets: [], edges: [] }
    const packageEdges = new Map(); // "srcPkg->dstPkg" -> { type, symbols, sources[], targets[] }

    // Group targets by package
    graph.nodes.forEach(node => {
        const packageName = node.label.split(':')[0]; // Extract "//path/to/package" from "//path/to/package:target"
        if (!packages.has(packageName)) {
            packages.set(packageName, { targets: [], edges: [] });
        }
        packages.get(packageName).targets.push(node);
    });

    // Group edges by package-to-package connections
    graph.edges.forEach(edge => {
        const srcPackage = edge.source.split(':')[0];
        const dstPackage = edge.target.split(':')[0];

        // Skip intra-package edges
        if (srcPackage === dstPackage) {
            return;
        }

        const edgeKey = `${srcPackage}->${dstPackage}`;
        if (!packageEdges.has(edgeKey)) {
            packageEdges.set(edgeKey, {
                source: srcPackage,
                target: dstPackage,
                type: edge.type,
                linkage: edge.linkage,
                symbols: [],
                sources: new Set(),
                targets: new Set()
            });
        }

        const pkgEdge = packageEdges.get(edgeKey);
        if (edge.symbols) {
            pkgEdge.symbols.push(...edge.symbols);
        }
        pkgEdge.sources.add(edge.source);
        pkgEdge.targets.add(edge.target);
    });

    // Build collapsed graph
    const collapsedGraph = {
        nodes: [],
        edges: []
    };

    // Create package nodes
    packages.forEach((info, packageName) => {
        collapsedGraph.nodes.push({
            id: packageName,
            label: packageName,
            type: 'package',
            targetCount: info.targets.length
        });
    });

    // Create package edges
    packageEdges.forEach((edgeInfo, key) => {
        collapsedGraph.edges.push({
            source: edgeInfo.source,
            target: edgeInfo.target,
            type: edgeInfo.type,
            linkage: edgeInfo.linkage,
            symbols: edgeInfo.symbols,
            sourceTargets: Array.from(edgeInfo.sources),
            targetTargets: Array.from(edgeInfo.targets)
        });
    });

    return collapsedGraph;
}

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
let currentView = 'package'; // Track current view: 'package', 'file', or 'binary'
let currentTarget = null; // Track which target we're viewing at file level
let currentBinary = null; // Track which binary we're viewing at binary level
let packagesCollapsed = false; // Track if packages are collapsed

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
                    const response = await fetch(`/api/target/${encodedLabel}`);
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
            const response = await fetch('/api/binaries/graph');
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
    if (currentView === 'file') {
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

// Show file-level graph for a target
async function showFileGraphForTarget(targetLabel) {
    try {
        console.log('Fetching file graph for target:', targetLabel);

        // Encode the target label for URL (strip leading // for the path)
        const encodedLabel = targetLabel.startsWith('//') ? targetLabel.substring(2) : targetLabel;
        const url = `/api/target/${encodedLabel}/graph`;
        console.log('Fetching from URL:', url);

        const response = await fetch(url);

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

// Select a target and show its file graph
function selectTarget(targetLabel, itemElement) {
    console.log('Selected target:', targetLabel);

    // Update UI selection
    document.querySelectorAll('.nav-item').forEach(el => el.classList.remove('selected'));
    if (itemElement) itemElement.classList.add('selected');

    // Update state
    currentView = 'file';
    currentTarget = targetLabel;
    currentBinary = null;

    // Show file-level graph for this target
    showFileGraphForTarget(targetLabel);
}

// Show binary-focused graph: internal targets/packages + external binaries
async function showBinaryFocusedGraph(binaryLabel) {
    try {
        console.log('Building binary-focused graph for:', binaryLabel);

        // Fetch full binary graph if not already loaded
        if (!binaryGraph) {
            const response = await fetch('/api/binaries/graph');
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

        allTargets.forEach(target => {
            const targetLabel = target.label || target.id;
            if (internalTargetSet.has(targetLabel)) {
                graphData.nodes.push({
                    ...target,
                    parent: 'internal-group'
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
                    graphData.edges.push({
                        source: focusedBinary.label,
                        target: depLabel,
                        type: 'static',
                        symbols: []
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
