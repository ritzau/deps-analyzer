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
                target: edge.target
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
                selector: 'node[type = "target-group"][focused = true]',
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

                displayDependencyGraph(data.graph);
                const graphLoading = document.getElementById('graphLoading');
                if (graphLoading) {
                    graphLoading.style.display = 'none';
                }
                updateLoadingProgress(2, 3);
                hasShownGraph = true;

                // Populate tree browser
                if (data.graph.nodes) {
                    populateTreeBrowser(data);
                }
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
});

// ============================================================================
// Tree Browser Implementation
// ============================================================================

let treeData = null;
let selectedNode = null;
let analysisData = null; // Store full analysis data
let packageGraph = null; // Store the original package-level graph
let cy = null; // Store the Cytoscape instance
let currentView = 'package'; // Track current view: 'package' or 'file'
let currentTarget = null; // Track which target we're viewing at file level

// Build tree structure from analysis data
function buildTreeData(data) {
    const tree = {
        targets: [],
        uncoveredFiles: data.uncoveredFiles || []
    };

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
    if (type === 'target') {
        toggle.textContent = '▶';
        node.classList.add('collapsed');
    } else {
        toggle.classList.add('empty');
    }
    content.appendChild(toggle);

    // Icon and label
    const label = document.createElement('span');
    label.className = 'tree-label';

    if (type === 'target') {
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

    // Add children container for targets
    if (type === 'target') {
        const children = document.createElement('div');
        children.className = 'tree-children';
        node.appendChild(children);

        // Click on arrow to toggle expansion only
        toggle.addEventListener('click', async (e) => {
            console.log('Toggle arrow clicked');
            e.stopPropagation();
            try {
                await toggleExpansion(node, item);
            } catch (error) {
                console.error('Error in toggleExpansion:', error);
            }
        });

        // Click on label to always select/zoom (regardless of expansion)
        label.addEventListener('click', async (e) => {
            console.log('Target label clicked:', item.label);
            e.stopPropagation();
            try {
                await selectTreeNode(node, item, 'target');
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
async function toggleExpansion(node, item) {
    const toggle = node.querySelector('.tree-toggle');
    const children = node.querySelector('.tree-children');

    if (node.classList.contains('collapsed')) {
        // Expand: load files if not already loaded
        node.classList.remove('collapsed');
        toggle.textContent = '▼';

        if (children.children.length === 0 && item.label) {
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
        if (packageGraph) {
            displayDependencyGraph(packageGraph);
        }
    } else if (type === 'target') {
        // Show file-level graph for this target
        console.log('Selected target:', item.label);
        currentView = 'file';
        currentTarget = item.label;
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

// Populate the tree browser
function populateTreeBrowser(data) {
    console.log('Populating tree browser with data:', data);
    const treeContent = document.getElementById('treeContent');
    treeContent.innerHTML = '';

    treeData = buildTreeData(data);

    // Project Root Node
    const projectNode = document.createElement('div');
    projectNode.className = 'tree-node project-node';
    projectNode.dataset.type = 'project';
    projectNode.dataset.id = data.workspace;

    const projectContent = document.createElement('div');
    projectContent.className = 'tree-node-content';

    const projectLabel = document.createElement('span');
    projectLabel.className = 'tree-label';
    projectLabel.textContent = `🏠 ${data.workspace.split('/').pop() || 'Project'}`;
    projectContent.appendChild(projectLabel);
    projectNode.appendChild(projectContent);

    // Click handler for project - show package graph
    projectContent.addEventListener('click', (e) => {
        console.log('Project node clicked');
        e.stopPropagation();
        selectTreeNode(projectNode, { workspace: data.workspace }, 'project');
    });

    treeContent.appendChild(projectNode);

    // Targets Section
    const targetsSection = document.createElement('div');
    targetsSection.className = 'tree-section';

    const targetsTitle = document.createElement('div');
    targetsTitle.className = 'tree-section-title';
    targetsTitle.innerHTML = `📦 Targets <span class="tree-count">${treeData.targets.length}</span>`;
    targetsSection.appendChild(targetsTitle);

    treeData.targets.forEach(target => {
        const targetNode = createTreeNode(target, 'target');
        targetsSection.appendChild(targetNode);
    });

    treeContent.appendChild(targetsSection);

    // Uncovered Files Section
    if (treeData.uncoveredFiles.length > 0) {
        const uncoveredSection = document.createElement('div');
        uncoveredSection.className = 'tree-section';

        const uncoveredTitle = document.createElement('div');
        uncoveredTitle.className = 'tree-section-title';
        uncoveredTitle.innerHTML = `⚠️ Uncovered <span class="tree-count">${treeData.uncoveredFiles.length}</span>`;
        uncoveredSection.appendChild(uncoveredTitle);

        treeData.uncoveredFiles.forEach(file => {
            const fileNode = createTreeNode(file, 'uncovered');
            uncoveredSection.appendChild(fileNode);
        });

        treeContent.appendChild(uncoveredSection);
    }
}
