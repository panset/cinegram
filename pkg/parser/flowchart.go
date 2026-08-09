package parser

import (
	"strings"

	"github.com/tejaspanse/cinegram/pkg/ast"
	"github.com/tejaspanse/cinegram/pkg/diag"
	"github.com/tejaspanse/cinegram/pkg/registry"
	"github.com/tejaspanse/cinegram/pkg/source"
	"github.com/tejaspanse/cinegram/pkg/symbol"
)

func init() { registry.Register(flowchartParser{}) }

// flowchartParser handles Mermaid `flowchart` and `graph` bodies.
type flowchartParser struct{}

func (flowchartParser) Keywords() []string { return []string{"flowchart", "graph"} }

var validDirections = map[string]bool{
	"LR": true, "RL": true, "TB": true, "TD": true, "BT": true,
}

// Parse reads the header line and then the body, stopping at the first
// top-level `scenario`.
func (p flowchartParser) Parse(c *source.Cursor, b *diag.Bag) (ast.Diagram, *symbol.Table) {
	table := symbol.NewTable()

	header, _ := c.Next()
	fc := &ast.Flowchart{
		HeaderText: header.Text,
		Direction:  "TB",
		StartPos:   header.Start(),
	}

	code, _ := stripComment(header.Text)
	if fields := strings.Fields(code); len(fields) > 1 {
		dir := strings.ToUpper(fields[1])
		if validDirections[dir] {
			fc.Direction = dir
		} else {
			b.ErrorHintf(header.PosAt(len(fields[0])+1),
				"valid directions are LR, RL, TB, TD and BT",
				"unknown flowchart direction %q", fields[1])
		}
	}

	fc.Statements = p.parseBody(c, b, table, "", false)
	return fc, table
}

// parseBody reads statements until it hits `end` (when inSubgraph), a
// top-level `scenario`, or end of input. group is the enclosing subgraph id.
func (p flowchartParser) parseBody(c *source.Cursor, b *diag.Bag, t *symbol.Table, group string, inSubgraph bool) []ast.Statement {
	var stmts []ast.Statement

	for {
		line, ok := c.Peek()
		if !ok {
			if inSubgraph {
				b.ErrorHintf(c.EOFPos(), "add an `end` line to close the subgraph",
					"unterminated subgraph %q", group)
			}
			return stmts
		}

		if line.Blank() {
			c.Next()
			continue
		}

		code, _ := stripComment(line.Text)
		code = strings.TrimSpace(code)
		word := firstWord(code)

		switch {
		case code == "":
			// Comment-only line.
			c.Next()
			stmts = append(stmts, &ast.RawStmt{Text: line.Text, StartPos: line.Start()})

		case word == "end" && code == "end":
			c.Next()
			if !inSubgraph {
				b.Errorf(line.Start(), "unexpected `end` outside a subgraph")
				continue
			}
			return stmts

		case isTopLevelKeyword(word) && !inSubgraph:
			// Hand control back: the rest of the file belongs to the
			// scenario half.
			return stmts

		case word == "subgraph":
			stmts = append(stmts, p.parseSubgraph(c, b, t, group))

		case word == "direction":
			c.Next()
			stmts = append(stmts, &ast.RawStmt{Text: line.Text, StartPos: line.Start()})

		case isRawKeyword(word):
			c.Next()
			stmts = append(stmts, &ast.RawStmt{Text: line.Text, StartPos: line.Start()})

		default:
			c.Next()
			stmts = append(stmts, p.parseGraphLine(line, code, b, t, group))
		}
	}
}

// rawKeywords are statements the parser preserves verbatim: they affect
// rendering but contribute no nodes or edges.
var rawKeywords = map[string]bool{
	"classDef":  true,
	"class":     true,
	"style":     true,
	"linkStyle": true,
	"click":     true,
	"accTitle":  true,
	"accDescr":  true,
}

func isRawKeyword(w string) bool { return rawKeywords[w] }

