package main

import (
	"bytes"
	"fmt"
	"image/gif"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// TestFrameTimesTileTheScenario covers the schedule `record` captures against.
// Times are computed from the index rather than accumulated, so an fps whose
// interval is not a whole number of milliseconds does not drift, and the last
// frame never runs past the end of the scenario.
func TestFrameTimesTileTheScenario(t *testing.T) {
	cases := []struct {
		duration, fps int
		want          []int
	}{
		{duration: 1000, fps: 10, want: []int{0, 100, 200, 300, 400, 500, 600, 700, 800, 900, 1000}},
		// 12fps is 83.33ms a frame: the case accumulation gets wrong.
		{duration: 500, fps: 12, want: []int{0, 83, 166, 250, 333, 416, 500}},
		{duration: 100, fps: 1, want: []int{0}},
	}

	for _, c := range cases {
		got := frameTimes(c.duration, c.fps)
		if len(got) != len(c.want) {
			t.Fatalf("%dms at %dfps gave %d frames (%v), want %d",
				c.duration, c.fps, len(got), got, len(c.want))
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%dms at %dfps: frame %d at %dms, want %dms",
					c.duration, c.fps, i, got[i], c.want[i])
			}
			if got[i] > c.duration {
				t.Errorf("%dms at %dfps: frame %d at %dms runs past the end",
					c.duration, c.fps, i, got[i])
			}
		}
	}
}

// TestFormatIsInferredFromTheOutputName pins what `-o demo.gif` plainly meant,
// with GIF as the fallback because it is the format that needs nothing
// installed.
func TestFormatIsInferredFromTheOutputName(t *testing.T) {
	cases := map[string]string{
		"out.gif": "gif", "out.mp4": "mp4", "out.webm": "webm",
		"out.MP4": "mp4", "a/b/demo.gif": "gif", "noextension": "gif",
	}
	for path, want := range cases {
		if got := formatFromPath(path); got != want {
			t.Errorf("formatFromPath(%q) = %q, want %q", path, got, want)
		}
	}
}

// TestRecordRejectsBadInvocations keeps the failures immediate and legible
// rather than surfacing after a minute of browser time.
func TestRecordRejectsBadInvocations(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "smoke.dgm")
	if err := os.WriteFile(src, []byte(serveSource), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, log bytes.Buffer
	if err := cmdRecord([]string{src}, &out, &log); err == nil {
		t.Error("record with no -o succeeded; it cannot guess where to write")
	} else if !strings.Contains(err.Error(), "-o") {
		t.Errorf("the error should say -o is missing: %v", err)
	}

	err := cmdRecord([]string{src, "-o", filepath.Join(dir, "x.gif"), "--format", "avi"}, &out, &log)
	if err == nil || !strings.Contains(err.Error(), "avi") {
		t.Errorf("unknown format should be rejected by name, got %v", err)
	}

	err = cmdRecord([]string{src, "-o", filepath.Join(dir, "x.gif"), "--fps", "0"}, &out, &log)
	if err == nil || !strings.Contains(err.Error(), "fps") {
		t.Errorf("0fps should be rejected, got %v", err)
	}
}

// TestEvenDimensions covers the rounding that pre-empts a failure deep inside
// ffmpeg: yuv420p needs even width and height, and one extra pixel is invisible.
func TestEvenDimensions(t *testing.T) {
	for in, want := range map[int]int{1280: 1280, 1281: 1282, 719: 720, 0: 0} {
		if got := even(in); got != want {
			t.Errorf("even(%d) = %d, want %d", in, got, want)
		}
	}
}

// TestFindFFmpegRejectsABadOverride mirrors the Chrome check: an explicit
// setting is validated rather than handed to exec to fail obscurely later.
func TestFindFFmpegRejectsABadOverride(t *testing.T) {
	t.Setenv("CINEGRAM_FFMPEG", "/definitely/not/an/encoder")
	if _, err := findFFmpeg(); err == nil {
		t.Error("expected an error for a CINEGRAM_FFMPEG that does not exist")
	} else if !strings.Contains(err.Error(), "CINEGRAM_FFMPEG") {
		t.Errorf("error should name the variable: %v", err)
	}
}

