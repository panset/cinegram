// Package gifenc encodes a sequence of frames as an animated GIF.
//
// It exists so that `cinegram record` produces something with nothing
// installed. ffmpeg is the obvious answer for mp4 and webm and this package
// makes no attempt to compete with it there — but "export this diagram as a
// GIF to drop into a pull request" should not require a package manager, and
// the standard library already contains a GIF writer. What it does not contain
// is a quantizer, so that is what this adds: a median-cut palette and a
// nearest-colour mapping onto it.
//
// Two properties are deliberate and worth keeping:
//
//   - One palette for the whole animation. Per-frame palettes cost several
//     times as much to compute and make the output shimmer, because a colour
//     that is quantized one way in frame 3 and another way in frame 4 flickers
//     even where nothing moved. A recording of a diagram is mostly one still
//     picture, so a shared palette is both cheaper and better.
//
//   - Everything here is deterministic. No map iteration reaches the output and
//     ties are broken by index, so the same frames always produce the same
//     bytes and a test can assert on them exactly.
package gifenc

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"io"
	"sort"
)

// MaxColors is the largest palette a GIF can carry.
const MaxColors = 256

// paletteFrames is how many frames the palette is sampled from.
//
// A recording of a diagram is one drawing with things moving over it, so the
// colours in frame 40 are very nearly the colours in frame 1. Sampling a
// spread of frames catches the animation's own palette — a particle, a failure
// mark — without walking every pixel of every frame.
const paletteFrames = 8

// paletteStride is the sampling step within a frame, in pixels. A diagram is
// large flat areas, so every ninth pixel describes it about as well as all of
// them and costs an order of magnitude less.
const paletteStride = 3

// Palette builds one palette of at most n colours for the whole sequence, by
// median cut over a stride-sampled subset of the frames.
func Palette(frames []image.Image, n int) color.Palette {
	if n < 2 {
		n = 2
	}
	if n > MaxColors {
		n = MaxColors
	}

	pixels := sample(frames)
	if len(pixels) == 0 {
		return color.Palette{color.RGBA{A: 255}}
	}

	boxes := medianCut(pixels, n)
	out := make(color.Palette, 0, len(boxes))
	for _, b := range boxes {
		out = append(out, b.average())
	}
	return out
}

// Encode writes frames as an animated GIF playing at fps and looping forever.
func Encode(w io.Writer, frames []image.Image, fps int) error {
	if len(frames) == 0 {
		return fmt.Errorf("gifenc: no frames to encode")
	}
	if fps < 1 {
		return fmt.Errorf("gifenc: fps must be at least 1, got %d", fps)
	}

	pal := Palette(frames, MaxColors)
	cache := newNearest(pal)

	out := &gif.GIF{LoopCount: 0}
	delays := Delays(len(frames), fps)

	// Every frame is written whole. Transparency-based disposal would shrink
	// the file, but it turns a seek into a replay from the last full frame, and
	// a GIF in a pull request is scrubbed as often as it is watched.
	bounds := frames[0].Bounds()
	for i, src := range frames {
		if b := src.Bounds(); b.Dx() != bounds.Dx() || b.Dy() != bounds.Dy() {
			return fmt.Errorf("gifenc: frame %d is %dx%d, but frame 0 is %dx%d",
				i, b.Dx(), b.Dy(), bounds.Dx(), bounds.Dy())
		}
		out.Image = append(out.Image, quantize(src, pal, cache))
		out.Delay = append(out.Delay, delays[i])
	}
	return gif.EncodeAll(w, out)
}

// Delays returns each frame's on-screen time in centiseconds.
//
// GIF measures delay in hundredths of a second, which does not divide evenly
// into most frame rates: at 12fps every frame "should" last 8.33cs, and
// rounding each one independently drifts the animation slower and slower.
// Boundaries are therefore computed from the total and differenced — the same
// trick the compiler uses to tile a flow's hops across its duration — so the
// delays sum to exactly the right length however the division falls.
func Delays(count, fps int) []int {
	out := make([]int, count)
	for i := range out {
		d := (100*(i+1)+fps/2)/fps - (100*i+fps/2)/fps
		// A zero delay means "as fast as possible" to some decoders and "one
		// tenth of a second" to others, so never emit one.
		if d < 1 {
			d = 1
		}
		out[i] = d
	}
	return out
}

