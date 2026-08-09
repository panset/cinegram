package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/tejaspanse/cinegram/pkg/compile"
	"github.com/tejaspanse/cinegram/pkg/gifenc"
	"github.com/tejaspanse/cinegram/pkg/loader"
)

// `record` is `frame` run in a loop and then encoded. That is the whole design,
// and it is why it needed so little: the page can already be told to sit at one
// exact millisecond, paused, so a sequence of screenshots is a sequence of
// deterministic stills rather than a race against playback. Nothing here
// depends on how fast the machine is.
//
// GIF is encoded in-process because "put this diagram in a pull request" should
// not require installing anything (see pkg/gifenc). mp4 and webm shell out to
// ffmpeg, which is the right tool and is not worth reimplementing.

// defaultFPS is a compromise: high enough that a particle crossing an edge
// reads as motion, low enough that a ten-second walkthrough is a file someone
// will actually attach to a pull request.
const defaultFPS = 12

// captureWorkers caps how many browsers run at once. Each is a fresh Chrome
// with its own virtual-time budget, which is what makes the frames
// deterministic; the ceiling is there because they are not cheap.
func captureWorkers() int {
	if n := runtime.NumCPU(); n < 4 {
		if n < 1 {
			return 1
		}
		return n
	}
	return 4
}

type recordOptions struct {
	input    string
	output   string
	format   string
	fps      int
	width    int
	height   int
	scenario string
	view     string
}

func cmdRecord(args []string, stdout, stderr io.Writer) error {
	opt := recordOptions{}
	input, output, err := parseArgsWith("record", args, func(fs *flag.FlagSet) {
		fs.StringVar(&opt.format, "format", "", "gif, mp4 or webm (default: from the -o extension)")
		fs.IntVar(&opt.fps, "fps", defaultFPS, "frames per second")
		fs.IntVar(&opt.width, "width", 1280, "viewport width")
		fs.IntVar(&opt.height, "height", 720, "viewport height")
		fs.StringVar(&opt.scenario, "scenario", "", "scenario id or name (default: the first)")
		fs.StringVar(&opt.view, "view", "", "view id (default: the one the document opens on)")
	})
	if err != nil {
		return err
	}
	if output == "" {
		return fmt.Errorf("record needs -o to say where the recording goes")
	}
	opt.input, opt.output = input, output

	if opt.format == "" {
		opt.format = formatFromPath(output)
	}
	switch opt.format {
	case "gif", "mp4", "webm":
	default:
		return fmt.Errorf("unknown --format %q: use gif, mp4 or webm", opt.format)
	}
	if opt.fps < 1 {
		return fmt.Errorf("--fps must be at least 1, got %d", opt.fps)
	}

	// Rounded up unconditionally rather than only for video: yuv420p needs even
	// dimensions and would otherwise fail deep inside ffmpeg, and one extra
	// pixel is invisible in a GIF.
	opt.width, opt.height = even(opt.width), even(opt.height)

	return runRecord(opt, stderr)
}

// formatFromPath infers the container from what the output is called, which is
// what someone typing `-o demo.gif` plainly meant.
func formatFromPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4":
		return "mp4"
	case ".webm":
		return "webm"
	default:
		return "gif"
	}
}

func even(n int) int {
	if n%2 != 0 {
		return n + 1
	}
	return n
}

