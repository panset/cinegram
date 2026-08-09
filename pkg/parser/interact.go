package parser

import (
	"github.com/tejaspanse/cinegram/pkg/ast"
	"github.com/tejaspanse/cinegram/pkg/diag"
	"github.com/tejaspanse/cinegram/pkg/source"
)

// Interaction shares the scenario half's discipline: `interact` produces named
// references and nothing else, so a click binding means the same thing over a
// flowchart as it will over any diagram type added later.
//
// Two constructs live here. `view <id> "<title>" from "<path>"` declares a
// document a click can drill into — the path is recorded as written and
// resolved by pkg/loader, which keeps this package free of I/O. `interact { }`
// holds the bindings themselves.

// topLevelKeywords are the block openers that belong to this package rather
// than to a diagram parser.
var topLevelKeywords = map[string]bool{
	"scenario":   true,
	"storyboard": true,
	"view":       true,
	"interact":   true,
}

// isTopLevelKeyword reports whether word opens a block that belongs to the
// scenario half of the language. A registry.DiagramParser must stop and hand
// the cursor back when it reaches one.
func isTopLevelKeyword(word string) bool { return topLevelKeywords[word] }

// bindingVerbs maps a click verb to the kind it produces.
var bindingVerbs = map[string]ast.BindingKind{
	"view":   ast.BindView,
	"reveal": ast.BindReveal,
	"step":   ast.BindStep,
	"url":    ast.BindURL,
}

// topLevel is everything the scenario half of a document contributes.
type topLevel struct {
	Scenarios    []*ast.Scenario
	Storyboards  []*ast.Storyboard
	Views        []*ast.ViewDecl
	Interactions []*ast.Binding
}

// parseTopLevel reads every scenario, view declaration and interact block from
// the cursor to end of input. The three may appear in any order.
func parseTopLevel(c *source.Cursor, b *diag.Bag) topLevel {
	s := newScanner(c, b)
	var out topLevel

	for {
		s.skipNewlines()
		t := s.peek()
		if t.kind == tokEOF {
			return out
		}

		switch {
		case s.atKeyword("scenario"):
			if sc := parseScenario(s); sc != nil {
				out.Scenarios = append(out.Scenarios, sc)
			}
		case s.atKeyword("storyboard"):
			if sb := parseStoryboard(s); sb != nil {
				out.Storyboards = append(out.Storyboards, sb)
			}
		case s.atKeyword("view"):
			if v := parseViewDecl(s); v != nil {
				out.Views = append(out.Views, v)
			}
		case s.atKeyword("interact"):
			out.Interactions = append(out.Interactions, parseInteract(s)...)
		default:
			b.ErrorHintf(t.at, "blocks start with `scenario`, `storyboard`, `view` or `interact`",
				"expected `scenario`, `storyboard`, `view` or `interact` but found %s", describe(t))
			s.skipToLineEnd()
			s.skipNewlines()
		}
	}
}

// parseViewDecl reads `view <id> ["<title>"] from "<path>"`.
func parseViewDecl(s *scanner) *ast.ViewDecl {
	kw := s.next() // "view"
	v := &ast.ViewDecl{At: kw.at}

	id := s.peek()
	if id.kind != tokIdent {
		s.bag.ErrorHintf(id.at, "write `view <id> from \"file.dgm\"`",
			"expected a view name but found %s", describe(id))
		s.skipToLineEnd()
		return nil
	}
	v.ID = s.next().text

	if t := s.peek(); t.kind == tokString {
		v.Title = s.next().text
	}

	if !s.atKeyword("from") {
		t := s.peek()
		s.bag.ErrorHintf(t.at, "write `view "+v.ID+" from \"file.dgm\"`",
			"expected `from` but found %s", describe(t))
		s.skipToLineEnd()
		return nil
	}
	s.next() // "from"

	path := s.peek()
	if path.kind != tokString {
		s.bag.ErrorHintf(path.at, "the path must be quoted, e.g. `from \"pod-a.dgm\"`",
			"expected a quoted file path but found %s", describe(path))
		s.skipToLineEnd()
		return nil
	}
	s.next()
	v.Path = path.text
	v.PathAt = path.at

	return v
}

