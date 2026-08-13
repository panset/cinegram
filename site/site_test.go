package site

import (
	"fmt"
	"strings"
	"testing"
)

// mem builds a ReadFileFunc over an in-memory tree, the same way the loader
// tests avoid a filesystem.
func mem(files map[string]string) func(string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		if content, ok := files[path]; ok {
			return []byte(content), nil
		}
		return nil, fmt.Errorf("no such file: %s", path)
	}
}

const plainExample = `%% A tiny demo.
flowchart LR
  a[One] --> b[Two]

scenario "run" { speed: 1.0 }

  step go "Go" {
    flow a -> b { dur: 500ms }
  }
`

// linkedTo returns an example that pulls in target, which makes target a
// sub-view rather than a demo of its own.
func linkedTo(target string) string {
	return fmt.Sprintf(`%%%% A demo with a drill-down.
flowchart LR
  a[One] --> b[Two]

view inner "Inside" from %q

interact {
  click b -> view inner { label: "drill" }
}

scenario "run" { speed: 1.0 }

  step go "Go" {
    flow a -> b { dur: 500ms }
  }
`, target)
}

func TestBuildRendersAPagePerStandaloneExample(t *testing.T) {
	out, _, err := Build(
		[]string{"examples/linked.dgm", "examples/plain.dgm", "examples/sub.dgm"},
		mem(map[string]string{
			"examples/linked.dgm": linkedTo("sub.dgm"),
			"examples/plain.dgm":  plainExample,
			"examples/sub.dgm":    plainExample,
		}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	for _, want := range []string{"index.html", ".nojekyll", "demos/linked.html", "demos/plain.html"} {
		if _, ok := out[want]; !ok {
			t.Errorf("missing output %s; have %v", want, sortedKeys(out))
		}
	}

	// sub.dgm is reachable from linked.dgm, so it is part of that demo's
	// bundle rather than a demo in its own right.
	if _, ok := out["demos/sub.html"]; ok {
		t.Error("sub.dgm is a sub-view and should not get its own page")
	}
}

func TestMutuallyReferencingExamplesStillPublish(t *testing.T) {
	// a.dgm and b.dgm drill into each other, which the loader supports (a
	// cycle terminates on its own). Each is the other's sub-view, so a naive
	// "referenced means no page" rule would drop both from the site.
	out, _, err := Build(
		[]string{"examples/a.dgm", "examples/b.dgm"},
		mem(map[string]string{
			"examples/a.dgm": linkedTo("b.dgm"),
			"examples/b.dgm": linkedTo("a.dgm"),
		}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if _, ok := out["demos/a.html"]; !ok {
		t.Errorf("neither side of the cycle was published; have %v", sortedKeys(out))
	}
	// b stays unpublished: the player reaches it from a's page.
	if _, ok := out["demos/b.html"]; ok {
		t.Error("b.dgm is reachable from the published a.html and should not get its own page")
	}
}

func TestExamplesSharingABasenameAreAnError(t *testing.T) {
	_, _, err := Build(
		[]string{"examples/api/dup.dgm", "examples/web/dup.dgm"},
		mem(map[string]string{
			"examples/api/dup.dgm": plainExample,
			"examples/web/dup.dgm": plainExample,
		}))
	if err == nil {
		t.Fatal("Build accepted two examples that would publish as the same page")
	}
	if !strings.Contains(err.Error(), "dup.html") {
		t.Errorf("error does not name the colliding page: %v", err)
	}
}

func TestDemoPagesAreCompleteDocuments(t *testing.T) {
	out, _, err := Build(
		[]string{"examples/plain.dgm"},
		mem(map[string]string{"examples/plain.dgm": plainExample}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	page := string(out["demos/plain.html"])
	if !strings.HasPrefix(page, "<!doctype html>") {
		t.Errorf("demo page does not start with a doctype: %.40q", page)
	}
	if !strings.Contains(page, "CINEGRAM_TIMELINE") {
		t.Error("demo page carries no timeline payload")
	}
}

func TestIndexLinksEveryDemoWithItsBlurb(t *testing.T) {
	out, _, err := Build(
		[]string{"examples/plain.dgm"},
		mem(map[string]string{"examples/plain.dgm": plainExample}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	index := string(out["index.html"])
	if !strings.Contains(index, `href="demos/plain.html"`) {
		t.Error("index does not link demos/plain.html")
	}
	// The first %% comment line is the example's own description of itself.
	if !strings.Contains(index, "A tiny demo.") {
		t.Error("index does not carry the example's leading comment as its blurb")
	}
	// The page title comes from the timeline, the same way the demo page's
	// <title> does.
	if !strings.Contains(index, "run") {
		t.Error("index does not name the demo by its title")
	}
}

func TestBlurbIsTheLeadingCommentBlock(t *testing.T) {
	// The examples' convention: a summary that may wrap across lines, with
	// `%% ---` separating it from implementation notes the index should not
	// show.
	wrapped := "%% A demo whose summary\n%% wraps across two lines.\n%% ---\n%% Internal notes.\n" +
		strings.TrimPrefix(plainExample, "%% A tiny demo.\n")

	out, _, err := Build(
		[]string{"examples/wrapped.dgm"},
		mem(map[string]string{"examples/wrapped.dgm": wrapped}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	index := string(out["index.html"])
	if !strings.Contains(index, "A demo whose summary wraps across two lines.") {
		t.Error("blurb does not join the wrapped comment lines")
	}
	if strings.Contains(index, "Internal notes.") {
		t.Error("blurb runs past the %% --- separator")
	}
}

func TestDirectivesAreNotBlurbs(t *testing.T) {
	// A Mermaid `%%{init: …}%%` directive is configuration, not a comment —
	// its JSON must not surface as the demo's description.
	directive := "%%{init: {\"theme\":\"dark\"}}%%\n" +
		strings.TrimPrefix(plainExample, "%% A tiny demo.\n")

	out, _, err := Build(
		[]string{"examples/directive.dgm"},
		mem(map[string]string{"examples/directive.dgm": directive}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if index := string(out["index.html"]); strings.Contains(index, "init:") {
		t.Error("index shows a %%{init} directive as a blurb")
	}
}

func TestBuildFailsOnAnExampleThatDoesNotCompile(t *testing.T) {
	broken := strings.Replace(plainExample, "flow a -> b", "flow a -> zz", 1)

	_, _, err := Build(
		[]string{"examples/broken.dgm"},
		mem(map[string]string{"examples/broken.dgm": broken}))
	if err == nil {
		t.Fatal("Build accepted an example that does not compile")
	}
	if !strings.Contains(err.Error(), "examples/broken.dgm") {
		t.Errorf("error does not name the broken file: %v", err)
	}
}
