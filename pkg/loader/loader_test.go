package loader

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// fakeFS reads from an in-memory map, so these tests exercise path resolution
// without a temp directory and without the parser ever touching disk.
func fakeFS(files map[string]string) ReadFileFunc {
	return func(path string) ([]byte, error) {
		content, ok := files[filepath.Clean(path)]
		if !ok {
			return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
		}
		return []byte(content), nil
	}
}

// doc builds a minimal source file declaring the given views.
func doc(body string) string {
	return "flowchart LR\n  a[A]\n  b[B]\n  a --> b\n\n" + body
}

func viewIDs(b *Bundle) []string {
	out := make([]string, 0, len(b.Units))
	for _, u := range b.Units {
		out = append(out, u.ViewID)
	}
	return out
}

func TestResolvesRelativeToTheReferencingFile(t *testing.T) {
	files := map[string]string{
		"diagrams/top.dgm": doc("view sub from \"nested/deep.dgm\"\n\ninteract {\n  click a -> view sub\n}\n"),
		// The nested file's own reference is relative to itself, not to the
		// entry file: this is the case a naive join gets wrong.
		"diagrams/nested/deep.dgm": doc("view leaf from \"leaf.dgm\"\n\ninteract {\n  click a -> view leaf\n}\n"),
		"diagrams/nested/leaf.dgm": doc(""),
		"diagrams/leaf.dgm":        doc("%% wrong one: relative to the entry file\n"),
	}

	b, err := Load("diagrams/top.dgm", fakeFS(files))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(b.Units), 3; got != want {
		t.Fatalf("loaded %d units (%v), want %d", got, viewIDs(b), want)
	}
	if got := b.Units[2].Path; got != "diagrams/nested/leaf.dgm" {
		t.Errorf("leaf resolved to %q, want diagrams/nested/leaf.dgm", got)
	}
	if b.HasErrors() {
		t.Errorf("unexpected diagnostics:\n%s", b.Bags()[0])
	}
}

// TestDiamondLoadsOnce pins that view identity is the resolved path, not the
// alias: two documents pointing at the same file share one view.
func TestDiamondLoadsOnce(t *testing.T) {
	files := map[string]string{
		"top.dgm": doc("view l from \"left.dgm\"\nview r from \"right.dgm\"\n\n" +
			"interact {\n  click a -> view l\n  click b -> view r\n}\n"),
		"left.dgm":   doc("view s from \"shared.dgm\"\n\ninteract {\n  click a -> view s\n}\n"),
		"right.dgm":  doc("view other from \"shared.dgm\"\n\ninteract {\n  click a -> view other\n}\n"),
		"shared.dgm": doc(""),
	}

	b, err := Load("top.dgm", fakeFS(files))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(b.Units), 4; got != want {
		t.Fatalf("loaded %d units (%v), want %d", got, viewIDs(b), want)
	}

	// Both aliases must resolve to the same canonical view.
	var left, right *Unit
	for _, u := range b.Units {
		switch u.ViewID {
		case "left":
			left = u
		case "right":
			right = u
		}
	}
	if left == nil || right == nil {
		t.Fatalf("missing left/right units in %v", viewIDs(b))
	}
	if left.Views["s"] != right.Views["other"] {
		t.Errorf("aliases resolved differently: %q vs %q", left.Views["s"], right.Views["other"])
	}
	if left.Views["s"] != "shared" {
		t.Errorf("shared view id = %q, want shared", left.Views["s"])
	}
}

// TestCycleTerminates checks that mutual references load rather than hang.
// Two diagrams that link to each other is a reasonable thing to draw, and the
// runtime keeps a navigation stack rather than a tree, so it must simply work.
func TestCycleTerminates(t *testing.T) {
	files := map[string]string{
		"a.dgm": doc("view other from \"b.dgm\"\n\ninteract {\n  click a -> view other\n}\n"),
		"b.dgm": doc("view back from \"a.dgm\"\n\ninteract {\n  click a -> view back\n}\n"),
	}

	b, err := Load("a.dgm", fakeFS(files))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(b.Units), 2; got != want {
		t.Fatalf("loaded %d units (%v), want %d", got, viewIDs(b), want)
	}
	if b.Units[1].Views["back"] != b.Root {
		t.Errorf("the cycle did not resolve back to the root: %q", b.Units[1].Views["back"])
	}
}

// TestMissingFileIsReportedNotFatal keeps one broken path from hiding the rest
// of the document's problems.
func TestMissingFileIsReportedNotFatal(t *testing.T) {
	files := map[string]string{
		"top.dgm": doc("view gone from \"nope.dgm\"\nview ok from \"real.dgm\"\n\n" +
			"interact {\n  click a -> view gone\n  click b -> view ok\n}\n"),
		"real.dgm": doc(""),
	}

	b, err := Load("top.dgm", fakeFS(files))
	if err != nil {
		t.Fatalf("Load returned a fatal error for a missing reference: %v", err)
	}
	if got, want := len(b.Units), 2; got != want {
		t.Fatalf("loaded %d units (%v), want %d — the good reference should still load", got, viewIDs(b), want)
	}
	got := b.Bags()[0].String()
	if !strings.Contains(got, `cannot read view "gone"`) {
		t.Errorf("missing file not reported:\n%s", got)
	}
	if !strings.Contains(got, "top.dgm") {
		t.Errorf("diagnostic is not attributed to the referencing file:\n%s", got)
	}
}

