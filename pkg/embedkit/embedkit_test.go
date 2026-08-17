package embedkit

import (
	"regexp"
	"strings"
	"testing"

	"github.com/tejaspanse/cinegram/pkg/emit/html"
)

// The loader fetches its siblings by name. Nothing else checks that those
// names are the ones Assets writes, so a rename in pkg/emit/html would ship a
// site whose diagrams 404 with every test still green.
func TestLoaderFetchesTheAssetsItShipsWith(t *testing.T) {
	kit := Assets()
	for name := range html.Assets() {
		if _, ok := kit[name]; !ok {
			t.Errorf("Assets() is missing %s, which pkg/emit/html serves", name)
		}
		if !strings.Contains(embedJS, name) {
			t.Errorf("cinegram-embed.js never loads %s; a page using the kit would render without it", name)
		}
	}
	for _, name := range []string{"cinegram-embed.js", "cinegram-embed.css"} {
		if _, ok := kit[name]; !ok {
			t.Errorf("Assets() is missing the kit's own %s", name)
		}
	}
}

// Same rule the preview page is held to by //pkg/emit/html:html_test — a site
// behind a firewall, or one with a CSP that names its own origin, has to work.
// The loader resolves everything relative to its own URL; a CDN slipping in
// here would be invisible until someone deployed it somewhere strict.
func TestKitReachesNoExternalOrigin(t *testing.T) {
	external := regexp.MustCompile(`(?i)\b(?:https?:)?//[a-z0-9-]+\.[a-z]`)
	for _, f := range []struct{ name, body string }{
		{"cinegram-embed.js", embedJS},
		{"cinegram-embed.css", embedCSS},
	} {
		for _, line := range strings.Split(f.body, "\n") {
			// Comments are prose about the outside world; the code is not.
			if t := strings.TrimSpace(line); strings.HasPrefix(t, "*") || strings.HasPrefix(t, "//") || strings.HasPrefix(t, "/*") {
				continue
			}
			if m := external.FindString(line); m != "" {
				t.Errorf("%s reaches outside the site: %q in %q", f.name, m, strings.TrimSpace(line))
			}
		}
	}
}

// The loader and the stylesheet are two halves of one contract: the loader
// puts these classes on, the stylesheet is the only thing that makes them mean
// anything. Splitting them across two files is what lets a site load the cheap
// half everywhere, so the seam is worth a test.
func TestLoaderAndStylesheetAgreeOnClassNames(t *testing.T) {
	for _, class := range []string{"cinegram--loading", "cinegram--mounted", "cinegram--failed"} {
		if !strings.Contains(embedJS, class) {
			t.Errorf("cinegram-embed.css styles .%s but the loader never sets it", class)
		}
		if !strings.Contains(embedCSS, "."+class) {
			t.Errorf("the loader sets .%s but cinegram-embed.css never styles it", class)
		}
	}
}
