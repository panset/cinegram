package parser

import (
	"strings"

	"github.com/tejaspanse/cinegram/pkg/ast"
	"github.com/tejaspanse/cinegram/pkg/diag"
	"github.com/tejaspanse/cinegram/pkg/registry"
	"github.com/tejaspanse/cinegram/pkg/source"
	"github.com/tejaspanse/cinegram/pkg/symbol"
)

func init() { registry.Register(sequenceParser{}) }

// sequenceParser handles Mermaid `sequenceDiagram` bodies.
//
// It exists to test the claim the architecture makes: that a second diagram
// type costs one parser and nothing else. Everything downstream — scenarios,
// `desc`, `set`/`gauge`, `focus`, interaction, narration, frame capture —
// works against symbol.Table alone, so none of it changes here.
//
// The model is deliberately thin. Participants become nodes and messages become
// edges; `Note`, `activate`, `loop`/`alt`/`opt`/`end` and `box` are recorded as
// RawStmt and reprinted verbatim. They are structure a reader needs and an
// animation has nothing to say about, and modelling them would buy nothing but
// ways to lose them.
type sequenceParser struct{}

func (sequenceParser) Keywords() []string { return []string{"sequenceDiagram"} }

// seqArrows are Mermaid's message operators, longest first so that `-->>`
// is never mistaken for `-->` followed by a stray `>`.
var seqArrows = []string{
	"<<-->>", "<<->>",
	"-->>", "--)", "--x", "-->",
	"->>", "-)", "-x", "->",
}

// seqKeywords open a line that is structure rather than a message.
//
// Matching on the keyword before looking for an arrow matters: `Note right of
// A: maps x->y` contains something that looks exactly like a message operator,
// and a note is not a message.
var seqKeywords = map[string]bool{
	"note": true, "activate": true, "deactivate": true,
	"loop": true, "alt": true, "else": true, "opt": true,
	"par": true, "and": true, "critical": true, "option": true,
	"break": true, "rect": true, "box": true, "end": true,
	"autonumber": true, "create": true, "destroy": true,
	"links": true, "link": true, "properties": true, "details": true,
}

func (p sequenceParser) Parse(c *source.Cursor, b *diag.Bag) (ast.Diagram, *symbol.Table) {
	table := symbol.NewTable()

	header, _ := c.Next()
	seq := &ast.Sequence{HeaderText: header.Text, StartPos: header.Start()}

	for {
		line, ok := c.Peek()
		if !ok {
			return seq, table
		}
		if line.Blank() {
			c.Next()
			seq.Statements = append(seq.Statements, &ast.RawStmt{Text: "", StartPos: line.Start()})
			continue
		}

		code, _ := stripComment(line.Text)
		code = strings.TrimSpace(code)
		word := firstWord(code)

		// The registry contract: hand the cursor back at every keyword the
		// scenario half owns, not just at `scenario`.
		if isTopLevelKeyword(word) {
			return seq, table
		}

		c.Next()

		switch {
		case code == "":
			seq.Statements = append(seq.Statements, &ast.RawStmt{Text: line.Text, StartPos: line.Start()})

		case word == "participant" || word == "actor":
			seq.Statements = append(seq.Statements, parseParticipant(line, code, word == "actor", table))

		case seqKeywords[strings.ToLower(word)]:
			seq.Statements = append(seq.Statements, &ast.RawStmt{Text: line.Text, StartPos: line.Start()})

		default:
			if st, ok := parseMessage(line, code, table, b); ok {
				seq.Statements = append(seq.Statements, st)
			} else {
				seq.Statements = append(seq.Statements, &ast.RawStmt{Text: line.Text, StartPos: line.Start()})
			}
		}
	}
}

// parseParticipant reads `participant A` or `participant A as Label`.
func parseParticipant(line source.Line, code string, actor bool, t *symbol.Table) ast.Statement {
	rest := strings.TrimSpace(code[len(firstWord(code)):])

	id, label := rest, ""
	if i := indexWord(rest, "as"); i >= 0 {
		id = strings.TrimSpace(rest[:i])
		label = strings.TrimSpace(rest[i+2:])
	}
	id = strings.TrimSpace(id)
	if label == "" {
		label = id
	}
	if id != "" {
		t.DeclareNode(id, label, "actor", "", line.Start())
	}

	return &ast.ParticipantStmt{
		ID: id, Label: label, Actor: actor,
		Text: line.Text, StartPos: line.Start(),
	}
}

// indexWord finds `word` in s as a whole space-delimited word, returning its
// byte offset. `participant Address as A` must not split on the "as" inside
// "Address".
func indexWord(s, word string) int {
	for i := 0; i+len(word) <= len(s); i++ {
		if s[i:i+len(word)] != word {
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

// parseMessage reads `A->>B: text`, registering one edge per occurrence.
func parseMessage(line source.Line, code string, t *symbol.Table, b *diag.Bag) (ast.Statement, bool) {
	at, op := findArrow(code)
	if at < 0 {
		return nil, false
	}

	from := strings.TrimSpace(code[:at])
	rest := code[at+len(op):]

	to, label := rest, ""
	if i := strings.Index(rest, ":"); i >= 0 {
		to = rest[:i]
		label = strings.TrimSpace(rest[i+1:])
	}
	to = strings.TrimSpace(to)

	if from == "" || to == "" {
		b.ErrorHintf(line.Start(), "write a message as `Client->>Server: text`",
			"message is missing a participant")
		return nil, false
	}

	// Mermaid creates a participant on first mention, and so does this.
	t.EnsureNode(from, line.Start())
	t.EnsureNode(to, line.Start())

	t.AddEdge(&symbol.Edge{
		From: from, To: to, Label: label,
		Style: arrowStyle(op), Head: arrowHead(op),
		Bidir: strings.HasPrefix(op, "<<"),
		At:    line.Start(),
	})

	return &ast.MessageStmt{
		From: from, To: to, Operator: op, Label: label,
		Text: line.Text, StartPos: line.Start(),
	}, true
}

// findArrow locates the leftmost message operator, preferring the longest match
// at that position.
func findArrow(code string) (int, string) {
	for i := 0; i < len(code); i++ {
		if code[i] != '-' && code[i] != '<' {
			continue
		}
		for _, op := range seqArrows {
			if strings.HasPrefix(code[i:], op) {
				return i, op
			}
		}
	}
	return -1, ""
}

func arrowStyle(op string) string {
	if strings.Contains(op, "--") {
		return string(ast.LinkDotted)
	}
	return string(ast.LinkNormal)
}

// arrowHead reads the terminator off the operator.
//
// The doubled `>>` is what carries an arrowhead in a sequence diagram; a single
// `>`, as in `A->B`, is a plain line. Getting that backwards would draw every
// undecorated message as though it pointed somewhere.
func arrowHead(op string) string {
	switch {
	case strings.HasSuffix(op, ">>"):
		return string(ast.ArrowNormal)
	case strings.HasSuffix(op, "x"):
		return string(ast.ArrowCross)
	case strings.HasSuffix(op, ")"):
		return "open"
	default:
		return string(ast.ArrowNone)
	}
}
