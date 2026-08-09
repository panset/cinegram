package parser

import (
	"strings"

	"github.com/tejaspanse/cinegram/pkg/diag"
	"github.com/tejaspanse/cinegram/pkg/source"
)

// The scenario half of the language is brace-structured rather than
// line-structured, so it gets a real tokenizer. Newlines are emitted as tokens
// because an action without an attribute block is terminated by end of line.

type tokKind int

const (
	tokEOF tokKind = iota
	tokNewline
	tokIdent
	tokString
	tokValue // a number with an optional unit suffix: 600ms, 1.2s, 3
	tokPunct // { } , : -> ( )
)

func (k tokKind) String() string {
	switch k {
	case tokEOF:
		return "end of input"
	case tokNewline:
		return "end of line"
	case tokIdent:
		return "name"
	case tokString:
		return "string"
	case tokValue:
		return "value"
	default:
		return "symbol"
	}
}

type tok struct {
	kind tokKind
	text string // for tokString, the unquoted content
	at   source.Pos
}

// scanner tokenizes the scenario region of a file, beginning at the cursor's
// current line and running to end of input.
type scanner struct {
	toks []tok
	i    int
	bag  *diag.Bag
}

func newScanner(c *source.Cursor, b *diag.Bag) *scanner {
	s := &scanner{bag: b}
	for {
		line, ok := c.Next()
		if !ok {
			break
		}
		s.scanLine(line)
	}
	s.toks = append(s.toks, tok{kind: tokEOF, at: c.EOFPos()})
	return s
}

func (s *scanner) scanLine(line source.Line) {
	txt := line.Raw
	i := line.Indent
	emitted := false

	for i < len(txt) {
		c := txt[i]
		switch {
		case c == ' ' || c == '\t':
			i++

		case c == '%' && i+1 < len(txt) && txt[i+1] == '%':
			i = len(txt) // comment runs to end of line

		case c == '"':
			str, next, ok := scanString(txt, i)
			if !ok {
				s.bag.ErrorHintf(line.PosAt(i-line.Indent), `add the closing quote`,
					"unterminated string")
				i = len(txt)
				break
			}
			s.emit(tok{kind: tokString, text: str, at: line.PosAt(i - line.Indent)})
			emitted = true
			i = next

		case c == '-' && i+1 < len(txt) && txt[i+1] == '>':
			s.emit(tok{kind: tokPunct, text: "->", at: line.PosAt(i - line.Indent)})
			emitted = true
			i += 2

		case strings.IndexByte("{},:()", c) >= 0:
			s.emit(tok{kind: tokPunct, text: string(c), at: line.PosAt(i - line.Indent)})
			emitted = true
			i++

		case c >= '0' && c <= '9':
			j := i
			for j < len(txt) && (txt[j] >= '0' && txt[j] <= '9' || txt[j] == '.') {
				j++
			}
			for j < len(txt) && isLetter(txt[j]) {
				j++ // unit suffix: ms, s
			}
			s.emit(tok{kind: tokValue, text: txt[i:j], at: line.PosAt(i - line.Indent)})
			emitted = true
			i = j

		case isIdentStart(c):
			j := i
			for j < len(txt) {
				ch := txt[j]
				if isIdentPart(ch) {
					j++
					continue
				}
				// A hyphen belongs to the identifier unless it begins the
				// `->` flow operator, since Mermaid ids commonly contain one.
				if ch == '-' && j+1 < len(txt) && txt[j+1] != '>' {
					j++
					continue
				}
				break
			}
			s.emit(tok{kind: tokIdent, text: txt[i:j], at: line.PosAt(i - line.Indent)})
			emitted = true
			i = j

		default:
			s.bag.Errorf(line.PosAt(i-line.Indent), "unexpected character %q", string(c))
			i++
		}
	}

	if emitted {
		s.emit(tok{kind: tokNewline, at: line.PosAt(len(txt) - line.Indent)})
	}
}

func (s *scanner) emit(t tok) { s.toks = append(s.toks, t) }

func scanString(s string, i int) (text string, next int, ok bool) {
	var sb strings.Builder
	i++ // opening quote
	for i < len(s) {
		switch s[i] {
		case '\\':
			if i+1 >= len(s) {
				return "", i, false
			}
			switch s[i+1] {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			default:
				sb.WriteByte(s[i+1])
			}
			i += 2
		case '"':
			return sb.String(), i + 1, true
		default:
			sb.WriteByte(s[i])
			i++
		}
	}
	return "", i, false
}

func isLetter(c byte) bool     { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
func isIdentStart(c byte) bool { return isLetter(c) || c == '_' }
func isIdentPart(c byte) bool {
	return isLetter(c) || c == '_' || c >= '0' && c <= '9'
}

// --- cursor helpers ---------------------------------------------------------

func (s *scanner) peek() tok { return s.toks[s.i] }

func (s *scanner) next() tok {
	t := s.toks[s.i]
	if t.kind != tokEOF {
		s.i++
	}
	return t
}

// at reports whether the current token is the given punctuation.
func (s *scanner) at(punct string) bool {
	t := s.peek()
	return t.kind == tokPunct && t.text == punct
}

// atKeyword reports whether the current token is the given identifier.
func (s *scanner) atKeyword(kw string) bool {
	t := s.peek()
	return t.kind == tokIdent && t.text == kw
}

// accept consumes the given punctuation if present.
func (s *scanner) accept(punct string) bool {
	if s.at(punct) {
		s.next()
		return true
	}
	return false
}

// expect consumes the given punctuation, reporting an error if it is absent.
func (s *scanner) expect(punct string) bool {
	if s.accept(punct) {
		return true
	}
	t := s.peek()
	s.bag.Errorf(t.at, "expected %q but found %s", punct, describe(t))
	return false
}

// skipNewlines advances past any run of blank lines.
func (s *scanner) skipNewlines() {
	for s.peek().kind == tokNewline {
		s.next()
	}
}

// skipToLineEnd recovers from a malformed statement by discarding the rest of
// the line, so one bad action does not cascade into the following ones.
func (s *scanner) skipToLineEnd() {
	for {
		t := s.peek()
		if t.kind == tokEOF || t.kind == tokNewline {
			return
		}
		if t.kind == tokPunct && t.text == "}" {
			return
		}
		s.next()
	}
}

func describe(t tok) string {
	switch t.kind {
	case tokEOF:
		return "end of input"
	case tokNewline:
		return "end of line"
	case tokString:
		return `string "` + t.text + `"`
	default:
		if t.text == "" {
			return t.kind.String()
		}
		return `"` + t.text + `"`
	}
}
