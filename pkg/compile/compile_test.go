package compile

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tejaspanse/diagramator/pkg/parser"
)

var update = flag.Bool("update", false, "rewrite golden files")

// TestGoldenTimelines compiles each fixture and compares the timeline JSON,
// which is the contract the renderer depends on.
func TestGoldenTimelines(t *testing.T) {
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

			tl := Compile(res.Document, res.Symbols, bag)
			if bag.HasErrors() {
				t.Fatalf("fixture should compile cleanly:\n%s", bag)
			}

			encoded, err := json.MarshalIndent(tl, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			got := string(encoded) + "\n"

			goldenPath := strings.TrimSuffix(src, ".dgm") + ".golden.json"
			if *update {
				root := os.Getenv("BUILD_WORKSPACE_DIRECTORY")
				if root == "" {
					t.Fatal("-update requires `bazel run`")
				}
				if err := os.WriteFile(filepath.Join(root, "pkg", "compile", goldenPath), []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("missing golden file (run with -update): %v", err)
			}
			if got != string(want) {
				t.Errorf("timeline differs from %s\n--- got ---\n%s", goldenPath, got)
			}
		})
	}
}

// TestTimingInvariants asserts the properties a renderer is entitled to assume,
// across every fixture rather than for one hand-checked case.
func TestTimingInvariants(t *testing.T) {
	sources, _ := filepath.Glob("testdata/*.dgm")
	for _, src := range sources {
		content, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		res, bag := parser.Parse(filepath.Base(src), string(content))
		tl := Compile(res.Document, res.Symbols, bag)

		for _, v := range tl.Views {
			for _, sc := range v.Scenarios {
				prevEnd := 0
				for _, st := range sc.Steps {
					if st.Start < prevEnd {
						t.Errorf("%s: step %q starts at %d, before the previous step ended at %d",
							src, st.ID, st.Start, prevEnd)
					}
					if st.End < st.Start {
						t.Errorf("%s: step %q ends before it starts", src, st.ID)
					}
					prevEnd = st.End

					for _, tr := range st.Tracks {
						if tr.End < tr.Start {
							t.Errorf("%s: track in step %q ends before it starts", src, st.ID)
						}
						if tr.Start < st.Start || tr.End > st.End {
							t.Errorf("%s: track [%d,%d] in step %q escapes the step span [%d,%d]",
								src, tr.Start, tr.End, st.ID, st.Start, st.End)
						}
					}
				}
				if len(sc.Steps) > 0 && sc.Duration != sc.Steps[len(sc.Steps)-1].End {
					t.Errorf("%s: scenario duration %d does not match final step end %d",
						src, sc.Duration, sc.Steps[len(sc.Steps)-1].End)
				}
			}
		}
	}
}

// TestFlowHopsPartitionExactly checks that a multi-hop flow's per-edge tracks
// tile its requested duration with no gap and no overlap, whatever the
// remainder of the division.
func TestFlowHopsPartitionExactly(t *testing.T) {
	const src = `flowchart LR
  a --> b
  b --> c
  c --> d

scenario "x"
  step s "uneven" {
    flow a -> b -> c -> d { dur: 1000ms }
  }
`
	res, bag := parser.Parse("inline.dgm", src)
	if bag.HasErrors() {
		t.Fatalf("unexpected diagnostics:\n%s", bag)
	}
	tl := Compile(res.Document, res.Symbols, bag)

	tracks := tl.Views[0].Scenarios[0].Steps[0].Tracks
	if len(tracks) != 3 {
		t.Fatalf("got %d tracks, want 3", len(tracks))
	}
	if tracks[0].Start != 0 {
		t.Errorf("first hop starts at %d, want 0", tracks[0].Start)
	}
	for i := 1; i < len(tracks); i++ {
		if tracks[i].Start != tracks[i-1].End {
			t.Errorf("hop %d starts at %d but hop %d ended at %d",
				i, tracks[i].Start, i-1, tracks[i-1].End)
		}
	}
	if got := tracks[len(tracks)-1].End; got != 1000 {
		t.Errorf("hops sum to %d, want the requested 1000", got)
	}
}

// TestReverseFlowUsesForwardEdge covers the response-path case: the flow runs
// against the direction the edge was drawn in, and must reuse that edge with
// Reverse set rather than being dropped.
func TestReverseFlowUsesForwardEdge(t *testing.T) {
	const src = `flowchart LR
  a --> b

scenario "x"
  step s "back" {
    flow b -> a
  }
`
	res, bag := parser.Parse("inline.dgm", src)
	if bag.HasErrors() {
		t.Fatalf("unexpected diagnostics:\n%s", bag)
	}
	tl := Compile(res.Document, res.Symbols, bag)

	tracks := tl.Views[0].Scenarios[0].Steps[0].Tracks
	if len(tracks) != 1 {
		t.Fatalf("got %d tracks, want 1", len(tracks))
	}
	if tracks[0].Edge != "e0" {
		t.Errorf("edge = %q, want the forward edge e0", tracks[0].Edge)
	}
	if !tracks[0].Reverse {
		t.Error("Reverse = false, want true when travelling against the drawn edge")
	}
}

