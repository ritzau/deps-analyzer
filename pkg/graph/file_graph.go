package graph

import (
	"github.com/ritzau/deps-analyzer/pkg/deps"
	"gonum.org/v1/gonum/graph/simple"
)

// FileNode represents a source file in the dependency graph
type FileNode struct {
	Path string // e.g., "util/math.cc" or "util/strings.h"
}

// FileGraph represents the file-level dependency graph
type FileGraph struct {
	graph  *simple.DirectedGraph
	nodes  map[string]*FileNode // Map from file path to node
	ids    map[string]int64     // Map from file path to graph ID
	nextID int64
}

// NewFileGraph creates a new file dependency graph
func NewFileGraph() *FileGraph {
	return &FileGraph{
		graph:  simple.NewDirectedGraph(),
		nodes:  make(map[string]*FileNode),
		ids:    make(map[string]int64),
		nextID: 0,
	}
}

// AddFile adds a file to the graph
func (fg *FileGraph) AddFile(path string) {
	if _, exists := fg.nodes[path]; exists {
		return
	}

	node := &FileNode{Path: path}
	fg.nodes[path] = node
	fg.ids[path] = fg.nextID

	// Add node to gonum graph
	fg.graph.AddNode(simple.Node(fg.nextID))

	fg.nextID++
}

// AddDependency adds a dependency edge from source to target
// Returns error if either file doesn't exist in the graph
func (fg *FileGraph) AddDependency(source, target string) error {
	// Ensure both nodes exist
	fg.AddFile(source)
	fg.AddFile(target)

	sourceID := fg.ids[source]
	targetID := fg.ids[target]

	// Add edge if it doesn't already exist
	if !fg.graph.HasEdgeFromTo(sourceID, targetID) {
		edge := fg.graph.NewEdge(fg.graph.Node(sourceID), fg.graph.Node(targetID))
		fg.graph.SetEdge(edge)
	}

	return nil
}

// GetNode returns a file node by path
func (fg *FileGraph) GetNode(path string) (*FileNode, bool) {
	node, exists := fg.nodes[path]
	return node, exists
}

// GetNodeByID returns a file node by its graph ID
func (fg *FileGraph) GetNodeByID(id int64) *FileNode {
	for path, nodeID := range fg.ids {
		if nodeID == id {
			return fg.nodes[path]
		}
	}
	return nil
}

// Graph returns the underlying directed graph
func (fg *FileGraph) Graph() *simple.DirectedGraph {
	return fg.graph
}

// Nodes returns all file nodes in the graph
func (fg *FileGraph) Nodes() []*FileNode {
	nodes := make([]*FileNode, 0, len(fg.nodes))
	for _, node := range fg.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// Edges returns all dependency edges as [source, target] pairs
func (fg *FileGraph) Edges() [][2]string {
	var edges [][2]string

	iter := fg.graph.Edges()
	for iter.Next() {
		edge := iter.Edge()
		sourceID := edge.From().ID()
		targetID := edge.To().ID()

		// Find the file paths for these IDs
		var sourcePath, targetPath string
		for path, id := range fg.ids {
			if id == sourceID {
				sourcePath = path
			}
			if id == targetID {
				targetPath = path
			}
		}

		edges = append(edges, [2]string{sourcePath, targetPath})
	}

	return edges
}

// GetDependencies returns all files that the given file depends on
func (fg *FileGraph) GetDependencies(path string) []string {
	id, exists := fg.ids[path]
	if !exists {
		return nil
	}

	node := fg.graph.Node(id)
	if node == nil {
		return nil
	}

	var deps []string
	iter := fg.graph.From(id)
	for iter.Next() {
		targetID := iter.Node().ID()
		for path, id := range fg.ids {
			if id == targetID {
				deps = append(deps, path)
				break
			}
		}
	}

	return deps
}

// BuildFileGraph builds a file dependency graph from .d file data
func BuildFileGraph(fileDeps []*deps.FileDependency) *FileGraph {
	fg := NewFileGraph()

	for _, dep := range fileDeps {
		// Add the source file
		if dep.SourceFile != "" {
			fg.AddFile(dep.SourceFile)

			// Add dependencies
			for _, depFile := range dep.Dependencies {
				fg.AddDependency(dep.SourceFile, depFile)
			}
		}
	}

	return fg
}

