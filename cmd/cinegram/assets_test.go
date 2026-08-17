package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tejaspanse/cinegram/pkg/embedkit"
)

func TestAssetsWritesTheWholeKit(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := run([]string{"assets", "-o", dir}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("assets: %v (stderr: %s)", err, stderr.String())
	}
	for name, want := range embedkit.Assets() {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: written bytes differ from the embedded copy", name)
		}
	}
}

// The second run is the one people actually make — the first was a year ago,
// and this one is after an upgrade. It must not fail, and it must not rewrite
// the 2.7 MB of mermaid that did not change.
func TestAssetsIsIdempotentAndReplacesStaleFiles(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := run([]string{"assets", "-o", dir}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("first run: %v", err)
	}

	stale := filepath.Join(dir, "runtime.js")
	if err := os.WriteFile(stale, []byte("// left over from an older cinegram\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(filepath.Join(dir, "mermaid.min.js"))
	if err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := run([]string{"assets", "-o", dir}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if got, _ := os.ReadFile(stale); !bytes.Equal(got, embedkit.Assets()["runtime.js"]) {
		t.Error("a stale runtime.js survived the second run")
	}
	after, err := os.Stat(filepath.Join(dir, "mermaid.min.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Error("mermaid.min.js was rewritten though it had not changed")
	}
	if !strings.Contains(stdout.String(), "1 of 5 files written") {
		t.Errorf("report should name what it did, got %q", stdout.String())
	}
}

func TestAssetsNeedsAnOutputDirectory(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"assets"}, nil, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "-o DIR") {
		t.Errorf("want an error naming -o DIR, got %v", err)
	}
}
