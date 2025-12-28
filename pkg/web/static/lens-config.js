/**
 * Lens-Based Visualization Configuration
 *
 * Defines the configuration structure and default lenses for the graph visualization system.
 */

/**
 * @typedef {Object} BaseSetConfig
 * @property {'full-graph'|'reachable-from-binary'|'package-level'} type - Base set type
 * @property {string} [binaryLabel] - Binary label for reachable-from-binary mode
 * @property {string} [packagePath] - Package path filter
 */

/**
 * @typedef {Object} NodeVisibility
 * @property {string[]} targetTypes - Which target types to show
 * @property {string[]} fileTypes - Which file types to show
 * @property {boolean} showUncovered - Show uncovered files
 * @property {boolean} showExternal - Show external dependencies
 * @property {boolean} showSystemLibraries - Show system libraries
 */

/**
 * @typedef {Object} DistanceRule
 * @property {number|'infinite'} distance - Distance from focused node
 * @property {NodeVisibility} nodeVisibility - What nodes to show
 * @property {number} collapseLevel - Hierarchy depth (positive=top-down, negative=bottom-up)
 * @property {boolean} showEdges - Whether to show edges
 * @property {string[]} [edgeTypes] - Which edge types to show
 */

/**
 * @typedef {Object} FilterConfig
 * @property {boolean} [hideExternal] - Hide external dependencies
 * @property {boolean} [hideUncovered] - Hide uncovered files
 * @property {boolean} [hideSystemLibs] - Hide system libraries
 */

/**
 * @typedef {Object} EdgeDisplayRules
 * @property {Set<string>} types - Edge types to show
 * @property {boolean} aggregateCollapsed - Show aggregated edges for collapsed nodes
 * @property {number} [minimumCount] - Minimum edge count to display
 */

/**
 * @typedef {Object} LensConfig
 * @property {string} name - Display name
 * @property {BaseSetConfig} baseSet - Base graph configuration
 * @property {DistanceRule[]} distanceRules - Rules by distance from focus
 * @property {FilterConfig} globalFilters - Always-applied filters
 * @property {EdgeDisplayRules} edgeRules - Edge visibility rules
 */

/**
 * Default lens: Package-level view
 * Shows targets but hides individual files
 */
const DEFAULT_PACKAGE_LENS = {
  name: "Package View",
  baseSet: { type: 'full-graph' },
  distanceRules: [
    {
      distance: 'infinite',
      nodeVisibility: {
        targetTypes: ['cc_binary', 'cc_shared_library', 'cc_library'],
        fileTypes: ['all'],  // Show all files
        showUncovered: false,
        showExternal: false,
        showSystemLibraries: true
      },
      collapseLevel: 3,  // Show down to file level
      showEdges: true,
      edgeTypes: ['static', 'dynamic', 'system_link', 'data', 'compile', 'symbol']
    }
  ],
  globalFilters: {},
  edgeRules: {
    types: new Set(['static', 'dynamic', 'system_link', 'data', 'compile', 'symbol']),
    aggregateCollapsed: true
  }
};

/**
 * Focus lens: Currently identical to default lens
 * This keeps the graph structure simple - focusing only adds visual highlighting
 * File expansion can be added later when we implement dynamic data loading
 */
const DEFAULT_FOCUS_LENS = {
  name: "Focus View",
  baseSet: { type: 'full-graph' },
  distanceRules: [
    {
      distance: 'infinite',  // Apply same rules to all distances for now
      nodeVisibility: {
        targetTypes: ['cc_binary', 'cc_shared_library', 'cc_library'],
        fileTypes: ['all'],  // Show all files
        showUncovered: false,
        showExternal: false,
        showSystemLibraries: true
      },
      collapseLevel: 3,  // Show down to file level (same as default)
      showEdges: true,
      edgeTypes: ['static', 'dynamic', 'system_link', 'data', 'compile', 'symbol']
    }
  ],
  globalFilters: {},
  edgeRules: {
    types: new Set(['static', 'dynamic', 'system_link', 'data', 'compile', 'symbol']),
    aggregateCollapsed: true
  }
};

/**
 * Clone a lens configuration (deep copy)
 * @param {LensConfig} lens - Lens to clone
 * @returns {LensConfig} Cloned lens
 */
function cloneLens(lens) {
  return {
    name: lens.name,
    baseSet: { ...lens.baseSet },
    distanceRules: lens.distanceRules.map(rule => ({
      ...rule,
      nodeVisibility: { ...rule.nodeVisibility },
      edgeTypes: rule.edgeTypes ? [...rule.edgeTypes] : undefined
    })),
    globalFilters: { ...lens.globalFilters },
    edgeRules: {
      types: new Set(lens.edgeRules.types),
      aggregateCollapsed: lens.edgeRules.aggregateCollapsed,
      minimumCount: lens.edgeRules.minimumCount
    }
  };
}
