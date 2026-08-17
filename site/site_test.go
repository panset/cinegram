package site

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
)

// mem builds an fs.FS over an in-memory tree, paths relative to examples/.
func mem(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for p, content := range files {
		fsys[p] = &fstest.MapFile{Data: []byte(content)}
	}
	return fsys
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
	out, _, err := Build(mem(map[string]string{
		"linked.dgm": linkedTo("sub.dgm"),
		"plain.dgm":  plainExample,
		"sub.dgm":    plainExample,
	}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	for _, want := range []string{
		"index.html", ".nojekyll",
		"demos/index.html", "demos/linked.html", "demos/plain.html",
		"demos/assets/mermaid.min.js", "demos/assets/runtime.js",
		"demos/assets/runtime.css", "demos/assets/site.css",
	} {
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

func TestTopLevelIndexRedirectsIntoDemos(t *testing.T) {
	out, _, err := Build(mem(map[string]string{"plain.dgm": plainExample}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	stub := string(out["index.html"])
	if !strings.Contains(stub, `url=demos/`) {
		t.Errorf("top-level index does not redirect into demos/: %q", stub)
	}
}

func TestMutuallyReferencingExamplesStillPublish(t *testing.T) {
	// a.dgm and b.dgm drill into each other, which the loader supports (a
	// cycle terminates on its own). Each is the other's sub-view, so a naive
	// "referenced means no page" rule would drop both from the site.
	out, _, err := Build(mem(map[string]string{
		"a.dgm": linkedTo("b.dgm"),
		"b.dgm": linkedTo("a.dgm"),
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

func TestStrippedNamesCollidingInAFolderAreAnError(t *testing.T) {
	// The numeric prefix is display order, not identity: once stripped,
	// these both want dup.html and one would silently shadow the other.
	_, _, err := Build(mem(map[string]string{
		"01-dup.dgm": plainExample,
		"02-dup.dgm": plainExample,
	}))
	if err == nil {
		t.Fatal("Build accepted two examples that would publish as the same page")
	}
	if !strings.Contains(err.Error(), "dup.html") {
		t.Errorf("error does not name the colliding page: %v", err)
	}
}

func TestSameBasenameInDifferentFoldersIsFine(t *testing.T) {
	out, _, err := Build(mem(map[string]string{
		"api/dup.dgm": plainExample,
		"web/dup.dgm": plainExample,
	}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	for _, want := range []string{"demos/api/dup.html", "demos/web/dup.html", "demos/api/index.html"} {
		if _, ok := out[want]; !ok {
			t.Errorf("missing output %s; have %v", want, sortedKeys(out))
		}
	}
}

func TestDemoPagesShareAssetsAndCarryChrome(t *testing.T) {
	out, _, err := Build(mem(map[string]string{
		"plain.dgm":      plainExample,
		"deep/other.dgm": plainExample,
	}))
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
	// Shared, not inlined: the page references the assets folder and does
	// not carry mermaid's megabytes.
	if !strings.Contains(page, `src="assets/mermaid.min.js"`) {
		t.Error("demo page does not reference the shared mermaid copy")
	}
	if len(page) > 200_000 {
		t.Errorf("demo page is %d bytes; shared assets should keep it small", len(page))
	}
	if !strings.Contains(page, "dgm-site-nav") {
		t.Error("demo page carries no sidebar")
	}
	if !strings.Contains(page, "Edit in playground") {
		t.Error("demo page carries no Edit-in-playground button")
	}

	// A page one folder down reaches the same assets one level up.
	deep := string(out["demos/deep/other.html"])
	if !strings.Contains(deep, `src="../assets/mermaid.min.js"`) {
		t.Error("nested page does not reach the shared assets via ../")
	}
}

func TestIndexLinksEveryDemoWithItsBlurb(t *testing.T) {
	out, _, err := Build(mem(map[string]string{"plain.dgm": plainExample}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	index := string(out["demos/index.html"])
	if !strings.Contains(index, `href="plain.html"`) {
		t.Error("index does not link plain.html")
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
	// The repo's presentation: the hero card links the playground.
	if !strings.Contains(index, "Try the playground") {
		t.Error("index carries no playground hero card")
	}
}

func TestAFolderEntryLinksWhatIsInsideIt(t *testing.T) {
	// A site organised into folders would otherwise greet a reader with a
	// landing page of bare folder names, where a flat one listed every demo.
	out, _, err := Build(mem(map[string]string{
		"01-basics/01-first.dgm":  plainExample,
		"01-basics/02-second.dgm": plainExample,
	}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	index := listing(string(out["demos/index.html"]))
	if !strings.Contains(index, `href="01-basics/index.html"`) {
		t.Error("root index does not link the folder itself")
	}
	for _, want := range []string{`href="01-basics/first.html"`, `href="01-basics/second.html"`} {
		if !strings.Contains(index, want) {
			t.Errorf("root index does not reach %s in one click; have:\n%s", want, index)
		}
	}
}

// listing is an index page's <main> — the sidebar carries the whole tree on
// every page, so a claim about what the listing itself shows has to exclude it.
func listing(page string) string {
	_, main, _ := strings.Cut(page, "<main")
	return main
}

func TestAFolderEntryStopsListingAfterSix(t *testing.T) {
	files := map[string]string{}
	for _, name := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		files["deep/"+name+".dgm"] = plainExample
	}
	out, _, err := Build(mem(files))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	index := listing(string(out["demos/index.html"]))
	if !strings.Contains(index, "and 2 more") {
		t.Errorf("root index does not say how much of the folder it left out; have:\n%s", index)
	}
	if strings.Contains(index, `href="deep/g.html"`) {
		t.Error("root index spells out a folder past the cap")
	}
}

func TestNumericPrefixesOrderAndDisappear(t *testing.T) {
	out, _, err := Build(mem(map[string]string{
		"02-second.dgm": plainExample,
		"01-first.dgm":  plainExample,
		"also.dgm":      plainExample,
	}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	index := string(out["demos/index.html"])
	first := strings.Index(index, `href="first.html"`)
	second := strings.Index(index, `href="second.html"`)
	also := strings.Index(index, `href="also.html"`)
	if first < 0 || second < 0 || also < 0 {
		t.Fatalf("index is missing prefix-stripped links; have:\n%s", index)
	}
	if !(first < second && second < also) {
		t.Error("index does not order by numeric prefix before the alphabet")
	}
	// The .source span deliberately shows the real filename, prefix and all;
	// links and page names must not.
	if strings.Contains(index, `href="01-first`) {
		t.Error("the numeric prefix leaked into a link")
	}
}

func TestBlurbIsTheLeadingCommentBlock(t *testing.T) {
	// The examples' convention: a summary that may wrap across lines, with
	// `%% ---` separating it from implementation notes the index should not
	// show.
	wrapped := "%% A demo whose summary\n%% wraps across two lines.\n%% ---\n%% Internal notes.\n" +
		strings.TrimPrefix(plainExample, "%% A tiny demo.\n")

	out, _, err := Build(mem(map[string]string{"wrapped.dgm": wrapped}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	index := string(out["demos/index.html"])
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

	out, _, err := Build(mem(map[string]string{"directive.dgm": directive}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if index := string(out["demos/index.html"]); strings.Contains(index, "init:") {
		t.Error("index shows a %%{init} directive as a blurb")
	}
}

func TestBuildFailsOnAnExampleThatDoesNotCompile(t *testing.T) {
	broken := strings.Replace(plainExample, "flow a -> b", "flow a -> zz", 1)

	_, _, err := Build(mem(map[string]string{"broken.dgm": broken}))
	if err == nil {
		t.Fatal("Build accepted an example that does not compile")
	}
	if !strings.Contains(err.Error(), "broken.dgm") {
		t.Errorf("error does not name the broken file: %v", err)
	}
}
