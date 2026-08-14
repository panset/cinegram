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
	"strings"

	"github.com/tejaspanse/cinegram/pkg/ast"
	"github.com/tejaspanse/cinegram/pkg/diag"
	"github.com/tejaspanse/cinegram/pkg/emit/mermaid"
	"github.com/tejaspanse/cinegram/pkg/ir"
	"github.com/tejaspanse/cinegram/pkg/loader"
	"github.com/tejaspanse/cinegram/pkg/symbol"
	"github.com/tejaspanse/cinegram/pkg/units"
	"github.com/tejaspanse/cinegram/pkg/voice"
)

const (
	// defaultHopMillis is how long a single flow hop takes when the source
	// does not say.
	defaultHopMillis = 600
	// minStepMillis is the floor under a step's span. The transport works in
	// ±1ms tolerances — advanceStep stops 1ms before a step's seam and treats
	// a step as begun only when it started more than 1ms ago — so a step of
	// 1ms or less has no interior stop point and the presenter's press would
	// re-select the same step forever. 4ms sits comfortably above those
	// tolerances and is invisible at any playback speed. Timing rules live
	// here, not in the runtime.
	minStepMillis = 4
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
	v := compileView(doc, table, "main", "", nil, assets{}, bag)
	return &ir.Timeline{Version: ir.Version, Root: v.ID, Views: []ir.View{v}}
}

// assets is everything the loader read from disk on a document's behalf, already
// inlined as data URIs: the storyboard images, and the narration clips keyed by
// the prose they speak.
//
// It travels as one value because Compile lowers a document that never had a
// filesystem and passes it empty — a zero assets is a document that draws and
// animates and says nothing aloud, which is the ordinary case.
type assets struct {
	frames map[string]string
	voice  map[string]voice.Loaded
}

// CompileBundle lowers every document a loader reached into one timeline, so a
// single page can host the whole set and clicks can navigate between them.
func CompileBundle(b *loader.Bundle) *ir.Timeline {
	t := &ir.Timeline{Version: ir.Version, Root: b.Root}
	for _, u := range b.Units {
		t.Views = append(t.Views, compileView(
			u.Result.Document, u.Result.Symbols, u.ViewID, u.Title, u.Views,
			assets{frames: u.FrameData, voice: u.VoiceData}, u.Bag))
	}
	return t
}

// compileView lowers one document. aliases maps the local `view` names the
// document used onto canonical view ids; it is nil when compiling standalone.
// a carries what the loader read from disk — storyboard images and narration —
// and is likewise empty when there was no filesystem to read from.
func compileView(doc *ast.Document, table *symbol.Table, id, title string, aliases map[string]string, a assets, bag *diag.Bag) ir.View {
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
	// A single scenario's name is a fair title for the view — there is nothing
	// else it could mean. With several, borrowing the first one's name labels
	// the whole page after one of its alternatives, so the header ends up
	// contradicting the scenario picker sitting next to it.
	if v.Title == "" && len(doc.Scenarios) == 1 {
		v.Title = doc.Scenarios[0].Name
	}

	// Allocated empty rather than left nil, because a nil slice marshals to
	// `null` and these keys are not optional — the renderer iterates all four
	// unconditionally. A flowchart with no subgraph is an ordinary diagram, and
	// it must not hand the runtime a null to walk.
	nodes, groups, edges := table.Nodes(), table.Groups(), table.Edges()
	v.Nodes = make([]ir.Node, 0, len(nodes))
	v.Groups = make([]ir.Group, 0, len(groups))
	v.Edges = make([]ir.Edge, 0, len(edges))

	for _, n := range nodes {
		v.Nodes = append(v.Nodes, ir.Node{
			ID: n.ID, Label: n.Label, Shape: n.Shape, Group: n.Group, Class: n.Class,
		})
	}
	for _, g := range groups {
		v.Groups = append(v.Groups, ir.Group{
			ID: g.ID, Label: g.Label, Parent: g.Parent, Children: g.Children,
		})
	}
	for _, e := range edges {
		v.Edges = append(v.Edges, ir.Edge{
			ID: e.ID, From: e.From, To: e.To, Label: e.Label,
			Style: e.Style, Head: e.Head, Bidir: e.Bidir,
		})
	}

	scenarios := resolveRetellings(resolveVariants(doc.Scenarios))
	v.Scenarios = make([]ir.Scenario, 0, len(scenarios))
	for i, sc := range scenarios {
		v.Scenarios = append(v.Scenarios, compileScenario(sc, i, table, a.voice, bag))
	}

	v.Storyboard = compileStoryboard(doc.Storyboards, a.frames)
	v.Bindings, v.Hidden = compileBindings(doc.Interactions, table, aliases)
	return v
}

