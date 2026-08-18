package envelope

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tejaspanse/cinegram/pkg/diag"
	"github.com/tejaspanse/cinegram/pkg/source"
)

// TestCollectCarriesTheWholeDiagnostic pins what a host reads off the wire: the
// hint (most of what makes a message actionable), the position, the severity as
// a word, and the filename — which lives on the bag, not on the diagnostic, so
// flattening is the only place it can be lost.
func TestCollectCarriesTheWholeDiagnostic(t *testing.T) {
	bag := diag.NewBag("guide.md")
	bag.ErrorHintf(source.Pos{Line: 9, Col: 15}, "known nodes: a, b", "%q is not a node in this diagram", "nope")

	got, errs := Collect([]*diag.Bag{bag})
	if errs != 1 {
		t.Fatalf("error count = %d, want 1", errs)
	}
	if len(got) != 1 {
		t.Fatalf("got %d diagnostics, want 1", len(got))
	}

	d := got[0]
	if d.File != "guide.md" {
		t.Errorf("file = %q, want the bag's filename", d.File)
	}
	if d.Line != 9 || d.Col != 15 {
		t.Errorf("position = %d:%d, want 9:15", d.Line, d.Col)
	}
	if d.Severity != "error" {
		t.Errorf("severity = %q, want the word rather than the enum", d.Severity)
	}
	if d.Hint != "known nodes: a, b" {
		t.Errorf("hint = %q; it is most of what makes the message useful", d.Hint)
	}
	if !strings.Contains(d.Message, "nope") {
		t.Errorf("message = %q, want the formatted text", d.Message)
	}
}

// TestCollectCountsOnlyErrors keeps the exit-status contract off the payload:
// warnings are reported but do not raise the count a caller branches on, and
// every bag's own filename travels with its own diagnostics.
func TestCollectCountsOnlyErrors(t *testing.T) {
	warned := diag.NewBag("warn.dgm")
	warned.WarnHintf(source.Pos{Line: 3, Col: 5}, "a hint", "a warning")

	failed := diag.NewBag("bad.dgm")
	failed.Errorf(source.Pos{Line: 1, Col: 1}, "an error")
	failed.Errorf(source.Pos{Line: 2, Col: 1}, "another error")

	got, errs := Collect([]*diag.Bag{warned, failed})
	if errs != 2 {
		t.Fatalf("error count = %d, want 2 (warnings must not count)", errs)
	}
	if len(got) != 3 {
		t.Fatalf("got %d diagnostics, want 3", len(got))
	}
	if got[0].File != "warn.dgm" || got[0].Severity != "warning" {
		t.Errorf("first entry = %+v, want the warning from warn.dgm", got[0])
	}
	for _, d := range got[1:] {
		if d.File != "bad.dgm" {
			t.Errorf("file = %q, want bad.dgm", d.File)
		}
	}
}

// TestCollectEmptyIsAnArray pins that a clean run serialises `[]` and never
// `null`, so a host can iterate the field without checking it first.
func TestCollectEmptyIsAnArray(t *testing.T) {
	got, errs := Collect([]*diag.Bag{diag.NewBag("ok.dgm")})
	if errs != 0 {
		t.Fatalf("clean bag reported %d errors", errs)
	}
	if got == nil {
		t.Fatal("Collect returned a nil slice; it must always be non-nil")
	}

	encoded, err := json.Marshal(Envelope{Diagnostics: got})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(encoded), `"diagnostics":[]`) {
		t.Errorf("clean run should emit an empty array, got:\n%s", encoded)
	}
}

// TestCollectNoBags is the degenerate case a WASM host hits before anything has
// been loaded: still an array, still zero errors.
func TestCollectNoBags(t *testing.T) {
	got, errs := Collect(nil)
	if errs != 0 || len(got) != 0 {
		t.Fatalf("Collect(nil) = %v, %d; want empty, 0", got, errs)
	}
	if got == nil {
		t.Fatal("Collect(nil) returned a nil slice")
	}
}

// TestCollectCarriesTheFix pins the machine-applicable half of a hint across
// the wire, including the thing that is easy to get wrong: the fix's position
// is the text to replace, which is not always where the diagnostic is reported
// — a misspelt attribute key is reported at the value and rewritten at the key.
func TestCollectCarriesTheFix(t *testing.T) {
	bag := diag.NewBag("scene.dgm")
	bag.WarnFixf(
		source.Pos{Line: 27, Col: 34},
		diag.Fix{Pos: source.Pos{Line: 27, Col: 26}, Old: "colour", New: "color"},
		"did you mean color?", "unknown flow attribute %q", "colour")

	got, _ := Collect([]*diag.Bag{bag})
	if len(got) != 1 {
		t.Fatalf("got %d diagnostics, want 1", len(got))
	}

	d := got[0]
	if d.Fix == nil {
		t.Fatal("the fix did not survive Collect")
	}
	if d.Fix.Line != 27 || d.Fix.Col != 26 {
		t.Errorf("fix position = %d:%d, want 27:26 — the key, not the value", d.Fix.Line, d.Fix.Col)
	}
	if d.Fix.Old != "colour" || d.Fix.New != "color" {
		t.Errorf("fix = %q -> %q, want colour -> color", d.Fix.Old, d.Fix.New)
	}
	if d.Line != 27 || d.Col != 34 {
		t.Errorf("diagnostic position = %d:%d, want 27:34 — reporting is unaffected", d.Line, d.Col)
	}

	encoded, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(encoded), `"fix":{"line":27,"col":26,"old":"colour","new":"color"}`) {
		t.Errorf("wire shape is not the documented one:\n%s", encoded)
	}
}

// TestFixIsOmittedWhenInvalid keeps a host from having to ask whether what it
// was handed can be applied. An edit with nothing to anchor to, or with an
// empty side, is not a narrower fix — it is no fix, and must not appear.
func TestFixIsOmittedWhenInvalid(t *testing.T) {
	unfixable := []struct {
		name string
		fix  diag.Fix
	}{
		{"no fix at all", diag.Fix{}},
		{"no position", diag.Fix{Old: "ingres", New: "ing"}},
		{"nothing to replace", diag.Fix{Pos: source.Pos{Line: 1, Col: 1}, New: "ing"}},
		{"nothing to replace it with", diag.Fix{Pos: source.Pos{Line: 1, Col: 1}, Old: "ingres"}},
	}

	for _, tc := range unfixable {
		t.Run(tc.name, func(t *testing.T) {
			bag := diag.NewBag("plain.dgm")
			bag.ErrorFixf(source.Pos{Line: 1, Col: 1}, tc.fix, "a hint", "a message")

			got, _ := Collect([]*diag.Bag{bag})
			if got[0].Fix != nil {
				t.Fatalf("fix = %+v, want none", got[0].Fix)
			}
			encoded, err := json.Marshal(got[0])
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if strings.Contains(string(encoded), `"fix"`) {
				t.Errorf("fix should be absent from the wire shape, got:\n%s", encoded)
			}
		})
	}
}

// TestHintIsOmittedWhenEmpty keeps the wire shape as narrow as it was: a
// diagnostic with no hint must not carry an empty field for one.
func TestHintIsOmittedWhenEmpty(t *testing.T) {
	bag := diag.NewBag("plain.dgm")
	bag.Errorf(source.Pos{Line: 1, Col: 1}, "a bare message")

	got, _ := Collect([]*diag.Bag{bag})
	encoded, err := json.Marshal(got[0])
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(encoded), `"hint"`) {
		t.Errorf("hint should be omitted when empty, got:\n%s", encoded)
	}
}
