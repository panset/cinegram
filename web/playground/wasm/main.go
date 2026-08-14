//go:build js && wasm

// Command cinegram_wasm exposes the compiler to a browser tab.
//
// The playground is a static page with no server behind it, so the whole
// pipeline — parser, loader, compile, emit — is compiled to WebAssembly and
// called synchronously from JavaScript. That is possible only because the
// library half does no I/O of its own: parser.Parse takes a string and
// loader.Load takes its read function as an argument, so an in-memory file set
// substitutes for a disk that does not exist here.
//
// Two functions land on the global object:
//
//	cinegramCompile(entry, files)    -> JSON {timeline, diagnostics, title}
//	cinegramRenderHTML(entry, files) -> JSON {html, diagnostics}
//
// `files` is a plain object mapping path to content: a JS string for text
// (a .dgm source) and a Uint8Array for binary (a storyboard image). Nothing is
// base64-encoded on the way in — the loader inlines images to data: URIs from
// raw bytes itself, so a round-trip through base64 would only cost time.
//
// Size tradeoff, deliberate: importing pkg/emit/html embeds mermaid.min.js
// (2.7 MB) and the runtime assets into the .wasm, which is most of its ~15-20 MB.
// That is the price of "Download HTML" producing bytes identical to what
// `cinegram preview -o` writes, rather than a second renderer that drifts.
//
// The compile runs on the main thread. It is milliseconds for real documents;
// moving it to a Worker is the fix if that ever stops being true.
package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"syscall/js"

	"github.com/tejaspanse/cinegram/pkg/compile"
	"github.com/tejaspanse/cinegram/pkg/diag"
	"github.com/tejaspanse/cinegram/pkg/emit/html"
	"github.com/tejaspanse/cinegram/pkg/envelope"
	"github.com/tejaspanse/cinegram/pkg/loader"
)

// compileResult is the envelope plus the name the page would give the document,
// so the playground can title a download without reimplementing DefaultTitle.
type compileResult struct {
	envelope.Envelope
	Title string `json:"title"`
}

// renderResult carries a whole self-contained page as a string, alongside the
// diagnostics found producing it. Like the envelope it has no failure mode: the
// page is emitted even when errors were found.
type renderResult struct {
	HTML        string                `json:"html"`
	Diagnostics []envelope.Diagnostic `json:"diagnostics"`
}

func main() {
	js.Global().Set("cinegramCompile", js.FuncOf(compileFn))
	js.Global().Set("cinegramRenderHTML", js.FuncOf(renderFn))

	// The page cannot know when instantiation finished from the fetch alone —
	// the exports above are set by this function, not by the loader — so it
	// leaves a callback behind for us to fire.
	if ready := js.Global().Get("onCinegramReady"); ready.Type() == js.TypeFunction {
		ready.Invoke()
	}

	// A Go wasm module whose main returns is torn down, taking the exported
	// functions with it. Block forever instead.
	select {}
}

// compileFn lowers a document set to a timeline. It is the playground's inner
// loop, called on every keystroke after a debounce.
func compileFn(this js.Value, args []js.Value) (result any) {
	// Nothing may panic across the JS boundary: a Go panic in wasm aborts the
	// whole module, and there is no way back short of reloading the page. A
	// compiler bug should cost one red strip, not the session.
	defer func() {
		if r := recover(); r != nil {
			result = mustJSON(compileResult{Envelope: errorEnvelope("", fmt.Sprintf("internal error: %v", r))})
		}
	}()

	if len(args) != 2 {
		return mustJSON(compileResult{Envelope: errorEnvelope("", "cinegramCompile expects (entry, files)")})
	}
	entry := args[0].String()

	bundle, err := loader.Load(entry, readerFor(filesFromJS(args[1])))
	if err != nil {
		// Only the entry file failing to read reaches here; every other
		// unreadable path is already a diagnostic on the declaration that
		// named it. Reporting it as one too gives the page a single shape to
		// render. Mirrors cmdCompile in cmd/cinegram.
		return mustJSON(compileResult{Envelope: errorEnvelope(entry, err.Error())})
	}

	timeline := compile.CompileBundle(bundle)
	diags, _ := envelope.Collect(bundle.Bags())
	return mustJSON(compileResult{
		Envelope: envelope.Envelope{Timeline: timeline, Diagnostics: diags},
		Title:    html.DefaultTitle(timeline),
	})
}

