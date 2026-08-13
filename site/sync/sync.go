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
	"os"
	"path/filepath"

	"github.com/tejaspanse/cinegram/site"
)

func main() {
	root := os.Getenv("BUILD_WORKSPACE_DIRECTORY")
	if root == "" {
		fmt.Fprintln(os.Stderr, "sync: run this with `bazel run //site:sync`, which sets BUILD_WORKSPACE_DIRECTORY")
		os.Exit(1)
	}

	paths, err := site.Examples(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sync: listing examples: %v\n", err)
		os.Exit(1)
	}

	pages, warnings, err := site.Build(paths, os.ReadFile)
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
		to := filepath.Join(root, "docs", rel)

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
	// level of docs/ (a CNAME, say) is not this tool's to delete.
	stale, err := os.ReadDir(filepath.Join(root, "docs", "demos"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "sync: %v\n", err)
		os.Exit(1)
	}
	for _, page := range stale {
		rel := "demos/" + page.Name()
		if _, ok := pages[rel]; ok {
			continue
		}
		if err := os.Remove(filepath.Join(root, "docs", rel)); err != nil {
			fmt.Fprintf(os.Stderr, "sync: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("removed docs/%s\n", rel)
		changed++
	}

	if changed == 0 {
		fmt.Println("already in sync")
	}
}
