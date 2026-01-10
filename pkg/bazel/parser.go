package bazel

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/ritzau/deps-analyzer/pkg/model"
)

// Parser handles parsing of Bazel output into the unified graph model
type Parser struct{}

// NewParser creates a new Bazel parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseQueryOutput parses the XML output from a Bazel query into a model.Graph
func (p *Parser) ParseQueryOutput(data []byte) (*model.Graph, error) {
	// Bazel outputs XML 1.1, but Go's XML parser only supports 1.0
	// Replace the version declaration
	xmlStr := string(data)
	xmlStr = strings.Replace(xmlStr, `<?xml version="1.1"`, `<?xml version="1.0"`, 1)

	var result QueryResult
	if err := xml.Unmarshal([]byte(xmlStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	graph := model.NewGraph()

	// 1. Create nodes
	for _, rule := range result.Rules {
		// Filter for relevant kinds
		kind := model.TargetKind(rule.Class)
		if !isRelevantKind(kind) {
			continue
		}

		node := &model.Node{
			ID:    rule.Name,
			Label: rule.Name,
			Type:  string(kind),
			Metadata: map[string]interface{}{
				"package": extractPackage(rule.Name),
				"kind":    rule.Class,
			},
		}

		// Parse attributes for metadata
		sources, headers := extractSources(rule)
		if len(sources) > 0 {
			node.Metadata["sources"] = sources
		}
		if len(headers) > 0 {
			node.Metadata["headers"] = headers
		}

		graph.AddNode(node)
	}

	// 2. Create edges
	for _, rule := range result.Rules {
		if _, exists := graph.Nodes[rule.Name]; !exists {
			continue
		}

		for _, list := range rule.Lists {
			switch list.Name {
			case "deps":
				for _, label := range list.Labels {
					edge := &model.Edge{
						Source: rule.Name,
						Target: label.Value,
						Type:   "static", // Default, might be dynamic if target is cc_shared_library
					}
					// Refine type if target is known
					if targetNode, ok := graph.Nodes[label.Value]; ok {
						if targetNode.Type == string(model.TargetKindSharedLibrary) {
							edge.Type = "dynamic"
						}
					}
					graph.AddEdge(edge)
				}
			case "dynamic_deps":
				for _, label := range list.Labels {
					graph.AddEdge(&model.Edge{
						Source: rule.Name,
						Target: label.Value,
						Type:   "dynamic",
					})
				}
			case "data":
				for _, label := range list.Labels {
					graph.AddEdge(&model.Edge{
						Source: rule.Name,
						Target: label.Value,
						Type:   "data",
					})
				}
			}
		}
	}

	return graph, nil
}

func isRelevantKind(kind model.TargetKind) bool {
	return kind == model.TargetKindBinary ||
		kind == model.TargetKindSharedLibrary ||
		kind == model.TargetKindLibrary
}

func extractPackage(label string) string {
	parts := strings.Split(label, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return label
}

func extractSources(rule RuleXML) ([]string, []string) {
	var sources, headers []string
	for _, list := range rule.Lists {
		if list.Name == "srcs" || list.Name == "hdrs" {
			for _, label := range list.Labels {
				if strings.HasSuffix(label.Value, ".cc") {
					sources = append(sources, label.Value)
				} else if strings.HasSuffix(label.Value, ".h") || strings.HasSuffix(label.Value, ".hpp") {
					headers = append(headers, label.Value)
				}
			}
		}
	}
	return sources, headers
}
