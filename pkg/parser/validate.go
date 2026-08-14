package parser

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/tejaspanse/cinegram/pkg/ast"
	"github.com/tejaspanse/cinegram/pkg/diag"
	"github.com/tejaspanse/cinegram/pkg/symbol"
)

// validateScenarios resolves every scenario reference against the symbol table
// the diagram parser produced.
//
// This pass is the only place the two halves of a document meet, and it works
// purely against symbol.Table — no flowchart types appear here, which is what
// keeps the animation layer reusable across diagram types.
func validateScenarios(scenarios []*ast.Scenario, t *symbol.Table, frames *frameSet, b *diag.Bag) {
	names := knownNames(t)
	validateVariants(scenarios, b)
	validateRetellings(scenarios, b)

	for _, sc := range scenarios {
		checkAttrs(sc.Attrs, scenarioAttrs, "scenario", b)
		seen := make(map[string]bool)
		for _, st := range sc.Steps {
			if st.ID != "" {
				if seen[st.ID] {
					b.Errorf(st.StartPos, "duplicate step id %q", st.ID)
				}
				seen[st.ID] = true
			}
			checkAttrs(st.Attrs, stepAttrs, "step", b)
			validateActions(st.Actions, t, names, frames, b)
		}
		// A variant with no steps of its own is odd but not empty: it inherits
		// its base's prefix, which is a legitimate way to say "just this much
		// of the story". Only a scenario with nothing at all is worth reporting.
		switch {
		case len(sc.Steps) > 0 || sc.Attrs.Has("variant"):
		case sc.Attrs.Has("retells"):
			// A retelling with no overrides is not empty — it inherits every
			// step — but it is a word-for-word copy of its base, which is
			// never what the author meant by writing it.
			b.WarnHintf(sc.StartPos, "override a step's `desc` to say it differently, or delete this scenario",
				"retelling %q overrides no narration", sc.Name)
		default:
			b.Warnf(sc.StartPos, "scenario %q has no steps", sc.Name)
		}
	}
}

// validateVariants checks scenario inheritance before the compiler splices it.
//
// A failure path is usually the happy path up to a point and then not, and
// writing the shared prefix twice means maintaining it twice. `variant:` names
// the scenario to inherit from and `until:` the last of its steps to take —
// inclusive, because "X happened, and then things diverged" is the sentence the
// pair is meant to read as.
//
// Inheritance is depth-1 on purpose. A chain would make the effective id of a
// step a function of an arbitrarily long ancestry, and `click … -> step` has to
// stay something an author can work out by looking.
func validateVariants(scenarios []*ast.Scenario, b *diag.Bag) {
	byName := make(map[string]*ast.Scenario, len(scenarios))
	var names []string
	for _, sc := range scenarios {
		if sc.Name == "" || byName[sc.Name] != nil {
			continue // an unnamed or duplicate scenario cannot be inherited from
		}
		byName[sc.Name] = sc
		names = append(names, sc.Name)
	}
	sort.Strings(names)

	for _, sc := range scenarios {
		want, hasVariant := sc.Attrs.Get("variant")
		until, hasUntil := sc.Attrs.Get("until")

		if !hasVariant {
			if hasUntil {
				b.ErrorHintf(until.At, `add `+"`variant: \"<scenario name>\"`"+` to say which scenario to inherit from`,
					"`until` names a step to inherit up to, but no scenario is being inherited from")
			}
			continue
		}

		if want.Raw == sc.Name {
			b.ErrorHintf(want.At, "name the scenario this one diverges from",
				"scenario %q is a variant of itself", sc.Name)
			continue
		}
		base, ok := byName[want.Raw]
		if !ok {
			hint, best := suggestFrom(want.Raw, names, "scenarios")
			b.ErrorFixf(want.At, valueFix(want, best), hint,
				"no scenario named %q to inherit from", want.Raw)
			continue
		}
		if base.Attrs.Has("variant") {
			b.ErrorHintf(want.At, "inherit from the original instead, or write this scenario out in full",
				"scenario %q is itself a variant; a variant cannot inherit from another variant", want.Raw)
			continue
		}

		// No `until` inherits the whole base and appends, which is the "and then
		// this also happened" case. With one, the prefix runs through that step.
		prefix := len(base.Steps)
		if hasUntil {
			i := stepIndex(base, until.Raw)
			if i < 0 {
				hint, best := suggestFrom(until.Raw, effectiveIDs(base), "steps")
				b.ErrorFixf(until.At, valueFix(until, best), hint,
					"scenario %q has no step %q to inherit up to", want.Raw, until.Raw)
				continue
			}
			prefix = i + 1
		}
		checkSpliceCollisions(sc, base, prefix, b)
	}
}

