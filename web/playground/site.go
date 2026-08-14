// Command site assembles the playground into a directory, or serves it.
//
//	bazel run //web/playground:site -- -o _site/playground   # what CI deploys
//	bazel run //web/playground:site -- --serve                # what you develop against
//
// The playground is four hand-written files plus five artifacts that already
// exist elsewhere in the tree: the three browser assets under
// pkg/emit/html/assets (the canonical copies — nothing is duplicated here, so
// nothing can drift), the .wasm this repo builds, and wasm_exec.js taken from
// the very Go SDK that compiled it. Everything is located through the runfiles
// tree, which is what makes `bazel run` the only way to invoke this: the paths
// are Bazel's to know, not this file's to guess.
//
// The examples are resolved one by one from the names in examples.json rather
// than copied wholesale. That is deliberate — it makes the manifest its own
// drift check, since an entry naming a file that no longer exists fails the
// assembly instead of producing a page whose picker 404s.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bazelbuild/rules_go/go/runfiles"
)

// The page's own files, copied to the root of the site.
var pageFiles = []string{"index.html", "playground.js", "playground.css", "examples.json"}

// The browser assets the emitted page uses too. Same bytes, same behaviour: the
// playground's player is not a second implementation.
var browserAssets = []string{"runtime.js", "runtime.css", "mermaid.min.js"}