// quantize maps one frame onto the shared palette.
func quantize(src image.Image, pal color.Palette, cache *nearest) *image.Paletted {
	b := src.Bounds()
	dst := image.NewPaletted(image.Rect(0, 0, b.Dx(), b.Dy()), pal)

	// A fast path for the common case: the capture pipeline decodes PNGs, which
	// are almost always NRGBA or RGBA, and At() on those allocates an interface
	// per pixel.
	if rgba, ok := asRGBA(src); ok {
		for y := 0; y < b.Dy(); y++ {
			row := rgba.Pix[(y+b.Min.Y-rgba.Rect.Min.Y)*rgba.Stride:]
			for x := 0; x < b.Dx(); x++ {
				p := row[(x+b.Min.X-rgba.Rect.Min.X)*4:]
				dst.Pix[y*dst.Stride+x] = uint8(cache.index(p[0], p[1], p[2]))
			}
		}
		return dst
	}

	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}

// asRGBA returns src as an *image.RGBA-shaped buffer when it is one.
//
// NRGBA and RGBA differ only in premultiplication, which is irrelevant here:
// the captures are fully opaque, so the two are byte-identical for this
// purpose and both can be read directly.
func asRGBA(src image.Image) (*image.RGBA, bool) {
	switch s := src.(type) {
	case *image.RGBA:
		return s, true
	case *image.NRGBA:
		return &image.RGBA{Pix: s.Pix, Stride: s.Stride, Rect: s.Rect}, true
	}
	return nil, false
}

// --- palette construction ---------------------------------------------------

type rgb struct{ r, g, b uint8 }

// sample collects the pixels the palette is built from.
func sample(frames []image.Image) []rgb {
	picked := spread(len(frames), paletteFrames)

	var out []rgb
	for _, i := range picked {
		src := frames[i]
		b := src.Bounds()
		if rgba, ok := asRGBA(src); ok {
			for y := b.Min.Y; y < b.Max.Y; y += paletteStride {
				row := rgba.Pix[(y-rgba.Rect.Min.Y)*rgba.Stride:]
				for x := b.Min.X; x < b.Max.X; x += paletteStride {
					p := row[(x-rgba.Rect.Min.X)*4:]
					out = append(out, rgb{p[0], p[1], p[2]})
				}
			}
			continue
		}
		for y := b.Min.Y; y < b.Max.Y; y += paletteStride {
			for x := b.Min.X; x < b.Max.X; x += paletteStride {
				r, g, bl, _ := src.At(x, y).RGBA()
				out = append(out, rgb{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8)})
			}
		}
	}
	return out
}

// spread picks at most n indices evenly across count, endpoints included.
func spread(count, n int) []int {
	if count <= n {
		out := make([]int, count)
		for i := range out {
			out[i] = i
		}
		return out
	}
	out := make([]int, n)
	for i := range out {
		out[i] = i * (count - 1) / (n - 1)
	}
	return out
}

// box is a region of colour space holding some of the sampled pixels.
type box struct {
	pixels []rgb
	// The bounding cube of the pixels inside, kept so splitting does not
	// recompute it.
	lo, hi rgb
}

func newBox(pixels []rgb) box {
	b := box{pixels: pixels, lo: rgb{255, 255, 255}, hi: rgb{0, 0, 0}}
	for _, p := range pixels {
		if p.r < b.lo.r {
			b.lo.r = p.r
		}
		if p.g < b.lo.g {
			b.lo.g = p.g
		}
		if p.b < b.lo.b {
			b.lo.b = p.b
		}
		if p.r > b.hi.r {
			b.hi.r = p.r
		}
		if p.g > b.hi.g {
			b.hi.g = p.g
		}
		if p.b > b.hi.b {
			b.hi.b = p.b
		}
	}
	return b
}

// longest returns the axis with the widest spread, and that width.
func (b box) longest() (axis int, span int) {
	spans := [3]int{
		int(b.hi.r) - int(b.lo.r),
		int(b.hi.g) - int(b.lo.g),
		int(b.hi.b) - int(b.lo.b),
	}
	axis = 0
	for i := 1; i < 3; i++ {
		if spans[i] > spans[axis] {
			axis = i
		}
	}
	return axis, spans[axis]
}

