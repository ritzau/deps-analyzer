package deps

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDFile(t *testing.T) {
	// Use the actual .d file from the example workspace
	examplePath := filepath.Join("..", "..", "example")
	dfilePath := filepath.Join(examplePath, "bazel-out", "darwin_x86_64-fastbuild", "bin", "util", "_objs", "util", "math.d")

	dep, err := ParseDFile(dfilePath)
	if err != nil {
		t.Fatalf("ParseDFile() error = %v", err)
	}

	if dep.SourceFile != "util/math.cc" {
		t.Errorf("Expected source file 'util/math.cc', got '%s'", dep.SourceFile)
	}

	// Check that dependencies include workspace files
	expectedDeps := map[string]bool{
		"util/math.h":    true,
		"util/strings.h": true,
	}

	foundDeps := make(map[string]bool)
	for _, dep := range dep.Dependencies {
		foundDeps[dep] = true
	}

	for expected := range expectedDeps {
		if !foundDeps[expected] {
			t.Errorf("Expected dependency '%s' not found. Got: %v", expected, dep.Dependencies)
		}
	}

	// Should not include system headers
	for _, dep := range dep.Dependencies {
		if filepath.IsAbs(dep) {
			t.Errorf("Should not include absolute path (system header): %s", dep)
		}
	}
}

func TestParseDFileCrossPackage(t *testing.T) {
	// Test core/engine.d which has cross-package dependencies
	examplePath := filepath.Join("..", "..", "example")
	dfilePath := filepath.Join(examplePath, "bazel-out", "darwin_x86_64-fastbuild", "bin", "core", "_objs", "core", "engine.d")

	dep, err := ParseDFile(dfilePath)
	if err != nil {
		t.Fatalf("ParseDFile() error = %v", err)
	}

	if dep.SourceFile != "core/engine.cc" {
		t.Errorf("Expected source file 'core/engine.cc', got '%s'", dep.SourceFile)
	}

	// Check for cross-package dependencies
	expectedDeps := map[string]bool{
		"core/engine.h":  true,
		"util/strings.h": true,
		"util/time.h":    true,
	}

	foundDeps := make(map[string]bool)
	for _, dep := range dep.Dependencies {
		foundDeps[dep] = true
	}

	for expected := range expectedDeps {
		if !foundDeps[expected] {
			t.Errorf("Expected dependency '%s' not found. Got: %v", expected, dep.Dependencies)
		}
	}
}

func TestIsWorkspaceFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"util/strings.h", true},
		{"core/engine.cc", true},
		{"/usr/include/stdio.h", false},
		{"external/some_dep/header.h", false},
		{"bazel-out/darwin_x86_64/bin/util/util.o", false},
		{"/Applications/Xcode.app/Developer/SDKs/MacOSX.sdk/usr/include/c++/v1/string", false},
	}

	for _, tt := range tests {
		result := isWorkspaceFile(tt.path)
		if result != tt.expected {
			t.Errorf("isWorkspaceFile(%q) = %v, want %v", tt.path, result, tt.expected)
		}
	}
}

func TestFindDFiles(t *testing.T) {
	examplePath := filepath.Join("..", "..", "example")

	dfiles, err := FindDFiles(examplePath)
	if err != nil {
		t.Fatalf("FindDFiles() error = %v", err)
	}

	// Should find at least 4 .d files (util, core targets)
	if len(dfiles) < 4 {
		t.Errorf("Expected at least 4 .d files, got %d", len(dfiles))
	}

	// All should be .d files
	for _, dfile := range dfiles {
		if filepath.Ext(dfile) != ".d" {
			t.Errorf("Expected .d file, got: %s", dfile)
		}
	}
}

func TestParseAllDFiles(t *testing.T) {
	examplePath := filepath.Join("..", "..", "example")

	deps, err := ParseAllDFiles(examplePath)
	if err != nil {
		t.Fatalf("ParseAllDFiles() error = %v", err)
	}

	// Should find dependencies for multiple source files
	if len(deps) < 4 {
		t.Errorf("Expected at least 4 file dependencies, got %d", len(deps))
	}

	// Check that we have at least one cross-package dependency (core -> util)
	foundCrossPackage := false
	for _, dep := range deps {
		if strings.HasPrefix(dep.SourceFile, "core/") {
			for _, depFile := range dep.Dependencies {
				if strings.HasPrefix(depFile, "util/") {
					foundCrossPackage = true
					break
				}
			}
		}
	}

	if !foundCrossPackage {
		t.Error("Expected to find at least one cross-package dependency (core -> util)")
	}
}

