package terrain_test

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/render/terrain"
)

// The AC-3 fixture's geometry. originY is negative and lift varies per cell,
// which is the ordinary shape of a projected render rather than a contrived one:
// minV is the least row*32 - h(c,row) over the mesh, so any map whose top row
// carries a positive altitude reports a negative origin.
const (
	staticPlaceOriginY = -19
	staticPlaceCols    = 3
	staticPlaceRows    = 3
)

// staticPlaceLift is the fixture's height source: distinct at every cell and
// asymmetric in col and row, so a builder that read one for the other, or that
// applied one cell's lift to another, lands on a different number.
func staticPlaceLift(col, row int) int { return 7*col - 3*row + 1 }

// staticPlaceSet builds AC-3's classes fresh on every call — byte 1 and byte 2
// drawable, byte 3 resolved but artless, every other byte naming no class.
//
// Fresh rather than shared: TestStaticPlacementsReadAndNeverWrite compares a set
// the builder has walked against one it has not, which only says anything if the
// two started identical and are not the same memory.
func staticPlaceSet() *terrain.StaticSet {
	var set terrain.StaticSet

	// Byte 1: a frame SMALLER than its canvas, the ordinary case.
	//   anchorX = 30 - 64/2 + 20/2 =  8
	//   anchorY = 80 - 96/2 + 24/2 = 44
	set.Classes[1] = &terrain.StaticClass{
		Width: 64, Height: 96, CenterX: 30, CenterY: 80,
		Frame: &terrain.StaticFrame{Width: 20, Height: 24, Pixels: make([]terrain.StaticPixel, 20*24)},
	}

	// Byte 2: a frame LARGER than its canvas, which an anchor clamped to the
	// canvas could not place.
	//   anchorX = 16 - 32/2 +  96/2 = 48
	//   anchorY = 30 - 32/2 + 128/2 = 78
	set.Classes[2] = &terrain.StaticClass{
		Width: 32, Height: 32, CenterX: 16, CenterY: 30,
		Frame: &terrain.StaticFrame{Width: 96, Height: 128, Pixels: make([]terrain.StaticPixel, 96*128)},
	}

	// Byte 3: resolved, artless. Its canvas and centre are deliberately set to
	// values that would place a sprite somewhere, so the skip is the nil Frame
	// and nothing else.
	set.Classes[3] = &terrain.StaticClass{Width: 40, Height: 40, CenterX: 20, CenterY: 38}

	return &set
}

// staticPlaceGrid is AC-3's map: two resolving cells on ADJACENT rows, a zero
// cell, a byte naming no loaded class, and a class with no drawable frame.
//
//	row 0:  .  1  .
//	row 1:  7  2  .
//	row 2:  3  .  .
func staticPlaceGrid() terrain.Grid {
	return terrain.Grid{
		Width:  staticPlaceCols,
		Height: staticPlaceRows,
		Tiles:  make([]uint16, staticPlaceCols*staticPlaceRows),
		Overlay: []uint8{
			0, 1, 0,
			7, 2, 0,
			3, 0, 0,
		},
	}
}

// staticPlaceWant is one hand-computed expected placement.
type staticPlaceWant struct {
	cell           image.Point
	topLeft        image.Point
	anchor         image.Point
	frameW, frameH int
	ground         image.Point
	byteOfTheClass uint8
	why            string
}

// The flat list, row-major. Cell centres are (col*32+16, row*32+16), so cell
// (1,0) is (48,16) and cell (1,1) is (48,48).
//
//	(1,0): destX = 48 -  8 =  40   destY = 16 - 44 = -28
//	(1,1): destX = 48 - 48 =   0   destY = 48 - 78 = -30
var staticPlaceFlat = []staticPlaceWant{
	{
		cell: image.Pt(1, 0), topLeft: image.Pt(40, -28), anchor: image.Pt(8, 44),
		frameW: 20, frameH: 24, ground: image.Pt(48, 16), byteOfTheClass: 1,
		why: "the frame's own half is added back after the canvas's is taken off, and destY is negative because the sprite reaches above the map's first row",
	},
	{
		cell: image.Pt(1, 1), topLeft: image.Pt(0, -30), anchor: image.Pt(48, 78),
		frameW: 96, frameH: 128, ground: image.Pt(48, 48), byteOfTheClass: 2,
		why: "the row below, and the second entry: row-major means this one comes after (1,0) whatever the two contain",
	},
}

