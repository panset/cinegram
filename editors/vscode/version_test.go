// The version exists in three places — the CLI constant, the extension
// manifest, and the extension changelog — and, like the browser assets, the
// only promise worth making about copies is one a build can check. The
// release workflow leans on this test: it verifies only that the tag matches
// the CLI constant, and this test carries the agreement the rest of the way.
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

func TestVersionsAgree(t *testing.T) {
	root := repotest.Root(t, "cmd/cinegram/version.go")

	cli := matchIn(t, root, "cmd/cinegram/version.go", cliVersionRE)

	manifestPath := filepath.Join(root, "editors/vscode/package.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("reading %s: %v", manifestPath, err)
	}
	var manifest struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parsing %s: %v", manifestPath, err)
	}

	changelog := matchIn(t, root, "editors/vscode/CHANGELOG.md", changelogVersionRE)

	if cli != manifest.Version || cli != changelog {
		t.Errorf("versions disagree:\n  cmd/cinegram/version.go        %s\n  editors/vscode/package.json    %s\n  editors/vscode/CHANGELOG.md    %s (top entry)\nbump all three together — see RELEASING.md",
			cli, manifest.Version, changelog)
	}
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
