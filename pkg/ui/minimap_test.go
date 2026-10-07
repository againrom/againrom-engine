package ui

import (
	"image"
	"image/color"
	"os"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/render/terrain"
)

// solidFog and mapFog are the two shapes a test's fog reader takes: every
// cell answering one state, or a lookup table keyed by cell — composeMinimap
// asks nothing about how fog is stored, only that fog(col,row) answers a
// state, which is exactly fogAt's own signature.
func solidFog(state uint8) func(int, int) uint8 {
	return func(int, int) uint8 { return state }
}

func mapFog(m map[image.Point]uint8, otherwise uint8) func(int, int) uint8 {
	return func(col, row int) uint8 {
		if s, ok := m[image.Pt(col, row)]; ok {
			return s
		}
		return otherwise
	}
}

// TestComposeMinimapIsPureAndTakesNoWindow is spec AC-10: the same
// arguments, called twice with nothing else running, produce byte-identical
// pixels. Nothing here opens a window, builds a Viewer or touches the GPU —
// composeMinimap is called directly, which is the whole of what "no window,
// no engine" means for a function reachable from a test at all.
func TestComposeMinimapIsPureAndTakesNoWindow(t *testing.T) {
	colours := []color.RGBA{
		{R: 10, G: 20, B: 30, A: 255}, {R: 40, G: 50, B: 60, A: 255},
		{R: 70, G: 80, B: 90, A: 255}, {R: 100, G: 110, B: 120, A: 255},
	}
	marks := []minimapMark{{Cell: image.Pt(0, 0), Local: true}, {Cell: image.Pt(1, 1), Local: false}}

	img1 := composeMinimap(colours, 2, 2, solidFog(FogVisible), marks, image.Rectangle{}, image.Pt(40, 40))
	img2 := composeMinimap(colours, 2, 2, solidFog(FogVisible), marks, image.Rectangle{}, image.Pt(40, 40))

	if img1.Bounds() != img2.Bounds() {
		t.Fatalf("two identical calls produced different bounds: %v vs %v", img1.Bounds(), img2.Bounds())
	}
	if !reflect.DeepEqual(img1.Pix, img2.Pix) {
		t.Fatal("two identical calls to composeMinimap produced different pixels")
	}
}

// TestComposeMinimapScaleFillsTheConstrainingAxis is the owner's own report:
// "the map itself for some reason doesn't occupy at least the full height or
// the full width, whichever would fill first" — replacing the old "whole
// pixel multiple" contract this test asserted before the fix, which is
// itself an instance of the bug: at cols=4, rows=3, box=(100,40), the OLD
// rule (largest whole-pixel block scale, floor(40/3)=13) drew 52x39 — a
// full pixel short of the box's own 40-tall edge on the very axis the box
// was tightest on.
//
// The new rule is rational, not a whole-pixel block: it reaches the box's
// own extent EXACTLY on whichever axis is tighter, and stays proportional
// (same ratio, so the same aspect as the source map) on the other, which is
// not necessarily a whole multiple of cols or rows any more.
//
// WITNESSED BY REVERTING: this test was run against the pre-fix
// composeMinimap (floor(box/cols) block scale), went red on the height
// assertion below (39, not 40), and was restored.
func TestComposeMinimapScaleFillsTheConstrainingAxis(t *testing.T) {
	cols, rows := 4, 3
	colours := make([]color.RGBA, cols*rows)
	for i := range colours {
		colours[i] = color.RGBA{R: uint8(i), G: uint8(i), B: uint8(i), A: 255}
	}
	box := image.Pt(100, 40) // Y is the tighter axis: box.Y*cols=160 < box.X*rows=300

	img := composeMinimap(colours, cols, rows, solidFog(FogVisible), nil, image.Rectangle{}, box)
	b := img.Bounds()

	if got := b.Dy(); got != box.Y {
		t.Errorf("height = %d, want the box's own full %d — the tighter axis must fill exactly", got, box.Y)
	}
	wantWidth := cols * box.Y / rows // the same ratio applied to the other axis's own cell count
	if got := b.Dx(); got != wantWidth {
		t.Errorf("width = %d, want %d (cols*box.Y/rows, the same scale kept on the other axis)", got, wantWidth)
	}
	if b.Dx() > box.X || b.Dy() > box.Y {
		t.Fatalf("bounds %v exceed the box %v", b, box)
	}
}

