package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tejaspanse/cinegram/pkg/ir"
)

// TestSheetAtIsTheLastInstantOfTheStep pins the capture moment, which is the
// one judgement call in the whole command.
//
// End would caption the cell with the *next* step's name, because the runtime
// resolves "which step is this" as the last step whose start is at or before
// the clock. Start would photograph the step before it had done anything. So
// End-1 — except for a step so short that End-1 is not inside it at all.
func TestSheetAtIsTheLastInstantOfTheStep(t *testing.T) {
	cases := []struct {
		name       string
		start, end int
		want       int
	}{
		{"a normal step", 0, 600, 599},
		{"one that does not start at zero", 600, 1300, 1299},
		{"3ms is long enough to back off from", 10, 13, 12},
		{"2ms is not", 10, 12, 12},
		{"1ms is not", 10, 11, 11},
		{"a zero-length marker", 10, 10, 10},
	}
	for _, c := range cases {
		got := sheetAt(ir.Step{Start: c.start, End: c.end})
		if got != c.want {
			t.Errorf("%s: sheetAt(%d..%d) = %d, want %d", c.name, c.start, c.end, got, c.want)
		}
		if got < c.start || got > c.end {
			t.Errorf("%s: sheetAt(%d..%d) = %d, which is outside the step",
				c.name, c.start, c.end, got)
		}
	}
}

// TestSheetGridShape covers the auto-columns formula and the explicit override.
// Auto is as square as the count allows, capped at four across so a cell stays
// wide enough to read at the size the whole sheet has to be viewed at.
func TestSheetGridShape(t *testing.T) {
	cases := []struct {
		n, cols      int
		wantC, wantR int
	}{
		{n: 1, cols: 0, wantC: 1, wantR: 1},
		{n: 2, cols: 0, wantC: 2, wantR: 1},
		{n: 4, cols: 0, wantC: 2, wantR: 2},
		{n: 5, cols: 0, wantC: 3, wantR: 2},
		{n: 9, cols: 0, wantC: 3, wantR: 3},
		{n: 10, cols: 0, wantC: 4, wantR: 3},
		{n: 20, cols: 0, wantC: 4, wantR: 5},
		// Explicit wins, including a single strip and a single column.
		{n: 6, cols: 6, wantC: 6, wantR: 1},
		{n: 6, cols: 1, wantC: 1, wantR: 6},
		{n: 6, cols: 4, wantC: 4, wantR: 2},
		// More columns than cells is not an empty column, it is a shorter row.
		{n: 3, cols: 8, wantC: 3, wantR: 1},
	}
	for _, c := range cases {
		l := layoutSheet(c.n, c.cols, 100, 50)
		if l.cols != c.wantC || l.rows != c.wantR {
			t.Errorf("layoutSheet(n=%d, cols=%d) = %dx%d, want %dx%d",
				c.n, c.cols, l.cols, l.rows, c.wantC, c.wantR)
		}
		if l.cols*l.rows < c.n {
			t.Errorf("layoutSheet(n=%d, cols=%d) has %d places for %d cells",
				c.n, c.cols, l.cols*l.rows, c.n)
		}
	}
}

