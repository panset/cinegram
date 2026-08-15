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
// Reading is injected rather than assumed, for the same reason pkg/loader
// takes its read function as an argument: Build stays testable against an
// in-memory tree.
package site

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tejaspanse/cinegram/pkg/compile"
	"github.com/tejaspanse/cinegram/pkg/emit/html"
	"github.com/tejaspanse/cinegram/pkg/loader"
)

// Build renders one self-contained demo page per standalone example plus an
// index, returned as docs/-relative paths. It also returns any warning
// diagnostics the published examples compiled with: a warning never blocks
// publishing, but a caller that swallowed it would ship a degraded demo the
// committer never heard about.
func Build(paths []string, read loader.ReadFileFunc) (map[string][]byte, []string, error) {
	sorted := append([]string(nil), paths...)
	sort.Strings(sorted)

	type loaded struct {
		path   string
		bundle *loader.Bundle
	}
	var all []loaded

	// Every path a bundle pulled in beyond its entry file is a sub-view.
	subViews := map[string]bool{}

	for _, path := range sorted {
		bundle, err := loader.Load(path, read)
		if err != nil {
			return nil, nil, fmt.Errorf("loading %s: %w", path, err)
		}
		for _, u := range bundle.Units[1:] {
			subViews[u.Path] = true
		}
		all = append(all, loaded{path: path, bundle: bundle})
	}

	// An example another example pulls in as a sub-view (`view … from`) is
	// part of that demo's bundle and gets no page of its own — the player
	// reaches it by navigation. But the rule cannot simply be "referenced
	// means no page": two examples referencing each other would drop each
	// other, and both would vanish from the site with no diagnostic. So a
	// referenced example stays unpublished only while a published demo
	// reaches it, sweeping in path order until nothing is stranded.
	var publish []loaded
	reachable := map[string]bool{}
	take := func(l loaded) {
		publish = append(publish, l)
		for _, u := range l.bundle.Units[1:] {
			reachable[u.Path] = true
		}
	}
	for _, l := range all {
		if !subViews[filepath.Clean(l.path)] {
			take(l)
		}
	}
	for _, l := range all {
		if subViews[filepath.Clean(l.path)] && !reachable[filepath.Clean(l.path)] {
			take(l)
		}
	}
	sort.Slice(publish, func(i, j int) bool { return publish[i].path < publish[j].path })

	out := map[string][]byte{
		// Pages skips its Jekyll pass when this file exists; nothing here
		// needs Jekyll, and skipping it keeps deploys immediate.
		".nojekyll": {},
	}
	var warnings []string
	var entries []indexEntry
	pageSource := map[string]string{}

	for _, l := range publish {
		// Compile before checking the bags: some problems only surface
		// during the timing pass, and a broken page must never reach docs/.
		// An unpublished sub-view is compiled as part of every bundle that
		// pulls it in, so its diagnostics surface here too.
		timeline := compile.CompileBundle(l.bundle)
		if l.bundle.HasErrors() {
			return nil, nil, fmt.Errorf("%s does not compile:\n%s", l.path, bagText(l.bundle))
		}
		if text := bagText(l.bundle); text != "" {
			warnings = append(warnings, text)
		}

		name := strings.TrimSuffix(filepath.Base(l.path), filepath.Ext(l.path)) + ".html"
		if prev, taken := pageSource[name]; taken {
			return nil, nil, fmt.Errorf("%s and %s would both publish as demos/%s", prev, l.path, name)
		}
		pageSource[name] = l.path

		page, err := html.Render(timeline, html.Options{})
		if err != nil {
			return nil, nil, fmt.Errorf("rendering %s: %w", l.path, err)
		}
		out["demos/"+name] = page

		source, err := read(l.path)
		if err != nil {
			return nil, nil, fmt.Errorf("reading %s: %w", l.path, err)
		}
		entries = append(entries, indexEntry{
			Title:  html.DefaultTitle(timeline),
			Href:   "demos/" + name,
			Source: filepath.Base(l.path),
			Blurb:  leadingComment(string(source)),
		})
	}

	index, err := renderIndex(entries)
	if err != nil {
		return nil, nil, err
	}
	out["index.html"] = index
	return out, warnings, nil
}

