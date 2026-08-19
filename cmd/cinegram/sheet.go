package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/tejaspanse/cinegram/pkg/compile"
	"github.com/tejaspanse/cinegram/pkg/ir"
	"github.com/tejaspanse/cinegram/pkg/loader"
)

// A contact sheet is what `record` refuses to be: one still image, not a
// motion picture. It exists for the reader who cannot watch anything — an
// agent, a pull-request diff, a printed page — and it answers the only
// question such a reader has of an animation, "what happens in it", in a
// single image read rather than a scrub through twelve frames a second.
//
// The whole trick is that the labels come free. The `?embed` page already
// hides the toolbar and the step list but keeps the caption, so photographing
// one moment per step yields a cell that says which step it is in the
// document's own words. Nothing here draws text; Go has no business rendering
// a typeface that the browser is already rendering correctly.

// sheetGutter is the neutral band between cells and around the outside. It is
// not decoration: without it two adjacent diagrams on the same background read
// as one wide diagram, and the reader has to guess where a cell ends.
const sheetGutter = 8

// sheetGutterColor is deliberately mid-grey — darker than a light page, lighter
// than a dark one — so the grid is visible whichever theme the browser
// screenshotted.
var sheetGutterColor = color.RGBA{0x9a, 0x9a, 0xa2, 0xff}

// sheetMaxSide is where a sheet stops being one image a reader can take in.
// Past it the cells are thumbnails and the captions are unreadable, so the run
// says so rather than silently producing something useless. It warns instead of
// failing: an author who asked for forty steps at 1280px wide may well have
// meant it.
const sheetMaxSide = 8000

// sheetColsCap is the widest auto grid. Four across keeps a cell wide enough to
// read at the size the whole sheet has to be viewed at; beyond that a
// twenty-step scenario would be one long strip.
const sheetColsCap = 4

type sheetOptions struct {
	input    string
	output   string
	manifest string
	scenario string
	view     string
	width    int
	height   int
	cols     int

	// read is how the source is fetched, defaulting to os.ReadFile, so a caller
	// holding the source in memory can produce a sheet without a file on disk.
	read loader.ReadFileFunc
}

// parseSheetOptions is the parse-and-validate half of cmdSheet, separated so
// the flag rules are unit-testable without a browser, exactly as
// parseRecordOptions is.
func parseSheetOptions(args []string) (sheetOptions, error) {
	opt := sheetOptions{}
	input, output, err := parseArgsWith("sheet", args, func(fs *flag.FlagSet) {
		fs.StringVar(&opt.manifest, "manifest", "",
			"also write the cell map as JSON to this file")
		fs.StringVar(&opt.scenario, "scenario", "", "scenario id or name (default: the first)")
		fs.StringVar(&opt.view, "view", "", "view id (default: the one the document opens on)")
		fs.IntVar(&opt.width, "width", 900, "viewport width, which is one cell's width")
		fs.IntVar(&opt.height, "height", 600, "viewport height, which is one cell's height")
		fs.IntVar(&opt.cols, "cols", 0, "columns in the grid (default: chosen from the step count)")
	})
	if err != nil {
		return opt, err
	}
	if output == "" {
		return opt, fmt.Errorf("sheet needs -o to say where the PNG goes")
	}
	opt.input, opt.output = input, output
	opt.manifest = resolvePath(opt.manifest)

	if opt.cols < 0 {
		return opt, fmt.Errorf("--cols cannot be negative, got %d", opt.cols)
	}
	if opt.width < 1 || opt.height < 1 {
		return opt, fmt.Errorf("--width and --height must be positive, got %dx%d", opt.width, opt.height)
	}
	return opt, nil
}

func cmdSheet(args []string, stdout, stderr io.Writer) error {
	opt, err := parseSheetOptions(args)
	if err != nil {
		return err
	}
	_, err = runSheet(opt, stderr)
	return err
}

// sheetAt is the moment a step is photographed: one millisecond before it ends.
//
// Not its start, which shows the step's flows before they have gone anywhere
// and its `set` values before they were written; and not its end either,
// because the runtime resolves "which step is this" as the last step whose
// start is at or before the clock, so End belongs to the step that follows and
// would caption the cell with the wrong name. End-1 is the last instant that is
// unambiguously this step, with everything it did already done.
//
// A step of one or two milliseconds is a boundary marker rather than a beat;
// backing off from its end would land outside it, so it is shot at End.
func sheetAt(st ir.Step) int {
	if st.End-st.Start <= 2 {
		return st.End
	}
	return st.End - 1
}

