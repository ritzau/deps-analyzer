// Use structured logger (loaded from logger.js)
const lensLogger = new Logger();

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
  setupResetControls();
  setupDefaultLensControls();
  setupDetailLensControls();

  // Sync UI controls with restored state
  syncUIWithState();
}

/**
 * Sync UI controls with current view state
 * Called on initialization to reflect any restored state from localStorage
 */
function syncUIWithState() {
  const state = viewStateManager.getState();

  // Sync active tab
  document
    .querySelectorAll(".tab-button")
    .forEach((b) => b.classList.remove("active"));
  const activeTabButton = document.querySelector(
    `.tab-button[data-tab="${state.activeTab}"]`,
  );
  if (activeTabButton) {
    activeTabButton.classList.add("active");
  }

  document
    .querySelectorAll(".tab-pane")
    .forEach((pane) => pane.classList.remove("active"));
  const activeTabPane = document.getElementById(state.activeTab + "Tab");
  if (activeTabPane) {
    activeTabPane.classList.add("active");
  }

  // Sync global filters (default lens)
  const filters = state.defaultLens.globalFilters;
  const hideExternalCheckbox = document.getElementById("hideExternal");
  if (hideExternalCheckbox) {
    hideExternalCheckbox.checked = filters.hideExternal || false;
  }

  const hideUncoveredCheckbox = document.getElementById("hideUncovered");
  if (hideUncoveredCheckbox) {
    hideUncoveredCheckbox.checked = filters.hideUncovered || false;
  }

  const hideSystemLibsCheckbox = document.getElementById("hideSystemLibs");
  if (hideSystemLibsCheckbox) {
    hideSystemLibsCheckbox.checked = filters.hideSystemLibs || false;
  }

  // Sync edge type checkboxes
  const edgeTypes = state.defaultLens.edgeRules.types;
  const showStaticCheckbox = document.getElementById("showStatic");
  if (showStaticCheckbox) {
    showStaticCheckbox.checked = edgeTypes.has("static");
  }

  const showDynamicCheckbox = document.getElementById("showDynamic");
  if (showDynamicCheckbox) {
    showDynamicCheckbox.checked = edgeTypes.has("dynamic");
  }

  const showDataCheckbox = document.getElementById("showData");
  if (showDataCheckbox) {
    showDataCheckbox.checked = edgeTypes.has("data");
  }

  const showCompileCheckbox = document.getElementById("showCompile");
  if (showCompileCheckbox) {
    showCompileCheckbox.checked = edgeTypes.has("compile");
  }

  const showSymbolCheckbox = document.getElementById("showSymbol");
  if (showSymbolCheckbox) {
    showSymbolCheckbox.checked = edgeTypes.has("symbol");
  }

  // Sync collapse edge types checkbox
  const collapseEdgeTypesCheckbox =
    document.getElementById("collapseEdgeTypes");
  if (collapseEdgeTypesCheckbox) {
    collapseEdgeTypesCheckbox.checked =
      state.defaultLens.edgeRules.collapseEdgeTypes || false;
  }

  // Sync collapse level radio buttons
  const collapseLevel = state.defaultLens.distanceRules[0]?.collapseLevel || 2;
  const collapseLevelRadio = document.querySelector(
    `input[name="collapseLevel"][value="${collapseLevel}"]`,
  );
  if (collapseLevelRadio) {
    collapseLevelRadio.checked = true;
  }

  // Sync navigation filters (rule types)
  const ruleTypes = state.navigationFilters.ruleTypes;
  const ruleTypeCheckboxes = document.querySelectorAll(
    '#ruleTypeMenu input[type="checkbox"]',
  );
  ruleTypeCheckboxes.forEach((checkbox) => {
    checkbox.checked = ruleTypes.has(checkbox.value);
  });

  lensLogger.debug("[LensControls] UI synced with restored state");
}

/**
 * Set up tab switching (Tree | Default | Focus)
 */