// TestStoryboardImagesAreInlined covers the reason image loading lives here at
// all: the parser never touches a filesystem, and the emitted page has to work
// without one.
func TestStoryboardImagesAreInlined(t *testing.T) {
	files := map[string]string{
		"diagrams/top.dgm": doc(
			"storyboard \"Screens\" {\n" +
				"  frame one { img: \"frames/one.svg\", caption: \"first\" }\n" +
				"  frame two { img: \"shared/two.png\" }\n" +
				"  frame words { caption: \"no picture at all\" }\n" +
				"}\n\n" +
				"scenario \"x\"\n  step s \"walk\" {\n    scene one\n  }\n"),
		// Resolved relative to the declaring file, exactly as a view path is.
		"diagrams/frames/one.svg": "<svg/>",
		"diagrams/shared/two.png": "\x89PNG\r\n",
	}

	b, err := Load("diagrams/top.dgm", fakeFS(files))
	if err != nil {
		t.Fatal(err)
	}
	if b.HasErrors() {
		t.Fatalf("unexpected diagnostics:\n%s", b.Bags()[0])
	}

	data := b.Units[0].FrameData
	if got, want := data["frames/one.svg"], "data:image/svg+xml;base64,PHN2Zy8+"; got != want {
		t.Errorf("svg frame = %q, want %q", got, want)
	}
	if !strings.HasPrefix(data["shared/two.png"], "data:image/png;base64,") {
		t.Errorf("png frame = %q, want a png data URI", data["shared/two.png"])
	}
	// A caption-only frame contributes nothing to read, so it contributes
	// nothing to the map either.
	if len(data) != 2 {
		t.Errorf("FrameData = %v, want exactly the two frames that named an image", data)
	}
	// base64 cannot produce a `<`, which is what keeps the payload from
	// breaking out of the <script> element the timeline is embedded in.
	for path, uri := range data {
		if strings.ContainsAny(uri, "<>") {
			t.Errorf("frame %q inlined to a URI containing markup: %q", path, uri)
		}
	}
}

// TestBadFrameImagesAreReportedNotFatal keeps one unreadable picture from
// hiding the next one, the same way a broken view path does.
func TestBadFrameImagesAreReportedNotFatal(t *testing.T) {
	files := map[string]string{
		"top.dgm": doc(
			"storyboard {\n" +
				"  frame gone { img: \"nope.svg\" }\n" +
				"  frame weird { img: \"notes.txt\" }\n" +
				"  frame fine { img: \"real.png\" }\n" +
				"}\n\n" +
				"scenario \"x\"\n  step s \"walk\" {\n    scene fine\n  }\n"),
		"real.png": "\x89PNG\r\n",
	}

	b, err := Load("top.dgm", fakeFS(files))
	if err != nil {
		t.Fatalf("Load returned a fatal error for an unreadable frame: %v", err)
	}
	got := b.Bags()[0].String()
	if !strings.Contains(got, `cannot read the image for frame "gone"`) {
		t.Errorf("missing image not reported:\n%s", got)
	}
	if !strings.Contains(got, `frame "weird" has an image type this cannot inline`) {
		t.Errorf("unsupported extension not reported:\n%s", got)
	}
	if _, ok := b.Units[0].FrameData["real.png"]; !ok {
		t.Error("the good frame did not load; one broken image stopped the rest")
	}
}

func TestMissingEntryFileIsFatal(t *testing.T) {
	if _, err := Load("nope.dgm", fakeFS(nil)); err == nil {
		t.Fatal("Load succeeded for a missing entry file")
	}
}

// TestDiagnosticsStayPerFile is what makes a multi-file build reviewable: a
// problem in a drilled-into diagram must name that diagram, not the entry.
func TestDiagnosticsStayPerFile(t *testing.T) {
	files := map[string]string{
		"top.dgm": doc("view sub from \"sub.dgm\"\n\ninteract {\n  click a -> view sub\n}\n"),
		"sub.dgm": doc("") + "scenario \"bad\"\n  step s \"x\" {\n    highlight nosuchnode\n  }\n",
	}

	b, err := Load("top.dgm", fakeFS(files))
	if err != nil {
		t.Fatal(err)
	}
	if b.Bags()[0].HasErrors() {
		t.Errorf("entry file wrongly carries the sub-diagram's errors:\n%s", b.Bags()[0])
	}
	sub := b.Bags()[1].String()
	if !strings.Contains(sub, "sub.dgm") || !strings.Contains(sub, "nosuchnode") {
		t.Errorf("sub-diagram diagnostics not attributed to sub.dgm:\n%s", sub)
	}
}

// TestViewIDsAreUniqueAndStable pins the ids that end up in location.hash.
func TestViewIDsAreUniqueAndStable(t *testing.T) {
	files := map[string]string{
		"top.dgm": doc("view one from \"a/shared.dgm\"\nview two from \"b/shared.dgm\"\n\n" +
			"interact {\n  click a -> view one\n  click b -> view two\n}\n"),
		"a/shared.dgm": doc(""),
		"b/shared.dgm": doc(""),
	}

	b, err := Load("top.dgm", fakeFS(files))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, id := range viewIDs(b) {
		if seen[id] {
			t.Errorf("duplicate view id %q in %v", id, viewIDs(b))
		}
		seen[id] = true
		if strings.ContainsAny(id, "/. ") {
			t.Errorf("view id %q is not safe in a URL fragment", id)
		}
	}
}
