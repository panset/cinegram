package sitegen

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
)

// mem builds an fs.FS over an in-memory tree of sources.
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

// slugs is every published document, in reading order.
func slugs(docs []*Doc) []string {
	out := make([]string, 0, len(docs))
	for _, d := range docs {
		out = append(out, d.Slug)
	}
	return out
}

func discover(t *testing.T, files map[string]string) (*Group, []*Doc) {
	t.Helper()
	root, docs, _, err := Discover(mem(files))
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	return root, docs
}

func TestASubViewIsReachedByDrillDownNotByItsOwnPage(t *testing.T) {
	_, docs := discover(t, map[string]string{
		"linked.dgm": linkedTo("sub.dgm"),
		"plain.dgm":  plainExample,
		"sub.dgm":    plainExample,
	})
	got := slugs(docs)
	sort.Strings(got)
	if want := []string{"linked", "plain"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("published %v, want %v — sub.dgm is part of linked's bundle", got, want)
	}
}

func TestMutuallyReferencingExamplesStillPublish(t *testing.T) {
	// a.dgm and b.dgm drill into each other, which the loader supports (a
	// cycle terminates on its own). Each is the other's sub-view, so a naive
	// "referenced means no page" rule would drop both from the site.
	_, docs := discover(t, map[string]string{
		"a.dgm": linkedTo("b.dgm"),
		"b.dgm": linkedTo("a.dgm"),
	})
	if got := slugs(docs); len(got) != 1 || got[0] != "a" {
		t.Errorf("published %v; want just a, with b reached from a's page", got)
	}
}

func TestStrippedNamesCollidingInAFolderAreAnError(t *testing.T) {
	// The numeric prefix is display order, not identity: once stripped, these
	// both want the same page and one would silently shadow the other.
	_, _, _, err := Discover(mem(map[string]string{
		"01-dup.dgm": plainExample,
		"02-dup.dgm": plainExample,
	}))
	if err == nil {
		t.Fatal("Discover accepted two examples that would publish as the same page")
	}
	if !strings.Contains(err.Error(), "01-dup.dgm") || !strings.Contains(err.Error(), "02-dup.dgm") {
		t.Errorf("error does not name both colliding sources: %v", err)
	}
}

func TestSameBasenameInDifferentFoldersIsFine(t *testing.T) {
	_, docs := discover(t, map[string]string{
		"api/dup.dgm": plainExample,
		"web/dup.dgm": plainExample,
	})
	got := slugs(docs)
	sort.Strings(got)
	if strings.Join(got, ",") != "api/dup,web/dup" {
		t.Errorf("published %v; the folder is part of the identity", got)
	}
}

func TestNumericPrefixesOrderAndDisappear(t *testing.T) {
	_, docs := discover(t, map[string]string{
		"02-second.dgm": plainExample,
		"01-first.dgm":  plainExample,
		"also.dgm":      plainExample,
	})
	// Prefixed entries first in numeric order, then the alphabet — and the
	// prefix is gone from the slug, which is what a URL is built from.
	if got := strings.Join(slugs(docs), ","); got != "first,second,also" {
		t.Errorf("reading order is %q, want \"first,second,also\"", got)
	}
	// The source path keeps the prefix: it is how the file is actually named.
	if docs[0].Source != "01-first.dgm" {
		t.Errorf("Source is %q; it should be the file as it exists", docs[0].Source)
	}
}

func TestReadingOrderIsDepthFirstAndCrossesFolders(t *testing.T) {
	_, docs := discover(t, map[string]string{
		"01-basics/01-first.dgm": plainExample,
		"01-basics/02-next.dgm":  plainExample,
		"02-more/01-last.dgm":    plainExample,
	})
	if got := strings.Join(slugs(docs), ","); got != "01-basics/first,01-basics/next,02-more/last" {
		t.Errorf("reading order is %q", got)
	}
	if docs[1].Next() != docs[2] || docs[2].Prev() != docs[1] {
		t.Error("prev/next do not cross the folder edge")
	}
	if docs[0].Prev() != nil || docs[2].Next() != nil {
		t.Error("the ends of the book point somewhere")
	}
}

func TestBlurbIsTheLeadingCommentBlock(t *testing.T) {
	// The examples' convention: a summary that may wrap across lines, with
	// `%% ---` separating it from implementation notes no index should show.
	wrapped := "%% A demo whose summary\n%% wraps across two lines.\n%% ---\n%% Internal notes.\n" +
		strings.TrimPrefix(plainExample, "%% A tiny demo.\n")

	_, docs := discover(t, map[string]string{"wrapped.dgm": wrapped})
	if got := docs[0].Blurb; got != "A demo whose summary wraps across two lines." {
		t.Errorf("blurb is %q", got)
	}
}

