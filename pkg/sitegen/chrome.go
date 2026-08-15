package sitegen

import (
	"fmt"
	stdhtml "html"
	"path"
	"strings"

	"github.com/tejaspanse/cinegram/pkg/emit/html"
	"github.com/tejaspanse/cinegram/pkg/loader"
)

// renderAll writes every folder's index and every page into out, walking the
// tree once. cur is the folder being rendered; root stays in hand because the
// sidebar always shows the whole tree.
func renderAll(out map[string][]byte, root, cur *folder, cfg Config, read loader.ReadFileFunc) error {
	if err := renderIndex(out, root, cur, cfg); err != nil {
		return err
	}
	for _, e := range cur.entries {
		if e.dir != nil {
			if err := renderAll(out, root, e.dir, cfg, read); err != nil {
				return err
			}
			continue
		}
		if err := renderPage(out, root, e.page, cfg, read); err != nil {
			return err
		}
	}
	return nil
}

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

func renderPage(out map[string][]byte, root *folder, p *page, cfg Config, read loader.ReadFileFunc) error {
	n := depth(p.out)

	var head strings.Builder
	fmt.Fprintf(&head, "<link rel=\"stylesheet\" href=\"%sassets/site.css\">\n", up(n))

	var header strings.Builder
	header.WriteString("<header class=\"dgm-site-top\">\n")
	header.WriteString(crumbs(p.out, p.title, cfg, n))
	header.WriteString("<div class=\"dgm-site-actions\">\n")
	if cfg.Playground != "" {
		doc, err := encodePlaygroundDoc(p.bundle, read)
		if err != nil {
			return fmt.Errorf("encoding %s for the playground: %w", p.src, err)
		}
		fmt.Fprintf(&header, "<a class=\"dgm-site-edit\" href=\"%s#doc=%s\">Edit in playground</a>\n",
			stdhtml.EscapeString(resolve(cfg.Playground, n)), doc)
	}
	for _, l := range cfg.Links {
		fmt.Fprintf(&header, "<a href=\"%s\">%s</a>\n",
			stdhtml.EscapeString(resolve(l.URL, n)), stdhtml.EscapeString(l.Name))
	}
	header.WriteString("</div>\n</header>\n")

	var footer strings.Builder
	if p.prev != nil || p.next != nil {
		footer.WriteString("<nav class=\"dgm-site-prevnext\">\n")
		if p.prev != nil {
			fmt.Fprintf(&footer, "<a class=\"dgm-site-prev\" href=\"%s\">← %s</a>\n",
				stdhtml.EscapeString(up(n)+p.prev.out), stdhtml.EscapeString(p.prev.title))
		} else {
			footer.WriteString("<span></span>\n")
		}
		if p.next != nil {
			fmt.Fprintf(&footer, "<a class=\"dgm-site-next\" href=\"%s\">%s →</a>\n",
				stdhtml.EscapeString(up(n)+p.next.out), stdhtml.EscapeString(p.next.title))
		}
		footer.WriteString("</nav>\n")
	}

	pageHTML, err := html.Render(p.timeline, html.Options{
		Title:      p.title,
		AssetsHref: up(n) + "assets",
		HeadExtra:  []byte(head.String()),
		Nav:        []byte(sidebar(root, p.out, cfg, n)),
		Header:     []byte(header.String()),
		Footer:     []byte(footer.String()),
	})
	if err != nil {
		return fmt.Errorf("rendering %s: %w", p.src, err)
	}
	out[p.out] = pageHTML
	return nil
}

