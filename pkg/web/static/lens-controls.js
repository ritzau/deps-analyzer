/**
 * Lens Controls
 *
 * Wires up UI controls to the view state manager.
 * Handles:
 * - Tab switching
 * - Focus mode toggle
 * - Reset controls
 * - Lens configuration updates
 */

/**
 * Initialize all lens controls
 * Call this after DOM is loaded
 */
function initializeLensControls() {
  setupTabSwitching();
  setupFocusModeToggle();
  setupResetControls();
  setupDefaultLensControls();
  setupFocusLensControls();
}

/**
 * Set up tab switching (Tree | Default | Focus)
 */
function setupTabSwitching() {
  document.querySelectorAll('.tab-button').forEach(button => {
    button.addEventListener('click', () => {
      // Update active button
      document.querySelectorAll('.tab-button').forEach(b => b.classList.remove('active'));
      button.classList.add('active');

      // Update active pane
      const tabName = button.getAttribute('data-tab');
      document.querySelectorAll('.tab-pane').forEach(pane => pane.classList.remove('active'));
      const tabPane = document.getElementById(tabName + 'Tab');
      if (tabPane) {
        tabPane.classList.add('active');
      }

      viewStateManager.setActiveTab(tabName);
    });
  });
}

/**
 * Set up focus mode toggle (single vs multi-select)
 */
function setupFocusModeToggle() {
  document.querySelectorAll('input[name="focusMode"]').forEach(radio => {
    radio.addEventListener('change', (e) => {
      viewStateManager.setFocusMode(e.target.value);
    });
  });
}

/**
 * Set up reset controls (Clear Focus, Reset Manual, Reset All)
 */
function setupResetControls() {
  const resetFocusBtn = document.getElementById('resetFocus');
  if (resetFocusBtn) {
    resetFocusBtn.addEventListener('click', () => {
      viewStateManager.clearFocus();
    });
  }

  const resetManualBtn = document.getElementById('resetManual');
  if (resetManualBtn) {
    resetManualBtn.addEventListener('click', () => {
      viewStateManager.resetManualOverrides();
    });
  }

  const resetAllBtn = document.getElementById('resetAll');
  if (resetAllBtn) {
    resetAllBtn.addEventListener('click', () => {
      viewStateManager.resetAll();
    });
  }
}

/**
 * Set up default lens configuration controls
 */
function setupDefaultLensControls() {
  // Base set type selector
  const baseSetType = document.getElementById('baseSetType');
  if (baseSetType) {
    baseSetType.addEventListener('change', (e) => {
      const type = e.target.value;
      const binarySelector = document.getElementById('binarySelector');

      // Show/hide binary selector based on type
      if (binarySelector) {
        if (type === 'reachable-from-binary') {
          binarySelector.style.display = 'block';
        } else {
          binarySelector.style.display = 'none';
        }
      }

      // Update lens configuration
      console.log('[LensControls] Base set type changed to:', type);
      const currentLens = cloneLens(viewStateManager.getState().defaultLens);
      currentLens.baseSet.type = type;

      if (type === 'reachable-from-binary') {
        const binarySelect = document.getElementById('baseSetBinary');
        if (binarySelect) {
          currentLens.baseSet.binaryLabel = binarySelect.value;
        }
      }

      console.log('[LensControls] Updating default lens with new base set');
      viewStateManager.updateDefaultLens(currentLens);
    });
  }

  // Global filters
  const filterIds = ['hideExternal', 'hideUncovered', 'hideSystemLibs'];
  filterIds.forEach(id => {
    const checkbox = document.getElementById(id);
    if (checkbox) {
      checkbox.addEventListener('change', () => {
        const currentLens = cloneLens(viewStateManager.getState().defaultLens);
        currentLens.globalFilters.hideExternal = document.getElementById('hideExternal')?.checked || false;
        currentLens.globalFilters.hideUncovered = document.getElementById('hideUncovered')?.checked || false;
        currentLens.globalFilters.hideSystemLibs = document.getElementById('hideSystemLibs')?.checked || false;
        viewStateManager.updateDefaultLens(currentLens);
      });
    }
  });

  // Edge type checkboxes
  const edgeTypeIds = ['showStatic', 'showDynamic', 'showData', 'showCompile', 'showSymbol'];
  edgeTypeIds.forEach(id => {
    const checkbox = document.getElementById(id);
    if (checkbox) {
      checkbox.addEventListener('change', () => {
        console.log('[LensControls] Edge type checkbox changed:', id);
        const currentLens = cloneLens(viewStateManager.getState().defaultLens);
        const types = new Set();

        if (document.getElementById('showStatic')?.checked) types.add('static');
        if (document.getElementById('showDynamic')?.checked) types.add('dynamic');
        if (document.getElementById('showData')?.checked) types.add('data');
        if (document.getElementById('showCompile')?.checked) types.add('compile');
        if (document.getElementById('showSymbol')?.checked) types.add('symbol');

        // Always keep system_link
        types.add('system_link');

        console.log('[LensControls] New edge types:', Array.from(types));
        currentLens.edgeRules.types = types;
        viewStateManager.updateDefaultLens(currentLens);
      });
    }
  });

  // Collapse edge types checkbox
  const collapseEdgeTypesCheckbox = document.getElementById('collapseEdgeTypes');
  if (collapseEdgeTypesCheckbox) {
    collapseEdgeTypesCheckbox.addEventListener('change', () => {
      console.log('[LensControls] Collapse edge types changed:', collapseEdgeTypesCheckbox.checked);
      const currentLens = cloneLens(viewStateManager.getState().defaultLens);
      currentLens.edgeRules.collapseEdgeTypes = collapseEdgeTypesCheckbox.checked;
      viewStateManager.updateDefaultLens(currentLens);
    });
  }

  // Collapse level radio buttons
  const collapseLevelRadios = document.querySelectorAll('input[name="collapseLevel"]');
  collapseLevelRadios.forEach(radio => {
    radio.addEventListener('change', (e) => {
      if (e.target.checked) {
        const level = parseInt(e.target.value);
        console.log('[LensControls] Collapse level changed to:', level);
        const currentLens = cloneLens(viewStateManager.getState().defaultLens);

        // Update the collapse level in the distance rule
        const rule = currentLens.distanceRules[0];
        if (rule) {
          rule.collapseLevel = level;

          // Also update file visibility to match
          if (level >= 3) {
            rule.nodeVisibility.fileTypes = ['all'];
            rule.nodeVisibility.showUncovered = true;
          } else {
            rule.nodeVisibility.fileTypes = ['none'];
            rule.nodeVisibility.showUncovered = false;
          }
        }

        // Clear all manual fold/unfold overrides since they were based on the previous collapse level
        viewStateManager.resetManualOverrides();
        console.log('[LensControls] Cleared manual overrides due to collapse level change');

        viewStateManager.updateDefaultLens(currentLens);
      }
    });
  });
}