// The displaced list at staticPlaceOriginY, each entry the flat one shifted by
// exactly -(lift + originY) in Y and by zero in X.
//
//	(1,0): lift  7*1 - 3*0 + 1 =  8   shift = -(8  - 19) = +11   destY = -17
//	(1,1): lift  7*1 - 3*1 + 1 =  5   shift = -(5  - 19) = +14   destY = -16
var staticPlaceDisplaced = []staticPlaceWant{
	{
		cell: image.Pt(1, 0), topLeft: image.Pt(40, -17), anchor: image.Pt(8, 44),
		frameW: 20, frameH: 24, ground: image.Pt(48, 27), byteOfTheClass: 1,
		why: "the anchor is geometry-independent: only the two vertical terms moved",
	},
	{
		cell: image.Pt(1, 1), topLeft: image.Pt(0, -16), anchor: image.Pt(48, 78),
		frameW: 96, frameH: 128, ground: image.Pt(48, 62), byteOfTheClass: 2,
		why: "a DIFFERENT shift from the row above, so a builder reading one cell's lift for another's fails here",
	},
}

func checkStaticPlacements(t *testing.T, geometry string, got []terrain.StaticPlacement, want []staticPlaceWant, set *terrain.StaticSet) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s: %d placements, want exactly %d — one per cell resolving to a drawable frame, and nothing for the zero cell, the unknown byte or the artless class (P-6)",
			geometry, len(got), len(want))
	}
	for i, w := range want {
		p := got[i]
		if p.Cell != w.cell {
			t.Errorf("%s: placement %d is for cell %v, want %v — the list is row-major, row ascending then col", geometry, i, p.Cell, w.cell)
			continue
		}
		if p.TopLeft != w.topLeft {
			t.Errorf("%s: cell %v top-left = %v, want %v — %s", geometry, w.cell, p.TopLeft, w.topLeft, w.why)
		}
		if p.Anchor != w.anchor {
			t.Errorf("%s: cell %v anchor = %v, want %v — %s", geometry, w.cell, p.Anchor, w.anchor, w.why)
		}
		if p.Frame == nil {
			t.Errorf("%s: cell %v carries no frame; only a drawable class places at all", geometry, w.cell)
			continue
		}
		if p.Frame.Width != w.frameW || p.Frame.Height != w.frameH {
			t.Errorf("%s: cell %v frame = %dx%d, want %dx%d — the placement carries the DRAWN frame's size, not the class canvas",
				geometry, w.cell, p.Frame.Width, p.Frame.Height, w.frameW, w.frameH)
		}
		if p.Frame != set.Classes[w.byteOfTheClass].Frame {
			t.Errorf("%s: cell %v carries a frame that is not the set's own; the pointer IS the texture cache key (DD-8)", geometry, w.cell)
		}
		if p.Class != set.Classes[w.byteOfTheClass] {
			t.Errorf("%s: cell %v carries class %p, want the set's own %p — the per-counter pass re-selects through this pointer (0031 DD-5)",
				geometry, w.cell, p.Class, set.Classes[w.byteOfTheClass])
		}
		if g := p.Ground(); g != w.ground {
			t.Errorf("%s: cell %v Ground() = %v, want %v — the top-left plus the frame's own anchor pixel", geometry, w.cell, g, w.ground)
		}
		if r := p.Rect(); r != (image.Rectangle{Min: w.topLeft, Max: w.topLeft.Add(image.Pt(w.frameW, w.frameH))}) {
			t.Errorf("%s: cell %v Rect() = %v, want the frame's exact world rectangle at %v", geometry, w.cell, r, w.topLeft)
		}
	}
}

