/**
 * Local Storage Persistence
 *
 * Saves and restores view settings to browser's localStorage.
 * Settings are persisted across page reloads.
 */

const STORAGE_KEY = 'deps-analyzer-view-settings';
const STORAGE_VERSION = 1;

/**
 * Save current view state to localStorage
 * @param {Object} state - View state to save
 */

// biome-ignore lint/correctness/noUnusedVariables: Used in view-state.js
function saveViewState(state) {
  try {
    const toSave = {
      version: STORAGE_VERSION,
      defaultLens: serializeLens(state.defaultLens),
      detailLens: serializeLens(state.detailLens),
      navigationFilters: {
        ruleTypes: Array.from(state.navigationFilters.ruleTypes),
        searchText: state.navigationFilters.searchText,
      },
      activeTab: state.activeTab,
    };

    localStorage.setItem(STORAGE_KEY, JSON.stringify(toSave));
  } catch (e) {
    console.warn('Failed to save view state to localStorage:', e);
  }
}

/**
 * Load view state from localStorage
 * @returns {Object|null} Restored state or null if not available
 */

// biome-ignore lint/correctness/noUnusedVariables: Use in view-state.js
function loadViewState() {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (!stored) {
      return null;
    }

    const parsed = JSON.parse(stored);

    // Version check - if version mismatch, ignore stored state
    if (parsed.version !== STORAGE_VERSION) {
      console.warn('Stored view state version mismatch, using defaults');
      return null;
    }

    return {
      defaultLens: deserializeLens(parsed.defaultLens),
      detailLens: deserializeLens(parsed.detailLens),
      navigationFilters: {
        ruleTypes: new Set(parsed.navigationFilters.ruleTypes),
        searchText: parsed.navigationFilters.searchText,
      },
      activeTab: parsed.activeTab,
    };
  } catch (e) {
    console.warn('Failed to load view state from localStorage:', e);
    return null;
  }
}

/**
 * Serialize lens configuration for storage
 * Converts Sets to Arrays for JSON serialization
 * @param {LensConfig} lens - Lens to serialize
 * @returns {Object} Serializable lens object
 */
function serializeLens(lens) {
  return {
    name: lens.name,
    baseSet: { ...lens.baseSet },
    distanceRules: lens.distanceRules.map((rule) => ({
      ...rule,
      nodeVisibility: { ...rule.nodeVisibility },
      edgeTypes: rule.edgeTypes ? [...rule.edgeTypes] : undefined,
    })),
    globalFilters: { ...lens.globalFilters },
    edgeRules: {
      types: Array.from(lens.edgeRules.types),
      aggregateCollapsed: lens.edgeRules.aggregateCollapsed,
      collapseEdgeTypes: lens.edgeRules.collapseEdgeTypes,
      minimumCount: lens.edgeRules.minimumCount,
    },
  };
}

/**
 * Deserialize lens configuration from storage
 * Converts Arrays back to Sets
 * @param {Object} serialized - Serialized lens object
 * @returns {LensConfig} Lens configuration
 */
function deserializeLens(serialized) {
  return {
    name: serialized.name,
    baseSet: { ...serialized.baseSet },
    distanceRules: serialized.distanceRules.map((rule) => ({
      ...rule,
      nodeVisibility: { ...rule.nodeVisibility },
      edgeTypes: rule.edgeTypes ? [...rule.edgeTypes] : undefined,
    })),
    globalFilters: { ...serialized.globalFilters },
    edgeRules: {
      types: new Set(serialized.edgeRules.types),
      aggregateCollapsed: serialized.edgeRules.aggregateCollapsed,
      collapseEdgeTypes: serialized.edgeRules.collapseEdgeTypes,
      minimumCount: serialized.edgeRules.minimumCount,
    },
  };
}
