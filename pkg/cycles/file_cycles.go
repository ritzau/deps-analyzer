package cycles

import (
	"github.com/ritzau/deps-analyzer/pkg/graph"
)

// FileCycle represents a circular dependency between source files
type FileCycle struct {
	Files []string // List of file paths in the cycle
}

// FindFileCycles finds all circular dependencies in the file dependency graph
func FindFileCycles(fg *graph.FileGraph) []FileCycle {
	tarjan := NewTarjanSCC(fg.Graph())
	sccs := tarjan.FindSCCs()

	cycles := make([]FileCycle, 0)
	for _, scc := range sccs {
		// Convert node IDs back to file paths
		files := make([]string, 0, len(scc))
		for _, nodeID := range scc {
			if node := fg.GetNodeByID(nodeID); node != nil {
				files = append(files, node.Path)
			}
		}

		if len(files) > 1 {
			cycles = append(cycles, FileCycle{
				Files: files,
			})
		}
	}

	return cycles
}