// renderFn produces the same self-contained page `cinegram preview -o` writes.
// The empty Options are what the CLI passes, so the bytes match.
func renderFn(this js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = mustJSON(renderResult{Diagnostics: errorDiagnostics("", fmt.Sprintf("internal error: %v", r))})
		}
	}()

	if len(args) != 2 {
		return mustJSON(renderResult{Diagnostics: errorDiagnostics("", "cinegramRenderHTML expects (entry, files)")})
	}
	entry := args[0].String()

	bundle, err := loader.Load(entry, readerFor(filesFromJS(args[1])))
	if err != nil {
		return mustJSON(renderResult{Diagnostics: errorDiagnostics(entry, err.Error())})
	}

	timeline := compile.CompileBundle(bundle)
	diags, _ := envelope.Collect(bundle.Bags())

	page, err := html.Render(timeline, html.Options{})
	if err != nil {
		return mustJSON(renderResult{Diagnostics: append(diags, errorDiagnostics(entry, err.Error())...)})
	}
	return mustJSON(renderResult{HTML: string(page), Diagnostics: diags})
}

// filesFromJS copies the page's virtual filesystem into Go.
//
// Text arrives as a JS string and binary as a Uint8Array; anything else is
// skipped, which surfaces as the ordinary "cannot read" diagnostic rather than
// as a crash. Keys are cleaned so that "./a.dgm" and "a.dgm" name one file,
// matching what loader.Load does to the paths it looks up.
func filesFromJS(v js.Value) map[string][]byte {
	files := map[string][]byte{}
	if v.Type() != js.TypeObject {
		return files
	}

	keys := js.Global().Get("Object").Call("keys", v)
	for i := 0; i < keys.Length(); i++ {
		key := keys.Index(i).String()
		val := v.Get(key)

		switch val.Type() {
		case js.TypeString:
			files[filepath.Clean(key)] = []byte(val.String())
		case js.TypeObject:
			// js.CopyBytesToGo accepts a Uint8Array only, and returns 0 for
			// anything else rather than panicking.
			buf := make([]byte, val.Get("length").Int())
			if n := js.CopyBytesToGo(buf, val); n == len(buf) {
				files[filepath.Clean(key)] = buf
			}
		}
	}
	return files
}

// readerFor serves the in-memory file set. A miss is fs.ErrNotExist so that the
// loader produces its normal diagnostic and hint instead of a bespoke message.
func readerFor(files map[string][]byte) loader.ReadFileFunc {
	return func(path string) ([]byte, error) {
		if data, ok := files[filepath.Clean(path)]; ok {
			return data, nil
		}
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
	}
}

// errorEnvelope reports a failure that happened before any bag existed.
func errorEnvelope(file, msg string) envelope.Envelope {
	return envelope.Envelope{Timeline: nil, Diagnostics: errorDiagnostics(file, msg)}
}

func errorDiagnostics(file, msg string) []envelope.Diagnostic {
	return []envelope.Diagnostic{{
		File:     file,
		Severity: diag.SeverityError.String(),
		Message:  msg,
	}}
}

// mustJSON encodes a result for the page. The shapes here are all encodable, so
// a failure is a bug — but it still has to cross as something the page can
// parse, hence the hand-written fallback rather than a panic.
func mustJSON(v any) string {
	encoded, err := json.Marshal(v)
	if err != nil {
		return `{"timeline":null,"diagnostics":[{"file":"","line":0,"col":0,"severity":"error","message":"cannot encode result"}]}`
	}
	return string(encoded)
}
