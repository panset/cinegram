// Package ast defines the syntax tree for Cinegram documents.
//
// A document is a Mermaid diagram body followed by zero or more scenario,
// view and interact blocks. The two halves are deliberately decoupled: Diagram
// is an interface so new Mermaid diagram types can be added, and neither the
// Scenario tree nor the interaction bindings ever mention flowchart concepts.
// Both refer to the diagram only through plain string references, which a later
// validation pass resolves against a symbol table.
//
// Every statement retains its Raw source text. Emitting Mermaid therefore
// reprints the original lines rather than regenerating them, which keeps the
// "strip the scenario blocks and you have valid Mermaid" guarantee true even
// for syntax the parser does not model semantically.
package ast

import (
	"strconv"

	"github.com/tejaspanse/cinegram/pkg/source"
)

// Document is a complete parsed source file.
type Document struct {
	Frontmatter  string   // raw YAML frontmatter including its --- fences, or ""
	Preamble     []string // comment and directive lines above the diagram header
	Diagram      Diagram
	Scenarios    []*Scenario
	Views        []*ViewDecl // documents a click can drill into
	Interactions []*Binding  // what each clickable element does
}

// Diagram is a diagram body of some Mermaid type.
type Diagram interface {
	// Kind is the Mermaid diagram keyword, e.g. "flowchart".
	Kind() string
	// Header is the raw header line, e.g. "flowchart LR".
	Header() string
	// Body returns the diagram's top-level statements.
	Body() []Statement
}

// Statement is one line (or block) of a diagram body.
type Statement interface {
	// Raw is the statement's original source text, trimmed of indentation.
	Raw() string
	// Pos is where the statement begins.
	Pos() source.Pos
}

// ---------------------------------------------------------------------------
// Flowchart
// ---------------------------------------------------------------------------

// Flowchart is a `flowchart`/`graph` diagram body.
type Flowchart struct {
	HeaderText string // e.g. "flowchart LR"
	Direction  string // LR, RL, TB, TD, BT
	Statements []Statement
	StartPos   source.Pos
}

func (f *Flowchart) Kind() string      { return "flowchart" }
func (f *Flowchart) Header() string    { return f.HeaderText }
func (f *Flowchart) Body() []Statement { return f.Statements }

// Shape is a Mermaid node shape.
type Shape string

const (
	ShapeRect             Shape = "rect"              // id[text]
	ShapeRound            Shape = "round"             // id(text)
	ShapeStadium          Shape = "stadium"           // id([text])
	ShapeSubroutine       Shape = "subroutine"        // id[[text]]
	ShapeCylinder         Shape = "cylinder"          // id[(text)]
	ShapeCircle           Shape = "circle"            // id((text))
	ShapeDoubleCircle     Shape = "doublecircle"      // id(((text)))
	ShapeRhombus          Shape = "rhombus"           // id{text}
	ShapeHexagon          Shape = "hexagon"           // id{{text}}
	ShapeParallelogram    Shape = "parallelogram"     // id[/text/]
	ShapeParallelogramAlt Shape = "parallelogram-alt" // id[\text\]
	ShapeTrapezoid        Shape = "trapezoid"         // id[/text\]
	ShapeTrapezoidAlt     Shape = "trapezoid-alt"     // id[\text/]
	ShapeAsymmetric       Shape = "asymmetric"        // id>text]
	ShapeNone             Shape = "none"              // bare id, no shape given
)

// NodeRef is a single node mention: either a declaration with a shape and
// label, or a bare reference to a node declared elsewhere.
type NodeRef struct {
	ID      string
	Label   string // display text; equals ID when no shape was given
	Shape   Shape
	Class   string // from the `id:::className` suffix
	Declare bool   // true when this mention carried a shape/label
	At      source.Pos
}

// LinkStyle is the visual weight of an edge.
type LinkStyle string

const (
	LinkNormal LinkStyle = "normal" // ---, -->
	LinkThick  LinkStyle = "thick"  // ===, ==>
	LinkDotted LinkStyle = "dotted" // -.-, -.->
	LinkInvis  LinkStyle = "invisible"
)

// ArrowHead is the terminator drawn at an edge's target end.
type ArrowHead string

