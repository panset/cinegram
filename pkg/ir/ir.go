// Package ir defines the animation timeline: the layout-agnostic contract
// between the Go compiler and whatever renders the animation.
//
// Two properties matter and are worth preserving in any future change:
//
//   - Times are absolute integer milliseconds from the start of the scenario.
//     A renderer is then a clock and a scrubber, with no timing logic of its
//     own and no accumulated float drift.
//
//   - Geometry never appears here. Tracks reference node, edge and group IDs
//     only, so the same timeline can drive a mermaid.js overlay today and a
//     Go-native SVG backend later without recompilation.
package ir

// Version is the timeline schema version, carried in every document so a
// renderer can reject input it does not understand.
const Version = 1

// Timeline is the complete compiled output for one source file.
type Timeline struct {
	Version   int        `json:"version"`
	Diagram   Diagram    `json:"diagram"`
	Nodes     []Node     `json:"nodes"`
	Groups    []Group    `json:"groups"`
	Edges     []Edge     `json:"edges"`
	Scenarios []Scenario `json:"scenarios"`
}

// Diagram carries the static structure, including the Mermaid source a
// renderer feeds to mermaid.js to obtain an SVG to animate over.
type Diagram struct {
	Type      string `json:"type"`
	Direction string `json:"direction,omitempty"`
	Mermaid   string `json:"mermaid"`
}

// Node is an animatable diagram element.
type Node struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Shape string `json:"shape,omitempty"`
	Group string `json:"group,omitempty"`
	Class string `json:"class,omitempty"`
}

// Group is a subgraph container.
type Group struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Parent   string   `json:"parent,omitempty"`
	Children []string `json:"children,omitempty"`
}

// Edge is a connection that flows can travel along.
type Edge struct {
	ID    string `json:"id"`
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label,omitempty"`
	Style string `json:"style,omitempty"`
	Head  string `json:"head,omitempty"`
	Bidir bool   `json:"bidir,omitempty"`
}

// Scenario is one compiled walkthrough.
type Scenario struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Duration int     `json:"duration"`
	Speed    float64 `json:"speed"`
	Loop     bool    `json:"loop"`
	Autoplay bool    `json:"autoplay"`
	Steps    []Step  `json:"steps"`
}

// Step is one beat of a scenario. Steps never overlap: each begins where the
// previous one ended, plus any explicit delay.
type Step struct {
	ID     string  `json:"id"`
	Name   string  `json:"name,omitempty"`
	Start  int     `json:"start"`
	End    int     `json:"end"`
	Tracks []Track `json:"tracks"`
}

// TrackKind names what a track animates.
type TrackKind string

const (
	TrackFlow      TrackKind = "flow"
	TrackHighlight TrackKind = "highlight"
	TrackNote      TrackKind = "note"
	TrackDim       TrackKind = "dim"
	TrackPulse     TrackKind = "pulse"
	TrackShow      TrackKind = "show"
	TrackHide      TrackKind = "hide"
)

// Track is a single animated element with absolute start and end times.
//
// A flow track always covers exactly one edge; a multi-hop `flow a -> b -> c`
// compiles to one track per hop with consecutive time ranges, so the renderer
// never has to reason about chains.
type Track struct {
	Kind  TrackKind `json:"kind"`
	Start int       `json:"start"`
	End   int       `json:"end"`

	// Flow tracks.
	Edge string `json:"edge,omitempty"`
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
	// Reverse is set when the flow travels against the direction the edge
	// was drawn in, which is the normal case for a response path.
	Reverse bool `json:"reverse,omitempty"`

	// Node tracks.
	Target string `json:"target,omitempty"`

	Label string `json:"label,omitempty"`
	Text  string `json:"text,omitempty"`
	Style string `json:"style,omitempty"`
	Color string `json:"color,omitempty"`
	Ease  string `json:"ease,omitempty"`
}
