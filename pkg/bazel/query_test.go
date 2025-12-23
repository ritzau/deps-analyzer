package bazel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ritzau/deps-analyzer/pkg/model"
)

func TestQueryWorkspace(t *testing.T) {
	// Find the example directory
	workspacePath := findExampleWorkspace(t)

	// Query the workspace
	workspace, err := QueryWorkspace(workspacePath)
	if err != nil {
		t.Fatalf("QueryWorkspace failed: %v", err)
	}

	// Verify we found targets
	if len(workspace.Targets) == 0 {
		t.Fatal("No targets found")
	}

	t.Logf("Found %d targets", len(workspace.Targets))

	// Verify specific targets exist
	requiredTargets := []struct {
		label string
		kind  model.TargetKind
	}{
		{"//main:test_app", model.TargetKindBinary},
		{"//core:core", model.TargetKindLibrary},
		{"//util:util", model.TargetKindLibrary},
		{"//graphics:graphics", model.TargetKindSharedLibrary},
		{"//audio:audio", model.TargetKindSharedLibrary},
	}

	for _, req := range requiredTargets {
		target, exists := workspace.Targets[req.label]
		if !exists {
			t.Errorf("Target %s not found", req.label)
			continue
		}
		if target.Kind != req.kind {
			t.Errorf("Target %s has wrong kind: got %s, want %s", req.label, target.Kind, req.kind)
		}
	}

	// Verify test_app has the expected dependencies
	testApp, exists := workspace.Targets["//main:test_app"]
	if !exists {
		t.Fatal("//main:test_app not found")
	}

	// Check deps (should have core, util, graphics_impl)
	expectedDeps := []string{"//core:core", "//util:util", "//graphics:graphics_impl"}
	if len(testApp.Deps) != len(expectedDeps) {
		t.Errorf("test_app deps: got %d, want %d", len(testApp.Deps), len(expectedDeps))
	}
	for _, dep := range expectedDeps {
		if !contains(testApp.Deps, dep) {
			t.Errorf("test_app missing dep: %s", dep)
		}
	}

	// Check dynamic_deps (should have graphics)
	if len(testApp.DynamicDeps) != 1 || testApp.DynamicDeps[0] != "//graphics:graphics" {
		t.Errorf("test_app dynamic_deps: got %v, want [//graphics:graphics]", testApp.DynamicDeps)
	}

	// Check data (should have audio)
	if len(testApp.Data) != 1 || testApp.Data[0] != "//audio:audio" {
		t.Errorf("test_app data: got %v, want [//audio:audio]", testApp.Data)
	}

	// Check linkopts (should include -ldl)
	if !contains(testApp.Linkopts, "-ldl") {
		t.Errorf("test_app missing -ldl in linkopts: %v", testApp.Linkopts)
	}

	// Verify dependencies are typed correctly
	t.Logf("Found %d dependencies", len(workspace.Dependencies))

	// Count by type
	byType := make(map[model.DependencyType]int)
	for _, dep := range workspace.Dependencies {
		byType[dep.Type]++
	}

	t.Logf("Dependency types: static=%d, dynamic=%d, data=%d",
		byType[model.DependencyStatic],
		byType[model.DependencyDynamic],
		byType[model.DependencyData])

	// Verify specific dependency types
	testCases := []struct {
		from string
		to   string
		typ  model.DependencyType
	}{
		{"//main:test_app", "//core:core", model.DependencyStatic},
		{"//main:test_app", "//graphics:graphics", model.DependencyDynamic},
		{"//main:test_app", "//audio:audio", model.DependencyData},
		{"//core:core", "//util:util", model.DependencyStatic},
	}

	for _, tc := range testCases {
		found := false
		for _, dep := range workspace.Dependencies {
			if dep.From == tc.from && dep.To == tc.to {
				found = true
				if dep.Type != tc.typ {
					t.Errorf("Dependency %s -> %s has wrong type: got %s, want %s",
						tc.from, tc.to, dep.Type, tc.typ)
				}
				break
			}
		}
		if !found {
			t.Errorf("Dependency not found: %s -> %s", tc.from, tc.to)
		}
	}

	// Test package-level dependencies
	t.Run("PackageDependencies", func(t *testing.T) {
		// Check packages were created
		if len(workspace.Packages) == 0 {
			t.Fatal("No packages found")
		}

		t.Logf("Found %d packages", len(workspace.Packages))

		// Get dependencies for //main package
		mainDeps := workspace.GetPackageDependencies("//main")
		t.Logf("//main has %d package dependencies", len(mainDeps))

		// Should depend on //core, //util, //graphics, //audio
		expectedDeps := map[string]bool{
			"//core":     false,
			"//util":     false,
			"//graphics": false,
			"//audio":    false,
		}

		for _, pkgDep := range mainDeps {
			if _, expected := expectedDeps[pkgDep.To]; expected {
				expectedDeps[pkgDep.To] = true
				t.Logf("  -> %s with %d edges", pkgDep.To, countEdges(pkgDep))
			}
		}

		for pkg, found := range expectedDeps {
			if !found {
				t.Errorf("//main missing expected dependency to %s", pkg)
			}
		}

		// Get all package dependencies
		allPkgDeps := workspace.GetAllPackageDependencies()
		t.Logf("Total package-to-package dependencies: %d", len(allPkgDeps))
	})
}

func countEdges(pkgDep model.PackageDependency) int {
	count := 0
	for _, edges := range pkgDep.Dependencies {
		count += len(edges)
	}
	return count
}

func findExampleWorkspace(t *testing.T) string {
	// Try relative path first (when running from pkg/bazel)
	examplePath := filepath.Join("..", "..", "example")
	if isWorkspace(examplePath) {
		return examplePath
	}

	// Start from current directory and walk up to find example/
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	for {
		examplePath = filepath.Join(dir, "example")
		if isWorkspace(examplePath) {
			return examplePath
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("Could not find example workspace (cwd=%s)", dir)
		}
		dir = parent
	}
}

func isWorkspace(path string) bool {
	if stat, err := os.Stat(path); err != nil || !stat.IsDir() {
		return false
	}

	// Check for WORKSPACE, WORKSPACE.bazel, or MODULE.bazel
	markers := []string{"WORKSPACE", "WORKSPACE.bazel", "MODULE.bazel"}
	for _, marker := range markers {
		if _, err := os.Stat(filepath.Join(path, marker)); err == nil {
			return true
		}
	}
	return false
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
