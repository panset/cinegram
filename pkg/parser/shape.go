package parser

import (
	"strings"

	"github.com/tejaspanse/cinegram/pkg/ast"
)

// shapeDef pairs a Mermaid node shape with its delimiters.
type shapeDef struct {
	open  string
	close string
	shape ast.Shape
}

// shapeDefs is ordered most-specific first. Several shapes share an opening
// delimiter (`[/` opens both a parallelogram and a trapezoid), so matching
// requires both ends; listing longer delimiters first keeps `[[` from being
// read as `[`.
var shapeDefs = []shapeDef{
	{"(((", ")))", ast.ShapeDoubleCircle},
	{"[[", "]]", ast.ShapeSubroutine},
	{"[(", ")]", ast.ShapeCylinder},
	{"[/", `\]`, ast.ShapeTrapezoid},
	{"[/", "/]", ast.ShapeParallelogram},
	{`[\`, "/]", ast.ShapeTrapezoidAlt},
	{`[\`, `\]`, ast.ShapeParallelogramAlt},
	{"((", "))", ast.ShapeCircle},
	{"([", "])", ast.ShapeStadium},
	{"{{", "}}", ast.ShapeHexagon},
	{"[", "]", ast.ShapeRect},
	{"(", ")", ast.ShapeRound},
	{"{", "}", ast.ShapeRhombus},
	{">", "]", ast.ShapeAsymmetric},
}

// parseNodeRef reads a single node segment such as `client[External Client]`,
// `db[(Postgres)]`, `pod1:::busy`, or a bare `svc`.
//
// ok is false only for an empty segment; anything else is treated as a bare id
// so that unfamiliar syntax degrades to a referenceable node rather than
// failing the whole line.
func parseNodeRef(seg string) (ref ast.NodeRef, offset int, ok bool) {
	trimmed := strings.TrimLeft(seg, " \t")
	offset = len(seg) - len(trimmed)
	trimmed = strings.TrimRight(trimmed, " \t")
	if trimmed == "" {
		return ast.NodeRef{}, offset, false
	}

	body, class := splitClassSuffix(trimmed)
	body = strings.TrimSpace(body)
	if body == "" {
		return ast.NodeRef{}, offset, false
	}

	openIdx := strings.IndexAny(body, "[({>")
	if openIdx < 0 {
		return ast.NodeRef{ID: body, Label: body, Shape: ast.ShapeNone, Class: class}, offset, true
	}

	id := strings.TrimSpace(body[:openIdx])
	rest := body[openIdx:]
	for _, d := range shapeDefs {
		if len(rest) < len(d.open)+len(d.close) {
			continue
		}
		if !strings.HasPrefix(rest, d.open) || !strings.HasSuffix(rest, d.close) {
			continue
		}
		label := rest[len(d.open) : len(rest)-len(d.close)]
		if id == "" {
			// Shapes with no id, e.g. a stray `[text]`, are not valid
			// Mermaid; fall through to treating the whole thing as an id.
			break
		}
		return ast.NodeRef{
			ID:      id,
			Label:   unquote(strings.TrimSpace(label)),
			Shape:   d.shape,
			Class:   class,
			Declare: true,
		}, offset, true
	}

	return ast.NodeRef{ID: body, Label: body, Shape: ast.ShapeNone, Class: class}, offset, true
}

// splitClassSuffix peels off a `:::className` suffix that sits outside any
// brackets, so that a colon inside a label (`ns[namespace: prod]`) is left
// alone.
func splitClassSuffix(s string) (body, class string) {
	depth := 0
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuote {
			if c == '"' {
				inQuote = false
			}
			continue
		}
		switch c {
		case '"':
			inQuote = true
		case '[', '(', '{':
			depth++
		case ']', ')', '}':
			if depth > 0 {
				depth--
			}
		case ':':
			if depth == 0 && strings.HasPrefix(s[i:], ":::") {
				return s[:i], strings.TrimSpace(s[i+3:])
			}
		}
	}
	return s, ""
}

// unquote strips one layer of surrounding double quotes.
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// stripComment removes a trailing `%%` comment that sits outside brackets and
// quotes, returning the code portion and the comment (including its `%%`).
func stripComment(s string) (code, comment string) {
	depth := 0
	inQuote := false
	for i := 0; i+1 < len(s); i++ {
		c := s[i]
		if inQuote {
			if c == '"' {
				inQuote = false
			}
			continue
		}
		switch c {
		case '"':
			inQuote = true
		case '[', '(', '{':
			depth++
		case ']', ')', '}':
			if depth > 0 {
				depth--
			}
		case '%':
			if depth == 0 && s[i+1] == '%' {
				return strings.TrimRight(s[:i], " \t"), s[i:]
			}
		}
	}
	return s, ""
}
