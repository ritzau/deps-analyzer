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
	DependencyCompile DependencyType = "compile" // Compile-time header dependency (from .d files)
	DependencySymbol  DependencyType = "symbol"  // Symbol-level linkage dependency (from nm analysis)
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

	// Visibility control
	Visibility []string `json:"visibility,omitempty"` // Visibility specifications (e.g., ["//visibility:public"])

	// System library linking options (not represented as Dependencies)
	Linkopts []string `json:"linkopts,omitempty"` // linkopts (for system libraries like -ldl)
}

// IsPublic returns true if the target has public visibility
func (t *Target) IsPublic() bool {
	for _, vis := range t.Visibility {
		if vis == "//visibility:public" {
			return true
		}
	}
	return false
}

// IsPrivate returns true if the target has private visibility or no visibility specified
func (t *Target) IsPrivate() bool {
	if len(t.Visibility) == 0 {
		return true // Default is private
	}
	for _, vis := range t.Visibility {
		if vis == "//visibility:private" {
			return true
		}
	}
	return false
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

// DependencyIssue represents a problem with dependencies
type DependencyIssue struct {
	From        string   `json:"from"`        // Source target label
	To          string   `json:"to"`          // Target dependency label
	Issue       string   `json:"issue"`       // Description of the issue
	Types       []string `json:"types"`       // Conflicting dependency types
	Severity    string   `json:"severity"`    // "warning" or "error"
	Description string   `json:"description"` // Detailed explanation
}

// Module represents the complete build graph (a Bazel workspace/module)
type Module struct {
	Name         string             `json:"name"`         // Workspace/module name
	Targets      map[string]*Target `json:"targets"`      // Map of label -> Target
	Dependencies []Dependency       `json:"dependencies"` // All target-level dependencies
	Issues       []DependencyIssue  `json:"issues"`       // Dependency issues/warnings
}

// GetPackages derives the package structure from targets
func (m *Module) GetPackages() map[string]*Package {
	packages := make(map[string]*Package)

	for _, target := range m.Targets {
		pkg, exists := packages[target.Package]
		if !exists {
			pkg = &Package{
				Path:    target.Package,
				Targets: make(map[string]*Target),
			}
			packages[target.Package] = pkg
		}
		pkg.Targets[target.Name] = target
	}

	return packages
}

// GetPackageCount returns the number of unique packages
func (m *Module) GetPackageCount() int {
	packageSet := make(map[string]bool)
	for _, target := range m.Targets {
		packageSet[target.Package] = true
	}
	return len(packageSet)
}

// GetPackageDependencies returns all dependencies for a given package
func (m *Module) GetPackageDependencies(packagePath string) []PackageDependency {
	// Map to aggregate dependencies by target package
	depsByPackage := make(map[string]*PackageDependency)

	// Iterate through all dependencies
	for _, dep := range m.Dependencies {
		fromTarget := m.Targets[dep.From]
		toTarget := m.Targets[dep.To]

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

// GetAllPackageDependencies returns all package-to-package dependencies in the module
func (m *Module) GetAllPackageDependencies() []PackageDependency {
	// Map to aggregate dependencies by package pair
	depsByPair := make(map[string]*PackageDependency)

	for _, dep := range m.Dependencies {
		fromTarget := m.Targets[dep.From]
		toTarget := m.Targets[dep.To]

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