const (
	ArrowNone   ArrowHead = "none"   // ---
	ArrowNormal ArrowHead = "arrow"  // -->
	ArrowCircle ArrowHead = "circle" // --o
	ArrowCross  ArrowHead = "cross"  // --x
)

// Link describes the operator joining two nodes.
type Link struct {
	Style    LinkStyle
	Head     ArrowHead
	Bidir    bool   // <--> form
	Label    string // from |text| or the -- text --> form
	Length   int    // dash count, controls Mermaid rank spacing
	Operator string // the raw operator text
}

// EdgeRef is one resolved from->to hop of an edge chain.
type EdgeRef struct {
	From string
	To   string
	Link Link
	At   source.Pos
}

// NodeStmt is a line that only declares nodes.
type NodeStmt struct {
	Nodes    []NodeRef
	Text     string
	StartPos source.Pos
}

func (s *NodeStmt) Raw() string     { return s.Text }
func (s *NodeStmt) Pos() source.Pos { return s.StartPos }

// EdgeStmt is a line containing one or more links, e.g. `a --> b --> c`.
// Nodes holds every node mentioned on the line, in order, so that inline
// declarations like `client[Client] --> ing[Ingress]` register correctly.
type EdgeStmt struct {
	Nodes    []NodeRef
	Edges    []EdgeRef
	Text     string
	StartPos source.Pos
}

func (s *EdgeStmt) Raw() string     { return s.Text }
func (s *EdgeStmt) Pos() source.Pos { return s.StartPos }

// SubgraphStmt is a `subgraph ... end` block. Subgraphs nest.
type SubgraphStmt struct {
	ID        string
	Label     string
	Direction string // from a nested `direction XX` statement
	Body      []Statement
	Text      string // the header line only
	EndText   string // the closing `end` line, preserved for round-tripping
	StartPos  source.Pos
}

func (s *SubgraphStmt) Raw() string     { return s.Text }
func (s *SubgraphStmt) Pos() source.Pos { return s.StartPos }

// RawStmt is any line the parser does not model semantically: classDef, style,
// linkStyle, click, comments, and anything unrecognised. It round-trips
// verbatim, which is what makes Mermaid emission lossless.
type RawStmt struct {
	Text     string
	StartPos source.Pos
}

func (s *RawStmt) Raw() string     { return s.Text }
func (s *RawStmt) Pos() source.Pos { return s.StartPos }

// ---------------------------------------------------------------------------
// Sequence
// ---------------------------------------------------------------------------

// Sequence is a `sequenceDiagram` body.
type Sequence struct {
	HeaderText string
	Statements []Statement
	StartPos   source.Pos
}

func (s *Sequence) Kind() string      { return "sequenceDiagram" }
func (s *Sequence) Header() string    { return s.HeaderText }
func (s *Sequence) Body() []Statement { return s.Statements }

// ParticipantStmt declares an actor, with or without a display alias.
type ParticipantStmt struct {
	ID    string
	Label string
	Actor bool // written as `actor` rather than `participant`

	Text     string
	StartPos source.Pos
}

func (s *ParticipantStmt) Raw() string     { return s.Text }
func (s *ParticipantStmt) Pos() source.Pos { return s.StartPos }

// MessageStmt is one message between two participants.
//
// Every occurrence is its own statement and its own edge, even when the same
// pair exchanges the same message twice: in a sequence diagram the arrows are
// the timeline, so two identical messages are two distinct things to animate.
type MessageStmt struct {
	From     string
	To       string
	Operator string // the arrow as written, e.g. "->>" or "--)"
	Label    string // the text after the colon

	Text     string
	StartPos source.Pos
}

func (s *MessageStmt) Raw() string     { return s.Text }
func (s *MessageStmt) Pos() source.Pos { return s.StartPos }

// ---------------------------------------------------------------------------
// Scenario (diagram-agnostic)
// ---------------------------------------------------------------------------

// Scenario is one animated walkthrough of the diagram.
type Scenario struct {
	Name     string
	Attrs    Attrs
	Steps    []*Step
	StartPos source.Pos
}

// Step is a named beat of a scenario. Actions inside a step start together;
// steps run one after another.
type Step struct {
	ID       string // explicit identifier, or generated during compilation
	Name     string
	Attrs    Attrs
	Actions  []Action
	StartPos source.Pos
}

