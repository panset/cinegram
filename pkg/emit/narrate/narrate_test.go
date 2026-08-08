package narrate

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tejaspanse/diagramator/pkg/compile"
	"github.com/tejaspanse/diagramator/pkg/parser"
)

var update = flag.Bool("update", false, "rewrite golden files")

// TestGoldenNarration pins the prose, because the prose is the product here:
// this output is what an agent reads instead of watching the animation, and a
// change to a sentence is a change to the contract.
func TestGoldenNarration(t *testing.T) {
	sources, err := filepath.Glob("testdata/*.dgm")
	if err != nil || len(sources) == 0 {
		t.Fatalf("no testdata found: %v", err)
	}

	for _, src := range sources {
		name := strings.TrimSuffix(filepath.Base(src), ".dgm")
		t.Run(name, func(t *testing.T) {
			content, err := os.ReadFile(src)
			if err != nil {
				t.Fatal(err)
			}
			res, bag := parser.Parse(filepath.Base(src), string(content))
			if bag.HasErrors() {
				t.Fatalf("fixture should parse cleanly:\n%s", bag)
			}
			tl := compile.Compile(res.Document, res.Symbols, bag)

			got := string(Markdown(Build(tl, name)))
			goldenPath := strings.TrimSuffix(src, ".dgm") + ".golden.md"

			if *update {
				root := os.Getenv("BUILD_WORKSPACE_DIRECTORY")
				if root == "" {
					t.Fatal("-update requires `bazel run`")
				}
				full := filepath.Join(root, "pkg", "emit", "narrate", goldenPath)
				if err := os.WriteFile(full, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("missing golden file (run with -update): %v", err)
			}
			if got != string(want) {
				t.Errorf("narration differs from %s\n--- got ---\n%s", goldenPath, got)
			}
		})
	}
}

// TestEventsAreOrderedByTime covers the one reordering the walkthrough does.
// Tracks reach the IR in source order, which `delay` and `seq` can make
// different from time order — and a walkthrough that lists a later action first
// misleads a reader who cannot see the timestamps.
func TestEventsAreOrderedByTime(t *testing.T) {
	const src = `flowchart LR
  a --> b
  b --> c

scenario "x"
  step s "out of order in the source" {
    highlight c { delay: 600ms }
    flow a -> b { dur: 400ms }
  }
`
	res, bag := parser.Parse("inline.dgm", src)
	if bag.HasErrors() {
		t.Fatalf("unexpected diagnostics:\n%s", bag)
	}
	doc := Build(compile.Compile(res.Document, res.Symbols, bag), "x")

	events := doc.Views[0].Scenarios[0].Steps[0].Events
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].Start > events[1].Start {
		t.Errorf("events are not in time order: %d then %d", events[0].Start, events[1].Start)
	}
	if events[0].Kind != "flow" {
		t.Errorf("first event is %q, want the flow that starts at 0", events[0].Kind)
	}
}

// TestJSONCarriesFieldsNotJustProse checks the structured half survives. An
// agent filtering for "every failing flow" must not have to parse the sentence
// back apart to do it.
func TestJSONCarriesFieldsNotJustProse(t *testing.T) {
	const src = `flowchart LR
  a --> b

scenario "x"
  step s "it fails" {
    flow a -> b { label: "try", status: fail }
  }
`
	res, bag := parser.Parse("inline.dgm", src)
	doc := Build(compile.Compile(res.Document, res.Symbols, bag), "x")

	e := doc.Views[0].Scenarios[0].Steps[0].Events[0]
	if e.Status != "fail" || e.From != "a" || e.To != "b" || e.Label != "try" {
		t.Errorf("event lost its fields: %+v", e)
	}
	if !strings.Contains(e.Text, "fails") {
		t.Errorf("sentence %q does not mention the failure", e.Text)
	}
}
