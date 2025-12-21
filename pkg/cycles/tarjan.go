package cycles

import (
	"gonum.org/v1/gonum/graph"
)

// TarjanSCC finds all strongly connected components using Tarjan's algorithm
type TarjanSCC struct {
	graph   graph.Directed
	index   int
	stack   []int64
	onStack map[int64]bool
	indices map[int64]int
	lowLink map[int64]int
	sccs    [][]int64
}

// NewTarjanSCC creates a new Tarjan SCC finder
func NewTarjanSCC(g graph.Directed) *TarjanSCC {
	return &TarjanSCC{
		graph:   g,
		index:   0,
		stack:   make([]int64, 0),
		onStack: make(map[int64]bool),
		indices: make(map[int64]int),
		lowLink: make(map[int64]int),
		sccs:    make([][]int64, 0),
	}
}

// FindSCCs finds all strongly connected components in the graph
func (t *TarjanSCC) FindSCCs() [][]int64 {
	nodes := t.graph.Nodes()
	for nodes.Next() {
		node := nodes.Node()
		if _, visited := t.indices[node.ID()]; !visited {
			t.strongConnect(node.ID())
		}
	}
	return t.sccs
}

// strongConnect performs the recursive Tarjan's algorithm
func (t *TarjanSCC) strongConnect(nodeID int64) {
	// Set the depth index for this node
	t.indices[nodeID] = t.index
	t.lowLink[nodeID] = t.index
	t.index++

	// Push node onto stack
	t.stack = append(t.stack, nodeID)
	t.onStack[nodeID] = true

	// Consider successors of node
	successors := t.graph.From(nodeID)
	for successors.Next() {
		successor := successors.Node()
		successorID := successor.ID()

		if _, visited := t.indices[successorID]; !visited {
			// Successor has not yet been visited; recurse on it
			t.strongConnect(successorID)
			t.lowLink[nodeID] = min(t.lowLink[nodeID], t.lowLink[successorID])
		} else if t.onStack[successorID] {
			// Successor is on stack and hence in the current SCC
			t.lowLink[nodeID] = min(t.lowLink[nodeID], t.indices[successorID])
		}
	}

	// If nodeID is a root node, pop the stack and create an SCC
	if t.lowLink[nodeID] == t.indices[nodeID] {
		scc := make([]int64, 0)
		for {
			w := t.stack[len(t.stack)-1]
			t.stack = t.stack[:len(t.stack)-1]
			t.onStack[w] = false
			scc = append(scc, w)
			if w == nodeID {
				break
			}
		}
		// Only add SCCs with more than one node (cycles)
		if len(scc) > 1 {
			t.sccs = append(t.sccs, scc)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
