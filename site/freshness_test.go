package site

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/tejaspanse/cinegram/internal/repotest"
)

// TestDocsAreFresh fails when the committed site under docs/ has fallen behind
// the examples or the renderer.
//
// GitHub Pages serves docs/ straight from the branch — there is no build step
// after the commit, so a stale artifact would ship as-is. This test is the
// same promise //editors/vscode:assets_test makes for the extension's asset
// copies: a committed copy of generated content is only honest if a build
// checks it.
func TestDocsAreFresh(t *testing.T) {
	root := repotest.Root(t, "examples")

	want, _, err := Build(os.DirFS(filepath.Join(root, "examples")))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	for _, rel := range sortedKeys(want) {
		got, err := os.ReadFile(filepath.Join(root, "docs", filepath.FromSlash(rel)))
		if err != nil {
			t.Errorf("docs/%s is missing\nrun: bazel run //site:sync", rel)
			continue
		}
		if !bytes.Equal(got, want[rel]) {
			t.Errorf("docs/%s is stale (%d bytes, regenerated is %d)\nrun: bazel run //site:sync",
				rel, len(got), len(want[rel]))
		}
	}

	// A demo page whose example was renamed or removed would otherwise stay
	// live on the site forever. The walk is recursive: the generated site
	// nests (demos/assets/, subfolders).
	demosDir := filepath.Join(root, "docs", "demos")
	err = filepath.WalkDir(demosDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(filepath.Join(root, "docs"), p)
		if err != nil {
			return err
		}
		if _, ok := want[filepath.ToSlash(rel)]; !ok {
			t.Errorf("docs/%s has no example behind it\nrun: bazel run //site:sync", filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking docs/demos: %v", err)
	}
}

// sortedKeys lists a map's keys in path order so failures list
// deterministically.
func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
