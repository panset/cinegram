package parser

import (
	"strings"

	"github.com/tejaspanse/diagramator/pkg/ast"
	"github.com/tejaspanse/diagramator/pkg/diag"
	"github.com/tejaspanse/diagramator/pkg/source"
)

// Comment linting lives in its own file rather than in validate.go, which is
// the seam where the two halves of the language meet and stays free of diagram
// vocabulary. `%%` is Mermaid's comment marker across every diagram type, so
// this check is shared by all of them and specific to none.

// checkCommentLine reports a comment that Mermaid itself will mishandle.
//
// A line of exactly `%%`, with nothing after it, breaks Mermaid's comment
// stripping: the pass that removes comment lines requires at least one
// character after the marker, and the empty one survives to be glued onto the
// following line — which is usually the diagram header, so the whole diagram
// then fails to parse with an error pointing at line 1.
//
// Diagramator reprints comments verbatim on purpose, so it hands the trap
// straight through to the renderer. Catching it here is the only place it can
// be caught before the page comes up blank.
func checkCommentLine(text string, at source.Pos, b *diag.Bag) {
	if strings.TrimSpace(text) != "%%" {
		return
	}
	// The marker goes in as an argument rather than inline: these are printf
	// formats, and a literal `%%` in one silently prints as a single `%`.
	b.WarnHintf(at, "write `%% -` or delete the line",
		"an empty `%s` comment breaks Mermaid's comment stripping and the diagram will not render", "%%")
}

// checkComments walks the diagram body for the same problem. Comments inside
// the body reach the parser as ast.RawStmt, which is exactly the round-tripping
// that makes them dangerous.
func checkComments(d ast.Diagram, b *diag.Bag) {
	if d == nil {
		return
	}
	walkStatements(d.Body(), func(s ast.Statement) {
		if raw, ok := s.(*ast.RawStmt); ok {
			checkCommentLine(raw.Raw(), raw.Pos(), b)
		}
	})
}

// walkStatements visits every statement, descending into subgraphs.
func walkStatements(body []ast.Statement, visit func(ast.Statement)) {
	for _, s := range body {
		visit(s)
		if sub, ok := s.(*ast.SubgraphStmt); ok {
			walkStatements(sub.Body, visit)
		}
	}
}
