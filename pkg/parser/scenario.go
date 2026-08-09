package parser

import "github.com/tejaspanse/cinegram/pkg/ast"

// The scenario grammar is deliberately free of diagram vocabulary: it produces
// nothing but named references, durations and styles. That is what lets the
// same animation layer sit on top of a flowchart today and a sequence diagram
// later.
//
// One rule governs the whole block structure: a `{ ... }` block holds
// statements, and a statement is either `key: value` — an attribute of the
// enclosing construct — or an action. Actions inside a step start together;
// steps run one after another.

// actionArity records how each action keyword consumes its operands.
type actionArity int

const (
	arityChain  actionArity = iota // flow a -> b -> c
	arityTarget                    // highlight a, b
	arityNote                      // note a "text"
	arityBare                      // wait 500ms
	arityBlock                     // seq { ... }
)

var actionKinds = map[string]struct {
	kind  ast.ActionKind
	arity actionArity
}{
	"flow":      {ast.ActionFlow, arityChain},
	"highlight": {ast.ActionHighlight, arityTarget},
	"dim":       {ast.ActionDim, arityTarget},
	"pulse":     {ast.ActionPulse, arityTarget},
	"show":      {ast.ActionShow, arityTarget},
	"hide":      {ast.ActionHide, arityTarget},
	"focus":     {ast.ActionFocus, arityTarget},
	"note":      {ast.ActionNote, arityNote},
	"scene":     {ast.ActionScene, arityTarget},
	"wait":      {ast.ActionWait, arityBare},
	"seq":       {ast.ActionSeq, arityBlock},
	"set":       {ast.ActionSet, arityTarget},
	"gauge":     {ast.ActionGauge, arityTarget},
	"unset":     {ast.ActionUnset, arityTarget},
}

// knownActions lists the vocabulary for the unknown-action hint, in the order
// a reader would want to meet them rather than the map's.
const knownActions = "flow, highlight, note, dim, pulse, focus, show, hide, " +
	"set, gauge, unset, scene, wait, seq"

// parseScenario reads one `scenario` block. The block is not brace-delimited:
// it ends at the first token that is not a `step`, which is what lets a `view`
// or `interact` block follow it without a terminator.
func parseScenario(s *scanner) *ast.Scenario {
	kw := s.next() // "scenario"
	sc := &ast.Scenario{StartPos: kw.at}

	if t := s.peek(); t.kind == tokString || t.kind == tokIdent {
		sc.Name = s.next().text
	}
	if s.at("{") {
		sc.Attrs = parseAttrBlock(s)
	}
	if s.peek().kind == tokNewline {
		s.next()
	}

	for {
		s.skipNewlines()
		if !s.atKeyword("step") {
			return sc
		}
		if st := parseStep(s); st != nil {
			sc.Steps = append(sc.Steps, st)
		}
	}
}

func parseStep(s *scanner) *ast.Step {
	kw := s.next() // "step"
	st := &ast.Step{StartPos: kw.at}

	// `step <id> "<name>"`, `step "<name>"`, or `step <id>` — all optional.
	if t := s.peek(); t.kind == tokIdent {
		st.ID = s.next().text
	}
	if t := s.peek(); t.kind == tokString {
		st.Name = s.next().text
	}
	if st.Name == "" {
		st.Name = st.ID
	}

	if !s.expect("{") {
		s.skipToLineEnd()
		return st
	}
	st.Actions, st.Attrs = parseBlock(s)
	return st
}

// parseBlock reads statements until the matching `}`, splitting them into
// actions and attributes of the enclosing construct.
func parseBlock(s *scanner) ([]ast.Action, ast.Attrs) {
	var actions []ast.Action
	var attrs ast.Attrs

	for {
		s.skipNewlines()
		t := s.peek()
		if t.kind == tokEOF {
			s.bag.ErrorHintf(t.at, "add the closing `}`", "unterminated block")
			return actions, attrs
		}
		if s.accept("}") {
			return actions, attrs
		}

		// `key: value` sets an attribute on the enclosing step or scenario.
		if t.kind == tokIdent && s.toks[s.i+1].kind == tokPunct && s.toks[s.i+1].text == ":" {
			key := s.next()
			s.next() // ':'
			if v, ok := parseValue(s); ok {
				attrs.Set(key.text, v)
			}
			s.accept(",")
			continue
		}

		if t.kind != tokIdent {
			s.bag.Errorf(t.at, "expected an action or attribute but found %s", describe(t))
			s.skipToLineEnd()
			continue
		}
		if a, ok := parseAction(s); ok {
			actions = append(actions, a)
		}
	}
}

