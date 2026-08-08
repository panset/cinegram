// Package compile lowers a parsed document into an ir.Timeline.
//
// All of the language's timing semantics live here, and they come down to one
// rule: actions inside a step start together, steps run one after another. The
// pass is therefore two-phase per step — first work out how long the step must
// be from the actions that have an intrinsic duration, then lay every action
// out against that span.
package compile

import (
	"strconv"

	"github.com/tejaspanse/diagramator/pkg/ast"
	"github.com/tejaspanse/diagramator/pkg/diag"
	"github.com/tejaspanse/diagramator/pkg/emit/mermaid"
	"github.com/tejaspanse/diagramator/pkg/ir"
	"github.com/tejaspanse/diagramator/pkg/loader"
	"github.com/tejaspanse/diagramator/pkg/symbol"
	"github.com/tejaspanse/diagramator/pkg/units"
)

const (
	// defaultHopMillis is how long a single flow hop takes when the source
	// does not say.
	defaultHopMillis = 600
	// defaultStepMillis is the span of a step whose actions are all
	// stateful (a lone highlight, say) and so carry no duration of their own.
	defaultStepMillis = 800
)

// Compile lowers a single document into a one-view timeline. Problems are
// reported into bag; the returned timeline is still well-formed so that callers
// can render a partial result alongside the diagnostics.
//
// A document that declares views compiles here with those bindings unresolved —
// use CompileBundle to follow them.
func Compile(doc *ast.Document, table *symbol.Table, bag *diag.Bag) *ir.Timeline {
	v := compileView(doc, table, "main", "", nil, bag)
	return &ir.Timeline{Version: ir.Version, Root: v.ID, Views: []ir.View{v}}
}

// CompileBundle lowers every document a loader reached into one timeline, so a
// single page can host the whole set and clicks can navigate between them.
func CompileBundle(b *loader.Bundle) *ir.Timeline {
	t := &ir.Timeline{Version: ir.Version, Root: b.Root}
	for _, u := range b.Units {
		t.Views = append(t.Views, compileView(
			u.Result.Document, u.Result.Symbols, u.ViewID, u.Title, u.Views, u.Bag))
	}
	return t
}

// compileView lowers one document. aliases maps the local `view` names the
// document used onto canonical view ids; it is nil when compiling standalone.
func compileView(doc *ast.Document, table *symbol.Table, id, title string, aliases map[string]string, bag *diag.Bag) ir.View {
	v := ir.View{
		ID:      id,
		Title:   title,
		Diagram: ir.Diagram{Mermaid: mermaid.Emit(doc)},
	}
	if doc.Diagram != nil {
		v.Diagram.Type = doc.Diagram.Kind()
		if fc, ok := doc.Diagram.(*ast.Flowchart); ok {
			v.Diagram.Direction = fc.Direction
		}
	}
	if v.Title == "" && len(doc.Scenarios) > 0 {
		v.Title = doc.Scenarios[0].Name
	}

	for _, n := range table.Nodes() {
		v.Nodes = append(v.Nodes, ir.Node{
			ID: n.ID, Label: n.Label, Shape: n.Shape, Group: n.Group, Class: n.Class,
		})
	}
	for _, g := range table.Groups() {
		v.Groups = append(v.Groups, ir.Group{
			ID: g.ID, Label: g.Label, Parent: g.Parent, Children: g.Children,
		})
	}
	for _, e := range table.Edges() {
		v.Edges = append(v.Edges, ir.Edge{
			ID: e.ID, From: e.From, To: e.To, Label: e.Label,
			Style: e.Style, Head: e.Head, Bidir: e.Bidir,
		})
	}

	for i, sc := range doc.Scenarios {
		v.Scenarios = append(v.Scenarios, compileScenario(sc, i, table, bag))
	}

	v.Bindings, v.Hidden = compileBindings(doc.Interactions, table, aliases)
	return v
}

