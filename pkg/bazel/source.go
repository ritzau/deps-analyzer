package bazel

import (
	"context"

	"github.com/ritzau/deps-analyzer/pkg/analysis/api"
	"github.com/ritzau/deps-analyzer/pkg/config"
	"github.com/ritzau/deps-analyzer/pkg/logging"
	"github.com/ritzau/deps-analyzer/pkg/model"
)

// TargetSource implements api.Source for Bazel targets
type TargetSource struct {
	executor Executor
	parser   *Parser
}

// NewTargetSource creates a new Bazel target source
func NewTargetSource() api.Source {
	return &TargetSource{
		executor: NewExecutor(),
		parser:   NewParser(),
	}
}

func (s *TargetSource) Name() string {
	return "BazelTarget"
}

func (s *TargetSource) Run(ctx context.Context, cfg *config.Config) (*model.Graph, error) {
	logger := logging.New("source.bazel")
	logger.Info("Starting Bazel query analysis", "workspace", cfg.Workspace)

	// Define the query
	query := "kind('cc_binary|cc_shared_library|cc_library', //...)"

	// Execute query
	output, err := s.executor.RunQuery(ctx, cfg.Workspace, query)
	if err != nil {
		return nil, err
	}

	logger.Info("Bazel query complete by executor", "bytes", len(output))

	// Parse output
	graph, err := s.parser.ParseQueryOutput(output)
	if err != nil {
		return nil, err
	}

	logger.Info("Bazel analysis complete", "nodes", len(graph.Nodes), "edges", len(graph.Edges))
	return graph, nil
}
