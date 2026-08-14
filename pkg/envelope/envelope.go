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
			out = append(out, Diagnostic{
				File:     bag.Filename,
				Line:     d.Pos.Line,
				Col:      d.Pos.Col,
				Severity: d.Severity.String(),
				Message:  d.Msg,
				Hint:     d.Hint,
			})
		}
	}
	return out, errs
}
