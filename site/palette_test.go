package site

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/tejaspanse/cinegram/internal/repotest"
	"github.com/tejaspanse/cinegram/pkg/embedkit"
	"github.com/tejaspanse/cinegram/pkg/emit/html"
)

// The palette lives in two files and no build step connects them.
//
// www/assets/stylesheets/mainframe.css owns it: --cg-* literals, both schemes,
// with the prose that says what each colour means. runtime.css repeats the same
// colours in its [data-dgm-skin="mainframe"] block, because go:embed cannot
// reach outside its package and www/ is not a Go package — a shared source
// would mean generating one of the two files, which is not worth it for a dozen
// hexes. So the copy is honest instead: this table is the correspondence, and
// the test below fails naming the pair that drifted.
//
// A divergence that is *meant* — a colour the player needs to differ on,
// because a stage is not a paragraph — is a line changed here with a reason,
// not a test deleted. Alpha is deliberately not pinned: --dgm-panel-soft and
// --dgm-accent-soft are the same colours at whatever transparency the chrome
// they paint needs, so only the rgb triple has to agree.
var mainframePalette = []struct {
	cg, dgm string
	// Empty means both schemes. Only --dgm-shadow is one-sided: the paper
	// shadow is ink at low alpha, and on the CRT ground a shadow is black.
	scheme string
}{
	{cg: "--cg-ground", dgm: "--dgm-bg"},
	{cg: "--cg-bar", dgm: "--dgm-panel"},      // the printed band is the chrome
	{cg: "--cg-bar", dgm: "--dgm-panel-soft"}, // and its translucent form
	{cg: "--cg-rule-strong", dgm: "--dgm-border"},
	{cg: "--cg-ink", dgm: "--dgm-fg"},
	{cg: "--cg-dim", dgm: "--dgm-muted"},
	{cg: "--cg-green", dgm: "--dgm-accent"}, // an unprotected field: actionable
	{cg: "--cg-green", dgm: "--dgm-accent-soft"},
	{cg: "--cg-turquoise", dgm: "--dgm-response"}, // protected text: what came back
	{cg: "--cg-yellow", dgm: "--dgm-busy"},        // a literal, ochre on paper
	{cg: "--cg-red", dgm: "--dgm-fail"},
	{cg: "--cg-field", dgm: "--dgm-warn-bg"},
	{cg: "--cg-yellow", dgm: "--dgm-warn-fg"},
	{cg: "--cg-ink", dgm: "--dgm-shadow", scheme: "light"},
}

// TestMainframePaletteAgrees checks the table above against both files.
func TestMainframePaletteAgrees(t *testing.T) {
	root := repotest.Root(t, filepath.Join("www", "assets", "stylesheets", "mainframe.css"))
	mainframe, err := os.ReadFile(filepath.Join(root, "www", "assets", "stylesheets", "mainframe.css"))
	if err != nil {
		t.Fatalf("reading mainframe.css: %v", err)
	}
	// The runtime sheet comes from the embed rather than from disk: it is the
	// same bytes the player loads, and it needs no data dependency to reach.
	runtime := string(html.Assets()["runtime.css"])
	if runtime == "" {
		t.Fatal("html.Assets() no longer ships runtime.css")
	}

	// Both files declare their light values twice — once so a page always has a
	// colour, once under prefers-color-scheme for the page whose theme nobody
	// has stamped — and the two must not disagree either.
	cg := map[string]map[string]string{
		"light": cssBlock(t, string(mainframe), `[data-md-color-scheme="default"] {`),
		"dark":  cssBlock(t, string(mainframe), `[data-md-color-scheme="slate"] {`),
	}
	skin := map[string]map[string]string{
		"light": cssBlock(t, runtime, `:root[data-dgm-skin='mainframe'] {`),
		"dark":  cssBlock(t, runtime, `:root[data-dgm-skin='mainframe'][data-theme='dark']`),
	}
	guard := cssBlock(t, runtime, `:root[data-dgm-skin='mainframe']:not([data-theme='dark'])`)

	for _, p := range mainframePalette {
		for _, scheme := range []string{"light", "dark"} {
			if p.scheme != "" && p.scheme != scheme {
				continue
			}
			from, ok := cg[scheme][p.cg]
			if !ok {
				t.Errorf("mainframe.css no longer declares %s in the %s palette; the "+
					"correspondence table in this file needs updating with it", p.cg, scheme)
				continue
			}
			to, ok := skin[scheme][p.dgm]
			if !ok {
				t.Errorf("the mainframe skin in runtime.css no longer declares %s (%s); "+
					"the correspondence table in this file needs updating with it", p.dgm, scheme)
				continue
			}
			want, err := rgbOf(from)
			if err != nil {
				t.Errorf("mainframe.css %s (%s): %v", p.cg, scheme, err)
				continue
			}
			got, err := rgbOf(to)
			if err != nil {
				t.Errorf("runtime.css %s (%s): %v", p.dgm, scheme, err)
				continue
			}
			if got != want {
				t.Errorf("palette drifted in the %s scheme: mainframe.css %s = %s (%s) but "+
					"runtime.css %s = %s (%s). Change both, or change the correspondence "+
					"table in this file if the divergence is meant",
					scheme, p.cg, from, want, p.dgm, to, got)
			}
		}
		// The guard block is the light block again; a token that changed in one
		// and not the other shows as a theme that flips colour on OS preference.
		if p.scheme == "dark" {
			continue
		}
		if want, got := skin["light"][p.dgm], guard[p.dgm]; want != got {
			t.Errorf("the skin's prefers-color-scheme block disagrees with its light "+
				"defaults: %s is %q there and %q above", p.dgm, got, want)
		}
	}

	// The type face is the other half of the theme, and it is one family in both
	// schemes, so it is set once — in the light block, which every skinned page
	// resolves through.
	if font := skin["light"]["--dgm-font"]; !strings.Contains(font, "IBM Plex Mono") {
		t.Errorf("the mainframe skin's --dgm-font is %q; mainframe.css leads its stack "+
			"with IBM Plex Mono, which IBM drew looking at these terminals", font)
	}
}

