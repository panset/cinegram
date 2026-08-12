// Package repotest locates the repository root from inside a test.
//
// Two tests keep committed copies of generated content honest —
// //editors/vscode:assets_test for the extension's asset copies and
// //site:site_test for the GitHub Pages site — and both need the same answer:
// which directory do workspace-relative paths resolve against here?
package repotest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Root finds the directory that workspace-relative paths resolve against.
//
// Under `bazel test` that is somewhere in the runfiles tree, which mirrors the
// workspace layout but contains only declared data — no MODULE.bazel to look
// for. Under `bazel run` or `go test` it is the source tree itself. Rather
// than encode which, Root probes for a path the caller knows must be present
// either way — something in the test's own data dependencies.
func Root(t testing.TB, probe string) string {
	t.Helper()

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
	t.Fatalf("cannot find %s from any of %s", probe, strings.Join(candidates, ", "))
	return ""
}