// TestVideoCommandLine pins the ffmpeg invocation without needing ffmpeg.
//
// Two details in it are easy to get wrong and silently produce a file nobody
// can play: -framerate must come before -i, or it sets the *output* rate and
// ffmpeg duplicates or drops frames to reach it; and yuv420p is what makes the
// result play in a browser at all. A stub that records its argv is enough to
// hold both, and it runs on a machine with no encoder installed.
func TestVideoCommandLine(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stub encoder is a shell script")
	}
	dir := t.TempDir()
	argvFile := filepath.Join(dir, "argv")
	stub := filepath.Join(dir, "fake-ffmpeg")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argvFile + "\ntouch \"${@: -1}\"\n"
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(dir, "out.mp4")
	err := encodeVideo(stub, dir, recordOptions{output: out, format: "mp4", fps: 12})
	if err != nil {
		t.Fatalf("encodeVideo: %v", err)
	}

	raw, err := os.ReadFile(argvFile)
	if err != nil {
		t.Fatalf("the stub was not invoked: %v", err)
	}
	args := strings.Fields(string(raw))

	iAt, rateAt := indexOf(args, "-i"), indexOf(args, "-framerate")
	if rateAt < 0 || iAt < 0 || rateAt > iAt {
		t.Errorf("-framerate must precede -i, got %v", args)
	}
	if indexOf(args, "yuv420p") < 0 {
		t.Errorf("mp4 must be yuv420p to play in a browser, got %v", args)
	}
	if args[len(args)-1] != out {
		t.Errorf("output must be the last argument, got %v", args)
	}
	if got := args[iAt+1]; got != filepath.Join(dir, "frame-%04d.png") {
		t.Errorf("input pattern = %q, want the zero-padded sequence framePath writes", got)
	}

	// webm picks a different codec but keeps the same shape.
	if err := encodeVideo(stub, dir, recordOptions{
		output: filepath.Join(dir, "out.webm"), format: "webm", fps: 12,
	}); err != nil {
		t.Fatalf("encodeVideo(webm): %v", err)
	}
	raw, _ = os.ReadFile(argvFile)
	if !strings.Contains(string(raw), "libvpx-vp9") {
		t.Errorf("webm should ask for VP9, got %s", raw)
	}
}

// TestVideoFailureSurfacesTheEncodersComplaint keeps a broken encode legible:
// ffmpeg says what is wrong on stderr and an exit code alone does not.
func TestVideoFailureSurfacesTheEncodersComplaint(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stub encoder is a shell script")
	}
	dir := t.TempDir()
	stub := filepath.Join(dir, "angry-ffmpeg")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\necho 'Unknown encoder' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := encodeVideo(stub, dir, recordOptions{
		output: filepath.Join(dir, "out.mp4"), format: "mp4", fps: 12,
	})
	if err == nil {
		t.Fatal("a failing encoder was reported as success")
	}
	if !strings.Contains(err.Error(), "Unknown encoder") {
		t.Errorf("the encoder's own complaint is missing from %v", err)
	}
}

// TestCaptureProgressTicksOncePerFrame pins the reporter contract the editor's
// progress bar is built on, without needing a browser: `shoot` only requires
// that the file it asked for exists afterwards, so a shell script that touches
// the path named by --screenshot is a complete stand-in — the same trick
// TestVideoCommandLine plays on ffmpeg.
//
// The properties that matter are that every frame ticks exactly once, that
// `done` never goes backwards (the pool finishes out of order, so reporting the
// index would), and that `total` is the frame count throughout.
func TestCaptureProgressTicksOncePerFrame(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stub browser is a shell script")
	}
	dir := t.TempDir()
	stub := filepath.Join(dir, "fake-chrome")
	script := "#!/bin/sh\n" +
		"for a in \"$@\"; do case \"$a\" in --screenshot=*) : > \"${a#--screenshot=}\";; esac; done\n"
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	times := frameTimes(1000, 12)

	var mu sync.Mutex
	var seen []int
	totals := map[int]bool{}
	report := func(done, total int) {
		mu.Lock()
		defer mu.Unlock()
		seen = append(seen, done)
		totals[total] = true
	}

	var log bytes.Buffer
	paths, err := captureFrames(stub, "http://127.0.0.1:1/", "v0", "s0", dir, times,
		recordOptions{width: 400, height: 300}, report, &log)
	if err != nil {
		t.Fatalf("captureFrames: %v", err)
	}
	if log.Len() != 0 {
		t.Errorf("a run where nothing failed should say nothing: %s", log.String())
	}
	if len(paths) != len(times) {
		t.Fatalf("captured %d paths, want %d", len(paths), len(times))
	}

	if len(seen) != len(times) {
		t.Fatalf("reporter fired %d times for %d frames", len(seen), len(times))
	}
	for i, done := range seen {
		if done != i+1 {
			t.Fatalf("report %d said %d done; the count must rise by one each time, got %v", i, done, seen)
		}
	}
	if len(totals) != 1 || !totals[len(times)] {
		t.Errorf("total should be a constant %d, got %v", len(times), totals)
	}

	// nil is the off switch, and it must not be called into.
	if _, err := captureFrames(stub, "http://127.0.0.1:1/", "v0", "s0", dir, times[:2],
		recordOptions{width: 400, height: 300}, nil, &log); err != nil {
		t.Fatalf("captureFrames without a reporter: %v", err)
	}
}

