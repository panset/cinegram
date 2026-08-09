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
)

// synced lists every file the extension keeps a copy of, canonical path first.
var synced = []struct{ canonical, copy string }{
	{"pkg/emit/html/assets/runtime.js", "editors/vscode/media/runtime.js"},
	{"pkg/emit/html/assets/runtime.css", "editors/vscode/media/runtime.css"},
	{"pkg/emit/html/assets/mermaid.min.js", "editors/vscode/media/mermaid.min.js"},
}

// TestAssetsAreInSync fails when the extension's copies have fallen behind.
//
// This is the guard on the one way the VS Code extension can quietly stop
// matching the compiler it renders for. Everything else about a diagram — the
// vocabulary, the timing rules, the diagram types — reaches the preview through
// the binary at runtime, so it cannot drift. These three files are the
// exception, because they are shipped rather than called.
func TestAssetsAreInSync(t *testing.T) {
	root := repoRoot(t)

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

// repoRoot finds whichever directory the workspace-relative paths above resolve
// against.
//
// Under `bazel test` that is somewhere in the runfiles tree, which mirrors the
// workspace layout but contains only declared data — no MODULE.bazel to look
// for. Under `bazel run` or `go test` it is the source tree itself. Rather than
// encode which, probe for a file that must be present either way.
func repoRoot(t *testing.T) string {
	t.Helper()
	probe := synced[0].canonical

	var candidates []string
	if dir := os.Getenv("BUILD_WORKSPACE_DIRECTORY"); dir != "" {
		candidates = append(candidates, dir)
	}
	if wd, err := os.Getwd(); err == nil {
		for dir := wd; ; {
			candidates = append(candidates, dir)
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, probe)); err == nil {
			return dir
		}
	}
	t.Fatalf("cannot find %s from any of %v", probe, candidates)
	return ""
}