// TestStaticPlacementsSyntheticMap covers SC-3 (AC-3): AC-3's map built flat
// and then displaced, against the two hand-computed tables, with the two
// skip kinds counted apart and one census serving both geometries.
func TestStaticPlacementsSyntheticMap(t *testing.T) {
	if terrain.CellSize != 32 {
		t.Fatalf("CellSize = %d, want 32; every top-left literal above is stated over 32-pixel cells", terrain.CellSize)
	}

	set := staticPlaceSet()
	g := staticPlaceGrid()

	flat, flatCounts, _ := terrain.StaticPlacements(g, set, nil, 0, terrain.AnimGateTiles)
	checkStaticPlacements(t, "flat", flat, staticPlaceFlat, set)

	displaced, displacedCounts, _ := terrain.StaticPlacements(g, set, staticPlaceLift, staticPlaceOriginY, terrain.AnimGateTiles)
	checkStaticPlacements(t, "displaced", displaced, staticPlaceDisplaced, set)

	want := terrain.StaticCounts{Placed: 2, NoClass: 1, NoFrame: 1}
	if flatCounts != want {
		t.Errorf("flat census = %+v, want %+v — byte 7 names no class and byte 3 names an artless one, and the two are DIFFERENT skips; the zero cells are counted nowhere",
			flatCounts, want)
	}
	if displacedCounts != flatCounts {
		t.Errorf("displaced census = %+v, want the flat %+v — which cells resolve is a question about bytes and classes, and altitudes reach only the anchors",
			displacedCounts, flatCounts)
	}
	if flatCounts.Placed != len(flat) {
		t.Errorf("census Placed = %d but the list holds %d", flatCounts.Placed, len(flat))
	}

	for i := range flat {
		if displaced[i].TopLeft.X != flat[i].TopLeft.X {
			t.Errorf("cell %v: displaced destX = %d, flat %d — the displacement is vertical only and destX takes no origin term (P-2)",
				flat[i].Cell, displaced[i].TopLeft.X, flat[i].TopLeft.X)
		}
		shift := staticPlaceLift(flat[i].Cell.X, flat[i].Cell.Y) + staticPlaceOriginY
		if want := flat[i].TopLeft.Y - shift; displaced[i].TopLeft.Y != want {
			t.Errorf("cell %v: displaced destY = %d, want %d (the flat %d shifted by exactly -(lift+originY))",
				flat[i].Cell, displaced[i].TopLeft.Y, want, flat[i].TopLeft.Y)
		}
		if displaced[i].Anchor != flat[i].Anchor {
			t.Errorf("cell %v: displaced anchor %v, want the flat %v — the frame's own anchor pixel is geometry-independent",
				flat[i].Cell, displaced[i].Anchor, flat[i].Anchor)
		}
	}
}

// TestStaticPlacementsRejectAMismatchedGrid covers SC-3's short-Overlay case and
// the rest of the refusal set: each of these yields NO placements and an empty
// census rather than a partial list a caller could mistake for a complete one.
//
// Each case's overlay would place two sprites if it were accepted, so a builder
// that walked what it was given regardless of the length would show up as
// placements here rather than as a silent index panic on some other map.
func TestStaticPlacementsRejectAMismatchedGrid(t *testing.T) {
	set := staticPlaceSet()
	full := staticPlaceGrid()

	cases := []struct {
		name string
		grid terrain.Grid
		set  *terrain.StaticSet
		why  string
	}{
		{
			name: "no overlay at all",
			grid: terrain.Grid{Width: 3, Height: 3, Tiles: full.Tiles},
			set:  set,
			why:  "a caller with no object layer — the pre-story grid literal, unaffected by the new field",
		},
		{
			name: "overlay one byte short",
			grid: terrain.Grid{Width: 3, Height: 3, Tiles: full.Tiles, Overlay: full.Overlay[:len(full.Overlay)-1]},
			set:  set,
			why:  "validAltitudes' rule applied to the other layer: exactly Width*Height or nothing",
		},
		{
			name: "overlay one byte long",
			grid: terrain.Grid{Width: 3, Height: 3, Tiles: full.Tiles, Overlay: append(append([]uint8(nil), full.Overlay...), 1)},
			set:  set,
			why:  "too many is as wrong as too few — a length mismatch means the caller's layers do not describe one map",
		},
		{
			name: "zero width",
			grid: terrain.Grid{Width: 0, Height: 3, Overlay: nil},
			set:  set,
			why:  "no cells, and row*Width indexing that means nothing",
		},
		{
			name: "negative height",
			grid: terrain.Grid{Width: 3, Height: -3, Overlay: full.Overlay},
			set:  set,
			why:  "a negative dimension multiplies to a negative cell count, which no length equals",
		},
		{
			name: "nil set",
			grid: full,
			set:  nil,
			why:  "the bundle types are field-only, so every reader owns its own nil guard; a nil set is an empty bundle, not an error",
		},
	}

	for _, c := range cases {
		got, counts, _ := terrain.StaticPlacements(c.grid, c.set, staticPlaceLift, staticPlaceOriginY, terrain.AnimGateTiles)
		if len(got) != 0 {
			t.Errorf("%s: %d placements, want none — %s", c.name, len(got), c.why)
		}
		if (counts != terrain.StaticCounts{}) {
			t.Errorf("%s: census = %+v, want the zero census — nothing was walked, so nothing was skipped either", c.name, counts)
		}
	}

	// An empty bundle over a valid grid is a different answer from a rejected
	// grid: every non-zero byte IS walked, and every one of them is a skip.
	var empty terrain.StaticSet
	got, counts, _ := terrain.StaticPlacements(full, &empty, nil, 0, terrain.AnimGateTiles)
	if want := (terrain.StaticCounts{NoClass: 4}); len(got) != 0 || counts != want {
		t.Errorf("empty bundle: %d placements and census %+v, want none and %+v — the grid's four non-zero bytes each name no loaded class",
			len(got), counts, want)
	}
}

