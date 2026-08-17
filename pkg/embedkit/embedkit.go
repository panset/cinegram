// Package embedkit ships the files a Material-family static site needs to
// play cinegrams inside its own pages.
//
// A site that wants a diagram in the middle of an article has a different
// problem from one that wants a page per diagram. `cinegram site` answers the
// second: it owns the whole page and can afford to. The first has a theme, a
// palette toggle, a search box and keyboard shortcuts already, and the player
// has to arrive as a guest — mounting into a div the author wrote, listening
// for keys only inside itself, leaving the address bar alone, and costing
// nothing at all on the pages that show no diagram.
//
// The kit is two files, a loader and the box it mounts into, plus the three
// runtime files from pkg/emit/html. Assets returns all five, keyed by the
// names the loader expects to find beside itself; `cinegram assets` is the
// command that writes them out. Cinegram's own site is built this way, so the
// path a downstream site takes is the path this repository takes.
//
// The two files here are the canonical copy — the same rule runtime.js lives
// under, and for the same reason: go:embed cannot reach outside its package,
// so a second copy anywhere is a copy that drifts.
package embedkit

import (
	_ "embed"

	"github.com/tejaspanse/cinegram/pkg/emit/html"
)

//go:embed assets/cinegram-embed.js
var embedJS string

//go:embed assets/cinegram-embed.css
var embedCSS string

// Assets returns the whole kit, keyed by the filename it must be written as.
// The loader resolves mermaid, the runtime and the timelines folder relative
// to its own URL, so the five files belong in one directory and the names are
// not the caller's to choose.
func Assets() map[string][]byte {
	out := map[string][]byte{
		"cinegram-embed.js":  []byte(embedJS),
		"cinegram-embed.css": []byte(embedCSS),
	}
	for name, content := range html.Assets() {
		out[name] = content
	}
	return out
}
