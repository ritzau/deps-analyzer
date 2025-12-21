package graph

import (
	"path/filepath"
	"testing"

	"github.com/ritzau/deps-analyzer/pkg/bazel"
)

func TestNewTargetGraph(t *testing.T) {
	tg := NewTargetGraph()
	if tg == nil {
		t.Fatal("NewTargetGraph() returned nil")
	}

	if len(tg.Nodes()) != 0 {
		t.Errorf("New graph should have 0 nodes, got %d", len(tg.Nodes()))
	}
}

func TestAddTarget(t *testing.T) {
	tg := NewTargetGraph()

	target := bazel.Target{
		Label: "//util:util",
		Kind:  "cc_library",
	}

	tg.AddTarget(target)

	if len(tg.Nodes()) != 1 {
		t.Errorf("Expected 1 node, got %d", len(tg.Nodes()))
	}

	node, exists := tg.GetNode("//util:util")
	if !exists {
		t.Error("Target not found in graph")
	}

	if node.Label != "//util:util" {
		t.Errorf("Expected label //util:util, got %s", node.Label)
	}
}

func TestAddDependency(t *testing.T) {
	tg := NewTargetGraph()

	util := bazel.Target{Label: "//util:util", Kind: "cc_library"}
	core := bazel.Target{Label: "//core:core", Kind: "cc_library"}

	tg.AddTarget(util)
	tg.AddTarget(core)

	// core depends on util
	err := tg.AddDependency("//core:core", "//util:util")
	if err != nil {
		t.Fatalf("Failed to add dependency: %v", err)
	}

	// Check edge exists
	edges := tg.Edges()
	if len(edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(edges))
	}

	if edges[0][0] != "//core:core" || edges[0][1] != "//util:util" {
		t.Errorf("Expected edge core->util, got %v", edges[0])
	}
}

func TestGetDependencies(t *testing.T) {
	tg := NewTargetGraph()

	util := bazel.Target{Label: "//util:util", Kind: "cc_library"}
	core := bazel.Target{Label: "//core:core", Kind: "cc_library"}
	main := bazel.Target{Label: "//main:main", Kind: "cc_binary"}

	tg.AddTarget(util)
	tg.AddTarget(core)
	tg.AddTarget(main)

	// main depends on core and util
	tg.AddDependency("//main:main", "//core:core")
	tg.AddDependency("//main:main", "//util:util")

	deps := tg.GetDependencies("//main:main")
	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
	}

	// Check both deps are present
	depsMap := make(map[string]bool)
	for _, dep := range deps {
		depsMap[dep] = true
	}

	if !depsMap["//core:core"] || !depsMap["//util:util"] {
		t.Errorf("Expected core and util as dependencies, got %v", deps)
	}
}

func TestBuildTargetGraph(t *testing.T) {
	examplePath := filepath.Join("..", "..", "example")

	tg, err := BuildTargetGraph(examplePath)
	if err != nil {
		t.Fatalf("BuildTargetGraph() error = %v", err)
	}

	// Should have 4 targets
	nodes := tg.Nodes()
	if len(nodes) < 4 {
		t.Errorf("Expected at least 4 nodes, got %d", len(nodes))
	}

	// Check that expected targets exist
	expectedTargets := []string{"//util:util", "//core:core", "//plugins:renderer_plugin", "//main:test_app"}
	for _, expected := range expectedTargets {
		if _, exists := tg.GetNode(expected); !exists {
			t.Errorf("Expected target %s not found", expected)
		}
	}

	// Check that core depends on util
	coreDeps := tg.GetDependencies("//core:core")
	foundUtil := false
	for _, dep := range coreDeps {
		if dep == "//util:util" {
			foundUtil = true
			break
		}
	}
	if !foundUtil {
		t.Error("Expected //core:core to depend on //util:util")
	}

	// Check edges exist
	edges := tg.Edges()
	if len(edges) == 0 {
		t.Error("Expected at least some dependency edges")
	}
}