// compileStoryboard flattens every storyboard block into the one panel the
// runtime draws.
//
// Frames are carried whether or not a scene names them: the panel is per view,
// the scenes are per scenario, and a frame used only by the second scenario
// still has to be there when the reader switches to it. A frame whose image
// failed to load compiles with an empty Image rather than being dropped —
// compilation stays total, and the loader has already put the error in the bag.
func compileStoryboard(storyboards []*ast.Storyboard, frameData map[string]string) *ir.Storyboard {
	if len(storyboards) == 0 {
		return nil
	}
	out := &ir.Storyboard{}
	for _, sb := range storyboards {
		// Several blocks merge into one panel, so the first title given names
		// the whole thing rather than the last one silently winning.
		if out.Title == "" {
			out.Title = sb.Title
		}
		for _, f := range sb.Frames {
			out.Frames = append(out.Frames, ir.Frame{
				ID:      f.Name,
				Caption: f.Caption,
				Image:   frameData[f.Img],
			})
		}
	}
	return out
}

// resolveVariants expands scenario inheritance into ordinary scenarios.
//
// The splice happens at AST level, before any timing runs, which is the whole
// reason this is cheap: a variant is literally the base's first N steps
// followed by its own, so every timing rule — step spans, seq chaining, hop
// occurrence counting, persistent-state windows — applies to it unchanged.
// Those are all reset per scenario inside compileScenario, so the inherited
// prefix replays from the start rather than continuing the base's counters.
//
// An unresolvable variant has already been reported by validation; here it
// simply compiles as its own steps alone, because compilation stays total.
func resolveVariants(scenarios []*ast.Scenario) []*ast.Scenario {
	byName := make(map[string]*ast.Scenario, len(scenarios))
	for _, sc := range scenarios {
		if sc.Name != "" && byName[sc.Name] == nil {
			byName[sc.Name] = sc
		}
	}

	out := make([]*ast.Scenario, 0, len(scenarios))
	for _, sc := range scenarios {
		out = append(out, spliceVariant(sc, byName))
	}
	return out
}

func spliceVariant(sc *ast.Scenario, byName map[string]*ast.Scenario) *ast.Scenario {
	want, ok := sc.Attrs.Get("variant")
	if !ok {
		return sc
	}
	base := byName[want.Raw]
	// Depth-1: a base that is itself a variant was rejected during validation,
	// and honouring it here would compile something the author was told is not
	// allowed.
	if base == nil || base == sc || base.Attrs.Has("variant") {
		return sc
	}

	prefix := len(base.Steps)
	if until, has := sc.Attrs.Get("until"); has {
		i := -1
		for j, st := range base.Steps {
			if st.EffectiveID(j) == until.Raw {
				i = j
				break
			}
		}
		if i < 0 {
			return sc // reported during validation
		}
		prefix = i + 1 // inclusive: the named step is the last one inherited
	}

	// The inherited steps are shared rather than copied. Nothing downstream
	// mutates an ast.Step, and copying would only invite the two scenarios to
	// drift.
	steps := make([]*ast.Step, 0, prefix+len(sc.Steps))
	steps = append(steps, base.Steps[:prefix]...)
	steps = append(steps, sc.Steps...)

	// The variant keeps its own attributes minus the two that described the
	// inheritance itself, which have now been consumed.
	merged := ast.Attrs{}
	for _, k := range sc.Attrs.Keys() {
		if k == "variant" || k == "until" {
			continue
		}
		v, _ := sc.Attrs.Get(k)
		merged.Set(k, v)
	}

	return &ast.Scenario{Name: sc.Name, Attrs: merged, Steps: steps, StartPos: sc.StartPos}
}

