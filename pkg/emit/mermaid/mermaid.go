// Package mermaid renders a parsed document back to plain Mermaid source.
//
// The output is deliberately a reprint rather than a regeneration: every
// statement carries its original text, and this package only re-indents and
// drops the scenario blocks. That is what makes the round-trip lossless even
// for Mermaid syntax the parser does not model semantically — a classDef, a
// click handler, or a construct added to Mermaid after this code was written
// all survive untouched.
package mermaid

import (
	"strings"

	"github.com/tejaspanse/cinegram/pkg/ast"
)

const indentUnit = "  "

// Emit renders doc as standalone Mermaid source, without any scenario blocks.
func Emit(doc *ast.Document) string {
	var b strings.Builder

	if doc.Frontmatter != "" {
		b.WriteString(doc.Frontmatter)
		b.WriteByte('\n')
	}
	for _, line := range doc.Preamble {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	if doc.Diagram == nil {
		return b.String()
	}

	b.WriteString(doc.Diagram.Header())
	b.WriteByte('\n')
	writeStatements(&b, doc.Diagram.Body(), 1)

	return b.String()
}

func writeStatements(b *strings.Builder, stmts []ast.Statement, depth int) {
	for _, st := range stmts {
		if sg, ok := st.(*ast.SubgraphStmt); ok {
			writeLine(b, sg.Text, depth)
			writeStatements(b, sg.Body, depth+1)
			writeLine(b, "end", depth)
			continue
		}
		writeLine(b, st.Raw(), depth)
	}
}

func writeLine(b *strings.Builder, text string, depth int) {
	if text == "" {
		b.WriteByte('\n')
		return
	}
	b.WriteString(strings.Repeat(indentUnit, depth))
	b.WriteString(text)
	b.WriteByte('\n')
}