// TestSheetLayoutRects checks the geometry itself: gutters between the cells and
// around the outside, reading order, no overlap, and an image exactly big
// enough to hold the grid.
func TestSheetLayoutRects(t *testing.T) {
	l := layoutSheet(5, 0, 100, 50) // 3 columns, 2 rows
	if l.imageW != sheetGutter+3*(100+sheetGutter) || l.imageH != sheetGutter+2*(50+sheetGutter) {
		t.Fatalf("image is %dx%d, want the grid plus its gutters", l.imageW, l.imageH)
	}

	bounds := image.Rect(0, 0, l.imageW, l.imageH)
	var rects []image.Rectangle
	for i := 0; i < 5; i++ {
		r := l.rect(i)
		if !r.In(bounds) {
			t.Errorf("cell %d at %v escapes the %v image", i, r, bounds)
		}
		if r.Dx() != 100 || r.Dy() != 50 {
			t.Errorf("cell %d is %dx%d, want 100x50", i, r.Dx(), r.Dy())
		}
		for j, other := range rects {
			if r.Overlaps(other) {
				t.Errorf("cell %d at %v overlaps cell %d at %v", i, r, j, other)
			}
		}
		rects = append(rects, r)
	}

	// Reading order: the second cell is to the right of the first, and the
	// fourth begins a new row back at the left margin.
	if rects[1].Min.Y != rects[0].Min.Y || rects[1].Min.X <= rects[0].Min.X {
		t.Errorf("cell 1 at %v should sit to the right of cell 0 at %v", rects[1], rects[0])
	}
	if rects[3].Min.X != rects[0].Min.X || rects[3].Min.Y <= rects[0].Min.Y {
		t.Errorf("cell 3 at %v should begin a new row under cell 0 at %v", rects[3], rects[0])
	}
	if rects[0].Min.X != sheetGutter || rects[0].Min.Y != sheetGutter {
		t.Errorf("the first cell is at %v; the gutter runs around the outside too", rects[0].Min)
	}
}

// TestCompositeSheetPlacesCellsWhereTheManifestSays is the property that makes
// the manifest worth having: a reader who trusts a rectangle in the JSON must
// find that cell's pixels there. Solid-colour stand-ins make it checkable
// without a browser.
func TestCompositeSheetPlacesCellsWhereTheManifestSays(t *testing.T) {
	colors := []color.RGBA{
		{0xff, 0x00, 0x00, 0xff},
		{0x00, 0xff, 0x00, 0xff},
		{0x00, 0x00, 0xff, 0xff},
		{0xff, 0xff, 0x00, 0xff},
		{0x00, 0xff, 0xff, 0xff},
	}
	cells := make([]image.Image, len(colors))
	for i, c := range colors {
		cell := image.NewRGBA(image.Rect(0, 0, 20, 10))
		for y := 0; y < 10; y++ {
			for x := 0; x < 20; x++ {
				cell.SetRGBA(x, y, c)
			}
		}
		cells[i] = cell
	}

	sc := &ir.Scenario{ID: "s0", Duration: 500, Steps: make([]ir.Step, len(colors))}
	for i := range sc.Steps {
		sc.Steps[i] = ir.Step{ID: "step", Start: i * 100, End: (i + 1) * 100}
	}

	l := layoutSheet(len(cells), 0, 20, 10)
	canvas := compositeSheet(l, cells)
	m := buildManifest("x.dgm", "v0", sc, l)

	if b := canvas.Bounds(); b.Dx() != m.Image.W || b.Dy() != m.Image.H {
		t.Fatalf("canvas is %v but the manifest says %dx%d", b, m.Image.W, m.Image.H)
	}

	for i, cell := range m.Cells {
		// Every corner and the middle, so a cell drawn at the wrong offset or
		// at the wrong size cannot pass.
		points := []image.Point{
			{cell.X, cell.Y},
			{cell.X + cell.Width - 1, cell.Y},
			{cell.X, cell.Y + cell.Height - 1},
			{cell.X + cell.Width - 1, cell.Y + cell.Height - 1},
			{cell.X + cell.Width/2, cell.Y + cell.Height/2},
		}
		for _, p := range points {
			if got := canvas.RGBAAt(p.X, p.Y); got != colors[i] {
				t.Errorf("cell %d: pixel at %v is %v, want %v", i, p, got, colors[i])
			}
		}
		// Just outside the cell is gutter, not the neighbour's colour bleeding.
		if cell.X >= sheetGutter {
			if got := canvas.RGBAAt(cell.X-1, cell.Y); got != sheetGutterColor {
				t.Errorf("cell %d: the pixel left of it is %v, want the gutter %v",
					i, got, sheetGutterColor)
			}
		}
	}

	// The trailing gap of a ragged last row stays gutter, so a five-cell sheet
	// in a 3x2 grid does not end with a stale rectangle.
	last := m.Cells[len(m.Cells)-1]
	if got := canvas.RGBAAt(last.X+last.Width+sheetGutter+1, last.Y+1); got != sheetGutterColor {
		t.Errorf("the empty place in the last row is %v, want the gutter %v", got, sheetGutterColor)
	}
}