// resolveRetellings expands a narration overlay into an ordinary scenario.
//
// A retelling is the same animation in different words: `retells` names the
// scenario whose steps, actions and timing it adopts wholesale, and its own
// `step <id> { desc: … }` entries replace the prose on the steps they name.
// That is what makes an audience ladder affordable — one diagram explained to a
// child, a newcomer and an engineer costs one copy of the animation instead of
// three, and the three cannot drift out of step with each other because there
// is only one set of steps.
//
// It runs *after* resolveVariants, so a retelling of a variant retells the
// already-spliced result and needs no knowledge of variants at all. Depth-1
// applies to retellings alone: a retelling's base must not itself be one.
//
// An unresolvable retelling has already been reported by validation; here it
// compiles as its own steps alone, because compilation stays total.
func resolveRetellings(scenarios []*ast.Scenario) []*ast.Scenario {
	byName := make(map[string]*ast.Scenario, len(scenarios))
	for _, sc := range scenarios {
		if sc.Name != "" && byName[sc.Name] == nil {
			byName[sc.Name] = sc
		}
	}

	out := make([]*ast.Scenario, 0, len(scenarios))
	for _, sc := range scenarios {
		out = append(out, spliceRetelling(sc, byName))
	}
	return out
}

func spliceRetelling(sc *ast.Scenario, byName map[string]*ast.Scenario) *ast.Scenario {
	want, ok := sc.Attrs.Get("retells")
	if !ok {
		return sc
	}
	base := byName[want.Raw]
	// Depth-1, for the same reason variants are: a base that is itself a
	// retelling was rejected during validation, and honouring it here would
	// compile something the author was told is not allowed.
	if base == nil || base == sc || base.Attrs.Has("retells") {
		return sc
	}

	// The retelling's steps are overrides keyed by the base step they name, not
	// beats of their own — a retelling never changes what happens, only what is
	// said about it.
	override := make(map[string]*ast.Step, len(sc.Steps))
	for i, st := range sc.Steps {
		override[st.EffectiveID(i)] = st
	}

	steps := make([]*ast.Step, 0, len(base.Steps))
	for i, bst := range base.Steps {
		ov, has := override[bst.EffectiveID(i)]
		if !has {
			// Unchanged steps are shared rather than copied, exactly as a
			// variant's inherited prefix is: nothing downstream mutates an
			// ast.Step.
			steps = append(steps, bst)
			continue
		}

		// An overridden step *is* copied, which is the one place this differs
		// from spliceVariant. Attrs wraps a map, so a shallow copy of the step
		// would write the new prose through to the base and every other
		// retelling of it.
		clone := *bst
		clone.Attrs = ast.Attrs{}
		for _, k := range bst.Attrs.Keys() {
			v, _ := bst.Attrs.Get(k)
			clone.Attrs.Set(k, v)
		}
		for _, k := range ov.Attrs.Keys() {
			v, _ := ov.Attrs.Get(k)
			clone.Attrs.Set(k, v)
		}
		// A retelling may retitle the step as well as re-explain it, since the
		// caption shows the name above the prose and "Alice opens a connection"
		// is not how you say it to a child.
		//
		// Writing no title has to keep the base's, though, and parseStep has
		// already defaulted an untitled step's Name to its ID — so a Name equal
		// to the ID is one that was never written. Mistaking the two would
		// replace every inherited title with a bare step id.
		if ov.Name != "" && ov.Name != ov.ID {
			clone.Name = ov.Name
		}
		steps = append(steps, &clone)
	}

	// Attributes are inherited and then overridden, which is the opposite of a
	// variant's "keep only your own" — and it has to be, because a retelling is
	// the *same* walkthrough. It plays at the base's speed, loops if the base
	// loops, and above all ends the way the base ends: a retelling of a failure
	// path that lost `outcome: fail` would lose the ✕ the picker draws, and the
	// kid's telling of a story that goes wrong would claim to go right.
	//
	// The retelling's own attributes win, so any of them can still be restated,
	// and `retells` itself drops out having been consumed. `audience` is only
	// ever the retelling's own, since the base is not written for one.
	merged := ast.Attrs{}
	for _, k := range base.Attrs.Keys() {
		v, _ := base.Attrs.Get(k)
		merged.Set(k, v)
	}
	for _, k := range sc.Attrs.Keys() {
		if k == "retells" {
			continue
		}
		v, _ := sc.Attrs.Get(k)
		merged.Set(k, v)
	}

	return &ast.Scenario{Name: sc.Name, Attrs: merged, Steps: steps, StartPos: sc.StartPos}
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

		case ast.BindURL:
			if len(bd.Targets) == 0 {
				continue
			}
			b.URL = bd.Targets[0].Name

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

func compileScenario(sc *ast.Scenario, index int, table *symbol.Table, clips map[string]voice.Loaded, bag *diag.Bag) ir.Scenario {
	out := ir.Scenario{
		ID:    "s" + strconv.Itoa(index),
		Name:  sc.Name,
		Speed: attrFloat(sc.Attrs, "speed", 1.0, bag),
		Loop:  attrBool(sc.Attrs, "loop", false, bag),
		// Off unless the author asks: a page that is already moving when the
		// reader arrives is noise, and every surface agrees (the reduced-motion
		// media query, presenter mode and reels already all suppressed it).
		Autoplay: attrBool(sc.Attrs, "autoplay", false, bag),
		Poster:   attrMillis(sc.Attrs, "poster", 0, bag),
		Stepwise: attrBool(sc.Attrs, "stepwise", false, bag),
		Outcome:  sc.Attrs.String("outcome"),
		Audience: sc.Attrs.String("audience"),
	}

	out.Steps = make([]ir.Step, 0, len(sc.Steps))

	p := newPersist()
	// Occurrence tracking is per scenario: each walkthrough consumes the
	// diagram's repeated messages from the start.
	seen := hopCount{}

	// `pace: voice` is opt-in because it lets the recordings move the clock, and
	// the clock is otherwise entirely the author's. Without it a step keeps the
	// length it was written to have and a long sentence is simply cut off by the
	// next beat; with it, no step ends before it has finished speaking.
	fitVoice := sc.Attrs.String("pace") == "voice"
	if p := sc.Attrs.String("pace"); p != "" && p != "voice" {
		v, _ := sc.Attrs.Get("pace")
		bag.WarnHintf(v.At, "the only pace is `voice`, which stretches each step to fit its narration",
			"unknown pace %q", p)
	}

	cursor := 0
	for i, st := range sc.Steps {
		start := cursor + attrMillis(st.Attrs, "delay", 0, bag)
		desc := st.Attrs.String("desc")
		clip := clips[voice.Key(desc)]

		span := stepSpan(st, bag)
		if span < minStepMillis {
			// A note rather than an error: the clamp is invisible to a viewer,
			// but an author who wrote dur: 1ms should learn why it plays as 4.
			bag.WarnHintf(st.StartPos,
				"give the step a duration of at least 4ms, or drop dur: to use its actions' own span",
				"step %q spans %dms, too short for the step transport; clamped to %dms",
				st.EffectiveID(i), span, minStepMillis)
			span = minStepMillis
		}
		// Stretching before layout rather than after is what makes this cheap and
		// correct: the flows inside keep the durations they were given, while a
		// stateful action that spans "its whole step" spans the longer one, so a
		// highlight holds until the sentence about it is finished.
		//
		// After the minimum clamp, not before: that warning is about the span the
		// author wrote, and a step stretched to fit a ten-second line would
		// otherwise hide the fact that `dur: 1ms` was a typo.
		if fitVoice && clip.Ms > span {
			span = clip.Ms
		}

		step := ir.Step{
			ID:    st.EffectiveID(i),
			Name:  st.Name,
			Desc:  desc,
			Audio: clip.Data,
			Start: start,
			End:   start + span,
		}
		step.Tracks = layout(st.Actions, start, start+span, table, bag, p, seen)
		out.Steps = append(out.Steps, step)
		cursor = step.End
	}
	out.Duration = cursor

	// Anything still set when the scenario runs out stays set to the end.
	p.finish(cursor)
	out.Persistent = p.out
	return out
}

// --- persistent state --------------------------------------------------------

// persist accumulates the state that outlives the step that wrote it.
//
// The whole point of doing this in the compiler is that the renderer should not
// have to. A badge is not "on from here" — it is on over a closed interval, and
// working out where that interval ends means looking ahead at every later
// action. The compiler already walks them in order, so it closes each window as
// it goes and hands the renderer a list of intervals it can test against t.
type persist struct {
	out  []ir.Track
	open map[string]int // "target\x00slot" -> index into out
}

func newPersist() *persist { return &persist{open: make(map[string]int)} }

// slotKey identifies what a write collides with. Two writes close each other's
// window exactly when this matches: per target, one `set` slot and one slot per
// distinct gauge label, so two gauges on a node coexist and a repeat of one
// replaces it.
func slotKey(target, slot string) string { return target + "\x00" + slot }

func gaugeSlot(a ast.Action) string { return "gauge:" + a.Attrs.String("label") }

// closeAt ends an open window.
//
// Times need not arrive in order — `at:` can fire an action earlier than one
// written above it — so a close that lands before the open is clamped to a
// zero-length window rather than producing an inverted one the renderer would
// treat as always-open.
func (p *persist) closeAt(key string, at int) {
	i, ok := p.open[key]
	if !ok {
		return
	}
	if at < p.out[i].Start {
		at = p.out[i].Start
	}
	p.out[i].End = at
	delete(p.open, key)
}

// write closes whatever held this slot and opens a new window.
func (p *persist) write(target, slot string, at int, tr ir.Track) {
	key := slotKey(target, slot)
	p.closeAt(key, at)

	tr.Start = at
	tr.End = at // provisional: the next write to this slot, or finish(), extends it
	p.out = append(p.out, tr)
	p.open[key] = len(p.out) - 1
}

// clear ends every window on a target, which is what `unset` means.
func (p *persist) clear(target string, at int) {
	prefix := target + "\x00"
	for key := range p.open {
		if strings.HasPrefix(key, prefix) {
			p.closeAt(key, at)
		}
	}
}

// finish runs every window still open to the end of the scenario.
func (p *persist) finish(end int) {
	for key, i := range p.open {
		stop := end
		if stop < p.out[i].Start {
			stop = p.out[i].Start
		}
		p.out[i].End = stop
		delete(p.open, key)
	}
}

// layoutPersist records a persistent action. It contributes no step track: the
// state it writes is scenario-scoped, and a step track would vanish the moment
// the clock left the step.
func layoutPersist(a ast.Action, at int, p *persist) {
	for _, tgt := range a.Targets {
		switch a.Kind {
		case ast.ActionUnset:
			p.clear(tgt.Name, at)

		case ast.ActionGauge:
			// An empty value retires the reading, the same way an empty badge
			// retires a badge.
			if a.Attrs.String("value") == "" {
				p.closeAt(slotKey(tgt.Name, gaugeSlot(a)), at)
				continue
			}
			p.write(tgt.Name, gaugeSlot(a), at, ir.Track{
				Kind:   ir.TrackGauge,
				Target: tgt.Name,
				Label:  a.Attrs.String("label"),
				Value:  a.Attrs.String("value"),
				Style:  a.Attrs.String("style"),
				Color:  a.Attrs.String("color"),
			})

		default: // ast.ActionSet
			badge, state := a.Attrs.String("badge"), a.Attrs.String("state")
			// `set x { badge: "" }` is how a badge is taken away. It closes the
			// set slot only — the node's gauges are separate readings and have
			// no business disappearing with it.
			if badge == "" && state == "" {
				p.closeAt(slotKey(tgt.Name, "set"), at)
				continue
			}
			p.write(tgt.Name, "set", at, ir.Track{
				Kind:   ir.TrackSet,
				Target: tgt.Name,
				Label:  badge,
				Value:  state,
				Style:  a.Attrs.String("style"),
				Color:  a.Attrs.String("color"),
			})
		}
	}
}

func isPersistent(k ast.ActionKind) bool {
	return k == ast.ActionSet || k == ast.ActionGauge || k == ast.ActionUnset
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
			total += offsetOf(child, bag) + seqSpan(child, bag)
		}
		return total

	default:
		return attrMillis(a.Attrs, "dur", 0, bag)
	}
}