func TestComposeMinimapSamplesWhenLargerThanTheBox(t *testing.T) {
	const cols, rows = 503, 17
	colours := make([]color.RGBA, cols*rows)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			colours[row*cols+col] = color.RGBA{R: uint8(col), G: uint8(row), B: 0x80, A: 255}
		}
	}
	box := image.Pt(100, 100)

	img := composeMinimap(colours, cols, rows, solidFog(FogVisible), nil, image.Rectangle{}, box)
	b := img.Bounds()

	if b.Dx() > box.X || b.Dy() > box.Y {
		t.Fatalf("sampled bounds %v exceed the box %v", b, box)
	}
	if b.Dx() >= cols || b.Dy() >= rows {
		t.Fatalf("bounds %v are not smaller than the %dx%d source map — this did not sample at all", b, cols, rows)
	}

	if b.Dx() != box.X {
		t.Fatalf("bounds = %v, want full limiting width %d", b, box.X)
	}
	if wantY := rows * box.X / cols; b.Dy() != wantY {
		t.Fatalf("height = %d, want proportional %d", b.Dy(), wantY)
	}

	for oy := 0; oy < b.Dy(); oy++ {
		for ox := 0; ox < b.Dx(); ox++ {
			col, row := ox*cols/box.X, oy*cols/box.X
			want := colours[row*cols+col]
			if got := img.RGBAAt(ox, oy); got != want {
				t.Fatalf("pixel (%d,%d) = %v, want the EXACT source cell (%d,%d)'s colour %v — "+
					"sampling must pick that cell precisely, not blend or drift to a neighbour",
					ox, oy, got, col, row, want)
			}
		}
	}
}

func TestDownscaledMinimapKeepsMarkerAndViewOnTheFilledPicture(t *testing.T) {
	const cols, rows = 503, 503
	box := image.Pt(100, 100)
	colours := make([]color.RGBA, cols*rows)
	for i := range colours {
		colours[i] = color.RGBA{R: 13, G: 29, B: 47, A: 255}
	}
	mark := minimapMark{Cell: image.Pt(250, 249), Local: true}
	view := image.Rect(501, 500, 503, 503)
	img := composeMinimap(colours, cols, rows, solidFog(FogVisible), []minimapMark{mark}, view, box)
	if img.Bounds().Size() != box {
		t.Fatalf("large map bounds = %v, want full %v", img.Bounds(), box)
	}
	if got := img.RGBAAt(99, 99); got != minimapViewColor {
		t.Fatalf("last viewport pixel = %v, want visible border %v", got, minimapViewColor)
	}
	if got := img.RGBAAt(49, 49); got != minimapLocalColor {
		t.Fatalf("downscaled local marker = %v, want %v", got, minimapLocalColor)
	}
	edge := composeMinimap(colours, cols, rows, solidFog(FogVisible), []minimapMark{{Cell: image.Pt(502, 502), Local: true}}, image.Rectangle{}, box)
	if got := edge.RGBAAt(99, 99); got != minimapLocalColor {
		t.Fatalf("last-cell marker = %v, want %v", got, minimapLocalColor)
	}
}

func TestDownscaledMinimapPixelAndClickNameTheSameCell(t *testing.T) {
	const cols, rows = 503, 17
	v, err := NewViewer("mm-rational-hit", grid(cols, rows), &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	layoutViewport(v, DefaultWindowW, DefaultWindowH)
	g, ok := v.minimapGeometry()
	if !ok || g.Num >= g.Den {
		t.Fatalf("expected reduced minimap geometry, got %+v", g)
	}
	colours := make([]color.RGBA, cols*rows)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			colours[row*cols+col] = color.RGBA{R: uint8(col), G: uint8(col >> 8), B: uint8(row), A: 255}
		}
	}
	img := composeMinimap(colours, cols, rows, solidFog(FogVisible), nil, image.Rectangle{}, g.extent)
	if img.Bounds().Size() != g.Content.Size() {
		t.Fatalf("drawn content %v differs from hit rectangle %v", img.Bounds(), g.Content)
	}
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			cell, ok := v.minimapCellAt(g.Content.Min.X+x, g.Content.Min.Y+y)
			if !ok {
				t.Fatalf("drawn pixel (%d,%d) has no clickable cell", x, y)
			}
			if got, want := img.RGBAAt(x, y), colours[cell.Y*cols+cell.X]; got != want {
				t.Fatalf("drawn pixel (%d,%d) is %v but click names cell %v with %v", x, y, got, cell, want)
			}
		}
	}
}

