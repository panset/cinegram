package parser

import (
	"strings"

	"github.com/tejaspanse/cinegram/pkg/ast"
	"github.com/tejaspanse/cinegram/pkg/diag"
	"github.com/tejaspanse/cinegram/pkg/registry"
	"github.com/tejaspanse/cinegram/pkg/source"
	"github.com/tejaspanse/cinegram/pkg/symbol"
)

func init() { registry.Register(stateParser{}) }

// stateParser handles Mermaid `stateDiagram-v2` and `stateDiagram` bodies.
//
// A state chart is the diagram type Cinegram's premise fits best: the picture
// is static, but "what happens when this event fires in this state" is exactly
// what a scenario is. So the model here is thicker than the sequence parser's —
// states, transitions, pseudostates and composites are all addressable — while
// everything a scenario has nothing to say about (notes, `direction`, `classDef`,
// concurrency dividers) still falls through to RawStmt and reprints verbatim.
//
// Both header spellings register here. Mermaid's v1 and v2 state renderers emit
// the same DOM in the version we vendor, so there is nothing for the runtime to
// distinguish and no reason to make an author who wrote the older keyword
// change it.
type stateParser struct{}

func (stateParser) Keywords() []string { return []string{"stateDiagram-v2", "stateDiagram"} }

// stateRawKeywords open a line the parser preserves verbatim: it affects
// rendering or accessibility but contributes no state and no transition.
var stateRawKeywords = map[string]bool{
	"classDef":       true,
	"class":          true,
	"style":          true,
	"accTitle":       true,
	"accDescr":       true,
	"accDescription": true,
}

// stereotypes are the pseudostates Mermaid draws with a shape of their own.
// Each is an ordinary node in the symbol table, carrying its stereotype as a
// Shape, so `flow who_closes -> FIN_WAIT_1` works exactly like any other flow.
//
// `<<history>>` and `<<deepHistory>>` are absent on purpose: Mermaid v11 does
// not render them, so accepting them would promise something the page cannot
// deliver.
var stereotypes = map[string]bool{
	"choice": true,
	"fork":   true,
	"join":   true,
}

func (p stateParser) Parse(c *source.Cursor, b *diag.Bag) (ast.Diagram, *symbol.Table) {
	table := symbol.NewTable()

	header, _ := c.Next()
	sd := &ast.StateDiagram{HeaderText: header.Text, StartPos: header.Start()}

	sd.Statements, _ = p.parseBody(c, b, table, "", false)
	return sd, table
}

// parseBody reads statements until it hits the composite's `}` (when nested), a
// top-level keyword, or end of input. scope is the enclosing composite id, "" at
// the top level. The returned closeText is the `}` line as written, so an
// emitter reprints what the author typed rather than a synthesized closer.
func (p stateParser) parseBody(c *source.Cursor, b *diag.Bag, t *symbol.Table,
	scope string, nested bool) (stmts []ast.Statement, closeText string) {

	for {
		line, ok := c.Peek()
		if !ok {
			if nested {
				b.ErrorHintf(c.EOFPos(), "add a `}` line to close the composite state",
					"unterminated composite state %q", scope)
			}
			return stmts, ""
		}

		code, _ := stripComment(line.Text)
		code = strings.TrimSpace(code)
		word := firstWord(code)

		// Blank and comment-only lines carry no meaning but do carry shape, and
		// a state diagram is usually written in paragraphs. Keeping them means
		// `cinegram mermaid` gives back the file the author recognises.
		if line.Blank() || code == "" {
			c.Next()
			stmts = append(stmts, &ast.RawStmt{Text: line.Text, StartPos: line.Start()})
			continue
		}

		// The registry contract: hand the cursor back UN-advanced at every
		// keyword the scenario half owns. Inside a composite the same words are
		// just state names, so the check is scoped to the top level.
		if isTopLevelKeyword(word) && !nested {
			return stmts, ""
		}

		if code == "}" {
			c.Next()
			if !nested {
				b.ErrorHintf(line.Start(), "remove it, or open a composite with `state X {` above",
					"unexpected `}` outside a composite state")
				stmts = append(stmts, &ast.RawStmt{Text: line.Text, StartPos: line.Start()})
				continue
			}
			return stmts, line.Text
		}

		c.Next()

		switch {
		case isDivider(code):
			// A `--` divider splits a composite into concurrent regions. It is
			// layout, not structure: the states on both sides stay ordinary
			// members of the enclosing composite, which is what makes a group
			// highlight cover the whole thing.
			stmts = append(stmts, &ast.RawStmt{Text: line.Text, StartPos: line.Start()})

		case word == "state":
			stmts = append(stmts, p.parseStateDecl(c, line, code, b, t, scope))

		// Notes are matched before transitions are looked for. `note right of A
		// : maps x --> y` contains something that looks exactly like a
		// transition operator, and a note is not a transition — the same hazard
		// the sequence parser documents for messages.
		case word == "note":
			stmts = append(stmts, p.parseNote(c, line, code, b)...)

		case word == "direction":
			stmts = append(stmts, &ast.RawStmt{Text: line.Text, StartPos: line.Start()})

		case stateRawKeywords[word]:
			stmts = append(stmts, &ast.RawStmt{Text: line.Text, StartPos: line.Start()})

		default:
			stmts = append(stmts, p.parseStateLine(line, code, b, t, scope))
		}
	}
}