func TestStaticPlacementsInventNoExclusion(t *testing.T) {
	var set terrain.StaticSet
	// Canvas 8x8, centre (4,6) for both, so only the frame differs.
	//   byte 1, frame  0x0:  anchor (4-4+0, 6-4+0) = (0,2)  dest (16-0, 16-2) = (16,14)
	//   byte 2, frame  3x2:  anchor (4-4+1, 6-4+1) = (1,3)  dest (48-1, 16-3) = (47,13)
	set.Classes[1] = &terrain.StaticClass{Width: 8, Height: 8, CenterX: 4, CenterY: 6,
		Frame: &terrain.StaticFrame{}}
	set.Classes[2] = &terrain.StaticClass{Width: 8, Height: 8, CenterX: 4, CenterY: 6,
		Frame: &terrain.StaticFrame{Width: 3, Height: 2}} // a size stated, no pixels behind it

	// No Tiles: the object layer is the only one this builder reads, and a
	// caller carrying no terrain is not a caller it may refuse.
	g := terrain.Grid{Width: 2, Height: 1, Overlay: []uint8{1, 2}}

	got, counts, _ := terrain.StaticPlacements(g, &set, nil, 0, terrain.AnimGateTiles)
	if want := (terrain.StaticCounts{Placed: 2}); counts != want || len(got) != 2 {
		t.Fatalf("%d placements and census %+v, want 2 and %+v — a non-nil Frame is the whole rule", len(got), counts, want)
	}
	if got[0].TopLeft != image.Pt(16, 14) || got[0].Anchor != image.Pt(0, 2) {
		t.Errorf("the 0x0 frame placed at %v anchor %v, want (16,14) anchor (0,2) — a frame of no area still has a top-left", got[0].TopLeft, got[0].Anchor)
	}
	if got[1].TopLeft != image.Pt(47, 13) || got[1].Anchor != image.Pt(1, 3) {
		t.Errorf("the pixel-less frame placed at %v anchor %v, want (47,13) anchor (1,3) — the size is read, the pixels are not", got[1].TopLeft, got[1].Anchor)
	}
	if r := got[0].Rect(); !r.Empty() || r.Min != image.Pt(16, 14) {
		t.Errorf("the 0x0 frame's Rect() = %v, want an empty rectangle at (16,14) — nothing for a cull to keep, and nothing for a blit to write", r)
	}
}

func TestStaticPlacementsNilLiftIsAZeroLift(t *testing.T) {
	set, g := staticPlaceSet(), staticPlaceGrid()

	absent, absentCounts, _ := terrain.StaticPlacements(g, set, nil, staticPlaceOriginY, terrain.AnimGateTiles)
	explicit, explicitCounts, _ := terrain.StaticPlacements(g, set, func(col, row int) int { return 0 }, staticPlaceOriginY, terrain.AnimGateTiles)

	if !reflect.DeepEqual(absent, explicit) || absentCounts != explicitCounts {
		t.Errorf("a nil lift built %v, a lift of 0 everywhere built %v; the two must be the same list at the same origin", absent, explicit)
	}
	if len(absent) == 0 {
		t.Fatal("no placements to compare")
	}
	if flat, _, _ := terrain.StaticPlacements(g, set, nil, 0, terrain.AnimGateTiles); reflect.DeepEqual(absent, flat) {
		t.Error("a nil lift at a non-zero originY built the SAME list as one at originY 0; the origin term applies whether or not a lift was supplied")
	}
}

func TestStaticPlacementsAreRowMajorAndComplete(t *testing.T) {
	const cols, rows = 4, 3

	set := staticPlaceSet()
	overlay := make([]uint8, cols*rows)
	for i := range overlay {
		overlay[i] = 1
	}
	g := terrain.Grid{Width: cols, Height: rows, Tiles: make([]uint16, cols*rows), Overlay: overlay}

	got, counts, _ := terrain.StaticPlacements(g, set, staticPlaceLift, staticPlaceOriginY, terrain.AnimGateTiles)
	if len(got) != cols*rows || counts.Placed != cols*rows {
		t.Fatalf("%d placements and census Placed %d, want %d — every cell here resolves to a drawable frame",
			len(got), counts.Placed, cols*rows)
	}

	i := 0
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			if want := image.Pt(col, row); got[i].Cell != want {
				t.Fatalf("placement %d is for cell %v, want %v — row-major is row ascending, THEN col (spec, \"Draw order\": the later cell owns an overlap)",
					i, got[i].Cell, want)
			}
			i++
		}
	}

	for _, p := range got {
		want := image.Pt(
			p.Cell.X*terrain.CellSize+terrain.CellSize/2,
			p.Cell.Y*terrain.CellSize+terrain.CellSize/2-staticPlaceLift(p.Cell.X, p.Cell.Y)-staticPlaceOriginY,
		)
		if g := p.Ground(); g != want {
			t.Errorf("cell %v: Ground() = %v, want the lifted cell centre %v", p.Cell, g, want)
		}
	}
}

