// Command sync writes the generated half of the website into www/.
//
// It is the fix for the failure that //site:site_test reports:
//
//	bazel run //site:sync
//
// Writing goes through BUILD_WORKSPACE_DIRECTORY, which only `bazel run` sets —
// under `bazel test` the sandbox would take the write somewhere that is thrown
// away. That is the same arrangement the golden fixtures and
// //editors/vscode:sync_assets use.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/tejaspanse/cinegram/site"
)

func main() {
	root := os.Getenv("BUILD_WORKSPACE_DIRECTORY")
	if root == "" {
		fmt.Fprintln(os.Stderr, "sync: run this with `bazel run //site:sync`, which sets BUILD_WORKSPACE_DIRECTORY")
		os.Exit(1)
	}

	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		fail(err)
	}
	pages, warnings, err := site.Build(os.DirFS(filepath.Join(root, "examples")), readme)
	if err != nil {
		fail(err)
	}
	// A warning never blocks publishing, but it ships: the committer is the
	// last person to see it before the page goes live.
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, w)
	}

	www := filepath.Join(root, "www")
	changed := 0
	for rel, content := range pages {
		to := filepath.Join(www, filepath.FromSlash(rel))

		if existing, err := os.ReadFile(to); err == nil && string(existing) == string(content) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			fail(err)
		}
		if err := os.WriteFile(to, content, 0o644); err != nil {
			fail(err)
		}
		fmt.Printf("updated www/%s\n", rel)
		changed++
	}

	// A page whose example was renamed or removed must not stay on the site.
	// Only the folders site.Generated names are swept: the home page, the
	// embedding guide and the plans are hand-written and are not this tool's
	// to delete. Emptied directories go too.
	var stale, dirs []string
	for _, gen := range site.Generated {
		base := filepath.Join(www, filepath.FromSlash(gen))
		err := filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				// The folder not existing yet is the state before the first
				// sync, not a problem to report.
				if os.IsNotExist(err) {
					return fs.SkipAll
				}
				return err
			}
			if d.IsDir() {
				if p != base {
					dirs = append(dirs, p)
				}
				return nil
			}
			rel, err := filepath.Rel(www, p)
			if err != nil {
				return err
			}
			if _, ok := pages[filepath.ToSlash(rel)]; !ok {
				stale = append(stale, p)
			}
			return nil
		})
		if err != nil {
			fail(err)
		}
	}
	for _, p := range stale {
		if err := os.Remove(p); err != nil {
			fail(err)
		}
		rel, _ := filepath.Rel(www, p)
		fmt.Printf("removed www/%s\n", filepath.ToSlash(rel))
		changed++
	}
	// Deepest first, so a chain of emptied folders unwinds in one pass; a
	// Remove on a non-empty directory fails, which is exactly the guard we
	// want against deleting something hand-placed.
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, d := range dirs {
		if entries, err := os.ReadDir(d); err == nil && len(entries) == 0 {
			if err := os.Remove(d); err == nil {
				rel, _ := filepath.Rel(www, d)
				fmt.Printf("removed www/%s/\n", filepath.ToSlash(rel))
				changed++
			}
		}
	}

	if changed == 0 {
		fmt.Println("already in sync")
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "sync: %v\n", err)
	os.Exit(1)
}