// checkSpliceCollisions rehearses the splice the compiler will perform.
//
// The merged scenario has to keep distinct step ids or `click … -> step` and
// the keyboard's step jumps would address two beats with one name. Positions
// shift: an inherited step keeps its index in the base, so an *anonymous* own
// step becomes `step<prefix+i>` rather than `step<i>`, and only explicitly
// named steps can actually collide.
func checkSpliceCollisions(sc, base *ast.Scenario, prefix int, b *diag.Bag) {
	inherited := make(map[string]bool, prefix)
	for i := 0; i < prefix && i < len(base.Steps); i++ {
		inherited[base.Steps[i].EffectiveID(i)] = true
	}
	for i, st := range sc.Steps {
		id := st.EffectiveID(prefix + i)
		if inherited[id] {
			b.ErrorHintf(st.StartPos, "rename this step; the inherited prefix already uses that id",
				"step id %q is already inherited from scenario %q", id, base.Name)
		}
	}
}

// validateRetellings checks narration overlays before the compiler splices them.
//
// A retelling adopts its base's steps, actions and timing wholesale and replaces
// only the words: `retells:` names the base, and each `step <id> { desc: … }`
// re-explains the step of that name. So the same animation can be told to a
// child, a newcomer and an engineer from one copy of the diagram, and the
// tellings cannot drift apart because there is only one set of steps.
//
// Three rules make the overlay readable, and all three are enforced here rather
// than left to surprise the author at runtime:
//
//   - A step of a retelling must name its base step *explicitly*. An anonymous
//     step falls back to `step<index>`, so it would silently override whichever
//     beat happened to sit at that position — a rename of the base would then
//     move the words to a different step with no diagnostic anywhere.
//   - A retelling carries no actions. Changing what happens is what a variant
//     is for; a retelling that could also animate would make "same animation,
//     different words" untrue and there would be no way to see which it did.
//   - Depth-1, as with variants: a retelling of a retelling would make the
//     words a step ends up with a function of an arbitrarily long ancestry.
//
// Retelling a *variant* is fine, though, and needs nothing special: the
// compiler resolves variants first, so the base is already spliced by the time
// the overlay lands. That is why membership is tested against effectiveStepIDs
// rather than the base's own step list.
func validateRetellings(scenarios []*ast.Scenario, b *diag.Bag) {
	byName := make(map[string]*ast.Scenario, len(scenarios))
	var names []string
	for _, sc := range scenarios {
		if sc.Name == "" || byName[sc.Name] != nil {
			continue // an unnamed or duplicate scenario cannot be retold
		}
		byName[sc.Name] = sc
		names = append(names, sc.Name)
	}
	sort.Strings(names)

	for _, sc := range scenarios {
		want, ok := sc.Attrs.Get("retells")
		if !ok {
			continue
		}

		if sc.Attrs.Has("variant") {
			b.ErrorHintf(want.At, "a variant changes what happens and a retelling changes what is said about it; split them into two scenarios",
				"scenario %q is both a variant and a retelling", sc.Name)
			continue
		}
		if want.Raw == sc.Name {
			b.ErrorHintf(want.At, "name the scenario whose animation this one re-explains",
				"scenario %q retells itself", sc.Name)
			continue
		}
		base, ok := byName[want.Raw]
		if !ok {
			hint, best := suggestFrom(want.Raw, names, "scenarios")
			b.ErrorFixf(want.At, valueFix(want, best), hint,
				"no scenario named %q to retell", want.Raw)
			continue
		}
		if base.Attrs.Has("retells") {
			b.ErrorHintf(want.At, "retell the original instead",
				"scenario %q is itself a retelling; a retelling cannot retell another retelling", want.Raw)
			continue
		}

		valid := effectiveStepIDs(base, scenarios)
		known := sortedKeys(valid)
		for _, st := range sc.Steps {
			if len(st.Actions) > 0 {
				b.ErrorHintf(st.StartPos, "keep the animation in "+quote(base.Name)+" and leave only `desc` here",
					"step %q of retelling %q has actions; a retelling overrides narration only", st.EffectiveID(0), sc.Name)
				continue
			}
			if st.ID == "" {
				b.ErrorHintf(st.StartPos, "write `step <id> { desc: … }` naming the step of "+quote(base.Name)+" to re-explain",
					"a step of retelling %q has no id to match against", sc.Name)
				continue
			}
			if !valid[st.ID] {
				// Hint without a fix, deliberately: the text to replace is the
				// step's id token, and ast.Step records only the position of the
				// `step` keyword before it — a fix anchored there would rewrite
				// the keyword.
				hint, _ := suggestFrom(st.ID, known, "steps")
				b.ErrorHintf(st.StartPos, hint,
					"scenario %q has no step %q to retell", base.Name, st.ID)
				continue
			}
			if !st.Attrs.Has("desc") {
				b.WarnHintf(st.StartPos, "add `desc:` with the words this audience should hear",
					"step %q of retelling %q overrides nothing", st.ID, sc.Name)
			}
		}
	}
}

