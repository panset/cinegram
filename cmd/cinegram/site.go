package main

import (
	"flag"
	"fmt"
	stdhtml "html"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tejaspanse/cinegram/pkg/sitegen"
)

// cmdSite turns a folder tree of .dgm files into a browsable static site —
// written with -o, or served (and rebuilt as files change) with --serve.
func cmdSite(args []string, stdout, stderr io.Writer) error {
	var serve, watch bool
	var addr, playground, hero, title string
	links := linkList{}
	input, output, err := parseArgsWith("site", args, func(fs *flag.FlagSet) {
		fs.BoolVar(&serve, "serve", false, "serve the site over HTTP instead of writing it")
		fs.StringVar(&addr, "addr", defaultAddr, "address to serve on")
		fs.BoolVar(&watch, "watch", false, "rebuild and reload when anything in the folder changes")
		fs.StringVar(&playground, "playground", "", "playground URL: adds an Edit-in-playground button to every page")
		fs.StringVar(&hero, "hero", "", "copy for the playground card on the root index (needs --playground)")
		fs.StringVar(&title, "title", "", "site title (default: the folder's name)")
		fs.Var(&links, "link", "header link as Name=URL; repeatable")
	})
	if err != nil {
		return err
	}

	info, err := os.Stat(input)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("site takes a folder of .dgm files; %s is a file (preview renders single files)", input)
	}

	cfg := sitegen.Config{
		Title:      title,
		Playground: playground,
		Hero:       hero,
		Links:      links.links,
	}
	if cfg.Title == "" {
		cfg.Title = filepath.Base(input)
	}

	if serve || watch {
		if output != "" {
			fmt.Fprintln(stderr, "cinegram: warning: -o is ignored with --serve/--watch; the site is served, not written")
		}
		return runSiteServe(input, addr, watch, cfg, stderr)
	}
	if output == "" {
		return fmt.Errorf("site needs -o DIR, or --serve")
	}

	pages, warnings, err := sitegen.Build(os.DirFS(input), cfg)
	if err != nil {
		return err
	}
	for _, w := range warnings {
		fmt.Fprintln(stderr, w)
	}
	for rel, content := range pages {
		to := filepath.Join(output, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(to, content, 0o644); err != nil {
			return err
		}
	}
	fmt.Fprintf(stdout, "wrote %d files to %s\n", len(pages), output)
	return nil
}

// linkList collects repeated --link Name=URL flags.
type linkList struct {
	links []sitegen.Link
}

func (l *linkList) String() string { return "" }

func (l *linkList) Set(v string) error {
	name, url, ok := strings.Cut(v, "=")
	if !ok || name == "" || url == "" {
		return fmt.Errorf("--link takes Name=URL, got %q", v)
	}
	l.links = append(l.links, sitegen.Link{Name: name, URL: url})
	return nil
}

// siteServer serves a built site from memory, rebuilding when the watcher
// sees the folder change. It shares preview's reload contract: the injected
// script polls /generation and reloads on a bump.
type siteServer struct {
	root  string
	cfg   sitegen.Config
	watch bool
	log   io.Writer

	mu     sync.Mutex
	pages  map[string][]byte
	err    error
	gen    int64
	stamps map[string]time.Time
}

func runSiteServe(root, addr string, watch bool, cfg sitegen.Config, stderr io.Writer) error {
	s := &siteServer{root: root, cfg: cfg, watch: watch, log: stderr}
	s.rebuild()
	s.mu.Lock()
	err := s.err
	s.mu.Unlock()
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("cannot listen on %s: %w (is another copy already running?)", addr, err)
	}
	defer ln.Close()

	stop := make(chan struct{})
	defer close(stop)
	if watch {
		go func() {
			ticker := time.NewTicker(pollInterval)
			defer ticker.Stop()
			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					if s.changed() {
						s.rebuild()
					}
				}
			}
		}()
	}

	fmt.Fprintf(stderr, "serving %s on http://%s/\n", root, ln.Addr())
	if watch {
		fmt.Fprintln(stderr, "watching for changes; pages reload themselves")
	}
	return http.Serve(ln, s.handler())
}

func (s *siteServer) rebuild() {
	pages, warnings, err := sitegen.Build(os.DirFS(s.root), s.cfg)
	if s.log != nil {
		for _, w := range warnings {
			fmt.Fprintln(s.log, w)
		}
		if err != nil {
			fmt.Fprintln(s.log, "cinegram:", err)
		}
	}
	s.mu.Lock()
	s.pages, s.err = pages, err
	s.gen++
	s.stamps = s.scan()
	s.mu.Unlock()
}

// scan stats every file under the root, so a change to any of them — a .dgm,
// a storyboard image, a new folder — triggers a rebuild.
func (s *siteServer) scan() map[string]time.Time {
	stamps := map[string]time.Time{}
	_ = filepath.WalkDir(s.root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") && p != s.root {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			stamps[p] = info.ModTime()
		}
		return nil
	})
	return stamps
}

func (s *siteServer) changed() bool {
	now := s.scan()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(now) != len(s.stamps) {
		return true
	}
	for p, t := range now {
		if was, ok := s.stamps[p]; !ok || !was.Equal(t) {
			return true
		}
	}
	return false
}

func (s *siteServer) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/generation", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		gen := s.gen
		s.mu.Unlock()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		fmt.Fprintf(w, "%d", gen)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(r.URL.Path, "/")
		if rel == "" || strings.HasSuffix(rel, "/") {
			rel += "index.html"
		}

		s.mu.Lock()
		content, ok := s.pages[rel]
		err := s.err
		s.mu.Unlock()

		// A broken tree shows its error on every page — with the reload
		// script, so fixing the file un-strands the browser.
		if err != nil {
			body := []byte("<!doctype html><meta charset=\"utf-8\"><title>cinegram</title><pre>cinegram: " +
				stdhtml.EscapeString(err.Error()) + "</pre>")
			if s.watch {
				body = injectReload(body)
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(body)
			return
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		if s.watch && strings.HasSuffix(rel, ".html") {
			content = injectReload(content)
		}
		if ctype := mime.TypeByExtension(filepath.Ext(rel)); ctype != "" {
			w.Header().Set("Content-Type", ctype)
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Write(content)
	})
	return mux
}
