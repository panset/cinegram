// Command diagramator compiles animated diagram sources into an animation
// timeline, plain Mermaid, or a self-contained animated HTML page.
//
// Usage:
//
//	diagramator compile <file.dgm>   # timeline JSON on stdout
//	diagramator mermaid <file.dgm>   # the diagram as plain Mermaid
//	diagramator preview <file.dgm>   # self-contained animated HTML
//	diagramator narrate <file.dgm>   # the animation as a written walkthrough
//	diagramator lint    <file.dgm>   # diagnostics only
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tejaspanse/diagramator/pkg/compile"
	"github.com/tejaspanse/diagramator/pkg/diag"
	"github.com/tejaspanse/diagramator/pkg/emit/html"
	"github.com/tejaspanse/diagramator/pkg/emit/mermaid"
	"github.com/tejaspanse/diagramator/pkg/emit/narrate"
	"github.com/tejaspanse/diagramator/pkg/ir"
	"github.com/tejaspanse/diagramator/pkg/loader"
	"github.com/tejaspanse/diagramator/pkg/parser"
	"github.com/tejaspanse/diagramator/pkg/units"
)

const version = "0.1.0"

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "diagramator:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		usage(stderr)
		return fmt.Errorf("no command given")
	}

	cmd, rest := args[0], args[1:]
	switch cmd {
	case "compile":
		return cmdCompile(rest, stdout, stderr)
	case "mermaid":
		return cmdMermaid(rest, stdout, stderr)
	case "preview":
		return cmdPreview(rest, stdout, stderr)
	case "lint":
		return cmdLint(rest, stdout, stderr)
	case "narrate":
		return cmdNarrate(rest, stdout, stderr)
	case "frame":
		return cmdFrame(rest, stdout, stderr)
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, "diagramator", version)
		return nil
	case "help", "--help", "-h":
		usage(stdout)
		return nil
	default:
		usage(stderr)
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `diagramator - animated architecture diagrams from a Mermaid-like DSL

Usage:
  diagramator compile <file.dgm> [-o out.json]   compile to an animation timeline
  diagramator mermaid <file.dgm> [-o out.mmd]    emit the diagram as plain Mermaid
  diagramator preview <file.dgm> [-o out.html]   build a self-contained animated page
  diagramator preview <file.dgm> --serve [--watch] [--addr host:port]
                                                 serve it, rebuilding as you edit
  diagramator frame   <file.dgm> --at 2400ms -o out.png
                                                 screenshot one exact moment
                                                 (--frames N -o dir/ for a sequence)
  diagramator narrate <file.dgm> [-o out.md] [--format=md|json]
                                                 the animation as a walkthrough
  diagramator lint    <file.dgm> [--format=text|json]
                                                 report diagnostics only
  diagramator version

Warnings never fail a build; errors do.
`)
}

// load parses a source file and reports diagnostics. It returns an error only
// when parsing produced errors, so warnings still let the caller proceed.
func load(path string, stderr io.Writer) (*parser.Result, *diag.Bag, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	res, bag := parser.Parse(filepath.Base(path), string(content))
	return res, bag, nil
}

func report(bag *diag.Bag, stderr io.Writer) error {
	return reportAll([]*diag.Bag{bag}, stderr)
}

// reportAll prints the diagnostics of every file in a bundle. Each bag is
// labelled with its own filename, so a problem in a drilled-into diagram is
// attributable to the file that actually contains it.
func reportAll(bags []*diag.Bag, stderr io.Writer) error {
	errs := 0
	for _, bag := range bags {
		if bag.Len() > 0 {
			fmt.Fprintln(stderr, bag)
		}
		errs += countErrors(bag)
	}
	if errs > 0 {
		return fmt.Errorf("%s", plural(errs, "error"))
	}
	return nil
}

