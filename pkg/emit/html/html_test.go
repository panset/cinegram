package html

import (
	"regexp"
	"strings"
	"testing"

	"github.com/tejaspanse/diagramator/pkg/ir"
)

func sample() *ir.Timeline {
	return &ir.Timeline{
		Version: ir.Version,
		Diagram: ir.Diagram{Type: "flowchart", Direction: "LR", Mermaid: "flowchart LR\n  a --> b\n"},
		Nodes:   []ir.Node{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}},
		Edges:   []ir.Edge{{ID: "e0", From: "a", To: "b"}},
		Scenarios: []ir.Scenario{{
			ID: "s0", Name: "demo", Duration: 600, Speed: 1,
			Steps: []ir.Step{{
				ID: "s", Name: "hop", Start: 0, End: 600,
				Tracks: []ir.Track{{Kind: ir.TrackFlow, Start: 0, End: 600, Edge: "e0"}},
			}},
		}},
	}
}

// TestPageIsSelfContained is the property the preview depends on: the page must
// work from the filesystem and inside a CSP-restricted VS Code webview, so it
// may not reference any external resource.
func TestPageIsSelfContained(t *testing.T) {
	page, err := Render(sample(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	got := string(page)

	external := regexp.MustCompile(`(?i)(src|href)\s*=\s*["']?(https?:)?//`)
	if m := external.FindString(got); m != "" {
		t.Errorf("page references an external resource: %q", m)
	}

	for _, want := range []string{"<!doctype html>", "globalThis[\"mermaid\"]", "window.Diagramator", "DIAGRAMATOR_TIMELINE"} {
		if !strings.Contains(got, want) {
			t.Errorf("page is missing %q", want)
		}
	}
}

// TestScriptPayloadCannotEscape checks that diagram text containing a closing
// script tag is neutralised rather than breaking out of the script element.
func TestScriptPayloadCannotEscape(t *testing.T) {
	tl := sample()
	tl.Diagram.Mermaid = "flowchart LR\n  a[\"</script><script>alert(1)</script>\"] --> b\n"

	page, err := Render(tl, Options{Title: "esc"})
	if err != nil {
		t.Fatal(err)
	}
	got := string(page)

	// Exactly the three script elements we emit: mermaid, runtime, bootstrap.
	if n := strings.Count(got, "</script>"); n != 3 {
		t.Errorf("found %d closing script tags, want 3 — the payload escaped", n)
	}
	if strings.Contains(got, "<script>alert(1)") {
		t.Error("injected script survived unescaped")
	}
}

// TestTitleFallsBackToScenario keeps the browser tab useful when no title is
// passed in.
func TestTitleFallsBackToScenario(t *testing.T) {
	page, err := Render(sample(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), "<title>demo</title>") {
		t.Error("expected the title to fall back to the first scenario name")
	}
}
