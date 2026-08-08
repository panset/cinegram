package parser

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/tejaspanse/diagramator/pkg/ast"
	"github.com/tejaspanse/diagramator/pkg/diag"
	"github.com/tejaspanse/diagramator/pkg/symbol"
)

// validateScenarios resolves every scenario reference against the symbol table
// the diagram parser produced.
//
// This pass is the only place the two halves of a document meet, and it works
// purely against symbol.Table — no flowchart types appear here, which is what
// keeps the animation layer reusable across diagram types.
func validateScenarios(scenarios []*ast.Scenario, t *symbol.Table, b *diag.Bag) {
	names := knownNames(t)

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
			validateActions(st.Actions, t, names, b)
		}
		if len(sc.Steps) == 0 {
			b.Warnf(sc.StartPos, "scenario %q has no steps", sc.Name)
		}
	}
}

func validateActions(actions []ast.Action, t *symbol.Table, names []string, b *diag.Bag) {
	for _, a := range actions {
		if a.Kind == ast.ActionSeq {
			checkAttrs(a.Attrs, attrsFor(ast.ActionSeq), "seq", b)
			validateActions(a.Body, t, names, b)
			continue
		}

		checkAttrs(a.Attrs, attrsFor(a.Kind), string(a.Kind), b)

		for _, tgt := range a.Targets {
			if _, ok := t.Node(tgt.Name); ok {
				continue
			}
			if _, ok := t.Group(tgt.Name); ok {
				continue
			}
			b.ErrorHintf(tgt.At, suggest(tgt.Name, names),
				"%q is not a node in this diagram", tgt.Name)
		}

		if a.Kind == ast.ActionFlow {
			validateFlowHops(a, t, b)
		}
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
			b.ErrorHintf(bd.Source.At, suggest(bd.Source.Name, names),
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
					b.ErrorHintf(tgt.At, suggestFrom(tgt.Name, viewNames, "views"),
						"no view named %q is declared", tgt.Name)
				}
			case ast.BindReveal:
				if !resolves(tgt.Name, t) {
					b.ErrorHintf(tgt.At, suggest(tgt.Name, names),
						"%q is not a node in this diagram", tgt.Name)
				} else if tgt.Name == bd.Source.Name {
					b.ErrorHintf(tgt.At, "a hidden element cannot be clicked to reveal itself",
						"%q reveals itself", tgt.Name)
				}
			case ast.BindStep:
				if !steps[tgt.Name] {
					b.ErrorHintf(tgt.At, suggestFrom(tgt.Name, sortedKeys(steps), "steps"),
						"no step named %q in this document", tgt.Name)
				}
			}
		}
	}

	for _, v := range doc.Views {
		if !broken[v] && !viewUsed(doc.Interactions, v.ID) {
			b.Warnf(v.At, "view %q is declared but nothing clicks through to it", v.ID)
		}
	}
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
// including the positional fallback an unnamed step gets.
func knownSteps(scenarios []*ast.Scenario) map[string]bool {
	out := make(map[string]bool)
	for _, sc := range scenarios {
		for i, st := range sc.Steps {
			out[st.EffectiveID(i)] = true
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

// suggest returns a "did you mean" hint for an unresolved node or group name.
func suggest(name string, candidates []string) string {
	return suggestFrom(name, candidates, "nodes")
}

// suggestFrom is suggest over an arbitrary namespace. noun names what the
// candidates are, so the fallback list reads correctly for views and steps as
// well as for nodes.
func suggestFrom(name string, candidates []string, noun string) string {
	best, bestDist := "", 1<<30
	for _, c := range candidates {
		d := editDistance(strings.ToLower(name), strings.ToLower(c))
		if d < bestDist {
			best, bestDist = c, d
		}
	}
	// Only suggest when the names are genuinely close; otherwise list what
	// is available, which is more useful than a bad guess.
	if best != "" && bestDist <= len(name)/2+1 {
		return "did you mean " + best + "?"
	}
	if len(candidates) == 0 {
		return "this document declares no " + noun
	}
	shown := candidates
	if len(shown) > 8 {
		shown = shown[:8]
	}
	return "known " + noun + ": " + strings.Join(shown, ", ")
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