func countErrors(bag *diag.Bag) int {
	n := 0
	for _, d := range bag.All() {
		if d.Severity == diag.SeverityError {
			n++
		}
	}
	return n
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// parseArgs pulls the single input path and an optional -o flag off argv.
//
// Flags are hoisted ahead of positional arguments first, because the flag
// package stops parsing at the first non-flag word and `preview file.dgm -o
// out.html` is the order people naturally type.
func parseArgs(name string, args []string) (input, output string, err error) {
	return parseArgsWith(name, args, nil)
}

// parseArgsWith is parseArgs with room for a command's own flags. `extra`
// registers them against the same set, so they hoist and parse identically.
func parseArgsWith(name string, args []string, extra func(*flag.FlagSet)) (input, output string, err error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.StringVar(&output, "o", "", "write output to this file instead of stdout")
	if extra != nil {
		extra(fs)
	}
	if err = fs.Parse(hoistFlags(args, valueFlagNames(fs))); err != nil {
		return "", "", err
	}
	switch fs.NArg() {
	case 1:
		return resolvePath(fs.Arg(0)), resolvePath(output), nil
	case 0:
		return "", "", fmt.Errorf("%s needs an input file", name)
	default:
		return "", "", fmt.Errorf("%s takes one input file, got %d", name, fs.NArg())
	}
}

// resolvePath interprets a relative path against the directory the user was
// actually standing in.
//
// `bazel run` executes the binary from its runfiles tree, not the caller's
// working directory, so a relative path typed on the command line would
// otherwise resolve against the wrong root. Bazel exports
// BUILD_WORKING_DIRECTORY for exactly this case; outside Bazel it is unset and
// the path is left alone.
func resolvePath(p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	if wd := os.Getenv("BUILD_WORKING_DIRECTORY"); wd != "" {
		return filepath.Join(wd, p)
	}
	return p
}

// valueFlagNames reads which flags consume the following argument straight off
// the registered set, so a new flag can never be forgotten here: a flag is a
// value flag unless its Value says it is boolean, which is the same test
// flag.Parse itself applies.
func valueFlagNames(fs *flag.FlagSet) map[string]bool {
	type boolFlag interface{ IsBoolFlag() bool }
	m := map[string]bool{}
	fs.VisitAll(func(f *flag.Flag) {
		if b, ok := f.Value.(boolFlag); ok && b.IsBoolFlag() {
			return
		}
		m["-"+f.Name] = true
		m["--"+f.Name] = true
	})
	return m
}

func hoistFlags(args []string, valueFlags map[string]bool) []string {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) > 1 && strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			if valueFlags[a] && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		positional = append(positional, a)
	}
	return append(flags, positional...)
}