// compileBindings lowers click bindings and derives the set of elements that
// start hidden.
func compileBindings(bindings []*ast.Binding, table *symbol.Table, aliases map[string]string) ([]ir.Binding, []string) {
	var out []ir.Binding
	var hidden []string
	seen := map[string]bool{}

	for _, bd := range bindings {
		b := ir.Binding{
			Source: bd.Source.Name,
			Kind:   string(bd.Kind),
			Label:  bd.Attrs.String("label"),
			Style:  bd.Attrs.String("style"),
		}

		switch bd.Kind {
		case ast.BindView:
			if len(bd.Targets) == 0 {
				continue // reported during validation
			}
			alias := bd.Targets[0].Name
			b.View = alias
			if canonical, ok := aliases[alias]; ok {
				b.View = canonical
			}

		case ast.BindStep:
			if len(bd.Targets) == 0 {
				continue
			}
			b.Step = bd.Targets[0].Name

		case ast.BindReveal:
			// Targets are expanded the same way Hidden is, so the renderer
			// toggles exactly the set it conceals and never has to know that
			// a group implies its contents.
			for _, tgt := range bd.Targets {
				for _, id := range expand(tgt.Name, table) {
					b.Targets = append(b.Targets, id)
					if !seen[id] {
						seen[id] = true
						hidden = append(hidden, id)
					}
				}
			}
		}

		out = append(out, b)
	}
	return out, hidden
}

// expand returns name plus, when it is a group, everything inside it.
//
// Concealing a subgraph frame while its members stayed drawn would look broken,
// so a group reveal hides the whole cluster. Group children mix node and nested
// group ids with no discriminator, so this recurses on whichever it finds.
func expand(name string, table *symbol.Table) []string {
	g, ok := table.Group(name)
	if !ok {
		return []string{name}
	}
	out := []string{name}
	for _, child := range g.Children {
		out = append(out, expand(child, table)...)
	}
	return out
}

func compileScenario(sc *ast.Scenario, index int, table *symbol.Table, bag *diag.Bag) ir.Scenario {
	out := ir.Scenario{
		ID:       "s" + strconv.Itoa(index),
		Name:     sc.Name,
		Speed:    attrFloat(sc.Attrs, "speed", 1.0, bag),
		Loop:     attrBool(sc.Attrs, "loop", false, bag),
		Autoplay: attrBool(sc.Attrs, "autoplay", true, bag),
	}

	cursor := 0
	for i, st := range sc.Steps {
		start := cursor + attrMillis(st.Attrs, "delay", 0, bag)
		span := stepSpan(st, bag)

		step := ir.Step{
			ID:    st.EffectiveID(i),
			Name:  st.Name,
			Desc:  st.Attrs.String("desc"),
			Start: start,
			End:   start + span,
		}
		step.Tracks = layout(st.Actions, start, start+span, table, bag)
		out.Steps = append(out.Steps, step)
		cursor = step.End
	}
	out.Duration = cursor
	return out
}

// stepSpan decides how long a step lasts: an explicit `dur` if given, otherwise
// the longest intrinsic action duration, otherwise a default.
func stepSpan(st *ast.Step, bag *diag.Bag) int {
	if st.Attrs.Has("dur") {
		return attrMillis(st.Attrs, "dur", defaultStepMillis, bag)
	}
	longest := 0
	for _, a := range st.Actions {
		if end := offsetOf(a, bag) + intrinsic(a, bag); end > longest {
			longest = end
		}
	}
	if longest == 0 {
		return defaultStepMillis
	}
	return longest
}

// intrinsic is an action's own duration, or 0 for a stateful action that has
// not been given one and should therefore span its whole step.
func intrinsic(a ast.Action, bag *diag.Bag) int {
	switch a.Kind {
	case ast.ActionFlow:
		hops := len(a.Targets) - 1
		if hops < 1 {
			hops = 1
		}
		return attrMillis(a.Attrs, "dur", defaultHopMillis*hops, bag)

	case ast.ActionWait:
		return attrMillis(a.Attrs, "dur", 0, bag)

	case ast.ActionSeq:
		total := 0
		for _, child := range a.Body {
			d := intrinsic(child, bag)
			if d == 0 {
				// A stateful action inside a seq has no span to inherit,
				// so it takes the default rather than collapsing to zero.
				d = defaultStepMillis
			}
			total += offsetOf(child, bag) + d
		}
		return total

	default:
		return attrMillis(a.Attrs, "dur", 0, bag)
	}
}

// offsetOf is how far into its container an action starts.
func offsetOf(a ast.Action, bag *diag.Bag) int {
	return attrMillis(a.Attrs, "delay", 0, bag) + attrMillis(a.Attrs, "at", 0, bag)
}

// layout places actions between start and end, emitting absolute-time tracks.
func layout(actions []ast.Action, start, end int, table *symbol.Table, bag *diag.Bag) []ir.Track {
	var tracks []ir.Track
	for _, a := range actions {
		tracks = append(tracks, layoutAction(a, start, end, table, bag)...)
	}
	return tracks
}