// seqSpan is how much of a seq's timeline a child consumes.
//
// A stateful action inside a seq has no step span to inherit, so it takes the
// default rather than collapsing to zero. Two kinds are the exception, for the
// same reason: they fire at an instant and what they write is not bounded by
// the seq at all, so giving either a slice of the chain would insert a silent
// pause with nothing in it.
//
// A persistent action is the obvious one. A `scene` is the other: it flips the
// storyboard panel and the panel then holds until something replaces it. That
// makes `seq { flow a -> b; scene x }` mean "show x when the chain gets here",
// which is the natural way to say "the screen changes when the arrow lands" —
// and, unlike computing an `at:` by hand, it stays correct when a hop's
// duration changes.
func seqSpan(child ast.Action, bag *diag.Bag) int {
	if isPersistent(child.Kind) || child.Kind == ast.ActionScene {
		return 0
	}
	if d := intrinsic(child, bag); d != 0 {
		return d
	}
	return defaultStepMillis
}

// offsetOf is how far into its container an action starts.
func offsetOf(a ast.Action, bag *diag.Bag) int {
	return attrMillis(a.Attrs, "delay", 0, bag) + attrMillis(a.Attrs, "at", 0, bag)
}

// layout places actions between start and end, emitting absolute-time tracks.
func layout(actions []ast.Action, start, end int, table *symbol.Table, bag *diag.Bag, p *persist, seen hopCount) []ir.Track {
	// Empty rather than nil for the same reason View's slices are: a step whose
	// actions all draw nothing — a lone `wait`, or a step that only sets its own
	// duration — is legitimate, and `"tracks": null` is a null the renderer
	// would have to walk.
	tracks := make([]ir.Track, 0, len(actions))
	for _, a := range actions {
		tracks = append(tracks, layoutAction(a, start, end, table, bag, p, seen)...)
	}
	return tracks
}