// sheetLayout is the grid, computed once and then used twice: to composite the
// image and to describe it in the manifest. One function so the two can never
// disagree about where a cell is — a manifest that misplaces a cell is worse
// than no manifest, because it is believed.
type sheetLayout struct {
	cols   int
	rows   int
	gutter int
	cellW  int
	cellH  int
	imageW int
	imageH int
}

// sheetCols is the auto grid: as square as the step count allows, capped so a
// cell stays wide enough to read.
func sheetCols(n int) int {
	if n < 1 {
		return 1
	}
	cols := int(math.Ceil(math.Sqrt(float64(n))))
	if cols > sheetColsCap {
		cols = sheetColsCap
	}
	return cols
}

// layoutSheet places n cells of the given size. cols of zero means auto.
func layoutSheet(n, cols, cellW, cellH int) sheetLayout {
	if n < 1 {
		n = 1
	}
	if cols < 1 {
		cols = sheetCols(n)
	}
	if cols > n {
		cols = n
	}
	rows := (n + cols - 1) / cols
	return sheetLayout{
		cols: cols, rows: rows, gutter: sheetGutter,
		cellW: cellW, cellH: cellH,
		// The gutter runs around the outside as well as between, so an edge
		// cell is bounded on all four sides and the sheet reads as a grid
		// rather than as four pictures that happen to touch.
		imageW: sheetGutter + cols*(cellW+sheetGutter),
		imageH: sheetGutter + rows*(cellH+sheetGutter),
	}
}

// rect is where the ith cell lives, filling left to right, top to bottom —
// reading order, because the cells are a sequence in time.
func (l sheetLayout) rect(i int) image.Rectangle {
	col, row := i%l.cols, i/l.cols
	x := l.gutter + col*(l.cellW+l.gutter)
	y := l.gutter + row*(l.cellH+l.gutter)
	return image.Rect(x, y, x+l.cellW, y+l.cellH)
}

// compositeSheet lays the captured stills out on one canvas.
//
// draw.Src rather than draw.Over: a screenshot is opaque and the cells must not
// blend with the gutter, and Src is also the cheaper operator. Each cell is
// clipped to its own rectangle, so a screenshot that came back larger than
// asked for crops instead of spilling into its neighbour.
func compositeSheet(l sheetLayout, cells []image.Image) *image.RGBA {
	canvas := image.NewRGBA(image.Rect(0, 0, l.imageW, l.imageH))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(sheetGutterColor), image.Point{}, draw.Src)
	for i, cell := range cells {
		if cell == nil {
			continue
		}
		draw.Draw(canvas, l.rect(i), cell, cell.Bounds().Min, draw.Src)
	}
	return canvas
}

// --- manifest ----------------------------------------------------------------

// sheetManifest is the map from pixels back to the document: which rectangle is
// which step, and what moment it shows. It is what makes the sheet addressable
// — a reader that spots something wrong in the third cell can name the step and
// re-shoot exactly that moment with `frame`.
//
// It is always computed, whether or not --manifest asks for it on disk, because
// the caller that most needs it (a tool driving cinegram) wants the bytes, not
// a file.
type sheetManifest struct {
	File     string      `json:"file"`
	View     string      `json:"view"`
	Scenario string      `json:"scenario"`
	Duration int         `json:"duration"`
	Cols     int         `json:"cols"`
	Rows     int         `json:"rows"`
	Gutter   int         `json:"gutter"`
	Cell     sheetSize   `json:"cell"`
	Image    sheetSize   `json:"image"`
	Cells    []sheetCell `json:"cells"`
}

type sheetSize struct {
	W int `json:"w"`
	H int `json:"h"`
}

