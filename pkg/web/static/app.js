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

// Load data when page loads
document.addEventListener('DOMContentLoaded', loadAnalysisData);

// Optionally refresh every 5 seconds (useful when file watching is added later)
// setInterval(loadAnalysisData, 5000);
