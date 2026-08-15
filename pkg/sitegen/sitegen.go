// Package sitegen turns a folder tree of .dgm files into a browsable static
// site: one player page per document at the same relative path, an index per
// folder, and a single shared copy of the runtime assets — the emitted pages
// use html.Options.AssetsHref instead of inlining 2.8 MB of mermaid each.
//
// The tree is the navigation. Folders give hierarchy; within a folder,
// entries sort by an optional numeric filename prefix (01-intro.dgm) and then
// by name, and the prefix is stripped from everything a reader sees. There is
// deliberately no manifest file: mdBook's SUMMARY.md model silently drops
// files it does not list, and a folder that *is* the list cannot drift from
// itself.
//
// Reading goes through an fs.FS so the generator is testable against a map
// and, by construction, cannot follow a `view … from` reference out of the
// root it was pointed at.
package sitegen

import (
	_ "embed"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/tejaspanse/cinegram/pkg/compile"
	"github.com/tejaspanse/cinegram/pkg/emit/html"
	"github.com/tejaspanse/cinegram/pkg/ir"
	"github.com/tejaspanse/cinegram/pkg/loader"
)

//go:embed assets/site.css
var siteCSS string

// Link is one navigation link carried in every page's header.
type Link struct {
	Name string
	// URL is used as written when absolute (a scheme or leading "/"); a
	// relative URL is taken as site-root-relative and rewritten per page
	// depth, so "playground/" works from every folder.
	URL string
}

// Config is the site-wide presentation. The zero value builds a plain site.
type Config struct {
	// Title names the site in the sidebar and the root index. Empty falls
	// back to "Cinegram".
	Title string
	// Playground enables the per-page "Edit in playground" button and, with
	// Hero, the card on the root index. Resolved like Link.URL.
	Playground string
	// Hero is the copy for the root index's playground card.
	Hero string
	// Links go in every page's header, in order.
	Links []Link
}

// Build renders the whole site into memory, keyed by output-relative path.
// It also returns the warning text of every published document: warnings
// never block a build, but a caller that swallowed them would publish a
// degraded page silently. A document that fails to compile is an error.
func Build(fsys fs.FS, cfg Config) (map[string][]byte, []string, error) {
	if cfg.Title == "" {
		cfg.Title = "Cinegram"
	}

	sources, err := findSources(fsys)
	if err != nil {
		return nil, nil, err
	}
	if len(sources) == 0 {
		return nil, nil, fmt.Errorf("no .dgm files found")
	}

	read := func(p string) ([]byte, error) { return fs.ReadFile(fsys, path.Clean(p)) }

	pages, warnings, err := loadAll(sources, read)
	if err != nil {
		return nil, nil, err
	}

	root, err := buildTree(pages)
	if err != nil {
		return nil, nil, err
	}
	chain(root)

	out := map[string][]byte{".nojekyll": {}}
	for name, content := range html.Assets() {
		out["assets/"+name] = content
	}
	out["assets/site.css"] = []byte(siteCSS)

	if err := renderAll(out, root, root, cfg, read); err != nil {
		return nil, nil, err
	}
	return out, warnings, nil
}

// page is one published document.
type page struct {
	src        string // source path within the tree
	out        string // output path: prefix-stripped, .html
	title      string
	blurb      string
	bundle     *loader.Bundle
	timeline   *ir.Timeline
	prev, next *page
}

// folder is one directory of the tree; entries hold its ordered children.
type folder struct {
	rel     string // "" for the root
	name    string // display name, prefix stripped
	entries []entry
}

type entry struct {
	key  sortKey
	dir  *folder
	page *page
}

