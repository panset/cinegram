package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const serveSource = `flowchart LR
  a[Entry] --> b[Middle]

scenario "walk"
  step one "go" {
    flow a -> b { dur: 400ms }
  }
`

// fakeFS is a source tree with mtimes, so the watcher can be driven without
// touching a disk or waiting on a clock.
type fakeFS struct {
	files  map[string]string
	stamps map[string]time.Time
}

func newFakeFS() *fakeFS {
	return &fakeFS{
		files:  map[string]string{"main.dgm": serveSource},
		stamps: map[string]time.Time{"main.dgm": time.Unix(1000, 0)},
	}
}

func (f *fakeFS) read(path string) ([]byte, error) {
	content, ok := f.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return []byte(content), nil
}

func (f *fakeFS) stat(path string) (time.Time, error) {
	t, ok := f.stamps[path]
	if !ok {
		return time.Time{}, os.ErrNotExist
	}
	return t, nil
}

func (f *fakeFS) touch(path string, content string) {
	f.files[path] = content
	f.stamps[path] = f.stamps[path].Add(time.Second)
}

func serveFixture(t *testing.T, watch bool) (*server, *fakeFS) {
	t.Helper()
	fs := newFakeFS()
	s := newServer("main.dgm", watch, nil)
	s.read = fs.read
	s.stat = fs.stat
	return s, fs
}

// TestServePageCompilesFromSource checks the server builds the real page rather
// than serving a snapshot taken at start-up.
func TestServePageCompilesFromSource(t *testing.T) {
	s, _ := serveFixture(t, false)

	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\n%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "DIAGRAMATOR_TIMELINE") {
		t.Error("page is missing the compiled timeline")
	}
	if !strings.Contains(body, "Entry") {
		t.Error("page does not contain the diagram's own text")
	}
	// Without --watch the served bytes must match what `preview -o` writes.
	if strings.Contains(body, "generation") {
		t.Error("reload script was injected without --watch")
	}
}

// TestServeInjectsReloadOnlyWhenWatching pins the rule that keeps the emitted
// file byte-identical to non-serve output: the reload script is added by the
// handler, never by the renderer.
func TestServeInjectsReloadOnlyWhenWatching(t *testing.T) {
	s, _ := serveFixture(t, true)

	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	body := rec.Body.String()

	if !strings.Contains(body, "location.reload()") {
		t.Fatal("watch mode did not inject the reload script")
	}
	// It has to land inside the document, not after it.
	if strings.Index(body, "location.reload()") > strings.LastIndex(body, "</body>") {
		t.Error("reload script was appended after </body>")
	}
}

// TestGenerationBumpsOnChange is the watch loop's whole contract: the counter
// moves when a watched file does, and not otherwise.
func TestGenerationBumpsOnChange(t *testing.T) {
	s, fs := serveFixture(t, true)

	// Build once to seed the watch set from the bundle.
	if _, err := s.page(); err != nil {
		t.Fatalf("initial build failed: %v", err)
	}
	start := s.generation()

	if s.poll() {
		t.Error("generation moved with nothing changed")
	}
	if s.generation() != start {
		t.Errorf("generation = %d after a quiet poll, want %d", s.generation(), start)
	}

	fs.touch("main.dgm", serveSource+"\n%% edited\n")
	if !s.poll() {
		t.Fatal("a changed file did not register")
	}
	if s.generation() != start+1 {
		t.Errorf("generation = %d after an edit, want %d", s.generation(), start+1)
	}

	// And the next request reflects the edit rather than the first build.
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/generation", nil))
	if got := rec.Body.String(); got != "1" {
		t.Errorf("/generation returned %q, want \"1\"", got)
	}
}

// TestWatchFollowsNewlyReferencedFiles covers the case a snapshot of the file
// set would miss: adding a `view` should start watching its target without a
// restart.
func TestWatchFollowsNewlyReferencedFiles(t *testing.T) {
	s, fs := serveFixture(t, true)
	if _, err := s.page(); err != nil {
		t.Fatalf("initial build failed: %v", err)
	}

	fs.files["sub.dgm"] = "flowchart LR\n  x[X] --> y[Y]\n"
	fs.stamps["sub.dgm"] = time.Unix(2000, 0)
	fs.touch("main.dgm", strings.Replace(serveSource,
		"scenario \"walk\"",
		"view sub from \"sub.dgm\"\n\ninteract {\n  click a -> view sub\n}\n\nscenario \"walk\"", 1))

	if _, err := s.page(); err != nil {
		t.Fatalf("rebuild failed: %v", err)
	}

	fs.touch("sub.dgm", "flowchart LR\n  x[X] --> y[Y]\n%% edited\n")
	if !s.poll() {
		t.Error("an edit to a newly referenced view did not register")
	}
}

