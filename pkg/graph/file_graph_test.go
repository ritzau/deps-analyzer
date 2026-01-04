package graph

import (
	"path/filepath"
	"testing"

	"github.com/ritzau/deps-analyzer/pkg/deps"
)

func TestNewFileGraph(t *testing.T) {
	fg := NewFileGraph()
	if fg == nil {
		t.Fatal("NewFileGraph() returned nil")
	}

	if len(fg.Nodes()) != 0 {
		t.Errorf("New graph should have 0 nodes, got %d", len(fg.Nodes()))
	}
}

func TestAddFile(t *testing.T) {
	fg := NewFileGraph()

	fg.AddFile("util/math.cc")

	if len(fg.Nodes()) != 1 {
		t.Errorf("Expected 1 node, got %d", len(fg.Nodes()))
	}

	node, exists := fg.GetNode("util/math.cc")
	if !exists {
		t.Error("File not found in graph")
	}

	if node.Path != "util/math.cc" {
		t.Errorf("Expected path util/math.cc, got %s", node.Path)
	}
}

func TestFileAddDependency(t *testing.T) {
	fg := NewFileGraph()

	fg.AddFile("util/math.cc")
	fg.AddFile("util/strings.h")

	err := fg.AddDependency("util/math.cc", "util/strings.h")
	if err != nil {
		t.Fatalf("Failed to add dependency: %v", err)
	}

	// Check edge exists
	edges := fg.Edges()
	if len(edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(edges))
	}

	if edges[0][0] != "util/math.cc" || edges[0][1] != "util/strings.h" {
		t.Errorf("Expected edge math.cc->strings.h, got %v", edges[0])
	}
}

func TestFileGetDependencies(t *testing.T) {
	fg := NewFileGraph()

	fg.AddFile("core/engine.cc")
	fg.AddFile("util/strings.h")
	fg.AddFile("util/time.h")

	_ = fg.AddDependency("core/engine.cc", "util/strings.h")
	_ = fg.AddDependency("core/engine.cc", "util/time.h")

	deps := fg.GetDependencies("core/engine.cc")
	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
	}

	// Check both deps are present
	depsMap := make(map[string]bool)
	for _, dep := range deps {
		depsMap[dep] = true
	}

	if !depsMap["util/strings.h"] || !depsMap["util/time.h"] {
		t.Errorf("Expected util/strings.h and util/time.h as dependencies, got %v", deps)
	}
}

func TestBuildFileGraph(t *testing.T) {
	examplePath := filepath.Join("..", "..", "example")

	fileDeps, err := deps.ParseAllDFiles(examplePath)
	if err != nil {
		t.Fatalf("ParseAllDFiles() error = %v", err)
	}

	fg := BuildFileGraph(fileDeps)

	// Should have multiple nodes (source files and headers)
	nodes := fg.Nodes()
	if len(nodes) < 4 {
		t.Errorf("Expected at least 4 nodes, got %d", len(nodes))
	}

	// Check that we have cross-package dependencies (core -> util)
	deps := fg.GetDependencies("core/engine.cc")
	foundCrossPackage := false
	for _, dep := range deps {
		// core/engine.cc should depend on util headers
		if dep == "util/strings.h" || dep == "util/time.h" {
			foundCrossPackage = true
			break
		}
	}

	if !foundCrossPackage {
		t.Error("Expected core/engine.cc to depend on util headers")
	}
}
