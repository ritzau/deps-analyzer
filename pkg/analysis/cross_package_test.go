package analysis

import (
	"path/filepath"
	"testing"

	"github.com/ritzau/deps-analyzer/pkg/deps"
	"github.com/ritzau/deps-analyzer/pkg/graph"
)

func TestFileToPackage(t *testing.T) {
	tests := []struct {
		filePath string
		expected string
	}{
		{"util/strings.h", "//util"},
		{"core/engine.cc", "//core"},
		{"main/main.cc", "//main"},
		{"plugins/renderer.h", "//plugins"},
	}

	for _, tt := range tests {
		result := fileToPackage(tt.filePath)
		if result != tt.expected {
			t.Errorf("fileToPackage(%q) = %q, want %q", tt.filePath, result, tt.expected)
		}
	}
}

func TestFindCrossPackageDeps(t *testing.T) {
	examplePath := filepath.Join("..", "..", "example")

	// Parse .d files and build file graph
	fileDeps, err := deps.ParseAllDFiles(examplePath)
	if err != nil {
		t.Fatalf("ParseAllDFiles() error = %v", err)
	}

	fg := graph.BuildFileGraph(fileDeps)

	// Find cross-package dependencies
	crossDeps := FindCrossPackageDeps(fg)

	// Should find at least one cross-package dependency (core -> util)
	if len(crossDeps) == 0 {
		t.Error("Expected at least one cross-package dependency")
	}

	// Check for specific cross-package dependencies we know exist
	found := make(map[string]bool)
	for _, dep := range crossDeps {
		key := dep.SourceFile + " -> " + dep.TargetFile
		found[key] = true

		// Verify package extraction is correct
		if dep.SourcePackage == dep.TargetPackage {
			t.Errorf("Cross-package dependency should have different packages: %v", dep)
		}
	}

	// core/engine.cc should depend on util headers
	expectedDeps := []string{
		"core/engine.cc -> util/strings.h",
		"core/engine.cc -> util/time.h",
	}

	for _, expected := range expectedDeps {
		if !found[expected] {
			t.Errorf("Expected cross-package dependency not found: %s", expected)
		}
	}
}