func quote(s string) string { return `"` + s + `"` }

// stepIndex finds a step of sc by the id it is addressable by.
func stepIndex(sc *ast.Scenario, id string) int {
	for i, st := range sc.Steps {
		if st.EffectiveID(i) == id {
			return i
		}
	}
	return -1
}

func effectiveIDs(sc *ast.Scenario) []string {
	out := make([]string, 0, len(sc.Steps))
	for i, st := range sc.Steps {
		out = append(out, st.EffectiveID(i))
	}
	return out
}

// effectiveStepIDs is every id a scenario is addressable by once inheritance is
// applied — the base's prefix plus its own steps at their shifted positions.
//
// It rehearses the same splice the compiler performs. Anything that reasons
// about which steps a scenario actually has at runtime has to go through here,
// or it will be wrong about exactly the scenarios variants exist for.
func effectiveStepIDs(sc *ast.Scenario, all []*ast.Scenario) map[string]bool {
	out := map[string]bool{}

	// A retelling is addressable by its base's step ids, not by its own entries:
	// those name the steps they re-explain rather than adding beats. Recursion
	// bottoms out immediately, since a base that is itself a retelling is
	// rejected.
	if want, ok := sc.Attrs.Get("retells"); ok {
		base := findScenario(all, want.Raw)
		if base == nil || base == sc || base.Attrs.Has("retells") {
			return out // unresolvable; already reported
		}
		return effectiveStepIDs(base, all)
	}

	prefix := 0
	if want, ok := sc.Attrs.Get("variant"); ok {
		if base := findScenario(all, want.Raw); base != nil && base != sc && !base.Attrs.Has("variant") {
			prefix = len(base.Steps)
			if until, has := sc.Attrs.Get("until"); has {
				if i := stepIndex(base, until.Raw); i >= 0 {
					prefix = i + 1
				} else {
					prefix = 0 // unresolvable; already reported
				}
			}
			for i := 0; i < prefix; i++ {
				out[base.Steps[i].EffectiveID(i)] = true
			}
		}
	}

	for i, st := range sc.Steps {
		out[st.EffectiveID(prefix+i)] = true
	}
	return out
}

// findScenario resolves a scenario by name, taking the first of any duplicates
// — which is what byName-style lookups elsewhere do, so the two agree.
func findScenario(all []*ast.Scenario, name string) *ast.Scenario {
	for _, sc := range all {
		if sc.Name == name {
			return sc
		}
	}
	return nil
}

// frameSet is the storyboard namespace a `scene` resolves against. It is built
// once per document and threaded through validation, the same way the symbol
// table is — a scene names a frame, never a node, and the two namespaces stay
// separate on purpose.
type frameSet struct {
	has   map[string]bool
	names []string
}

func (f *frameSet) contains(name string) bool { return f != nil && f.has[name] }

func (f *frameSet) list() []string {
	if f == nil {
		return nil
	}
	return f.names
}

