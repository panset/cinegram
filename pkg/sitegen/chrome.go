package sitegen

import (
	"fmt"
	stdhtml "html"
	"path"
	"strings"

	"github.com/tejaspanse/cinegram/pkg/emit/html"
)

// renderAll writes every folder's index and every page into out, walking the
// tree once. cur is the folder being rendered; root stays in hand because the
// sidebar always shows the whole tree.
func renderAll(out map[string][]byte, root, cur *Group, cfg Config) error {
	if err := renderIndex(out, root, cur, cfg); err != nil {
		return err
	}
	for _, e := range cur.Entries {
		if e.Group != nil {
			if err := renderAll(out, root, e.Group, cfg); err != nil {
				return err
			}
			continue
		}
		if err := renderPage(out, root, e.Doc, cfg); err != nil {
			return err
		}
	}
	return nil
}

// pageOf is a document's output path in an HTML site.
func pageOf(d *Doc) string { return d.Slug + ".html" }

// depth is how many folders deep an output path sits, which is how many
// "../" it takes to reach the site root from it.
func depth(outPath string) int {
	return strings.Count(outPath, "/")
}

func up(n int) string {
	return strings.Repeat("../", n)
}

// resolve turns a configured URL into one usable from a page n levels deep:
// absolute URLs pass through, anything else is site-root-relative.
func resolve(url string, n int) string {
	if url == "" || strings.Contains(url, "://") || strings.HasPrefix(url, "/") || strings.HasPrefix(url, "#") {
		return url
	}
	return up(n) + url
}

func renderPage(out map[string][]byte, root *Group, d *Doc, cfg Config) error {
	outPath := pageOf(d)
	n := depth(outPath)

	var head strings.Builder
	fmt.Fprintf(&head, "<link rel=\"stylesheet\" href=\"%sassets/site.css\">\n", up(n))

	var header strings.Builder
	header.WriteString("<header class=\"dgm-site-top\">\n")
	header.WriteString(crumbs(outPath, d.Title, cfg, n))
	header.WriteString("<div class=\"dgm-site-actions\">\n")
	if cfg.Playground != "" {
		fmt.Fprintf(&header, "<a class=\"dgm-site-edit\" href=\"%s#doc=%s\">Edit in playground</a>\n",
			stdhtml.EscapeString(resolve(cfg.Playground, n)), d.Share)
	}
	for _, l := range cfg.Links {
		fmt.Fprintf(&header, "<a href=\"%s\">%s</a>\n",
			stdhtml.EscapeString(resolve(l.URL, n)), stdhtml.EscapeString(l.Name))
	}
	// Last in the row, and the page's only theme control: Render emits none of
	// its own once it is handed a Nav, precisely so this one is it.
	header.WriteString(html.ThemeToggleHTML())
	header.WriteString("</div>\n</header>\n")

	var footer strings.Builder
	if d.prev != nil || d.next != nil {
		footer.WriteString("<nav class=\"dgm-site-prevnext\">\n")
		if d.prev != nil {
			fmt.Fprintf(&footer, "<a class=\"dgm-site-prev\" href=\"%s\">← %s</a>\n",
				stdhtml.EscapeString(up(n)+pageOf(d.prev)), stdhtml.EscapeString(d.prev.Title))
		} else {
			footer.WriteString("<span></span>\n")
		}
		if d.next != nil {
			fmt.Fprintf(&footer, "<a class=\"dgm-site-next\" href=\"%s\">%s →</a>\n",
				stdhtml.EscapeString(up(n)+pageOf(d.next)), stdhtml.EscapeString(d.next.Title))
		}
		footer.WriteString("</nav>\n")
	}

	pageHTML, err := html.Render(d.Timeline, html.Options{
		Title:      d.Title,
		AssetsHref: up(n) + "assets",
		HeadExtra:  []byte(head.String()),
		Nav:        []byte(sidebar(root, outPath, cfg, n)),
		Header:     []byte(header.String()),
		Footer:     []byte(footer.String()),
	})
	if err != nil {
		return fmt.Errorf("rendering %s: %w", d.Source, err)
	}
	out[outPath] = pageHTML
	return nil
}

