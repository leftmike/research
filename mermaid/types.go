// Command mermaid is a self-contained terminal renderer for Mermaid diagrams.
//
// It renders graph/flowchart, stateDiagram, classDiagram, erDiagram, and
// sequenceDiagram blocks as Unicode box-drawing art; unsupported diagram types
// fall back to the raw source in a framed box. It is a Go port of the Rust
// renderer at xai-org/grok-build crates/codegen/xai-grok-markdown/src/mermaid.rs.
package main

// Layout tunables (columns/rows are terminal cells).
const (
	maxLabel      = 28 // edge/relationship labels truncate to this width
	pad           = 1  // horizontal padding inside a box
	gapX          = 3  // minimum horizontal gap between boxes
	gapY          = 2  // minimum vertical gap between ranks
	wrapWidth     = 24 // node labels wrap to at most this many columns per line
	maxLines      = 4  // ...and at most this many lines (overflow truncated with …)
	maxNodes      = 128
	maxEdges      = 512
	maxGroups     = 24
	maxGroupDepth = 6
	maxCanvasCell = 1 << 21
	entityLookahd = 10
	maxMembers    = 8
	seqGap        = 5
)

// labelBreakChars are identifier-boundary characters preferred as break points
// when a single word is too wide to fit, so it is not sliced mid-segment.
const labelBreakChars = "_-./"

// cont marks the trailing column of a wide glyph (never emitted to output).
const cont = '\x00'

// Edge-bit flags recording which of the four neighbours a cell connects to.
const (
	bU uint8 = 1 << iota // up
	bD                   // down
	bL                   // left
	bR                   // right
)

// Line-style flags accumulated per cell so a shared run picks a consistent glyph.
const (
	styDot   uint8 = 1
	styThick uint8 = 2
	stySolid uint8 = 4
)

// Node shapes.
const (
	shapeRect = iota
	shapeRound
	shapeDiamond
)

// Edge head decorations.
const (
	headNone = iota
	headArrow
	headCircle
	headCross
	headTriangle
	headDiamondFill
	headDiamondOpen
)

// Edge line kinds.
const (
	lineSolid = iota
	lineDotted
	lineThick
)

// Flow directions.
const (
	dirDown = iota
	dirUp
	dirRight
	dirLeft
)

// Cell style classes, mapped to colors when rendering to a terminal.
const (
	clsEmpty = iota
	clsBorder
	clsText
	clsEdge
	clsEdgeLabel
)

type node struct {
	label string
	shape int
}

type edge struct {
	from, to int
	label    string
	hasLabel bool
	headTo   int
	headFrom int
	line     int
}

type group struct {
	id     string
	label  string
	parent int // index into graph.groups, or -1
}

// graph is the shared model for flowchart, state, class, and ER diagrams.
type graph struct {
	nodes     []node
	edges     []edge
	index     map[string]int
	groups    []group
	nodeGroup []int // group index per node, or -1
	curGroup  int   // -1 when not inside a subgraph
	overCap   bool
	dir       int
}

func newGraph(dir int) *graph {
	return &graph{index: map[string]int{}, curGroup: -1, dir: dir}
}

// nodeIndex returns the index of node id, creating it if absent. A non-empty
// label overwrites the stored label and shape. Reports false when the node cap
// is exceeded.
func (g *graph) nodeIndex(id string, label string, hasLabel bool, shape int) (int, bool) {
	if i, ok := g.index[id]; ok {
		if hasLabel {
			g.nodes[i].label = label
			g.nodes[i].shape = shape
		}
		return i, true
	}
	if len(g.nodes) >= maxNodes {
		g.overCap = true
		return 0, false
	}
	if !hasLabel {
		label = id
	}
	g.index[id] = len(g.nodes)
	g.nodes = append(g.nodes, node{label: label, shape: shape})
	g.nodeGroup = append(g.nodeGroup, g.curGroup)
	return len(g.nodes) - 1, true
}

// nodeLabel sets (or creates with) a rounded node whose label is label.
func (g *graph) nodeLabel(id, label string) (int, bool) {
	if i, ok := g.index[id]; ok {
		g.nodes[i].label = label
		return i, true
	}
	return g.nodeIndex(id, label, true, shapeRound)
}
