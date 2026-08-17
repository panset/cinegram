package site

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// README.md is the guide. Keeping it that way is the point of this file: it is
// the first thing a visitor to the repository reads, it is what ships in the
// Go module, and a second copy under www/ would be wrong within a month.
//
// So the site's guide pages are cut out of it. The cut is by ## heading, and
// the grouping below is the whole of the editorial judgement — everything else
// is mechanical, and a heading this table does not mention is a build failure
// rather than a page quietly missing from the site.
//
// The slugs carry a numeric prefix for the same reason the examples do:
// Zensical sorts filenames to build its navigation, and a guide that opens on
// Commands and works back to The language is not a guide. Nobody reads the
// prefix — page titles come from each page's own H1.
var guidePages = []guidePage{
	{
		Slug:     "01-language",
		Title:    "The language",
		Summary:  "What a `.dgm` file is made of, and the three diagram kinds it can animate.",
		Sections: []string{"The language", "Sequence diagrams", "State diagrams"},
	},
	{
		Slug:     "02-storytelling",
		Title:    "Telling the story",
		Summary:  "Turning a diagram that moves into one that explains: narration, state that persists, where to look, and what happens when it goes wrong.",
		Sections: []string{"Narration", "Persistent state", "Attention", "Storyboard", "Failure paths"},
	},
	{
		Slug:     "03-presenting",
		Title:    "Presenting and sharing",
		Summary:  "Standing in front of a room with one, cutting it to a clip, and sending someone a single moment.",
		Sections: []string{"Presenter mode", "Reels", "Sharing a moment"},
	},
	{
		Slug:     "04-interaction",
		Title:    "Interaction",
		Summary:  "Clicking into a diagram: drill-down, reveals and the bindings behind them.",
		Sections: []string{"Interaction"},
	},
	{
		Slug:     "05-commands",
		Title:    "Commands",
		Summary:  "Every subcommand of the CLI and what it writes.",
		Sections: []string{"Commands"},
	},
	{
		Slug:     "06-project",
		Title:    "The project",
		Summary:  "How the pieces fit, how to build it, what is finished and what is not.",
		Sections: []string{"How it fits together", "Building", "Status", "The VS Code extension", "License"},
	},
}

type guidePage struct {
	Slug     string
	Title    string
	Summary  string
	Sections []string
}

// repoBlob is where a link to a file in the repository has to point once the
// page is on a website rather than in a checkout.
const repoBlob = "https://github.com/panset/cinegram/blob/main/"

// writeGuide cuts README.md into the pages above, plus an index over them.
func writeGuide(out map[string][]byte, readme []byte) error {
	sections, order, err := splitSections(string(readme))
	if err != nil {
		return err
	}

	// Which page each ## heading ended up on, so a cross-reference between
	// two of them can be rewritten into a link between two pages.
	home := map[string]string{}
	for _, p := range guidePages {
		for _, s := range p.Sections {
			if was, taken := home[s]; taken {
				return fmt.Errorf("guide: %q is on both the %s and %s pages", s, was, p.Slug)
			}
			if _, ok := sections[s]; !ok {
				return fmt.Errorf("guide: the %s page wants a %q section, but README.md has no such heading",
					p.Slug, s)
			}
			home[s] = p.Slug
		}
	}
	// The reverse: a heading nobody claimed. Left alone it would simply not be
	// on the site, which is the kind of omission that survives for years.
	var orphans []string
	for _, s := range order {
		if _, ok := home[s]; !ok {
			orphans = append(orphans, strconv.Quote(s))
		}
	}
	if len(orphans) > 0 {
		return fmt.Errorf("guide: README.md has %s, which no page in site/guide.go claims — "+
			"add the heading to a page's Sections, or the site silently drops it",
			strings.Join(orphans, ", "))
	}

	anchors := anchorHome(home)
	for _, p := range guidePages {
		var b strings.Builder
		b.WriteString(generatedBy("guide/"+p.Slug+".md", "README.md"))
		fmt.Fprintf(&b, "# %s\n\n", p.Title)
		for _, s := range p.Sections {
			body, err := rewriteLinks(sections[s], p.Slug, anchors)
			if err != nil {
				return fmt.Errorf("guide: in README.md's %q section: %w", s, err)
			}
			// The ## headings of the README become the ## headings of the
			// page, under the page's own H1. Nothing is renumbered: the
			// anchors readers already have in their notes keep working.
			fmt.Fprintf(&b, "## %s\n%s", s, body)
		}
		out[path.Join("guide", p.Slug+".md")] = []byte(strings.TrimRight(b.String(), "\n") + "\n")
	}

	out["guide/index.md"] = guideIndex()
	return nil
}

