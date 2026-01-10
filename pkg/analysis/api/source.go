package api

import (
	"context"

	"github.com/ritzau/deps-analyzer/pkg/config"
	"github.com/ritzau/deps-analyzer/pkg/model"
)

// Source represents a data source for the dependency graph.
// Implementations should encapsulate the logic for gathering data (e.g., running Bazel, parsing files)
// and transforming it into the unified Graph model.
type Source interface {
	// Name returns the unique name of the source (e.g., "BazelQuery", "CompileDeps").
	Name() string

	// Run executes the analysis and returns a partial graph.
	// It should respect the context for cancellation.
	Run(ctx context.Context, cfg *config.Config) (*model.Graph, error)
}