func TestDirectivesAreNotBlurbs(t *testing.T) {
	// A Mermaid `%%{init: …}%%` directive is configuration, not a comment —
	// its JSON must not surface as the demo's description.
	directive := "%%{init: {\"theme\":\"dark\"}}%%\n" +
		strings.TrimPrefix(plainExample, "%% A tiny demo.\n")

	_, docs := discover(t, map[string]string{"directive.dgm": directive})
	if strings.Contains(docs[0].Blurb, "init:") {
		t.Errorf("a %%%%{init} directive became a blurb: %q", docs[0].Blurb)
	}
}

func TestDiscoverFailsOnAnExampleThatDoesNotCompile(t *testing.T) {
	broken := strings.Replace(plainExample, "flow a -> b", "flow a -> zz", 1)

	_, _, _, err := Discover(mem(map[string]string{"broken.dgm": broken}))
	if err == nil {
		t.Fatal("Discover accepted an example that does not compile")
	}
	if !strings.Contains(err.Error(), "broken.dgm") {
		t.Errorf("error does not name the broken file: %v", err)
	}
}

func TestDocCarriesItsSourceAndAShareLink(t *testing.T) {
	_, docs := discover(t, map[string]string{"plain.dgm": plainExample})
	if docs[0].Text != plainExample {
		t.Error("Doc.Text is not the source as written")
	}
	// The fragment is opaque here — what matters is that it exists, since a
	// renderer writes it into an href without being able to check it.
	if docs[0].Share == "" {
		t.Error("Doc.Share is empty; the Edit-in-playground link would go nowhere")
	}
}

// --- the HTML site --------------------------------------------------------

func TestBuildRendersAPagePerDocumentPlusFolderIndexes(t *testing.T) {
	out, _, err := Build(mem(map[string]string{
		"plain.dgm":      plainExample,
		"deep/other.dgm": plainExample,
	}), Config{Title: "Demo", Playground: "playground/", Hero: "Try it"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	for _, want := range []string{
		".nojekyll", "index.html", "plain.html", "deep/index.html", "deep/other.html",
		"assets/mermaid.min.js", "assets/runtime.js", "assets/runtime.css", "assets/site.css",
	} {
		if _, ok := out[want]; !ok {
			t.Errorf("missing output %s", want)
		}
	}
}

func TestPagesShareAssetsAndCarryChrome(t *testing.T) {
	out, _, err := Build(mem(map[string]string{
		"plain.dgm":      plainExample,
		"deep/other.dgm": plainExample,
	}), Config{Title: "Demo", Playground: "playground/"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	page := string(out["plain.html"])
	if !strings.HasPrefix(page, "<!doctype html>") {
		t.Errorf("page does not start with a doctype: %.40q", page)
	}
	if !strings.Contains(page, "CINEGRAM_TIMELINE") {
		t.Error("page carries no timeline payload")
	}
	// Shared, not inlined: the page references the assets folder and does not
	// carry mermaid's megabytes.
	if !strings.Contains(page, `src="assets/mermaid.min.js"`) {
		t.Error("page does not reference the shared mermaid copy")
	}
	if len(page) > 200_000 {
		t.Errorf("page is %d bytes; shared assets should keep it small", len(page))
	}
	if !strings.Contains(page, "dgm-site-nav") {
		t.Error("page carries no sidebar")
	}
	if !strings.Contains(page, "Edit in playground") {
		t.Error("page carries no Edit-in-playground button")
	}

	// A page one folder down reaches the same assets one level up.
	if deep := string(out["deep/other.html"]); !strings.Contains(deep, `src="../assets/mermaid.min.js"`) {
		t.Error("nested page does not reach the shared assets via ../")
	}
}

func TestIndexLinksEveryDemoWithItsBlurb(t *testing.T) {
	out, _, err := Build(mem(map[string]string{"plain.dgm": plainExample}),
		Config{Title: "Demo", Playground: "playground/", Hero: "Try it"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	index := string(out["index.html"])
	if !strings.Contains(index, `href="plain.html"`) {
		t.Error("index does not link plain.html")
	}
	if !strings.Contains(index, "A tiny demo.") {
		t.Error("index does not carry the example's leading comment as its blurb")
	}
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
	}), Config{Title: "Demo"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	index := listing(string(out["index.html"]))
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
	out, _, err := Build(mem(files), Config{Title: "Demo"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	index := listing(string(out["index.html"]))
	if !strings.Contains(index, "and 2 more") {
		t.Errorf("root index does not say how much of the folder it left out; have:\n%s", index)
	}
	if strings.Contains(index, `href="deep/g.html"`) {
		t.Error("root index spells out a folder past the cap")
	}
}
