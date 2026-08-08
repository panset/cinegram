// Package parser turns Diagramator source into an ast.Document.
//
// A document is parsed in two independent halves. The diagram body is handed to
// whichever registry.DiagramParser claims its opening keyword; the `scenario`,
// `view` and `interact` blocks that follow are parsed by this package's shared,
// diagram-agnostic parser. Neither half knows about the other — they meet only
// in the validation pass, which resolves the names those blocks mention against
// the symbol table the diagram parser produced.
package parser

import (
	"strings"

	"github.com/tejaspanse/diagramator/pkg/ast"
	"github.com/tejaspanse/diagramator/pkg/diag"
	"github.com/tejaspanse/diagramator/pkg/registry"
	"github.com/tejaspanse/diagramator/pkg/source"
	"github.com/tejaspanse/diagramator/pkg/symbol"
)

// Result is everything a successful parse produces.
type Result struct {
	Document *ast.Document
	Symbols  *symbol.Table
	File     *source.File
}

// Parse parses content. Diagnostics are returned in the Bag rather than as an
// error so that callers can report several problems at once; check
// bag.HasErrors to decide whether Result is trustworthy.
func Parse(filename, content string) (*Result, *diag.Bag) {
	file := source.New(filename, content)
	bag := diag.NewBag(filename)
	cur := source.NewCursor(file)

	doc := &ast.Document{}
	doc.Frontmatter = parseFrontmatter(cur)

	doc.Preamble = skipToHeader(cur, bag)

	line, ok := cur.Peek()
	if !ok {
		bag.ErrorHintf(cur.EOFPos(),
			"start the file with a diagram header such as `flowchart LR`",
			"empty document")
		return &Result{Document: doc, Symbols: symbol.NewTable(), File: file}, bag
	}

	keyword := firstWord(line.Text)
	dp, found := registry.Lookup(keyword)
	if !found {
		bag.ErrorHintf(line.Start(),
			"supported diagram types: "+strings.Join(registry.Keywords(), ", "),
			"unknown diagram type %q", keyword)
		return &Result{Document: doc, Symbols: symbol.NewTable(), File: file}, bag
	}

	diagram, table := dp.Parse(cur, bag)
	doc.Diagram = diagram

	top := parseTopLevel(cur, bag)
	doc.Scenarios = top.Scenarios
	doc.Views = top.Views
	doc.Interactions = top.Interactions

	checkComments(doc.Diagram, bag)
	validateScenarios(doc.Scenarios, table, bag)
	validateInteract(doc, table, bag)
	validateCoverage(doc, table, bag)

	return &Result{Document: doc, Symbols: table, File: file}, bag
}

// skipToHeader advances past blank lines and the comments or `%%{init}%%`
// directives that commonly sit above a Mermaid header, returning those lines so
// emitters can put them back.
func skipToHeader(c *source.Cursor, b *diag.Bag) []string {
	var preamble []string
	for {
		line, ok := c.Peek()
		if !ok {
			return preamble
		}
		switch {
		case line.Blank():
			c.Next()
		case strings.HasPrefix(line.Text, "%%"):
			c.Next()
			checkCommentLine(line.Text, line.Start(), b)
			preamble = append(preamble, line.Text)
		default:
			return preamble
		}
	}
}

// parseFrontmatter consumes a leading `---` fenced block, returning it verbatim
// (fences included) so emitters can reproduce it untouched.
func parseFrontmatter(c *source.Cursor) string {
	line, ok := c.Peek()
	if !ok || strings.TrimSpace(line.Text) != "---" {
		return ""
	}
	mark := c.Mark()
	var out []string
	first, _ := c.Next()
	out = append(out, first.Raw)
	for {
		l, ok := c.Next()
		if !ok {
			// No closing fence: this was not frontmatter after all.
			c.Reset(mark)
			return ""
		}
		out = append(out, l.Raw)
		if strings.TrimSpace(l.Text) == "---" {
			return strings.Join(out, "\n")
		}
	}
}
