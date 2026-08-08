package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"testing"

	"github.com/tejaspanse/diagramator/pkg/diag"
	"github.com/tejaspanse/diagramator/pkg/source"
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
		{"", ""}, // an unset -o must stay empty so output goes to stdout
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
			err := lintJSON(tc.bags, &out)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}

			var got []jsonDiagnostic
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