// TestCompositeClipsAnOversizedCell keeps a screenshot that came back larger
// than asked for from spilling into its neighbour.
func TestCompositeClipsAnOversizedCell(t *testing.T) {
	big := image.NewRGBA(image.Rect(0, 0, 60, 40))
	red := color.RGBA{0xff, 0, 0, 0xff}
	for y := 0; y < 40; y++ {
		for x := 0; x < 60; x++ {
			big.SetRGBA(x, y, red)
		}
	}
	l := layoutSheet(2, 2, 20, 10)
	canvas := compositeSheet(l, []image.Image{big, nil})

	r := l.rect(1)
	if got := canvas.RGBAAt(r.Min.X, r.Min.Y); got != sheetGutterColor {
		t.Errorf("the oversized first cell bled into the second: %v", got)
	}
}

// TestParseSheetOptions covers the flag surface, including the hoisting that
// lets the input file be typed before its flags.
func TestParseSheetOptions(t *testing.T) {
	if _, err := parseSheetOptions([]string{"a.dgm"}); err == nil {
		t.Error("sheet with no -o succeeded; it cannot guess where to write")
	} else if !strings.Contains(err.Error(), "-o") {
		t.Errorf("the error should say -o is missing: %v", err)
	}

	if _, err := parseSheetOptions([]string{"-o", "s.png"}); err == nil {
		t.Error("sheet with no input file succeeded")
	}

	opt, err := parseSheetOptions([]string{
		"a.dgm", "-o", "s.png", "--cols", "3", "--manifest", "m.json",
		"--scenario", "outage", "--view", "v1", "--width", "800", "--height", "500",
	})
	if err != nil {
		t.Fatalf("parseSheetOptions: %v", err)
	}
	if opt.input != "a.dgm" || opt.output != "s.png" {
		t.Errorf("input/output = %q/%q", opt.input, opt.output)
	}
	if opt.cols != 3 || opt.manifest != "m.json" {
		t.Errorf("--cols/--manifest = %d/%q", opt.cols, opt.manifest)
	}
	if opt.scenario != "outage" || opt.view != "v1" {
		t.Errorf("--scenario/--view = %q/%q", opt.scenario, opt.view)
	}
	if opt.width != 800 || opt.height != 500 {
		t.Errorf("--width/--height = %dx%d", opt.width, opt.height)
	}

	// Defaults: auto columns, and no manifest unless one is asked for.
	opt, err = parseSheetOptions([]string{"a.dgm", "-o", "s.png"})
	if err != nil {
		t.Fatalf("parseSheetOptions: %v", err)
	}
	if opt.cols != 0 {
		t.Errorf("--cols defaults to %d, want 0 meaning auto", opt.cols)
	}
	if opt.manifest != "" {
		t.Errorf("--manifest defaults to %q, want empty", opt.manifest)
	}

	if _, err := parseSheetOptions([]string{"a.dgm", "-o", "s.png", "--cols", "-2"}); err == nil {
		t.Error("a negative --cols was accepted")
	}
	if _, err := parseSheetOptions([]string{"a.dgm", "-o", "s.png", "--width", "0"}); err == nil {
		t.Error("a zero --width was accepted")
	}
}

// TestSheetRejectsBadInvocations keeps the failure immediate rather than
// arriving after a browser has started.
func TestSheetRejectsBadInvocations(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "smoke.dgm")
	if err := os.WriteFile(src, []byte(serveSource), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, log bytes.Buffer
	if err := cmdSheet([]string{src}, &out, &log); err == nil {
		t.Error("sheet with no -o succeeded")
	}
}

