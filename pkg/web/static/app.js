// Update loading message
function updateLoadingMessage(message) {
    const loadingMsg = document.getElementById('loadingMessage');
    if (loadingMsg) {
        loadingMsg.textContent = message;
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
    const cy = cytoscape({
        container: document.getElementById('cy'),

        elements: [
            // Nodes
            ...graphData.nodes.map(node => ({
                data: {
                    id: node.id,
                    label: node.label,
                    type: node.type
                }
            })),
            // Edges
            ...graphData.edges.map(edge => ({
                data: {
                    source: edge.source,
                    target: edge.target
                }
            }))
        ],

        style: [
            {
                selector: 'node',
                style: {
                    'shape': 'roundrectangle',
                    'background-color': '#3b82f6',
                    'label': 'data(label)',
                    'color': 'white',
                    'text-valign': 'center',
                    'text-halign': 'center',
                    'font-size': '11px',
                    'font-weight': 'bold',
                    'text-wrap': 'wrap',
                    'text-max-width': '120px',
                    'width': 'label',
                    'height': 'label',
                    'padding': '12px',
                    'border-width': '2px',
                    'border-color': '#1e40af'
                }
            },
            {
                selector: 'node[type*="binary"]',
                style: {
                    'background-color': '#10b981',
                    'border-color': '#059669'
                }
            },
            {
                selector: 'edge',
                style: {
                    'width': 2,
                    'line-color': '#9ca3af',
                    'target-arrow-color': '#9ca3af',
                    'target-arrow-shape': 'triangle',
                    'curve-style': 'bezier',
                    'arrow-scale': 1.5
                }
            },
            {
                selector: 'node:selected',
                style: {
                    'border-width': '4px',
                    'border-color': '#fbbf24'
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

    // Add interactivity
    cy.on('tap', 'node', function(evt) {
        const node = evt.target;
        console.log('Tapped node:', node.data('label'));
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

            // Update stats when we have coverage data
            if (data.totalFiles > 0) {
                updateLoadingMessage('[1/4] Coverage analysis complete...');

                document.getElementById('workspace').textContent = data.workspace;
                document.getElementById('totalFiles').textContent = data.totalFiles;
                document.getElementById('coveredFiles').textContent = data.coveredFiles;
                document.getElementById('uncoveredCount').textContent = data.uncoveredFiles.length;

                const percentage = data.coveragePercent;
                document.getElementById('coveragePercent').textContent = percentage.toFixed(0) + '%';

                const progressFill = document.getElementById('progressFill');
                progressFill.style.width = percentage + '%';

                // Color code the progress bar
                progressFill.className = 'progress-fill'; // Reset classes
                if (percentage === 100) {
                    progressFill.classList.add('success');
                } else if (percentage < 80) {
                    progressFill.classList.add('warning');
                }

                // Show uncovered files or success message ONLY when we have data
                if (!hasShownCoverageResult) {
                    if (data.uncoveredFiles && data.uncoveredFiles.length > 0) {
                        displayUncoveredFiles(data.uncoveredFiles);
                        document.getElementById('uncoveredSection').style.display = 'block';
                        document.getElementById('successMessage').style.display = 'none';
                    } else if (data.totalFiles > 0) {
                        document.getElementById('uncoveredSection').style.display = 'none';
                        document.getElementById('successMessage').style.display = 'block';
                    }
                    hasShownCoverageResult = true;
                }

                // Show graph section with loading spinner once coverage is complete
                if (!graphSectionShown) {
                    updateLoadingMessage('[2/4] Building dependency graph...');
                    document.getElementById('graphSection').style.display = 'block';
                    graphSectionShown = true;
                }
            }

            // Display graph when it becomes available (hide loading spinner)
            if (data.graph && !hasShownGraph) {
                displayDependencyGraph(data.graph);
                // Hide the graph loading spinner
                const graphLoading = document.getElementById('graphLoading');
                if (graphLoading) {
                    graphLoading.style.display = 'none';
                }
                hasShownGraph = true;
            }

            // Show cross-package deps when they become available
            if (data.crossPackageDeps && data.crossPackageDeps.length > 0 && !hasShownCrossDeps) {
                updateLoadingMessage('[3/4] Analyzing file dependencies...');
                displayCrossPackageDeps(data.crossPackageDeps);
                document.getElementById('crossPackageSection').style.display = 'block';
                hasShownCrossDeps = true;
            }

            // Show cycles when they become available
            if (data.fileCycles && data.fileCycles.length > 0 && !hasShownCycles) {
                displayFileCycles(data.fileCycles);
                document.getElementById('cyclesSection').style.display = 'block';
                hasShownCycles = true;
            }

            // Hide loading overlay when we have coverage data
            if (data.totalFiles > 0) {
                hideLoadingOverlay();
            }

            // Stop polling if we have all the data (cross-package deps and cycles are optional)
            const hasCrossData = hasShownCrossDeps || (data.crossPackageDeps && data.crossPackageDeps.length === 0);
            const hasCycleData = hasShownCycles || (data.fileCycles && data.fileCycles.length === 0);
            if (data.totalFiles > 0 && hasShownGraph && hasCrossData && hasCycleData) {
                updateLoadingMessage('[4/4] Analysis complete!');
                console.log('Analysis complete, stopping auto-refresh');
                if (refreshInterval) {
                    clearInterval(refreshInterval);
                    refreshInterval = null;
                }
            }
        }
    } catch (e) {
        console.error('Error checking for updates:', e);
    }
}

// Load data when page loads
document.addEventListener('DOMContentLoaded', function() {
    // Start polling immediately
    loadAndCheckComplete();

    // Poll every 1 second for updates during analysis
    refreshInterval = setInterval(loadAndCheckComplete, 1000);
});