func layoutAction(a ast.Action, start, end int, table *symbol.Table, bag *diag.Bag, p *persist, seen hopCount) []ir.Track {
	at := start + offsetOf(a, bag)

	switch {
	case a.Kind == ast.ActionWait:
		return nil // a wait only consumes time; it draws nothing

	case isPersistent(a.Kind):
		layoutPersist(a, at, p)
		return nil // recorded at scenario level, not as a track of this step

	case a.Kind == ast.ActionSeq:
		var tracks []ir.Track
		cursor := at
		for _, child := range a.Body {
			d := seqSpan(child, bag)
			childStart := cursor + offsetOf(child, bag)
			tracks = append(tracks, layoutAction(child, childStart, childStart+d, table, bag, p, seen)...)
			cursor = childStart + d
		}
		return tracks

	case a.Kind == ast.ActionFlow:
		return layoutFlow(a, at, table, bag, seen)

	default:
		return layoutState(a, at, end, bag)
	}
}

// hopCount tracks how many times a scenario has already animated a pair, so
// repeated messages between the same two participants are consumed in order
// rather than all landing on the first one.
type hopCount map[[2]string]int

// occurrence decides which of several parallel edges a hop should use.
//
// With one candidate there is nothing to choose and every hop takes it — which
// is every flowchart, and why this changes nothing there. With several, an
// explicit `msg:` wins, and otherwise the scenario consumes them in order: a
// sequence diagram where A messages B three times should animate the three
// arrows in turn rather than the first one three times.
func occurrence(a ast.Action, from, to string, table *symbol.Table, seen hopCount, bag *diag.Bag) int {
	if table.CountEdges(from, to) < 2 {
		return 1
	}

	// The counter is keyed per pool, and the pool is directional: FindEdgeN
	// exhausts same-direction edges before reversed ones, so when the diagram
	// draws arrows both ways between a pair, `a -> b` and `b -> a` consume
	// from disjoint pools and must not share a count. When only one direction
	// is drawn, both spellings resolve against that pool and the key agrees.
	key := [2]string{from, to}
	if forward, _ := table.EdgesBetween(from, to); len(forward) == 0 {
		key = [2]string{to, from}
	}

	if a.Attrs.Has("msg") {
		n := attrInt(a.Attrs, "msg", 1, bag)
		// An explicit pick also moves the cursor, so the implicit flows that
		// follow continue after it instead of replaying from the start.
		seen[key] = n
		return n
	}

	seen[key]++
	return seen[key]
}

// layoutFlow splits a hop chain into one track per edge. Hop boundaries are
// computed from the total rather than accumulated, so the hops always sum to
// exactly the requested duration however it divides.
func layoutFlow(a ast.Action, at int, table *symbol.Table, bag *diag.Bag, seen hopCount) []ir.Track {
	hops := len(a.Targets) - 1
	if hops < 1 {
		return nil
	}
	total := attrMillis(a.Attrs, "dur", defaultHopMillis*hops, bag)

	var tracks []ir.Track
	for i := 0; i < hops; i++ {
		from, to := a.Targets[i].Name, a.Targets[i+1].Name
		edge, reversed, ok := table.FindEdgeN(from, to, occurrence(a, from, to, table, seen, bag))
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
			Side:   a.Attrs.String("side"),
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

func attrInt(a ast.Attrs, key string, def int, bag *diag.Bag) int {
	v, ok := a.Get(key)
	if !ok {
		return def
	}
	f, err := units.ParseFloat(v.Raw)
	if err != nil {
		bag.Errorf(v.At, "attribute %q: %v", key, err)
		return def
	}
	return int(f)
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