/**
 * Set up focus lens configuration controls
 */
function setupFocusLensControls() {
  // Distance 0 (focused nodes) file visibility
  const focusD0Files = document.getElementById('focusD0Files');
  if (focusD0Files) {
    focusD0Files.addEventListener('change', (e) => {
      const currentLens = cloneLens(viewStateManager.getState().focusLens);

      // Find distance 0 rule
      const rule = currentLens.distanceRules.find(r => r.distance === 0);
      if (rule) {
        if (e.target.value === 'all') {
          rule.nodeVisibility.fileTypes = ['all'];
          rule.collapseLevel = 3;  // Show files
        } else {
          rule.nodeVisibility.fileTypes = ['none'];
          rule.collapseLevel = 2;  // Hide files
        }
      }

      viewStateManager.updateFocusLens(currentLens);
    });
  }

  // Distance 1 (neighbors) file visibility
  const focusD1Files = document.getElementById('focusD1Files');
  if (focusD1Files) {
    focusD1Files.addEventListener('change', (e) => {
      const currentLens = cloneLens(viewStateManager.getState().focusLens);

      // Find distance 1 rule
      const rule = currentLens.distanceRules.find(r => r.distance === 1);
      if (rule) {
        if (e.target.value === 'all') {
          rule.nodeVisibility.fileTypes = ['all'];
          rule.collapseLevel = 3;  // Show files
        } else {
          rule.nodeVisibility.fileTypes = ['none'];
          rule.collapseLevel = 2;  // Hide files
        }
      }

      viewStateManager.updateFocusLens(currentLens);
    });
  }

  // Distance infinite (rest of graph) visibility
  const focusInfiniteView = document.getElementById('focusInfiniteView');
  if (focusInfiniteView) {
    focusInfiniteView.addEventListener('change', (e) => {
      const currentLens = cloneLens(viewStateManager.getState().focusLens);

      // Find infinite distance rule
      const rule = currentLens.distanceRules.find(r => r.distance === 'infinite');
      if (rule) {
        if (e.target.value === 'hide') {
          // Hide completely
          rule.nodeVisibility.targetTypes = [];
        } else if (e.target.value === 'collapsed') {
          // Show collapsed
          rule.nodeVisibility.targetTypes = ['cc_binary', 'cc_shared_library', 'cc_library'];
          rule.collapseLevel = 1;  // Package level
        } else {
          // Same as default
          // Copy from default lens
          const defaultRule = viewStateManager.getState().defaultLens.distanceRules[0];
          if (defaultRule) {
            rule.nodeVisibility = { ...defaultRule.nodeVisibility };
            rule.collapseLevel = defaultRule.collapseLevel;
          }
        }
      }

      viewStateManager.updateFocusLens(currentLens);
    });
  }
}

/**
 * Populate binary selector dropdown
 * Call this when binary data is loaded
 *
 * @param {Array} binaries - Array of binary info objects
 */
function populateBinarySelector(binaries) {
  const selector = document.getElementById('baseSetBinary');
  if (!selector) return;

  selector.innerHTML = '';
  binaries.forEach(binary => {
    const option = document.createElement('option');
    option.value = binary.label;
    option.textContent = simplifyLabel(binary.label);
    selector.appendChild(option);
  });

  // Add change listener
  selector.addEventListener('change', (e) => {
    const currentLens = cloneLens(viewStateManager.getState().defaultLens);
    currentLens.baseSet.binaryLabel = e.target.value;
    viewStateManager.updateDefaultLens(currentLens);
  });
}