function setupTabSwitching() {
  document.querySelectorAll(".tab-button").forEach((button) => {
    button.addEventListener("click", () => {
      // Update active button
      document
        .querySelectorAll(".tab-button")
        .forEach((b) => b.classList.remove("active"));
      button.classList.add("active");

      // Update active pane
      const tabName = button.getAttribute("data-tab");
      document
        .querySelectorAll(".tab-pane")
        .forEach((pane) => pane.classList.remove("active"));
      const tabPane = document.getElementById(tabName + "Tab");
      if (tabPane) {
        tabPane.classList.add("active");
      }

      viewStateManager.setActiveTab(tabName);
    });
  });
}

/**
 * Set up reset controls
 * Note: Reset controls removed - users can click background to clear selection
 */
function setupResetControls() {
  // No controls to set up - function kept for API compatibility
}

/**
 * Set up default lens configuration controls
 */
function setupDefaultLensControls() {
  // Global filters
  const filterIds = ["hideExternal", "hideUncovered", "hideSystemLibs"];
  filterIds.forEach((id) => {
    const checkbox = document.getElementById(id);
    if (checkbox) {
      checkbox.addEventListener("change", () => {
        const currentLens = cloneLens(viewStateManager.getState().defaultLens);
        currentLens.globalFilters.hideExternal =
          document.getElementById("hideExternal")?.checked || false;
        currentLens.globalFilters.hideUncovered =
          document.getElementById("hideUncovered")?.checked || false;
        currentLens.globalFilters.hideSystemLibs =
          document.getElementById("hideSystemLibs")?.checked || false;
        viewStateManager.updateDefaultLens(currentLens);
      });
    }
  });

  // Edge type checkboxes
  const edgeTypeIds = [
    "showStatic",
    "showDynamic",
    "showData",
    "showCompile",
    "showSymbol",
  ];
  edgeTypeIds.forEach((id) => {
    const checkbox = document.getElementById(id);
    if (checkbox) {
      checkbox.addEventListener("change", () => {
        lensLogger.debug("[LensControls] Edge type checkbox changed:", id);
        const types = new Set();

        if (document.getElementById("showStatic")?.checked) types.add("static");
        if (document.getElementById("showDynamic")?.checked)
          types.add("dynamic");
        if (document.getElementById("showData")?.checked) types.add("data");
        if (document.getElementById("showCompile")?.checked)
          types.add("compile");
        if (document.getElementById("showSymbol")?.checked) types.add("symbol");

        // Always keep system_link
        types.add("system_link");

        lensLogger.debug("[LensControls] New edge types:", Array.from(types));

        // Update both default and detail lenses to use same edge rules (atomic)
        const currentDefaultLens = cloneLens(
          viewStateManager.getState().defaultLens,
        );
        currentDefaultLens.edgeRules.types = types;

        const currentDetailLens = cloneLens(
          viewStateManager.getState().detailLens,
        );
        currentDetailLens.edgeRules.types = types;

        viewStateManager.updateBothLenses(
          currentDefaultLens,
          currentDetailLens,
        );
      });
    }
  });

  // Collapse edge types checkbox
  const collapseEdgeTypesCheckbox =
    document.getElementById("collapseEdgeTypes");
  if (collapseEdgeTypesCheckbox) {
    collapseEdgeTypesCheckbox.addEventListener("change", () => {
      lensLogger.debug(
        "[LensControls] Collapse edge types changed:",
        collapseEdgeTypesCheckbox.checked,
      );

      // Update both default and detail lenses to use same edge rules (atomic)
      const currentDefaultLens = cloneLens(
        viewStateManager.getState().defaultLens,
      );
      currentDefaultLens.edgeRules.collapseEdgeTypes =
        collapseEdgeTypesCheckbox.checked;

      const currentDetailLens = cloneLens(
        viewStateManager.getState().detailLens,
      );
      currentDetailLens.edgeRules.collapseEdgeTypes =
        collapseEdgeTypesCheckbox.checked;

      viewStateManager.updateBothLenses(currentDefaultLens, currentDetailLens);
    });
  }

  // Collapse level radio buttons
  const collapseLevelRadios = document.querySelectorAll(
    'input[name="collapseLevel"]',
  );
  collapseLevelRadios.forEach((radio) => {
    radio.addEventListener("change", (e) => {
      if (e.target.checked) {
        const level = parseInt(e.target.value);
        lensLogger.debug("[LensControls] Collapse level changed to:", level);
        const currentLens = cloneLens(viewStateManager.getState().defaultLens);

        // Update the collapse level in the distance rule
        const rule = currentLens.distanceRules[0];
        if (rule) {
          rule.collapseLevel = level;

          // Also update file visibility to match
          if (level >= 3) {
            rule.nodeVisibility.fileTypes = ["all"];
            rule.nodeVisibility.showUncovered = true;
          } else {
            rule.nodeVisibility.fileTypes = ["none"];
            rule.nodeVisibility.showUncovered = false;
          }
        }

        // Update lens and clear selection atomically (single backend request)
        // Otherwise the detail lens keeps being used with old settings
        viewStateManager.updateDefaultLensAndClearSelection(currentLens);
      }
    });
  });
}

