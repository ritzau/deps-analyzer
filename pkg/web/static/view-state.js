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

      // Layer 2: Focus lens
      focusLens: cloneLens(DEFAULT_FOCUS_LENS),
      focusMode: 'single',  // 'single' or 'multi-select'
      focusedNodes: new Set(),

      // Layer 3: Manual overrides
      manualOverrides: new Map(),  // nodeId -> {collapsed: boolean, timestamp: number}

      // UI state
      activeTab: 'tree'  // 'tree' | 'default' | 'focus'
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
   * Update focus lens configuration
   * @param {LensConfig} lens - New focus lens
   */
  updateFocusLens(lens) {
    this.state.focusLens = lens;
    this.notifyListeners();
  }

  /**
   * Update focus on a node
   * In single mode: replace focused set
   * In multi mode: toggle node in focused set
   *
   * @param {string} nodeId - Node ID to focus
   */
  updateFocus(nodeId) {
    console.log('[ViewState] updateFocus called with nodeId:', nodeId, 'mode:', this.state.focusMode);

    if (this.state.focusMode === 'single') {
      // Single mode: replace focused set
      this.state.focusedNodes = new Set([nodeId]);
      console.log('[ViewState] Single mode - focused nodes now:', Array.from(this.state.focusedNodes));
    } else {
      // Multi mode: toggle in set
      if (this.state.focusedNodes.has(nodeId)) {
        this.state.focusedNodes.delete(nodeId);
        console.log('[ViewState] Multi mode - removed:', nodeId);
      } else {
        this.state.focusedNodes.add(nodeId);
        console.log('[ViewState] Multi mode - added:', nodeId);
      }
      console.log('[ViewState] Multi mode - focused nodes now:', Array.from(this.state.focusedNodes));
    }
    this.notifyListeners();
  }

  /**
   * Clear all focused nodes
   */
  clearFocus() {
    this.state.focusedNodes = new Set();
    this.notifyListeners();
  }

  /**
   * Set manual override for a node's collapse state
   *
   * @param {string} nodeId - Node ID
   * @param {boolean} collapsed - Collapsed state
   */
  setManualOverride(nodeId, collapsed) {
    this.state.manualOverrides.set(nodeId, {
      collapsed: collapsed,
      timestamp: Date.now()
    });
    this.notifyListeners();
  }

  /**
   * Clear manual override for a node (revert to lens rules)
   *
   * @param {string} nodeId - Node ID
   */
  clearManualOverride(nodeId) {
    this.state.manualOverrides.delete(nodeId);
    this.notifyListeners();
  }

  /**
   * Clear all manual overrides
   */
  resetManualOverrides() {
    this.state.manualOverrides.clear();
    this.notifyListeners();
  }

  /**
   * Reset all layers (focus + manual)
   */
  resetAll() {
    this.state.focusedNodes = new Set();
    this.state.manualOverrides.clear();
    this.notifyListeners();
  }

  /**
   * Set focus mode (single or multi-select)
   *
   * @param {'single'|'multi-select'} mode - Focus mode
   */
  setFocusMode(mode) {
    this.state.focusMode = mode;

    // If switching to single mode with multiple focused nodes, keep only first
    if (mode === 'single' && this.state.focusedNodes.size > 1) {
      const first = Array.from(this.state.focusedNodes)[0];
      this.state.focusedNodes = new Set([first]);
    }

    this.notifyListeners();
  }

  /**
   * Set active tab
   *
   * @param {'tree'|'default'|'focus'} tab - Tab name
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
   * Get debug info about current state
   * @returns {Object} Debug information
   */
  getDebugInfo() {
    return {
      focusMode: this.state.focusMode,
      focusedNodeCount: this.state.focusedNodes.size,
      focusedNodes: Array.from(this.state.focusedNodes),
      manualOverrideCount: this.state.manualOverrides.size,
      activeTab: this.state.activeTab,
      defaultLensName: this.state.defaultLens.name,
      focusLensName: this.state.focusLens.name
    };
  }
}

// Global instance
const viewStateManager = new ViewStateManager();
