package parser

import (
	"github.com/tejaspanse/diagramator/pkg/ast"
	"github.com/tejaspanse/diagramator/pkg/diag"
	"github.com/tejaspanse/diagramator/pkg/symbol"
)

// Coverage diagnostics answer a different question from the rest of validation.
// Everything else asks "does this name resolve?"; these ask "did you mean to
// leave this out?" — the mistakes that produce a document which compiles
// perfectly and animates less than its author thought.
//
// They are warnings without exception. Every one of them has a legitimate
// answer of "yes, on purpose".

// validateCoverage reports the omissions a resolving pass cannot see.
func validateCoverage(doc *ast.Document, t *symbol.Table, b *diag.Bag) {
	checkUnreferenced(doc, t, b)
	checkStepScope(doc, b)
	checkScenarioNames(doc, b)
}

// checkUnreferenced finds elements no part of the document ever mentions.
//
// A node wired into an edge is part of the picture whether or not a scenario
// animates it, so being an edge endpoint counts as a reference. What this
// catches is the node declared and then forgotten — the one whose absence from
// the walkthrough is a typo in the walkthrough rather than a decision.
func checkUnreferenced(doc *ast.Document, t *symbol.Table, b *diag.Bag) {
	seen := map[string]bool{}

	for _, e := range t.Edges() {
		seen[e.From] = true
		seen[e.To] = true
	}
	for _, sc := range doc.Scenarios {
		for _, st := range sc.Steps {
			markActions(st.Actions, seen)
		}
	}
	for _, bd := range doc.Interactions {
		seen[bd.Source.Name] = true
		// Only reveal targets are diagram elements. A view binding's target is
		// a view alias — a different namespace — and marking it here would let
		// a node that happens to share the alias's name escape the warning.
		if bd.Kind == ast.BindReveal {
			for _, tgt := range bd.Targets {
				seen[tgt.Name] = true
			}
		}
	}

	for _, n := range t.Nodes() {
		if seen[n.ID] {
			continue
		}
		b.WarnHintf(n.At, "is "+n.ID+" supposed to be animated, or connected to something?",
			"%q is declared but nothing references it", n.ID)
	}

	// A subgraph is a container, so it counts as referenced when anything
	// inside it is. Otherwise every purely organisational grouping — which is
	// most of them — would be reported.
	for _, g := range t.Groups() {
		if seen[g.ID] || groupTouched(g, t, seen) {
			continue
		}
		b.WarnHintf(g.At, "is "+g.ID+" supposed to be animated, or to hold something?",
			"subgraph %q is declared but nothing references it or anything inside it", g.ID)
	}
}

func groupTouched(g *symbol.Group, t *symbol.Table, seen map[string]bool) bool {
	for _, child := range g.Children {
		if seen[child] {
			return true
		}
		if sub, ok := t.Group(child); ok && groupTouched(sub, t, seen) {
			return true
		}
	}
	return false
}

func markActions(actions []ast.Action, seen map[string]bool) {
	for _, a := range actions {
		for _, tgt := range a.Targets {
			seen[tgt.Name] = true
		}
		markActions(a.Body, seen)
	}
}

// checkStepScope catches a `click … -> step` whose target is not in the
// scenario the reader will be looking at.
//
// The runtime binds a step click against the *selected* scenario, and the
// selected scenario starts as the first one. A binding to a step that only
// exists further down the file resolves during validation — the step is real —
// and then does nothing when clicked, which is the worst kind of working.
func checkStepScope(doc *ast.Document, b *diag.Bag) {
	if len(doc.Scenarios) < 2 {
		return
	}

	first := map[string]bool{}
	for i, st := range doc.Scenarios[0].Steps {
		first[st.EffectiveID(i)] = true
	}

	for _, bd := range doc.Interactions {
		if bd.Kind != ast.BindStep {
			continue
		}
		for _, tgt := range bd.Targets {
			if first[tgt.Name] {
				continue
			}
			if owner, ok := stepOwner(doc, tgt.Name); ok {
				b.WarnHintf(tgt.At,
					"a step click acts on the scenario currently selected; give scenario "+
						owner+" a step of this name too, or bind to a step in the first scenario",
					"step %q is only in scenario %q, so this click does nothing until that scenario is chosen",
					tgt.Name, owner)
			}
		}
	}
}

// stepOwner names the first scenario containing a step id.
func stepOwner(doc *ast.Document, id string) (string, bool) {
	for _, sc := range doc.Scenarios {
		for i, st := range sc.Steps {
			if st.EffectiveID(i) == id {
				name := sc.Name
				if name == "" {
					name = "(unnamed)"
				}
				return name, true
			}
		}
	}
	return "", false
}

// checkScenarioNames reports duplicates. The picker lists scenarios by name and
// a deep link names one by id, so two identical names leave the reader choosing
// between two indistinguishable entries.
func checkScenarioNames(doc *ast.Document, b *diag.Bag) {
	seen := map[string]bool{}
	for _, sc := range doc.Scenarios {
		if sc.Name == "" {
			continue
		}
		if seen[sc.Name] {
			b.WarnHintf(sc.StartPos, "give each scenario a distinct name; the picker shows nothing else",
				"duplicate scenario name %q", sc.Name)
		}
		seen[sc.Name] = true
	}
}