// renderIndex writes a folder's listing page.
func renderIndex(out map[string][]byte, root, cur *Group, cfg Config) error {
	outPath := path.Join(cur.Path, "index.html")
	if cur.Path == "" {
		outPath = "index.html"
	}
	n := depth(outPath)

	title := cfg.Title
	if cur.Path != "" {
		title = cur.Name
	}

	var b strings.Builder
	// The same skin attribute html.Render writes, because a listing sits beside
	// the diagram pages and has to be the same product.
	fmt.Fprintf(&b, "<!doctype html>\n<html lang=\"en\" data-dgm-skin=%q>\n<head>\n<meta charset=\"utf-8\">\n", html.Skin)
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	fmt.Fprintf(&b, "<title>%s</title>\n", stdhtml.EscapeString(title))
	fmt.Fprintf(&b, "<link rel=\"stylesheet\" href=\"%sassets/runtime.css\">\n", up(n))
	fmt.Fprintf(&b, "<link rel=\"stylesheet\" href=\"%sassets/site.css\">\n", up(n))
	// A listing builds its own head rather than going through html.Render, so it
	// has to ask for the pre-paint stamp itself — after the stylesheets, whose
	// tokens it decides between, and before anything paints. A page with a
	// diagram on it gets the same three lines from Render.
	b.WriteString(html.ThemeBootScript())
	b.WriteString("</head>\n<body class=\"dgm-standalone dgm-sited\">\n")
	b.WriteString(sidebar(root, outPath, cfg, n))
	b.WriteString("<div class=\"dgm-site-main\">\n<header class=\"dgm-site-top\">\n")
	b.WriteString(crumbs(outPath, "", cfg, n))
	b.WriteString("<div class=\"dgm-site-actions\">\n")
	for _, l := range cfg.Links {
		fmt.Fprintf(&b, "<a href=\"%s\">%s</a>\n",
			stdhtml.EscapeString(resolve(l.URL, n)), stdhtml.EscapeString(l.Name))
	}
	// A listing is a page of this site like any other, and dark or light is a
	// property of the site rather than of the pages with a diagram on them.
	b.WriteString(html.ThemeToggleHTML())
	b.WriteString("</div>\n</header>\n<main class=\"dgm-site-list\">\n")

	fmt.Fprintf(&b, "<h1>%s</h1>\n", stdhtml.EscapeString(title))
	if cur.Path == "" && cfg.Playground != "" && cfg.Hero != "" {
		fmt.Fprintf(&b, "<p class=\"dgm-site-hero\"><a href=\"%s\">Try the playground</a><span>%s</span></p>\n",
			stdhtml.EscapeString(resolve(cfg.Playground, n)), stdhtml.EscapeString(cfg.Hero))
	}

	b.WriteString("<ul class=\"dgm-site-demos\">\n")
	for _, e := range cur.Entries {
		if e.Group != nil {
			fmt.Fprintf(&b, "<li class=\"is-folder\"><a class=\"title\" href=\"%s\">%s/</a>\n",
				stdhtml.EscapeString(up(n)+path.Join(e.Group.Path, "index.html")), stdhtml.EscapeString(e.Group.Name))
			if inside := contents(e.Group, n); inside != "" {
				fmt.Fprintf(&b, "<p class=\"blurb\">%s</p>\n", inside)
			}
			b.WriteString("</li>\n")
			continue
		}
		fmt.Fprintf(&b, "<li><a class=\"title\" href=\"%s\">%s</a><span class=\"source\">%s</span>\n",
			stdhtml.EscapeString(up(n)+pageOf(e.Doc)), stdhtml.EscapeString(e.Doc.Title),
			stdhtml.EscapeString(path.Base(e.Doc.Source)))
		if e.Doc.Blurb != "" {
			fmt.Fprintf(&b, "<p class=\"blurb\">%s</p>\n", stdhtml.EscapeString(e.Doc.Blurb))
		}
		b.WriteString("</li>\n")
	}
	b.WriteString("</ul>\n</main>\n</div>\n")
	// The only script a listing carries, and the only reason it carries one:
	// the theme control above is wired by the runtime, which is the single
	// implementation every cinegram surface shares. mermaid stays out of it —
	// there is no diagram on this page — and the runtime is already in the
	// reader's cache from the last page that had one.
	fmt.Fprintf(&b, "<script src=\"%sassets/runtime.js\"></script>\n", up(n))
	b.WriteString("</body>\n</html>\n")

	out[outPath] = []byte(b.String())
	return nil
}

