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

// TestGeneratedPagesAreFresh fails when the committed pages under www/ have
// fallen behind the examples, the README or the renderer.
//
// Zensical builds the site from www/, and Bazel cannot run Zensical. So this
// is where the guarantee has to live: everything Zensical reads that a program
// wrote is regenerated here and diffed. It is the same promise
// //editors/vscode:assets_test makes for the extension's asset copies — a
// committed copy of generated content is only honest if a build checks it.
func TestGeneratedPagesAreFresh(t *testing.T) {
	root := repotest.Root(t, "examples")

	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("reading README.md: %v", err)
	}
	want, _, err := Build(os.DirFS(filepath.Join(root, "examples")), readme)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	www := filepath.Join(root, "www")
	for _, rel := range sortedKeys(want) {
		got, err := os.ReadFile(filepath.Join(www, filepath.FromSlash(rel)))
		if err != nil {
			t.Errorf("www/%s is missing\nrun: bazel run //site:sync", rel)
			continue
		}
		if !bytes.Equal(got, want[rel]) {
			t.Errorf("www/%s is stale (%d bytes, regenerated is %d)\nrun: bazel run //site:sync",
				rel, len(got), len(want[rel]))
		}
	}

	// A page whose example was renamed or removed would otherwise stay on the
	// site forever. Only the generated folders are walked: everything else
	// under www/ is prose nobody generated.
	for _, gen := range Generated {
		base := filepath.Join(www, filepath.FromSlash(gen))
		err := filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return fs.SkipAll
				}
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(www, p)
			if err != nil {
				return err
			}
			if _, ok := want[filepath.ToSlash(rel)]; !ok {
				t.Errorf("www/%s has nothing behind it\nrun: bazel run //site:sync", filepath.ToSlash(rel))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking www/%s: %v", gen, err)
		}
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