func main() {
	out := flag.String("o", "", "directory to assemble the site into")
	serve := flag.Bool("serve", false, "serve the site over HTTP instead of writing it")
	addr := flag.String("addr", "127.0.0.1:8732", "address to serve on")
	flag.Parse()

	files, err := assemble()
	if err != nil {
		fmt.Fprintf(os.Stderr, "site: %v\n", err)
		os.Exit(1)
	}

	if *serve {
		if err := listen(*addr, files); err != nil {
			fmt.Fprintf(os.Stderr, "site: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *out == "" {
		fmt.Fprintln(os.Stderr, "site: -o <dir> or --serve")
		os.Exit(2)
	}
	if err := write(resolvePath(*out), files); err != nil {
		fmt.Fprintf(os.Stderr, "site: %v\n", err)
		os.Exit(1)
	}
}

// --- assembly ---------------------------------------------------------------

// assemble reads every file the site is made of into memory, keyed by the path
// it takes on the site. The whole thing is about ten megabytes, most of it the
// .wasm; holding it is simpler than streaming it and lets --serve answer from
// the same map that -o writes.
func assemble() (map[string][]byte, error) {
	rf, err := runfiles.New()
	if err != nil {
		return nil, fmt.Errorf("runfiles: %w (run this through `bazel run`)", err)
	}
	find := locator(rf)

	files := map[string][]byte{}
	add := func(dest, rlocation string) error {
		data, err := find(rlocation)
		if err != nil {
			return err
		}
		files[dest] = data
		return nil
	}

	for _, name := range pageFiles {
		if err := add(name, "web/playground/"+name); err != nil {
			return nil, err
		}
	}
	for _, name := range browserAssets {
		if err := add(name, "pkg/emit/html/assets/"+name); err != nil {
			return nil, err
		}
	}
	if err := add("cinegram.wasm", "web/playground/wasm/cinegram.wasm"); err != nil {
		return nil, err
	}
	// From the same SDK that compiled the .wasm, so the loader and the module
	// cannot be a version apart. Go 1.24 keeps it in lib/wasm/, not misc/wasm/.
	if err := add("wasm_exec.js", "go_sdk/lib/wasm/wasm_exec.js"); err != nil {
		return nil, err
	}

	examples, err := manifest(files["examples.json"])
	if err != nil {
		return nil, err
	}
	for _, ex := range examples {
		for _, name := range ex.Files {
			if err := add("examples/"+name, "examples/"+name); err != nil {
				return nil, fmt.Errorf("example %q: %w", ex.ID, err)
			}
		}
	}
	return files, nil
}

type example struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Entry string   `json:"entry"`
	Files []string `json:"files"`
}

// manifest reads examples.json and checks the invariants the page relies on,
// because a malformed manifest is otherwise a blank picker at runtime rather
// than a failed build.
func manifest(data []byte) ([]example, error) {
	var examples []example
	if err := json.Unmarshal(data, &examples); err != nil {
		return nil, fmt.Errorf("examples.json: %w", err)
	}
	if len(examples) == 0 {
		return nil, fmt.Errorf("examples.json lists no examples")
	}

	seen := map[string]bool{}
	for _, ex := range examples {
		switch {
		case ex.ID == "":
			return nil, fmt.Errorf("examples.json: an entry has no id")
		case seen[ex.ID]:
			return nil, fmt.Errorf("examples.json: %q appears twice", ex.ID)
		case ex.Title == "":
			return nil, fmt.Errorf("examples.json: %q has no title", ex.ID)
		}
		seen[ex.ID] = true

		var carried bool
		for _, name := range ex.Files {
			if name == ex.Entry {
				carried = true
			}
		}
		if !carried {
			return nil, fmt.Errorf("examples.json: %q does not list its own entry %q", ex.ID, ex.Entry)
		}
	}
	return examples, nil
}

// locator returns a reader for workspace-relative paths, tolerating the several
// names the main repository answers to.
//
// Under bzlmod the main module's runfiles live under `_main`, but a repo mapping
// may let it be addressed by its module name instead, and a plain relative path
// works when the process is already standing in the runfiles root. Which of
// those applies is settled once, by the first file that resolves, and every
// later lookup uses it — so a genuine miss (a name in examples.json with no file
// behind it) reports the path that was asked for rather than the last of three
// guesses. An external repo (`go_sdk/…`) is passed through untouched: repo
// mapping resolves those.
func locator(rf *runfiles.Runfiles) func(string) ([]byte, error) {
	prefixes := []string{"", "_main/", "cinegram/"}
	prefix := ""
	settled := false

	read := func(rlocation string) ([]byte, error) {
		where, err := rf.Rlocation(rlocation)
		if err != nil {
			return nil, err
		}
		return os.ReadFile(where)
	}

	return func(rlocation string) ([]byte, error) {
		// An external repo names itself; only the main module's own paths need
		// the prefix, so it must not be pasted onto one of these.
		if strings.HasPrefix(rlocation, "go_sdk/") {
			data, err := read(rlocation)
			if err != nil {
				return nil, fmt.Errorf("locating %s: %w", rlocation, err)
			}
			return data, nil
		}

		if settled {
			data, err := read(prefix + rlocation)
			if err != nil {
				return nil, fmt.Errorf("locating %s: %w", rlocation, err)
			}
			return data, nil
		}

		for _, candidate := range prefixes {
			data, err := read(candidate + rlocation)
			if err == nil {
				prefix, settled = candidate, true
				return data, nil
			}
		}
		return nil, fmt.Errorf("locating %s: not in the runfiles tree under any of %q", rlocation, prefixes)
	}
}

// --- output ------------------------------------------------------------------

func write(dir string, files map[string][]byte) error {
	for _, dest := range sortedKeys(files) {
		to := filepath.Join(dir, filepath.FromSlash(dest))
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(to, files[dest], 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("wrote %d files to %s\n", len(files), dir)
	return nil
}

// resolvePath interprets a relative path against the directory the caller was
// standing in. `bazel run` executes from the runfiles tree, so a relative -o
// would otherwise land somewhere surprising. Same convention as the CLI's own
// resolvePath.
func resolvePath(p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	if wd := os.Getenv("BUILD_WORKING_DIRECTORY"); wd != "" {
		return filepath.Join(wd, p)
	}
	return p
}

// --- dev server ---------------------------------------------------------------

func listen(addr string, files map[string][]byte) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	fmt.Printf("playground on http://%s/\n", ln.Addr())

	// ServeContent picks the content type from the name's extension, and Go's
	// table maps .wasm to application/wasm — which is what
	// WebAssembly.instantiateStreaming insists on. The page has an
	// arrayBuffer() fallback for hosts that get this wrong; this one does not.
	started := time.Now()
	return http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" || name == "." {
			name = "index.html"
		}
		data, ok := files[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		// The site is rebuilt by restarting this command, so nothing here is
		// worth a cache: a stale runtime.js is a confusing half-hour.
		w.Header().Set("Cache-Control", "no-store")
		http.ServeContent(w, r, name, started, bytes.NewReader(data))
	}))
}

func sortedKeys(files map[string][]byte) []string {
	out := make([]string, 0, len(files))
	for k := range files {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
