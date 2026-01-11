package deps

import (
	"context"
	"testing"

	"github.com/ritzau/deps-analyzer/pkg/config"
)

// MockClient is a mock implementation of Client
type MockClient struct {
	MockDeps []*FileDependency
	MockErr  error
}

func (m *MockClient) ParseAllDFiles(workspaceRoot string) ([]*FileDependency, error) {
	return m.MockDeps, m.MockErr
}

func TestCompileDepsSource_Run(t *testing.T) {
	mockClient := &MockClient{
		MockDeps: []*FileDependency{
			{
				SourceFile:   "pkg/main.cc",
				Dependencies: []string{"pkg/header.h", "util/util.h"},
			},
			{
				SourceFile:   "util/util.cc",
				Dependencies: []string{"util/util.h"},
			},
		},
	}

	source := &CompileDepsSource{
		client: mockClient,
	}

	cfg := &config.Config{Workspace: "/tmp"}
	graph, err := source.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	// Verify nodes
	// 4 nodes: main.cc, header.h, util.h, util.cc
	// Actually, header.h and util.h are added as nodes too.
	// main.cc -> header.h (edge)
	// main.cc -> util.h (edge)
	// util.cc -> util.h (edge)
	expectedNodes := map[string]bool{
		"pkg/main.cc":  false,
		"pkg/header.h": false,
		"util/util.h":  false,
		"util/util.cc": false,
	}

	if len(graph.Nodes) != len(expectedNodes) {
		t.Errorf("Expected %d nodes, got %d", len(expectedNodes), len(graph.Nodes))
	}

	for id := range expectedNodes {
		if _, ok := graph.Nodes[id]; !ok {
			t.Errorf("Node %s not found", id)
		}
	}

	// Verify edges
	if len(graph.Edges) != 3 {
		t.Errorf("Expected 3 edges, got %d", len(graph.Edges))
	}

	// Check for specific edge
	found := false
	for _, edge := range graph.Edges {
		if edge.Source == "pkg/main.cc" && edge.Target == "util/util.h" {
			found = true
			if edge.Type != "compile" {
				t.Errorf("Expected edge type compile, got %s", edge.Type)
			}
			break
		}
	}
	if !found {
		t.Error("Edge pkg/main.cc -> util/util.h not found")
	}
}
