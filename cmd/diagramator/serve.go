package main

import (
	"bytes"
	"fmt"
	stdhtml "html"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/tejaspanse/diagramator/pkg/compile"
	"github.com/tejaspanse/diagramator/pkg/emit/html"
	"github.com/tejaspanse/diagramator/pkg/loader"
)

// The serve loop exists because the alternative was a three-command ritual:
// compile to a temp file, start a static file server somewhere else, remember
// to re-run both. Serving the page from source means what you reload is what
// the file says now, and the watcher means you usually do not even reload.

const defaultAddr = "127.0.0.1:8731"

// pollInterval is how often the watcher restats the source set. Fast enough to
// feel immediate after a save, slow enough to be free.
const pollInterval = 300 * time.Millisecond

// server renders the page on demand rather than holding a compiled snapshot.
//
// Reading and stat-ing are injected for the same reason pkg/loader injects
// reading: it makes the whole thing testable against a map with no filesystem
// and no clock.
type server struct {
	input string
	read  loader.ReadFileFunc
	stat  func(string) (time.Time, error)
	watch bool
	log   io.Writer

	mu     sync.Mutex
	gen    int64
	stamps map[string]time.Time
}

func newServer(input string, watch bool, log io.Writer) *server {
	return &server{
		input:  input,
		read:   os.ReadFile,
		stat:   statTime,
		watch:  watch,
		log:    log,
		stamps: map[string]time.Time{},
	}
}

func statTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/generation", s.handleGeneration)
	mux.HandleFunc("/", s.handlePage)
	return mux
}

// handleGeneration is what the injected reload script polls. Plain text and
// no-store, because a cached counter is a counter that never changes.
func (s *server) handleGeneration(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	gen := s.gen
	s.mu.Unlock()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprintf(w, "%d", gen)
}

func (s *server) handlePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	page, err := s.page()
	if err != nil {
		// A broken source should show the error in the browser rather than an
		// empty page — and, when watching, the error page carries the reload
		// script too, or fixing the file would leave the browser stranded on
		// the error with nothing listening for the recovery.
		body := []byte("<!doctype html><meta charset=\"utf-8\"><title>diagramator</title><pre>diagramator: " +
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(page)
}

// page compiles the current source into a complete page.
func (s *server) page() ([]byte, error) {
	bundle, err := loader.Load(s.input, s.read)
	if err != nil {
		return nil, err
	}
	timeline := compile.CompileBundle(bundle)

	// Every rebuild re-learns the file set: a newly added `view` should start
	// being watched without restarting the server.
	s.track(bundle)

	if s.log != nil {
		for _, bag := range bundle.Bags() {
			if bag.Len() > 0 {
				fmt.Fprintln(s.log, bag)
			}
		}
	}

	page, err := html.Render(timeline, html.Options{})
	if err != nil {
		return nil, err
	}
	if s.watch {
		page = injectReload(page)
	}
	return page, nil
}

// track records the current mtime of every file in the bundle, seeding the
// watcher so the first poll after a build does not fire spuriously.
func (s *server) track(b *loader.Bundle) {
	next := make(map[string]time.Time, len(b.Units))
	for _, u := range b.Units {
		if t, err := s.stat(u.Path); err == nil {
			next[u.Path] = t
		}
	}
	s.mu.Lock()
	s.stamps = next
	s.mu.Unlock()
}

// poll restats the tracked set once, bumping the generation if anything moved.
// It is a single pass rather than a loop so a test can drive it directly.
func (s *server) poll() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	changed := false
	for path, was := range s.stamps {
		now, err := s.stat(path)
		if err != nil {
			// A file that vanished counts as a change: it is usually an editor
			// writing through a rename, and the next build will say so. The
			// zero stamp remembers that the absence has already fired, so a
			// file that stays missing bumps the generation once, not every
			// poll — and its return reads as one more change.
			if !was.IsZero() {
				s.stamps[path] = time.Time{}
				changed = true
			}
			continue
		}
		if !now.Equal(was) {
			s.stamps[path] = now
			changed = true
		}
	}
	if changed {
		s.gen++
	}
	return changed
}

func (s *server) generation() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gen
}

// watchLoop polls until stop is closed.
func (s *server) watchLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			s.poll()
		}
	}
}

// reloadScript is injected only when serving, so the file `preview -o` writes
// stays byte-identical to what it has always been. It polls rather than holding
// a socket open because a poll is a dozen lines and needs nothing on the
// server that is not already there.
const reloadScript = `<script>
(function () {
  var known = null;
  function check() {
    fetch('generation', { cache: 'no-store' })
      .then(function (r) { return r.text(); })
      .then(function (gen) {
        if (known === null) { known = gen; return; }
        if (gen !== known) location.reload();
      })
      .catch(function () { /* server gone; keep trying */ });
  }
  setInterval(check, 400);
  check();
})();
</script>
`

func injectReload(page []byte) []byte {
	marker := []byte("</body>")
	i := bytes.LastIndex(page, marker)
	if i < 0 {
		return append(page, []byte(reloadScript)...)
	}
	out := make([]byte, 0, len(page)+len(reloadScript))
	out = append(out, page[:i]...)
	out = append(out, []byte(reloadScript)...)
	out = append(out, page[i:]...)
	return out
}

// serve runs the server until interrupted. It returns the listener's address so
// callers that asked for an ephemeral port can find out which one they got.
func (s *server) listen(addr string) (net.Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("cannot listen on %s: %w (is another copy already running?)", addr, err)
	}
	return ln, nil
}

func runServe(input, addr string, watch bool, stderr io.Writer) error {
	s := newServer(input, watch, stderr)

	ln, err := s.listen(addr)
	if err != nil {
		return err
	}
	defer ln.Close()

	// Build once up front: it seeds the watch set and surfaces a broken
	// document immediately rather than on first request.
	if _, err := s.page(); err != nil {
		return err
	}

	stop := make(chan struct{})
	defer close(stop)
	if watch {
		go s.watchLoop(stop)
	}

	fmt.Fprintf(stderr, "serving %s on http://%s/\n", input, ln.Addr())
	if watch {
		fmt.Fprintln(stderr, "watching for changes; the page reloads itself")
	}
	return http.Serve(ln, s.handler())
}