func TestComposeMinimapFogRule(t *testing.T) {
	colourA := color.RGBA{R: 200, G: 100, B: 40, A: 255}
	colourB := color.RGBA{R: 20, G: 220, B: 180, A: 255}
	colours := []color.RGBA{colourA, colourB} // cell (0,0)=A, cell (1,0)=B
	box := image.Pt(2, 1)                     // scale exactly 1: one pixel per cell

	visible := composeMinimap(colours, 2, 1, solidFog(FogVisible), nil, image.Rectangle{}, box)
	if got := visible.RGBAAt(0, 0); got != colourA {
		t.Fatalf("both cells visible: pixel(0,0) = %v, want %v", got, colourA)
	}
	if got := visible.RGBAAt(1, 0); got != colourB {
		t.Fatalf("both cells visible: pixel(1,0) = %v, want %v", got, colourB)
	}

	// Cell (0,0) turns unseen; cell (1,0) is UNTOUCHED and stays visible —
	// the isolation half of the claim.
	unseenFog := mapFog(map[image.Point]uint8{{X: 0, Y: 0}: FogUnseen}, FogVisible)
	mixed1 := composeMinimap(colours, 2, 1, unseenFog, nil, image.Rectangle{}, box)
	if got, want := mixed1.RGBAAt(0, 0), (color.RGBA{A: 255}); got != want {
		t.Errorf("unseen cell (0,0) = %v, want black %v", got, want)
	}
	if got := mixed1.RGBAAt(1, 0); got != colourB {
		t.Errorf("changing cell (0,0)'s fog moved cell (1,0)'s pixel: got %v, want the untouched %v", got, colourB)
	}

	// Cell (0,0) turns explored instead: exactly half of the SAME cell's
	// visible answer, and (1,0) still untouched.
	exploredFog := mapFog(map[image.Point]uint8{{X: 0, Y: 0}: FogExplored}, FogVisible)
	mixed2 := composeMinimap(colours, 2, 1, exploredFog, nil, image.Rectangle{}, box)
	wantHalf := color.RGBA{R: colourA.R / 2, G: colourA.G / 2, B: colourA.B / 2, A: colourA.A}
	if got := mixed2.RGBAAt(0, 0); got != wantHalf {
		t.Errorf("explored cell (0,0) = %v, want half of visible %v = %v", got, colourA, wantHalf)
	}
	if got := mixed2.RGBAAt(1, 0); got != colourB {
		t.Errorf("changing cell (0,0)'s fog moved cell (1,0)'s pixel: got %v, want the untouched %v", got, colourB)
	}
}

func TestComposeMinimapMarksTwoColours(t *testing.T) {
	colours := []color.RGBA{{R: 1, G: 2, B: 3, A: 255}, {R: 4, G: 5, B: 6, A: 255}}
	marks := []minimapMark{
		{Cell: image.Pt(0, 0), Local: true},
		{Cell: image.Pt(1, 0), Local: false},
	}
	img := composeMinimap(colours, 2, 1, solidFog(FogVisible), marks, image.Rectangle{}, image.Pt(2, 1))

	if got := img.RGBAAt(0, 0); got != minimapLocalColor {
		t.Errorf("local mark pixel = %v, want minimapLocalColor %v", got, minimapLocalColor)
	}
	if got := img.RGBAAt(1, 0); got != minimapOtherColor {
		t.Errorf("non-local mark pixel = %v, want minimapOtherColor %v", got, minimapOtherColor)
	}
}

