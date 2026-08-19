package langref

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tejaspanse/cinegram/internal/repotest"
)

// canonical is the copy that ships to users, and the one an author edits.
const canonical = "skills/cinegram/references/language.md"

// TestReferenceIsInSync fails when the embedded copy has fallen behind.
//
// This is the guard on the one way `cinegram mcp` could quietly start teaching
// a language the compiler no longer speaks. Everything else an MCP client
// learns about a document it learns by calling the binary, so it cannot drift;
// the reference is the exception, because it is shipped rather than computed.
func TestReferenceIsInSync(t *testing.T) {
	// The canonical file doubles as the probe: it is in this test's data
	// either way the test is run.
	root := repotest.Root(t, canonical)

	want, err := os.ReadFile(filepath.Join(root, canonical))
	if err != nil {
		t.Fatalf("reading %s: %v", canonical, err)
	}
	if got := Markdown(); got != string(want) {
		t.Errorf("pkg/langref/assets/language.md is stale (%d bytes, canonical is %d)\nrun: bazel run //pkg/langref:sync",
			len(got), len(want))
	}
}

// TestMarkdownIsTheReference is the sanity check that the embed found the file
// at all: a go:embed of a file that exists but is empty compiles fine.
func TestMarkdownIsTheReference(t *testing.T) {
	if len(Markdown()) < 1000 {
		t.Fatalf("the embedded reference is %d bytes, which cannot be the whole language", len(Markdown()))
	}
}