// contentsShown caps how much of a folder its parent's listing spells out. A
// folder entry is a way in, not a second copy of the tree the sidebar already
// carries whole.
const contentsShown = 6

// contents links a folder's direct children under its entry in the parent
// index. Without it a site whose root holds only folders greets a reader with
// nothing but folder names, where a flat one showed every demo and its blurb —
// this keeps any demo one click from the landing page either way. Blurbs stay
// on the folder's own index: a listing of listings has to stay scannable.
func contents(f *Group, n int) string {
	var parts []string
	for _, e := range f.Entries {
		if len(parts) == contentsShown {
			return strings.Join(parts, " · ") +
				fmt.Sprintf(" · and %d more", len(f.Entries)-contentsShown)
		}
		href, text := "", ""
		if e.Group != nil {
			href, text = up(n)+path.Join(e.Group.Path, "index.html"), e.Group.Name+"/"
		} else {
			href, text = up(n)+pageOf(e.Doc), e.Doc.Title
		}
		parts = append(parts, fmt.Sprintf("<a href=\"%s\">%s</a>",
			stdhtml.EscapeString(href), stdhtml.EscapeString(text)))
	}
	return strings.Join(parts, " · ")
}

// sidebar renders the whole tree, marking the current output path. Folders
// are <details> — collapsible, and open by default so the shape of the site
// is visible without hunting.
func sidebar(root *Group, current string, cfg Config, n int) string {
	var b strings.Builder
	b.WriteString("<nav class=\"dgm-site-nav\">\n")
	class := ""
	if current == "index.html" {
		class = " class=\"is-current\""
	}
	fmt.Fprintf(&b, "<p class=\"dgm-site-name\"><a%s href=\"%sindex.html\">%s</a></p>\n",
		class, up(n), stdhtml.EscapeString(cfg.Title))
	writeTree(&b, root, current, n)
	b.WriteString("</nav>\n")
	return b.String()
}

func writeTree(b *strings.Builder, f *Group, current string, n int) {
	b.WriteString("<ul>\n")
	for _, e := range f.Entries {
		if e.Group != nil {
			idx := path.Join(e.Group.Path, "index.html")
			cls := ""
			if current == idx {
				cls = " class=\"is-current\""
			}
			fmt.Fprintf(b, "<li class=\"is-folder\"><details open><summary><a%s href=\"%s\">%s</a></summary>\n",
				cls, stdhtml.EscapeString(up(n)+idx), stdhtml.EscapeString(e.Group.Name))
			writeTree(b, e.Group, current, n)
			b.WriteString("</details></li>\n")
			continue
		}
		cls := ""
		if current == pageOf(e.Doc) {
			cls = " class=\"is-current\""
		}
		fmt.Fprintf(b, "<li><a%s href=\"%s\">%s</a></li>\n",
			cls, stdhtml.EscapeString(up(n)+pageOf(e.Doc)), stdhtml.EscapeString(e.Doc.Title))
	}
	b.WriteString("</ul>\n")
}

// crumbs is the trail from the site name down to here. pageTitle is empty on
// an index page — the folder itself is the destination.
func crumbs(outPath, pageTitle string, cfg Config, n int) string {
	var b strings.Builder
	b.WriteString("<nav class=\"dgm-site-crumbs\">\n")
	fmt.Fprintf(&b, "<a href=\"%sindex.html\">%s</a>\n", up(n), stdhtml.EscapeString(cfg.Title))

	dir := path.Dir(outPath)
	var segs []string
	if dir != "." && dir != "" {
		segs = strings.Split(dir, "/")
	}
	for i, seg := range segs {
		rel := strings.Join(segs[:i+1], "/")
		fmt.Fprintf(&b, "<span>/</span> <a href=\"%s\">%s</a>\n",
			stdhtml.EscapeString(up(n)+rel+"/index.html"), stdhtml.EscapeString(displayName(seg)))
	}
	if pageTitle != "" {
		fmt.Fprintf(&b, "<span>/</span> <span class=\"is-here\">%s</span>\n", stdhtml.EscapeString(pageTitle))
	}
	b.WriteString("</nav>\n")
	return b.String()
}
