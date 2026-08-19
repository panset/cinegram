package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/tejaspanse/cinegram/pkg/compile"
	"github.com/tejaspanse/cinegram/pkg/ir"
	"github.com/tejaspanse/cinegram/pkg/loader"
)

// Frame capture is the serve loop plus a browser plus the deep link from Phase
// 6. That composition is the whole trick: the page can already be told to sit
// at one exact millisecond, paused, so a screenshot of it is deterministic
// rather than a race against the animation.
//
// Shelling out to a browser the machine already has keeps the no-third-party
// rule intact. Vendoring a headless renderer to draw a diagram that mermaid.js
// draws would be absurd.

// chromeCandidates are tried in order when CINEGRAM_CHROME is unset.
var chromeCandidates = []string{
	"google-chrome",
	"google-chrome-stable",
	"chromium",
	"chromium-browser",
	"microsoft-edge",
}

var macChromePaths = []string{
	"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
	"/Applications/Chromium.app/Contents/MacOS/Chromium",
	"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
}

// findChrome locates a browser that can take a headless screenshot.
func findChrome() (string, error) {
	if custom := os.Getenv("CINEGRAM_CHROME"); custom != "" {
		if _, err := os.Stat(custom); err != nil {
			return "", fmt.Errorf("CINEGRAM_CHROME is set to %q, which does not exist", custom)
		}
		return custom, nil
	}

	for _, name := range chromeCandidates {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	if runtime.GOOS == "darwin" {
		for _, path := range macChromePaths {
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf(
		"no headless browser found: looked for %s on PATH.\n"+
			"Set CINEGRAM_CHROME to a Chrome or Chromium binary to use one somewhere else",
		strings.Join(chromeCandidates, ", "))
}

// captureOptions is what `frame` was asked for.
type captureOptions struct {
	input    string
	output   string
	at       int
	frames   int
	scenario string
	view     string
	width    int
	height   int
	reel     bool

	// read is how the source is fetched, defaulting to os.ReadFile. Injected
	// for the same reason pkg/loader injects it: a caller holding the source in
	// memory — an editor host, or the MCP server — has no file to point at.
	read loader.ReadFileFunc
}

// readOrFile is the one place the injected reader defaults, so a zero-valued
// options struct still reads the filesystem.
func readOrFile(read loader.ReadFileFunc) loader.ReadFileFunc {
	if read == nil {
		return os.ReadFile
	}
	return read
}

// runCapture renders one moment, or several evenly spaced ones.
func runCapture(opt captureOptions, stderr io.Writer) error {
	chrome, err := findChrome()
	if err != nil {
		return err
	}

	read := readOrFile(opt.read)
	bundle, err := loader.Load(opt.input, read)
	if err != nil {
		return err
	}
	timeline := compile.CompileBundle(bundle)
	if err := reportAll(bundle.Bags(), stderr); err != nil {
		return err
	}

	viewID := opt.view
	if viewID == "" {
		viewID = timeline.Root
	}
	scenarioID, duration, err := pickScenario(timeline, viewID, opt.scenario)
	if err != nil {
		return err
	}

	base, stop, err := servePage(opt.input, read, stderr)
	if err != nil {
		return err
	}
	defer stop()

	shots, err := captureTargets(opt, duration)
	if err != nil {
		return err
	}

	// frame deliberately shoots the full page chrome; --reel swaps in the
	// story layout instead.
	query := ""
	if opt.reel {
		query = "?reel"
	}

	for _, shot := range shots {
		url := frameURL(base, viewID, scenarioID, shot.at, query)
		if err := shoot(chrome, url, shot.path, opt.width, opt.height); err != nil {
			return err
		}
		fmt.Fprintf(stderr, "wrote %s (%s at %dms)\n", shot.path, scenarioID, shot.at)
	}
	return nil
}

// servePage starts the page on a loopback port and returns its base URL along
// with a function that shuts it down.
//
// An ephemeral port, because several captures may be running at once and the
// authoring server may already hold the default. Watching is off: the reload
// script would poll for the length of the capture and its only possible effect
// is to reload a page mid-screenshot.
//
// read is how the server fetches the source on every request. It is a parameter
// rather than always os.ReadFile because the server's own read field is already
// injectable, and a caller holding the source in memory has no file to serve.
func servePage(input string, read loader.ReadFileFunc, stderr io.Writer) (base string, stop func(), err error) {
	srv := newServer(input, false, stderr)
	srv.read = readOrFile(read)
	ln, err := srv.listen("127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}

	httpSrv := &http.Server{Handler: srv.handler()}
	go httpSrv.Serve(ln)

	return fmt.Sprintf("http://%s/", ln.Addr()), func() {
		httpSrv.Close()
		ln.Close()
	}, nil
}

// frameURL addresses one exact moment of one scenario. query selects the page
// mode — "", "?embed" or "?reel" — emitted verbatim.
//
// The query comes before the fragment, which is not a stylistic choice: the
// runtime reads its mode flags from location.search and the moment from the
// hash, and a hash that started before the query would swallow it.
func frameURL(base, viewID, scenarioID string, at int, query string) string {
	return fmt.Sprintf("%s%s#v=%s&s=%s&t=%d", base, query, viewID, scenarioID, at)
}

type shot struct {
	at   int
	path string
}

// framePath names the nth PNG of a sequence. Zero-padded so a lexical sort is
// a numeric one, which is what ffmpeg's `-i frame-%04d.png` expects.
func framePath(dir string, n int) string {
	return filepath.Join(dir, fmt.Sprintf("frame-%04d.png", n))
}

// captureTargets works out what to shoot and where to put it.
func captureTargets(opt captureOptions, duration int) ([]shot, error) {
	if opt.frames <= 1 {
		if opt.output == "" {
			return nil, fmt.Errorf("frame needs -o to say where the PNG goes")
		}
		return []shot{{at: opt.at, path: opt.output}}, nil
	}

	if opt.output == "" {
		return nil, fmt.Errorf("--frames needs -o to name a directory")
	}
	if err := os.MkdirAll(opt.output, 0o755); err != nil {
		return nil, err
	}

	// Evenly spaced across the scenario, endpoints included, computed from the
	// total rather than accumulated so the last frame lands exactly on the end.
	out := make([]shot, 0, opt.frames)
	for i := 0; i < opt.frames; i++ {
		out = append(out, shot{
			at:   duration * i / (opt.frames - 1),
			path: framePath(opt.output, i+1),
		})
	}
	return out, nil
}

// pickScenario resolves the view and scenario to capture, and reports the
// scenario's duration for frame spacing.
func pickScenario(t *ir.Timeline, viewID, scenarioID string) (string, int, error) {
	for i := range t.Views {
		v := &t.Views[i]
		if v.ID != viewID {
			continue
		}
		if len(v.Scenarios) == 0 {
			return "", 0, fmt.Errorf("view %q has no scenarios to capture", viewID)
		}
		if scenarioID == "" {
			return v.Scenarios[0].ID, v.Scenarios[0].Duration, nil
		}
		for _, sc := range v.Scenarios {
			if sc.ID == scenarioID || sc.Name == scenarioID {
				return sc.ID, sc.Duration, nil
			}
		}
		return "", 0, fmt.Errorf("view %q has no scenario %q", viewID, scenarioID)
	}
	return "", 0, fmt.Errorf("no view %q in this document", viewID)
}

// shoot drives the browser for one screenshot.
//
// virtual-time-budget is the load-bearing flag: mermaid renders asynchronously,
// and without it the screenshot lands on an empty stage. It advances the page's
// clock rather than sleeping, so the wait costs far less than it looks.
// Chrome is invoked with the smallest flag set that works. Additions that look
// like obvious hardening — a scratch --user-data-dir, --disable-breakpad,
// --disable-dev-shm-usage — were tried and made it hang past 60s on a machine
// where the plain set captures in about two seconds, so they are deliberately
// not here.
func shoot(chrome, url, out string, width, height int) error {
	args := []string{
		"--headless=new",
		"--disable-gpu",
		"--hide-scrollbars",
		"--virtual-time-budget=6000",
		fmt.Sprintf("--window-size=%d,%d", width, height),
		"--screenshot=" + out,
		url,
	}

	cmd := exec.Command(chrome, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cannot run %s: %w", chrome, err)
	}
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("%s failed: %w\n%s", filepath.Base(chrome), err, tail(stderr.String()))
		}
	case <-time.After(60 * time.Second):
		cmd.Process.Kill()
		return fmt.Errorf("%s did not finish within 60s", filepath.Base(chrome))
	}

	if _, err := os.Stat(out); err != nil {
		return fmt.Errorf("%s reported success but wrote no file to %s", filepath.Base(chrome), out)
	}
	return nil
}

// tail keeps the last few lines of a browser's chatter, which is where the
// actual complaint always is.
func tail(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > 6 {
		lines = lines[len(lines)-6:]
	}
	return strings.Join(lines, "\n")
}