// findSources walks the tree for .dgm files, skipping dotfiles and dot-dirs —
// a .git or .cache under the root is nobody's diagram.
func findSources(fsys fs.FS) ([]string, error) {
	var out []string
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(d.Name(), ".") && p != "." {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(p, ".dgm") {
			out = append(out, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

// loadAll loads and compiles every source, then decides which get pages: a
// file another document pulls in via `view … from` is reached by drill-down
// rather than listed — unless nothing published reaches it, in which case it
// publishes after all so no document is stranded invisible. This is the same
// sweep the demos index has always used.
func loadAll(sources []string, read loader.ReadFileFunc) ([]*page, []string, error) {
	type loaded struct {
		src    string
		bundle *loader.Bundle
	}
	var all []loaded
	subViews := map[string]bool{}
	for _, src := range sources {
		bundle, err := loader.Load(src, read)
		if err != nil {
			return nil, nil, fmt.Errorf("loading %s: %w", src, err)
		}
		for _, u := range bundle.Units[1:] {
			subViews[u.Path] = true
		}
		all = append(all, loaded{src: src, bundle: bundle})
	}

	var publish []loaded
	reachable := map[string]bool{}
	take := func(l loaded) {
		publish = append(publish, l)
		for _, u := range l.bundle.Units[1:] {
			reachable[u.Path] = true
		}
	}
	for _, l := range all {
		if !subViews[path.Clean(l.src)] {
			take(l)
		}
	}
	for _, l := range all {
		if subViews[path.Clean(l.src)] && !reachable[path.Clean(l.src)] {
			take(l)
		}
	}

	var pages []*page
	var warnings []string
	for _, l := range publish {
		timeline := compile.CompileBundle(l.bundle)
		if l.bundle.HasErrors() {
			return nil, nil, fmt.Errorf("%s does not compile:\n%s", l.src, bagText(l.bundle))
		}
		if text := bagText(l.bundle); text != "" {
			warnings = append(warnings, text)
		}
		source, err := read(l.src)
		if err != nil {
			return nil, nil, fmt.Errorf("reading %s: %w", l.src, err)
		}
		dir := path.Dir(l.src)
		if dir == "." {
			dir = ""
		}
		name := stripPrefix(strings.TrimSuffix(path.Base(l.src), ".dgm")) + ".html"
		// DefaultTitle falls back to the root view's id, which is derived
		// from the filename — numeric ordering prefix and all. A reader
		// ordering their files with 01- prefixes did not name anything
		// "01-checkout", so the display fallback strips it.
		title := html.DefaultTitle(timeline)
		if title == timeline.Root {
			title = displayName(strings.TrimSuffix(path.Base(l.src), ".dgm"))
		}
		pages = append(pages, &page{
			src:      l.src,
			out:      path.Join(dir, name),
			title:    title,
			blurb:    leadingComment(string(source)),
			bundle:   l.bundle,
			timeline: timeline,
		})
	}
	return pages, warnings, nil
}

// buildTree hangs the pages on their folders and orders every folder's
// entries. Two sources whose stripped names collide in one folder would
// silently overwrite each other's page, so that is an error naming both.
func buildTree(pages []*page) (*folder, error) {
	folders := map[string]*folder{"": {rel: ""}}
	dirOf := func(rel string) *folder {
		if f, ok := folders[rel]; ok {
			return f
		}
		f := &folder{rel: rel, name: displayName(path.Base(rel))}
		folders[rel] = f
		return f
	}
	// Ensure every ancestor exists and is linked exactly once.
	var link func(rel string) *folder
	link = func(rel string) *folder {
		f := dirOf(rel)
		if rel == "" {
			return f
		}
		parentRel := path.Dir(rel)
		if parentRel == "." {
			parentRel = ""
		}
		parent := link(parentRel)
		for _, e := range parent.entries {
			if e.dir == f {
				return f
			}
		}
		parent.entries = append(parent.entries, entry{key: orderKey(path.Base(rel)), dir: f})
		return f
	}

	seen := map[string]string{}
	for _, p := range pages {
		if prev, taken := seen[p.out]; taken {
			return nil, fmt.Errorf("%s and %s would both publish as %s", prev, p.src, p.out)
		}
		seen[p.out] = p.src
		dir := path.Dir(p.src)
		if dir == "." {
			dir = ""
		}
		f := link(dir)
		f.entries = append(f.entries, entry{key: orderKey(path.Base(p.src)), page: p})
	}

	for _, f := range folders {
		es := f.entries
		sort.SliceStable(es, func(i, j int) bool { return es[i].key.less(es[j].key) })
	}
	return folders[""], nil
}

// chain threads prev/next through the pages in depth-first tree order — the
// site reads front to back like a book, and the arrows cross folder edges.
func chain(root *folder) {
	var last *page
	var walk func(f *folder)
	walk = func(f *folder) {
		for _, e := range f.entries {
			if e.page != nil {
				if last != nil {
					last.next = e.page
					e.page.prev = last
				}
				last = e.page
			} else {
				walk(e.dir)
			}
		}
	}
	walk(root)
}

// sortKey orders a folder's entries: an optional numeric prefix first, then
// the name. A prefixed entry sorts before an unprefixed one, so "01-a" beats
// "about" without 1 having to fight the alphabet.
type sortKey struct {
	prefixed bool
	num      int
	name     string
}

func (k sortKey) less(o sortKey) bool {
	if k.prefixed != o.prefixed {
		return k.prefixed
	}
	if k.prefixed && k.num != o.num {
		return k.num < o.num
	}
	return k.name < o.name
}

var prefixRe = regexp.MustCompile(`^(\d+)[-_. ]\s*`)

func orderKey(base string) sortKey {
	if m := prefixRe.FindStringSubmatch(base); m != nil {
		n := 0
		for _, c := range m[1] {
			n = n*10 + int(c-'0')
		}
		return sortKey{prefixed: true, num: n, name: base[len(m[0]):]}
	}
	return sortKey{name: base}
}

func stripPrefix(base string) string {
	return prefixRe.ReplaceAllString(base, "")
}

// displayName is what a folder is called in nav: prefix stripped, separators
// opened into spaces.
func displayName(base string) string {
	s := stripPrefix(base)
	return strings.NewReplacer("-", " ", "_", " ").Replace(s)
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

// leadingComment returns a document's leading %% comment block — the blurb
// its index entry carries. A `%% ---` separator ends the block; a `%%{…}`
// line is a Mermaid directive, not a comment, and ends it the same way.
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
