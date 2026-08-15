// Package ir defines the animation timeline: the layout-agnostic contract
// between the Go compiler and whatever renders the animation.
//
// Two properties matter and are worth preserving in any future change:
//
//   - Times are absolute integer milliseconds from the start of the scenario.
//     A renderer is then a clock and a scrubber, with no timing logic of its
//     own and no accumulated float drift.
//
//   - Geometry never appears here. Tracks and bindings reference node, edge and
//     group IDs only, so the same timeline can drive a mermaid.js overlay today
//     and a Go-native SVG backend later without recompilation.
package ir

// Version is the timeline schema version, carried in every document so a
// renderer can reject input it does not understand.
//
// Version 2 introduced Views: a timeline holds a set of diagrams that clicks
// navigate between, where version 1 held exactly one.
//
// Later additions — the storyboard panel, scenario outcomes — are omitempty and
// purely additive, and the runtime that reads this JSON is emitted into the
// same page as the JSON itself, so an older renderer never meets a newer
// timeline. Nothing since has needed a version bump.
const Version = 2

// Timeline is the complete compiled output for a source file and every
// document it references.
type Timeline struct {
	Version int    `json:"version"`
	Root    string `json:"root"` // ID of the view the renderer opens on
	Views   []View `json:"views"`
}

// View is one diagram together with its scenarios and click bindings.
type View struct {
	ID        string     `json:"id"`
	Title     string     `json:"title,omitempty"`
	Diagram   Diagram    `json:"diagram"`
	Nodes     []Node     `json:"nodes"`
	Groups    []Group    `json:"groups"`
	Edges     []Edge     `json:"edges"`
	Scenarios []Scenario `json:"scenarios"`
	Bindings  []Binding  `json:"bindings,omitempty"`

	// Storyboard is what the human sees while this diagram animates. Every
	// declared frame is carried, referenced or not, so a `scene` in any
	// scenario of this view resolves against one list.
	Storyboard *Storyboard `json:"storyboard,omitempty"`

	// Hidden lists elements that start concealed because a reveal binding
	// points at them. It is derived during compilation, not authored: being
	// revealable is what makes an element initially hidden.
	Hidden []string `json:"hidden,omitempty"`
}

// Binding makes one element clickable.
//
// Reveal state is deliberately not a Track. Tracks are timeline state that the
// clock owns and resets on every seek; a reveal is interaction state that
// persists until the viewer leaves the view.
type Binding struct {
	Source string `json:"source"`
	Kind   string `json:"kind"` // view | reveal | step | url

	View    string   `json:"view,omitempty"` // target view ID, for kind "view"
	Step    string   `json:"step,omitempty"` // target step ID, for kind "step"
	URL     string   `json:"url,omitempty"`  // destination, for kind "url"
	Targets []string `json:"targets,omitempty"`

	Label string `json:"label,omitempty"`
	Style string `json:"style,omitempty"`
}

// Storyboard is the side-stage panel: the screens a person would be looking at
// while the diagram animates.
type Storyboard struct {
	Title  string  `json:"title,omitempty"`
	Frames []Frame `json:"frames"`
}

// Frame is one thing the storyboard panel can show.
//
// Image is a self-contained `data:` URI rather than a path, because the page
// has to work from the filesystem and under a webview CSP — the loader reads
// the file and inlines it, and this stays as free of the filesystem as the rest
// of the timeline. It is empty for a caption-only frame.
type Frame struct {
	ID      string `json:"id"`
	Caption string `json:"caption,omitempty"`
	Image   string `json:"image,omitempty"`
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

	// Poster is the millisecond the page rests at before anyone presses play —
	// a walkthrough's most telling moment rather than its blank opening. Zero
	// means the start. Stepwise turns Play into the presenter transport: each
	// press plays exactly one step and stops at its end.
	Poster   int  `json:"poster,omitempty"`
	Stepwise bool `json:"stepwise,omitempty"`

	// Outcome is how this walkthrough ended: empty when unstated, "ok", or
	// "fail". It uses the same closed vocabulary as a flow's Status, and for
	// the same reason — a renderer marks a failure in the scenario picker
	// rather than merely recolouring its name.
	Outcome string `json:"outcome,omitempty"`

	Steps []Step `json:"steps"`

	// Persistent holds state that outlives the step that set it: a badge, a
	// gauge reading, a standing state class.
	//
	// It cannot live in a Step, because a renderer skips steps whose window
	// excludes the current time — a badge set in step 1 has to survive step 6.
	// So the compiler flattens it to scenario level and works out each window
	// itself: a track opens when its action fires and closes when a later
	// action writes the same slot on the same target, or at Duration. The
	// renderer then applies any persistent track whose window contains t, and
	// seek-determinism follows for free.
	Persistent []Track `json:"persistent,omitempty"`
}

// Step is one beat of a scenario. Steps never overlap: each begins where the
// previous one ended, plus any explicit delay.
type Step struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`

	// Desc is the prose explaining what this step means, as opposed to Name,
	// which only labels it. A renderer shows it as narration alongside the
	// animation; `narrate`-style tooling reads it as the body text.
	Desc string `json:"desc,omitempty"`

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

	// TrackFocus names what should hold attention. The track lists only what is
	// focused; working out that everything else recedes — and that a group
	// means its contents — is the renderer's job, because it is the side that
	// knows the containment tree it already has in View.Groups.
	TrackFocus TrackKind = "focus"

	// TrackScene names a storyboard frame in Target rather than a diagram
	// element. The renderer shows the latest scene track whose Start is at or
	// before t — stickily, because what a person is looking at does not vanish
	// when a step ends — which keeps a seek identical to having played there.
	TrackScene TrackKind = "scene"

	// Persistent kinds, found in Scenario.Persistent rather than Step.Tracks.
	//
	// TrackSet carries a badge in Label and a state name in Value; either may
	// be empty. TrackGauge names the reading in Label and its current value in
	// Value — a string, because the timeline renders readings and never does
	// arithmetic on them.
	TrackSet   TrackKind = "set"
	TrackGauge TrackKind = "gauge"
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
	Value string `json:"value,omitempty"`
	Style string `json:"style,omitempty"`
	Color string `json:"color,omitempty"`
	Ease  string `json:"ease,omitempty"`

	// Side is which way a note is placed relative to its target: above (the
	// default), below, left or right. A preference, not a position — the
	// renderer still has to keep the note inside the stage.
	Side string `json:"side,omitempty"`

	// Status is a flow's outcome: empty or "ok" for one that succeeded, "fail"
	// for one that did not. It is deliberately separate from Style, which is a
	// free-form CSS hook the renderer passes through: Status is a closed set
	// with meaning, so a renderer may draw a failure differently — an error
	// particle, a struck-through arrival — rather than just recolouring it.
	Status string `json:"status,omitempty"`
}
