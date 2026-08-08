// Package source holds diagram source text and maps between byte offsets and
// human-readable line/column positions.
//
// Diagramator's diagram bodies are line-oriented, exactly like Mermaid's, so
// the unit of iteration here is a Line rather than a token. Diagram parsers
// walk a Cursor; the scenario lexer builds tokens on top of the same Lines.
package source

import (
	"fmt"
	"strings"
)

// Pos is a 1-based line/column position within a File.
type Pos struct {
	Line int
	Col  int
}

func (p Pos) String() string { return fmt.Sprintf("%d:%d", p.Line, p.Col) }

// IsValid reports whether p refers to a real location.
func (p Pos) IsValid() bool { return p.Line > 0 }

// Line is one physical line of source.
//
// Raw is preserved verbatim so emitters can round-trip constructs the parser
// does not model semantically. Text is the trimmed form parsers actually match
// against, and Indent records how far Text was shifted so that offsets within
// Text can be mapped back to real columns.
type Line struct {
	Num    int    // 1-based line number
	Raw    string // original text, newline stripped
	Text   string // Raw with surrounding whitespace removed
	Indent int    // count of leading whitespace characters in Raw
}

// Blank reports whether the line has no content.
func (l Line) Blank() bool { return l.Text == "" }

// Start is the position of the first non-whitespace character.
func (l Line) Start() Pos { return Pos{Line: l.Num, Col: l.Indent + 1} }

// PosAt maps a byte offset within Text to a position in the original line.
func (l Line) PosAt(offset int) Pos {
	return Pos{Line: l.Num, Col: l.Indent + offset + 1}
}

// File is a parsed-into-lines source document.
type File struct {
	Name  string
	Lines []Line
}

// New splits content into Lines. It accepts LF and CRLF endings.
func New(name, content string) *File {
	f := &File{Name: name}
	content = strings.ReplaceAll(content, "\r\n", "\n")
	for i, raw := range strings.Split(content, "\n") {
		trimmed := strings.TrimLeft(raw, " \t")
		f.Lines = append(f.Lines, Line{
			Num:    i + 1,
			Raw:    raw,
			Text:   strings.TrimRight(trimmed, " \t"),
			Indent: len(raw) - len(trimmed),
		})
	}
	return f
}

// Cursor is a forward-only reader over a File's lines.
type Cursor struct {
	file *File
	i    int
}

// NewCursor returns a Cursor positioned at the first line of f.
func NewCursor(f *File) *Cursor { return &Cursor{file: f} }

// File returns the underlying file.
func (c *Cursor) File() *File { return c.file }

// Done reports whether the cursor is past the last line.
func (c *Cursor) Done() bool { return c.i >= len(c.file.Lines) }

// Peek returns the current line without consuming it.
func (c *Cursor) Peek() (Line, bool) {
	if c.Done() {
		return Line{}, false
	}
	return c.file.Lines[c.i], true
}

// PeekAhead returns the line n positions ahead of the cursor.
func (c *Cursor) PeekAhead(n int) (Line, bool) {
	if c.i+n >= len(c.file.Lines) || c.i+n < 0 {
		return Line{}, false
	}
	return c.file.Lines[c.i+n], true
}

// Next consumes and returns the current line.
func (c *Cursor) Next() (Line, bool) {
	l, ok := c.Peek()
	if ok {
		c.i++
	}
	return l, ok
}

// SkipBlank advances past blank lines.
func (c *Cursor) SkipBlank() {
	for {
		l, ok := c.Peek()
		if !ok || !l.Blank() {
			return
		}
		c.i++
	}
}

// Mark returns an opaque cursor offset for later use with Reset.
func (c *Cursor) Mark() int { return c.i }

// Reset moves the cursor back to a previously marked offset.
func (c *Cursor) Reset(mark int) { c.i = mark }

// EOFPos is the position just past the end of the file.
func (c *Cursor) EOFPos() Pos {
	if len(c.file.Lines) == 0 {
		return Pos{Line: 1, Col: 1}
	}
	last := c.file.Lines[len(c.file.Lines)-1]
	return Pos{Line: last.Num, Col: len(last.Raw) + 1}
}
