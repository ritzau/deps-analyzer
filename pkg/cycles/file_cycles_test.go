package cycles

import (
	"testing"

	"github.com/ritzau/deps-analyzer/pkg/graph"
)

func TestFindFileCycles_NoCycles(t *testing.T) {
	fg := graph.NewFileGraph()

	// Create a simple acyclic dependency chain: A -> B -> C
	fg.AddFile("a.cc")
	fg.AddFile("b.h")
	fg.AddFile("c.h")
	fg.AddDependency("a.cc", "b.h")
	fg.AddDependency("b.h", "c.h")

	cycles := FindFileCycles(fg)

	if len(cycles) != 0 {
		t.Errorf("Expected no cycles, but found %d", len(cycles))
	}
}

func TestFindFileCycles_SimpleCycle(t *testing.T) {
	fg := graph.NewFileGraph()

	// Create a simple cycle: A -> B -> A
	fg.AddFile("a.h")
	fg.AddFile("b.h")
	fg.AddDependency("a.h", "b.h")
	fg.AddDependency("b.h", "a.h")

	cycles := FindFileCycles(fg)

	if len(cycles) != 1 {
		t.Fatalf("Expected 1 cycle, but found %d", len(cycles))
	}

	cycle := cycles[0]
	if len(cycle.Files) != 2 {
		t.Errorf("Expected cycle of length 2, got %d", len(cycle.Files))
	}

	// Check that both files are in the cycle
	filesInCycle := make(map[string]bool)
	for _, file := range cycle.Files {
		filesInCycle[file] = true
	}

	if !filesInCycle["a.h"] || !filesInCycle["b.h"] {
		t.Errorf("Expected cycle to contain a.h and b.h, got %v", cycle.Files)
	}
}

func TestFindFileCycles_ThreeNodeCycle(t *testing.T) {
	fg := graph.NewFileGraph()

	// Create a three-node cycle: A -> B -> C -> A
	fg.AddFile("a.h")
	fg.AddFile("b.h")
	fg.AddFile("c.h")
	fg.AddDependency("a.h", "b.h")
	fg.AddDependency("b.h", "c.h")
	fg.AddDependency("c.h", "a.h")

	cycles := FindFileCycles(fg)

	if len(cycles) != 1 {
		t.Fatalf("Expected 1 cycle, but found %d", len(cycles))
	}

	cycle := cycles[0]
	if len(cycle.Files) != 3 {
		t.Errorf("Expected cycle of length 3, got %d", len(cycle.Files))
	}
}

func TestFindFileCycles_MultipleCycles(t *testing.T) {
	fg := graph.NewFileGraph()

	// Create two separate cycles:
	// Cycle 1: A -> B -> A
	fg.AddFile("a.h")
	fg.AddFile("b.h")
	fg.AddDependency("a.h", "b.h")
	fg.AddDependency("b.h", "a.h")

	// Cycle 2: C -> D -> E -> C
	fg.AddFile("c.h")
	fg.AddFile("d.h")
	fg.AddFile("e.h")
	fg.AddDependency("c.h", "d.h")
	fg.AddDependency("d.h", "e.h")
	fg.AddDependency("e.h", "c.h")

	cycles := FindFileCycles(fg)

	if len(cycles) != 2 {
		t.Fatalf("Expected 2 cycles, but found %d", len(cycles))
	}

	// Check that one cycle has 2 files and the other has 3
	cycleSizes := make(map[int]int)
	for _, cycle := range cycles {
		cycleSizes[len(cycle.Files)]++
	}

	if cycleSizes[2] != 1 || cycleSizes[3] != 1 {
		t.Errorf("Expected one 2-node cycle and one 3-node cycle, got: %v", cycleSizes)
	}
}

func TestFindFileCycles_CycleWithAcyclicParts(t *testing.T) {
	fg := graph.NewFileGraph()

	// Create a graph with both cyclic and acyclic parts:
	// A -> B -> C (acyclic)
	// D -> E -> D (cyclic)
	fg.AddFile("a.cc")
	fg.AddFile("b.h")
	fg.AddFile("c.h")
	fg.AddDependency("a.cc", "b.h")
	fg.AddDependency("b.h", "c.h")

	fg.AddFile("d.h")
	fg.AddFile("e.h")
	fg.AddDependency("d.h", "e.h")
	fg.AddDependency("e.h", "d.h")

	cycles := FindFileCycles(fg)

	if len(cycles) != 1 {
		t.Fatalf("Expected 1 cycle, but found %d", len(cycles))
	}

	cycle := cycles[0]
	if len(cycle.Files) != 2 {
		t.Errorf("Expected cycle of length 2, got %d", len(cycle.Files))
	}
}