// collectFrames flattens every storyboard block into one namespace. Multiple
// blocks merge; a name declared twice is an error, because a `scene` naming it
// would have no way to say which one it meant.
func collectFrames(storyboards []*ast.Storyboard, b *diag.Bag) *frameSet {
	out := &frameSet{has: map[string]bool{}}
	for _, sb := range storyboards {
		for _, f := range sb.Frames {
			if out.has[f.Name] {
				b.ErrorHintf(f.At, "frame names are shared across every storyboard block in a file",
					"duplicate frame %q", f.Name)
				continue
			}
			out.has[f.Name] = true
			out.names = append(out.names, f.Name)
		}
	}
	sort.Strings(out.names)
	return out
}

func validateActions(actions []ast.Action, t *symbol.Table, names []string, frames *frameSet, b *diag.Bag) {
	for _, a := range actions {
		if a.Kind == ast.ActionSeq {
			checkAttrs(a.Attrs, attrsFor(ast.ActionSeq), "seq", b)
			validateActions(a.Body, t, names, frames, b)
			continue
		}

		checkAttrs(a.Attrs, attrsFor(a.Kind), string(a.Kind), b)

		// A scene's target is a storyboard frame, not a diagram element, so it
		// resolves against its own namespace and never reaches the node lookup.
		if a.Kind == ast.ActionScene {
			for _, tgt := range a.Targets {
				if !frames.contains(tgt.Name) {
					hint, best := suggestFrom(tgt.Name, frames.list(), "frames")
					b.ErrorFixf(tgt.At, targetFix(tgt, best), hint,
						"no storyboard frame named %q", tgt.Name)
				}
			}
			continue
		}

		for _, tgt := range a.Targets {
			if _, ok := t.Node(tgt.Name); ok {
				continue
			}
			if _, ok := t.Group(tgt.Name); ok {
				continue
			}
			hint, best := suggest(tgt.Name, names)
			b.ErrorFixf(tgt.At, targetFix(tgt, best), hint,
				"%q is not a node in this diagram", tgt.Name)
		}

		if a.Kind == ast.ActionFlow {
			validateFlowHops(a, t, b)
		}
		if a.Kind == ast.ActionGauge {
			validateGauge(a, b)
		}
	}
}

// validateGauge checks the two attributes a gauge cannot do without.
//
// `label` is not decoration: it is half the identity of the reading, so two
// gauges on one node stay separate and a later `gauge` with the same label
// replaces rather than stacks. Without it the compiler has no slot to write.
func validateGauge(a ast.Action, b *diag.Bag) {
	const form = `write it as: gauge n3 { label: "term", value: 3 }`
	if a.Attrs.String("label") == "" {
		b.ErrorHintf(a.At, form, "gauge needs a label naming what it reads")
	}
	if !a.Attrs.Has("value") {
		b.ErrorHintf(a.At, form, "gauge needs a value")
	}
}

// validateFlowHops checks that each hop of a flow chain has an edge to travel
// along. An edge declared in the opposite direction counts: response paths
// routinely run backwards over a request edge.
func validateFlowHops(a ast.Action, t *symbol.Table, b *diag.Bag) {
	for i := 0; i+1 < len(a.Targets); i++ {
		from, to := a.Targets[i], a.Targets[i+1]
		if _, ok := t.Node(from.Name); !ok {
			continue // already reported as an unknown node
		}
		if _, ok := t.Node(to.Name); !ok {
			continue
		}
		if _, _, ok := t.FindEdge(from.Name, to.Name); !ok {
			b.ErrorHintf(to.At,
				"add `"+from.Name+" --> "+to.Name+"` to the diagram, or route the flow through nodes that are connected",
				"no edge between %q and %q to animate along", from.Name, to.Name)
			continue
		}

		// Several arrows between the same pair is the normal case in a sequence
		// diagram, where every message is its own line. The compiler consumes
		// them in order, which is usually what was meant — but "usually" is
		// worth saying out loud, because the alternative is silently animating
		// an arrow the author was not thinking of.
		if n := t.CountEdges(from.Name, to.Name); n > 1 && !a.Attrs.Has("msg") {
			b.WarnHintf(to.At,
				"flows between this pair are matched in order; add `msg: <n>` to pick one explicitly",
				"%d messages run between %q and %q", n, from.Name, to.Name)
		}
	}
}

