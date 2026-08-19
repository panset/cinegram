package compile

import (
	"encoding/json"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tejaspanse/cinegram/pkg/ir"
	"github.com/tejaspanse/cinegram/pkg/loader"
	"github.com/tejaspanse/cinegram/pkg/parser"
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

// TestPersistentStateLivesOutsideSteps pins the reason Persistent exists at
// all: a renderer skips steps whose window excludes t, so state set in the
// first step has to be reachable from the last.
func TestPersistentStateLivesOutsideSteps(t *testing.T) {
	const src = `flowchart LR
  a --> b

scenario "x"
  step one "set it" {
    set a { badge: "leader" }
    flow a -> b { dur: 400ms }
  }
  step two "later" {
    flow a -> b { dur: 600ms }
  }
`
	tl := compileSource(t, src)
	sc := tl.Views[0].Scenarios[0]

	for _, st := range sc.Steps {
		for _, tr := range st.Tracks {
			if tr.Kind == "set" || tr.Kind == "gauge" {
				t.Errorf("step %q carries a %s track; persistent state must not be a step track", st.ID, tr.Kind)
			}
		}
	}
	if len(sc.Persistent) != 1 {
		t.Fatalf("got %d persistent tracks, want 1", len(sc.Persistent))
	}
	// Unreplaced state runs to the end of the scenario, not the end of its step.
	if got := sc.Persistent[0]; got.Start != 0 || got.End != sc.Duration {
		t.Errorf("badge window = [%d,%d], want [0,%d]", got.Start, got.End, sc.Duration)
	}
}

// TestPersistentRewriteClosesPreviousWindow covers the bookkeeping that makes
// scrubbing show one value rather than every value ever written: windows for a
// slot must partition time, not overlap.
func TestPersistentRewriteClosesPreviousWindow(t *testing.T) {
	const src = `flowchart LR
  a --> b

scenario "x"
  step one "first" {
    gauge a { label: "term", value: 1 }
    flow a -> b { dur: 400ms }
  }
  step two "second" {
    gauge a { label: "term", value: 2 }
    gauge a { label: "votes", value: 3 }
    flow a -> b { dur: 600ms }
  }
`
	tl := compileSource(t, src)
	sc := tl.Views[0].Scenarios[0]

	if len(sc.Persistent) != 3 {
		t.Fatalf("got %d persistent tracks, want 3", len(sc.Persistent))
	}
	first, second, votes := sc.Persistent[0], sc.Persistent[1], sc.Persistent[2]

	if first.End != second.Start {
		t.Errorf("term windows overlap or gap: first ends %d, second starts %d", first.End, second.Start)
	}
	if first.Value != "1" || second.Value != "2" {
		t.Errorf("term values = %q then %q, want 1 then 2", first.Value, second.Value)
	}
	// A different label is a different slot, so it coexists rather than
	// replacing — two readings on one node is the normal case.
	if votes.Label != "votes" || votes.End != sc.Duration {
		t.Errorf("votes window = %+v, want an independent slot open to the end", votes)
	}
}

// TestClearingSemantics separates the two ways state is taken away: an empty
// badge retires the badge, `unset` retires everything on the node. Conflating
// them would make a badge change silently drop the node's gauges.
func TestClearingSemantics(t *testing.T) {
	const src = `flowchart LR
  a --> b

scenario "x"
  step one "set both" {
    set a { badge: "leader" }
    gauge a { label: "term", value: 1 }
    flow a -> b { dur: 400ms }
  }
  step two "drop the badge only" {
    set a { badge: "" }
    flow a -> b { dur: 400ms }
  }
  step three "drop everything" {
    unset a
    flow a -> b { dur: 400ms }
  }
`
	tl := compileSource(t, src)
	sc := tl.Views[0].Scenarios[0]
	steps := sc.Steps

	byKind := map[string]int{}
	for _, tr := range sc.Persistent {
		byKind[string(tr.Kind)] = tr.End
	}
	if got, want := byKind["set"], steps[1].Start; got != want {
		t.Errorf("badge closed at %d, want the empty-badge write at %d", got, want)
	}
	if got, want := byKind["gauge"], steps[2].Start; got != want {
		t.Errorf("gauge closed at %d, want the unset at %d — an empty badge must not retire it", got, want)
	}
}

// TestPersistentActionsAreInstantaneous checks they neither size a step nor
// consume a slot of a seq: they fire at a moment, and their effect is bounded
// by the next write rather than by the container they were written in.
func TestPersistentActionsAreInstantaneous(t *testing.T) {
	const src = `flowchart LR
  a --> b
  b --> c

scenario "x"
  step s "chained" {
    seq {
      flow a -> b { dur: 300ms }
      set b { badge: "here" }
      flow b -> c { dur: 400ms }
    }
  }
`
	tl := compileSource(t, src)
	sc := tl.Views[0].Scenarios[0]

	if sc.Duration != 700 {
		t.Errorf("scenario duration = %d, want 300+400 with the set costing nothing", sc.Duration)
	}
	if len(sc.Persistent) != 1 || sc.Persistent[0].Start != 300 {
		t.Fatalf("persistent = %+v, want one track firing at 300", sc.Persistent)
	}
}

func compileSource(t *testing.T, src string) *ir.Timeline {
	t.Helper()
	res, bag := parser.Parse("inline.dgm", src)
	if bag.HasErrors() {
		t.Fatalf("unexpected diagnostics:\n%s", bag)
	}
	tl := Compile(res.Document, res.Symbols, bag)
	if bag.HasErrors() {
		t.Fatalf("unexpected compile diagnostics:\n%s", bag)
	}
	return tl
}

// TestStoryboardCarriesInlinedImages joins the two halves the compiler owns
// here: the loader turns a path into a data URI, and this is where the frame
// picks it up. A frame whose image could not be read still compiles — with an
// empty Image — because the bag already carries the error and dropping it would
// leave a `scene` pointing at nothing.
func TestStoryboardCarriesInlinedImages(t *testing.T) {
	files := map[string]string{
		"top.dgm": `flowchart LR
  a[A]
  b[B]
  a --> b

storyboard "Screens" {
  frame shown { img: "pics/shown.svg", caption: "has a picture" }
  frame broken { img: "pics/gone.svg", caption: "does not" }
  frame words { caption: "never had one" }
}

scenario "x"
  step s "walk" {
    flow a -> b
    scene shown
  }
`,
		"pics/shown.svg": "<svg/>",
	}

	bundle, err := loader.Load("top.dgm", func(path string) ([]byte, error) {
		content, ok := files[filepath.Clean(path)]
		if !ok {
			return nil, fs.ErrNotExist
		}
		return []byte(content), nil
	})
	if err != nil {
		t.Fatal(err)
	}

	sb := CompileBundle(bundle).Views[0].Storyboard
	if sb == nil || len(sb.Frames) != 3 {
		t.Fatalf("storyboard = %+v, want all three declared frames", sb)
	}
	if sb.Frames[0].Image != "data:image/svg+xml;base64,PHN2Zy8+" {
		t.Errorf("frame %q image = %q, want the inlined data URI", sb.Frames[0].ID, sb.Frames[0].Image)
	}
	if sb.Frames[1].Image != "" || sb.Frames[1].Caption != "does not" {
		t.Errorf("unreadable frame = %+v, want it carried with an empty image", sb.Frames[1])
	}
	if sb.Frames[2].Image != "" {
		t.Errorf("caption-only frame = %+v, want no image", sb.Frames[2])
	}
	if !bundle.HasErrors() {
		t.Error("the unreadable image was not reported; compile stayed total but silent")
	}
}

// TestSceneSpansItsStep pins that `scene` needed no timing code of its own: it
// is a stateful action, so it inherits the rule that a highlight follows.
func TestSceneSpansItsStep(t *testing.T) {
	const src = `flowchart LR
  a --> b

storyboard {
  frame one { caption: "one" }
}

scenario "x"
  step s "spans" {
    flow a -> b { dur: 900ms }
    scene one
  }
`
	tl := compileSource(t, src)
	tracks := tl.Views[0].Scenarios[0].Steps[0].Tracks

	var scene *ir.Track
	for i := range tracks {
		if tracks[i].Kind == ir.TrackScene {
			scene = &tracks[i]
		}
	}
	if scene == nil {
		t.Fatalf("no scene track in %+v", tracks)
	}
	if scene.Target != "one" || scene.Start != 0 || scene.End != 900 {
		t.Errorf("scene = %+v, want frame \"one\" over the full step [0,900]", *scene)
	}
}

// TestVariantSplicesTheInheritedPrefix pins the shape of inheritance: `until`
// is inclusive, and the merged scenario is the base's prefix followed by the
// variant's own steps with the timing recomputed from scratch.
func TestVariantSplicesTheInheritedPrefix(t *testing.T) {
	const src = `flowchart LR
  a --> b

scenario "base"
  step one "first" {
    flow a -> b { dur: 400ms }
  }
  step two "second" {
    flow a -> b { dur: 500ms }
  }
  step three "not inherited" {
    flow a -> b { dur: 900ms }
  }

scenario "diverges" { variant: "base", until: two, outcome: fail }
  step wrong "instead" {
    flow a -> b { dur: 300ms }
  }
`
	tl := compileSource(t, src)
	sc := tl.Views[0].Scenarios[1]

	var ids []string
	for _, st := range sc.Steps {
		ids = append(ids, st.ID)
	}
	if strings.Join(ids, ",") != "one,two,wrong" {
		t.Errorf("steps = %v, want the base through `two` (inclusive) then the variant's own", ids)
	}
	if sc.Duration != 400+500+300 {
		t.Errorf("duration = %d, want the inherited prefix plus the divergent step", sc.Duration)
	}
	if sc.Outcome != "fail" {
		t.Errorf("outcome = %q, want fail", sc.Outcome)
	}
	// The base is untouched: splicing must not mutate what it inherited from.
	if got := len(tl.Views[0].Scenarios[0].Steps); got != 3 {
		t.Errorf("base has %d steps after the splice, want 3", got)
	}
}

// TestVariantResetsPerScenarioState covers the reason the splice happens at AST
// level rather than by copying compiled steps: hop occurrence counting and
// persistent-state windows are reset per scenario, so an inherited prefix has
// to replay from the start rather than continue the base's bookkeeping.
func TestVariantResetsPerScenarioState(t *testing.T) {
	const src = `flowchart LR
  a --> b
  a --> b

scenario "base"
  step one "first message" {
    flow a -> b { dur: 400ms }
    set a { badge: "trying" }
  }
  step two "second message" {
    flow a -> b { dur: 400ms }
  }

scenario "variant" { variant: "base", until: one }
  step other "its own ending" {
    flow a -> b { dur: 400ms }
  }
`
	tl := compileSource(t, src)
	base, variant := tl.Views[0].Scenarios[0], tl.Views[0].Scenarios[1]

	// Two parallel edges: each scenario consumes them from the start, so the
	// inherited first step animates e0 in both.
	if base.Steps[0].Tracks[0].Edge != variant.Steps[0].Tracks[0].Edge {
		t.Errorf("inherited step used edge %q, base used %q — the occurrence counter did not reset",
			variant.Steps[0].Tracks[0].Edge, base.Steps[0].Tracks[0].Edge)
	}
	if len(variant.Persistent) != 1 {
		t.Fatalf("variant persistent = %+v, want the inherited badge", variant.Persistent)
	}
	if got := variant.Persistent[0]; got.Start != 0 || got.End != variant.Duration {
		t.Errorf("inherited badge window = [%d,%d], want [0,%d] — it must close at the variant's own end",
			got.Start, got.End, variant.Duration)
	}
}

// TestUnresolvableVariantStillCompiles keeps compilation total: validation has
// already bagged the error, and refusing to lower the scenario would take the
// rest of the page down with it.
func TestUnresolvableVariantStillCompiles(t *testing.T) {
	const src = `flowchart LR
  a --> b

scenario "orphan" { variant: "nothing named this" }
  step only "on its own" {
    flow a -> b { dur: 400ms }
  }
`
	res, bag := parser.Parse("inline.dgm", src)
	if !bag.HasErrors() {
		t.Fatal("expected the unresolvable variant to be reported")
	}
	sc := Compile(res.Document, res.Symbols, bag).Views[0].Scenarios[0]
	if len(sc.Steps) != 1 || sc.Steps[0].ID != "only" || sc.Duration != 400 {
		t.Errorf("scenario = %+v, want its own steps alone", sc)
	}
}

// TestSceneInASeqCostsNoTime is what makes "the screen changes when the arrow
// lands" expressible without arithmetic: a scene inside a seq fires where the
// chain has reached and consumes none of it, so the hops around it keep their
// durations and adding a scene never inserts a silent pause.
func TestSceneInASeqCostsNoTime(t *testing.T) {
	const src = `flowchart LR
  a --> b
  b --> c

storyboard {
  frame one { caption: "one" }
  frame two { caption: "two" }
}

scenario "x"
  step s "chained" {
    seq {
      flow a -> b { dur: 300ms }
      scene one
      flow b -> c { dur: 400ms }
      scene two
    }
  }
`
	tl := compileSource(t, src)
	sc := tl.Views[0].Scenarios[0]

	if sc.Duration != 700 {
		t.Errorf("duration = %d, want 300+400 with the scenes costing nothing", sc.Duration)
	}

	scenes := map[string]int{}
	for _, tr := range sc.Steps[0].Tracks {
		if tr.Kind == ir.TrackScene {
			scenes[tr.Target] = tr.Start
		}
	}
	if scenes["one"] != 300 {
		t.Errorf("scene one starts at %d, want 300 — where the first hop lands", scenes["one"])
	}
	if scenes["two"] != 700 {
		t.Errorf("scene two starts at %d, want 700 — where the second hop lands", scenes["two"])
	}
}

// TestSceneTakesAnOffset covers the other way to place one: `at`/`delay` work on
// a scene exactly as on any other action, and a scene pushed past the actions
// around it stretches the step rather than escaping it.
func TestSceneTakesAnOffset(t *testing.T) {
	const src = `flowchart LR
  a --> b

storyboard {
  frame late { caption: "late" }
}

scenario "x"
  step s "offset" {
    flow a -> b { dur: 400ms }
    scene late { at: 900ms }
  }
`
	tl := compileSource(t, src)
	step := tl.Views[0].Scenarios[0].Steps[0]

	if step.End != 900 {
		t.Errorf("step span = %d, want 900 — the scene's offset sizes the step", step.End)
	}
	for _, tr := range step.Tracks {
		if tr.Kind == ir.TrackScene && tr.Start != 900 {
			t.Errorf("scene starts at %d, want its 900ms offset", tr.Start)
		}
	}
}

// TestEmptyListsAreNeverNull pins a property the renderer depends on and that
// nothing had exercised: a nil slice marshals to `null`, and the renderer walks
// these lists unconditionally.
//
// Every example shipped with the project happens to contain a subgraph, so a
// flowchart without one — which is most flowcharts anyone writes first — was
// the case that reached `view.groups.filter` with a null and took the whole
// page down. A step whose actions all draw nothing was the same bug waiting
// one line further on.
func TestEmptyListsAreNeverNull(t *testing.T) {
	const src = `flowchart LR
  a --> b

scenario "x"
  step nothing "a step that draws nothing" {
    wait 300ms
  }
  step something "and one that does" {
    flow a -> b { dur: 400ms }
  }
`
	encoded, err := json.Marshal(compileSource(t, src))
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]any
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatal(err)
	}
	view := raw["views"].([]any)[0].(map[string]any)

	for _, key := range []string{"nodes", "groups", "edges", "scenarios"} {
		if got, ok := view[key]; !ok || got == nil {
			t.Errorf("view.%s is %v; the renderer iterates it unconditionally", key, got)
		}
	}

	scenario := view["scenarios"].([]any)[0].(map[string]any)
	if scenario["steps"] == nil {
		t.Error("scenario.steps is null")
	}
	for i, s := range scenario["steps"].([]any) {
		if s.(map[string]any)["tracks"] == nil {
			t.Errorf("step %d has null tracks; a step may legitimately draw nothing", i)
		}
	}

	// The blunt version of the same check, so a field added later is covered
	// without anyone having to remember this test exists.
	if strings.Contains(string(encoded), ": null") || strings.Contains(string(encoded), ":null") {
		t.Errorf("timeline contains a null:\n%s", encoded)
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

// TestMicroStepsAreClampedToTheTransportFloor pins the timing rule that keeps
// presenter mode able to leave every step: advanceStep stops 1ms before a
// seam, so a step of 1ms or less would trap the press on itself forever. The
// clamp lives here in the compiler — timing rules do — and it announces
// itself, so an author who wrote dur: 1ms learns why it plays as 4.
func TestMicroStepsAreClampedToTheTransportFloor(t *testing.T) {
	const src = `flowchart LR
  a[A]
  b[B]
  a --> b

scenario "micro"
  step blink "blink" {
    dur: 1ms
    highlight a
  }
  step walk "walk" {
    flow a -> b { dur: 500ms }
  }
`
	res, bag := parser.Parse("inline.dgm", src)
	if bag.HasErrors() {
		t.Fatalf("unexpected diagnostics:\n%s", bag)
	}
	tl := Compile(res.Document, res.Symbols, bag)
	if bag.HasErrors() {
		t.Fatalf("clamped fixture should still compile:\n%s", bag)
	}

	steps := tl.Views[0].Scenarios[0].Steps
	if got := steps[0].End - steps[0].Start; got != minStepMillis {
		t.Errorf("1ms step spans %dms, want the %dms floor", got, minStepMillis)
	}
	// The stateful action inside still spans its whole (clamped) step.
	if tr := steps[0].Tracks[0]; tr.Start != steps[0].Start || tr.End != steps[0].End {
		t.Errorf("track [%d,%d] does not span the clamped step [%d,%d]",
			tr.Start, tr.End, steps[0].Start, steps[0].End)
	}
	// The next step starts where the clamped one ends: the clock stays whole.
	if steps[1].Start != steps[0].End {
		t.Errorf("step after the clamp starts at %d, want %d", steps[1].Start, steps[0].End)
	}

	// The clamp announces itself as a warning, never an error.
	var found bool
	for _, d := range bag.All() {
		if strings.Contains(d.Msg, "too short for the step transport") {
			found = true
		}
	}
	if !found {
		t.Errorf("no clamp warning surfaced; diagnostics:\n%s", bag)
	}

	// An ordinary step is untouched.
	if got := steps[1].End - steps[1].Start; got != 500 {
		t.Errorf("ordinary step spans %dms, want 500", got)
	}
}
