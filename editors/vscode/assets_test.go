// Package assets exists only to hold a test.
//
// editors/vscode/media/ carries copies of the three browser assets that
// pkg/emit/html/assets/ owns. The copies are not a convenience — they are
// forced twice over. `go:embed` cannot reach outside its own package directory,
// so the canonical files have to live next to the Go code that inlines them;
// and `markdown.previewScripts` takes static, extension-relative paths, so the
// extension cannot point at a file outside its own folder either.
//
// Two copies of a file is a promise to keep them equal, and the only promise
// worth making is one a build can check. This test is that check.
package assets

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/tejaspanse/cinegram/internal/repotest"
)

// synced lists every file the extension keeps a copy of, canonical path first.
//
// The licence is here for the same reason the browser assets are: a .vsix holds
// nothing outside the extension folder, and the Marketplace shows the licence
// from inside the package. The repository's own LICENSE stays canonical.
var synced = []struct{ canonical, copy string }{
	{"pkg/emit/html/assets/runtime.js", "editors/vscode/media/runtime.js"},
	{"pkg/emit/html/assets/runtime.css", "editors/vscode/media/runtime.css"},
	{"pkg/emit/html/assets/mermaid.min.js", "editors/vscode/media/mermaid.min.js"},
	{"LICENSE", "editors/vscode/LICENSE.txt"},
}

// TestAssetsAreInSync fails when the extension's copies have fallen behind.
//
// This is the guard on the one way the VS Code extension can quietly stop
// matching the compiler it renders for. Everything else about a diagram — the
// vocabulary, the timing rules, the diagram types — reaches the preview through
// the binary at runtime, so it cannot drift. These three files are the
// exception, because they are shipped rather than called.
func TestAssetsAreInSync(t *testing.T) {
	// The first synced file doubles as the probe: it is in this test's data
	// either way the test is run.
	root := repotest.Root(t, synced[0].canonical)

	for _, pair := range synced {
		want, err := os.ReadFile(filepath.Join(root, pair.canonical))
		if err != nil {
			t.Fatalf("reading %s: %v", pair.canonical, err)
		}
		got, err := os.ReadFile(filepath.Join(root, pair.copy))
		if err != nil {
			t.Fatalf("reading %s: %v\nrun: bazel run //editors/vscode:sync_assets", pair.copy, err)
		}
		if !bytes.Equal(want, got) {
			t.Errorf("%s is stale (%d bytes, canonical is %d)\nrun: bazel run //editors/vscode:sync_assets",
				pair.copy, len(got), len(want))
		}
	}
}
