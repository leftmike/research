package graph

// This file defines the in-memory graph store and the Go API used to load data
// into it. The parser is read-only, so nodes and relationships are created
// programmatically rather than via Cypher write clauses.

// Node is a vertex in the graph: a unique id, zero or more labels, and a set of
// properties.
type Node struct {
	ID     int64
	Labels []string
	Props  map[string]Value
}

// Relationship is a directed, typed edge connecting two nodes, with its own
// properties.
type Relationship struct {
	ID    int64
	Type  string
	Start *Node
	End   *Node
	Props map[string]Value
}

// Graph is an in-memory collection of nodes and relationships. The zero value
// is not usable; call New.
type Graph struct {
	nodes  map[int64]*Node
	rels   map[int64]*Relationship
	adjOut map[int64][]*Relationship // relationships keyed by start node id
	adjIn  map[int64][]*Relationship // relationships keyed by end node id
	nextID int64
}

// New returns an empty graph ready to be populated.
func New() *Graph {
	return &Graph{
		nodes:  make(map[int64]*Node),
		rels:   make(map[int64]*Relationship),
		adjOut: make(map[int64][]*Relationship),
		adjIn:  make(map[int64][]*Relationship),
	}
}

// AddNode creates a node with the given labels and properties and returns it.
// The labels and properties are copied so the caller may reuse the arguments.
func (g *Graph) AddNode(labels []string, props map[string]Value) *Node {
	n := &Node{
		ID:     g.nextID,
		Labels: append([]string(nil), labels...),
		Props:  copyProps(props),
	}
	g.nextID++
	g.nodes[n.ID] = n
	return n
}

// AddRelationship creates a directed relationship of the given type from start
// to end with the given properties and returns it. Both nodes must already
// belong to this graph.
func (g *Graph) AddRelationship(typ string, start, end *Node, props map[string]Value) *Relationship {
	r := &Relationship{
		ID:    g.nextID,
		Type:  typ,
		Start: start,
		End:   end,
		Props: copyProps(props),
	}
	g.nextID++
	g.rels[r.ID] = r
	g.adjOut[start.ID] = append(g.adjOut[start.ID], r)
	g.adjIn[end.ID] = append(g.adjIn[end.ID], r)
	return r
}

// Nodes returns the number of nodes in the graph.
func (g *Graph) Nodes() int { return len(g.nodes) }

// Relationships returns the number of relationships in the graph.
func (g *Graph) Relationships() int { return len(g.rels) }

// copyProps makes a shallow copy of a property map, returning nil for an empty
// input so nodes/relationships without properties share no backing map.
func copyProps(props map[string]Value) map[string]Value {
	if len(props) == 0 {
		return nil
	}
	m := make(map[string]Value, len(props))
	for k, v := range props {
		m[k] = v
	}
	return m
}

// hasLabel reports whether the node carries the given label.
func (n *Node) hasLabel(label string) bool {
	for _, l := range n.Labels {
		if l == label {
			return true
		}
	}
	return false
}

// hasLabels reports whether the node carries every label in the set.
func (n *Node) hasLabels(labels []string) bool {
	for _, l := range labels {
		if !n.hasLabel(l) {
			return false
		}
	}
	return true
}

// relStep is a relationship paired with the node reached by traversing it from
// a given starting node.
type relStep struct {
	rel  *Relationship
	node *Node
}

// step enumerates the relationships incident to node that match the requested
// direction and (optional) set of types, together with the node reached on the
// other end. An empty types slice matches any type.
//
// dirRight follows outgoing edges (start->end), dirLeft follows incoming edges,
// and dirBoth follows either.
func (g *Graph) step(node *Node, dir relDir, types []string) []relStep {
	var out []relStep
	if dir == dirRight || dir == dirBoth {
		for _, r := range g.adjOut[node.ID] {
			if relTypeMatches(r, types) {
				out = append(out, relStep{rel: r, node: r.End})
			}
		}
	}
	if dir == dirLeft || dir == dirBoth {
		for _, r := range g.adjIn[node.ID] {
			if relTypeMatches(r, types) {
				out = append(out, relStep{rel: r, node: r.Start})
			}
		}
	}
	return out
}

// relDir is the traversal direction for a relationship pattern.
type relDir int

const (
	dirRight relDir = iota // start -> end
	dirLeft                // end -> start
	dirBoth                // either direction
)

func relTypeMatches(r *Relationship, types []string) bool {
	if len(types) == 0 {
		return true
	}
	for _, t := range types {
		if r.Type == t {
			return true
		}
	}
	return false
}
