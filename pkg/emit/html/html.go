// Package html renders a timeline as a single self-contained page.
//
// Everything is inlined — mermaid, the runtime, the stylesheet and the timeline
// JSON — for two reasons. The page has to work when opened straight off the
// filesystem, where an external fetch would be blocked; and a VS Code webview
// applies a strict CSP that rules out remote assets. Building for the stricter
// case from the start means the plugin can reuse this output unchanged.
package html

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html"

	"github.com/tejaspanse/diagramator/pkg/ir"
)

//go:embed assets/mermaid.min.js
var mermaidJS string

//go:embed assets/runtime.js
var runtimeJS string

//go:embed assets/runtime.css
var runtimeCSS string

// Options controls page generation.
type Options struct {
	// Title appears in the browser tab and page header.
	Title string
}

// Render returns a complete HTML document for the timeline.
func Render(t *ir.Timeline, opts Options) ([]byte, error) {
	title := opts.Title
	if title == "" {
		title = DefaultTitle(t)
	}

	payload, err := json.Marshal(struct {
		*ir.Timeline
		Title string `json:"title"`
	}{Timeline: t, Title: title})
	if err != nil {
		return nil, fmt.Errorf("encoding timeline: %w", err)
	}

	var b bytes.Buffer
	b.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n")
	b.WriteString("<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	fmt.Fprintf(&b, "<title>%s</title>\n", html.EscapeString(title))
	b.WriteString("<style>\n")
	b.WriteString(runtimeCSS)
	b.WriteString("\n</style>\n</head>\n<body>\n")
	b.WriteString("<div id=\"diagramator\"></div>\n")

	b.WriteString("<script>\n")
	b.WriteString(mermaidJS)
	b.WriteString("\n</script>\n")

	b.WriteString("<script>\n")
	b.WriteString(runtimeJS)
	b.WriteString("\n</script>\n")

	b.WriteString("<script>\n")
	// The payload goes in as a JSON string literal parsed at runtime rather
	// than as an object literal: it keeps diagram text containing `</script>`
	// or U+2028 from breaking out of the script element.
	fmt.Fprintf(&b, "var DIAGRAMATOR_TIMELINE = JSON.parse(%s);\n", jsStringLiteral(payload))
	// The player is kept on window so a host (a VS Code webview, a test) can
	// drive playback from outside the page.
	b.WriteString("window.DIAGRAMATOR_PLAYER = Diagramator.mount(document.getElementById('diagramator'), DIAGRAMATOR_TIMELINE);\n")
	b.WriteString("</script>\n</body>\n</html>\n")

	return b.Bytes(), nil
}

// DefaultTitle names a timeline after the view it opens on, since that is what
// the reader sees first. Render uses it when Options.Title is empty; it is
// exported for callers that need the same name outside a page, such as narrate.
func DefaultTitle(t *ir.Timeline) string {
	for _, v := range t.Views {
		if v.ID != t.Root {
			continue
		}
		if v.Title != "" {
			return v.Title
		}
		if len(v.Scenarios) == 1 && v.Scenarios[0].Name != "" {
			return v.Scenarios[0].Name
		}
		if v.ID != "" {
			return v.ID
		}
	}
	return "Diagramator"
}

// jsStringLiteral encodes data as a JavaScript string literal that is safe to
// embed inside a <script> element.
//
// encoding/json already escapes <, > and & as \u003c, \u003e and \u0026, and
// escapes U+2028/U+2029 — line terminators in JavaScript but legal inside JSON
// strings. That covers every way this payload could break out of the script
// element, so no further rewriting is needed.
func jsStringLiteral(data []byte) string {
	quoted, err := json.Marshal(string(data))
	if err != nil {
		return `""`
	}
	return string(quoted)
}
