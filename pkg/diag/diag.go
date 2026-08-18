// Package diag collects parser and compiler diagnostics with source positions.
//
// Diagnostics carry an optional Hint because the primary consumers of this
// language are humans writing DSL by hand and AI agents generating it; both
// benefit far more from "did you mean" than from a bare syntax error.
package diag

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tejaspanse/cinegram/pkg/source"
)

// Severity classifies a diagnostic.
type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
)

func (s Severity) String() string {
	if s == SeverityWarning {
		return "warning"
	}
	return "error"
}

// Fix is the machine-applicable half of a hint: replace Old with New at Pos.
//
// It is a plain value struct with no slice, map or pointer field, because
// Bag.add de-duplicates diagnostics with `==` and a diagnostic carrying a
// reference type could not be compared at all. One edit is enough: every site
// that offers a fix is a "did you mean" over a single token.
//
// Pos is where the replaced text starts, which is not always where the
// diagnostic itself is reported — a misspelt attribute key is reported at the
// value, because that is where the reader is looking, while the text to swap is
// the key.
type Fix struct {
	Pos source.Pos `json:"pos"`
	Old string     `json:"old"`
	New string     `json:"new"`
}

// Valid reports whether f names a real edit. An applier must ignore anything
// else: a fix with no position has nothing to anchor to, and one with an empty
// side would either delete or insert rather than replace.
func (f Fix) Valid() bool { return f.Pos.IsValid() && f.Old != "" && f.New != "" }

// Diagnostic is a single problem found in a source file.
type Diagnostic struct {
	Pos      source.Pos `json:"pos"`
	Severity Severity   `json:"-"`
	Msg      string     `json:"msg"`
	Hint     string     `json:"hint,omitempty"`
	Fix      Fix        `json:"fix,omitzero"`
}

// Bag accumulates diagnostics during a parse or compile pass.
type Bag struct {
	Filename string
	items    []Diagnostic
}

// NewBag returns a Bag that labels its diagnostics with filename.
func NewBag(filename string) *Bag { return &Bag{Filename: filename} }

// Errorf records an error at pos.
func (b *Bag) Errorf(pos source.Pos, format string, args ...any) {
	b.add(Diagnostic{Pos: pos, Severity: SeverityError, Msg: fmt.Sprintf(format, args...)})
}

// ErrorHintf records an error at pos along with a suggested fix.
func (b *Bag) ErrorHintf(pos source.Pos, hint, format string, args ...any) {
	b.add(Diagnostic{Pos: pos, Severity: SeverityError, Msg: fmt.Sprintf(format, args...), Hint: hint})
}

// ErrorFixf records an error at pos carrying both a hint and the same
// suggestion as an edit a tool can apply.
//
// The hint is not redundant: it is what a human reads, and it survives when the
// fix does not. An invalid fix is stored as the zero one rather than rejected,
// so a caller can pass whatever its suggester produced without first asking
// whether there was anything to suggest.
func (b *Bag) ErrorFixf(pos source.Pos, fix Fix, hint, format string, args ...any) {
	b.add(Diagnostic{Pos: pos, Severity: SeverityError, Msg: fmt.Sprintf(format, args...), Hint: hint, Fix: validOrZero(fix)})
}

// Warnf records a warning at pos.
func (b *Bag) Warnf(pos source.Pos, format string, args ...any) {
	b.add(Diagnostic{Pos: pos, Severity: SeverityWarning, Msg: fmt.Sprintf(format, args...)})
}

// WarnHintf records a warning at pos along with a suggested fix.
func (b *Bag) WarnHintf(pos source.Pos, hint, format string, args ...any) {
	b.add(Diagnostic{Pos: pos, Severity: SeverityWarning, Msg: fmt.Sprintf(format, args...), Hint: hint})
}

// WarnFixf records a warning at pos carrying both a hint and the same
// suggestion as an edit a tool can apply.
func (b *Bag) WarnFixf(pos source.Pos, fix Fix, hint, format string, args ...any) {
	b.add(Diagnostic{Pos: pos, Severity: SeverityWarning, Msg: fmt.Sprintf(format, args...), Hint: hint, Fix: validOrZero(fix)})
}

// validOrZero drops an unusable fix, so that "has a fix" and "has a valid fix"
// are the same question everywhere downstream.
func validOrZero(f Fix) Fix {
	if f.Valid() {
		return f
	}
	return Fix{}
}

// add records d, ignoring an exact repeat.
//
// Several passes legitimately inspect the same attribute — validation checks
// its type, then the timing pass reads it once to size a step and again to lay
// out the action. A bad value would otherwise be reported once per read, which
// tells the author nothing extra and buries the other problems.
func (b *Bag) add(d Diagnostic) {
	for _, existing := range b.items {
		if existing == d {
			return
		}
	}
	b.items = append(b.items, d)
}

// All returns the accumulated diagnostics sorted by position.
func (b *Bag) All() []Diagnostic {
	out := make([]Diagnostic, len(b.items))
	copy(out, b.items)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Pos.Line != out[j].Pos.Line {
			return out[i].Pos.Line < out[j].Pos.Line
		}
		return out[i].Pos.Col < out[j].Pos.Col
	})
	return out
}

// HasErrors reports whether any diagnostic has error severity.
func (b *Bag) HasErrors() bool {
	for _, d := range b.items {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Len returns the number of accumulated diagnostics.
func (b *Bag) Len() int { return len(b.items) }

// Err returns an error describing the collected errors, or nil if there are none.
func (b *Bag) Err() error {
	if !b.HasErrors() {
		return nil
	}
	return fmt.Errorf("%s", b.String())
}

// String renders every diagnostic in "file:line:col: severity: msg" form.
func (b *Bag) String() string {
	var sb strings.Builder
	for i, d := range b.All() {
		if i > 0 {
			sb.WriteByte('\n')
		}
		name := b.Filename
		if name == "" {
			name = "<input>"
		}
		fmt.Fprintf(&sb, "%s:%s: %s: %s", name, d.Pos, d.Severity, d.Msg)
		if d.Hint != "" {
			fmt.Fprintf(&sb, "\n  hint: %s", d.Hint)
		}
	}
	return sb.String()
}