/**
 * Set up detail lens configuration controls
 */
function setupDetailLensControls() {
  // Distance 0 (selected nodes) file visibility
  const detailD0Files = document.getElementById("detailD0Files");
  if (detailD0Files) {
    detailD0Files.addEventListener("change", (e) => {
      const currentLens = cloneLens(viewStateManager.getState().detailLens);

      // Find distance 0 rule
      const rule = currentLens.distanceRules.find((r) => r.distance === 0);
      if (rule) {
        if (e.target.value === "all") {
          rule.nodeVisibility.fileTypes = ["all"];
          rule.collapseLevel = 3; // Show files
        } else {
          rule.nodeVisibility.fileTypes = ["none"];
          rule.collapseLevel = 2; // Hide files
        }
      }

      viewStateManager.updateDetailLens(currentLens);
    });
  }

  // Distance 1 (neighbors) file visibility
  const detailD1Files = document.getElementById("detailD1Files");
  if (detailD1Files) {
    detailD1Files.addEventListener("change", (e) => {
      const currentLens = cloneLens(viewStateManager.getState().detailLens);

      // Find distance 1 rule
      const rule = currentLens.distanceRules.find((r) => r.distance === 1);
      if (rule) {
        if (e.target.value === "all") {
          rule.nodeVisibility.fileTypes = ["all"];
          rule.collapseLevel = 3; // Show files
        } else {
          rule.nodeVisibility.fileTypes = ["none"];
          rule.collapseLevel = 2; // Hide files
        }
      }

      viewStateManager.updateDetailLens(currentLens);
    });
  }

  // Distance infinite (rest of graph) visibility
  const detailInfiniteView = document.getElementById("detailInfiniteView");
  if (detailInfiniteView) {
    detailInfiniteView.addEventListener("change", (e) => {
      const currentLens = cloneLens(viewStateManager.getState().detailLens);

      // Find infinite distance rule
      const rule = currentLens.distanceRules.find(
        (r) => r.distance === "infinite",
      );
      if (rule) {
        if (e.target.value === "hide") {
          // Hide completely
          rule.nodeVisibility.targetTypes = [];
        } else if (e.target.value === "collapsed") {
          // Show collapsed
          rule.nodeVisibility.targetTypes = [
            "cc_binary",
            "cc_shared_library",
            "cc_library",
          ];
          rule.collapseLevel = 1; // Package level
        } else {
          // Same as default
          // Copy from default lens
          const defaultRule =
            viewStateManager.getState().defaultLens.distanceRules[0];
          if (defaultRule) {
            rule.nodeVisibility = { ...defaultRule.nodeVisibility };
            rule.collapseLevel = defaultRule.collapseLevel;
          }
        }
      }

      viewStateManager.updateDetailLens(currentLens);
    });
  }
}