func parseAction(s *scanner) (ast.Action, bool) {
	kw := s.next()
	spec, known := actionKinds[kw.text]
	if !known {
		s.bag.ErrorHintf(kw.at, "known actions: "+knownActions,
			"unknown action %q", kw.text)
		s.skipToLineEnd()
		return ast.Action{}, false
	}

	a := ast.Action{Kind: spec.kind, At: kw.at}

	switch spec.arity {
	case arityChain:
		for {
			t := s.peek()
			if t.kind != tokIdent {
				s.bag.ErrorHintf(t.at, "write a hop chain such as `flow client -> lb -> ing`",
					"expected a node name but found %s", describe(t))
				s.skipToLineEnd()
				return ast.Action{}, false
			}
			s.next()
			a.Targets = append(a.Targets, ast.Target{Name: t.text, At: t.at})
			if !s.accept("->") {
				break
			}
		}
		if len(a.Targets) < 2 {
			s.bag.ErrorHintf(kw.at, "a flow needs at least two hops, e.g. `flow a -> b`",
				"flow has only %d node", len(a.Targets))
			s.skipToLineEnd()
			return ast.Action{}, false
		}

	case arityTarget:
		for {
			t := s.peek()
			if t.kind != tokIdent {
				break
			}
			s.next()
			a.Targets = append(a.Targets, ast.Target{Name: t.text, At: t.at})
			if !s.accept(",") {
				break
			}
		}
		if len(a.Targets) == 0 {
			s.bag.Errorf(kw.at, "%s needs at least one node", kw.text)
			s.skipToLineEnd()
			return ast.Action{}, false
		}

	case arityNote:
		t := s.peek()
		if t.kind != tokIdent {
			s.bag.ErrorHintf(t.at, `write it as: note <node> "text"`,
				"expected a node name but found %s", describe(t))
			s.skipToLineEnd()
			return ast.Action{}, false
		}
		s.next()
		a.Targets = append(a.Targets, ast.Target{Name: t.text, At: t.at})

		body := s.peek()
		if body.kind != tokString {
			s.bag.ErrorHintf(body.at, `note text must be quoted, e.g. note ing "host: api.example.com"`,
				"expected note text but found %s", describe(body))
			s.skipToLineEnd()
			return ast.Action{}, false
		}
		s.next()
		a.Text = body.text

	case arityBare:
		t := s.peek()
		if t.kind != tokValue {
			s.bag.ErrorHintf(t.at, "write a duration such as `wait 500ms`",
				"expected a duration but found %s", describe(t))
			s.skipToLineEnd()
			return ast.Action{}, false
		}
		s.next()
		a.Attrs.Set("dur", ast.Value{Raw: t.text, At: t.at})

	case arityBlock:
		if !s.expect("{") {
			s.skipToLineEnd()
			return ast.Action{}, false
		}
		body, attrs := parseBlock(s)
		a.Body = body
		a.Attrs = attrs
		return a, true
	}

	if s.at("{") {
		merged := parseAttrBlock(s)
		for _, k := range merged.Keys() {
			v, _ := merged.Get(k)
			a.Attrs.Set(k, v)
		}
	}
	return a, true
}

// parseAttrBlock reads a `{ key: value, ... }` block. Pairs may be separated by
// commas, newlines, or both, so multi-line attribute blocks read naturally.
func parseAttrBlock(s *scanner) ast.Attrs {
	var attrs ast.Attrs
	if !s.expect("{") {
		return attrs
	}
	for {
		s.skipNewlines()
		t := s.peek()
		if t.kind == tokEOF {
			s.bag.ErrorHintf(t.at, "add the closing `}`", "unterminated attribute block")
			return attrs
		}
		if s.accept("}") {
			return attrs
		}
		if t.kind != tokIdent {
			s.bag.Errorf(t.at, "expected an attribute name but found %s", describe(t))
			s.skipToLineEnd()
			continue
		}
		key := s.next()
		if !s.expect(":") {
			s.skipToLineEnd()
			continue
		}
		if v, ok := parseValue(s); ok {
			attrs.Set(key.text, v)
		}
		s.accept(",")
	}
}

// parseValue reads a single attribute value.
func parseValue(s *scanner) (ast.Value, bool) {
	t := s.peek()
	switch t.kind {
	case tokString, tokIdent, tokValue:
		s.next()
		return ast.Value{Raw: t.text, At: t.at}, true
	default:
		s.bag.Errorf(t.at, "expected a value but found %s", describe(t))
		s.skipToLineEnd()
		return ast.Value{}, false
	}
}