// TestStaticPlacementsReadTheLiftOncePerPlacingCell pins what the builder asks
// its height source. A projection lookup is a four-corner mean over a borrowed
// altitude slice and a map carries 65536 cells, so reading it twice per cell —
// or reading it for cells that place nothing — is a cost with nothing to show
// for it.
func TestStaticPlacementsReadTheLiftOncePerPlacingCell(t *testing.T) {
	set := staticPlaceSet()
	g := staticPlaceGrid()

	seen := map[image.Point]int{}
	lift := func(col, row int) int {
		if col < 0 || row < 0 || col >= g.Width || row >= g.Height {
			t.Errorf("the lift was asked for cell (%d,%d), which is off a %dx%d map", col, row, g.Width, g.Height)
		}
		seen[image.Pt(col, row)]++
		return staticPlaceLift(col, row)
	}

	got, _, _ := terrain.StaticPlacements(g, set, lift, staticPlaceOriginY, terrain.AnimGateTiles)

	for _, p := range got {
		if n := seen[p.Cell]; n != 1 {
			t.Errorf("cell %v placed but its lift was read %d times, want exactly once", p.Cell, n)
		}
	}
	if len(seen) != len(got) {
		t.Errorf("the lift was read for %d cells but only %d placed; a skipped cell must ask the projection nothing", len(seen), len(got))
	}
}

func TestStaticPlacementsReadAndNeverWrite(t *testing.T) {
	set, g := staticPlaceSet(), staticPlaceGrid()

	first, firstCounts, _ := terrain.StaticPlacements(g, set, staticPlaceLift, staticPlaceOriginY, terrain.AnimGateTiles)

	if pristine := staticPlaceSet(); !reflect.DeepEqual(set, pristine) {
		t.Error("the class bundle changed while the list was built; the builder reads it and nothing else (P-3)")
	}
	if pristine := staticPlaceGrid(); !reflect.DeepEqual(g, pristine) {
		t.Error("the map's own layers changed while the list was built")
	}

	second, secondCounts, _ := terrain.StaticPlacements(g, set, staticPlaceLift, staticPlaceOriginY, terrain.AnimGateTiles)
	if !reflect.DeepEqual(first, second) || firstCounts != secondCounts {
		t.Error("two builds of one input differ; the list is deterministic, which is what lets both renderers place from it unchanged (P-3)")
	}
}

// TestStaticPlacementAccessorsAreTotal covers the two accessors on values the
// builder never produces — a zero placement, and a frame of no area. Both are
// reachable from a hand-built list, and neither may panic: the window's cull
// calls Rect() on whatever it is handed.
func TestStaticPlacementAccessorsAreTotal(t *testing.T) {
	var zero terrain.StaticPlacement
	if r := zero.Rect(); !r.Empty() {
		t.Errorf("the zero placement's Rect() = %v, want an empty rectangle — no frame, no area, and nothing for a cull to intersect", r)
	}
	if p := zero.Ground(); p != (image.Point{}) {
		t.Errorf("the zero placement's Ground() = %v, want the origin — the sum of two zero points", p)
	}

	// Ground() reads neither the frame nor the cell: it is the sum of the two
	// points the placement already carries, which is what leaves T5's comparison
	// two derivations instead of one.
	p := terrain.StaticPlacement{
		Cell:    image.Pt(9, 9),
		TopLeft: image.Pt(-7, 12),
		Anchor:  image.Pt(5, -3),
		Frame:   &terrain.StaticFrame{Width: 0, Height: 6},
	}
	if got := p.Ground(); got != image.Pt(-2, 9) {
		t.Errorf("Ground() = %v, want (-2,9) — TopLeft + Anchor, at either sign", got)
	}
	if r := p.Rect(); r.Min != image.Pt(-7, 12) || r.Max != image.Pt(-7, 18) {
		t.Errorf("Rect() = %v, want (-7,12)-(-7,18) — the half-open [TopLeft, TopLeft+size), zero-width and all", r)
	}
}