// sheetCell describes one photograph. Every field is present even when empty:
// a consumer parsing this should not have to distinguish "no description" from
// "the key moved".
type sheetCell struct {
	Index  int    `json:"index"`
	Step   string `json:"step"`
	Name   string `json:"name"`
	Desc   string `json:"desc"`
	At     int    `json:"at"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// buildManifest pairs the steps with the layout that photographed them.
func buildManifest(input, viewID string, sc *ir.Scenario, l sheetLayout) sheetManifest {
	m := sheetManifest{
		File:     input,
		View:     viewID,
		Scenario: sc.ID,
		Duration: sc.Duration,
		Cols:     l.cols,
		Rows:     l.rows,
		Gutter:   l.gutter,
		Cell:     sheetSize{W: l.cellW, H: l.cellH},
		Image:    sheetSize{W: l.imageW, H: l.imageH},
		Cells:    make([]sheetCell, 0, len(sc.Steps)),
	}
	for i, st := range sc.Steps {
		r := l.rect(i)
		m.Cells = append(m.Cells, sheetCell{
			Index: i, Step: st.ID, Name: st.Name, Desc: st.Desc,
			At: sheetAt(st), Start: st.Start, End: st.End,
			X: r.Min.X, Y: r.Min.Y, Width: r.Dx(), Height: r.Dy(),
		})
	}
	return m
}

// --- the run ------------------------------------------------------------------

// runSheet shoots one frame per step and composites them.
//
// It returns the manifest bytes rather than only writing them, so a caller that
// wants the cell map — the thing that makes the image navigable — does not have
// to write a file and read it back.
func runSheet(opt sheetOptions, stderr io.Writer) ([]byte, error) {
	chrome, err := findChrome()
	if err != nil {
		return nil, err
	}

	read := readOrFile(opt.read)
	bundle, err := loader.Load(opt.input, read)
	if err != nil {
		return nil, err
	}
	timeline := compile.CompileBundle(bundle)
	if err := reportAll(bundle.Bags(), stderr); err != nil {
		return nil, err
	}

	viewID := opt.view
	if viewID == "" {
		viewID = timeline.Root
	}
	scenarioID, _, err := pickScenario(timeline, viewID, opt.scenario)
	if err != nil {
		return nil, err
	}
	sc := findScenario(timeline, viewID, scenarioID)
	if sc == nil || len(sc.Steps) == 0 {
		return nil, fmt.Errorf("scenario %q has no steps to lay out", scenarioID)
	}

	layout := layoutSheet(len(sc.Steps), opt.cols, opt.width, opt.height)
	if layout.imageW > sheetMaxSide || layout.imageH > sheetMaxSide {
		fmt.Fprintf(stderr,
			"cinegram: warning: the sheet is %dx%d, past %dpx a side; the captions will be hard to read (try --cols, or a smaller --width/--height)\n",
			layout.imageW, layout.imageH, sheetMaxSide)
	}

	base, stop, err := servePage(opt.input, read, stderr)
	if err != nil {
		return nil, err
	}
	defer stop()

	dir, err := os.MkdirTemp("", "cinegram-sheet-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	shots := make([]shot, len(sc.Steps))
	for i, st := range sc.Steps {
		shots[i] = shot{at: sheetAt(st), path: framePath(dir, i+1)}
	}

	fmt.Fprintf(stderr, "sheeting %s (%s): %d steps as a %dx%d grid of %dx%d cells\n",
		opt.input, scenarioID, len(shots), layout.cols, layout.rows, layout.cellW, layout.cellH)

	// `?embed` is what makes the labels free: it hides the toolbar and the step
	// list but keeps the caption, so each cell is captioned by the document.
	if err := captureFrames(chrome, base, viewID, scenarioID, "?embed", shots,
		opt.width, opt.height, nil, stderr); err != nil {
		return nil, err
	}

	cells, err := decodeCells(shotPaths(shots))
	if err != nil {
		return nil, err
	}
	if err := writePNG(opt.output, compositeSheet(layout, cells)); err != nil {
		return nil, err
	}

	encoded, err := json.MarshalIndent(buildManifest(opt.input, viewID, sc, layout), "", "  ")
	if err != nil {
		return nil, err
	}
	encoded = append(encoded, '\n')
	if opt.manifest != "" {
		if err := os.WriteFile(opt.manifest, encoded, 0o644); err != nil {
			return nil, err
		}
		fmt.Fprintf(stderr, "wrote %s\n", opt.manifest)
	}

	info, err := os.Stat(opt.output)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(stderr, "wrote %s (%dx%d, %d KB)\n",
		opt.output, layout.imageW, layout.imageH, info.Size()/1024)
	return encoded, nil
}

// findScenario returns the scenario itself, which pickScenario deliberately
// does not: `frame` and `record` need only an id and a duration, while a sheet
// needs the steps.
func findScenario(t *ir.Timeline, viewID, scenarioID string) *ir.Scenario {
	for i := range t.Views {
		v := &t.Views[i]
		if v.ID != viewID {
			continue
		}
		for j := range v.Scenarios {
			if v.Scenarios[j].ID == scenarioID {
				return &v.Scenarios[j]
			}
		}
	}
	return nil
}

func decodeCells(paths []string) ([]image.Image, error) {
	out := make([]image.Image, 0, len(paths))
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			return nil, err
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("decoding %s: %w", filepath.Base(p), err)
		}
		out = append(out, img)
	}
	return out, nil
}

// writePNG encodes through a temp file in the same directory and renames, so an
// interrupted run does not leave a half-written sheet where the previous good
// one was — the same rule writeGIF follows.
func writePNG(out string, img image.Image) error {
	tmp, err := os.CreateTemp(filepath.Dir(out), ".cinegram-*.png")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if err := png.Encode(tmp, img); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), out)
}
