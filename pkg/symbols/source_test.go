package symbols

import (
	"context"
	"testing"

	"github.com/ritzau/deps-analyzer/pkg/config"
)

// MockClient mocks the Client interface
type MockClient struct {
	// For BuildSymbolGraphInternal testing
	MockObjectFiles []string
	MockSymbols     map[string][]Symbol
	MockErr         error

	// For SymbolSource testing (high level mock)
	MockDeps []SymbolDependency
}

func (m *MockClient) FindObjectFiles(workspaceRoot string) ([]string, error) {
	return m.MockObjectFiles, m.MockErr
}

func (m *MockClient) RunNM(objectFile string) ([]Symbol, error) {
	if syms, ok := m.MockSymbols[objectFile]; ok {
		return syms, nil
	}
	return nil, nil
}

func (m *MockClient) BuildSymbolGraph(workspaceRoot string, fileToTarget map[string]string, targetToKind map[string]string) ([]SymbolDependency, error) {
	if m.MockDeps != nil {
		return m.MockDeps, m.MockErr
	}
	// Fallback to internal logic using the mock primitives
	return buildSymbolGraphInternal(m, workspaceRoot, fileToTarget, targetToKind)
}

func TestSymbolSource_Run(t *testing.T) {
	mockClient := &MockClient{
		MockDeps: []SymbolDependency{
			{SourceFile: "main.cc", TargetFile: "lib.cc", Symbol: "foo", Linkage: LinkageStatic},
		},
	}

	source := &SymbolSource{
		client: mockClient,
	}

	cfg := &config.Config{Workspace: "/tmp"}
	graph, err := source.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	if len(graph.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(graph.Edges))
	}
	edge := graph.Edges[0]
	if edge.Source != "main.cc" || edge.Target != "lib.cc" {
		t.Errorf("Edge mismatch: %v -> %v", edge.Source, edge.Target)
	}
}

func TestBuildSymbolGraphInternal(t *testing.T) {
	// Setup mocks
	// main.o -> U foo
	// lib.o  -> T foo
	mockClient := &MockClient{
		MockObjectFiles: []string{
			"bazel-out/bin/main/_objs/main/main.o",
			"bazel-out/bin/lib/_objs/lib/lib.o",
		},
		MockSymbols: map[string][]Symbol{
			"bazel-out/bin/main/_objs/main/main.o": {
				{Name: "foo", Type: "U"},
			},
			"bazel-out/bin/lib/_objs/lib/lib.o": {
				{Name: "foo", Type: "T", File: "bazel-out/bin/lib/_objs/lib/lib.o"},
			},
		},
	}

	deps, err := buildSymbolGraphInternal(mockClient, "/workspace", nil, nil)
	if err != nil {
		t.Fatalf("buildSymbolGraphInternal() error: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}

	dep := deps[0]
	// Note: objectFileToSourceFile conversion:
	// bazel-out/bin/main/_objs/main/main.o -> main/main.cc
	if dep.Symbol != "foo" {
		t.Errorf("Expected symbol foo, got %s", dep.Symbol)
	}
	// Verify source/target resolution
	// main.o -> main/main.cc
	// lib.o -> lib/lib.cc
	expectedSource := "main/main.cc"
	expectedTarget := "lib/lib.cc"
	if dep.SourceFile != expectedSource {
		t.Errorf("Expected source %s, got %s", expectedSource, dep.SourceFile)
	}
	if dep.TargetFile != expectedTarget {
		t.Errorf("Expected target %s, got %s", expectedTarget, dep.TargetFile)
	}
}
