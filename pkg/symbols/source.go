package symbols

import (
	"context"

	"github.com/ritzau/deps-analyzer/pkg/analysis/api"
	"github.com/ritzau/deps-analyzer/pkg/config"
	"github.com/ritzau/deps-analyzer/pkg/logging"
	"github.com/ritzau/deps-analyzer/pkg/model"
)

// SymbolSource implements api.Source for symbol-level dependencies (nm analysis)
type SymbolSource struct {
	client Client
}

// NewSymbolSource creates a new symbol dependencies source
func NewSymbolSource() api.Source {
	return &SymbolSource{
		client: NewClient(),
	}
}

func (s *SymbolSource) Name() string {
	return "SymbolDeps"
}

func (s *SymbolSource) Run(ctx context.Context, cfg *config.Config) (*model.Graph, error) {
	logger := logging.New("source.symbols")
	logger.Info("Starting symbol dependency analysis", "workspace", cfg.Workspace)

	// Note: We currently pass nil/nil for fileToTarget and targetToKind maps.
	// This means we won't calculate linkage types (Static/Dynamic) in this isolated mode.
	// To support that, we'd need to share target context between sources.
	symbolDeps, err := s.client.BuildSymbolGraph(cfg.Workspace, nil, nil)
	if err != nil {
		return nil, err
	}

	logger.Info("Found symbol dependencies", "count", len(symbolDeps))

	graph := model.NewGraph()

	for _, dep := range symbolDeps {
		if dep.SourceFile == "" || dep.TargetFile == "" {
			continue
		}

		// Ensure nodes exist
		graph.AddNode(&model.Node{
			ID:    dep.SourceFile,
			Label: dep.SourceFile,
			Type:  "file",
		})
		graph.AddNode(&model.Node{
			ID:    dep.TargetFile,
			Label: dep.TargetFile,
			Type:  "file",
		})

		// Add edge (avoid self-loops if source==target)
		if dep.SourceFile != dep.TargetFile {
			graph.AddEdge(&model.Edge{
				Source: dep.SourceFile,
				Target: dep.TargetFile,
				Type:   "symbol",
				Metadata: map[string]interface{}{
					"symbol": dep.Symbol,
				},
			})
		}
	}

	logger.Info("Symbol analysis complete", "nodes", len(graph.Nodes), "edges", len(graph.Edges))
	return graph, nil
}