func TestMinimapMarksDropsWhatTheFogGateWouldDrop(t *testing.T) {
	v, err := NewViewer("mm-marks", grid(3, 3), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.SetLocalOwner(1)

	plane := make([]byte, 3*3) // zero value is FogUnseen throughout
	plane[0] = FogVisible      // cell (0,0)
	v.SetFog(plane, 3, 3)

	v.SetEntities([]MapEntity{
		{ID: 1, Cell: image.Pt(1, 1), Owner: 2}, // non-local, unseen: must be dropped
		{ID: 2, Cell: image.Pt(0, 0), Owner: 2}, // non-local, visible: must be marked, not local
		{ID: 3, Cell: image.Pt(1, 1), Owner: 1}, // local, unseen: must be marked anyway, local
	})

	marks := v.minimapMarks()
	byCell := make(map[image.Point]minimapMark, len(marks))
	for _, m := range marks {
		byCell[m.Cell] = m
	}

	if _, ok := byCell[image.Pt(1, 1)]; !ok {
		t.Fatal("no mark landed on (1,1); the local entity there should have survived")
	} else if !byCell[image.Pt(1, 1)].Local {
		t.Error("the mark on (1,1) is not Local; it should be the LOCAL entity's")
	}
	if m, ok := byCell[image.Pt(0, 0)]; !ok {
		t.Error("the non-local entity on the visible cell (0,0) did not reach minimapMarks")
	} else if m.Local {
		t.Error("the mark on (0,0) is Local; it should be the non-local entity's")
	}
	if len(marks) != 2 {
		t.Fatalf("minimapMarks() = %d marks (%+v), want 2 — the non-local unseen entity must be dropped", len(marks), marks)
	}
}

// TestMinimapMarksRetireAFallenUnitPastHeal pins the fallen-unit rule
// (DIV-1455): a unit keeps its owner's mark while it lives or while Heal can
// still raise it, and has none once it cannot. The fog gate applies unchanged
// to whatever keeps a mark. Row 0 is in sight and row 1 is unseen.
func TestMinimapMarksRetireAFallenUnitPastHeal(t *testing.T) {
	for _, c := range []struct {
		name  string
		e     MapEntity
		local bool
		want  bool
	}{
		{"local living unit under fog", MapEntity{Cell: image.Pt(0, 1), Owner: 1, HP: 20, MaxHP: 20, Restorable: true}, true, true},
		{"local downed unit under fog", MapEntity{Cell: image.Pt(0, 1), Owner: 1, Life: LifeDowned, MaxHP: 20, Restorable: true}, true, true},
		{"local fallen unit at -9 under fog", MapEntity{Cell: image.Pt(0, 1), Owner: 1, Life: LifeDead, HP: -9, MaxHP: 20, Restorable: true}, true, true},
		{"local body at -10 under fog", MapEntity{Cell: image.Pt(0, 1), Owner: 1, Life: LifeDead, HP: -10, MaxHP: 20, Untargetable: true}, true, false},
		{"local body at -35 in sight", MapEntity{Cell: image.Pt(0, 0), Owner: 1, Life: LifeDead, HP: -35, MaxHP: 20, Untargetable: true}, true, false},
		{"local archived dead record under fog", MapEntity{Cell: image.Pt(0, 1), Owner: 1, Life: LifeDead, HP: -45, Untargetable: true}, true, false},
		{"hostile fallen unit at -9 in sight", MapEntity{Cell: image.Pt(0, 0), Owner: 2, Life: LifeDead, HP: -9, MaxHP: 20, Restorable: true}, false, true},
		{"hostile fallen unit at -9 under fog", MapEntity{Cell: image.Pt(0, 1), Owner: 2, Life: LifeDead, HP: -9, MaxHP: 20, Restorable: true}, false, false},
		{"hostile body at -10 in sight", MapEntity{Cell: image.Pt(0, 0), Owner: 2, Life: LifeDead, HP: -10, MaxHP: 20, Untargetable: true}, false, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			v, err := NewViewer("mm-fallen", grid(2, 2), &terrain.Tileset{})
			if err != nil {
				t.Fatalf("NewViewer: %v", err)
			}
			v.SetLocalOwner(1)
			v.SetFog([]byte{FogVisible, FogVisible, FogUnseen, FogUnseen}, 2, 2)
			c.e.ID = 7
			v.SetEntities([]MapEntity{c.e})
			marks := v.minimapMarks()
			if got := len(marks) == 1; got != c.want {
				t.Fatalf("minimapMarks() = %+v, want a mark: %t", marks, c.want)
			}
			if c.want && (marks[0].Cell != c.e.Cell || marks[0].Local != c.local) {
				t.Fatalf("mark = %+v, want cell %v local %t", marks[0], c.e.Cell, c.local)
			}
		})
	}
}

