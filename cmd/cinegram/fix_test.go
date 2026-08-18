package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tejaspanse/cinegram/pkg/envelope"
)

// writeTemp puts content in a scratch file and returns its path.
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestApplyFixesRightToLeft is the reason edits are ordered at all: two fixes on
// one line, the first of which changes the line's length. Applied left to right
// the second would land at the wrong column; applied right to left neither
// moves the other, and no offset has to be tracked.
func TestApplyFixesRightToLeft(t *testing.T) {
	path := writeTemp(t, "two.dgm", "    flow client -> ing { colour: red, durr: 300ms }\n")

	applied, skipped, err := applyFixes(path, []envelope.Fix{
		{Line: 1, Col: 26, Old: "colour", New: "color"},
		{Line: 1, Col: 39, Old: "durr", New: "dur"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if applied != 2 || len(skipped) != 0 {
		t.Fatalf("applied %d, skipped %v; want 2 and none", applied, skipped)
	}

	want := "    flow client -> ing { color: red, dur: 300ms }\n"
	if got := readFile(t, path); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

// TestApplyFixesKeepsIndentation pins that a fix is a splice rather than a
// rewrite of the line: leading whitespace is part of how a .dgm reads, and the
// column a diagnostic reports counts through it.
func TestApplyFixesKeepsIndentation(t *testing.T) {
	path := writeTemp(t, "indent.dgm", "\t\t  highlight ingres\n")

	applied, _, err := applyFixes(path, []envelope.Fix{{Line: 1, Col: 15, Old: "ingres", New: "ing"}})
	if err != nil {
		t.Fatal(err)
	}
	if applied != 1 {
		t.Fatalf("applied %d, want 1", applied)
	}
	if got, want := readFile(t, path), "\t\t  highlight ing\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestApplyFixesQuotedValue covers the case the position alone gets wrong: a
// string token is positioned at its opening quote, so the edit spans the quotes
// and swapping only the word between them would leave the file unparseable.
func TestApplyFixesQuotedValue(t *testing.T) {
	path := writeTemp(t, "quoted.dgm", `scenario "no such base" { variant: "bse" }`+"\n")

	applied, skipped, err := applyFixes(path, []envelope.Fix{
		{Line: 1, Col: 36, Old: `"bse"`, New: `"base"`},
	})
	if err != nil {
		t.Fatal(err)
	}
	if applied != 1 || len(skipped) != 0 {
		t.Fatalf("applied %d, skipped %v; want 1 and none", applied, skipped)
	}
	if got, want := readFile(t, path), `scenario "no such base" { variant: "base" }`+"\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestApplyFixesSkipsStaleOld is the safety property the whole feature rests
// on. A fix describes bytes the compiler saw; if the file has moved on — edited
// under the tool, or an escape the parser unquoted that does not reconstruct —
// the edit is reported and abandoned, never guessed at.
func TestApplyFixesSkipsStaleOld(t *testing.T) {
	const original = "  highlight already_fixed\n"
	path := writeTemp(t, "stale.dgm", original)

	applied, skipped, err := applyFixes(path, []envelope.Fix{{Line: 1, Col: 13, Old: "ingres", New: "ing"}})
	if err != nil {
		t.Fatal(err)
	}
	if applied != 0 {
		t.Errorf("applied %d, want 0", applied)
	}
	if len(skipped) != 1 || skipped[0].Old != "ingres" {
		t.Errorf("skipped = %v, want the one stale fix", skipped)
	}
	if got := readFile(t, path); got != original {
		t.Errorf("file was rewritten: %q", got)
	}
}

// TestApplyFixesCRLF checks that a Windows file survives a fix. The parser
// normalises line endings before it counts columns, so the carriage return is
// invisible to a position and must be invisible to the splice too — a run that
// silently converted a file to LF would show up as a whole-file diff.
func TestApplyFixesCRLF(t *testing.T) {
	path := writeTemp(t, "crlf.dgm", "flowchart LR\r\n  highlight ingres\r\n  wait 500ms\r\n")

	applied, skipped, err := applyFixes(path, []envelope.Fix{{Line: 2, Col: 13, Old: "ingres", New: "ing"}})
	if err != nil {
		t.Fatal(err)
	}
	if applied != 1 || len(skipped) != 0 {
		t.Fatalf("applied %d, skipped %v; want 1 and none", applied, skipped)
	}

	want := "flowchart LR\r\n  highlight ing\r\n  wait 500ms\r\n"
	if got := readFile(t, path); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestApplyFixesColPastEndOfLine covers the degenerate input rather than a real
// one: a position that does not exist must be skipped, not panic and not append.
func TestApplyFixesColPastEndOfLine(t *testing.T) {
	const original = "short\n"
	path := writeTemp(t, "past.dgm", original)

	applied, skipped, err := applyFixes(path, []envelope.Fix{
		{Line: 1, Col: 99, Old: "x", New: "y"},
		{Line: 7, Col: 1, Old: "x", New: "y"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if applied != 0 || len(skipped) != 2 {
		t.Fatalf("applied %d, skipped %d; want 0 and 2", applied, len(skipped))
	}
	if got := readFile(t, path); got != original {
		t.Errorf("file was rewritten: %q", got)
	}
}

// TestApplyFixesKeepsFileMode checks that a fix edits a file rather than
// replacing it: an executable or read-group bit set by the author has nothing
// to do with linting and must survive the write.
func TestApplyFixesKeepsFileMode(t *testing.T) {
	path := writeTemp(t, "mode.dgm", "  highlight ingres\n")
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := applyFixes(path, []envelope.Fix{{Line: 1, Col: 13, Old: "ingres", New: "ing"}}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("mode = %o, want 600", got)
	}
}

// lintFixSource is a document with one misspelt reference and nothing else
// wrong, so a clean second lint is proof the fix landed rather than proof the
// diagnostics were merely reshuffled.
const lintFixSource = `flowchart LR
  client[Client]
  ing[Ingress]

  client --> ing

scenario "one hop"

  step go "the request arrives" {
    flow client -> ing
    highlight ingres
  }
`

// TestLintFixRoundTrip drives the command the way a caller does: lint --fix
// repairs the file, and the lint that follows is clean and exits 0. The two
// halves together are the contract — a fix that left the document failing would
// be worse than no fix at all.
func TestLintFixRoundTrip(t *testing.T) {
	t.Setenv("BUILD_WORKING_DIRECTORY", "")
	path := writeTemp(t, "round.dgm", lintFixSource)

	var stdout, stderr bytes.Buffer
	if err := run([]string{"lint", path, "--fix"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("lint --fix failed: %v\n%s", err, stderr.String())
	}

	got := readFile(t, path)
	if strings.Contains(got, "ingres\n") || !strings.Contains(got, "highlight ing\n") {
		t.Fatalf("file was not repaired:\n%s", got)
	}
	if !strings.Contains(stderr.String(), "fixed "+path+":11:15: ingres -> ing") {
		t.Errorf("stderr does not name the edit:\n%s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "applied 1 fix") {
		t.Errorf("stderr has no summary:\n%s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok") {
		t.Errorf("stdout = %q, want the ok line", stdout.String())
	}

	// The second run is the real assertion: nothing left to fix, nothing left
	// to report, and --strict cannot find a warning either.
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"lint", path, "--strict"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("second lint is not clean: %v\n%s", err, stderr.String())
	}
}

// TestLintFixComposesWithJSONAndStrict pins that --fix moves only the file:
// the payload still goes to stdout as an array, the fix log stays on stderr,
// and the exit status is the one the repaired document earns — here 1, because
// a warning survives the fix and --strict was asked for.
func TestLintFixComposesWithJSONAndStrict(t *testing.T) {
	t.Setenv("BUILD_WORKING_DIRECTORY", "")
	path := writeTemp(t, "compose.dgm", lintFixSource+`
scenario "unused" { wobble: 3 }

  step other "nothing wrong here" {
    flow client -> ing
  }
`)

	var stdout, stderr bytes.Buffer
	err := run([]string{"lint", path, "--fix", "--format=json", "--strict"}, nil, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "--strict") {
		t.Fatalf("error = %v, want the --strict warning failure", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(stdout.String()), "[") {
		t.Errorf("stdout is not the JSON array:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "fixed ") {
		t.Errorf("fix log leaked onto stdout:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "fixed ") {
		t.Errorf("fix log is missing from stderr:\n%s", stderr.String())
	}
	if strings.Contains(readFile(t, path), "ingres") {
		t.Error("the error was not fixed")
	}
}
