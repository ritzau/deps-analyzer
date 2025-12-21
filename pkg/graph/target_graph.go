package graph

import (
	"fmt"

	"github.com/ritzau/deps-analyzer/pkg/bazel"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

// TargetNode represents a Bazel target in the graph
type TargetNode struct {
	id    int64
	Label string // e.g., "//util:util"
	Kind  string // e.g., "cc_library"
}

func (n TargetNode) ID() int64 { return n.id }

// TargetGraph represents a directed graph of Bazel target dependencies
type TargetGraph struct {
	graph *simple.DirectedGraph
	nodes map[string]*TargetNode // label -> node
	ids   map[string]int64       // label -> id
}

// NewTargetGraph creates a new target dependency graph
func NewTargetGraph() *TargetGraph {
	return &TargetGraph{
		graph: simple.NewDirectedGraph(),
		nodes: make(map[string]*TargetNode),
		ids:   make(map[string]int64),
	}
}

// AddTarget adds a target to the graph
func (tg *TargetGraph) AddTarget(target bazel.Target) {
	if _, exists := tg.nodes[target.Label]; exists {
		return // Already added
	}

	id := int64(len(tg.nodes))
	node := &TargetNode{
		id:    id,
		Label: target.Label,
		Kind:  target.Kind,
	}

	tg.graph.AddNode(node)
	tg.nodes[target.Label] = node
	tg.ids[target.Label] = id
}

// AddDependency adds a dependency edge from -> to
func (tg *TargetGraph) AddDependency(from, to string) error {
	fromNode, fromExists := tg.nodes[from]
	toNode, toExists := tg.nodes[to]

	if !fromExists {
		return fmt.Errorf("source target not found: %s", from)
	}
	if !toExists {
		return fmt.Errorf("destination target not found: %s", to)
	}

	edge := tg.graph.NewEdge(fromNode, toNode)
	tg.graph.SetEdge(edge)
	return nil
}

// GetNode returns the node for a given target label
func (tg *TargetGraph) GetNode(label string) (*TargetNode, bool) {
	node, exists := tg.nodes[label]
	return node, exists
}

// Nodes returns all nodes in the graph
func (tg *TargetGraph) Nodes() []*TargetNode {
	var nodes []*TargetNode
	for _, node := range tg.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// Edges returns all dependency edges as (from, to) pairs
func (tg *TargetGraph) Edges() [][2]string {
	var edges [][2]string

	it := tg.graph.Edges()
	for it.Next() {
		edge := it.Edge()
		fromNode := edge.From().(*TargetNode)
		toNode := edge.To().(*TargetNode)
		edges = append(edges, [2]string{fromNode.Label, toNode.Label})
	}

	return edges
}

// GetDependencies returns the targets that a given target depends on
func (tg *TargetGraph) GetDependencies(label string) []string {
	node, exists := tg.nodes[label]
	if !exists {
		return nil
	}

	var deps []string
	it := tg.graph.From(node.ID())
	for it.Next() {
		depNode := it.Node().(*TargetNode)
		deps = append(deps, depNode.Label)
	}

	return deps
}

// GetReverseDependencies returns the targets that depend on a given target
func (tg *TargetGraph) GetReverseDependencies(label string) []string {
	node, exists := tg.nodes[label]
	if !exists {
		return nil
	}

	var revDeps []string
	it := tg.graph.To(node.ID())
	for it.Next() {
		depNode := it.Node().(*TargetNode)
		revDeps = append(revDeps, depNode.Label)
	}

	return revDeps
}

// Graph returns the underlying gonum graph for advanced analysis
func (tg *TargetGraph) Graph() graph.Directed {
	return tg.graph
}

// BuildTargetGraph builds a complete target dependency graph from Bazel queries
func BuildTargetGraph(workspace string) (*TargetGraph, error) {
	// Query all CC targets
	targets, err := bazel.QueryAllCCTargets(workspace)
	if err != nil {
		return nil, fmt.Errorf("querying targets: %w", err)
	}

	tg := NewTargetGraph()

	// Add all targets as nodes
	for _, target := range targets {
		tg.AddTarget(target)
	}

	// Query and add dependencies for each target
	for _, target := range targets {
		deps, err := bazel.QueryDeps(workspace, target.Label)
		if err != nil {
			return nil, fmt.Errorf("querying deps for %s: %w", target.Label, err)
		}

		for _, dep := range deps {
			// Only add edges for targets in our workspace (not external deps)
			if _, exists := tg.nodes[dep]; exists {
				if err := tg.AddDependency(target.Label, dep); err != nil {
					// Skip errors for missing targets (might be external)
					continue
				}
			}
		}
	}

	return tg, nil
}
