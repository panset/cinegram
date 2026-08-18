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

	"github.com/tejaspanse/cinegram/pkg/ir"
)

//go:embed assets/mermaid.min.js
var mermaidJS string

//go:embed assets/runtime.js
var runtimeJS string

//go:embed assets/runtime.css
var runtimeCSS string

// Skin is the data-dgm-skin value the emitted page opts into, and the name of
// the palette block runtime.css defines for it. Exported because the page is
// not the only surface cinegram owns end to end — pkg/sitegen writes the same
// attribute onto the listings that sit beside these pages — and two spellings
// of one name is a site that is half skinned with nothing failing to say so.
const Skin = "mainframe"

// ThemeBootScript is the inline <head> script that puts the reader's theme on
// the root element before the first paint.
//
// It cannot come from runtime.js, which loads at the end of the body: a page
// opened from disk with dark chosen would paint a white document first and
// then flip. Three surfaces need the same three lines — this page, the listings
// pkg/sitegen writes, the playground's hand-written index — so they come from
// here, and the playground's copy is pinned against this string by a test.
//
// Nothing stored *removes* the attribute rather than picking a side, and that
// is the state a page nobody has pressed the control on is in: runtime.css
// resolves an unstamped page through prefers-color-scheme, so following the
// system costs no script and no flash. Anything else in the key — a value from
// another page on the origin, or an older build's third state — reads the same
// way, which is also how runtime.js reads it. localStorage throws outright in
// some private modes, which is a page that cannot remember a preference, not a
// page that fails to render.
func ThemeBootScript() string {
	return `<script>
(function () {
  try {
    var t = localStorage.getItem('dgm.theme');
    if (t === 'light' || t === 'dark') document.documentElement.setAttribute('data-theme', t);
    else document.documentElement.removeAttribute('data-theme');
  } catch (e) { /* a browser that will not remember still renders */ }
})();
</script>
`
}

// ThemeToggleHTML is the page's theme control as markup, for a page that wants
// it present and styled before any script has run. runtime.js finds it by its
// data-dgm-theme-toggle marker on load and wires the light/dark flip onto it;
// chrome built in script asks Cinegram.themeToggle() for the same button
// instead. The glyph is left to the runtime, which owns the icon set.
func ThemeToggleHTML() string {
	return `<button type="button" class="dgm-page-theme" data-dgm-theme-toggle aria-live="polite" aria-label="Theme"></button>
`
}

// Options controls page generation.
type Options struct {
	// Title appears in the browser tab and page header.
	Title string

	// AssetsHref switches the page from inlining the three runtime assets to
	// referencing them from this directory by relative path ("assets", or
	// "../assets" from a nested page). Self-contained output remains the
	// default and the contract preview and the VS Code webview depend on;
	// this mode exists for a site that serves many pages from one origin and
	// wants one copy of mermaid, not one per page. The files themselves come
	// from Assets(), written once by the caller.
	AssetsHref string

	// Site chrome, injected verbatim by a site generator: extra <head>
	// markup, a sidebar, a bar above the player and one below. Each fragment
	// is already-safe HTML — the generator owns its escaping, exactly as it
	// owns the fragments' styling. All optional; all ignored is the ordinary
	// standalone page.
	HeadExtra, Nav, Header, Footer []byte
}

// Assets returns the runtime files a shared-asset site serves once, keyed by
// the filenames pages emitted with a non-empty AssetsHref refer to.
func Assets() map[string][]byte {
	return map[string][]byte{
		"mermaid.min.js": []byte(mermaidJS),
		"runtime.js":     []byte(runtimeJS),
		"runtime.css":    []byte(runtimeCSS),
	}
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
	// The mainframe skin is opt-in, and this page is one of the surfaces that
	// opts in: runtime.css keys it off the root element, so a page cinegram owns
	// end to end wears the theme while the same stylesheet, contributed into a
	// document somebody else owns, stays neutral. That is why the attribute is
	// here and not in the token block.
	fmt.Fprintf(&b, "<!doctype html>\n<html lang=\"en\" data-dgm-skin=%q>\n<head>\n", Skin)
	b.WriteString("<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	fmt.Fprintf(&b, "<title>%s</title>\n", html.EscapeString(title))
	if opts.AssetsHref != "" {
		fmt.Fprintf(&b, "<link rel=\"stylesheet\" href=\"%s/runtime.css\">\n", html.EscapeString(opts.AssetsHref))
	} else {
		b.WriteString("<style>\n")
		b.WriteString(runtimeCSS)
		b.WriteString("\n</style>\n")
	}
	// After the stylesheet, because it decides which half of that sheet's
	// tokens the first paint resolves through, and before anything the page
	// draws — the whole point of it is to beat the first frame.
	b.WriteString(ThemeBootScript())
	b.Write(opts.HeadExtra)
	// dgm-standalone is what tells the stylesheet it owns the document: the page
	// fills the window and paints its own background. The same sheet is loaded
	// by hosts that put a diagram inside a page they already own — the VS Code
	// Markdown preview contributes it next to the editor's own styles — and
	// there the class is absent, so those rules never fire. dgm-sited marks a
	// page that additionally carries site chrome, so the site stylesheet can
	// lay the sidebar beside the player without runtime.css knowing.
	if len(opts.Nav) > 0 {
		b.WriteString("</head>\n<body class=\"dgm-standalone dgm-sited\">\n")
		b.Write(opts.Nav)
		b.WriteString("<div class=\"dgm-site-main\">\n")
	} else {
		b.WriteString("</head>\n<body class=\"dgm-standalone\">\n")
		// The theme control, but only on the page that has no other chrome: a
		// site generator puts one in the header it is already drawing, and two
		// controls for one page-wide state is a page that can disagree with
		// itself. Outside #cinegram, which the player empties on every render.
		b.WriteString(ThemeToggleHTML())
	}
	b.Write(opts.Header)
	b.WriteString("<div id=\"cinegram\"></div>\n")
	b.Write(opts.Footer)
	if len(opts.Nav) > 0 {
		b.WriteString("</div>\n")
	}

	if opts.AssetsHref != "" {
		fmt.Fprintf(&b, "<script src=\"%s/mermaid.min.js\"></script>\n", html.EscapeString(opts.AssetsHref))
		fmt.Fprintf(&b, "<script src=\"%s/runtime.js\"></script>\n", html.EscapeString(opts.AssetsHref))
	} else {
		b.WriteString("<script>\n")
		b.WriteString(mermaidJS)
		b.WriteString("\n</script>\n")

		b.WriteString("<script>\n")
		b.WriteString(runtimeJS)
		b.WriteString("\n</script>\n")
	}

	b.WriteString("<script>\n")
	// The payload goes in as a JSON string literal parsed at runtime rather
	// than as an object literal: it keeps diagram text containing `</script>`
	// or U+2028 from breaking out of the script element.
	fmt.Fprintf(&b, "var CINEGRAM_TIMELINE = JSON.parse(%s);\n", jsStringLiteral(payload))
	// The player is kept on window so a host (a VS Code webview, a test) can
	// drive playback from outside the page.
	b.WriteString("window.CINEGRAM_PLAYER = Cinegram.mount(document.getElementById('cinegram'), CINEGRAM_TIMELINE);\n")
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
	return "Cinegram"
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
