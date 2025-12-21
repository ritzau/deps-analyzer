// Fetch and display analysis data
async function loadAnalysisData() {
    const loadingEl = document.getElementById('loading');
    const errorEl = document.getElementById('error');
    const contentEl = document.getElementById('content');

    try {
        const response = await fetch('/api/analysis');
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json();

        // Hide loading, show content
        loadingEl.style.display = 'none';
        contentEl.style.display = 'block';

        // Update summary stats
        document.getElementById('workspace').textContent = data.workspace;
        document.getElementById('totalFiles').textContent = data.totalFiles;
        document.getElementById('coveredFiles').textContent = data.coveredFiles;
        document.getElementById('uncoveredCount').textContent = data.uncoveredFiles.length;

        // Update coverage percentage
        const percentage = data.coveragePercent;
        document.getElementById('coveragePercent').textContent = percentage.toFixed(0) + '%';

        // Update progress bar
        const progressFill = document.getElementById('progressFill');
        progressFill.style.width = percentage + '%';

        // Color code the progress bar
        if (percentage === 100) {
            progressFill.classList.add('success');
        } else if (percentage < 80) {
            progressFill.classList.add('warning');
        }

        // Show uncovered files or success message
        if (data.uncoveredFiles && data.uncoveredFiles.length > 0) {
            displayUncoveredFiles(data.uncoveredFiles);
            document.getElementById('uncoveredSection').style.display = 'block';
        } else {
            document.getElementById('successMessage').style.display = 'block';
        }

        // Show dependency graph if available
        if (data.graph && data.graph.nodes && data.graph.nodes.length > 0) {
            displayDependencyGraph(data.graph);
            document.getElementById('graphSection').style.display = 'block';
        }

    } catch (error) {
        console.error('Error loading analysis data:', error);
        loadingEl.style.display = 'none';
        errorEl.textContent = 'Failed to load analysis data: ' + error.message;
        errorEl.style.display = 'block';
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
                    'background-color': '#3b82f6',
                    'label': 'data(label)',
                    'color': '#333',
                    'text-valign': 'center',
                    'text-halign': 'center',
                    'font-size': '12px',
                    'width': '60px',
                    'height': '60px',
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
            name: 'breadthfirst',
            directed: true,
            spacingFactor: 1.5,
            padding: 30,
            avoidOverlap: true
        }
    });

    // Add interactivity
    cy.on('tap', 'node', function(evt) {
        const node = evt.target;
        console.log('Tapped node:', node.data('label'));
    });

    // Zoom to fit on load
    cy.fit(50);
}

// Load data when page loads
document.addEventListener('DOMContentLoaded', loadAnalysisData);

// Optionally refresh every 5 seconds (useful when file watching is added later)
// setInterval(loadAnalysisData, 5000);