// TestCaptureRetriesAStarvedBrowser covers the failure that actually happens in
// the field: four browsers contending on a busy machine push one past the
// timeout in `shoot`, while the same frame captures in seconds on its own.
// Without a retry that single straggler discards every frame already captured.
//
// One frame, so the stub's counter needs no locking and the attempt sequence is
// exactly determined.
func TestCaptureRetriesAStarvedBrowser(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stub browser is a shell script")
	}
	dir := t.TempDir()
	counter := filepath.Join(dir, "attempts")

	// Fails the first two invocations, succeeds on the third — the last attempt
	// captureAttempts allows.
	stub := filepath.Join(dir, "flaky-chrome")
	script := "#!/bin/sh\n" +
		"n=$(cat " + counter + " 2>/dev/null || echo 0); n=$((n+1)); echo $n > " + counter + "\n" +
		"if [ \"$n\" -lt 3 ]; then echo 'starved' >&2; exit 1; fi\n" +
		"for a in \"$@\"; do case \"$a\" in --screenshot=*) : > \"${a#--screenshot=}\";; esac; done\n"
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	var ticks int
	var log bytes.Buffer
	paths, err := captureFrames(stub, "http://127.0.0.1:1/", "v0", "s0", dir, []int{0},
		recordOptions{width: 400, height: 300},
		func(done, total int) { ticks++ }, &log)
	if err != nil {
		t.Fatalf("a frame that succeeded on the third attempt still failed the run: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("captured %d paths, want 1", len(paths))
	}

	// The progress bar must count the frame once, not once per attempt.
	if ticks != 1 {
		t.Errorf("reporter fired %d times for one frame; a retry must not double-count", ticks)
	}

	raw, _ := os.ReadFile(counter)
	if got := strings.TrimSpace(string(raw)); got != "3" {
		t.Errorf("the browser ran %s times, want 3", got)
	}

	// A silent retry would make a recording that takes three times as long look
	// like a hang, so the attempts are said out loud.
	if n := strings.Count(log.String(), "failed, retrying"); n != 2 {
		t.Errorf("got %d retry notices, want 2:\n%s", n, log.String())
	}
}

// TestCaptureGivesUpAfterTheAttempts keeps the retry from hiding a real
// failure: a browser that never works still fails the run, and says how hard it
// tried.
func TestCaptureGivesUpAfterTheAttempts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stub browser is a shell script")
	}
	dir := t.TempDir()
	stub := filepath.Join(dir, "broken-chrome")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\necho 'no display' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var log bytes.Buffer
	_, err := captureFrames(stub, "http://127.0.0.1:1/", "v0", "s0", dir, []int{0},
		recordOptions{width: 400, height: 300}, nil, &log)
	if err == nil {
		t.Fatal("a browser that never succeeds was reported as success")
	}
	if !strings.Contains(err.Error(), "3 attempts") {
		t.Errorf("the error should say how many attempts were made: %v", err)
	}
	if !strings.Contains(err.Error(), "no display") {
		t.Errorf("the browser's own complaint is missing from %v", err)
	}
}

// TestProgressReporterIsOptIn keeps --progress purely additive: without it there
// is no reporter at all, so nothing new reaches stderr.
func TestProgressReporterIsOptIn(t *testing.T) {
	var log bytes.Buffer
	if progressReporter(recordOptions{}, &log) != nil {
		t.Error("a reporter was made without --progress")
	}

	r := progressReporter(recordOptions{progress: true}, &log)
	if r == nil {
		t.Fatal("--progress made no reporter")
	}
	r(3, 7)
	if got := log.String(); got != "cinegram-progress capture 3 7\n" {
		t.Errorf("progress line = %q", got)
	}
}