func layoutAction(a ast.Action, start, end int, table *symbol.Table, bag *diag.Bag) []ir.Track {
	at := start + offsetOf(a, bag)

	switch a.Kind {
	case ast.ActionWait:
		return nil // a wait only consumes time; it draws nothing

	case ast.ActionSeq:
		var tracks []ir.Track
		cursor := at
		for _, child := range a.Body {
			d := intrinsic(child, bag)
			if d == 0 {
				d = defaultStepMillis
			}
			childStart := cursor + offsetOf(child, bag)
			tracks = append(tracks, layoutAction(child, childStart, childStart+d, table, bag)...)
			cursor = childStart + d
		}
		return tracks

	case ast.ActionFlow:
		return layoutFlow(a, at, table, bag)

	default:
		return layoutState(a, at, end, bag)
	}
}

// layoutFlow splits a hop chain into one track per edge. Hop boundaries are
// computed from the total rather than accumulated, so the hops always sum to
// exactly the requested duration however it divides.
func layoutFlow(a ast.Action, at int, table *symbol.Table, bag *diag.Bag) []ir.Track {
	hops := len(a.Targets) - 1
	if hops < 1 {
		return nil
	}
	total := attrMillis(a.Attrs, "dur", defaultHopMillis*hops, bag)

	var tracks []ir.Track
	for i := 0; i < hops; i++ {
		from, to := a.Targets[i].Name, a.Targets[i+1].Name
		edge, reversed, ok := table.FindEdge(from, to)
		if !ok {
			// Already reported during validation; skip so the rest of the
			// timeline still compiles.
			continue
		}
		tracks = append(tracks, ir.Track{
			Kind:    ir.TrackFlow,
			Start:   at + total*i/hops,
			End:     at + total*(i+1)/hops,
			Edge:    edge.ID,
			From:    from,
			To:      to,
			Reverse: reversed,
			Label:   a.Attrs.String("label"),
			Style:   a.Attrs.String("style"),
			Color:   a.Attrs.String("color"),
			Ease:    a.Attrs.String("ease"),
			Status:  a.Attrs.String("status"),
		})
	}
	return tracks
}

// layoutState places a stateful action. Without an explicit duration it spans
// the rest of its step, which is what makes `highlight ing` alongside a flow do
// the obvious thing.
func layoutState(a ast.Action, at, end int, bag *diag.Bag) []ir.Track {
	stop := end
	if a.Attrs.Has("dur") {
		stop = at + attrMillis(a.Attrs, "dur", 0, bag)
	}
	if stop < at {
		stop = at
	}

	kind := ir.TrackKind(a.Kind)
	tracks := make([]ir.Track, 0, len(a.Targets))
	for _, tgt := range a.Targets {
		tracks = append(tracks, ir.Track{
			Kind:   kind,
			Start:  at,
			End:    stop,
			Target: tgt.Name,
			Text:   a.Text,
			Label:  a.Attrs.String("label"),
			Style:  a.Attrs.String("style"),
			Color:  a.Attrs.String("color"),
			Ease:   a.Attrs.String("ease"),
		})
	}
	return tracks
}

// --- attribute readers ------------------------------------------------------
//
// Values were already type-checked during validation, so a failure here means
// the caller skipped that pass; fall back to the default rather than panicking.

func attrMillis(a ast.Attrs, key string, def int, bag *diag.Bag) int {
	v, ok := a.Get(key)
	if !ok {
		return def
	}
	ms, err := units.ParseMillis(v.Raw)
	if err != nil {
		bag.Errorf(v.At, "attribute %q: %v", key, err)
		return def
	}
	return ms
}

func attrFloat(a ast.Attrs, key string, def float64, bag *diag.Bag) float64 {
	v, ok := a.Get(key)
	if !ok {
		return def
	}
	f, err := units.ParseFloat(v.Raw)
	if err != nil {
		bag.Errorf(v.At, "attribute %q: %v", key, err)
		return def
	}
	return f
}

func attrBool(a ast.Attrs, key string, def bool, bag *diag.Bag) bool {
	v, ok := a.Get(key)
	if !ok {
		return def
	}
	b, err := units.ParseBool(v.Raw)
	if err != nil {
		bag.Errorf(v.At, "attribute %q: %v", key, err)
		return def
	}
	return b
}
