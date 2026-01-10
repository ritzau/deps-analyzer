package model

// Graph represents a unified dependency graph containing nodes and edges.
// It serves as the common data model for all analysis sources and the visualization layer.
type Graph struct {
	Nodes map[string]*Node `json:"nodes"`
	Edges []*Edge          `json:"edges"`
}

// NewGraph creates a new empty graph.
func NewGraph() *Graph {
	return &Graph{
		Nodes: make(map[string]*Node),
		Edges: make([]*Edge, 0),
	}
}

// Node represents a vertex in the dependency graph.
// It can represent a Bazel target, a source file, a package, or a system library.
type Node struct {
	ID       string                 `json:"id"`
	Label    string                 `json:"label"`
	Type     string                 `json:"type"`             // e.g., "cc_library", "source", "package"
	Parent   string                 `json:"parent,omitempty"` // ID of the parent node (for hierarchy)
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Edge represents a directed connection between two nodes.
type Edge struct {
	Source   string                 `json:"source"`
	Target   string                 `json:"target"`
	Type     string                 `json:"type"` // e.g., "static", "dynamic", "compile", "symbol"
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AddNode adds a node to the graph. If a node with the same ID exists, it updates it.
func (g *Graph) AddNode(node *Node) {
	if node.Metadata == nil {
		node.Metadata = make(map[string]interface{})
	}
	g.Nodes[node.ID] = node
}

// AddEdge adds an edge to the graph.
func (g *Graph) AddEdge(edge *Edge) {
	if edge.Metadata == nil {
		edge.Metadata = make(map[string]interface{})
	}
	g.Edges = append(g.Edges, edge)
}
