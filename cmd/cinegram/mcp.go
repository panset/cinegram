package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tejaspanse/cinegram/pkg/compile"
	"github.com/tejaspanse/cinegram/pkg/emit/html"
	"github.com/tejaspanse/cinegram/pkg/emit/mermaid"
	"github.com/tejaspanse/cinegram/pkg/emit/narrate"
	"github.com/tejaspanse/cinegram/pkg/envelope"
	"github.com/tejaspanse/cinegram/pkg/ir"
	"github.com/tejaspanse/cinegram/pkg/langref"
	"github.com/tejaspanse/cinegram/pkg/loader"
	"github.com/tejaspanse/cinegram/pkg/mcp"
	"github.com/tejaspanse/cinegram/pkg/parser"
	"github.com/tejaspanse/cinegram/pkg/units"
)

// `cinegram mcp` is the same commands an agent already drives, offered down a
// pipe instead of through a shell.
//
// Nothing here is new capability: every tool is one subcommand's code path,
// called in-process. What changes is the shape of the conversation. A model
// running `cinegram frame … -o /tmp/x.png` has to be told where to put the
// file, remember the path, and be able to read a PNG off disk; the same model
// calling the `frame` tool gets the image back in the reply. The tools that
// return data return exactly the bytes the CLI would have written to stdout, so
// a host that already parses `lint --format=json` parses this unchanged.
//
// The CLI stays primary. This is an alternative front door for hosts that speak
// MCP, not a replacement, and every tool here is deliberately a thin wrapper so
// the two can never disagree about what a document means.

// mcpDefaultName is what an inline source is called when the caller does not
// say. Loading needs a name even with no file: relative `view … from` and
// storyboard `img` paths resolve against it, and compileSource refuses an empty
// one rather than silently resolving against the process's working directory.
const mcpDefaultName = "inline.dgm"

// cmdMCP serves the protocol on stdin/stdout until the host closes the pipe.
//
// stdout belongs entirely to the protocol from here on, which is why every
// handler below runs its subcommand code against a bytes.Buffer standing in for
// stderr: a diagnostic printed the usual way would land in the middle of a
// JSON-RPC message and end the session.
func cmdMCP(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) > 0 {
		return fmt.Errorf("mcp takes no arguments, got %q", args[0])
	}
	return mcpServer().Serve(stdin, stdout)
}

// mcpServer is the whole surface: five tools named after the subcommands they
// are, and the language reference as a resource.
//
// v1 returns content blocks only — no outputSchema, no structuredContent. Both
// are additive in the protocol (a client that wants structure reads the schema
// and the extra member; one that does not sees the same blocks it sees now), so
// they can be added later without moving anything a caller depends on.
func mcpServer() *mcp.Server {
	const chromeNote = " Needs a headless Chrome or Chromium: found on PATH, or named by $CINEGRAM_CHROME. " +
		"Without one this call fails and says so; prefer `narrate` or `lint` when no browser is available."

	return &mcp.Server{
		Name:    "cinegram",
		Title:   "Cinegram",
		Version: version,
		Tools: []mcp.Tool{
			{
				Name:        "lint",
				Title:       "Lint a diagram",
				Description: "Check a .dgm document and return every diagnostic as JSON: file, line, col, severity, message, hint, and — when the diagnostic carries a machine-applicable correction — a fix of {line, col, old, new}. An empty array means the document is clean.",
				InputSchema: mcpSchema(),
				Handler:     mcpLint,
			},
			{
				Name:        "narrate",
				Title:       "Narrate a diagram",
				Description: "Read the animation back as prose: every step in order, with its timing and each track stated as a sentence. This is how to check what a document actually animates without looking at it. format=md (default) or json.",
				InputSchema: mcpSchema(`"format":{"type":"string","enum":["md","json"],"description":"md for a written walkthrough (default), json for the same walkthrough as data"}`),
				Handler:     mcpNarrate,
			},
			{
				Name:        "mermaid",
				Title:       "Extract the Mermaid diagram",
				Description: "Return the diagram half of a .dgm document as plain Mermaid, with the animation blocks stripped. Use it to hand the diagram to something that renders Mermaid and nothing else.",
				InputSchema: mcpSchema(),
				Handler:     mcpMermaid,
			},
			{
				Name:        "frame",
				Title:       "Screenshot one moment",
				Description: "Photograph the animation at one exact moment and return the PNG. The deep link lands paused, so the image is deterministic rather than a race against the clock." + chromeNote,
				InputSchema: mcpSchema(
					`"at":{"type":"string","description":"the moment to capture, e.g. 2400ms or 2.4s (default: 0)"}`,
					`"scenario":{"type":"string","description":"scenario id or name (default: the first)"}`,
					`"view":{"type":"string","description":"view id (default: the one the document opens on)"}`,
					`"width":{"type":"integer","description":"viewport width (default: 1400)"}`,
					`"height":{"type":"integer","description":"viewport height (default: 900)"}`,
				),
				Handler: mcpFrame,
			},
			{
				Name:        "sheet",
				Title:       "Contact sheet of a scenario",
				Description: "Photograph every step of one scenario into a single labelled grid and return it, together with the manifest that says which rectangle is which step and what moment it shows. One image read checks a whole animation." + chromeNote,
				InputSchema: mcpSchema(
					`"scenario":{"type":"string","description":"scenario id or name (default: the first)"}`,
					`"view":{"type":"string","description":"view id (default: the one the document opens on)"}`,
					`"cols":{"type":"integer","description":"columns in the grid (default: chosen from the step count, at most 4)"}`,
					`"width":{"type":"integer","description":"one cell's width (default: 900)"}`,
					`"height":{"type":"integer","description":"one cell's height (default: 600)"}`,
				),
				Handler: mcpSheet,
			},
		},
		Resources: []mcp.Resource{{
			URI:         "cinegram://reference/language.md",
			Name:        "language.md",
			Title:       "The .dgm language",
			Description: "The complete authoring reference for .dgm: the block structure, every action and attribute, and the rules each diagram type follows. Read this before writing .dgm content.",
			MIMEType:    "text/markdown",
			Read:        func() (string, error) { return langref.Markdown(), nil },
		}},
	}
}

