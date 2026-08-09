package gifenc

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"testing"
)

// gradient builds a frame whose colours vary across both axes, with a moving
// block so consecutive frames genuinely differ.
func gradient(w, h, phase int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 4), uint8(y * 4), uint8(phase * 20), 255})
		}
	}
	for y := 2; y < 8 && y < h; y++ {
		for x := phase; x < phase+6 && x < w; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	return img
}

func frames(n, w, h int) []image.Image {
	out := make([]image.Image, n)
	for i := range out {
		out[i] = gradient(w, h, i)
	}
	return out
}

// TestDelaysTileTheDuration is the property that keeps a recording the length
// it was asked for. GIF delays are whole centiseconds and 12fps is 8.33cs a
// frame; rounding each independently would drift the animation slower with
// every frame.
func TestDelaysTileTheDuration(t *testing.T) {
	for _, fps := range []int{1, 5, 10, 12, 15, 24, 25, 30, 60} {
		got := Delays(fps, fps)
		sum := 0
		for _, d := range got {
			if d < 1 {
				t.Errorf("fps %d: a frame has delay %d; a zero delay means different things to different decoders", fps, d)
			}
			sum += d
		}
		// Exactly fps frames should be exactly one second, whatever the
		// remainder of 100/fps.
		if want := 100; sum != want && fps <= 100 {
			t.Errorf("fps %d: %d frames sum to %dcs, want %dcs", fps, fps, sum, want)
		}
	}

	// And the drift stays bounded over a long run, not just at the one-second
	// mark: 12fps for 10 seconds is 120 frames and 1000 centiseconds.
	if sum := total(Delays(120, 12)); sum != 1000 {
		t.Errorf("120 frames at 12fps sum to %dcs, want 1000", sum)
	}
}

func total(xs []int) int {
	n := 0
	for _, x := range xs {
		n += x
	}
	return n
}

// TestEncodeRoundTrips checks the output is a GIF a decoder agrees with: the
// right number of frames, the right size, looping forever.
func TestEncodeRoundTrips(t *testing.T) {
	var buf bytes.Buffer
	if err := Encode(&buf, frames(9, 40, 24), 12); err != nil {
		t.Fatal(err)
	}

	got, err := gif.DecodeAll(&buf)
	if err != nil {
		t.Fatalf("the encoder produced something image/gif cannot read: %v", err)
	}
	if len(got.Image) != 9 {
		t.Errorf("decoded %d frames, want 9", len(got.Image))
	}
	if len(got.Delay) != len(got.Image) {
		t.Errorf("%d delays for %d frames", len(got.Delay), len(got.Image))
	}
	if got.LoopCount != 0 {
		t.Errorf("LoopCount = %d, want 0 (loop forever)", got.LoopCount)
	}
	for i, img := range got.Image {
		if b := img.Bounds(); b.Dx() != 40 || b.Dy() != 24 {
			t.Errorf("frame %d is %dx%d, want 40x24", i, b.Dx(), b.Dy())
		}
	}
}

// TestOnePaletteForTheWholeAnimation pins the decision that makes a recording
// stop shimmering: a colour that survives quantization must survive it the same
// way in every frame.
func TestOnePaletteForTheWholeAnimation(t *testing.T) {
	var buf bytes.Buffer
	if err := Encode(&buf, frames(6, 40, 24), 10); err != nil {
		t.Fatal(err)
	}
	got, err := gif.DecodeAll(&buf)
	if err != nil {
		t.Fatal(err)
	}

	first := got.Image[0].Palette
	for i, img := range got.Image[1:] {
		if len(img.Palette) != len(first) {
			t.Fatalf("frame %d has %d colours, frame 0 has %d", i+1, len(img.Palette), len(first))
		}
		for j := range first {
			if img.Palette[j] != first[j] {
				t.Fatalf("frame %d entry %d is %v, frame 0 has %v", i+1, j, img.Palette[j], first[j])
			}
		}
	}
}

// TestPaletteIsDeterministic is what lets the rest of these tests assert
// exactly rather than approximately: no map iteration reaches the output and
// equal-width boxes are split in index order.
func TestPaletteIsDeterministic(t *testing.T) {
	src := frames(9, 40, 24)
	first := Palette(src, MaxColors)
	for run := 0; run < 4; run++ {
		got := Palette(src, MaxColors)
		if len(got) != len(first) {
			t.Fatalf("run %d produced %d colours, the first run produced %d", run, len(got), len(first))
		}
		for i := range first {
			if got[i] != first[i] {
				t.Fatalf("run %d entry %d is %v, the first run had %v", run, i, got[i], first[i])
			}
		}
	}
}

// TestPaletteRespectsItsBudget covers both ends: a picture with more colours
// than the budget is cut down to it, and one with fewer is not padded out with
// duplicates.
func TestPaletteRespectsItsBudget(t *testing.T) {
	if got := len(Palette(frames(4, 40, 24), 16)); got > 16 {
		t.Errorf("palette has %d colours, want at most 16", got)
	}

	flat := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			c := color.RGBA{0, 0, 0, 255}
			if x < 4 {
				c = color.RGBA{255, 255, 255, 255}
			}
			flat.Set(x, y, c)
		}
	}
	if got := len(Palette([]image.Image{flat}, MaxColors)); got != 2 {
		t.Errorf("a two-colour picture produced %d palette entries, want 2", got)
	}
}

// TestQuantizationIsCloseEnough guards the nearest-colour cache: bucketing by
// the top five bits per channel must not send a pixel to a wildly wrong entry.
func TestQuantizationIsCloseEnough(t *testing.T) {
	src := gradient(64, 40, 0)
	pal := Palette([]image.Image{src}, MaxColors)
	out := quantize(src, pal, newNearest(pal))

	worst := 0
	for y := 0; y < 40; y++ {
		for x := 0; x < 64; x++ {
			wr, wg, wb, _ := src.At(x, y).RGBA()
			gr, gg, gb, _ := out.At(x, y).RGBA()
			d := sq(int(wr>>8)-int(gr>>8)) + sq(int(wg>>8)-int(gg>>8)) + sq(int(wb>>8)-int(gb>>8))
			if d > worst {
				worst = d
			}
		}
	}
	// 32 per channel is the cache cell size; anything beyond that means the
	// lookup is landing outside the cell it was supposed to approximate.
	if worst > 3*32*32 {
		t.Errorf("worst squared colour error is %d, which is further than the cache's own resolution", worst)
	}
}

func sq(n int) int { return n * n }

// TestEncodeRejectsWhatItCannotEncode keeps the failure legible rather than
// letting image/gif produce a corrupt file from mismatched frames.
func TestEncodeRejectsWhatItCannotEncode(t *testing.T) {
	if err := Encode(&bytes.Buffer{}, nil, 12); err == nil {
		t.Error("encoding no frames succeeded")
	}
	if err := Encode(&bytes.Buffer{}, frames(2, 8, 8), 0); err == nil {
		t.Error("encoding at 0fps succeeded")
	}
	mixed := []image.Image{gradient(16, 16, 0), gradient(20, 16, 1)}
	if err := Encode(&bytes.Buffer{}, mixed, 12); err == nil {
		t.Error("encoding frames of different sizes succeeded")
	}
}
