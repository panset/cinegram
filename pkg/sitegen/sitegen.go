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
// Deciding what publishes is separate from rendering it. Discover answers
// "which documents get pages, in what order, under which folders"; Build
// renders that answer as a standalone HTML site. Cinegram's own site renders
// the same answer as Markdown for its Zensical build (see //site), which is
// the reason for the split — two renderers disagreeing about which examples
// exist is a bug nobody would find until a page went missing.
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

// Doc is one document the tree publishes.
type Doc struct {
	// Source is the path within the tree, exactly as found.
	Source string
	// Slug is the published path with no extension: the source's folders as
	// written, and its own filename with any numeric ordering prefix
	// stripped. A renderer appends whatever suffix its pages use.
	Slug string
	// Title is what a reader is shown, and Blurb the one-line summary an
	// index entry carries — the document's leading %% comment block.
	Title string
	Blurb string
	// Text is the source exactly as written. A renderer that shows the .dgm
	// beside its animation uses this rather than reading the file again,
	// which would race an edit and could disagree with the timeline above it.
	Text string
	// Share is the playground's share-link fragment for the whole document,
	// sub-views and storyboard images included.
	Share string

	Bundle   *loader.Bundle
	Timeline *ir.Timeline

	prev, next *Doc
}

// Prev and Next are the documents either side of this one in reading order,
// which runs depth-first through the whole tree and crosses folder edges. Nil
// at the two ends.
func (d *Doc) Prev() *Doc { return d.prev }
func (d *Doc) Next() *Doc { return d.next }

// Group is one folder of the tree.
type Group struct {
	// Path is the folder's path within the tree; "" is the root.
	Path string
	// Name is the display name: prefix stripped, separators opened up.
	Name string
	// Entries are the folder's children in reading order, subfolders and
	// documents interleaved.
	Entries []Entry
}

// Entry is one child of a Group: exactly one of Group or Doc is set.
type Entry struct {
	Group *Group
	Doc   *Doc

	key sortKey
}

// Discover resolves a tree of .dgm files into what a site publishes: the
// folder structure, the documents that get their own page, and the order both
// read in. Returned warnings are the published documents' compile warnings —
// they never block a build, but a caller that swallowed them would publish a
// degraded page silently. A document that fails to compile is an error.
func Discover(fsys fs.FS) (root *Group, docs []*Doc, warnings []string, err error) {
	sources, err := findSources(fsys)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(sources) == 0 {
		return nil, nil, nil, fmt.Errorf("no .dgm files found")
	}

	read := func(p string) ([]byte, error) { return fs.ReadFile(fsys, path.Clean(p)) }

	found, warnings, err := loadAll(sources, read)
	if err != nil {
		return nil, nil, nil, err
	}
	root, err = buildTree(found)
	if err != nil {
		return nil, nil, nil, err
	}
	return root, chain(root), warnings, nil
}

// Build renders the whole site into memory, keyed by output-relative path,
// alongside the warnings Discover reports.
func Build(fsys fs.FS, cfg Config) (map[string][]byte, []string, error) {
	if cfg.Title == "" {
		cfg.Title = "Cinegram"
	}

	root, _, warnings, err := Discover(fsys)
	if err != nil {
		return nil, nil, err
	}

	out := map[string][]byte{".nojekyll": {}}
	for name, content := range html.Assets() {
		out["assets/"+name] = content
	}
	out["assets/site.css"] = []byte(siteCSS)

	if err := renderAll(out, root, root, cfg); err != nil {
		return nil, nil, err
	}
	return out, warnings, nil
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
// publishes after all so no document is stranded invisible.
func loadAll(sources []string, read loader.ReadFileFunc) ([]*Doc, []string, error) {
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

	var docs []*Doc
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
		share, err := encodePlaygroundDoc(l.bundle, read)
		if err != nil {
			return nil, nil, fmt.Errorf("encoding %s for the playground: %w", l.src, err)
		}
		dir := path.Dir(l.src)
		if dir == "." {
			dir = ""
		}
		// DefaultTitle falls back to the root view's id, which is derived
		// from the filename — numeric ordering prefix and all. A reader
		// ordering their files with 01- prefixes did not name anything
		// "01-checkout", so the display fallback strips it.
		title := html.DefaultTitle(timeline)
		if title == timeline.Root {
			title = displayName(strings.TrimSuffix(path.Base(l.src), ".dgm"))
		}
		docs = append(docs, &Doc{
			Source:   l.src,
			Slug:     path.Join(dir, stripPrefix(strings.TrimSuffix(path.Base(l.src), ".dgm"))),
			Title:    title,
			Blurb:    leadingComment(string(source)),
			Text:     string(source),
			Share:    share,
			Bundle:   l.bundle,
			Timeline: timeline,
		})
	}
	return docs, warnings, nil
}

// buildTree hangs the documents on their folders and orders every folder's
// entries. Two sources whose stripped names collide in one folder would
// silently overwrite each other's page, so that is an error naming both.
func buildTree(docs []*Doc) (*Group, error) {
	groups := map[string]*Group{"": {Path: ""}}
	dirOf := func(rel string) *Group {
		if f, ok := groups[rel]; ok {
			return f
		}
		f := &Group{Path: rel, Name: displayName(path.Base(rel))}
		groups[rel] = f
		return f
	}
	// Ensure every ancestor exists and is linked exactly once.
	var link func(rel string) *Group
	link = func(rel string) *Group {
		f := dirOf(rel)
		if rel == "" {
			return f
		}
		parentRel := path.Dir(rel)
		if parentRel == "." {
			parentRel = ""
		}
		parent := link(parentRel)
		for _, e := range parent.Entries {
			if e.Group == f {
				return f
			}
		}
		parent.Entries = append(parent.Entries, Entry{key: orderKey(path.Base(rel)), Group: f})
		return f
	}

	seen := map[string]string{}
	for _, d := range docs {
		if prev, taken := seen[d.Slug]; taken {
			return nil, fmt.Errorf("%s and %s would both publish as %s", prev, d.Source, d.Slug)
		}
		seen[d.Slug] = d.Source
		dir := path.Dir(d.Source)
		if dir == "." {
			dir = ""
		}
		f := link(dir)
		f.Entries = append(f.Entries, Entry{key: orderKey(path.Base(d.Source)), Doc: d})
	}

	for _, f := range groups {
		es := f.Entries
		sort.SliceStable(es, func(i, j int) bool { return es[i].key.less(es[j].key) })
	}
	return groups[""], nil
}

// chain threads prev/next through the documents in depth-first tree order —
// the site reads front to back like a book, and the arrows cross folder
// edges. It returns that same reading order as a flat list.
func chain(root *Group) []*Doc {
	var order []*Doc
	var last *Doc
	var walk func(f *Group)
	walk = func(f *Group) {
		for _, e := range f.Entries {
			if e.Doc != nil {
				if last != nil {
					last.next = e.Doc
					e.Doc.prev = last
				}
				last = e.Doc
				order = append(order, e.Doc)
			} else {
				walk(e.Group)
			}
		}
	}
	walk(root)
	return order
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

// DisplayName is what a folder is called in nav: any numeric ordering prefix
// stripped, separators opened into spaces.
func DisplayName(base string) string { return displayName(base) }

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
