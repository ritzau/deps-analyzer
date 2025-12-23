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

// Package represents a Bazel package with its targets
type Package struct {
	Path    string             `json:"path"`    // Package path (e.g., "//main")
	Targets map[string]*Target `json:"targets"` // Map of target name -> Target
}

// PackageDependency represents dependencies between two packages
type PackageDependency struct {
	From         string                     `json:"from"`         // Source package path
	To           string                     `json:"to"`           // Target package path
	Dependencies map[DependencyType][]Edge  `json:"dependencies"` // Grouped by type
}

// Edge represents a single dependency edge between targets
type Edge struct {
	FromTarget string `json:"fromTarget"` // Source target label
	ToTarget   string `json:"toTarget"`   // Target dependency label
}

// Workspace represents the complete build graph
type Workspace struct {
	Targets      map[string]*Target `json:"targets"`      // Map of label -> Target
	Dependencies []Dependency       `json:"dependencies"` // All target-level dependencies
	Packages     map[string]*Package `json:"packages"`    // Map of package path -> Package
}

// GetPackageDependencies returns all dependencies for a given package
func (w *Workspace) GetPackageDependencies(packagePath string) []PackageDependency {
	// Map to aggregate dependencies by target package
	depsByPackage := make(map[string]*PackageDependency)

	// Iterate through all dependencies
	for _, dep := range w.Dependencies {
		fromTarget := w.Targets[dep.From]
		toTarget := w.Targets[dep.To]

		if fromTarget == nil || toTarget == nil {
			continue
		}

		// Only include if source is from our package
		if fromTarget.Package != packagePath {
			continue
		}

		// Skip dependencies within the same package
		if fromTarget.Package == toTarget.Package {
			continue
		}

		// Get or create package dependency
		pkgDep, exists := depsByPackage[toTarget.Package]
		if !exists {
			pkgDep = &PackageDependency{
				From:         packagePath,
				To:           toTarget.Package,
				Dependencies: make(map[DependencyType][]Edge),
			}
			depsByPackage[toTarget.Package] = pkgDep
		}

		// Add edge
		edge := Edge{
			FromTarget: dep.From,
			ToTarget:   dep.To,
		}
		pkgDep.Dependencies[dep.Type] = append(pkgDep.Dependencies[dep.Type], edge)
	}

	// Convert map to slice
	result := make([]PackageDependency, 0, len(depsByPackage))
	for _, pkgDep := range depsByPackage {
		result = append(result, *pkgDep)
	}

	return result
}

// GetAllPackageDependencies returns all package-to-package dependencies in the workspace
func (w *Workspace) GetAllPackageDependencies() []PackageDependency {
	// Map to aggregate dependencies by package pair
	depsByPair := make(map[string]*PackageDependency)

	for _, dep := range w.Dependencies {
		fromTarget := w.Targets[dep.From]
		toTarget := w.Targets[dep.To]

		if fromTarget == nil || toTarget == nil {
			continue
		}

		// Skip dependencies within the same package
		if fromTarget.Package == toTarget.Package {
			continue
		}

		// Create key for package pair
		key := fromTarget.Package + " -> " + toTarget.Package

		// Get or create package dependency
		pkgDep, exists := depsByPair[key]
		if !exists {
			pkgDep = &PackageDependency{
				From:         fromTarget.Package,
				To:           toTarget.Package,
				Dependencies: make(map[DependencyType][]Edge),
			}
			depsByPair[key] = pkgDep
		}

		// Add edge
		edge := Edge{
			FromTarget: dep.From,
			ToTarget:   dep.To,
		}
		pkgDep.Dependencies[dep.Type] = append(pkgDep.Dependencies[dep.Type], edge)
	}

	// Convert map to slice
	result := make([]PackageDependency, 0, len(depsByPair))
	for _, pkgDep := range depsByPair {
		result = append(result, *pkgDep)
	}

	return result
}