// Examples lists the .dgm sources under root's examples/ directory, the input
// Build's real callers share. os.ReadDir rather than filepath.Glob: a
// workspace path containing a glob metacharacter ([, ?, *) would silently
// match nothing as a pattern.
func Examples(root string) ([]string, error) {
	dir := filepath.Join(root, "examples")
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, item := range items {
		if !item.IsDir() && filepath.Ext(item.Name()) == ".dgm" {
			out = append(out, filepath.Join(dir, item.Name()))
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no .dgm files in %s", dir)
	}
	return out, nil
}

func bagText(b *loader.Bundle) string {
	var parts []string
	for _, bag := range b.Bags() {
		if bag.Len() > 0 {
			parts = append(parts, bag.String())
		}
	}
	return strings.Join(parts, "\n")
}

// leadingComment returns a document's leading %% comment block, which the
// examples use to describe themselves. The block may wrap across lines; a
// `%% ---` separator ends it, marking where the summary gives way to notes
// meant for someone reading the source. A `%%{…}` line is a Mermaid
// directive, not a comment, and ends the block the same way.
func leadingComment(source string) string {
	var parts []string
	for _, line := range strings.Split(source, "\n") {
		line = strings.TrimSpace(line)
		if line == "" && len(parts) == 0 {
			continue
		}
		rest, ok := strings.CutPrefix(line, "%%")
		if !ok || strings.HasPrefix(rest, "{") || strings.TrimSpace(rest) == "---" {
			break
		}
		parts = append(parts, strings.TrimSpace(rest))
	}
	return strings.Join(parts, " ")
}

type indexEntry struct {
	Title  string
	Href   string
	Source string
	Blurb  string
}

func renderIndex(entries []indexEntry) ([]byte, error) {
	var b strings.Builder
	if err := indexTemplate.Execute(&b, entries); err != nil {
		return nil, fmt.Errorf("rendering index: %w", err)
	}
	return []byte(b.String()), nil
}

// The index is the one page the demo pages do not render for us. It is styled
// inline for the same reason the demo pages inline everything: the site must
// work with no network fetches beyond the pages themselves.
var indexTemplate = template.Must(template.New("index").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Cinegram demos</title>
<style>
:root {
  color-scheme: light dark;
  --bg: #f8fafc; --card: #ffffff; --ink: #0f172a; --muted: #475569;
  --line: #e2e8f0; --accent: #2563eb;
}
@media (prefers-color-scheme: dark) {
  :root { --bg: #0b1120; --card: #101a33; --ink: #e2e8f0; --muted: #94a3b8;
          --line: #1e293b; --accent: #60a5fa; }
}
* { box-sizing: border-box; }
body {
  margin: 0; background: var(--bg); color: var(--ink);
  font: 16px/1.55 system-ui, -apple-system, "Segoe UI", sans-serif;
}
main { max-width: 44rem; margin: 0 auto; padding: 3rem 1.25rem 4rem; }
h1 { margin: 0; font-size: 1.7rem; }
.tagline { margin: .35rem 0 0; color: var(--muted); }
.repo { margin: .75rem 0 0; font-size: .9rem; }
a { color: var(--accent); }
ul.demos { list-style: none; margin: 2.25rem 0 0; padding: 0; }
ul.demos li {
  background: var(--card); border: 1px solid var(--line); border-radius: 10px;
  padding: 1rem 1.15rem; margin: .8rem 0;
}
ul.demos a.title { font-weight: 600; font-size: 1.05rem; text-decoration: none; }
ul.demos a.title:hover { text-decoration: underline; }
.source { color: var(--muted); font-size: .8rem; font-family: ui-monospace, monospace; margin-left: .5rem; }
.try { margin: 1.4rem 0 0; }
.try a {
  display: inline-block; background: var(--accent); color: var(--bg);
  border-radius: 8px; padding: .5rem .9rem; font-weight: 600;
  text-decoration: none;
}
.try span { color: var(--muted); font-size: .9rem; margin-left: .6rem; }
.blurb { margin: .3rem 0 0; color: var(--muted); font-size: .92rem; }
footer { margin-top: 3rem; color: var(--muted); font-size: .85rem; }
</style>
</head>
<body>
<main>
<h1>Cinegram</h1>
<p class="tagline">Mermaid draws the system; Cinegram plays the story.</p>
<p class="repo">Animated, explorable architecture diagrams. Every page below is one
self-contained HTML file compiled from a <code>.dgm</code> source in
<a href="https://github.com/panset/cinegram">the repository</a>&rsquo;s
<code>examples/</code> directory.</p>
<p class="try"><a href="playground/">Try the playground</a><span>Write your own in
the browser &mdash; the compiler runs in the tab, nothing is uploaded.</span></p>
<ul class="demos">
{{- range .}}
<li>
<a class="title" href="{{.Href}}">{{.Title}}</a><span class="source">{{.Source}}</span>
{{- if .Blurb}}
<p class="blurb">{{.Blurb}}</p>
{{- end}}
</li>
{{- end}}
</ul>
<footer>Generated from the examples by <code>bazel run //site:sync</code>;
these pages are committed, not built &mdash; the deploy workflow only uploads
them.</footer>
</main>
</body>
</html>
`))