func (b box) average() color.RGBA {
	if len(b.pixels) == 0 {
		return color.RGBA{A: 255}
	}
	var r, g, bl int
	for _, p := range b.pixels {
		r += int(p.r)
		g += int(p.g)
		bl += int(p.b)
	}
	n := len(b.pixels)
	return color.RGBA{uint8(r / n), uint8(g / n), uint8(bl / n), 255}
}

func channel(p rgb, axis int) uint8 {
	switch axis {
	case 0:
		return p.r
	case 1:
		return p.g
	default:
		return p.b
	}
}

// medianCut splits colour space until there are n boxes or nothing left worth
// splitting.
//
// The box chosen at each step is the one with the widest single-channel spread,
// with ties broken by the earlier index. That tie-break is what makes the
// output reproducible: without it the result would depend on the order the
// scheduler happened to compare equal boxes in, and the encoder's tests could
// only ever assert approximately.
func medianCut(pixels []rgb, n int) []box {
	boxes := []box{newBox(pixels)}

	for len(boxes) < n {
		pick, best := -1, 0
		for i := range boxes {
			if len(boxes[i].pixels) < 2 {
				continue
			}
			if _, span := boxes[i].longest(); span > best {
				pick, best = i, span
			}
		}
		if pick < 0 {
			break // every box is a single colour; more entries would be waste
		}

		axis, _ := boxes[pick].longest()
		pxs := boxes[pick].pixels
		sort.SliceStable(pxs, func(i, j int) bool {
			return channel(pxs[i], axis) < channel(pxs[j], axis)
		})
		mid := splitPoint(pxs, axis)

		boxes[pick] = newBox(pxs[:mid])
		boxes = append(boxes, newBox(pxs[mid:]))
	}
	return boxes
}

// splitPoint is where a sorted box is cut in two.
//
// The median is the starting point, but it is then slid to the nearest place
// the channel value actually changes. Splitting inside a run of identical
// values produces two boxes of the same colour — the caller then spends a
// palette entry on the duplicate and, worse, keeps splitting the half that
// still holds the real variation into ever smaller pieces. A picture that is
// two thirds white and one third black is enough to trigger it.
//
// The box is only ever split when its span is non-zero, so at least one change
// of value exists and the forward scan always terminates before the end.
func splitPoint(pxs []rgb, axis int) int {
	mid := len(pxs) / 2
	for mid > 0 && channel(pxs[mid], axis) == channel(pxs[mid-1], axis) {
		mid--
	}
	if mid > 0 {
		return mid
	}
	for mid = 1; mid < len(pxs); mid++ {
		if channel(pxs[mid], axis) != channel(pxs[0], axis) {
			break
		}
	}
	return mid
}

// --- nearest-colour lookup --------------------------------------------------

// nearest maps a colour to its palette entry, through a coarse cache.
//
// The cache is a 32×32×32 grid over RGB — one entry per five high bits per
// channel. A full search is 256 distance computations and a frame is a million
// pixels; caching by the top bits turns almost all of that into one array
// lookup, and the error it can introduce is bounded by the cell size, which is
// well under what quantizing to 256 colours already costs.
type nearest struct {
	pal   color.Palette
	cache []int16 // -1 until filled
}

const cacheBits = 5
const cacheSide = 1 << cacheBits

func newNearest(pal color.Palette) *nearest {
	n := &nearest{pal: pal, cache: make([]int16, cacheSide*cacheSide*cacheSide)}
	for i := range n.cache {
		n.cache[i] = -1
	}
	return n
}

func (n *nearest) index(r, g, b uint8) int {
	shift := uint(8 - cacheBits)
	key := int(r>>shift)*cacheSide*cacheSide + int(g>>shift)*cacheSide + int(b>>shift)
	if got := n.cache[key]; got >= 0 {
		return int(got)
	}

	best, bestDist := 0, 1<<30
	for i, c := range n.pal {
		pr, pg, pb, _ := c.RGBA()
		dr := int(r) - int(pr>>8)
		dg := int(g) - int(pg>>8)
		db := int(b) - int(pb>>8)
		if d := dr*dr + dg*dg + db*db; d < bestDist {
			best, bestDist = i, d
		}
	}
	n.cache[key] = int16(best)
	return best
}