// mcpSchema builds one tool's input schema around the source pair every tool
// shares.
//
// additionalProperties is false so a client that invented a field is told
// rather than silently ignored, and the XOR between `path` and `source` is
// described in prose rather than expressed as a JSON Schema `oneOf`: client
// support for the keyword is uneven, and a rule only half the clients enforce
// still has to be enforced in the handler. It is enforced there, once.
func mcpSchema(extra ...string) json.RawMessage {
	props := []string{
		`"path":{"type":"string","description":"path to a .dgm file on disk. Give exactly one of path or source."}`,
		`"source":{"type":"string","description":"the .dgm document itself, for a draft that is not on disk yet. Give exactly one of path or source."}`,
		`"as":{"type":"string","description":"with source, the filename the document should be treated as having, so relative view and storyboard paths resolve (default: ` + mcpDefaultName + `)"}`,
	}
	return json.RawMessage(`{"type":"object","properties":{` +
		strings.Join(append(props, extra...), ",") +
		`},"additionalProperties":false}`)
}

// mcpInput is every argument any of the five tools takes. One struct rather
// than five because the schemas are what tell a client which fields a given
// tool reads, and a field a tool ignores costs nothing.
type mcpInput struct {
	Path   string `json:"path"`
	Source string `json:"source"`
	As     string `json:"as"`

	Format   string `json:"format"`
	At       string `json:"at"`
	Scenario string `json:"scenario"`
	View     string `json:"view"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Cols     int    `json:"cols"`
}

// decodeInput rejects what the schema already forbade, so a client that ignored
// additionalProperties gets the same answer as one that read it.
func decodeInput(args json.RawMessage) (mcpInput, error) {
	var in mcpInput
	dec := json.NewDecoder(bytes.NewReader(args))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return in, fmt.Errorf("bad arguments: %w", err)
	}
	return in, nil
}

// entry resolves the source pair into what the loader wants: a path to treat as
// the entry document, and the function that reads it.
//
// It is compileSource's stdin case reused verbatim — an in-memory document is
// exactly the problem `compile - --as <path>` already solved, and solving it
// twice is how the two would come to disagree about which paths resolve where.
func (in mcpInput) entry() (string, loader.ReadFileFunc, error) {
	switch {
	case in.Path != "" && in.Source != "":
		return "", nil, fmt.Errorf("give either path or source, not both")
	case in.Path != "":
		if in.As != "" {
			return "", nil, fmt.Errorf("`as` names an inline source; a document read from path already has a name")
		}
		return compileSource(resolvePath(in.Path), "", nil)
	case in.Source != "":
		name := in.As
		if name == "" {
			name = mcpDefaultName
		}
		return compileSource("-", name, strings.NewReader(in.Source))
	default:
		return "", nil, fmt.Errorf("give one of path (a .dgm file on disk) or source (the document itself)")
	}
}

// mcpLoad is the load-and-compile half every tool but `mermaid` starts with.
//
// It compiles as well as loads for the reason `lint` does: some problems — a
// bad duration on an action that validation skipped — only surface during the
// timing pass. The timeline comes back alongside the bundle so that a caller
// wanting both does not compile twice for it.
func mcpLoad(in mcpInput) (*loader.Bundle, *ir.Timeline, error) {
	entry, read, err := in.entry()
	if err != nil {
		return nil, nil, err
	}
	bundle, err := loader.Load(entry, read)
	if err != nil {
		return nil, nil, err
	}
	return bundle, compile.CompileBundle(bundle), nil
}

// mcpFold turns a subcommand's error and whatever it printed on the way into
// one message the model can act on. The diagnostics come first because they say
// what is wrong; the error says only that something was.
func mcpFold(diags *bytes.Buffer, err error) mcp.Result {
	text := strings.TrimRight(diags.String(), "\n")
	if text == "" {
		return mcp.Errorf("%v", err)
	}
	return mcp.Errorf("%s\n%v", text, err)
}

// mcpLint returns the same array `lint --format=json` writes to stdout.
//
// Diagnostics are the payload here, not a failure: a document with three errors
// is a successful lint. IsError is reserved for the one case where there is no
// array to return at all — the entry document could not be read, or the
// arguments did not name one.
func mcpLint(args json.RawMessage) mcp.Result {
	in, err := decodeInput(args)
	if err != nil {
		return mcp.Errorf("%v", err)
	}
	bundle, _, err := mcpLoad(in)
	if err != nil {
		return mcp.Errorf("%v", err)
	}

	out, _ := envelope.Collect(bundle.Bags())
	encoded, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return mcp.Errorf("%v", err)
	}
	return mcp.Result{Content: []mcp.Content{mcp.Text(string(encoded))}}
}

func mcpNarrate(args json.RawMessage) mcp.Result {
	in, err := decodeInput(args)
	if err != nil {
		return mcp.Errorf("%v", err)
	}
	format := in.Format
	if format == "" {
		format = "md"
	}
	if format != "md" && format != "json" {
		return mcp.Errorf("unknown format %q: use md or json", format)
	}

	bundle, timeline, err := mcpLoad(in)
	if err != nil {
		return mcp.Errorf("%v", err)
	}
	var diags bytes.Buffer
	if err := reportAll(bundle.Bags(), &diags); err != nil {
		return mcpFold(&diags, err)
	}

	doc := narrate.Build(timeline, html.DefaultTitle(timeline))
	if format == "json" {
		encoded, err := narrate.JSON(doc)
		if err != nil {
			return mcp.Errorf("%v", err)
		}
		return mcp.Result{Content: []mcp.Content{mcp.Text(string(encoded))}}
	}
	return mcp.Result{Content: []mcp.Content{mcp.Text(string(narrate.Markdown(doc)))}}
}

// mcpMermaid parses only, exactly as cmdMermaid does: the diagram half of a
// document is whatever the parser round-tripped, and pulling in the views a
// `view … from` names would answer a question nobody asked.
func mcpMermaid(args json.RawMessage) mcp.Result {
	in, err := decodeInput(args)
	if err != nil {
		return mcp.Errorf("%v", err)
	}
	entry, read, err := in.entry()
	if err != nil {
		return mcp.Errorf("%v", err)
	}
	content, err := read(entry)
	if err != nil {
		return mcp.Errorf("%v", err)
	}

	res, bag := parser.Parse(filepath.Base(entry), string(content))
	var diags bytes.Buffer
	if err := report(bag, &diags); err != nil {
		return mcpFold(&diags, err)
	}
	return mcp.Result{Content: []mcp.Content{mcp.Text(mermaid.Emit(res.Document))}}
}

// mcpFrame returns the PNG itself rather than a path to one.
//
// That is the whole reason a frame tool is worth having: a model calling the
// CLI has to invent a filename, remember it, and be able to read a binary file
// back; a model calling this looks at the picture in the reply.
func mcpFrame(args json.RawMessage) mcp.Result {
	in, err := decodeInput(args)
	if err != nil {
		return mcp.Errorf("%v", err)
	}
	// Asked for first, so that the answer to "is there a browser" is the whole
	// message when there is not — it names the environment variable that fixes
	// it, and nothing else here would.
	if _, err := findChrome(); err != nil {
		return mcp.Errorf("%v", err)
	}

	at := in.At
	if at == "" {
		at = "0"
	}
	ms, err := units.ParseMillis(at)
	if err != nil {
		return mcp.Errorf("at %q: %v", at, err)
	}

	_, timeline, err := mcpLoad(in)
	if err != nil {
		return mcp.Errorf("%v", err)
	}
	var diags bytes.Buffer
	// Resolved here rather than left to runCapture's own defaulting, because
	// the caption has to name what was actually photographed: "the first
	// scenario" is not an answer a model can use to ask for the next one.
	viewID := in.View
	if viewID == "" {
		viewID = timeline.Root
	}
	scenarioID, _, err := pickScenario(timeline, viewID, in.Scenario)
	if err != nil {
		return mcpFold(&diags, err)
	}

	dir, err := os.MkdirTemp("", "cinegram-mcp-frame-")
	if err != nil {
		return mcp.Errorf("%v", err)
	}
	defer os.RemoveAll(dir)
	out := filepath.Join(dir, "frame.png")

	entry, read, err := in.entry()
	if err != nil {
		return mcp.Errorf("%v", err)
	}
	opt := captureOptions{
		input: entry, output: out, at: ms, frames: 1,
		scenario: scenarioID, view: viewID,
		width: mcpOr(in.Width, 1400), height: mcpOr(in.Height, 900),
		read: read,
	}
	if err := runCapture(opt, &diags); err != nil {
		return mcpFold(&diags, err)
	}

	image, err := mcpPNG(out)
	if err != nil {
		return mcp.Errorf("%v", err)
	}
	return mcp.Result{Content: []mcp.Content{
		image,
		mcp.Text(fmt.Sprintf("view %s, scenario %s, at %dms", viewID, scenarioID, ms)),
	}}
}

// mcpSheet returns the grid and the map of it together, which is what makes the
// image addressable: a model that sees something wrong in the fourth cell reads
// the step id out of the manifest and re-shoots that one moment with `frame`.
func mcpSheet(args json.RawMessage) mcp.Result {
	in, err := decodeInput(args)
	if err != nil {
		return mcp.Errorf("%v", err)
	}
	if _, err := findChrome(); err != nil {
		return mcp.Errorf("%v", err)
	}

	entry, read, err := in.entry()
	if err != nil {
		return mcp.Errorf("%v", err)
	}
	dir, err := os.MkdirTemp("", "cinegram-mcp-sheet-")
	if err != nil {
		return mcp.Errorf("%v", err)
	}
	defer os.RemoveAll(dir)
	out := filepath.Join(dir, "sheet.png")

	var diags bytes.Buffer
	manifest, err := runSheet(sheetOptions{
		input: entry, output: out,
		scenario: in.Scenario, view: in.View, cols: in.Cols,
		width: mcpOr(in.Width, 900), height: mcpOr(in.Height, 600),
		read: read,
	}, &diags)
	if err != nil {
		return mcpFold(&diags, err)
	}

	image, err := mcpPNG(out)
	if err != nil {
		return mcp.Errorf("%v", err)
	}
	return mcp.Result{Content: []mcp.Content{image, mcp.Text(string(manifest))}}
}

// mcpPNG reads a written screenshot back as a content block. The file is a
// temporary the handler owns: the protocol carries the bytes, so nothing is
// left behind for a caller to clean up or a later run to trip over.
func mcpPNG(path string) (mcp.Content, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return mcp.Content{}, err
	}
	return mcp.Image(base64.StdEncoding.EncodeToString(data), "image/png"), nil
}

// mcpOr defaults a numeric argument, since JSON has no way to say "absent" for
// a number that a Go int can hold apart from the zero the schema forbids.
func mcpOr(v, fallback int) int {
	if v <= 0 {
		return fallback
	}
	return v
}