// TestSheetManifestShape pins the wire format a caller parses: every key is
// present even when its value is empty, and the rectangles are the ones the
// layout used.
func TestSheetManifestShape(t *testing.T) {
	sc := &ir.Scenario{
		ID: "happy-path", Duration: 1200,
		Steps: []ir.Step{
			{ID: "submit", Name: "Shopper submits", Desc: "the order goes out", Start: 0, End: 600},
			{ID: "confirm", Start: 600, End: 1200}, // no name, no desc
		},
	}
	l := layoutSheet(len(sc.Steps), 0, 100, 50)
	encoded, err := json.MarshalIndent(buildManifest("checkout.dgm", "v0", sc, l), "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("the manifest is not valid JSON: %v", err)
	}
	for _, key := range []string{"file", "view", "scenario", "duration", "cols", "rows",
		"gutter", "cell", "image", "cells"} {
		if _, ok := got[key]; !ok {
			t.Errorf("the manifest has no %q key: %s", key, encoded)
		}
	}
	if got["file"] != "checkout.dgm" || got["scenario"] != "happy-path" {
		t.Errorf("file/scenario = %v/%v", got["file"], got["scenario"])
	}
	if size, ok := got["cell"].(map[string]any); !ok || size["w"] != 100.0 || size["h"] != 50.0 {
		t.Errorf("cell size = %v, want 100x50", got["cell"])
	}

	cells, ok := got["cells"].([]any)
	if !ok || len(cells) != len(sc.Steps) {
		t.Fatalf("cells = %v, want one per step", got["cells"])
	}
	for i, raw := range cells {
		cell, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("cell %d is not an object: %v", i, raw)
		}
		for _, key := range []string{"index", "step", "name", "desc", "at",
			"start", "end", "x", "y", "width", "height"} {
			if _, present := cell[key]; !present {
				t.Errorf("cell %d has no %q key; empty values must still be carried: %v",
					i, key, cell)
			}
		}
		if cell["index"] != float64(i) {
			t.Errorf("cell %d says index %v", i, cell["index"])
		}
		if cell["step"] != sc.Steps[i].ID {
			t.Errorf("cell %d names step %v, want %q", i, cell["step"], sc.Steps[i].ID)
		}
		if cell["at"] != float64(sheetAt(sc.Steps[i])) {
			t.Errorf("cell %d shoots at %v, want %d", i, cell["at"], sheetAt(sc.Steps[i]))
		}
		r := l.rect(i)
		if cell["x"] != float64(r.Min.X) || cell["y"] != float64(r.Min.Y) {
			t.Errorf("cell %d is at %v,%v but the layout puts it at %v",
				i, cell["x"], cell["y"], r.Min)
		}
	}
}

// TestSheetEndToEnd drives a real browser, so it is opt-in, exactly as the
// record and frame end-to-end tests are:
//
//	CINEGRAM_CHROME="/path/to/Chrome" go test ./cmd/cinegram -run TestSheetEndToEnd
func TestSheetEndToEnd(t *testing.T) {
	if os.Getenv("CINEGRAM_CHROME") == "" {
		t.Skip("set CINEGRAM_CHROME to build a sheet end to end")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "smoke.dgm")
	if err := os.WriteFile(src, []byte(serveSource), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "smoke.png")
	manifest := filepath.Join(dir, "smoke.json")

	var log bytes.Buffer
	encoded, err := runSheet(sheetOptions{
		input: src, output: out, manifest: manifest, width: 400, height: 300,
	}, &log)
	if err != nil {
		t.Fatalf("sheet failed: %v\n%s", err, log.String())
	}

	onDisk, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatalf("no manifest written: %v", err)
	}
	if !bytes.Equal(onDisk, encoded) {
		t.Error("the returned manifest and the written one differ")
	}

	var m sheetManifest
	if err := json.Unmarshal(encoded, &m); err != nil {
		t.Fatalf("the manifest does not parse: %v", err)
	}
	if len(m.Cells) != 1 {
		t.Errorf("the one-step scenario made %d cells", len(m.Cells))
	}

	f, err := os.Open(out)
	if err != nil {
		t.Fatalf("no sheet written: %v", err)
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatalf("the sheet is not a readable image: %v", err)
	}
	if cfg.Width != m.Image.W || cfg.Height != m.Image.H {
		t.Errorf("the sheet is %dx%d but the manifest says %dx%d",
			cfg.Width, cfg.Height, m.Image.W, m.Image.H)
	}
}