// TestStatefulActionSpansStep pins the rule that a highlight without its own
// duration lasts as long as the step it sits in.
func TestStatefulActionSpansStep(t *testing.T) {
	const src = `flowchart LR
  a --> b

scenario "x"
  step s "spans" {
    flow a -> b { dur: 900ms }
    highlight b
    dim a { dur: 200ms }
  }
`
	res, bag := parser.Parse("inline.dgm", src)
	if bag.HasErrors() {
		t.Fatalf("unexpected diagnostics:\n%s", bag)
	}
	tl := Compile(res.Document, res.Symbols, bag)

	step := tl.Views[0].Scenarios[0].Steps[0]
	if step.End != 900 {
		t.Fatalf("step span = %d, want 900 from the flow", step.End)
	}

	byKind := map[string]struct{ start, end int }{}
	for _, tr := range step.Tracks {
		byKind[string(tr.Kind)] = struct{ start, end int }{tr.Start, tr.End}
	}
	if h := byKind["highlight"]; h.start != 0 || h.end != 900 {
		t.Errorf("highlight = [%d,%d], want the full step [0,900]", h.start, h.end)
	}
	if d := byKind["dim"]; d.start != 0 || d.end != 200 {
		t.Errorf("dim = [%d,%d], want its explicit [0,200]", d.start, d.end)
	}
}

// TestSeqRunsSequentially checks that seq children chain instead of starting
// together like ordinary step actions.
func TestSeqRunsSequentially(t *testing.T) {
	const src = `flowchart LR
  a --> b
  b --> c

scenario "x"
  step s "chained" {
    seq {
      flow a -> b { dur: 300ms }
      flow b -> c { dur: 400ms }
    }
  }
`
	res, bag := parser.Parse("inline.dgm", src)
	if bag.HasErrors() {
		t.Fatalf("unexpected diagnostics:\n%s", bag)
	}
	tl := Compile(res.Document, res.Symbols, bag)

	step := tl.Views[0].Scenarios[0].Steps[0]
	if step.End != 700 {
		t.Errorf("step span = %d, want 300+400", step.End)
	}
	if len(step.Tracks) != 2 {
		t.Fatalf("got %d tracks, want 2", len(step.Tracks))
	}
	if step.Tracks[0].Start != 0 || step.Tracks[0].End != 300 {
		t.Errorf("first seq child = [%d,%d], want [0,300]", step.Tracks[0].Start, step.Tracks[0].End)
	}
	if step.Tracks[1].Start != 300 || step.Tracks[1].End != 700 {
		t.Errorf("second seq child = [%d,%d], want [300,700]", step.Tracks[1].Start, step.Tracks[1].End)
	}
}

// TestRevealHidesGroupsTransitively pins the rule that revealing a subgraph
// conceals everything inside it. Hiding the frame while its members stayed
// drawn would look broken, and the renderer must not have to work that out.
func TestRevealHidesGroupsTransitively(t *testing.T) {
	const src = `flowchart LR
  a[A]

  subgraph outer[Outer]
    b[B]
    subgraph inner[Inner]
      c[C]
    end
  end

  a --> b
  b --> c

interact {
  click a -> reveal outer
}

scenario "x"
  step s "walk" {
    flow a -> b
  }
`
	res, bag := parser.Parse("inline.dgm", src)
	if bag.HasErrors() {
		t.Fatalf("unexpected diagnostics:\n%s", bag)
	}
	tl := Compile(res.Document, res.Symbols, bag)
	v := tl.Views[0]

	want := map[string]bool{"outer": true, "b": true, "inner": true, "c": true}
	if len(v.Hidden) != len(want) {
		t.Fatalf("hidden = %v, want %d elements", v.Hidden, len(want))
	}
	for _, id := range v.Hidden {
		if !want[id] {
			t.Errorf("hidden contains %q, which is not inside outer", id)
		}
	}

	// The binding's own targets are expanded the same way, so the renderer
	// toggles exactly the set it conceals.
	if len(v.Bindings) != 1 {
		t.Fatalf("got %d bindings, want 1", len(v.Bindings))
	}
	if len(v.Bindings[0].Targets) != len(v.Hidden) {
		t.Errorf("binding targets = %v, want the same set as hidden %v",
			v.Bindings[0].Targets, v.Hidden)
	}
}

// TestBindingsLowerTheirKinds checks each verb reaches the IR in the shape the
// runtime reads.
func TestBindingsLowerTheirKinds(t *testing.T) {
	const src = `flowchart LR
  a[A]
  b[B]
  c[C]
  a --> b
  b --> c

view sub "Sub" from "sub.dgm"

interact {
  click a -> view sub { label: "drill" }
  click b -> step only
  click c -> reveal b
}

scenario "x"
  step only "walk" {
    flow a -> b
  }
`
	res, bag := parser.Parse("inline.dgm", src)
	if bag.HasErrors() {
		t.Fatalf("unexpected diagnostics:\n%s", bag)
	}
	tl := Compile(res.Document, res.Symbols, bag)
	got := tl.Views[0].Bindings

	if len(got) != 3 {
		t.Fatalf("got %d bindings, want 3", len(got))
	}
	if got[0].Kind != "view" || got[0].View != "sub" || got[0].Label != "drill" {
		t.Errorf("view binding = %+v", got[0])
	}
	if got[1].Kind != "step" || got[1].Step != "only" {
		t.Errorf("step binding = %+v", got[1])
	}
	if got[2].Kind != "reveal" || len(got[2].Targets) != 1 || got[2].Targets[0] != "b" {
		t.Errorf("reveal binding = %+v", got[2])
	}
}