// EffectiveID is the identifier a step is addressable by, given its position in
// the scenario. Steps need not be named, so an unnamed one falls back to its
// index. Compilation and the `click … -> step` binding must agree on this, so
// both go through here rather than repeating the convention.
func (s *Step) EffectiveID(index int) string {
	if s.ID != "" {
		return s.ID
	}
	return "step" + strconv.Itoa(index)
}

// ActionKind names an animation primitive.
type ActionKind string

const (
	ActionFlow      ActionKind = "flow"
	ActionHighlight ActionKind = "highlight"
	ActionNote      ActionKind = "note"
	ActionDim       ActionKind = "dim"
	ActionPulse     ActionKind = "pulse"
	ActionShow      ActionKind = "show"
	ActionHide      ActionKind = "hide"
	ActionFocus     ActionKind = "focus"
	ActionWait      ActionKind = "wait"
	ActionSeq       ActionKind = "seq"

	// Persistent actions. Unlike the rest, these do not describe a span of the
	// step they sit in: they fire at a moment and the state they write outlives
	// the step, until something overwrites it or the scenario ends.
	ActionSet   ActionKind = "set"
	ActionGauge ActionKind = "gauge"
	ActionUnset ActionKind = "unset"
)

// Action is a single animation instruction inside a step.
//
// One struct covers every kind rather than an interface per primitive: the
// primitives differ only in which fields they populate, and a flat shape keeps
// the compiler's timing pass a simple switch instead of a visitor.
type Action struct {
	Kind ActionKind

	// Targets holds node references. For ActionFlow it is the hop chain
	// (`a -> b -> c`); for the others it is the set of affected nodes.
	Targets []Target

	Text  string   // note body
	Attrs Attrs    // duration, label, style, ...
	Body  []Action // nested actions, for ActionSeq

	At source.Pos
}

// Target is a reference from a scenario into the diagram. It is intentionally
// just a name plus a position: resolution happens in a separate pass so this
// package stays free of diagram-type specifics.
type Target struct {
	Name string
	At   source.Pos
}

// ViewDecl names another document that a click can drill into.
//
// Only the path as written is recorded. Resolving it against the filesystem is
// the loader's job, which keeps this package and the parser free of I/O.
type ViewDecl struct {
	ID     string // local alias used by `click … -> view <ID>`
	Title  string // display title, or "" to fall back to the target's own
	Path   string // path as written, relative to the declaring file
	At     source.Pos
	PathAt source.Pos // reported against when the path cannot be resolved
}

// BindingKind names what clicking an element does.
type BindingKind string

const (
	BindView   BindingKind = "view"
	BindReveal BindingKind = "reveal"
	BindStep   BindingKind = "step"
	BindURL    BindingKind = "url"
)

// Binding makes one diagram element clickable.
//
// Like Action it names things and never models them, so the interaction half
// stays as diagram-agnostic as the scenario half: Source and Targets are plain
// references that the validation pass resolves against the symbol table.
type Binding struct {
	Source Target // node or group that becomes clickable
	Kind   BindingKind

	// Targets is the view alias for BindView, the step id for BindStep, the
	// destination for BindURL, and the set of elements to conceal for
	// BindReveal.
	Targets []Target

	Attrs Attrs
	At    source.Pos
}

// Attrs is an ordered attribute map from a `{ key: value }` block.
type Attrs struct {
	keys   []string
	values map[string]Value
}

// Value is an attribute value together with where it appeared.
type Value struct {
	Raw string
	At  source.Pos
}

// Set records key=v, replacing any previous value for key.
func (a *Attrs) Set(key string, v Value) {
	if a.values == nil {
		a.values = make(map[string]Value)
	}
	if _, seen := a.values[key]; !seen {
		a.keys = append(a.keys, key)
	}
	a.values[key] = v
}

// Get returns the value for key.
func (a Attrs) Get(key string) (Value, bool) {
	v, ok := a.values[key]
	return v, ok
}

// String returns the raw text of key, or "" if unset.
func (a Attrs) String(key string) string { return a.values[key].Raw }

// Has reports whether key was set.
func (a Attrs) Has(key string) bool { _, ok := a.values[key]; return ok }

// Keys returns the attribute names in the order they were written.
func (a Attrs) Keys() []string { return a.keys }