// isDivider reports whether a line is a concurrency divider: two or more
// dashes and nothing else.
func isDivider(code string) bool {
	if len(code) < 2 {
		return false
	}
	return strings.Trim(code, "-") == ""
}

// parseStateLine classifies everything that is not a keyword: a description, a
// transition, or a bare state name.
func (p stateParser) parseStateLine(line source.Line, code string, b *diag.Bag,
	t *symbol.Table, scope string) ast.Statement {

	arrow := strings.Index(code, "-->")
	colon := strings.Index(code, ":")

	// A colon before the arrow makes this a description, not a transition. That
	// is Mermaid's own rule, and it is what lets `W : compares x --> y` describe
	// a state rather than declaring an edge to a state called `y`.
	if colon >= 0 && (arrow < 0 || colon < arrow) {
		return p.parseDescription(line, code, colon, b, t, scope)
	}

	if arrow >= 0 {
		return p.parseTransition(line, code, arrow, b, t, scope)
	}

	// A bare identifier declares a state. Anything with whitespace in it is
	// syntax this parser does not model, and round-trips instead of being
	// guessed at.
	if isStateID(code) {
		t.EnsureNode(code, line.Start())
		joinScope(t, scope, code)
		return &ast.StateStmt{ID: code, Label: code, Text: line.Text, StartPos: line.Start()}
	}

	return &ast.RawStmt{Text: line.Text, StartPos: line.Start()}
}

// parseDescription reads `id : some text`, which labels an existing state.
func (p stateParser) parseDescription(line source.Line, code string, colon int,
	b *diag.Bag, t *symbol.Table, scope string) ast.Statement {

	id := strings.TrimSpace(code[:colon])
	label := strings.TrimSpace(code[colon+1:])

	if id == startEndMarker {
		b.ErrorHintf(line.Start(), "`[*]` is a marker, not a state; describe the state it points at instead",
			"`[*]` cannot carry a description")
		return &ast.RawStmt{Text: line.Text, StartPos: line.Start()}
	}
	if !isStateID(id) {
		return &ast.RawStmt{Text: line.Text, StartPos: line.Start()}
	}

	t.DeclareNode(id, label, "state", "", line.Start())
	joinScope(t, scope, id)
	return &ast.StateStmt{ID: id, Label: label, Text: line.Text, StartPos: line.Start()}
}

// parseTransition reads `A --> B` with an optional `: event` label.
//
// The split is a leftmost split on the single `-->` operator rather than a call
// into link.go: a state diagram has exactly one transition operator, and
// Mermaid's flowchart link disambiguation would only find ways to misread it.
func (p stateParser) parseTransition(line source.Line, code string, arrow int,
	b *diag.Bag, t *symbol.Table, scope string) ast.Statement {

	from := strings.TrimSpace(code[:arrow])
	rest := code[arrow+len("-->"):]

	to, label := rest, ""
	if i := strings.Index(rest, ":"); i >= 0 {
		to = rest[:i]
		label = strings.TrimSpace(rest[i+1:])
	}
	to = strings.TrimSpace(to)

	if from == "" || to == "" {
		b.ErrorHintf(line.Start(), "write a transition as `A --> B` or `A --> B : event`",
			"transition is missing a state on one side")
		return &ast.RawStmt{Text: line.Text, StartPos: line.Start()}
	}
	// Chains would need every intermediate hop to be its own edge, and Mermaid
	// does not accept them either — so this is a real error rather than a
	// limitation worth hiding.
	if strings.Contains(to, "-->") {
		b.ErrorHintf(line.Start(), "split it into one transition per line: `a --> b` then `b --> c`",
			"a state transition joins exactly two states")
		return &ast.RawStmt{Text: line.Text, StartPos: line.Start()}
	}

	fromID, ok := p.endpoint(from, true, line, b, t, scope)
	if !ok {
		return &ast.RawStmt{Text: line.Text, StartPos: line.Start()}
	}
	toID, ok := p.endpoint(to, false, line, b, t, scope)
	if !ok {
		return &ast.RawStmt{Text: line.Text, StartPos: line.Start()}
	}

	// One edge per written occurrence. Two `A --> B` lines are two arrows on the
	// page, so they are two edges here, and the `msg: <n>` machinery that picks
	// between a sequence diagram's repeated messages picks between these with no
	// changes at all.
	t.AddEdge(&symbol.Edge{
		From: fromID, To: toID, Label: label,
		Style: string(ast.LinkNormal), Head: string(ast.ArrowNormal),
		At: line.Start(),
	})

	return &ast.TransitionStmt{
		From: fromID, To: toID, Label: label,
		Text: line.Text, StartPos: line.Start(),
	}
}