func TestMinimapPresentOnFreshViewer(t *testing.T) {
	v, err := NewViewer("mm-present", grid(4, 4), &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := v.minimapPresent(); !ok {
		t.Fatal("fresh viewer has no minimap")
	}
}

func TestMissionVHasNoBinding(t *testing.T) {
	for name, binding := range bindingSource(t) {
		if strings.Contains(binding, "ebiten.KeyV") {
			t.Fatalf("V still bound to %s", name)
		}
	}
}

// TestMinimapFunctionsTakeNoCursorPosition is spec AC-12's structural half:
// every minimap function that could plausibly be handed a screen position
// is checked, by its exact reflected signature, to take none. A cursor
// position added to any of these later — a hit test grown in place rather
// than added as new code — fails here at compile-adjacent precision rather
// than needing to be noticed by eye.
func TestMinimapFunctionsTakeNoCursorPosition(t *testing.T) {
	check := func(name string, fn any, want []reflect.Type) {
		t.Helper()
		rt := reflect.TypeOf(fn)
		if rt.Kind() != reflect.Func {
			t.Fatalf("%s is not a func value", name)
		}
		if rt.NumIn() != len(want) {
			t.Fatalf("%s takes %d parameters, want %d: %v", name, rt.NumIn(), len(want), want)
		}
		for i, w := range want {
			if got := rt.In(i); got != w {
				t.Fatalf("%s parameter %d = %v, want %v", name, i, got, w)
			}
		}
	}

	viewerPtr := reflect.TypeOf((*Viewer)(nil))
	intT := reflect.TypeOf(int(0))
	colourSliceT := reflect.TypeOf([]color.RGBA(nil))
	fogFuncT := reflect.TypeOf((func(int, int) uint8)(nil))
	marksT := reflect.TypeOf([]minimapMark(nil))
	pointT := reflect.TypeOf(image.Point{})
	// The view rectangle 0140 added is in CELLS, not in window pixels — which
	// is the whole reason this signature test still means what it meant: an
	// image.Rectangle of cells is not a cursor position, and the composer still
	// cannot be handed one.
	rectT := reflect.TypeOf(image.Rectangle{})

	check("composeMinimap", composeMinimap, []reflect.Type{colourSliceT, intT, intT, fogFuncT, marksT, rectT, pointT})
	check("(*Viewer).minimapTerrainColours", (*Viewer).minimapTerrainColours, []reflect.Type{viewerPtr})
	check("(*Viewer).minimapMarks", (*Viewer).minimapMarks, []reflect.Type{viewerPtr})
	check("(*Viewer).minimapPresent", (*Viewer).minimapPresent, []reflect.Type{viewerPtr})
}

// TestCommandFileNamesTheMinimapCapture is the NEGATION of a test this file
// used to carry (0140, owner). 0118 AC-12 required command.go — the one file in
// this package that reads a cursor position to decide a gesture, a selection or
// an order — to name no minimap symbol at all, "not even the word in a comment,
// because a comment today is a call site tomorrow". The owner reversed the
// requirement: the comment became a call site on purpose. So the structural
// assertion is inverted rather than deleted, and it now fails if the capture is
// ever quietly removed — which is the same thing this test was always for, only
// pointing the other way.
//
// THE PATH IS RELATIVE, "command.go", AND NOT DERIVED FROM runtime.Caller:
// `go test` runs a package's test binary with its working directory set to
// the package's own source directory, which is already exactly where this
// file and command.go both live — and unlike a relative path, a source
// path recorded by the runtime is rewritten by -trimpath (this project's
// own test invocation, README's "run go test -trimpath locally") to a
// module-relative form no os.ReadFile call can open.
func TestCommandFileNamesTheMinimapCapture(t *testing.T) {
	const path = "command.go"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	src := string(data)
	for _, sym := range []string{"minimapCaptures", "minimapGrab", "CenterOnMinimapPixel"} {
		if !strings.Contains(src, sym) {
			t.Errorf("%s no longer names %q: the minimap takes the press since 0140, and the "+
				"file that dispatches a press is where that is decided", path, sym)
		}
	}
}

// A minimap press centers the view without issuing an order; a press just
// outside its box remains on the map.
func TestMinimapConsumesTheClickAndCentresTheView(t *testing.T) {
	build := func(t *testing.T) *Viewer {
		t.Helper()
		v, err := NewViewer("mm-click", grid(40, 40), &terrain.Tileset{})
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		layoutViewport(v, DefaultWindowW, DefaultWindowH)
		v.commandMode = true
		return v
	}

	shown := build(t)
	g, ok := shown.minimapGeometry()
	if !ok {
		t.Fatal("setup: the shown viewer has no minimap geometry")
	}

	// A press at the terrain's own top-left corner names cell (0,0); the
	// camera then centres there, which on a map this size means the clamp
	// pulls it to the world's own corner.
	before := image.Pt(int(shown.cam.X), int(shown.cam.Y))
	shown.cam.CenterOn(float64(20*32), float64(20*32)) // start in the middle
	middle := image.Pt(int(shown.cam.X), int(shown.cam.Y))
	if middle == before {
		t.Fatal("setup: centring on the middle moved the camera nowhere; the fixture cannot show a move")
	}

	if _, ok := shown.command(appInput{CursorX: g.Content.Min.X, CursorY: g.Content.Min.Y,
		PrimaryPressed: true}); ok {
		t.Error("a press on the minimap produced an order; it must be swallowed")
	}
	after := image.Pt(int(shown.cam.X), int(shown.cam.Y))
	if after == middle {
		t.Error("a press on the minimap's top-left cell did not move the camera")
	}
	if after.X > middle.X || after.Y > middle.Y {
		t.Errorf("a press on the top-left cell moved the camera to %v from %v, want it nearer the origin",
			after, middle)
	}

	// ONE PIXEL OUTSIDE THE BOX IS THE MAP'S.
	outside := build(t)
	outside.cam.CenterOn(float64(20*32), float64(20*32))
	outsideBefore := image.Pt(int(outside.cam.X), int(outside.cam.Y))
	if outside.minimapCaptures(g.Box.Min.X-1, g.Box.Min.Y-1) {
		t.Error("a pixel outside the box is captured")
	}
	outside.command(appInput{CursorX: g.Box.Min.X - 1, CursorY: g.Box.Min.Y - 1, PrimaryPressed: true})
	if got := (image.Pt(int(outside.cam.X), int(outside.cam.Y))); got != outsideBefore {
		t.Errorf("a press outside the minimap moved the camera to %v from %v", got, outsideBefore)
	}
}

func TestMinimapCapturesTheFullColumnSlotNotJustItsOwnBox(t *testing.T) {
	v, err := NewViewer("mm-slot", grid(60, 60), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	layoutViewport(v, MissionFrameW-MissionPanelW, MissionFrameH)
	v.commandMode = true

	g, ok := v.minimapGeometry()
	if !ok {
		t.Fatal("setup: no minimap geometry")
	}
	gapX, gapY := g.Box.Min.X-1, g.Box.Min.Y+10
	if gapX < v.cam.ViewW {
		t.Fatalf("premise: gap probe x=%d is not past the map viewport at %d", gapX, v.cam.ViewW)
	}
	if image.Pt(gapX, gapY).In(g.Box) {
		t.Fatalf("premise: probe point (%d,%d) is inside the minimap's own drawn box %+v, not the gap beside it", gapX, gapY, g.Box)
	}
	if v.mapSurfaceCaptures(gapX, gapY) {
		t.Fatalf("premise: probe point (%d,%d) is on the map surface, not the right column", gapX, gapY)
	}

	if !v.minimapCaptures(gapX, gapY) {
		t.Errorf("minimapCaptures(%d,%d) = false, want true: the point is inside the column's reserved slot, left of the box's own drawn square", gapX, gapY)
	}

	if _, ok := v.command(appInput{CursorX: gapX, CursorY: gapY, PrimaryPressed: true}); ok {
		t.Error("a press in the minimap's reserved slot but outside its drawn box produced an order; it must be swallowed like a press on the box itself")
	}
}

func TestMinimapBoxIsSquareAndDoesNotMoveWithTheSelection(t *testing.T) {
	v, err := NewViewer("mm-square", grid(80, 80), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	// THE SHIPPED WINDOW, not DefaultWindowW/H. The box is bounded by half the
	// view on either axis, and at 1024x768 that bound bites: half of 768 is 384
	// and the square comes out 16 pixels short of sidebarWidth. The window the
	// game actually opens is MenuWindowW x MenuWindowH (1280x960), where it does
	// not bite — and "the box is exactly the panel's width" is a claim about
	// that window, so it is asserted there.
	layoutViewport(v, MenuWindowW, MenuWindowH)
	v.SetFont(panelFont())
	v.SetEntities([]MapEntity{
		{ID: 5, Cell: image.Pt(3, 3), HP: 10, MaxHP: 10},
		{ID: 7, Cell: image.Pt(9, 9), HP: 63, MaxHP: 100, Name: "Human Swordsman", Speed: 17,
			Combat: UnitCombat{Known: true, DamageBase: 10, DamageSpread: 6, ToHit: 49, Defence: 8},
			Char: UnitCharacter{Known: true, Body: 43, Reaction: 26, Mind: 15, Spirit: 15,
				Weapon: "Iron Short Sword", Experience: 1593, Sight: 6}},
	})

	var first minimapGeom
	for i, sel := range []selection{nil, {5}, {7}} {
		v.sel = sel
		g, ok := v.minimapGeometry()
		if !ok {
			t.Fatalf("selection %v: no geometry", sel)
		}
		if g.Box.Dx() != g.Box.Dy() {
			t.Errorf("selection %v: the box is %dx%d, want a square", sel, g.Box.Dx(), g.Box.Dy())
		}
		if i == 0 {
			first = g
			continue
		}
		if g.Box != first.Box {
			t.Errorf("selection %v moved the box to %v from %v — the box must not be a function "+
				"of what is selected", sel, g.Box, first.Box)
		}
	}

	// AND IT IS hudMinimapReserve, NOT sidebarWidth. Before round 2 of
	// adversarial review the box read sidebarWidth (160), which made it match
	// the unit panel's own width exactly; round 2 decoded id 5's own rect at
	// 160x158 (SHOP-FIGURE-041), and honouring the owner's square directive
	// over an undirected width means the box clamps to the shorter side, 158,
	// not 160. The panel and the minimap are two separately decoded numbers
	// that happen to share a source claim, not one constant read twice —
	// asserted against the constant the minimap itself reads, not against a
	// number copied here.
	if got := first.Box.Dx(); got != hudMinimapReserve {
		t.Errorf("the box is %d wide, want hudMinimapReserve = %d", got, hudMinimapReserve)
	}
	v.sel = selection{7}
	box, ok := characterPanelBoxRect(image.Pt(v.frameW, v.frameH))
	if !ok || box.Dx() != sidebarWidth {
		t.Errorf("the character panel box is %v (ok=%v), want a width of %d — its own decoded"+
			" width, unrelated to the minimap's", box, ok, sidebarWidth)
	}
}

// TestMinimapCellAtIsTheScaleRuleRunBackwards — the hit test and the picture
// must agree cell for cell, which is the one thing that cannot be checked by
// looking at either alone. Both branches of the scale rule are exercised: a
// small map drawn at a whole-pixel scale, and a map large enough to be sampled.
func TestMinimapCellAtIsTheScaleRuleRunBackwards(t *testing.T) {
	for _, tc := range []struct {
		name       string
		cols, rows int
	}{
		{"scaled", 40, 40},
		{"sampled", 600, 600},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, err := NewViewer("mm-hit", grid(tc.cols, tc.rows), &terrain.Tileset{})
			if err != nil {
				t.Fatalf("NewViewer: %v", err)
			}
			layoutViewport(v, DefaultWindowW, DefaultWindowH)
			g, ok := v.minimapGeometry()
			if !ok {
				t.Fatal("no geometry")
			}
			if tc.name == "scaled" && g.Num < g.Den {
				t.Fatalf("the %q case came up reduced (%d/%d); it cannot show what it is for",
					tc.name, g.Num, g.Den)
			}
			if tc.name == "sampled" && g.Num >= g.Den {
				t.Fatalf("the %q case came up expanded (%d/%d); it cannot show what it is for",
					tc.name, g.Num, g.Den)
			}

			// THE FOUR CORNERS OF THE TERRAIN NAME THE FOUR CORNERS OF THE MAP.
			// The far ones are the interesting half: integer division at the
			// last pixel is exactly where an off-by-one puts the camera on a
			// cell the grid does not have.
			for _, c := range []struct {
				name string
				x, y int
				want image.Point
			}{
				{"top-left", g.Content.Min.X, g.Content.Min.Y, image.Pt(0, 0)},
				{"bottom-right", g.Content.Max.X - 1, g.Content.Max.Y - 1, image.Pt(tc.cols-1, tc.rows-1)},
			} {
				got, ok := v.minimapCellAt(c.x, c.y)
				if !ok {
					t.Errorf("%s pixel (%d,%d) names no cell", c.name, c.x, c.y)
					continue
				}
				if c.name == "top-left" && got != c.want {
					t.Errorf("%s pixel names cell %v, want %v", c.name, got, c.want)
				}
				// The far corner is asserted as a BOUND rather than an
				// identity: a reduced pixel can represent several cells, so
				// what must hold is that no pixel names a cell
				// outside the grid and that the far pixel is near the far edge.
				if got.X < 0 || got.Y < 0 || got.X >= tc.cols || got.Y >= tc.rows {
					t.Errorf("%s pixel names cell %v, which is outside a %dx%d grid",
						c.name, got, tc.cols, tc.rows)
				}
			}

			// EVERY PIXEL OF THE TERRAIN NAMES A CELL INSIDE THE GRID. The
			// sweep is what a corner check cannot be: an off-by-one in the
			// middle of the mapping shows here and nowhere else.
			for y := g.Content.Min.Y; y < g.Content.Max.Y; y++ {
				for x := g.Content.Min.X; x < g.Content.Max.X; x++ {
					cell, ok := v.minimapCellAt(x, y)
					if !ok {
						t.Fatalf("pixel (%d,%d) inside the terrain names no cell", x, y)
					}
					if cell.X < 0 || cell.Y < 0 || cell.X >= tc.cols || cell.Y >= tc.rows {
						t.Fatalf("pixel (%d,%d) names cell %v, outside a %dx%d grid",
							x, y, cell, tc.cols, tc.rows)
					}
				}
			}

			// A PIXEL INSIDE THE BOX BUT OUTSIDE THE TERRAIN NAMES NONE, and is
			// still captured. The border is the always-present case of that.
			if _, ok := v.minimapCellAt(g.Box.Min.X, g.Box.Min.Y); ok {
				t.Error("the box's own border pixel names a cell")
			}
			if !v.minimapCaptures(g.Box.Min.X, g.Box.Min.Y) {
				t.Error("the box's own border pixel is not captured")
			}
		})
	}
}

// TestMinimapGeometryFillsWhicheverAxisIsTighter is the owner's own report:
// "the map itself for some reason doesn't occupy at least the full height or
// the full width, whichever would fill first" — asserted at the geometry
// level a player actually sees (Content, inside the authored square box),
// not only at composeMinimap's own returned image, since it is
// minimapGeometry's centring arithmetic, not the composer, that owner report
// R3 traced the dead margin to.
//
// 80x50 IS CHOSEN so neither axis divides the 156-pixel content square
// (hudMinimapReserve - 2) evenly, which is exactly the case the old
// whole-pixel-block scale left short on its own tighter axis.
//
// WITNESSED BY REVERTING: run against the pre-fix minimapGeometry (floor
// block scale, centred), this went red on the width assertion (154, two
// pixels short of the box's own 156) before the fix, and was restored.
func TestMinimapGeometryFillsWhicheverAxisIsTighter(t *testing.T) {
	v, err := NewViewer("mm-fill", grid(80, 50), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	layoutViewport(v, DefaultWindowW, DefaultWindowH)
	g, ok := v.minimapGeometry()
	if !ok {
		t.Fatal("no geometry")
	}

	square := hudMinimapReserve - 2 // the authored content square, border excluded
	fillsWidth := g.Content.Dx() == square
	fillsHeight := g.Content.Dy() == square
	if !fillsWidth && !fillsHeight {
		t.Fatalf("content %v fills neither axis of the %dx%d square — the owner's own report", g.Content, square, square)
	}
	if g.Content.Dx() > square || g.Content.Dy() > square {
		t.Fatalf("content %v exceeds the %dx%d square", g.Content, square, square)
	}
	// Aspect is kept, not stretched: width/height tracks cols/rows within one
	// pixel of rounding either way.
	wantRatio := float64(80) / float64(50)
	gotRatio := float64(g.Content.Dx()) / float64(g.Content.Dy())
	if d := wantRatio - gotRatio; d > 0.05 || d < -0.05 {
		t.Fatalf("content %v ratio %.3f, want close to cols/rows = %.3f (aspect must not stretch)", g.Content, gotRatio, wantRatio)
	}
}

func TestComposeMinimapOutlinesTheCameraView(t *testing.T) {
	const cols, rows, scale = 8, 8, 4
	colours := make([]color.RGBA, cols*rows)
	for i := range colours {
		colours[i] = color.RGBA{R: 9, G: 9, B: 9, A: 255}
	}
	box := image.Pt(cols*scale, rows*scale)

	view := image.Rect(2, 2, 6, 6) // cells; 8x8 output pixels at scale 4
	img := composeMinimap(colours, cols, rows, solidFog(FogVisible), nil, view, box)
	bare := composeMinimap(colours, cols, rows, solidFog(FogVisible), nil, image.Rectangle{}, box)

	at := func(im *image.RGBA, x, y int) color.RGBA {
		o := im.PixOffset(x, y)
		return color.RGBA{R: im.Pix[o], G: im.Pix[o+1], B: im.Pix[o+2], A: im.Pix[o+3]}
	}
	x0, y0, x1, y1 := 2*scale, 2*scale, 6*scale-1, 6*scale-1

	for _, p := range []image.Point{{X: x0, Y: y0}, {X: x1, Y: y0}, {X: x0, Y: y1}, {X: x1, Y: y1},
		{X: (x0 + x1) / 2, Y: y0}, {X: x0, Y: (y0 + y1) / 2}} {
		if got := at(img, p.X, p.Y); got != minimapViewColor {
			t.Errorf("edge pixel (%d,%d) is %v, want the view colour %v", p.X, p.Y, got, minimapViewColor)
		}
	}
	if got := at(img, (x0+x1)/2, (y0+y1)/2); got != at(bare, (x0+x1)/2, (y0+y1)/2) {
		t.Error("the middle of the view rectangle was painted: this is a fill, not an outline")
	}
	if got := at(img, x0-1, y0-1); got != at(bare, x0-1, y0-1) {
		t.Error("a pixel outside the view rectangle was painted")
	}

	// AN EMPTY RANGE DRAWS NOTHING — camera.VisibleTiles clips to the world and
	// can return one, and no cell is what gets outlined.
	if !reflect.DeepEqual(bare.Pix,
		composeMinimap(colours, cols, rows, solidFog(FogVisible), nil, image.Rect(3, 3, 3, 3), box).Pix) {
		t.Error("an empty view range painted something")
	}

	// AND IT NEVER VANISHES — ONE CELL IS ONE MARK, in the sampled branch as
	// well as the scaled one. Asserted at the smallest range there is, in the
	// branch where a collapse would be conceivable.
	sampled := composeMinimap(make([]color.RGBA, 40*40), 40, 40, solidFog(FogVisible), nil,
		image.Rect(4, 4, 5, 5), image.Pt(10, 10))
	bareSampled := composeMinimap(make([]color.RGBA, 40*40), 40, 40, solidFog(FogVisible), nil,
		image.Rectangle{}, image.Pt(10, 10))
	if reflect.DeepEqual(bareSampled.Pix, sampled.Pix) {
		t.Error("a one-cell view over a sampled map drew nothing at all")
	}
}
