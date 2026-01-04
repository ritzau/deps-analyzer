// Use structured logger (loaded from logger.js)
const viewStateLogger = new Logger();

/**
 * View State Manager
 *
 * Manages the three-layer view state:
 * - Layer 1: Default lens (applies to all nodes)
 * - Layer 2: Focus lens (overrides for focused nodes)
 * - Layer 3: Manual overrides (per-node collapse state)
 *
 * Provides methods to update state and notify listeners of changes.
 */

class ViewStateManager {
  constructor() {
    // Try to load saved state from localStorage
    const savedState = loadViewState();

    // Migrate old 'detail' tab to 'default' (tabs were merged)
    let activeTab = savedState?.activeTab || 'tree';
    if (activeTab === 'detail') {
      activeTab = 'default';
    }

    this.state = {
      // Layer 1: Default lens
      defaultLens: savedState?.defaultLens || cloneLens(DEFAULT_PACKAGE_LENS),

      // Layer 2: Detail lens
      detailLens: savedState?.detailLens || cloneLens(DEFAULT_DETAIL_LENS),
      selectedNodes: new Set(), // Never persist selection

      // Navigation filters
      navigationFilters: savedState?.navigationFilters || {
        ruleTypes: new Set(['cc_binary', 'cc_library', 'cc_shared_library']),
        searchText: '',
      },

      // UI state
      activeTab: activeTab, // 'tree' | 'default'
    };

    this.listeners = [];

    // Log if we restored state
    if (savedState) {
      viewStateLogger.debug('[ViewState] Restored state from localStorage');
    }
  }

  /**
   * Get current state (read-only)
   * @returns {Object} Current view state
   */
  getState() {
    return this.state;
  }

  /**
   * Update navigation filters (client-side only, does NOT trigger graph re-fetch)
   * @param {Set<string>} ruleTypes - Set of rule types to show
   * @param {string} searchText - Search text for filtering by label
   */
  updateNavigationFilters(ruleTypes, searchText) {
    viewStateLogger.debug('[ViewState] Updating navigation filters', {
      ruleTypes: Array.from(ruleTypes),
      searchText,
    });
    this.state.navigationFilters.ruleTypes = ruleTypes;
    this.state.navigationFilters.searchText = searchText;

    // Save to localStorage
    saveViewState(this.state);

    // Re-render navigation list only (don't trigger graph re-fetch)
    if (window.filterAndRenderNavigationList) {
      window.filterAndRenderNavigationList();
    }
  }

  /**
   * Update default lens configuration
   * @param {LensConfig} lens - New default lens
   */
  updateDefaultLens(lens) {
    viewStateLogger.debug('[ViewState] updateDefaultLens called');
    viewStateLogger.debug('[ViewState] New lens edge types:', Array.from(lens.edgeRules.types));
    this.state.defaultLens = lens;
    this.notifyListeners();
  }

  /**
   * Update detail lens configuration
   * @param {LensConfig} lens - New detail lens
   */
  updateDetailLens(lens) {
    this.state.detailLens = lens;
    this.notifyListeners();
  }

  /**
   * Update both default and detail lenses atomically (single notification)
   * Use this when edge settings or other global settings need to apply to both lenses
   * @param {LensConfig} defaultLens - New default lens
   * @param {LensConfig} detailLens - New detail lens
   */
  updateBothLenses(defaultLens, detailLens) {
    viewStateLogger.debug('[ViewState] updateBothLenses called (atomic update)');
    this.state.defaultLens = defaultLens;
    this.state.detailLens = detailLens;
    this.notifyListeners(); // Only notify once
  }

  /**
   * Set selection to a specific set of nodes (replaces entire selection)
   * @param {string[]} nodeIds - Array of node IDs to select
   */
  setSelection(nodeIds) {
    viewStateLogger.debug('[ViewState] setSelection called with:', nodeIds);
    this.state.selectedNodes = new Set(nodeIds);
    this.notifyListeners();
  }

