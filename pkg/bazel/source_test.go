package bazel

import (
	"context"
	"testing"

	"github.com/ritzau/deps-analyzer/pkg/config"
)

func TestTargetSource_Run(t *testing.T) {
	// Sample XML output
	xmlOutput := `
		<query version="2">
			<rule class="cc_library" location="/workspace/BUILD:1:1" name="//pkg:lib">
				<string name="name" value="lib"/>
				<list name="srcs">
					<label value="//pkg:lib.cc"/>
				</list>
				<list name="deps">
					<label value="//other:dep"/>
				</list>
			</rule>
		</query>`

	mockExecutor := &MockExecutor{
		MockOutput: []byte(xmlOutput),
	}

	source := &TargetSource{
		executor: mockExecutor,
		parser:   NewParser(),
	}

	cfg := &config.Config{
		Workspace: "/tmp/workspace",
	}

	graph, err := source.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	if graph == nil {
		t.Fatal("Expected graph, got nil")
	}

	if len(graph.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(graph.Nodes))
	}

	if _, ok := graph.Nodes["//pkg:lib"]; !ok {
		t.Error("Node //pkg:lib not found in graph")
	}
}
