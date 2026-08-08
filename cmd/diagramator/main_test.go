package main

import "testing"

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
