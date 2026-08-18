package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"strings"
	"testing"

	"github.com/tejaspanse/cinegram/pkg/diag"
	"github.com/tejaspanse/cinegram/pkg/envelope"
	"github.com/tejaspanse/cinegram/pkg/source"
)

// TestResolvePathUnderBazelRun covers the case that `bazel run` creates: the
// binary executes from its runfiles tree, so a relative path the user typed
// must be resolved against where they actually ran the command.
func TestResolvePathUnderBazelRun(t *testing.T) {
	t.Setenv("BUILD_WORKING_DIRECTORY", "/home/dev/project")

	tests := []struct{ in, want string }{
		{"examples/k8s.dgm", "/home/dev/project/examples/k8s.dgm"},
		{"out.html", "/home/dev/project/out.html"},
		{"/tmp/absolute.html", "/tmp/absolute.html"},
		{"", ""},   // an unset -o must stay empty so output goes to stdout
		{"-", "-"}, // stdin is a name, not a path to join to anything
	}
	for _, tc := range tests {
		if got := resolvePath(tc.in); got != tc.want {
			t.Errorf("resolvePath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestResolvePathOutsideBazel checks that a plain `go run` or an installed
// binary still treats relative paths as relative to the real cwd.
func TestResolvePathOutsideBazel(t *testing.T) {
	t.Setenv("BUILD_WORKING_DIRECTORY", "")

	for _, in := range []string{"examples/k8s.dgm", "out.html", "/tmp/x.html"} {
		if got := resolvePath(in); got != in {
			t.Errorf("resolvePath(%q) = %q, want it unchanged", in, got)
		}
	}
}

// TestParseArgs covers flag hoisting, since the flag package stops parsing at
// the first positional and `preview file.dgm -o out.html` is the natural order.
func TestParseArgs(t *testing.T) {
	t.Setenv("BUILD_WORKING_DIRECTORY", "")

	tests := []struct {
		name    string
		args    []string
		wantIn  string
		wantOut string
		wantErr bool
	}{
		{name: "flag after positional", args: []string{"a.dgm", "-o", "b.html"}, wantIn: "a.dgm", wantOut: "b.html"},
		{name: "flag before positional", args: []string{"-o", "b.html", "a.dgm"}, wantIn: "a.dgm", wantOut: "b.html"},
		{name: "equals form", args: []string{"a.dgm", "-o=b.html"}, wantIn: "a.dgm", wantOut: "b.html"},
		{name: "no flag", args: []string{"a.dgm"}, wantIn: "a.dgm"},
		{name: "no input", args: []string{"-o", "b.html"}, wantErr: true},
		{name: "two inputs", args: []string{"a.dgm", "b.dgm"}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in, out, err := parseArgs("preview", tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got input=%q output=%q", in, out)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if in != tc.wantIn {
				t.Errorf("input = %q, want %q", in, tc.wantIn)
			}
			if out != tc.wantOut {
				t.Errorf("output = %q, want %q", out, tc.wantOut)
			}
		})
	}
}

// TestDefaultOutputPath pins the rule that `preview` with no -o writes beside
// its input rather than into whatever directory the binary ran from.
func TestDefaultOutputPath(t *testing.T) {
	tests := []struct{ in, want string }{
		{"/home/dev/examples/k8s-request.dgm", "/home/dev/examples/k8s-request.html"},
		{"/home/dev/no-extension", "/home/dev/no-extension.html"},
		{"/home/dev/dotted.name.dgm", "/home/dev/dotted.name.html"},
	}
	for _, tc := range tests {
		if got := defaultOutputPath(tc.in); got != tc.want {
			t.Errorf("defaultOutputPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestParseArgsWithFormat covers a flag a command adds for itself, including
// the `--format json` spelling that has to survive being hoisted past the
// positional argument.
func TestParseArgsWithFormat(t *testing.T) {
	t.Setenv("BUILD_WORKING_DIRECTORY", "")

	tests := []struct {
		name       string
		args       []string
		wantIn     string
		wantFormat string
	}{
		{name: "equals form", args: []string{"a.dgm", "--format=json"}, wantIn: "a.dgm", wantFormat: "json"},
		{name: "separate value", args: []string{"a.dgm", "--format", "json"}, wantIn: "a.dgm", wantFormat: "json"},
		{name: "single dash", args: []string{"-format", "json", "a.dgm"}, wantIn: "a.dgm", wantFormat: "json"},
		{name: "default", args: []string{"a.dgm"}, wantIn: "a.dgm", wantFormat: "md"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var format string
			in, _, err := parseArgsWith("narrate", tc.args, func(fs *flag.FlagSet) {
				fs.StringVar(&format, "format", "md", "")
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if in != tc.wantIn {
				t.Errorf("input = %q, want %q", in, tc.wantIn)
			}
			if format != tc.wantFormat {
				t.Errorf("format = %q, want %q", format, tc.wantFormat)
			}
		})
	}
}

// TestCompileSourceOverlay covers the seam a ```dgm block in a Markdown file
// depends on: the source arrives on stdin with no path of its own, but the
// relative paths inside it still have to resolve against the file the block
// lives in.
func TestCompileSourceOverlay(t *testing.T) {
	const src = "flowchart LR\n  a[A]\n"

	entry, read, err := compileSource("-", "docs/guide.md", strings.NewReader(src))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry != "docs/guide.md" {
		t.Errorf("entry = %q, want the --as path", entry)
	}

	// The entry itself comes from stdin...
	got, err := read(entry)
	if err != nil {
		t.Fatalf("reading the entry: %v", err)
	}
	if string(got) != src {
		t.Errorf("entry content = %q, want %q", got, src)
	}

	// ...while anything it references falls through to the real filesystem,
	// which is what makes `view … from "pod-a.dgm"` resolve beside the .md.
	if _, err := read("docs/definitely-not-here.dgm"); err == nil {
		t.Error("a reference that is not the entry should have hit the filesystem and failed")
	}
}

// TestCompileSourceRejectsMismatchedFlags keeps the two ways of naming a source
// from being combined into something with no meaning.
func TestCompileSourceRejectsMismatchedFlags(t *testing.T) {
	if _, _, err := compileSource("-", "", strings.NewReader("")); err == nil {
		t.Error("stdin with no --as should be refused: there is nothing to resolve against")
	}
	if _, _, err := compileSource("a.dgm", "b.md", strings.NewReader("")); err == nil {
		t.Error("--as alongside a real input file should be refused")
	}

	// A plain file still reads straight off disk.
	entry, read, err := compileSource("a.dgm", "", strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry != "a.dgm" {
		t.Errorf("entry = %q, want %q", entry, "a.dgm")
	}
	if read == nil {
		t.Error("read function is nil")
	}
}

// TestEnvelopeAlwaysCarriesBothHalves pins the contract the VS Code preview
// renders against: one JSON shape, both fields always present, so the host
// branches on the data rather than on an exit status it cannot see.
func TestEnvelopeAlwaysCarriesBothHalves(t *testing.T) {
	failed := diag.NewBag("guide.md")
	failed.ErrorHintf(source.Pos{Line: 9, Col: 15}, "known nodes: a, b", "%q is not a node in this diagram", "nope")

	diags, errs := envelope.Collect([]*diag.Bag{failed})
	if errs != 1 {
		t.Fatalf("error count = %d, want 1", errs)
	}

	var out bytes.Buffer
	// A nil timeline is the unreadable-entry case, and must still produce a
	// document the host can parse rather than an empty body.
	if err := writeEnvelope(nil, diags, "", &out); err != nil {
		t.Fatalf("writeEnvelope: %v", err)
	}

	var got envelope.Envelope
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out.String())
	}
	if len(got.Diagnostics) != 1 {
		t.Fatalf("got %d diagnostics, want 1", len(got.Diagnostics))
	}
	if got.Diagnostics[0].Hint == "" {
		t.Error("the hint was dropped; it is most of what makes the message useful")
	}
	if got.Diagnostics[0].File != "guide.md" {
		t.Errorf("file = %q, want the containing document", got.Diagnostics[0].File)
	}

	// An empty bundle still serialises `[]`, never `null`, so a host can
	// iterate the field without checking it first.
	out.Reset()
	if err := writeEnvelope(nil, mustCollect(t), "", &out); err != nil {
		t.Fatalf("writeEnvelope: %v", err)
	}
	if !strings.Contains(out.String(), `"diagnostics": []`) {
		t.Errorf("clean run should emit an empty array, got:\n%s", out.String())
	}
}

func mustCollect(t *testing.T) []envelope.Diagnostic {
	t.Helper()
	diags, errs := envelope.Collect([]*diag.Bag{diag.NewBag("ok.dgm")})
	if errs != 0 {
		t.Fatalf("clean bag reported %d errors", errs)
	}
	return diags
}

// TestLintJSONExitCodes pins the contract a caller scripts against: the payload
// is always valid JSON on stdout, and the exit status still separates warnings
// from errors rather than that distinction being folded into the data.
func TestLintJSONExitCodes(t *testing.T) {
	warned := diag.NewBag("warn.dgm")
	warned.WarnHintf(source.Pos{Line: 3, Col: 5}, "a hint", "a warning")

	failed := diag.NewBag("bad.dgm")
	failed.Errorf(source.Pos{Line: 1, Col: 1}, "an error")

	tests := []struct {
		name    string
		bags    []*diag.Bag
		wantErr bool
		wantLen int
	}{
		{name: "clean", bags: []*diag.Bag{diag.NewBag("ok.dgm")}, wantLen: 0},
		{name: "warnings only", bags: []*diag.Bag{warned}, wantLen: 1},
		{name: "errors", bags: []*diag.Bag{failed}, wantErr: true, wantLen: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := lintJSON(tc.bags, &out, false)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}

			var got []envelope.Diagnostic
			if err := json.Unmarshal(out.Bytes(), &got); err != nil {
				t.Fatalf("output is not valid JSON: %v\n%s", err, out.String())
			}
			if len(got) != tc.wantLen {
				t.Fatalf("got %d diagnostics, want %d", len(got), tc.wantLen)
			}
			if tc.wantLen > 0 && got[0].File == "" {
				t.Error("diagnostic is missing its filename")
			}
		})
	}
}

// TestLintStrictExitCodes pins the only thing --strict is allowed to move: the
// exit status. The JSON payload is the same document either way — a script that
// already reads it does not have to be re-taught when strictness is turned on —
// and an error still claims the failure on its own terms, so the message a
// build already prints does not change shape because --strict was passed.
func TestLintStrictExitCodes(t *testing.T) {
	warned := diag.NewBag("warn.dgm")
	warned.WarnHintf(source.Pos{Line: 3, Col: 5}, "a hint", "a warning")

	failed := diag.NewBag("bad.dgm")
	failed.Errorf(source.Pos{Line: 1, Col: 1}, "an error")

	tests := []struct {
		name      string
		bags      []*diag.Bag
		strict    bool
		wantErr   string // "" means the run must succeed
		wantLen   int
		payloadOf string // cases sharing a key must produce identical bytes
	}{
		{name: "clean", bags: []*diag.Bag{diag.NewBag("ok.dgm")}, payloadOf: "clean"},
		{name: "clean strict", bags: []*diag.Bag{diag.NewBag("ok.dgm")}, strict: true, payloadOf: "clean"},
		{name: "warnings only", bags: []*diag.Bag{warned}, wantLen: 1, payloadOf: "warn"},
		{name: "warnings only strict", bags: []*diag.Bag{warned}, strict: true, wantErr: "1 warning (--strict)", wantLen: 1, payloadOf: "warn"},
		{name: "errors", bags: []*diag.Bag{failed}, wantErr: "1 error", wantLen: 1, payloadOf: "error"},
		{name: "errors strict", bags: []*diag.Bag{failed}, strict: true, wantErr: "1 error", wantLen: 1, payloadOf: "error"},
	}

	payloads := map[string]string{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := lintJSON(tc.bags, &out, tc.strict)
			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("error = %v, want success", err)
			case tc.wantErr != "" && err == nil:
				t.Fatalf("no error, want %q", tc.wantErr)
			case tc.wantErr != "" && err.Error() != tc.wantErr:
				t.Fatalf("error = %q, want %q", err, tc.wantErr)
			}

			var got []envelope.Diagnostic
			if err := json.Unmarshal(out.Bytes(), &got); err != nil {
				t.Fatalf("output is not valid JSON: %v\n%s", err, out.String())
			}
			if len(got) != tc.wantLen {
				t.Fatalf("got %d diagnostics, want %d", len(got), tc.wantLen)
			}

			if prev, seen := payloads[tc.payloadOf]; seen && prev != out.String() {
				t.Errorf("--strict changed the payload:\n%s\nwant:\n%s", out.String(), prev)
			}
			payloads[tc.payloadOf] = out.String()
		})
	}
}

// TestReportStrict checks that strictness is an exit status and nothing else:
// the author reads exactly the same diagnostics on stderr either way.
func TestReportStrict(t *testing.T) {
	warned := diag.NewBag("warn.dgm")
	warned.WarnHintf(source.Pos{Line: 3, Col: 5}, "a hint", "a warning")

	mixed := diag.NewBag("mixed.dgm")
	mixed.Errorf(source.Pos{Line: 1, Col: 1}, "an error")
	mixed.Warnf(source.Pos{Line: 2, Col: 2}, "a warning")

	tests := []struct {
		name          string
		bags          []*diag.Bag
		wantErr       string // without --strict
		wantStrictErr string // with it
	}{
		{name: "clean", bags: []*diag.Bag{diag.NewBag("ok.dgm")}},
		{name: "one warning", bags: []*diag.Bag{warned}, wantStrictErr: "1 warning (--strict)"},
		{name: "two warnings", bags: []*diag.Bag{warned, warned}, wantStrictErr: "2 warnings (--strict)"},
		// An error already fails the build, and says so in the words it
		// always used: --strict adds a reason to fail, not a second message.
		{name: "error wins", bags: []*diag.Bag{mixed}, wantErr: "1 error", wantStrictErr: "1 error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var lenient, strict bytes.Buffer
			gotErr := reportStrict(tc.bags, &lenient, false)
			gotStrictErr := reportStrict(tc.bags, &strict, true)

			if errText(gotErr) != tc.wantErr {
				t.Errorf("error = %q, want %q", errText(gotErr), tc.wantErr)
			}
			if errText(gotStrictErr) != tc.wantStrictErr {
				t.Errorf("--strict error = %q, want %q", errText(gotStrictErr), tc.wantStrictErr)
			}
			if lenient.String() != strict.String() {
				t.Errorf("--strict changed stderr:\n%s\nwant:\n%s", strict.String(), lenient.String())
			}
			// reportAll is the default path six callers still use; it must
			// stay exactly the lenient one.
			var viaReportAll bytes.Buffer
			if got := errText(reportAll(tc.bags, &viaReportAll)); got != tc.wantErr {
				t.Errorf("reportAll error = %q, want %q", got, tc.wantErr)
			}
			if viaReportAll.String() != lenient.String() {
				t.Errorf("reportAll stderr diverged from reportStrict(..., false)")
			}
		})
	}
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