// parseInteract reads an `interact { ... }` block of click bindings.
func parseInteract(s *scanner) []*ast.Binding {
	kw := s.next() // "interact"
	if !s.expect("{") {
		s.skipToLineEnd()
		return nil
	}

	var out []*ast.Binding
	for {
		s.skipNewlines()
		t := s.peek()
		if t.kind == tokEOF {
			s.bag.ErrorHintf(t.at, "add the closing `}`", "unterminated interact block")
			return out
		}
		if s.accept("}") {
			if len(out) == 0 {
				s.bag.Warnf(kw.at, "interact block has no bindings")
			}
			return out
		}
		if !s.atKeyword("click") {
			s.bag.ErrorHintf(t.at, "every binding starts with `click`",
				"expected `click` but found %s", describe(t))
			s.skipToLineEnd()
			continue
		}
		if bd, ok := parseBinding(s); ok {
			out = append(out, bd)
		}
	}
}

// parseBinding reads `click <source> -> <verb> <targets> [{ attrs }]`.
func parseBinding(s *scanner) (*ast.Binding, bool) {
	kw := s.next() // "click"
	bd := &ast.Binding{At: kw.at}

	src := s.peek()
	if src.kind != tokIdent {
		s.bag.ErrorHintf(src.at, "name the node or group that should become clickable",
			"expected a click source but found %s", describe(src))
		s.skipToLineEnd()
		return nil, false
	}
	s.next()
	bd.Source = ast.Target{Name: src.text, At: src.at}

	if !s.expect("->") {
		s.skipToLineEnd()
		return nil, false
	}

	verb := s.peek()
	kind, known := bindingVerbs[verb.text]
	if verb.kind != tokIdent || !known {
		s.bag.ErrorHintf(verb.at, "known click targets: view, reveal, step, url",
			"expected a click target but found %s", describe(verb))
		s.skipToLineEnd()
		return nil, false
	}
	s.next()
	bd.Kind = kind

	// A URL is the one target that is not a name in this document, so it is
	// quoted text rather than an identifier — `//`, `?` and `&` would not
	// survive the tokenizer otherwise.
	if kind == ast.BindURL {
		dest := s.peek()
		if dest.kind != tokString {
			s.bag.ErrorHintf(dest.at, `quote the destination, e.g. click svc -> url "https://example.com/runbook"`,
				"expected a quoted URL but found %s", describe(dest))
			s.skipToLineEnd()
			return nil, false
		}
		s.next()
		bd.Targets = append(bd.Targets, ast.Target{Name: dest.text, At: dest.at})
		if s.at("{") {
			bd.Attrs = parseAttrBlock(s)
		}
		return bd, true
	}

	// `reveal` takes a comma-separated set; `view` and `step` take one name.
	// Parsing the list either way keeps the error for a stray comma at the
	// binding level, where it can say what the verb actually accepts.
	for {
		t := s.peek()
		if t.kind != tokIdent {
			s.bag.Errorf(t.at, "expected a name after `%s` but found %s", verb.text, describe(t))
			s.skipToLineEnd()
			return nil, false
		}
		s.next()
		bd.Targets = append(bd.Targets, ast.Target{Name: t.text, At: t.at})
		if !s.accept(",") {
			break
		}
	}
	if kind != ast.BindReveal && len(bd.Targets) > 1 {
		s.bag.ErrorHintf(bd.Targets[1].At, "only `reveal` accepts several targets",
			"`%s` takes one name, got %d", verb.text, len(bd.Targets))
		return nil, false
	}

	if s.at("{") {
		bd.Attrs = parseAttrBlock(s)
	}
	return bd, true
}
