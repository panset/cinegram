package parser

import (
	"sort"
	"strings"

	"github.com/tejaspanse/diagramator/pkg/ast"
	"github.com/tejaspanse/diagramator/pkg/diag"
	"github.com/tejaspanse/diagramator/pkg/symbol"
	"github.com/tejaspanse/diagramator/pkg/units"
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
			checkAttrs(a.Attrs, actionAttrs, "seq", b)
			validateActions(a.Body, t, names, b)
			continue
		}

		checkAttrs(a.Attrs, actionAttrs, string(a.Kind), b)

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

// Attribute vocabularies. Unknown keys are warnings rather than errors so that
// a forward-looking source file still compiles against an older binary.
var (
	scenarioAttrs = map[string]string{
		"speed": "float", "loop": "bool", "autoplay": "bool", "style": "string",
	}
	stepAttrs = map[string]string{
		"dur": "duration", "delay": "duration", "style": "string", "label": "string",
	}
	actionAttrs = map[string]string{
		"label": "string", "dur": "duration", "delay": "duration", "at": "duration",
		"style": "string", "color": "string", "ease": "string",
		"repeat": "float", "bidi": "bool",
	}
)

func checkAttrs(attrs ast.Attrs, allowed map[string]string, what string, b *diag.Bag) {
	for _, k := range attrs.Keys() {
		v, _ := attrs.Get(k)
		typ, ok := allowed[k]
		if !ok {
			b.Warnf(v.At, "unknown %s attribute %q", what, k)
			continue
		}
		var err error
		switch typ {
		case "duration":
			_, err = units.ParseMillis(v.Raw)
		case "float":
			_, err = units.ParseFloat(v.Raw)
		case "bool":
			_, err = units.ParseBool(v.Raw)
		}
		if err != nil {
			b.Errorf(v.At, "attribute %q: %v", k, err)
		}
	}
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

// suggest returns a "did you mean" hint for an unresolved name.
func suggest(name string, candidates []string) string {
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
		return "the diagram declares no nodes"
	}
	shown := candidates
	if len(shown) > 8 {
		shown = shown[:8]
	}
	return "known nodes: " + strings.Join(shown, ", ")
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
