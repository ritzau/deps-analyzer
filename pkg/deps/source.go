package deps

import (
	"context"

	"github.com/ritzau/deps-analyzer/pkg/analysis/api"
	"github.com/ritzau/deps-analyzer/pkg/config"
	"github.com/ritzau/deps-analyzer/pkg/logging"
	"github.com/ritzau/deps-analyzer/pkg/model"
)

// CompileDepsSource implements api.Source for compile-time dependencies (.d files)
type CompileDepsSource struct {
	client Client
}

// NewCompileDepsSource creates a new compile dependencies source
func NewCompileDepsSource() api.Source {
	return &CompileDepsSource{
		client: NewClient(),
	}
}

func (s *CompileDepsSource) Name() string {
	return "CompileDeps"
}

func (s *CompileDepsSource) Run(ctx context.Context, cfg *config.Config) (*model.Graph, error) {
	logger := logging.New("source.compile_deps")
	logger.Info("Starting compile dependencies analysis", "workspace", cfg.Workspace)

	// Reuse existing logic to parse all .d files via client
	deps, err := s.client.ParseAllDFiles(cfg.Workspace)
	if err != nil {
		return nil, err
	}

	logger.Info("Parsed .d files", "count", len(deps))

	graph := model.NewGraph()

	for _, dep := range deps {
		// Ensure source file node exists
		sourceNode := &model.Node{
			ID:    dep.SourceFile,
			Label: dep.SourceFile,
			Type:  "file",
			Metadata: map[string]interface{}{
				"file_type": "source",
			},
		}
		graph.AddNode(sourceNode)

		for _, depFile := range dep.Dependencies {
			// Ensure dependency file node exists
			// Note: We might want to deduplicate nodes across sources later,
			// but Graph.AddNode handles ID collisions by updating metadata (last write wins)
			depNode := &model.Node{
				ID:    depFile,
				Label: depFile,
				Type:  "file",
				Metadata: map[string]interface{}{
					"file_type": "header", // Assumption, could be source included
				},
			}
			graph.AddNode(depNode)

			// Add edge
			edge := &model.Edge{
				Source: dep.SourceFile,
				Target: depFile,
				Type:   "compile",
			}
			graph.AddEdge(edge)
		}
	}

	logger.Info("Compile deps analysis complete", "nodes", len(graph.Nodes), "edges", len(graph.Edges))
	return graph, nil
}
