// Package symbol holds the symbol table a diagram parser produces: the nodes,
// groups and edges a scenario is allowed to reference.
//
// This is the seam that keeps animation diagram-agnostic. A diagram parser's
// only obligation is to emit a Table; scenario validation and timeline
// compilation work against Table alone and never inspect diagram syntax.
package symbol

import (
	"strconv"

	"github.com/tejaspanse/cinegram/pkg/source"
)

// Node is an addressable element of a diagram.
type Node struct {
	ID    string     `json:"id"`
	Label string     `json:"label"`
	Shape string     `json:"shape,omitempty"`
	Group string     `json:"group,omitempty"`
	Class string     `json:"class,omitempty"`
	At    source.Pos `json:"-"`
}

// Group is a container of nodes and other groups (a Mermaid subgraph).
type Group struct {
	ID       string     `json:"id"`
	Label    string     `json:"label"`
	Parent   string     `json:"parent,omitempty"`
	Children []string   `json:"children"`
	At       source.Pos `json:"-"`
}

// Edge is a connection between two nodes.
type Edge struct {
	ID    string     `json:"id"`
	From  string     `json:"from"`
	To    string     `json:"to"`
	Label string     `json:"label,omitempty"`
	Style string     `json:"style,omitempty"`
	Head  string     `json:"head,omitempty"`
	Bidir bool       `json:"bidir,omitempty"`
	At    source.Pos `json:"-"`
}

// Table is the set of symbols a diagram exposes to scenarios.
type Table struct {
	nodes      map[string]*Node
	nodeOrder  []string
	groups     map[string]*Group
	groupOrder []string
	edges      []*Edge
	byPair     map[[2]string][]*Edge
}

// NewTable returns an empty symbol table.
func NewTable() *Table {
	return &Table{
		nodes:  make(map[string]*Node),
		groups: make(map[string]*Group),
		byPair: make(map[[2]string][]*Edge),
	}
}

// EnsureNode returns the node with the given id, creating a placeholder whose
// label defaults to the id if it has not been seen. Mermaid lets a node be
// referenced before (or without) an explicit declaration, so this is the normal
// path rather than an error case.
func (t *Table) EnsureNode(id string, at source.Pos) *Node {
	if n, ok := t.nodes[id]; ok {
		return n
	}
	n := &Node{ID: id, Label: id, At: at}
	t.nodes[id] = n
	t.nodeOrder = append(t.nodeOrder, id)
	return n
}

// DeclareNode records an explicit declaration, upgrading any placeholder.
// A later declaration with a real label wins over an implicit one.
func (t *Table) DeclareNode(id, label, shape, class string, at source.Pos) *Node {
	n := t.EnsureNode(id, at)
	if label != "" {
		n.Label = label
	}
	if shape != "" {
		n.Shape = shape
	}
	if class != "" {
		n.Class = class
	}
	return n
}

// Node looks up a node by id.
func (t *Table) Node(id string) (*Node, bool) {
	n, ok := t.nodes[id]
	return n, ok
}

// Nodes returns every node in declaration order.
func (t *Table) Nodes() []*Node {
	out := make([]*Node, 0, len(t.nodeOrder))
	for _, id := range t.nodeOrder {
		out = append(out, t.nodes[id])
	}
	return out
}

// DeclareGroup records a subgraph.
func (t *Table) DeclareGroup(id, label, parent string, at source.Pos) *Group {
	if g, ok := t.groups[id]; ok {
		return g
	}
	g := &Group{ID: id, Label: label, Parent: parent, At: at}
	t.groups[id] = g
	t.groupOrder = append(t.groupOrder, id)
	if parent != "" {
		if p, ok := t.groups[parent]; ok {
			p.Children = append(p.Children, id)
		}
	}
	return g
}

// AddToGroup records that node id belongs to group gid.
func (t *Table) AddToGroup(gid, id string) {
	g, ok := t.groups[gid]
	if !ok {
		return
	}
	for _, c := range g.Children {
		if c == id {
			return
		}
	}
	g.Children = append(g.Children, id)
	if n, ok := t.nodes[id]; ok {
		n.Group = gid
	}
}

// Group looks up a group by id.
func (t *Table) Group(id string) (*Group, bool) {
	g, ok := t.groups[id]
	return g, ok
}

// Groups returns every group in declaration order.
func (t *Table) Groups() []*Group {
	out := make([]*Group, 0, len(t.groupOrder))
	for _, id := range t.groupOrder {
		out = append(out, t.groups[id])
	}
	return out
}

// AddEdge records an edge and assigns it a stable id.
func (t *Table) AddEdge(e *Edge) *Edge {
	if e.ID == "" {
		e.ID = edgeID(len(t.edges))
	}
	t.edges = append(t.edges, e)
	t.byPair[[2]string{e.From, e.To}] = append(t.byPair[[2]string{e.From, e.To}], e)
	return e
}

// Edges returns every edge in declaration order.
func (t *Table) Edges() []*Edge { return t.edges }

// FindEdge locates an edge between from and to. It also matches edges declared
// in the opposite direction, reporting reversed=true, because animations
// routinely travel back along a request path that was drawn one way.
func (t *Table) FindEdge(from, to string) (e *Edge, reversed bool, ok bool) {
	return t.FindEdgeN(from, to, 1)
}

// EdgesBetween returns the edges drawn from→to and those drawn to→from.
func (t *Table) EdgesBetween(from, to string) (forward, backward []*Edge) {
	return t.byPair[[2]string{from, to}], t.byPair[[2]string{to, from}]
}

// FindEdgeN picks the nth (1-based) edge a flow from→to can travel along.
//
// Same-direction edges are preferred and exhausted before reversed ones. In a
// flowchart that changes nothing — there is normally one arrow, and a response
// runs back along it. In a sequence diagram it is the difference between right
// and wrong: a reply is its own message with its own line, and matching it
// against the request's line would light the wrong arrow and travel backwards
// along it.
//
// n beyond the number of candidates clamps to the last rather than failing. A
// scenario that animates the same pair more times than the diagram draws it is
// reusing an arrow deliberately, which is the ordinary case for a flowchart.
func (t *Table) FindEdgeN(from, to string, n int) (e *Edge, reversed bool, ok bool) {
	if n < 1 {
		n = 1
	}
	forward, backward := t.EdgesBetween(from, to)

	if len(forward) > 0 {
		if n > len(forward) {
			n = len(forward)
		}
		return forward[n-1], false, true
	}
	if len(backward) > 0 {
		if n > len(backward) {
			n = len(backward)
		}
		return backward[n-1], true, true
	}
	return nil, false, false
}

// CountEdges is how many edges a from→to flow could pick between, in whichever
// direction it will end up matching.
func (t *Table) CountEdges(from, to string) int {
	forward, backward := t.EdgesBetween(from, to)
	if len(forward) > 0 {
		return len(forward)
	}
	return len(backward)
}

func edgeID(i int) string { return "e" + strconv.Itoa(i) }