func (p flowchartParser) parseSubgraph(c *source.Cursor, b *diag.Bag, t *symbol.Table, parent string) ast.Statement {
	line, _ := c.Next()
	code, _ := stripComment(line.Text)
	rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(code), "subgraph"))

	id, label := parseSubgraphHeader(rest)
	if id == "" {
		b.ErrorHintf(line.Start(), "give the subgraph an id, e.g. `subgraph cluster[Kubernetes Cluster]`",
			"subgraph is missing a name")
		id = "subgraph"
	}
	t.DeclareGroup(id, label, parent, line.Start())
	if parent != "" {
		// Nested subgraphs are children of their parent group.
		if g, ok := t.Group(parent); ok {
			_ = g
			t.AddToGroup(parent, id)
		}
	}

	sg := &ast.SubgraphStmt{
		ID:       id,
		Label:    label,
		Text:     line.Text,
		StartPos: line.Start(),
	}
	sg.Body = p.parseBody(c, b, t, id, true)
	sg.EndText = "end"

	for _, st := range sg.Body {
		if d, ok := st.(*ast.RawStmt); ok && strings.HasPrefix(d.Text, "direction ") {
			sg.Direction = strings.TrimSpace(strings.TrimPrefix(d.Text, "direction"))
		}
	}
	return sg
}

// parseSubgraphHeader splits `cluster[Kubernetes Cluster]`, `cluster`, or a
// bare title into an id and a display label.
func parseSubgraphHeader(rest string) (id, label string) {
	if rest == "" {
		return "", ""
	}
	if ref, _, ok := parseNodeRef(rest); ok && ref.Declare {
		return ref.ID, ref.Label
	}
	// A bare title doubles as its own id, matching Mermaid.
	id = unquote(rest)
	return id, id
}

// parseGraphLine handles node declarations and edge chains.
func (p flowchartParser) parseGraphLine(line source.Line, code string, b *diag.Bag, t *symbol.Table, group string) ast.Statement {
	segs, offs, links := splitLinks(code)

	// Offsets from splitLinks are relative to code, which stripComment may
	// have trimmed from the left of Text only via TrimSpace above; recover the
	// shift so positions point at the right column.
	shift := strings.Index(line.Text, code)
	if shift < 0 {
		shift = 0
	}

	refs := make([]ast.NodeRef, 0, len(segs))
	for i, seg := range segs {
		ref, inner, ok := parseNodeRef(seg)
		if !ok {
			pos := line.PosAt(shift + offs[i])
			if len(links) > 0 {
				b.ErrorHintf(pos, "every link needs a node on both sides",
					"missing node beside link operator")
			} else {
				b.Errorf(pos, "could not parse node declaration")
			}
			return &ast.RawStmt{Text: line.Text, StartPos: line.Start()}
		}
		ref.At = line.PosAt(shift + offs[i] + inner)
		refs = append(refs, ref)
	}

	for _, ref := range refs {
		if ref.Declare {
			t.DeclareNode(ref.ID, ref.Label, string(ref.Shape), ref.Class, ref.At)
		} else {
			t.EnsureNode(ref.ID, ref.At)
		}
		if group != "" {
			t.AddToGroup(group, ref.ID)
		}
	}

	if len(links) == 0 {
		return &ast.NodeStmt{Nodes: refs, Text: line.Text, StartPos: line.Start()}
	}

	edges := make([]ast.EdgeRef, 0, len(links))
	for i, m := range links {
		from, to := refs[i], refs[i+1]
		at := line.PosAt(shift + m.start)
		t.AddEdge(&symbol.Edge{
			From:  from.ID,
			To:    to.ID,
			Label: m.link.Label,
			Style: string(m.link.Style),
			Head:  string(m.link.Head),
			Bidir: m.link.Bidir,
			At:    at,
		})
		edges = append(edges, ast.EdgeRef{From: from.ID, To: to.ID, Link: m.link, At: at})
	}

	return &ast.EdgeStmt{Nodes: refs, Edges: edges, Text: line.Text, StartPos: line.Start()}
}

func firstWord(s string) string {
	s = strings.TrimSpace(s)
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			return s[:i]
		}
	}
	return s
}
