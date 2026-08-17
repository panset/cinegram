// Package site renders the sources of cinegram's own website.
//
// The site is built by Zensical from www/. Most of it is prose someone wrote;
// this package generates the parts that would otherwise drift — the examples
// tour, one page per .dgm, and the guide, split out of README.md. Both are
// committed, and //site:site_test fails while the committed copy has fallen
// behind the examples, the README or the renderer. That is the same
// arrangement //editors/vscode:assets_test uses for the extension's asset
// copies, and it is what lets a Python site builder that Bazel cannot run
// still be held to a Bazel-checked input.
//
// The example pages carry no player of their own: each is a Markdown page
// with a `.cinegram` div in it, mounted at read time by the embed kit from
// //pkg/embedkit. So cinegram's own site reaches its diagrams by exactly the
// route it documents for everyone else's, and a regression in that route
// takes this site down first.
//
// What is *not* generated — the home page, the embedding guide, the plans —
// is hand-written under www/ and untouched by sync. The sweep owns three
// folders whole (examples/, guide/, assets/cinegram/timelines/) and nothing
// outside them, so the two kinds of page cannot be confused for each other.
package site

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/tejaspanse/cinegram/pkg/ir"
	"github.com/tejaspanse/cinegram/pkg/sitegen"
)

// marshalTimeline writes a timeline the way `cinegram compile` writes one, so
// a reader who regenerates a single page's JSON by hand gets a clean diff
// rather than a reformatting.
func marshalTimeline(t *ir.Timeline) ([]byte, error) {
	encoded, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

// Generated names the folders under www/ that sync writes and sweeps. A file
// under one of them with no source behind it is deleted; a file anywhere else
// under www/ is somebody's prose and is left alone.
var Generated = []string{"examples", "guide", "assets/cinegram/timelines"}

// TimelineDir is where the compiled timelines go. The embed kit resolves
// `timelines/` relative to its own URL, so this is fixed by where
// `cinegram assets` writes the loader, not chosen here.
const TimelineDir = "assets/cinegram/timelines"

// Build renders everything generated under www/, keyed by its path relative
// to www/. Warnings are the published examples' compile warnings: they never
// block a build, but a caller that swallowed them would ship a degraded page
// the committer never heard about.
func Build(examples fs.FS, readme []byte) (map[string][]byte, []string, error) {
	root, docs, warnings, err := sitegen.Discover(examples)
	if err != nil {
		return nil, nil, err
	}

	out := map[string][]byte{}
	if err := writeGuide(out, readme); err != nil {
		return nil, nil, err
	}
	if err := writeTour(out, root, docs); err != nil {
		return nil, nil, err
	}
	return out, warnings, nil
}

// tourSlug is an example's path within the tour, extension aside.
//
// It keeps the source's numeric ordering prefix, where Doc.Slug drops it. The
// examples are a tour and are meant to be read in order, and Zensical builds
// its navigation by sorting filenames — so "01-" is the only thing holding
// k8s-request in front of deploy-pipeline. The prefix shows in the URL and
// nowhere a reader looks: the sidebar and the page heading both come from the
// document's title.
func tourSlug(d *sitegen.Doc) string {
	return strings.TrimSuffix(d.Source, ".dgm")
}

// writeTour writes a page per example, an index per folder, and the compiled
// timeline each page plays.
func writeTour(out map[string][]byte, root *sitegen.Group, docs []*sitegen.Doc) error {
	for _, d := range docs {
		timeline, err := marshalTimeline(d.Timeline)
		if err != nil {
			return fmt.Errorf("%s: %w", d.Source, err)
		}
		out[path.Join(TimelineDir, tourSlug(d)+".json")] = timeline
		out[path.Join("examples", tourSlug(d)+".md")] = examplePage(d)
	}
	writeIndexes(out, root)
	return nil
}

func writeIndexes(out map[string][]byte, g *sitegen.Group) {
	out[path.Join("examples", g.Path, "index.md")] = folderIndex(g)
	for _, e := range g.Entries {
		if e.Group != nil {
			writeIndexes(out, e.Group)
		}
	}
}

// examplePage is one example: what it shows, the diagram itself, a way into
// the playground, and the source that produced it.
func examplePage(d *sitegen.Doc) []byte {
	slug := tourSlug(d)

	var b strings.Builder
	b.WriteString(generatedBy("examples/"+slug+".md", d.Source))
	fmt.Fprintf(&b, "# %s\n\n", d.Title)
	if d.Blurb != "" {
		fmt.Fprintf(&b, "%s\n\n", d.Blurb)
	}

	// The player is mounted here by assets/cinegram/cinegram-embed.js. Depth
	// is not a problem for the timeline path — the loader resolves it against
	// its own URL — but it is for the playground link, which is an ordinary
	// relative href written into the page.
	fmt.Fprintf(&b, "<div class=\"cinegram\" data-cinegram=\"%s\" data-height=\"%d\"></div>\n\n",
		slug, reservedHeight(d.Timeline))

	up := strings.Repeat("../", strings.Count(slug, "/")+1)
	fmt.Fprintf(&b, "[Edit in the playground](%splayground/#doc=%s){ .md-button }\n\n", up, d.Share)

	fmt.Fprintf(&b, "??? abstract \"The source — `%s`\"\n\n", d.Source)
	// Indented into the collapsed block, fenced so nothing in a .dgm is read
	// as Markdown. `dgm` is not a highlighting language anywhere yet; it is
	// written out so it becomes one the day it is.
	b.WriteString("    ```dgm\n")
	for _, line := range strings.Split(strings.TrimRight(d.Text, "\n"), "\n") {
		if line == "" {
			b.WriteString("\n")
			continue
		}
		fmt.Fprintf(&b, "    %s\n", line)
	}
	b.WriteString("    ```\n")

	if p, n := d.Prev(), d.Next(); p != nil || n != nil {
		b.WriteString("\n")
		if p != nil {
			fmt.Fprintf(&b, "← [%s](%s.md)  \n", p.Title, relSlug(slug, tourSlug(p)))
		}
		if n != nil {
			fmt.Fprintf(&b, "→ [%s](%s.md)\n", n.Title, relSlug(slug, tourSlug(n)))
		}
	}
	return []byte(b.String())
}

// folderIndex lists what is in a folder, with each entry's blurb.
func folderIndex(g *sitegen.Group) []byte {
	title := g.Name
	if g.Path == "" {
		title = "Examples"
	}

	var b strings.Builder
	b.WriteString(generatedBy(path.Join("examples", g.Path, "index.md"), "examples/"+g.Path))
	fmt.Fprintf(&b, "# %s\n\n", title)
	if g.Path == "" {
		b.WriteString("Every one of these is a `.dgm` file in [`examples/`]" +
			"(https://github.com/panset/cinegram/tree/main/examples) in the repository, " +
			"playing here through the same [embed kit](../embedding.md) any other site would use. " +
			"They read front to back.\n\n")
	}

	for _, e := range g.Entries {
		if e.Group != nil {
			fmt.Fprintf(&b, "## [%s](%s/index.md)\n\n", e.Group.Name, path.Base(e.Group.Path))
			var names []string
			for _, inner := range e.Group.Entries {
				if inner.Doc != nil {
					names = append(names, inner.Doc.Title)
				} else {
					names = append(names, inner.Group.Name+"/")
				}
			}
			fmt.Fprintf(&b, "%s\n\n", strings.Join(names, " · "))
			continue
		}
		fmt.Fprintf(&b, "## [%s](%s.md)\n\n", e.Doc.Title, path.Base(tourSlug(e.Doc)))
		if e.Doc.Blurb != "" {
			fmt.Fprintf(&b, "%s\n\n", e.Doc.Blurb)
		}
	}
	return []byte(b.String())
}

// reservedHeight is what a page's `.cinegram` box holds open until its player
// mounts, in pixels. The loader drops it once the player can measure itself,
// so this only has to stop the article shifting as each diagram arrives.
//
// Step count is the only predictor available without laying the page out, and
// it is a weak one: measured across the twelve examples at a 688px column,
// five steps produced anything from 796 to 1162 pixels, and a seven-step
// diagram came out shorter than a five-step one. What varies is mostly the
// diagram above the list, which nothing here can know.
//
// So this is a least-squares line through those twelve measurements, and it
// is wrong by up to ~280px on some of them. That is the accuracy on offer;
// the alternative is one flat number, which is wrong by 900 on the longest.
// Re-measure the constants if the player's chrome changes.
func reservedHeight(t *ir.Timeline) int {
	steps := 0
	for _, v := range t.Views {
		if v.ID != t.Root {
			continue
		}
		for _, s := range v.Scenarios {
			if len(s.Steps) > steps {
				steps = len(s.Steps)
			}
		}
	}
	const (
		chrome  = 450 // stage, caption, transport, bar and the box's padding
		perStep = 90
		most    = 2200
	)
	h := chrome + perStep*steps
	if h > most {
		return most
	}
	return h
}

// relSlug is the path from one example page to another, both given as slugs
// relative to examples/.
func relSlug(from, to string) string {
	up := strings.Repeat("../", strings.Count(from, "/"))
	return up + to
}

// generatedBy is the header every generated page carries. Zensical passes an
// HTML comment through untouched, so it survives into the served page and
// answers "why did my edit disappear" wherever someone is reading.
func generatedBy(out, source string) string {
	return fmt.Sprintf("<!-- Generated from %s by `bazel run //site:sync`. Do not edit: "+
		"//site:site_test fails while this file disagrees with its source, and the next "+
		"sync overwrites it. -->\n\n", source)
}