func guideIndex() []byte {
	var b strings.Builder
	b.WriteString(generatedBy("guide/index.md", "README.md"))
	b.WriteString("# Guide\n\n")
	b.WriteString("These pages are [README.md](" + repoBlob + "README.md) cut into sections. " +
		"The README is the source; editing a page here is editing something that gets overwritten.\n\n")
	for _, p := range guidePages {
		fmt.Fprintf(&b, "## [%s](%s.md)\n\n%s\n\n", p.Title, p.Slug, p.Summary)
	}
	return []byte(b.String())
}

var headingRe = regexp.MustCompile(`^##\s+(.*)$`)

// splitSections cuts a Markdown document at its ## headings, returning each
// section's body by heading text and the order they appeared in. Everything
// before the first ## is dropped: on the README that is the title and the
// pitch, and the site has a home page of its own that does that job better.
//
// Fenced blocks are tracked, because the README's shell examples contain
// comment lines that begin with ## and are not headings.
func splitSections(doc string) (map[string]string, []string, error) {
	sections := map[string]string{}
	var order []string

	current := ""
	var body strings.Builder
	fenced := false

	flush := func() {
		if current != "" {
			sections[current] = strings.TrimRight(body.String(), "\n") + "\n\n"
		}
		body.Reset()
	}

	for _, line := range strings.Split(doc, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
		}
		if !fenced {
			if m := headingRe.FindStringSubmatch(line); m != nil {
				flush()
				current = strings.TrimSpace(m[1])
				if _, taken := sections[current]; taken {
					return nil, nil, fmt.Errorf("README.md has two %q headings; "+
						"the split cannot tell which page a link to it means", current)
				}
				sections[current] = ""
				order = append(order, current)
				continue
			}
		}
		if current != "" {
			body.WriteString(line)
			body.WriteString("\n")
		}
	}
	flush()
	if fenced {
		return nil, nil, fmt.Errorf("README.md ends inside a fenced code block")
	}
	return sections, order, nil
}

// anchorHome maps the anchor a heading generates to the page it now lives on.
func anchorHome(home map[string]string) map[string]string {
	out := map[string]string{}
	for section, slug := range home {
		out[anchorOf(section)] = slug
	}
	return out
}

var notAnchor = regexp.MustCompile(`[^a-z0-9 -]`)

// anchorOf is the id a Markdown renderer gives a heading: lowercased, symbols
// dropped, spaces hyphenated. Both GitHub and the site's toc agree on this
// much, which is all the README's own cross-references need.
func anchorOf(heading string) string {
	s := strings.ToLower(strings.TrimSpace(heading))
	s = notAnchor.ReplaceAllString(s, "")
	return strings.ReplaceAll(s, " ", "-")
}

var linkRe = regexp.MustCompile(`\]\(([^)\s]+)\)`)

// rewriteLinks repoints the README's links for a page on the website.
//
// A README link is relative to the repository root; the same text on a site
// page means something else or nothing at all. Three cases, and an error for
// anything that is not one of them — a link this function does not understand
// is a 404 that a --strict build would catch on the site and a reader would
// hit in the meantime.
func rewriteLinks(body, slug string, anchors map[string]string) (string, error) {
	var bad []string
	out := linkRe.ReplaceAllStringFunc(body, func(m string) string {
		target := linkRe.FindStringSubmatch(m)[1]
		switch {
		// Already absolute, or a mail/anchor scheme: leave it.
		case strings.Contains(target, "://"), strings.HasPrefix(target, "mailto:"):
			return m
		// A cross-reference to another README heading, which is now either
		// this page or a sibling.
		case strings.HasPrefix(target, "#"):
			on, ok := anchors[strings.TrimPrefix(target, "#")]
			if !ok {
				bad = append(bad, target+" (no ## heading generates that anchor)")
				return m
			}
			if on == slug {
				return m
			}
			return fmt.Sprintf("](%s.md%s)", on, target)
		// A path in the repository. The site does not serve the repository,
		// so it has to leave for GitHub.
		default:
			return fmt.Sprintf("](%s%s)", repoBlob, target)
		}
	})
	if len(bad) > 0 {
		sort.Strings(bad)
		return "", fmt.Errorf("cannot rewrite %s", strings.Join(bad, "; "))
	}
	return out, nil
}