// TestServeReportsBrokenSourceInThePage keeps the reload loop usable when the
// document does not parse: an error you can read beats a blank page.
func TestServeReportsBrokenSourceInThePage(t *testing.T) {
	s, fs := serveFixture(t, true)
	delete(fs.files, "main.dgm")

	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "diagramator:") {
		t.Errorf("body does not explain the failure: %q", rec.Body.String())
	}
}

// TestInjectReloadPlacement covers the fallback for a page with no </body>,
// which should still get the script rather than silently losing it.
func TestInjectReloadPlacement(t *testing.T) {
	withBody := injectReload([]byte("<html><body>hi</body></html>"))
	if !bytes.Contains(withBody, []byte("location.reload()")) {
		t.Fatal("script missing")
	}
	if bytes.Index(withBody, []byte("location.reload()")) > bytes.Index(withBody, []byte("</body>")) {
		t.Error("script landed after </body>")
	}

	none := injectReload([]byte("no closing tag"))
	if !bytes.Contains(none, []byte("location.reload()")) {
		t.Error("script was dropped when there was no </body>")
	}
}

// TestFrameSpacing checks the arithmetic behind --frames: endpoints included,
// computed from the total so the last one lands exactly on the end.
func TestFrameSpacing(t *testing.T) {
	dir := t.TempDir()
	shots, err := captureTargets(captureOptions{output: dir, frames: 5}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(shots) != 5 {
		t.Fatalf("got %d shots, want 5", len(shots))
	}
	want := []int{0, 250, 500, 750, 1000}
	for i, s := range shots {
		if s.at != want[i] {
			t.Errorf("frame %d at %dms, want %dms", i+1, s.at, want[i])
		}
	}
}

// TestCaptureNeedsAnOutput checks both forms refuse to guess where to write.
func TestCaptureNeedsAnOutput(t *testing.T) {
	if _, err := captureTargets(captureOptions{frames: 1}, 1000); err == nil {
		t.Error("single frame with no -o should be an error")
	}
	if _, err := captureTargets(captureOptions{frames: 4}, 1000); err == nil {
		t.Error("--frames with no -o should be an error")
	}
}

// TestFrameEndToEnd actually drives a browser, so it is opt-in.
//
// Run it outside the build sandbox — Bazel's darwin sandbox denies Chrome the
// Mach ports and crash-reporting paths it insists on, and the browser dies
// before it renders anything:
//
//	DIAGRAMATOR_CHROME="/path/to/Chrome" \
//	  bazel-bin/cmd/diagramator/diagramator_test_/diagramator_test \
//	  -test.run TestFrameEndToEnd
func TestFrameEndToEnd(t *testing.T) {
	if os.Getenv("DIAGRAMATOR_CHROME") == "" {
		t.Skip("set DIAGRAMATOR_CHROME to run the capture end to end")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "smoke.dgm")
	if err := os.WriteFile(src, []byte(serveSource), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "frame.png")

	var log bytes.Buffer
	err := runCapture(captureOptions{
		input: src, output: out, at: 200, frames: 1, width: 800, height: 600,
	}, &log)
	if err != nil {
		t.Fatalf("capture failed: %v\n%s", err, log.String())
	}

	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("no PNG written: %v", err)
	}
	if info.Size() < 1000 {
		t.Errorf("PNG is %d bytes, which is too small to be a rendered page", info.Size())
	}
}

// TestFindChromeRejectsABadOverride checks the explicit setting is validated
// rather than handed to exec to fail obscurely later.
func TestFindChromeRejectsABadOverride(t *testing.T) {
	t.Setenv("DIAGRAMATOR_CHROME", "/definitely/not/a/browser")
	if _, err := findChrome(); err == nil {
		t.Error("expected an error for a DIAGRAMATOR_CHROME that does not exist")
	} else if !strings.Contains(err.Error(), "DIAGRAMATOR_CHROME") {
		t.Errorf("error should name the variable: %v", err)
	}
}
