// Command diagramator compiles animated diagram sources into an animation
// timeline, plain Mermaid, or a self-contained animated HTML page.
//
// Usage:
//
//	diagramator compile <file.dgm>   # timeline JSON on stdout
//	diagramator mermaid <file.dgm>   # the diagram as plain Mermaid
//	diagramator preview <file.dgm>   # self-contained animated HTML
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
	"github.com/tejaspanse/diagramator/pkg/ir"
	"github.com/tejaspanse/diagramator/pkg/parser"
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
  diagramator lint    <file.dgm>                 report diagnostics only
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
	if bag.Len() > 0 {
		fmt.Fprintln(stderr, bag)
	}
	if bag.HasErrors() {
		return fmt.Errorf("%s", plural(countErrors(bag), "error"))
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
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.StringVar(&output, "o", "", "write output to this file instead of stdout")
	if err = fs.Parse(hoistFlags(args)); err != nil {
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

// valueFlags are the flags that consume the following argument.
var valueFlags = map[string]bool{"-o": true, "--o": true}

func hoistFlags(args []string) []string {
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
	res, bag, err := load(input, stderr)
	if err != nil {
		return err
	}
	timeline := compile.Compile(res.Document, res.Symbols, bag)
	if err := report(bag, stderr); err != nil {
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
	input, output, err := parseArgs("preview", args)
	if err != nil {
		return err
	}
	if output == "" {
		output = defaultOutputPath(input)
	}

	res, bag, err := load(input, stderr)
	if err != nil {
		return err
	}
	timeline := compile.Compile(res.Document, res.Symbols, bag)
	if err := report(bag, stderr); err != nil {
		return err
	}
	if len(timeline.Scenarios) == 0 {
		fmt.Fprintln(stderr, "diagramator: warning: no scenarios, the page will render a static diagram")
	}

	page, err := html.Render(timeline, html.Options{Title: title(timeline)})
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

func title(t *ir.Timeline) string {
	if len(t.Scenarios) > 0 && t.Scenarios[0].Name != "" {
		return t.Scenarios[0].Name
	}
	return "Diagramator"
}

func cmdLint(args []string, stdout, stderr io.Writer) error {
	input, _, err := parseArgs("lint", args)
	if err != nil {
		return err
	}
	res, bag, err := load(input, stderr)
	if err != nil {
		return err
	}
	// Compile as well: some problems (a bad duration on an action that
	// validation skipped) only surface during the timing pass.
	compile.Compile(res.Document, res.Symbols, bag)
	if err := report(bag, stderr); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "%s: ok\n", input)
	return nil
}