// The site wears the skin by asking for it: `--cinegram-skin` on :root, which
// cinegram-embed.js reads and turns into `data-dgm-skin` on <html>. Three
// things have to line up for a diagram on these pages to look like the page
// around it, and none of them is visible from either file alone.
func TestSiteAsksForTheMainframeSkin(t *testing.T) {
	root := repotest.Root(t, filepath.Join("www", "assets", "stylesheets", "mainframe.css"))
	raw, err := os.ReadFile(filepath.Join(root, "www", "assets", "stylesheets", "mainframe.css"))
	if err != nil {
		t.Fatalf("reading mainframe.css: %v", err)
	}
	mainframe := cssComment.ReplaceAllString(string(raw), "")

	// 1. The site declares a skin, on the element the loader reads it from.
	m := regexp.MustCompile(`(?s):root\s*\{[^}]*--cinegram-skin:\s*([a-z0-9-]+)`).
		FindStringSubmatch(mainframe)
	if m == nil {
		t.Fatal("mainframe.css declares no --cinegram-skin on :root, so every player on " +
			"the site wears the runtime's neutral palette instead of this file's")
	}
	name := m[1]

	// 2. runtime.css has that skin. A typo here is a site that silently does
	//    not change, since an attribute nothing matches costs nothing.
	runtime := string(html.Assets()["runtime.css"])
	if want := "[data-dgm-skin='" + name + "']"; !strings.Contains(runtime, want) {
		t.Errorf("mainframe.css asks for the %q skin but runtime.css defines no %s block",
			name, want)
	}

	// 3. The loader still reads the property and still stamps the attribute.
	loader := string(embedkit.Assets()["cinegram-embed.js"])
	for _, hook := range []string{"--cinegram-skin", "data-dgm-skin"} {
		if !strings.Contains(loader, hook) {
			t.Errorf("cinegram-embed.js no longer mentions %s; the site's skin declaration "+
				"reaches nothing", hook)
		}
	}

	// And the tokens live in one place. Setting a --dgm-* on the host box works
	// — custom properties inherit, and that is how this file did it before the
	// skin existed — which is exactly the trap: it beats the skin silently, for
	// the site's players only, out of reach of the drift table above. A
	// deliberate override is a line here saying which token and why.
	if i := strings.Index(mainframe, "--dgm-"); i >= 0 {
		t.Errorf("mainframe.css declares %.24s…; the player's palette comes from the "+
			"mainframe skin in runtime.css now, and a token set here would shadow it "+
			"on this site alone", strings.TrimSpace(mainframe[i:]))
	}
}

var cssComment = regexp.MustCompile(`(?s)/\*.*?\*/`)

// cssBlock returns the custom properties declared in the block whose selector
// starts with prefix. Comments come out of the whole document first, which both
// keeps a commented-out declaration from counting and means the scan for the
// closing brace cannot trip over one inside a comment. Neither file nests a
// rule inside a palette block, so the first `}` ends it.
func cssBlock(t *testing.T, doc, prefix string) map[string]string {
	t.Helper()

	doc = cssComment.ReplaceAllString(doc, "")
	// The selector is matched with its whitespace collapsed, since a two-line
	// selector list is the same rule as a one-line one.
	flat := regexp.MustCompile(`\s+`).ReplaceAllString(doc, " ")
	prefix = regexp.MustCompile(`\s+`).ReplaceAllString(prefix, " ")

	i := strings.Index(flat, prefix)
	if i < 0 {
		t.Fatalf("no rule matching %q; if the selector was renamed, rename it here too", prefix)
	}
	open := strings.Index(flat[i:], "{")
	end := strings.Index(flat[i:], "}")
	if open < 0 || end < open {
		t.Fatalf("rule %q has no body", prefix)
	}

	out := map[string]string{}
	for _, decl := range strings.Split(flat[i+open+1:i+end], ";") {
		name, value, ok := strings.Cut(decl, ":")
		if !ok {
			continue
		}
		name, value = strings.TrimSpace(name), strings.TrimSpace(value)
		if strings.HasPrefix(name, "--") {
			out[name] = value
		}
	}
	return out
}

// rgbOf normalises the two forms the palette is written in — #rrggbb and
// rgba() — to their red, green and blue components. Alpha is dropped: see the
// table's comment.
func rgbOf(value string) (string, error) {
	if hex, ok := strings.CutPrefix(value, "#"); ok {
		if len(hex) != 6 {
			return "", fmt.Errorf("%q is not a six-digit hex colour", value)
		}
		n, err := strconv.ParseUint(hex, 16, 32)
		if err != nil {
			return "", fmt.Errorf("%q is not a hex colour: %v", value, err)
		}
		return fmt.Sprintf("%d,%d,%d", n>>16&0xff, n>>8&0xff, n&0xff), nil
	}
	if args, ok := strings.CutPrefix(value, "rgba("); ok {
		args = strings.TrimSuffix(args, ")")
		parts := strings.Split(args, ",")
		if len(parts) < 3 {
			return "", fmt.Errorf("%q has fewer than three colour components", value)
		}
		for i := range parts[:3] {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return strings.Join(parts[:3], ","), nil
	}
	return "", fmt.Errorf("%q is neither a hex colour nor an rgba(); this test only "+
		"understands the two forms the palette uses", value)
}
