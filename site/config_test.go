package site

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/tejaspanse/cinegram/internal/repotest"
	"github.com/tejaspanse/cinegram/pkg/embedkit"
)

// TestConfigLoadsTheEmbedKit is here because Zensical will not say a word.
//
// `extra_css` and `extra_javascript` naming a file that does not exist is not
// an error to it — a build with no kit installed reports "No issues found" and
// publishes a site on which every diagram silently fails to load. So the one
// thing that would catch a rename in pkg/embedkit, or a typo in the config,
// has to be checked here.
func TestConfigLoadsTheEmbedKit(t *testing.T) {
	root := repotest.Root(t, "zensical.toml")
	config, err := os.ReadFile(filepath.Join(root, "zensical.toml"))
	if err != nil {
		t.Fatalf("reading zensical.toml: %v", err)
	}

	kit := embedkit.Assets()

	// Two kinds of file get listed, and they fail in different ways.
	//
	// A kit asset is written by `cinegram assets` into one flat folder, so its
	// basename is the whole of the contract and naming it anywhere else 404s.
	// Everything else is the site's own — mainframe.css and whatever joins it
	// — and is committed under docs_dir, so the check that matters there is
	// simply that the file is on disk. Zensical reports neither.
	//
	// `listed` holds kit basenames only. A site asset that happened to share a
	// name with one would otherwise read as the kit being loaded site-wide.
	listed := map[string]bool{}
	for _, field := range []string{"extra_css", "extra_javascript"} {
		for _, ref := range tomlStringList(string(config), field) {
			dir, name := filepath.Split(ref)
			if _, isKit := kit[name]; isKit || dir == "assets/cinegram/" {
				if want := "assets/cinegram/"; dir != want {
					t.Errorf("%s lists %q; the kit is installed into %s, so that file will 404",
						field, ref, want)
				}
				listed[name] = true
				continue
			}
			// docs_dir, per zensical.toml. The paths are relative to it.
			if _, err := os.Stat(filepath.Join(root, "www", filepath.FromSlash(ref))); err != nil {
				t.Errorf("%s lists %q, which is not a kit asset and is not committed "+
					"under www/; Zensical will not say a word and it will 404", field, ref)
			}
		}
	}

	for _, name := range []string{"cinegram-embed.css", "cinegram-embed.js"} {
		if _, ok := kit[name]; !ok {
			t.Fatalf("the kit no longer ships %s; this test needs updating with it", name)
		}
		if !listed[name] {
			t.Errorf("zensical.toml does not load %s, so no diagram on the site will mount", name)
		}
	}
	// The other three are fetched by the loader on demand, and listing one
	// here would put mermaid's 2.6 MB on every page of the site.
	for name := range kit {
		if strings.HasPrefix(name, "cinegram-embed.") {
			continue
		}
		if listed[name] {
			t.Errorf("zensical.toml loads %s site-wide; the loader fetches it only on the "+
				"pages that need it, and listing it costs every other page", name)
		}
	}
}

// tomlStringList pulls a single-line array of quoted strings out of a TOML
// document. Enough for two known keys, and not worth a parser.
func tomlStringList(doc, key string) []string {
	re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `\s*=\s*\[([^\]]*)\]`)
	m := re.FindStringSubmatch(doc)
	if m == nil {
		return nil
	}
	var out []string
	for _, part := range strings.Split(m[1], ",") {
		if s := strings.Trim(strings.TrimSpace(part), `"`); s != "" {
			out = append(out, s)
		}
	}
	return out
}
