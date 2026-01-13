package lens

// LensConfig defines how graph nodes and edges should be filtered and displayed
type LensConfig struct {
	Name          string           `json:"name"`
	BaseSet       BaseSetConfig    `json:"baseSet"`
	DistanceRules []DistanceRule   `json:"distanceRules"`
	GlobalFilters GlobalFilters    `json:"globalFilters"`
	EdgeRules     EdgeDisplayRules `json:"edgeRules"`
}

// BaseSetConfig determines the base set of nodes to consider
type BaseSetConfig struct {
	Type        string  `json:"type"` // "full-graph", "reachable-from-binary", "package-level"
	BinaryLabel *string `json:"binaryLabel,omitempty"`
	PackagePath *string `json:"packagePath,omitempty"`
}

// DistanceRule defines visibility and display rules based on distance from selected nodes
type DistanceRule struct {
	Distance       interface{}    `json:"distance"` // int or "infinite"
	NodeVisibility NodeVisibility `json:"nodeVisibility"`
	CollapseLevel  int            `json:"collapseLevel"`
	ShowEdges      bool           `json:"showEdges"`
	EdgeTypes      []string       `json:"edgeTypes,omitempty"`
}

// NodeVisibility determines which types of nodes should be visible
type NodeVisibility struct {
	TargetTypes         []string `json:"targetTypes"`
	FileTypes           []string `json:"fileTypes"`
	ShowUncovered       bool     `json:"showUncovered"`
	ShowExternal        bool     `json:"showExternal"`
	ShowSystemLibraries bool     `json:"showSystemLibraries"`
}

// GlobalFilters are always-applied visibility filters
type GlobalFilters struct {
	HideExternal   bool `json:"hideExternal,omitempty"`
	HideUncovered  bool `json:"hideUncovered,omitempty"`
	HideSystemLibs bool `json:"hideSystemLibs,omitempty"`
	ShowOnlyLdd    bool `json:"showOnlyLdd,omitempty"`
}

// EdgeDisplayRules control which edges are shown
type EdgeDisplayRules struct {
	Types              []string `json:"types"`
	AggregateCollapsed bool     `json:"aggregateCollapsed"`
	CollapseEdgeTypes  bool     `json:"collapseEdgeTypes"`
	MinimumCount       *int     `json:"minimumCount,omitempty"`
}

// NodeState tracks the computed state for a node during rendering
type NodeState struct {
	Visible     bool
	Collapsed   bool
	Distance    interface{} // int or "infinite"
	AppliedLens string      // "default" or "detail"
	Rule        *DistanceRule
}