func write(path string, data []byte, stdout io.Writer) error {
	if path == "" {
		_, err := stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func cmdCompile(args []string, stdout, stderr io.Writer) error {
	input, output, err := parseArgs("compile", args)
	if err != nil {
		return err
	}
	bundle, err := loader.Load(input, os.ReadFile)
	if err != nil {
		return err
	}
	timeline := compile.CompileBundle(bundle)
	if err := reportAll(bundle.Bags(), stderr); err != nil {
		return err
	}

	encoded, err := json.MarshalIndent(timeline, "", "  ")
	if err != nil {
		return err
	}
	return write(output, append(encoded, '\n'), stdout)
}

func cmdMermaid(args []string, stdout, stderr io.Writer) error {
	input, output, err := parseArgs("mermaid", args)
	if err != nil {
		return err
	}
	res, bag, err := load(input, stderr)
	if err != nil {
		return err
	}
	if err := report(bag, stderr); err != nil {
		return err
	}
	return write(output, []byte(mermaid.Emit(res.Document)), stdout)
}

func cmdPreview(args []string, stdout, stderr io.Writer) error {
	var addr string
	var serve, watch bool
	input, output, err := parseArgsWith("preview", args, func(fs *flag.FlagSet) {
		fs.BoolVar(&serve, "serve", false, "serve the page over HTTP instead of writing a file")
		fs.StringVar(&addr, "addr", defaultAddr, "address to serve on")
		fs.BoolVar(&watch, "watch", false, "rebuild and reload when the source changes")
	})
	if err != nil {
		return err
	}

	// --watch on its own plainly means "serve and watch": there is nothing to
	// watch for when the output is a file written once.
	if serve || watch {
		if output != "" {
			fmt.Fprintln(stderr, "diagramator: warning: -o is ignored with --serve/--watch; the page is served, not written")
		}
		return runServe(input, addr, watch, stderr)
	}

	if output == "" {
		output = defaultOutputPath(input)
	}

	bundle, err := loader.Load(input, os.ReadFile)
	if err != nil {
		return err
	}
	timeline := compile.CompileBundle(bundle)
	if err := reportAll(bundle.Bags(), stderr); err != nil {
		return err
	}
	if rootHasNoScenarios(timeline) {
		fmt.Fprintln(stderr, "diagramator: warning: no scenarios, the page will render a static diagram")
	}

	page, err := html.Render(timeline, html.Options{})
	if err != nil {
		return err
	}
	if err := os.WriteFile(output, page, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "wrote %s (%d KB)\n", output, len(page)/1024)
	return nil
}

// defaultOutputPath is where `preview` writes when given no -o: beside the
// input, with the extension swapped for .html.
func defaultOutputPath(input string) string {
	return strings.TrimSuffix(input, filepath.Ext(input)) + ".html"
}

func rootView(t *ir.Timeline) *ir.View {
	for i := range t.Views {
		if t.Views[i].ID == t.Root {
			return &t.Views[i]
		}
	}
	return nil
}

func rootHasNoScenarios(t *ir.Timeline) bool {
	v := rootView(t)
	return v == nil || len(v.Scenarios) == 0
}

func cmdLint(args []string, stdout, stderr io.Writer) error {
	var format string
	input, _, err := parseArgsWith("lint", args, func(fs *flag.FlagSet) {
		fs.StringVar(&format, "format", "text", "text or json")
	})
	if err != nil {
		return err
	}
	// Lint follows view references too, so a `from` path that points at
	// nothing is caught here rather than at preview time.
	bundle, err := loader.Load(input, os.ReadFile)
	if err != nil {
		return err
	}
	// Compile as well: some problems (a bad duration on an action that
	// validation skipped) only surface during the timing pass.
	compile.CompileBundle(bundle)

	if format == "json" {
		return lintJSON(bundle.Bags(), stdout)
	}
	if format != "text" {
		return fmt.Errorf("unknown --format %q: use text or json", format)
	}
	if err := reportAll(bundle.Bags(), stderr); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "%s: ok\n", input)
	return nil
}

// jsonDiagnostic is the machine-readable shape of a diagnostic. It is declared
// here rather than in pkg/diag because it is a wire format: the fields are
// flat, the severity is a word rather than an enum, and it should not move when
// the internal type does.
type jsonDiagnostic struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Col      int    `json:"col"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
}

// lintJSON writes every diagnostic in the bundle as one array on stdout.
//
// Exit-code semantics are unchanged — warnings 0, errors 1 — so a caller can
// branch on the status and read the detail, rather than having to choose.
func lintJSON(bags []*diag.Bag, stdout io.Writer) error {
	out := []jsonDiagnostic{}
	errs := 0

	for _, bag := range bags {
		for _, d := range bag.All() {
			if d.Severity == diag.SeverityError {
				errs++
			}
			out = append(out, jsonDiagnostic{
				File:     bag.Filename,
				Line:     d.Pos.Line,
				Col:      d.Pos.Col,
				Severity: d.Severity.String(),
				Message:  d.Msg,
				Hint:     d.Hint,
			})
		}
	}

	encoded, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	if _, err := stdout.Write(append(encoded, '\n')); err != nil {
		return err
	}
	if errs > 0 {
		return fmt.Errorf("%s", plural(errs, "error"))
	}
	return nil
}

func cmdFrame(args []string, stdout, stderr io.Writer) error {
	var at, scenario, view string
	var frames, width, height int
	input, output, err := parseArgsWith("frame", args, func(fs *flag.FlagSet) {
		fs.StringVar(&at, "at", "0", "the moment to capture, e.g. 2400ms or 2.4s")
		fs.IntVar(&frames, "frames", 1, "capture N evenly spaced moments into -o as a directory")
		fs.StringVar(&scenario, "scenario", "", "scenario id or name (default: the first)")
		fs.StringVar(&view, "view", "", "view id (default: the one the document opens on)")
		fs.IntVar(&width, "width", 1400, "viewport width")
		fs.IntVar(&height, "height", 900, "viewport height")
	})
	if err != nil {
		return err
	}

	ms, err := units.ParseMillis(at)
	if err != nil {
		return fmt.Errorf("--at %q: %w", at, err)
	}

	return runCapture(captureOptions{
		input: input, output: output, at: ms, frames: frames,
		scenario: scenario, view: view, width: width, height: height,
	}, stderr)
}

func cmdNarrate(args []string, stdout, stderr io.Writer) error {
	var format string
	input, output, err := parseArgsWith("narrate", args, func(fs *flag.FlagSet) {
		fs.StringVar(&format, "format", "md", "md or json")
	})
	if err != nil {
		return err
	}
	bundle, err := loader.Load(input, os.ReadFile)
	if err != nil {
		return err
	}
	timeline := compile.CompileBundle(bundle)
	if err := reportAll(bundle.Bags(), stderr); err != nil {
		return err
	}

	doc := narrate.Build(timeline, html.DefaultTitle(timeline))
	switch format {
	case "md":
		return write(output, narrate.Markdown(doc), stdout)
	case "json":
		encoded, err := narrate.JSON(doc)
		if err != nil {
			return err
		}
		return write(output, encoded, stdout)
	default:
		return fmt.Errorf("unknown --format %q: use md or json", format)
	}
}
