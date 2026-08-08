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