// validateInteract resolves click bindings and view declarations.
//
// Like validateScenarios this works purely against symbol.Table, so a binding
// means the same thing over any diagram type. The one reference it cannot
// settle here is the file a `view` points at: that needs the filesystem, so
// pkg/loader reports it and this pass checks only what is knowable from the
// document alone.
func validateInteract(doc *ast.Document, t *symbol.Table, b *diag.Bag) {
	names := knownNames(t)

	views := make(map[string]bool, len(doc.Views))
	viewNames := make([]string, 0, len(doc.Views))
	// A view that is already broken does not also need telling that nobody
	// clicks it: the first problem is the one worth fixing.
	broken := make(map[*ast.ViewDecl]bool, len(doc.Views))

	for _, v := range doc.Views {
		if views[v.ID] {
			b.Errorf(v.At, "duplicate view id %q", v.ID)
			broken[v] = true
			continue
		}
		views[v.ID] = true
		viewNames = append(viewNames, v.ID)

		if v.Path == "" {
			b.ErrorHintf(v.PathAt, "give the path to a .dgm file",
				"view %q has an empty path", v.ID)
			broken[v] = true
		} else if filepath.IsAbs(v.Path) {
			b.ErrorHintf(v.PathAt, "use a path relative to this file so the document stays portable",
				"view %q has an absolute path", v.ID)
			broken[v] = true
		}
	}

	steps := knownSteps(doc.Scenarios)

	// A source may only be bound once: two behaviours on one click has no
	// sensible resolution, and silently picking the first would be worse than
	// saying so.
	bound := make(map[string]bool, len(doc.Interactions))

	for _, bd := range doc.Interactions {
		checkAttrs(bd.Attrs, bindingAttrs, "click", b)

		if !resolves(bd.Source.Name, t) {
			hint, best := suggest(bd.Source.Name, names)
			b.ErrorFixf(bd.Source.At, targetFix(bd.Source, best), hint,
				"%q is not a node in this diagram", bd.Source.Name)
		} else if bound[bd.Source.Name] {
			b.ErrorHintf(bd.Source.At, "give each clickable element a single behaviour",
				"%q already has a click binding", bd.Source.Name)
		}
		bound[bd.Source.Name] = true

		for _, tgt := range bd.Targets {
			switch bd.Kind {
			case ast.BindView:
				if !views[tgt.Name] {
					hint, best := suggestFrom(tgt.Name, viewNames, "views")
					b.ErrorFixf(tgt.At, targetFix(tgt, best), hint,
						"no view named %q is declared", tgt.Name)
				}
			case ast.BindReveal:
				if !resolves(tgt.Name, t) {
					hint, best := suggest(tgt.Name, names)
					b.ErrorFixf(tgt.At, targetFix(tgt, best), hint,
						"%q is not a node in this diagram", tgt.Name)
				} else if tgt.Name == bd.Source.Name {
					b.ErrorHintf(tgt.At, "a hidden element cannot be clicked to reveal itself",
						"%q reveals itself", tgt.Name)
				}
			case ast.BindStep:
				if !steps[tgt.Name] {
					hint, best := suggestFrom(tgt.Name, sortedKeys(steps), "steps")
					b.ErrorFixf(tgt.At, targetFix(tgt, best), hint,
						"no step named %q in this document", tgt.Name)
				}
			case ast.BindURL:
				validateURL(tgt, b)
			}
		}
	}

	for _, v := range doc.Views {
		if !broken[v] && !viewUsed(doc.Interactions, v.ID) {
			b.Warnf(v.At, "view %q is declared but nothing clicks through to it", v.ID)
		}
	}
}

// validateURL checks a click destination.
//
// An empty one is an error — nothing can be done with it. Anything other than
// http(s) is a warning rather than an error: a page opened inside a VS Code
// webview or an intranet may legitimately want a scheme this parser has never
// heard of, but a browser will refuse most of them from `window.open`, and
// silently doing nothing on click is the worst of the outcomes.
func validateURL(tgt ast.Target, b *diag.Bag) {
	if strings.TrimSpace(tgt.Name) == "" {
		b.ErrorHintf(tgt.At, "give a destination, e.g. \"https://example.com/runbook\"",
			"click url has an empty destination")
		return
	}
	lower := strings.ToLower(tgt.Name)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return
	}
	b.WarnHintf(tgt.At, "use an absolute http:// or https:// URL",
		"click url %q is not an http(s) address; the browser may refuse to open it", tgt.Name)
}