func indexOf(xs []string, want string) int {
	for i, x := range xs {
		if x == want {
			return i
		}
	}
	return -1
}

// TestRecordEndToEnd drives a real browser, so it is opt-in, and it must run
// outside the build sandbox for the same reason TestFrameEndToEnd does:
//
//	CINEGRAM_CHROME="/path/to/Chrome" \
//	  bazel-bin/cmd/cinegram/cinegram_test_/cinegram_test -test.run TestRecord
func TestRecordEndToEnd(t *testing.T) {
	if os.Getenv("CINEGRAM_CHROME") == "" {
		t.Skip("set CINEGRAM_CHROME to run the recording end to end")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "smoke.dgm")
	if err := os.WriteFile(src, []byte(serveSource), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "smoke.gif")

	var log bytes.Buffer
	// Tiny and slow: the scenario is 400ms, so 5fps is three frames, and three
	// browsers is enough to prove the pool without spending a minute on it.
	err := runRecord(recordOptions{
		input: src, output: out, format: "gif", fps: 5, width: 400, height: 300,
		progress: true,
	}, &log)
	if err != nil {
		t.Fatalf("record failed: %v\n%s", err, log.String())
	}

	// The progress protocol, through a real run: one capture line per frame and
	// then exactly one encode line, with the two human-readable lines still
	// around them because --progress is additive.
	assertProgressProtocol(t, log.String(), len(frameTimes(400, 5)))

	f, err := os.Open(out)
	if err != nil {
		t.Fatalf("no GIF written: %v", err)
	}
	defer f.Close()

	got, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("the recording is not a readable GIF: %v", err)
	}
	if want := len(frameTimes(400, 5)); len(got.Image) != want {
		t.Errorf("GIF has %d frames, want %d", len(got.Image), want)
	}
	if b := got.Image[0].Bounds(); b.Dx() != 400 || b.Dy() != 300 {
		t.Errorf("GIF is %dx%d, want the 400x300 viewport", b.Dx(), b.Dy())
	}
	if got.LoopCount != 0 {
		t.Errorf("LoopCount = %d, want a GIF that loops", got.LoopCount)
	}
}

// assertProgressProtocol checks what a host parsing stderr is entitled to
// assume: `capture i n` for every frame, in order, then one `encode`, and the
// pre-existing human lines untouched.
func assertProgressProtocol(t *testing.T, log string, frames int) {
	t.Helper()

	var captures []string
	encodes := 0
	for _, line := range strings.Split(log, "\n") {
		switch {
		case strings.HasPrefix(line, "cinegram-progress capture "):
			captures = append(captures, line)
		case line == "cinegram-progress encode":
			encodes++
		}
	}

	if len(captures) != frames {
		t.Errorf("got %d capture lines, want %d:\n%s", len(captures), frames, log)
	}
	for i, line := range captures {
		if want := fmt.Sprintf("cinegram-progress capture %d %d", i+1, frames); line != want {
			t.Errorf("capture line %d = %q, want %q", i, line, want)
		}
	}
	if encodes != 1 {
		t.Errorf("got %d encode lines, want exactly 1:\n%s", encodes, log)
	}
	if !strings.Contains(log, "frames at 5fps") || !strings.Contains(log, "wrote ") {
		t.Errorf("--progress must not disturb the human-readable lines:\n%s", log)
	}
}

// TestRecordVideoEndToEnd additionally needs ffmpeg, so it skips twice over.
func TestRecordVideoEndToEnd(t *testing.T) {
	if os.Getenv("CINEGRAM_CHROME") == "" {
		t.Skip("set CINEGRAM_CHROME to run the recording end to end")
	}
	if _, err := findFFmpeg(); err != nil {
		t.Skipf("mp4 needs ffmpeg: %v", err)
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "smoke.dgm")
	if err := os.WriteFile(src, []byte(serveSource), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "smoke.mp4")

	var log bytes.Buffer
	err := runRecord(recordOptions{
		input: src, output: out, format: "mp4", fps: 5, width: 400, height: 300,
	}, &log)
	if err != nil {
		t.Fatalf("record failed: %v\n%s", err, log.String())
	}

	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("no mp4 written: %v", err)
	}
	if info.Size() < 1000 {
		t.Errorf("mp4 is %d bytes, which is too small to hold a video", info.Size())
	}
}
