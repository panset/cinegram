// Package site renders the GitHub Pages demo site into memory.
//
// The site is nothing but committed artifacts: GitHub Pages serves docs/
// straight from the main branch, so there is no build step anywhere but the
// committer's machine. `bazel run //site:sync` writes what Build returns into
// docs/, and the freshness test in this package fails the build whenever the
// committed copy has fallen behind the examples or the renderer — the same
// arrangement //editors/vscode:assets_test uses for the extension's asset
// copies.
//
// The heavy lifting is pkg/sitegen — the same generator `cinegram site`
// exposes — rooted at docs/demos/, which is where the demo pages have always
// lived. What remains here is the repo's own presentation (title, playground,
// links) and the top-level redirect into demos/.
//
// A page's URL is its source's path under examples/, so reorganising that
// folder republishes the demos at new URLs and sync deletes the old pages.
// The examples were grouped into a tour (01-basics/, 02-storytelling/, …)
// knowing that, and no redirects were left behind: the site is a gallery
// people arrive at through the index, not a set of deep links under warranty.
package site

import (
	"io/fs"

	"github.com/tejaspanse/cinegram/pkg/sitegen"
)

// Build renders the docs/ payload from the examples tree: the generated site
// under demos/, a top-level redirect into it, and .nojekyll so Pages skips
// its Jekyll pass. Returned warnings are the published examples' compile
// warnings — they never block publishing, but a caller that swallowed them
// would ship a degraded demo the committer never heard about.
func Build(examples fs.FS) (map[string][]byte, []string, error) {
	pages, warnings, err := sitegen.Build(examples, sitegen.Config{
		Title: "Cinegram",
		// Relative to the generated site's root (docs/demos/), which is why
		// the playground — assembled beside docs/ content at /playground/ —
		// is one level up.
		Playground: "../playground/",
		Hero:       "Write your own in the browser — the compiler runs in the tab, nothing is uploaded.",
		Links: []sitegen.Link{
			{Name: "Playground", URL: "../playground/"},
			{Name: "GitHub", URL: "https://github.com/panset/cinegram"},
		},
	})
	if err != nil {
		return nil, nil, err
	}

	out := map[string][]byte{
		".nojekyll":  {},
		"index.html": []byte(redirectStub),
	}
	for rel, content := range pages {
		out["demos/"+rel] = content
	}
	return out, warnings, nil
}

// redirectStub is all that remains of the old hand-written landing page: the
// generated index under demos/ carries the hero, the links and the sidebar
// now, and two landing pages would only drift apart.
const redirectStub = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta http-equiv="refresh" content="0; url=demos/">
<title>Cinegram</title>
</head>
<body>
<p><a href="demos/">Cinegram demos</a></p>
</body>
</html>
`