// resolves reports whether name is an element a click or reveal can address.
// Groups count: a subgraph is as clickable as a node.
func resolves(name string, t *symbol.Table) bool {
	if _, ok := t.Node(name); ok {
		return true
	}
	_, ok := t.Group(name)
	return ok
}

func viewUsed(bindings []*ast.Binding, id string) bool {
	for _, bd := range bindings {
		if bd.Kind != ast.BindView {
			continue
		}
		for _, tgt := range bd.Targets {
			if tgt.Name == id {
				return true
			}
		}
	}
	return false
}

// knownSteps collects the ids every step in the document is addressable by,
// including the positional fallback an unnamed step gets and the shift that
// inheriting a prefix applies to it.
func knownSteps(scenarios []*ast.Scenario) map[string]bool {
	out := make(map[string]bool)
	for _, sc := range scenarios {
		for id := range effectiveStepIDs(sc, scenarios) {
			out[id] = true
		}
	}
	return out
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func knownNames(t *symbol.Table) []string {
	var out []string
	for _, n := range t.Nodes() {
		out = append(out, n.ID)
	}
	for _, g := range t.Groups() {
		out = append(out, g.ID)
	}
	sort.Strings(out)
	return out
}

// suggest returns a "did you mean" hint for an unresolved node or group name,
// along with the candidate it named, if any.
func suggest(name string, candidates []string) (hint, best string) {
	return suggestFrom(name, candidates, "nodes")
}

// suggestFrom is suggest over an arbitrary namespace. noun names what the
// candidates are, so the fallback list reads correctly for views and steps as
// well as for nodes.
//
// best is the winning candidate and is returned only on the "did you mean"
// branch: the distance bound below is the whole fixability policy, so a caller
// building a machine-applicable edit needs no threshold of its own — an empty
// best means there was nothing confident enough to rewrite.
func suggestFrom(name string, candidates []string, noun string) (hint, best string) {
	bestDist := 1 << 30
	for _, c := range candidates {
		d := editDistance(strings.ToLower(name), strings.ToLower(c))
		if d < bestDist {
			best, bestDist = c, d
		}
	}
	// Only suggest when the names are genuinely close; otherwise list what
	// is available, which is more useful than a bad guess.
	if best != "" && bestDist <= len(name)/2+1 {
		return "did you mean " + best + "?", best
	}
	if len(candidates) == 0 {
		return "this document declares no " + noun, ""
	}
	shown := candidates
	if len(shown) > 8 {
		shown = shown[:8]
	}
	return "known " + noun + ": " + strings.Join(shown, ", "), ""
}

// The three fix constructors below turn a winning candidate into the edit that
// applies it. They all return the zero fix for an empty candidate, so every
// call site reads the same whether or not there was anything to suggest.

// targetFix replaces a misspelt reference — a node, group, frame, view alias or
// step id — where it was written. A target is always a bare identifier, so the
// text on disk is exactly its name.
func targetFix(t ast.Target, best string) diag.Fix {
	if best == "" {
		return diag.Fix{}
	}
	return diag.Fix{Pos: t.At, Old: t.Name, New: best}
}

// valueFix replaces a misspelt attribute value.
//
// A quoted value is stored unescaped and positioned at its opening quote, so
// the edit has to carry the quotes on both sides of the swap to line up with
// the bytes on disk. Where an escape makes the reconstruction differ from what
// was actually written, the applier's Old check simply fails and the fix is
// skipped — a wrong edit is far worse than one not offered.
func valueFix(v ast.Value, best string) diag.Fix {
	if best == "" {
		return diag.Fix{}
	}
	if v.Quoted {
		return diag.Fix{Pos: v.At, Old: `"` + v.Raw + `"`, New: `"` + best + `"`}
	}
	return diag.Fix{Pos: v.At, Old: v.Raw, New: best}
}

// attrKeyFix replaces a misspelt attribute key.
//
// The diagnostic is reported at the value, which is where the reader is
// looking, but the text to swap is the key — so the fix carries the key's own
// position. An attribute the parser synthesized was never typed and has no key
// position, which leaves nothing to rewrite.
func attrKeyFix(v ast.Value, key, best string) diag.Fix {
	if best == "" {
		return diag.Fix{}
	}
	return diag.Fix{Pos: v.KeyAt, Old: key, New: best}
}

func editDistance(a, b string) int {
	if a == b {
		return 0
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}
