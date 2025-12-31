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
    this.state = {
      // Layer 1: Default lens
      defaultLens: cloneLens(DEFAULT_PACKAGE_LENS),

      // Layer 2: Detail lens
      detailLens: cloneLens(DEFAULT_DETAIL_LENS),
      selectedNodes: new Set(),

      // UI state
      activeTab: 'tree'  // 'tree' | 'default' | 'detail'
    };

    this.listeners = [];
  }

  /**
   * Get current state (read-only)
   * @returns {Object} Current view state
   */
  getState() {
    return this.state;
  }

  /**
   * Update default lens configuration
   * @param {LensConfig} lens - New default lens
   */
  updateDefaultLens(lens) {
    console.log('[ViewState] updateDefaultLens called');
    console.log('[ViewState] New lens edge types:', Array.from(lens.edgeRules.types));
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
    console.log('[ViewState] updateBothLenses called (atomic update)');
    this.state.defaultLens = defaultLens;
    this.state.detailLens = detailLens;
    this.notifyListeners(); // Only notify once
  }

  /**
   * Set selection to a specific set of nodes (replaces entire selection)
   * @param {string[]} nodeIds - Array of node IDs to select
   */
  setSelection(nodeIds) {
    console.log('[ViewState] setSelection called with:', nodeIds);
    this.state.selectedNodes = new Set(nodeIds);
    this.notifyListeners();
  }

  /**
   * Toggle a node in the selection (add if not present, remove if present)
   * @param {string} nodeId - Node ID to toggle
   */
  toggleSelection(nodeId) {
    console.log('[ViewState] toggleSelection called with nodeId:', nodeId);

    if (this.state.selectedNodes.has(nodeId)) {
      this.state.selectedNodes.delete(nodeId);
      console.log('[ViewState] Removed from selection:', nodeId);
    } else {
      this.state.selectedNodes.add(nodeId);
      console.log('[ViewState] Added to selection:', nodeId);
    }
    console.log('[ViewState] Selected nodes now:', Array.from(this.state.selectedNodes));
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
    this.listeners.forEach(callback => {
      try {
        callback(this.state);
      } catch (error) {
        console.error('Error in view state listener:', error);
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
    if (!oldState) return true;  // Initial load always needs layout

    // Check if base set changed (different graph structure)
    const oldBase = oldState.defaultLens.baseSet;
    const newBase = newState.defaultLens.baseSet;

    if (oldBase.type !== newBase.type) {
      console.log('[ViewState] Full re-layout: baseSet type changed');
      return true;
    }

    if (oldBase.binaryLabel !== newBase.binaryLabel) {
      console.log('[ViewState] Full re-layout: binaryLabel changed');
      return true;
    }

    if (oldBase.packagePath !== newBase.packagePath) {
      console.log('[ViewState] Full re-layout: packagePath changed');
      return true;
    }

    // Check if selected nodes changed (different graph structure)
    const oldSelection = Array.from(oldState.selectedNodes).sort().join(',');
    const newSelection = Array.from(newState.selectedNodes).sort().join(',');
    if (oldSelection !== newSelection) {
      console.log('[ViewState] Full re-layout: selected nodes changed');
      return true;
    }

    // Changes to collapse levels, edge types, or visibility settings do NOT require full re-layout
    // These are visual changes that can use cached positions
    console.log('[ViewState] No full re-layout needed - visual change only');
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
      detailLensName: this.state.detailLens.name
    };
  }
}

// Global instance
const viewStateManager = new ViewStateManager();