  /**
   * Toggle a node in the selection (add if not present, remove if present)
   * @param {string} nodeId - Node ID to toggle
   */
  toggleSelection(nodeId) {
    viewStateLogger.debug('[ViewState] toggleSelection called with nodeId:', nodeId);

    if (this.state.selectedNodes.has(nodeId)) {
      this.state.selectedNodes.delete(nodeId);
      viewStateLogger.debug('[ViewState] Removed from selection:', nodeId);
    } else {
      this.state.selectedNodes.add(nodeId);
      viewStateLogger.debug('[ViewState] Added to selection:', nodeId);
    }
    viewStateLogger.debug('[ViewState] Selected nodes now:', Array.from(this.state.selectedNodes));
    this.notifyListeners();
  }

  /**
   * Clear all selected nodes
   */
  clearSelection() {
    this.state.selectedNodes = new Set();
    this.notifyListeners();
  }

  /**
   * Update default lens and clear selection atomically (single notification)
   * Use this when changing default lens settings where keeping selection doesn't make sense
   * @param {LensConfig} lens - New default lens
   */
  updateDefaultLensAndClearSelection(lens) {
    viewStateLogger.debug('[ViewState] updateDefaultLensAndClearSelection called (atomic update)');
    this.state.defaultLens = lens;
    this.state.selectedNodes = new Set();
    this.notifyListeners(); // Only notify once
  }

  /**
   * Set active tab
   *
   * @param {'tree'|'default'|'detail'} tab - Tab name
   */
  setActiveTab(tab) {
    this.state.activeTab = tab;
    this.notifyListeners();
  }

  /**
   * Register state change listener
   *
   * @param {Function} callback - Callback function called with new state
   */
  addListener(callback) {
    this.listeners.push(callback);
  }

  /**
   * Remove state change listener
   *
   * @param {Function} callback - Callback to remove
   */
  removeListener(callback) {
    const index = this.listeners.indexOf(callback);
    if (index > -1) {
      this.listeners.splice(index, 1);
    }
  }

  /**
   * Notify all listeners of state change
   * @private
   */
  notifyListeners() {
    // Save state to localStorage
    saveViewState(this.state);

    this.listeners.forEach((callback) => {
      try {
        callback(this.state);
      } catch (error) {
        viewStateLogger.error('Error in view state listener:', error);
      }
    });
  }

  /**
   * Determine if a state change requires clearing cached positions (full re-layout)
   * This is needed when the graph topology actually changes, not just visual properties.
   *
   * @param {Object} oldState - Previous state
   * @param {Object} newState - New state
   * @returns {boolean} True if positions should be cleared
   */
  needsFullRelayout(oldState, newState) {
    if (!oldState) return true; // Initial load always needs layout

    // Check if base set changed (different graph structure)
    const oldBase = oldState.defaultLens.baseSet;
    const newBase = newState.defaultLens.baseSet;

    if (oldBase.type !== newBase.type) {
      viewStateLogger.debug('[ViewState] Full re-layout: baseSet type changed');
      return true;
    }

    if (oldBase.binaryLabel !== newBase.binaryLabel) {
      viewStateLogger.debug('[ViewState] Full re-layout: binaryLabel changed');
      return true;
    }

    if (oldBase.packagePath !== newBase.packagePath) {
      viewStateLogger.debug('[ViewState] Full re-layout: packagePath changed');
      return true;
    }

    // Check if selected nodes changed (different graph structure)
    const oldSelection = Array.from(oldState.selectedNodes).sort().join(',');
    const newSelection = Array.from(newState.selectedNodes).sort().join(',');
    if (oldSelection !== newSelection) {
      viewStateLogger.debug('[ViewState] Full re-layout: selected nodes changed');
      return true;
    }

    // Changes to collapse levels, edge types, or visibility settings do NOT require full re-layout
    // These are visual changes that can use cached positions
    viewStateLogger.debug('[ViewState] No full re-layout needed - visual change only');
    return false;
  }

  /**
   * Get debug info about current state
   * @returns {Object} Debug information
   */
  getDebugInfo() {
    return {
      selectedNodeCount: this.state.selectedNodes.size,
      selectedNodes: Array.from(this.state.selectedNodes),
      activeTab: this.state.activeTab,
      defaultLensName: this.state.defaultLens.name,
      detailLensName: this.state.detailLens.name,
    };
  }
}

// Global instance
const _viewStateManager = new ViewStateManager();
