// Command sync copies the language reference into pkg/langref.
//
// It is the fix for the failure that //pkg/langref:langref_test reports:
//
//	bazel run //pkg/langref:sync
//
// Writing goes through BUILD_WORKSPACE_DIRECTORY, which only `bazel run` sets —
// under `bazel test` the sandbox would take the write somewhere that is thrown
// away. That is the same arrangement //editors/vscode:sync_assets and the
// golden fixtures use.
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// synced is the one pair, canonical first. It is deliberately a second copy of
// what langref_test.go names rather than a shared variable: the test is what
// fails when they disagree, and a test that imported its own expectations from
// the thing it checks would pass either way.
var synced = [][2]string{
	{"skills/cinegram/references/language.md", "pkg/langref/assets/language.md"},
}

func main() {
	root := os.Getenv("BUILD_WORKSPACE_DIRECTORY")
	if root == "" {
		fmt.Fprintln(os.Stderr, "sync: run this with `bazel run //pkg/langref:sync`, which sets BUILD_WORKSPACE_DIRECTORY")
		os.Exit(1)
	}

	changed := 0
	for _, pair := range synced {
		from := filepath.Join(root, pair[0])
		to := filepath.Join(root, pair[1])

		data, err := os.ReadFile(from)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sync: %v\n", err)
			os.Exit(1)
		}

		if existing, err := os.ReadFile(to); err == nil && string(existing) == string(data) {
			continue
		}

		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "sync: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(to, data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "sync: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("updated %s\n", pair[1])
		changed++
	}

	if changed == 0 {
		fmt.Println("already in sync")
	}
}