// endpoint resolves one side of a transition to a node id, registering it.
func (p stateParser) endpoint(text string, isSource bool, line source.Line,
	b *diag.Bag, t *symbol.Table, scope string) (string, bool) {

	if text == startEndMarker {
		return declareMarker(t, scope, isSource, line.Start()), true
	}
	if !isStateID(text) {
		b.ErrorHintf(line.Start(), "a transition endpoint is a single state name or `[*]`",
			"%q is not a state name", text)
		return "", false
	}
	t.EnsureNode(text, line.Start())
	joinScope(t, scope, text)
	return text, true
}

// parseStateDecl reads the `state` forms: a composite block, an alias, a
// stereotype, or a bare declaration.
func (p stateParser) parseStateDecl(c *source.Cursor, line source.Line, code string,
	b *diag.Bag, t *symbol.Table, scope string) ast.Statement {

	rest := strings.TrimSpace(strings.TrimPrefix(code, "state"))

	if strings.HasSuffix(rest, "{") {
		return p.parseComposite(c, line, strings.TrimSpace(strings.TrimSuffix(rest, "{")), b, t, scope)
	}

	// `state ID <<choice>>`
	if i := strings.Index(rest, "<<"); i >= 0 {
		id := strings.TrimSpace(rest[:i])
		stereo := strings.TrimSpace(rest[i:])
		stereo = strings.TrimSuffix(strings.TrimPrefix(stereo, "<<"), ">>")
		if !stereotypes[stereo] {
			b.WarnHintf(line.Start(), "Cinegram models <<choice>>, <<fork>> and <<join>>; anything else is passed through to Mermaid untouched",
				"unknown state stereotype %q", stereo)
			return &ast.RawStmt{Text: line.Text, StartPos: line.Start()}
		}
		if !isStateID(id) {
			return &ast.RawStmt{Text: line.Text, StartPos: line.Start()}
		}
		// The stereotype rides in as the Shape, which is exactly what it is:
		// the reason Mermaid draws a diamond instead of a box.
		t.DeclareNode(id, id, stereo, "", line.Start())
		joinScope(t, scope, id)
		return &ast.StateStmt{ID: id, Label: id, Stereotype: stereo,
			Text: line.Text, StartPos: line.Start()}
	}

	// `state "Label" as ID`
	id, label := splitStateAlias(rest)
	if !isStateID(id) {
		b.WarnHintf(line.Start(),
			"write `state ID`, `state \"Label\" as ID`, `state ID <<choice>>`, or open a block with `state ID {` and close it with `}` on its own line",
			"could not read this `state` declaration; it is passed through to Mermaid untouched")
		return &ast.RawStmt{Text: line.Text, StartPos: line.Start()}
	}
	t.DeclareNode(id, label, "state", "", line.Start())
	joinScope(t, scope, id)
	return &ast.StateStmt{ID: id, Label: label, Text: line.Text, StartPos: line.Start()}
}

// parseComposite reads a `state X {` block and everything up to its `}`.
func (p stateParser) parseComposite(c *source.Cursor, line source.Line, head string,
	b *diag.Bag, t *symbol.Table, parent string) ast.Statement {

	id, label := splitStateAlias(head)
	if id == "" {
		b.ErrorHintf(line.Start(), "give the composite a name, e.g. `state Teardown {`",
			"composite state is missing a name")
		id = "state"
	}

	t.DeclareGroup(id, label, parent, line.Start())
	// A composite nested in another composite is a child of it, so a group
	// reveal or highlight walks the whole tree without knowing how deep it goes.
	if parent != "" {
		t.AddToGroup(parent, id)
	}

	cs := &ast.StateCompositeStmt{
		ID:       id,
		Label:    label,
		Text:     line.Text,
		StartPos: line.Start(),
	}
	cs.Body, cs.CloseText = p.parseBody(c, b, t, id, true)

	for _, st := range cs.Body {
		if d, ok := st.(*ast.RawStmt); ok && strings.HasPrefix(d.Text, "direction ") {
			cs.Direction = strings.TrimSpace(strings.TrimPrefix(d.Text, "direction"))
		}
	}
	return cs
}