// renderIndex writes a folder's listing page.
func renderIndex(out map[string][]byte, root, cur *folder, cfg Config) error {
	outPath := path.Join(cur.rel, "index.html")
	if cur.rel == "" {
		outPath = "index.html"
	}
	n := depth(outPath)

	title := cfg.Title
	if cur.rel != "" {
		title = cur.name
	}

	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	fmt.Fprintf(&b, "<title>%s</title>\n", stdhtml.EscapeString(title))
	fmt.Fprintf(&b, "<link rel=\"stylesheet\" href=\"%sassets/runtime.css\">\n", up(n))
	fmt.Fprintf(&b, "<link rel=\"stylesheet\" href=\"%sassets/site.css\">\n", up(n))
	b.WriteString("</head>\n<body class=\"dgm-standalone dgm-sited\">\n")
	b.WriteString(sidebar(root, outPath, cfg, n))
	b.WriteString("<div class=\"dgm-site-main\">\n<header class=\"dgm-site-top\">\n")
	b.WriteString(crumbs(outPath, "", cfg, n))
	b.WriteString("<div class=\"dgm-site-actions\">\n")
	for _, l := range cfg.Links {
		fmt.Fprintf(&b, "<a href=\"%s\">%s</a>\n",
			stdhtml.EscapeString(resolve(l.URL, n)), stdhtml.EscapeString(l.Name))
	}
	b.WriteString("</div>\n</header>\n<main class=\"dgm-site-list\">\n")

	fmt.Fprintf(&b, "<h1>%s</h1>\n", stdhtml.EscapeString(title))
	if cur.rel == "" && cfg.Playground != "" && cfg.Hero != "" {
		fmt.Fprintf(&b, "<p class=\"dgm-site-hero\"><a href=\"%s\">Try the playground</a><span>%s</span></p>\n",
			stdhtml.EscapeString(resolve(cfg.Playground, n)), stdhtml.EscapeString(cfg.Hero))
	}

	b.WriteString("<ul class=\"dgm-site-demos\">\n")
	for _, e := range cur.entries {
		if e.dir != nil {
			fmt.Fprintf(&b, "<li class=\"is-folder\"><a class=\"title\" href=\"%s\">%s/</a></li>\n",
				stdhtml.EscapeString(up(n)+path.Join(e.dir.rel, "index.html")), stdhtml.EscapeString(e.dir.name))
			continue
		}
		fmt.Fprintf(&b, "<li><a class=\"title\" href=\"%s\">%s</a><span class=\"source\">%s</span>\n",
			stdhtml.EscapeString(up(n)+e.page.out), stdhtml.EscapeString(e.page.title),
			stdhtml.EscapeString(path.Base(e.page.src)))
		if e.page.blurb != "" {
			fmt.Fprintf(&b, "<p class=\"blurb\">%s</p>\n", stdhtml.EscapeString(e.page.blurb))
		}
		b.WriteString("</li>\n")
	}
	b.WriteString("</ul>\n</main>\n</div>\n</body>\n</html>\n")

	out[outPath] = []byte(b.String())
	return nil
}

// sidebar renders the whole tree, marking the current output path. Folders
// are <details> — collapsible, and open by default so the shape of the site
// is visible without hunting.
func sidebar(root *folder, current string, cfg Config, n int) string {
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

func writeTree(b *strings.Builder, f *folder, current string, n int) {
	b.WriteString("<ul>\n")
	for _, e := range f.entries {
		if e.dir != nil {
			idx := path.Join(e.dir.rel, "index.html")
			cls := ""
			if current == idx {
				cls = " class=\"is-current\""
			}
			fmt.Fprintf(b, "<li class=\"is-folder\"><details open><summary><a%s href=\"%s\">%s</a></summary>\n",
				cls, stdhtml.EscapeString(up(n)+idx), stdhtml.EscapeString(e.dir.name))
			writeTree(b, e.dir, current, n)
			b.WriteString("</details></li>\n")
			continue
		}
		cls := ""
		if current == e.page.out {
			cls = " class=\"is-current\""
		}
		fmt.Fprintf(b, "<li><a%s href=\"%s\">%s</a></li>\n",
			cls, stdhtml.EscapeString(up(n)+e.page.out), stdhtml.EscapeString(e.page.title))
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
