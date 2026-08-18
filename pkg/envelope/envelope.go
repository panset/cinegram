// Package envelope holds the machine-readable wire format the CLI and any
// other host — the VS Code preview, a WASM build running in a browser tab —
// hand a timeline and its diagnostics across in.
//
// It lives outside cmd/cinegram because more than one main needs to produce
// exactly this shape, and outside pkg/diag because it is a wire format rather
// than the compiler's own representation: the fields are flat, the severity is
// a word rather than an enum, and it should not move when the internal type
// does.
package envelope

import (
	"github.com/tejaspanse/cinegram/pkg/diag"
	"github.com/tejaspanse/cinegram/pkg/ir"
)

// Envelope pairs a timeline with the diagnostics found producing it.
//
// It exists for hosts that render the result in place — a VS Code preview shows
// the message where the diagram would have been — and so has no failure mode of
// its own: the timeline is emitted even when errors were found, and the caller
// decides whether a partial diagram is worth drawing. Like Diagnostic, it
// is declared here because it is a wire format.
type Envelope struct {
	Timeline    *ir.Timeline `json:"timeline"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// Diagnostic is the machine-readable shape of a diagnostic. It is declared
// here rather than in pkg/diag because it is a wire format: the fields are
// flat, the severity is a word rather than an enum, and it should not move when
// the internal type does.
type Diagnostic struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Col      int    `json:"col"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`

	// Fix is present only when the diagnostic carries a machine-applicable
	// edit, and is appended last so a host reading the shape it already knew
	// sees nothing move. Its position is the text to replace, which need not be
	// the position of the diagnostic itself: a misspelt attribute key is
	// reported at the value and rewritten at the key.
	Fix *Fix `json:"fix,omitempty"`
}

// Fix is the machine-applicable edit a diagnostic may carry: at Line and Col,
// the source reads Old, and replacing it with New resolves the diagnostic.
//
// The position is flattened into two fields for the same reason the
// diagnostic's is — this is a wire format, and a host should not have to know
// the compiler's own Pos type to act on it. A caller applies the edit by
// checking that Old really is there and splicing New in its place; a fix whose
// Old no longer matches is one the file has moved on from.
type Fix struct {
	Line int    `json:"line"`
	Col  int    `json:"col"`
	Old  string `json:"old"`
	New  string `json:"new"`
}

// Collect flattens every bag into the wire shape, returning the
// error count alongside so callers that still key off exit status do not have
// to walk the result again. Always a non-nil slice: `[]` is a valid answer and
// `null` is not one a caller should have to handle.
func Collect(bags []*diag.Bag) ([]Diagnostic, int) {
	out := []Diagnostic{}
	errs := 0

	for _, bag := range bags {
		for _, d := range bag.All() {
			if d.Severity == diag.SeverityError {
				errs++
			}
			e := Diagnostic{
				File:     bag.Filename,
				Line:     d.Pos.Line,
				Col:      d.Pos.Col,
				Severity: d.Severity.String(),
				Message:  d.Msg,
				Hint:     d.Hint,
			}
			// Only a usable edit crosses the wire: a host must never have to
			// re-derive whether what it was handed can be applied.
			if d.Fix.Valid() {
				e.Fix = &Fix{Line: d.Fix.Pos.Line, Col: d.Fix.Pos.Col, Old: d.Fix.Old, New: d.Fix.New}
			}
			out = append(out, e)
		}
	}
	return out, errs
}
