package model

// TargetKind represents the type of Bazel target
type TargetKind string

const (
	TargetKindBinary        TargetKind = "cc_binary"
	TargetKindSharedLibrary TargetKind = "cc_shared_library"
	TargetKindLibrary       TargetKind = "cc_library"
)

// DependencyType represents the type of dependency between targets
type DependencyType string

const (
	DependencyStatic  DependencyType = "static"  // Static linkage (deps to cc_library)
	DependencyDynamic DependencyType = "dynamic" // Dynamic linkage (dynamic_deps or deps to cc_shared_library)
	DependencyData    DependencyType = "data"    // Runtime data dependency
)

// Target represents a Bazel build target
type Target struct {
	Label   string     `json:"label"`   // Full label (e.g., "//main:test_app")
	Kind    TargetKind `json:"kind"`    // cc_binary, cc_shared_library, or cc_library
	Package string     `json:"package"` // Package path (e.g., "//main")
	Name    string     `json:"name"`    // Target name (e.g., "test_app")

	// Source files
	Sources []string `json:"sources,omitempty"` // .cc files
	Headers []string `json:"headers,omitempty"` // .h files

	// Dependencies (raw lists from BUILD file)
	Deps        []string `json:"deps,omitempty"`         // Regular deps attribute
	DynamicDeps []string `json:"dynamicDeps,omitempty"`  // dynamic_deps attribute
	Data        []string `json:"data,omitempty"`         // data attribute
	Linkopts    []string `json:"linkopts,omitempty"`     // linkopts (for system libraries)
}

// Dependency represents a typed dependency between two targets
type Dependency struct {
	From string         `json:"from"` // Source target label
	To   string         `json:"to"`   // Target dependency label
	Type DependencyType `json:"type"` // Type of dependency
}

// Workspace represents the complete build graph
type Workspace struct {
	Targets      map[string]*Target `json:"targets"`      // Map of label -> Target
	Dependencies []Dependency       `json:"dependencies"` // All dependencies with types
}
