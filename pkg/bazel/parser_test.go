package bazel

import (
	"testing"
)

func TestParseXML(t *testing.T) {
	tests := []struct {
		name      string
		xmlOutput string
		wantRules int
		wantErr   bool
	}{
		{
			name: "Valid Output",
			xmlOutput: `
				<query version="2">
					<rule class="cc_library" location="/workspace/BUILD:1:1" name="//pkg:lib">
						<string name="name" value="lib"/>
						<list name="srcs">
							<label value="//pkg:lib.cc"/>
						</list>
						<list name="hdrs">
							<label value="//pkg:lib.h"/>
						</list>
						<list name="deps">
							<label value="//other:dep"/>
						</list>
					</rule>
					<rule class="cc_binary" location="/workspace/BUILD:10:1" name="//pkg:bin">
						<string name="name" value="bin"/>
						<list name="srcs">
							<label value="//pkg:main.cc"/>
						</list>
						<list name="deps">
							<label value="//pkg:lib"/>
						</list>
					</rule>
					<source-file name="//pkg:lib.cc" location="/workspace/pkg/lib.cc:1:1"/>
				</query>`,
			wantRules: 2,
			wantErr:   false,
		},
		{
			name:      "Empty Output",
			xmlOutput: ``,
			wantRules: 0,
			wantErr:   true, // EOF usually returns error for unmarshal
		},
		{
			name: "No Rules",
			xmlOutput: `
				<query version="2">
					<source-file name="//pkg:lib.cc" location="/workspace/pkg/lib.cc:1:1"/>
				</query>`,
			wantRules: 0,
			wantErr:   false,
		},
		{
			name:      "Malformed XML",
			xmlOutput: `<query>...unclosed tags`,
			wantRules: 0,
			wantErr:   true,
		},
	}

	parser := NewParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph, err := parser.ParseQueryOutput([]byte(tt.xmlOutput))
			if (err != nil) != tt.wantErr {
				// Special handling for empty output/EOF which might vary slightly in error type but should fail
				if tt.wantErr && err == nil {
					t.Errorf("ParseQueryOutput() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !tt.wantErr && err != nil {
					// encoding/xml might return EOF for empty string
					if err.Error() == "EOF" && tt.xmlOutput == "" {
						// This is actually expected behavior for Unmarshal on empty string
						// But our wrapper might handle it differently.
						// Let's check if we expect error validation
					} else {
						t.Errorf("ParseQueryOutput() error = %v, wantErr %v", err, tt.wantErr)
						return
					}
				}
			}

			if graph != nil {
				// Count valid nodes (excluding skipped kinds)
				if len(graph.Nodes) != tt.wantRules {
					// Note: ParseQueryOutput ignores unknown kinds, while the test expectation
					// assumed raw rule count. But our test data only uses cc_library/cc_binary
					// which are accepted.
					t.Errorf("ParseQueryOutput() nodes count = %v, want %v", len(graph.Nodes), tt.wantRules)
				}
			}
		})
	}
}

// Test specific rule parsing details
func TestParseQueryOutput_RuleDetails(t *testing.T) {
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

	parser := NewParser()
	graph, err := parser.ParseQueryOutput([]byte(xmlOutput))
	if err != nil {
		t.Fatalf("ParseQueryOutput() unexpected error: %v", err)
	}

	if len(graph.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(graph.Nodes))
	}

	node, exists := graph.Nodes["//pkg:lib"]
	if !exists {
		t.Fatalf("Expected node //pkg:lib not found")
	}

	if node.Type != "cc_library" {
		t.Errorf("Expected type cc_library, got %s", node.Type)
	}

	// Check metadata
	if srcs, ok := node.Metadata["sources"].([]string); !ok || len(srcs) != 1 || srcs[0] != "//pkg:lib.cc" {
		t.Errorf("Expected source //pkg:lib.cc in metadata, got %v", node.Metadata["sources"])
	}

	// Check edges
	if len(graph.Edges) != 1 {
		t.Fatalf("Expected 1 edge, got %d", len(graph.Edges))
	}
	edge := graph.Edges[0]
	if edge.Source != "//pkg:lib" || edge.Target != "//other:dep" {
		t.Errorf("Expected edge //pkg:lib -> //other:dep, got %s -> %s", edge.Source, edge.Target)
	}
}