func runRecord(opt recordOptions, stderr io.Writer) error {
	chrome, err := findChrome()
	if err != nil {
		return err
	}
	// Fail before spending a minute in the browser, not after.
	var ffmpeg string
	if opt.format != "gif" {
		if ffmpeg, err = findFFmpeg(); err != nil {
			return err
		}
	}

	bundle, err := loader.Load(opt.input, os.ReadFile)
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
	if duration <= 0 {
		return fmt.Errorf("scenario %q has no duration to record", scenarioID)
	}

	base, stop, err := servePage(opt.input, stderr)
	if err != nil {
		return err
	}
	defer stop()

	dir, err := os.MkdirTemp("", "cinegram-record-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	times := frameTimes(duration, opt.fps)
	fmt.Fprintf(stderr, "recording %s (%s, %dms) as %d frames at %dfps\n",
		opt.input, scenarioID, duration, len(times), opt.fps)

	paths, err := captureFrames(chrome, base, viewID, scenarioID, dir, times, opt)
	if err != nil {
		return err
	}

	if opt.format == "gif" {
		if err := writeGIF(opt.output, paths, opt.fps); err != nil {
			return err
		}
	} else if err := encodeVideo(ffmpeg, dir, opt); err != nil {
		return err
	}

	info, err := os.Stat(opt.output)
	if err != nil {
		return err
	}
	fmt.Fprintf(stderr, "wrote %s (%d KB)\n", opt.output, info.Size()/1024)
	return nil
}

// frameTimes is the schedule of moments to capture.
//
// Boundaries are computed from the index rather than accumulated, for the same
// reason the compiler tiles a flow's hops that way: with 12fps the interval is
// not a whole number of milliseconds, and adding it up would drift.
func frameTimes(duration, fps int) []int {
	n := duration*fps/1000 + 1
	out := make([]int, n)
	for i := range out {
		at := i * 1000 / fps
		if at > duration {
			at = duration
		}
		out[i] = at
	}
	return out
}

// captureFrames screenshots every moment, several browsers at a time.
//
// Each frame is an independent, fully-specified URL, so the work parallelises
// with nothing shared but the server. The first failure cancels the rest rather
// than letting twenty browsers all time out in turn.
func captureFrames(chrome, base, viewID, scenarioID, dir string, times []int, opt recordOptions) ([]string, error) {
	paths := make([]string, len(times))
	for i := range times {
		paths[i] = framePath(dir, i+1)
	}

	var (
		mu       sync.Mutex
		firstErr error
		next     int
		wg       sync.WaitGroup
	)

	worker := func() {
		defer wg.Done()
		for {
			mu.Lock()
			if firstErr != nil || next >= len(times) {
				mu.Unlock()
				return
			}
			i := next
			next++
			mu.Unlock()

			// `?embed` strips the page furniture, so the recording is the
			// diagram and its narration rather than a screenshot of a toolbar.
			url := frameURL(base, viewID, scenarioID, times[i], true)
			if err := shoot(chrome, url, paths[i], opt.width, opt.height); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("frame %d (%dms): %w", i+1, times[i], err)
				}
				mu.Unlock()
				return
			}
		}
	}

	n := captureWorkers()
	if n > len(times) {
		n = len(times)
	}
	wg.Add(n)
	for i := 0; i < n; i++ {
		go worker()
	}
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	return paths, nil
}

// --- video ------------------------------------------------------------------
//
// ffmpeg is required for mp4 and webm and is deliberately kept out of the GIF
// path entirely: the point of the pure-Go encoder is that `record -o out.gif`
// works on a machine with nothing installed, and a shared code path would erode
// that one refactor at a time.

// findFFmpeg locates an encoder, mirroring findChrome: an explicit override
// first — and a wrong one is a hard error rather than a silent fallback, since
// setting it says you meant a particular binary.
func findFFmpeg() (string, error) {
	if custom := os.Getenv("CINEGRAM_FFMPEG"); custom != "" {
		if _, err := os.Stat(custom); err != nil {
			return "", fmt.Errorf("CINEGRAM_FFMPEG is set to %q, which does not exist", custom)
		}
		return custom, nil
	}
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf(
		"mp4 and webm need ffmpeg, which is not on PATH.\n" +
			"Install it (brew install ffmpeg, apt install ffmpeg), set CINEGRAM_FFMPEG " +
			"to point at one, or record a GIF instead — that needs nothing installed")
}

// ffmpegTimeout is generous: encoding is fast, but a several-hundred-frame webm
// on a busy machine is not, and killing a nearly-finished encode would waste
// the browser time that produced the frames.
const ffmpegTimeout = 120 * time.Second

// encodeVideo turns the captured PNG sequence into a video.
func encodeVideo(ffmpeg, dir string, opt recordOptions) error {
	// -framerate before -i, because it describes the *input* sequence; after
	// -i it would set the output rate and duplicate or drop frames instead.
	args := []string{
		"-y",
		"-framerate", fmt.Sprint(opt.fps),
		"-i", filepath.Join(dir, "frame-%04d.png"),
	}
	switch opt.format {
	case "mp4":
		// yuv420p and faststart are what make the file play in a browser and
		// start before it has finished downloading, which is the whole point of
		// producing an mp4 rather than a GIF.
		args = append(args, "-pix_fmt", "yuv420p", "-movflags", "+faststart")
	case "webm":
		args = append(args, "-c:v", "libvpx-vp9", "-pix_fmt", "yuv420p", "-b:v", "0", "-crf", "32")
	}
	args = append(args, opt.output)

	cmd := exec.Command(ffmpeg, args...)
	var errBuf strings.Builder
	cmd.Stderr = &errBuf

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cannot run %s: %w", ffmpeg, err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("ffmpeg failed: %w\n%s", err, tail(errBuf.String()))
		}
	case <-time.After(ffmpegTimeout):
		cmd.Process.Kill()
		return fmt.Errorf("ffmpeg did not finish within %s", ffmpegTimeout)
	}
	return nil
}

// writeGIF decodes the captured PNGs and encodes them in process.
func writeGIF(out string, paths []string, fps int) error {
	frames := make([]image.Image, 0, len(paths))
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("decoding %s: %w", filepath.Base(p), err)
		}
		frames = append(frames, img)
	}

	// Written through a temp file in the same directory and renamed, so an
	// interrupted encode does not leave a half-written GIF where the previous
	// good one was.
	tmp, err := os.CreateTemp(filepath.Dir(out), ".cinegram-*.gif")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if err := gifenc.Encode(tmp, frames, fps); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), out)
}
