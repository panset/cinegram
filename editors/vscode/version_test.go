// The version exists in five places — the CLI constant, the extension manifest
// and changelog, and the npm and PyPI manifests — and, like the browser
// assets, the only promise worth making about copies is one a build can check.
// The release workflow leans on this test: it verifies only that the tag
// matches the CLI constant, and this test carries the agreement the rest of
// the way.
//
// The two packaging manifests earn their place here rather than somewhere
// under packaging/: their versions are not decoration but the release each
// shim downloads at run time, so `npx cinegram@X` fetching something other
// than vX is exactly the drift this test exists to prevent.
package assets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/tejaspanse/cinegram/internal/repotest"
)

var cliVersionRE = regexp.MustCompile(`const version = "([^"]+)"`)
var changelogVersionRE = regexp.MustCompile(`(?m)^## \[(\d+\.\d+\.\d+)\]`)
var pyprojectVersionRE = regexp.MustCompile(`(?m)^version = "(\d+\.\d+\.\d+)"$`)

func TestVersionsAgree(t *testing.T) {
	root := repotest.Root(t, "cmd/cinegram/version.go")

	cli := matchIn(t, root, "cmd/cinegram/version.go", cliVersionRE)
	extension := jsonVersionIn(t, root, "editors/vscode/package.json")
	changelog := matchIn(t, root, "editors/vscode/CHANGELOG.md", changelogVersionRE)
	npm := jsonVersionIn(t, root, "packaging/npm/package.json")
	pypi := matchIn(t, root, "packaging/pypi/pyproject.toml", pyprojectVersionRE)

	if cli != extension || cli != changelog || cli != npm || cli != pypi {
		t.Errorf("versions disagree:\n  cmd/cinegram/version.go        %s\n  editors/vscode/package.json    %s\n  editors/vscode/CHANGELOG.md    %s (top entry)\n  packaging/npm/package.json     %s\n  packaging/pypi/pyproject.toml  %s\nbump all five together — see RELEASING.md",
			cli, extension, changelog, npm, pypi)
	}
}

// jsonVersionIn reads the "version" field of a JSON manifest.
func jsonVersionIn(t *testing.T, root, rel string) string {
	t.Helper()
	path := filepath.Join(root, rel)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var manifest struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	if manifest.Version == "" {
		t.Fatalf("%s names no version", rel)
	}
	return manifest.Version
}

// matchIn returns the first capture of re in the named file, failing the test
// when the pattern is absent — a missing version is as wrong as a stale one.
func matchIn(t *testing.T, root, rel string, re *regexp.Regexp) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	m := re.FindSubmatch(data)
	if m == nil {
		t.Fatalf("%s does not match %v", rel, re)
	}
	return string(m[1])
}