// parseNote reads either form of note, returning the lines it consumed.
//
// The multi-line body is consumed raw, with no classification of any kind: note
// prose is prose, and a sentence containing `scenario`, `-->` or a stray `}`
// must not steer the parser. Only `end note` ends it. The lines come back as
// ordinary RawStmts rather than a block statement because nothing downstream
// needs to know they belong together — they reprint in order either way.
func (p stateParser) parseNote(c *source.Cursor, line source.Line, code string, b *diag.Bag) []ast.Statement {
	stmts := []ast.Statement{&ast.RawStmt{Text: line.Text, StartPos: line.Start()}}
	if strings.Contains(code, ":") {
		return stmts
	}

	for {
		body, ok := c.Next()
		if !ok {
			b.ErrorHintf(c.EOFPos(), "add an `end note` line to close it",
				"unterminated note block")
			return stmts
		}
		stmts = append(stmts, &ast.RawStmt{Text: body.Text, StartPos: body.Start()})
		if body.Text == "end note" {
			return stmts
		}
	}
}

// startEndMarker is Mermaid's start/end pseudostate as written in source.
const startEndMarker = "[*]"

// declareMarker registers the node Mermaid will draw for a `[*]` endpoint and
// returns the id it will carry in the SVG.
//
// `[*]` is not a legal identifier, so a scenario cannot name it — yet the first
// arrow into a machine and the last one out of it are exactly the arrows a
// walkthrough wants to animate. Mermaid already names these nodes internally:
// `root_start` and `root_end` at the top level, `<Composite>_start` and
// `<Composite>_end` inside a composite. Adopting that spelling verbatim means
// the marker binds to the SVG by id like every other node, with nothing in the
// runtime translating between two vocabularies.
//
// Mermaid collapses every top-level `[*] -->` onto one start node and every
// `--> [*]` onto one end node, and declaring lazily here does the same.
func declareMarker(t *symbol.Table, scope string, isSource bool, at source.Pos) string {
	owner := scope
	if owner == "" {
		owner = "root"
	}
	kind := "end"
	if isSource {
		kind = "start"
	}
	id := owner + "_" + kind

	t.DeclareNode(id, kind, kind, "", at)
	joinScope(t, scope, id)
	return id
}

// joinScope adds id to the composite it was written inside, if any.
func joinScope(t *symbol.Table, scope, id string) {
	if scope != "" {
		t.AddToGroup(scope, id)
	}
}

// splitStateAlias reads `"Label" as ID`, `ID`, or `"Label"`, returning the id
// and its display label.
func splitStateAlias(rest string) (id, label string) {
	if rest == "" {
		return "", ""
	}
	if i := indexWordUnquoted(rest, "as"); i >= 0 {
		label = unquote(strings.TrimSpace(rest[:i]))
		id = unquote(strings.TrimSpace(rest[i+2:]))
		if label == "" {
			label = id
		}
		return id, label
	}
	id = unquote(rest)
	return id, id
}

// indexWordUnquoted is indexWord restricted to text outside double quotes.
//
// The label is the part that gets quoted, and a label is prose: `state "A label
// with as inside it" as s3` has two whole-word `as`, and only the second one is
// the keyword. Skipping quoted spans picks the right one without having to
// guess from the right-hand side.
func indexWordUnquoted(s, word string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			j := strings.IndexByte(s[i+1:], '"')
			if j < 0 {
				return -1
			}
			i += j + 1
			continue
		}
		if !strings.HasPrefix(s[i:], word) {
			continue
		}
		beforeOK := i == 0 || s[i-1] == ' ' || s[i-1] == '\t'
		after := i + len(word)
		afterOK := after == len(s) || s[after] == ' ' || s[after] == '\t'
		if beforeOK && afterOK {
			return i
		}
	}
	return -1
}

// isStateID reports whether s can be a state name: one token, no quotes, no
// brackets. Anything else is syntax this parser does not model.
func isStateID(s string) bool {
	if s == "" {
		return false
	}
	return strings.IndexAny(s, " \t\"[]{}<>:") < 0
}
