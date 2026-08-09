package main

import (
	"bytes"
	"image/gif"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
	}, &log)
	if err != nil {
		t.Fatalf("record failed: %v\n%s", err, log.String())
	}

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
