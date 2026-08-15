// Command sync writes the GitHub Pages site into docs/.
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

	pages, warnings, err := site.Build(os.DirFS(filepath.Join(root, "examples")))
	if err != nil {
		fmt.Fprintf(os.Stderr, "sync: %v\n", err)
		os.Exit(1)
	}
	// A warning never blocks publishing, but it ships: the committer is the
	// last person to see it before the page goes live.
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, w)
	}

	changed := 0
	for rel, content := range pages {
		to := filepath.Join(root, "docs", filepath.FromSlash(rel))

		if existing, err := os.ReadFile(to); err == nil && string(existing) == string(content) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "sync: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(to, content, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "sync: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("updated docs/%s\n", rel)
		changed++
	}

	// A page whose example was renamed or removed must not stay live on the
	// site. Only docs/demos/ is swept: anything a committer places at the top
	// level of docs/ (a CNAME, say) is not this tool's to delete. The walk is
	// recursive because the generated site nests (demos/assets/, subfolders),
	// and emptied directories go too.
	demosDir := filepath.Join(root, "docs", "demos")
	var stale []string
	var dirs []string
	err = filepath.WalkDir(demosDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != demosDir {
				dirs = append(dirs, p)
			}
			return nil
		}
		rel, err := filepath.Rel(filepath.Join(root, "docs"), p)
		if err != nil {
			return err
		}
		if _, ok := pages[filepath.ToSlash(rel)]; !ok {
			stale = append(stale, p)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "sync: %v\n", err)
		os.Exit(1)
	}
	for _, p := range stale {
		if err := os.Remove(p); err != nil {
			fmt.Fprintf(os.Stderr, "sync: %v\n", err)
			os.Exit(1)
		}
		rel, _ := filepath.Rel(filepath.Join(root, "docs"), p)
		fmt.Printf("removed docs/%s\n", filepath.ToSlash(rel))
		changed++
	}
	// Deepest first, so a chain of emptied folders unwinds in one pass; a
	// Remove on a non-empty directory fails, which is exactly the guard we
	// want against deleting something hand-placed.
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, d := range dirs {
		if entries, err := os.ReadDir(d); err == nil && len(entries) == 0 {
			if err := os.Remove(d); err == nil {
				rel, _ := filepath.Rel(filepath.Join(root, "docs"), d)
				fmt.Printf("removed docs/%s/\n", filepath.ToSlash(rel))
				changed++
			}
		}
	}

	if changed == 0 {
		fmt.Println("already in sync")
	}
}
