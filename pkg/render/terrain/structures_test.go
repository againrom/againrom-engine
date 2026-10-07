package terrain_test

import (
	"fmt"
	"image"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/render/terrain"
)

// The structure layer's geometry, over hand-computed strips
// (docs/0054-structure-art: AC-3).
//
// EVERY LITERAL BELOW IS STATED OVER 32-PIXEL CELLS, and TestStructureCellSize
// pins that. Nothing here opens an archive or reads an install: a class is a
// struct literal, a frame is a struct literal, and the whole of what is under
// test is integer arithmetic over them.

// structCellSize is the cell size every destX/destY literal in this file is
// stated over. It is a local constant so a change to terrain.CellSize fails the
// pin below rather than silently re-scaling every expectation.
const structCellSize = 32

func TestStructureCellSize(t *testing.T) {
	if terrain.CellSize != structCellSize {
		t.Fatalf("CellSize = %d, want %d; every destX/destY literal in this file is stated over %d-pixel cells",
			terrain.CellSize, structCellSize, structCellSize)
	}
}

// structFrames builds n distinct frames, each CellSize square, so a test can say
// WHICH frame an entry selected by pointer identity rather than by comparing
// pixels. The frames differ only in their identity, which is the whole of what a
// grid index decides.
func structFrames(n int) []*terrain.StaticFrame {
	out := make([]*terrain.StaticFrame, n)
	for i := range out {
		out[i] = &terrain.StaticFrame{
			Width:  terrain.CellSize,
			Height: terrain.CellSize,
			Pixels: make([]terrain.StaticPixel, terrain.CellSize*terrain.CellSize),
		}
	}
	return out
}

// structClass is a drawable class of the stated extents whose sheet holds exactly
// its base block — TileWidth*FullHeight frames, one per grid index.
func structClass(tw, th, fh int) *terrain.StructureClass {
	return &terrain.StructureClass{
		TileWidth:  tw,
		TileHeight: th,
		FullHeight: fh,
		Frames:     structFrames(tw * fh),
	}
}

// structGrid is a w x h map carrying the given placement records and nothing
// else: no tiles, no altitudes, no object grid. The builder reads the extent and
// the records, so a grid with anything more in it would be asserting that it does
// not.
func structGrid(w, h int, recs ...terrain.StructureRecord) terrain.Grid {
	return terrain.Grid{Width: w, Height: h, Structures: recs}
}

// structAt is a record anchored at whole cell (col, row) carrying key k. The
// stored coordinates are 1/256 of a cell, so a whole cell is <<8 — the shift
// AnchorCell undoes.
func structAt(col, row int, k uint32) terrain.StructureRecord {
	return terrain.StructureRecord{X: uint32(col) << 8, Y: uint32(row) << 8, Key: k}
}

func structSet(byKey map[byte]*terrain.StructureClass) *terrain.StructureSet {
	s := new(terrain.StructureSet)
	for b, c := range byKey {
		s.Classes[b] = c
	}
	return s
}

// wantEntry is one expected entry, spelled as the four things the contract fixes:
// the rectangle cell, the frame's top-left in world pixels, and the grid index.
type wantEntry struct {
	col, row     int
	destX, destY int
	gridIndex    int
}

func checkEntries(t *testing.T, got []terrain.StructurePlacement, want []wantEntry) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%d entries, want %d", len(got), len(want))
	}
	for i, w := range want {
		g := got[i]
		if g.Cell != (image.Point{X: w.col, Y: w.row}) ||
			g.TopLeft != (image.Point{X: w.destX, Y: w.destY}) ||
			g.GridIndex != w.gridIndex {
			t.Errorf("entry %d = cell %v top-left %v grid %d, want cell (%d,%d) top-left (%d,%d) grid %d",
				i, g.Cell, g.TopLeft, g.GridIndex, w.col, w.row, w.destX, w.destY, w.gridIndex)
		}
	}
}

// AC-3, first arm. A 3 x 2 class of FullHeight 5, hand-computed cell for cell and
// frame for frame against the contract's own strip.
//
// The rectangle is two rows of three cells at anchor (1,1). The list comes out in
// the builder's order — rows ascending, columns DESCENDING — so the back row's
// three cells lead, right to left, each carrying its whole strip; the overhang
// (grid rows 0..2) rides on the back row alone, and the front row is one frame per
// cell.
//
// rowTop = ROW0 - TileHeight + FullHeight = ROW0 + 3. For the back row that is 3
// with limit 0, so k walks 3,2,1,0 — four frames, three of them overhang. For the
// front row rowTop = limit = 4, one frame. Grid index is k*3 + COL0, and
// destY steps UP by one cell per step of k below rowTop.
func TestStructureStripIsCellForCellAndFrameForFrame(t *testing.T) {
	set := structSet(map[byte]*terrain.StructureClass{7: structClass(3, 2, 5)})
	places, counts, _ := terrain.StructurePlacements(structGrid(8, 8, structAt(1, 1, 7)), set, nil, 0)

	const c = structCellSize
	checkEntries(t, places, []wantEntry{
		// back row (map row 1), columns descending: 3, 2, 1
		{3, 1, 3 * c, 1*c - 0*c, 3*3 + 2}, {3, 1, 3 * c, 1*c - 1*c, 2*3 + 2},
		{3, 1, 3 * c, 1*c - 2*c, 1*3 + 2}, {3, 1, 3 * c, 1*c - 3*c, 0*3 + 2},
		{2, 1, 2 * c, 1*c - 0*c, 3*3 + 1}, {2, 1, 2 * c, 1*c - 1*c, 2*3 + 1},
		{2, 1, 2 * c, 1*c - 2*c, 1*3 + 1}, {2, 1, 2 * c, 1*c - 3*c, 0*3 + 1},
		{1, 1, 1 * c, 1*c - 0*c, 3*3 + 0}, {1, 1, 1 * c, 1*c - 1*c, 2*3 + 0},
		{1, 1, 1 * c, 1*c - 2*c, 1*3 + 0}, {1, 1, 1 * c, 1*c - 3*c, 0*3 + 0},
		// front row (map row 2), columns descending: one frame each
		{3, 2, 3 * c, 2 * c, 4*3 + 2},
		{2, 2, 2 * c, 2 * c, 4*3 + 1},
		{1, 2, 1 * c, 2 * c, 4*3 + 0},
	})

	// Every entry addresses a distinct grid index, and together they are the
	// whole grid: TileWidth*FullHeight = 15 frames for a 3 x 2 rectangle.
	seen := map[int]bool{}
	for _, p := range places {
		if seen[p.GridIndex] {
			t.Errorf("grid index %d appears twice", p.GridIndex)
		}
		seen[p.GridIndex] = true
		if p.Frame == nil {
			t.Errorf("grid index %d selected no frame; the sheet holds the whole base block", p.GridIndex)
		}
	}
	if len(seen) != 3*5 {
		t.Errorf("%d distinct grid indices, want %d — the strip must cover the grid exactly once", len(seen), 3*5)
	}

	want := terrain.StructureCounts{
		Placements: terrain.StructurePlacementCounts{Drawn: 1},
		Cells:      terrain.StructureCellCounts{Strips: 6, Frames: 15, Overhang: 9},
	}
	if counts != want {
		t.Errorf("counts = %+v, want %+v", counts, want)
	}
}

// AC-3, second arm. A class whose FullHeight equals its TileHeight yields exactly
// one frame per cell and no overhang at all — the degenerate strip, and the shape
// the ROW0 == 0 arm must not invent an extra frame for.
func TestStructureFullHeightEqualToTileHeightIsOneFramePerCell(t *testing.T) {
	set := structSet(map[byte]*terrain.StructureClass{1: structClass(2, 2, 2)})
	places, counts, _ := terrain.StructurePlacements(structGrid(6, 6, structAt(0, 0, 1)), set, nil, 0)

	const c = structCellSize
	checkEntries(t, places, []wantEntry{
		{1, 0, 1 * c, 0, 0*2 + 1}, {0, 0, 0, 0, 0*2 + 0},
		{1, 1, 1 * c, 1 * c, 1*2 + 1}, {0, 1, 0, 1 * c, 1*2 + 0},
	})
	if counts.Cells.Overhang != 0 {
		t.Errorf("overhang = %d, want 0 — FullHeight == TileHeight has no overhang", counts.Cells.Overhang)
	}
	if counts.Cells.Strips != counts.Cells.Frames {
		t.Errorf("%d strips against %d frames, want one frame per cell",
			counts.Cells.Strips, counts.Cells.Frames)
	}
}

// AC-3, third arm. A rectangle overrunning the last column and the last row: the
// out-of-extent cells are dropped and counted, and NO CELL OF THE NEXT ROW IS
// TOUCHED — the column is tested against the map's width rather than allowed to
// wrap through the row-major index, which is the failure that would put a
// building's right-hand cells at the far left of the row below.
func TestStructureOutOfExtentCellsAreDroppedAndCounted(t *testing.T) {
	// A 3 x 3 class anchored at (2,2) of a 4 x 4 map: columns 2,3,4 and rows
	// 2,3,4, so one column and one row lie outside.
	set := structSet(map[byte]*terrain.StructureClass{1: structClass(3, 3, 3)})
	places, counts, _ := terrain.StructurePlacements(structGrid(4, 4, structAt(2, 2, 1)), set, nil, 0)

	for _, p := range places {
		if p.Cell.X < 0 || p.Cell.X >= 4 || p.Cell.Y < 0 || p.Cell.Y >= 4 {
			t.Errorf("entry at cell %v is outside the 4x4 map", p.Cell)
		}
		if p.Cell.X < 2 || p.Cell.Y < 2 {
			t.Errorf("entry at cell %v is outside the rectangle anchored at (2,2); "+
				"an off-map column wrapped into another row", p.Cell)
		}
	}
	// 9 rectangle cells, 4 of them inside (columns 2..3 x rows 2..3).
	if counts.Cells.Strips != 4 || counts.Cells.Outside != 5 {
		t.Errorf("strips %d outside %d, want 4 and 5", counts.Cells.Strips, counts.Cells.Outside)
	}
	if counts.Cells.Strips+counts.Cells.Outside != 3*3 {
		t.Errorf("strips + outside = %d, want the rectangle's %d cells",
			counts.Cells.Strips+counts.Cells.Outside, 3*3)
	}
	if counts.Placements.Drawn != 1 {
		t.Errorf("drawn = %d, want 1 — a partly clipped placement still drew", counts.Placements.Drawn)
	}
}

// AC-3, fourth arm. A placement anchored wholly outside the map yields nothing,
// and is still counted as DRAWN: it resolved a class the loader could draw, and
// what happened to it is a per-CELL fact, not a per-placement skip.
func TestStructureAnchoredWhollyOutsideYieldsNothing(t *testing.T) {
	set := structSet(map[byte]*terrain.StructureClass{1: structClass(2, 2, 4)})
	places, counts, _ := terrain.StructurePlacements(structGrid(4, 4, structAt(9, 9, 1)), set, nil, 0)

	if len(places) != 0 {
		t.Fatalf("%d entries, want none", len(places))
	}
	if counts.Cells.Strips != 0 || counts.Cells.Outside != 4 {
		t.Errorf("strips %d outside %d, want 0 and 4", counts.Cells.Strips, counts.Cells.Outside)
	}
}

// AC-1's builder half, and the three per-placement skips counted apart: a key
// naming no class, a class the loader resolved but could not draw, and a
// VariableSize class. Each is a SKIP and never an error, and none of the three is
// the other.
func TestStructurePerPlacementSkipsAreCountedApart(t *testing.T) {
	set := structSet(map[byte]*terrain.StructureClass{
		1: structClass(1, 1, 1),
		2: {TileWidth: 1, TileHeight: 1, FullHeight: 1}, // resolved, no art
		3: {TileWidth: 1, TileHeight: 1, FullHeight: 1, VariableSize: true, Frames: structFrames(1)},
		4: {TileWidth: 0, TileHeight: 1, FullHeight: 1, Frames: structFrames(1)}, // a zero extent
	})
	g := structGrid(8, 8,
		structAt(0, 0, 1), structAt(1, 0, 2), structAt(2, 0, 3), structAt(3, 0, 4), structAt(4, 0, 9))
	places, counts, _ := terrain.StructurePlacements(g, set, nil, 0)

	if len(places) != 1 {
		t.Fatalf("%d entries, want 1 — only the whole class draws", len(places))
	}
	want := terrain.StructurePlacementCounts{Drawn: 1, NoClass: 1, Undrawable: 2, VariableSize: 1}
	if counts.Placements != want {
		t.Errorf("placement counts = %+v, want %+v", counts.Placements, want)
	}
	sum := counts.Placements.Drawn + counts.Placements.NoClass +
		counts.Placements.Undrawable + counts.Placements.VariableSize
	if sum != len(g.Structures) {
		t.Errorf("counters sum to %d, want the %d records", sum, len(g.Structures))
	}
}

func TestVerticalWoodenBridgeExpandsTheAuthorSizedNinePatch(t *testing.T) {
	frames := structFrames(9)
	set := structSet(map[byte]*terrain.StructureClass{
		33: {
			TileWidth: 1, TileHeight: 1, FullHeight: 1,
			VariableSize: true, VariableLayout: terrain.VariableStructureVerticalNinePatch,
			Frames: frames,
		},
	})
	rec := structAt(2, 3, 33)
	rec.VariableWidth, rec.VariableHeight = 3, 6
	places, counts, animated := terrain.StructurePlacements(structGrid(10, 12, rec), set, nil, 0)

	if len(places) != 18 || len(animated) != 0 {
		t.Fatalf("placements/animated = %d/%d, want 18/0", len(places), len(animated))
	}
	if want := (terrain.StructurePlacementCounts{Drawn: 1}); counts.Placements != want {
		t.Fatalf("placement counts = %+v, want %+v", counts.Placements, want)
	}
	if want := (terrain.StructureCellCounts{Strips: 18, Frames: 18}); counts.Cells != want {
		t.Fatalf("cell counts = %+v, want %+v", counts.Cells, want)
	}

	wantRows := [][]int{
		{2, 1, 0},
		{5, 4, 3}, {5, 4, 3}, {5, 4, 3}, {5, 4, 3},
		{8, 7, 6},
	}
	n := 0
	for row0, rowFrames := range wantRows {
		for colOrder, frameIndex := range rowFrames {
			p := places[n]
			wantCol, wantRow := 4-colOrder, 3+row0
			if p.Cell != (image.Point{X: wantCol, Y: wantRow}) ||
				p.TopLeft != (image.Point{X: wantCol * structCellSize, Y: wantRow * structCellSize}) ||
				p.GridIndex != frameIndex || p.Frame != frames[frameIndex] {
				t.Fatalf("placement %d = cell %v top %v grid %d frame %p; want (%d,%d) (%d,%d) grid %d frame %p",
					n, p.Cell, p.TopLeft, p.GridIndex, p.Frame,
					wantCol, wantRow, wantCol*structCellSize, wantRow*structCellSize,
					frameIndex, frames[frameIndex])
			}
			n++
		}
	}
}

func TestStructureOrderIsRowsUpColumnsDownTiesInRecordOrder(t *testing.T) {
	a, b, c := structClass(1, 1, 1), structClass(1, 1, 1), structClass(1, 1, 1)
	set := structSet(map[byte]*terrain.StructureClass{1: a, 2: b, 3: c})
	g := structGrid(8, 8,
		structAt(2, 2, 1), // the tie's first record
		structAt(2, 2, 2), // the tie's second
		structAt(1, 2, 3), // same row, lower column: must come AFTER
		structAt(2, 1, 1), // the row above: must come FIRST
		structAt(2, 2, 3), // the tie's third
	)
	places, _, _ := terrain.StructurePlacements(g, set, nil, 0)

	if len(places) != 5 {
		t.Fatalf("%d entries, want 5", len(places))
	}
	wantCells := []image.Point{{X: 2, Y: 1}, {X: 2, Y: 2}, {X: 2, Y: 2}, {X: 2, Y: 2}, {X: 1, Y: 2}}
	for i, w := range wantCells {
		if places[i].Cell != w {
			t.Fatalf("entry %d at cell %v, want %v", i, places[i].Cell, w)
		}
	}
	// The three on (2,2) are the three records in the order the map holds them.
	wantClasses := []*terrain.StructureClass{a, b, c}
	for i, w := range wantClasses {
		if places[i+1].Class != w {
			t.Errorf("tie entry %d took class %p, want %p — record order was not preserved",
				i, places[i+1].Class, w)
		}
	}
}

func TestStructureBuilderIsPureAndTotal(t *testing.T) {
	set := structSet(map[byte]*terrain.StructureClass{
		1: structClass(3, 2, 5), // overhang
		2: structClass(2, 2, 2), // none
		3: structClass(4, 4, 6), // overhang, and wider than the maps below
	})
	grids := []terrain.Grid{
		{},
		{Width: 4, Height: 4},
		structGrid(4, 4, structAt(0, 0, 1), structAt(3, 3, 3), structAt(2, 0, 2)),
		structGrid(1, 1, structAt(0, 0, 3)),
		structGrid(-3, 9, structAt(0, 0, 1)),
		structGrid(5, 5, terrain.StructureRecord{X: 0xffffffff, Y: 0xffffffff, Key: 0xffffff01}),
	}
	for _, g := range grids {
		before := append([]terrain.StructureRecord(nil), g.Structures...)
		one, c1, _ := terrain.StructurePlacements(g, set, nil, 0)
		two, c2, _ := terrain.StructurePlacements(g, set, nil, 0)
		if !reflect.DeepEqual(one, two) || c1 != c2 {
			t.Errorf("two builds of one input disagree (%dx%d)", g.Width, g.Height)
		}
		if !reflect.DeepEqual(before, g.Structures) {
			t.Errorf("the builder mutated the record list (%dx%d)", g.Width, g.Height)
		}
	}
	// A nil bundle is an empty one, never an error and never a panic.
	if places, counts, _ := terrain.StructurePlacements(grids[2], nil, nil, 0); places != nil ||
		counts != (terrain.StructureCounts{}) {
		t.Errorf("a nil bundle gave %d entries and %+v, want none and a zero census", len(places), counts)
	}
}

func TestStructurePlacementCarriesNoAnchor(t *testing.T) {
	want := map[string]bool{"StructureID": true, "Cell": true, "TopLeft": true, "GridIndex": true, "Class": true, "Frame": true}
	ty := reflect.TypeOf(terrain.StructurePlacement{})
	for i := 0; i < ty.NumField(); i++ {
		if !want[ty.Field(i).Name] {
			t.Errorf("StructurePlacement has field %q; a structure has no anchor pixel and no canvas, "+
				"so this type carries identity, the cell, the top-left, the grid index, the class and the frame — and nothing else",
				ty.Field(i).Name)
		}
		delete(want, ty.Field(i).Name)
	}
	for name := range want {
		t.Errorf("StructurePlacement lost field %q", name)
	}
}

func TestStructureDestXDependsOnTheColumnAlone(t *testing.T) {
	set := structSet(map[byte]*terrain.StructureClass{
		1: structClass(3, 2, 5),
		2: structClass(2, 4, 4),
	})
	hills := func(c, r int) int { return 7*c - 3*r }
	for _, corner := range []func(c, r int) int{nil, hills} {
		for _, originY := range []int{0, -19, 64} {
			for _, key := range []uint32{1, 2} {
				places, _, _ := terrain.StructurePlacements(
					structGrid(9, 9, structAt(2, 3, key)), set, corner, originY)
				if len(places) == 0 {
					t.Fatal("no entries; the fixture must place something")
				}
				for _, p := range places {
					if p.TopLeft.X != p.Cell.X*terrain.CellSize {
						t.Fatalf("cell %v has destX %d, want %d — destX carries a term it must not",
							p.Cell, p.TopLeft.X, p.Cell.X*terrain.CellSize)
					}
				}
			}
		}
	}
}

// The key is the LOW BYTE of the stored key and nothing else: a key well
// above 255 resolves through its low byte, and no other byte of it is
// consulted. The mask is 0xff — a 0x7f mask would send key 0x81 to class
// 1.
func TestStructureKeyResolvesThroughItsLowByte(t *testing.T) {
	low, high := structClass(1, 1, 1), structClass(1, 1, 1)
	set := structSet(map[byte]*terrain.StructureClass{0x01: low, 0x81: high})
	places, _, _ := terrain.StructurePlacements(structGrid(4, 4,
		structAt(0, 0, 0xdead0081), structAt(1, 0, 0x00000001)), set, nil, 0)

	if len(places) != 2 {
		t.Fatalf("%d entries, want 2", len(places))
	}
	byCell := map[int]*terrain.StructureClass{}
	for _, p := range places {
		byCell[p.Cell.X] = p.Class
	}
	if byCell[0] != high {
		t.Errorf("key 0xdead0081 resolved to %p, want the class at byte 0x81 (%p)", byCell[0], high)
	}
	if byCell[1] != low {
		t.Errorf("key 0x00000001 resolved to %p, want the class at byte 0x01 (%p)", byCell[1], low)
	}
}

// Byte 0 reaches no class: the registry's ID domain is 1..66, so Classes[0] is
// nil always and a key whose low byte is 0 is a NoClass skip rather than a
// placement of whatever happened to be filled there.
func TestStructureByteZeroNamesNoClass(t *testing.T) {
	set := structSet(map[byte]*terrain.StructureClass{1: structClass(1, 1, 1)})
	places, counts, _ := terrain.StructurePlacements(structGrid(4, 4, structAt(0, 0, 0x100)), set, nil, 0)
	if len(places) != 0 || counts.Placements.NoClass != 1 {
		t.Errorf("%d entries and %d no-class, want none and 1", len(places), counts.Placements.NoClass)
	}
	if set.Classes[0] != nil {
		t.Error("Classes[0] is filled; byte 0 can name no structure class")
	}
}

func TestStructureShortSheetSelectsNoFrameAndStillPlaces(t *testing.T) {
	c := structClass(2, 1, 3)
	c.Frames = c.Frames[:3] // the grid wants 6, the sheet holds 3
	set := structSet(map[byte]*terrain.StructureClass{1: c})
	places, counts, _ := terrain.StructurePlacements(structGrid(4, 4, structAt(0, 0, 1)), set, nil, 0)

	if counts.Cells.Frames != 6 {
		t.Fatalf("%d frames, want the grid's 6 — a short sheet drops no entry", counts.Cells.Frames)
	}
	nil3 := 0
	for _, p := range places {
		if p.Frame == nil {
			nil3++
			if p.GridIndex < len(c.Frames) {
				t.Errorf("grid index %d has no frame though the sheet holds it", p.GridIndex)
			}
		}
	}
	if nil3 != 3 {
		t.Errorf("%d frameless entries, want 3", nil3)
	}
}

// --- the lift (AC-5) ---
//
// A structure stands at ONE height and the whole of it is lifted by that one
// value, so the tests below measure the lift AS DRAWN — the vertical difference
// between the flat list and the displaced one, entry for entry — rather than by
// calling an accessor. What is compared is what a window would paint.

// structLiftOf returns the lift the builder applied to every entry of a
// one-placement map, and fails if the entries do not agree about it.
//
// It builds the SAME map twice, once flat and once with the corner accessor, and
// differences the two lists position by position. So a builder that moved an
// entry, dropped one, or lifted two cells of one building differently is caught
// here rather than in a value comparison against a second copy of the formula.
func structLiftOf(t *testing.T, g terrain.Grid, set *terrain.StructureSet, corner func(c, r int) int, originY int) int {
	t.Helper()
	flat, _, _ := terrain.StructurePlacements(g, set, nil, 0)
	lifted, _, _ := terrain.StructurePlacements(g, set, corner, originY)
	if len(flat) != len(lifted) {
		t.Fatalf("%d flat entries against %d lifted; a geometry moved an entry into or out of the list",
			len(flat), len(lifted))
	}
	if len(flat) == 0 {
		t.Fatal("no entries; the fixture must place something")
	}
	lift := 0
	for i := range flat {
		if flat[i].Cell != lifted[i].Cell || flat[i].GridIndex != lifted[i].GridIndex {
			t.Fatalf("entry %d moved between geometries: %v/%d against %v/%d",
				i, flat[i].Cell, flat[i].GridIndex, lifted[i].Cell, lifted[i].GridIndex)
		}
		if flat[i].TopLeft.X != lifted[i].TopLeft.X {
			t.Fatalf("entry %d moved horizontally: destX %d against %d — the displacement is vertical only",
				i, flat[i].TopLeft.X, lifted[i].TopLeft.X)
		}
		// destY = row*CellSize - lift - originY, so the lift is what the flat
		// entry's Y exceeds the lifted one's by, less the origin.
		d := flat[i].TopLeft.Y - lifted[i].TopLeft.Y - originY
		if i == 0 {
			lift = d
			continue
		}
		if d != lift {
			t.Fatalf("entry %d is lifted by %d and entry 0 by %d; ONE structure stands at ONE height (P-3)",
				i, d, lift)
		}
	}
	return lift
}

// wantLift is the contract's own arithmetic, written out here so the assertion
// compares two independent statements of it rather than one function against
// itself. Every division truncates toward zero, which is Go's own.
func wantLift(anchorCol, anchorRow, tw, th int, corner func(c, r int) int) int {
	x2 := 2*anchorCol + tw
	y2 := 2*anchorRow + th
	return (corner(x2/2, y2/2) + corner((x2+1)/2, y2/2) +
		corner(x2/2, (y2+1)/2) + corner((x2+1)/2, (y2+1)/2)) / 4
}

// AC-5. Three classes at ONE anchor, covering the three parity combinations, over
// a corner field whose four samples are all different.
//
// The heights are deliberately NEGATIVE, so the mean's truncation direction is
// observable: /4 truncates toward zero and >>2 floors, and the two differ exactly
// there.
func TestStructureLiftAtEveryParity(t *testing.T) {
	// A corner field with no symmetry: no two of the four samples any class takes
	// can coincide by accident, and every one of them is negative.
	corner := func(c, r int) int { return -(1 + 7*c + 3*r) }

	const anchorCol, anchorRow = 2, 3
	for _, tc := range []struct{ tw, th int }{
		{3, 3}, // odd x odd — the centre cell's own four corners
		{4, 3}, // even x odd
		{4, 4}, // even x even
	} {
		set := structSet(map[byte]*terrain.StructureClass{1: structClass(tc.tw, tc.th, tc.th+2)})
		g := structGrid(16, 16, structAt(anchorCol, anchorRow, 1))

		got := structLiftOf(t, g, set, corner, 0)
		want := wantLift(anchorCol, anchorRow, tc.tw, tc.th, corner)
		if got != want {
			t.Errorf("%dx%d: lift = %d, want %d", tc.tw, tc.th, got, want)
		}
	}

	t.Run("the odd/odd case is the centre cell's four-corner mean", func(t *testing.T) {
		// A 3x3 at (2,3) has its centre cell at (3,4), and the contract's four
		// samples must be that cell's own four corners — BY ARITHMETIC, not by a
		// special case, which is why this is asserted against an independent
		// four-corner mean rather than against wantLift again.
		set := structSet(map[byte]*terrain.StructureClass{1: structClass(3, 3, 5)})
		g := structGrid(16, 16, structAt(anchorCol, anchorRow, 1))
		const cc, cr = anchorCol + 1, anchorRow + 1
		want := (corner(cc, cr) + corner(cc+1, cr) + corner(cc, cr+1) + corner(cc+1, cr+1)) / 4
		if got := structLiftOf(t, g, set, corner, 0); got != want {
			t.Errorf("lift = %d, want the centre cell (%d,%d)'s four-corner mean %d", got, cc, cr, want)
		}
	})

	t.Run("the mean truncates toward zero", func(t *testing.T) {
		// Four corners summing to -6: /4 is -1 and >>2 is -2. A halving written
		// as a shift is a different function at either sign, and the object path
		// this one is aligned with truncates.
		field := func(c, r int) int {
			if c == 4 && r == 4 {
				return -3
			}
			return -1
		}
		set := structSet(map[byte]*terrain.StructureClass{1: structClass(3, 3, 3)})
		g := structGrid(16, 16, structAt(3, 3, 1))
		if got, want := structLiftOf(t, g, set, field, 0), -1; got != want {
			t.Errorf("lift = %d, want %d — the mean of -1,-1,-1,-3 truncates toward zero", got, want)
		}
	})
}

// The lift is SUBTRACTED, so higher ground raises a structure UP the screen, and
// so is originY. Both terms reach destY alone.
func TestStructureLiftIsSubtractedAndSoIsTheOrigin(t *testing.T) {
	set := structSet(map[byte]*terrain.StructureClass{1: structClass(2, 2, 4)})
	g := structGrid(12, 12, structAt(4, 4, 1))

	high := func(c, r int) int { return 40 }
	low := func(c, r int) int { return 8 }

	// A structure on higher ground stands further UP the image.
	hi, _, _ := terrain.StructurePlacements(g, set, high, 0)
	lo, _, _ := terrain.StructurePlacements(g, set, low, 0)
	if hi[0].TopLeft.Y >= lo[0].TopLeft.Y {
		t.Errorf("at height 40 destY is %d and at height 8 it is %d; the lift must be subtracted",
			hi[0].TopLeft.Y, lo[0].TopLeft.Y)
	}
	if got := structLiftOf(t, g, set, high, 0); got != 40 {
		t.Errorf("lift = %d, want 40 — a flat corner field lifts by its own height", got)
	}

	// originY is subtracted too, and it is a term of its own: at one corner field
	// two origins differ by exactly the origin difference, in Y and by zero in X.
	a, _, _ := terrain.StructurePlacements(g, set, high, 0)
	b, _, _ := terrain.StructurePlacements(g, set, high, 17)
	for i := range a {
		if a[i].TopLeft.Y-b[i].TopLeft.Y != 17 {
			t.Fatalf("entry %d moved by %d for an origin of 17", i, a[i].TopLeft.Y-b[i].TopLeft.Y)
		}
		if a[i].TopLeft.X != b[i].TopLeft.X {
			t.Fatalf("entry %d moved horizontally with the origin", i)
		}
	}
}

func TestStructureLiftIsOnePerStructureOverRandomFields(t *testing.T) {
	set := structSet(map[byte]*terrain.StructureClass{
		1: structClass(3, 2, 5),
		2: structClass(2, 4, 4),
	})
	// A deterministic pseudo-random field: no clock, no math/rand, and the same
	// grids on every run, so a failure is reproducible from the seed alone.
	seed := uint32(0x9e3779b9)
	next := func() int {
		seed ^= seed << 13
		seed ^= seed >> 17
		seed ^= seed << 5
		return int(int8(seed)) // signed, as an altitude is read
	}
	for trial := 0; trial < 32; trial++ {
		field := map[[2]int]int{}
		corner := func(c, r int) int {
			k := [2]int{c, r}
			if v, ok := field[k]; ok {
				return v
			}
			v := next()
			field[k] = v
			return v
		}
		g := structGrid(20, 20, structAt(2, 2, 1), structAt(11, 9, 2))
		flat, _, _ := terrain.StructurePlacements(g, set, nil, 0)
		lifted, _, _ := terrain.StructurePlacements(g, set, corner, 0)
		if len(flat) != len(lifted) {
			t.Fatalf("trial %d: %d flat entries against %d lifted", trial, len(flat), len(lifted))
		}
		// Group the deltas by the structure each entry belongs to. The two
		// placements are of two different classes and share no rectangle cell
		// here, so the class pointer identifies the structure.
		byClass := map[*terrain.StructureClass]int{}
		for i := range flat {
			d := flat[i].TopLeft.Y - lifted[i].TopLeft.Y
			if flat[i].TopLeft.X != lifted[i].TopLeft.X {
				t.Fatalf("trial %d entry %d moved horizontally", trial, i)
			}
			if prev, seen := byClass[flat[i].Class]; seen && prev != d {
				t.Fatalf("trial %d: one structure's entries are lifted by %d and by %d", trial, prev, d)
			}
			byClass[flat[i].Class] = d
		}
		if len(byClass) != 2 {
			t.Fatalf("trial %d: %d structures placed, want 2", trial, len(byClass))
		}
	}
}

// --- animation: the timeline, the mask and the ranks (AC-2, AC-4) ---

// structAnimClass is a class whose sheet carries the base block TWICE over plus
// one animation block: TW*FH*2 + P*L frames, which is the identity 64 of the 66
// shipped classes satisfy (SPR256-STR-041). mask is the grid picture, live bytes
// and '-' mixed; timeline is the expansion a counter walks.
//
// The ranks are computed here, in the test, by a walk written out in full rather
// than by calling the loader's — the two must agree by arriving at the same
// numbers from the same mask, not by sharing a function.
func structAnimClass(tw, th, fh, phases int, mask string, timeline []int) *terrain.StructureClass {
	cells := tw * fh
	if len(mask) != cells {
		panic("fixture: the mask must be the grid's own length")
	}
	rank := make([]int, cells)
	live := 0
	for i := 0; i < cells; i++ {
		if mask[i] == '-' {
			rank[i] = -1
			continue
		}
		rank[i] = live
		live++
	}
	return &terrain.StructureClass{
		TileWidth:  tw,
		TileHeight: th,
		FullHeight: fh,
		Frames:     structFrames(cells*2 + phases*live),
		Timeline:   timeline,
		Rank:       rank,
		Live:       live,
	}
}

// AC-4's animation arm. Every arm of *The three blocks* is taken exactly when its
// stated condition holds, at every counter of two whole cycles.
//
// The class is 2x1 with FullHeight 3 — six grid cells — and its mask retires
// three of them, so the live cells rank 0, 1, 2 in grid order and the animation
// block's stride is 3 per phase.
func TestStructureSelectsTheThreeBlocks(t *testing.T) {
	const tw, fh, phases = 2, 3, 3
	const mask = "x-x-x-" // live at grid indices 0, 2, 4 — ranks 0, 1, 2
	timeline := []int{0, 1, 2, 3, 1}
	c := structAnimClass(tw, 1, fh, phases, mask, timeline)
	cells := tw * fh

	// Two whole cycles, so a counter that wrapped is exercised as well as one
	// that did not.
	for counter := uint32(0); counter < uint32(2*len(timeline)); counter++ {
		phase := timeline[int(counter)%len(timeline)]
		for i := 0; i < cells; i++ {
			got := terrain.SelectStructureFrame(c, i, counter, true, false)

			want := i // the base arm
			if phase > 0 && mask[i] != '-' {
				rank := 0
				for k := 0; k < i; k++ {
					if mask[k] != '-' {
						rank++
					}
				}
				want = cells + (phase-1)*3 + rank
			}
			if got != want {
				t.Fatalf("counter %d phase %d grid %d: frame %d, want %d", counter, phase, i, got, want)
			}
			if got < 0 || got >= len(c.Frames) {
				t.Fatalf("counter %d grid %d selected %d, outside a sheet of %d frames (P-4)",
					counter, i, got, len(c.Frames))
			}
		}
	}

	t.Run("a phase of 0 draws base", func(t *testing.T) {
		// timeline[0] is 0, so counter 0 is the zero-phase arm at a LIVE cell —
		// the one case a gate that only tested the mask would get wrong.
		if got := terrain.SelectStructureFrame(c, 0, 0, true, false); got != 0 {
			t.Errorf("a live cell at phase 0 selected %d, want its base frame 0", got)
		}
	})

	t.Run("a retired cell draws base at every counter", func(t *testing.T) {
		for counter := uint32(0); counter < 16; counter++ {
			if got := terrain.SelectStructureFrame(c, 1, counter, true, false); got != 1 {
				t.Fatalf("counter %d: the retired cell 1 selected %d, want 1", counter, got)
			}
		}
	})

	t.Run("the switch off draws base at every counter", func(t *testing.T) {
		for counter := uint32(0); counter < 16; counter++ {
			for i := 0; i < cells; i++ {
				if got := terrain.SelectStructureFrame(c, i, counter, false, false); got != i {
					t.Fatalf("counter %d grid %d: the switch is off and it selected %d", counter, i, got)
				}
			}
		}
	})

	t.Run("a class with no cycle draws base", func(t *testing.T) {
		bare := structClass(tw, 1, fh)
		for counter := uint32(0); counter < 8; counter++ {
			for i := 0; i < cells; i++ {
				if got := terrain.SelectStructureFrame(bare, i, counter, true, false); got != i {
					t.Fatalf("counter %d grid %d: a class with no timeline selected %d", counter, i, got)
				}
			}
		}
	})
}

func TestStructureSelectionNeverLeavesTheSheet(t *testing.T) {
	const tw, fh = 2, 3
	const mask = "xxxxxx"
	timeline := []int{1, 2, 3}

	for _, frames := range []int{0, 1, 5, 6, 7, 9, 12, 15} {
		c := structAnimClass(tw, 1, fh, 3, mask, timeline)
		if frames <= len(c.Frames) {
			c.Frames = c.Frames[:frames]
		}
		for counter := uint32(0); counter < 12; counter++ {
			for i := 0; i < tw*fh; i++ {
				got := terrain.SelectStructureFrame(c, i, counter, true, false)
				if got < 0 {
					t.Fatalf("%d frames, counter %d, grid %d: selected %d", frames, counter, i, got)
				}
				if got >= len(c.Frames) && got != i {
					t.Fatalf("%d frames, counter %d, grid %d: selected %d, which is neither inside "+
						"the sheet nor that index's own base frame", frames, counter, i, got)
				}
				if got != i && got < tw*fh {
					t.Fatalf("%d frames, counter %d, grid %d: selected %d, inside the base block but "+
						"not this index's own frame", frames, counter, i, got)
				}
			}
		}
	}
}

func TestAnimateStructuresPatchesFramesAndNothingElse(t *testing.T) {
	const mask = "x-x-x-"
	c := structAnimClass(2, 1, 3, 3, mask, []int{1, 2})
	set := structSet(map[byte]*terrain.StructureClass{1: c})
	places, _, animated := terrain.StructurePlacements(structGrid(8, 8, structAt(1, 1, 1)), set, nil, 0)

	if len(places) != 6 {
		t.Fatalf("%d entries, want the grid's 6", len(places))
	}
	// Three of the six grid cells are live, so exactly three entries can change.
	if len(animated) != 3 {
		t.Fatalf("%d animated entries, want 3 — the mask leaves three cells live", len(animated))
	}
	for _, i := range animated {
		if c.Rank[places[i].GridIndex] < 0 {
			t.Fatalf("entry %d is in the subset with a retired grid cell %d", i, places[i].GridIndex)
		}
	}

	var scratch []terrain.StructurePlacement
	for counter := uint32(0); counter < 6; counter++ {
		out := terrain.AnimateStructures(scratch, places, animated, counter, true, false)
		if len(out) != len(places) {
			t.Fatalf("the pass returned %d entries, want %d", len(out), len(places))
		}
		for i := range out {
			if out[i].Cell != places[i].Cell || out[i].TopLeft != places[i].TopLeft ||
				out[i].GridIndex != places[i].GridIndex || out[i].Class != places[i].Class {
				t.Fatalf("counter %d entry %d moved: %+v against %+v", counter, i, out[i], places[i])
			}
			want := c.Frames[terrain.SelectStructureFrame(c, out[i].GridIndex, counter, true, false)]
			if out[i].Frame != want {
				t.Fatalf("counter %d entry %d drew the wrong frame", counter, i)
			}
		}
		scratch = out
	}

	t.Run("the switch off hands back the built list itself", func(t *testing.T) {
		out := terrain.AnimateStructures(nil, places, animated, 3, false, false)
		if len(out) != len(places) || &out[0] != &places[0] {
			t.Error("the pass copied the list with the switch off; it can change nothing there")
		}
	})

	t.Run("an empty subset hands back the built list itself", func(t *testing.T) {
		out := terrain.AnimateStructures(nil, places, nil, 3, true, false)
		if len(out) != len(places) || &out[0] != &places[0] {
			t.Error("the pass copied the list with an empty subset")
		}
	})

	t.Run("a class with no cycle enters no subset", func(t *testing.T) {
		bare := structSet(map[byte]*terrain.StructureClass{1: structClass(2, 1, 3)})
		_, _, animated := terrain.StructurePlacements(structGrid(8, 8, structAt(1, 1, 1)), bare, nil, 0)
		if len(animated) != 0 {
			t.Errorf("%d animated entries for a class with no timeline, want none", len(animated))
		}
	})
}

// --- the two passes, and the merge with the object plane (AC-7) ---

// structObject is one object placement standing on a cell. Only its Cell reaches
// the merge, which compares rectangle rows and nothing else.
func structObject(col, row int) terrain.StaticPlacement {
	return terrain.StaticPlacement{Cell: image.Point{X: col, Y: row}}
}

// planeCells renders a merged order as one readable string per ref, so a failed
// expectation says WHAT was drawn in what order rather than which index moved.
func planeCells(order []terrain.PlaneRef, structures []terrain.StructurePlacement,
	objects []terrain.StaticPlacement) []string {
	out := make([]string, 0, len(order))
	for _, ref := range order {
		if ref.Structure {
			p := structures[ref.Index]
			out = append(out, fmt.Sprintf("S(%d,%d)", p.Cell.X, p.Cell.Y))
			continue
		}
		p := objects[ref.Index]
		out = append(out, fmt.Sprintf("O(%d,%d)", p.Cell.X, p.Cell.Y))
	}
	return out
}

// AC-7. Two structures on adjacent rows whose art overlaps, one Flat and one
// not, beside a static-object layer.
//
// The flat one precedes EVERY static object; the other falls in rectangle-row
// order among them; within a row columns descend; equal cells keep record order.
func TestStructurePlanesMergeByRow(t *testing.T) {
	// The non-flat class is 1x1 with FullHeight 3, so its art reaches two whole
	// cells above its own row and overlaps the flat one standing in front of it.
	tall := structClass(1, 1, 3)
	flat := structClass(1, 1, 1)
	flat.Flat = true
	set := structSet(map[byte]*terrain.StructureClass{1: tall, 2: flat})

	g := structGrid(8, 8,
		structAt(4, 2, 1), // the tall one, row 2
		structAt(4, 3, 2), // the flat one, row 3 — the row in front of it
	)
	structures, _, _ := terrain.StructurePlacements(g, set, nil, 0)
	objects := []terrain.StaticPlacement{
		structObject(1, 0), structObject(6, 2), structObject(0, 5),
	}

	order := terrain.PlaneOrder(structures, objects)
	got := planeCells(order, structures, objects)

	// The flat entry first, whatever its row; then the merge: the object on row
	// 0, then the tall structure's three entries on row 2 BEFORE the object on
	// that same row, then the object on row 5.
	want := []string{"S(4,3)", "O(1,0)", "S(4,2)", "S(4,2)", "S(4,2)", "O(6,2)", "O(0,5)"}
	if len(got) != len(want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}

	t.Run("no entry is drawn twice", func(t *testing.T) {
		seen := map[terrain.PlaneRef]bool{}
		for _, ref := range order {
			if seen[ref] {
				t.Fatalf("%+v appears twice in the order", ref)
			}
			seen[ref] = true
		}
		if len(seen) != len(structures)+len(objects) {
			t.Fatalf("%d refs, want one per drawable (%d structures, %d objects)",
				len(seen), len(structures), len(objects))
		}
	})

	t.Run("a flat class is in the early pass only", func(t *testing.T) {
		// Every flat entry precedes every object AND every non-flat structure
		// entry, which is what "before everything else on the map" means.
		firstOther := len(order)
		for i, ref := range order {
			if !ref.Structure || !structures[ref.Index].Class.Flat {
				firstOther = i
				break
			}
		}
		for _, ref := range order[firstOther:] {
			if ref.Structure && structures[ref.Index].Class.Flat {
				t.Fatal("a flat entry is drawn after a non-flat drawable")
			}
		}
	})
}

// The merge compares ROWS and never columns, and at an equal row the structure
// side goes first. Both are separable only by a fixture where the two orders
// disagree, which is what the columns below are for.
func TestStructureMergeComparesRowsAndLeadsAtATie(t *testing.T) {
	set := structSet(map[byte]*terrain.StructureClass{1: structClass(1, 1, 1)})
	structures, _, _ := terrain.StructurePlacements(structGrid(12, 12, structAt(1, 4, 1)), set, nil, 0)
	// The structure stands at a LOW column on a MIDDLE row. The first object is
	// on an EARLIER row at a HIGHER column, so a merge comparing columns stops
	// before it and emits the structure first; the second is on a LATER row at a
	// lower column. Only the row comparison puts the first object in front.
	objects := []terrain.StaticPlacement{structObject(9, 2), structObject(0, 5)}

	got := planeCells(terrain.PlaneOrder(structures, objects), structures, objects)
	want := []string{"O(9,2)", "S(1,4)", "O(0,5)"}
	for i := range want {
		if i >= len(got) || got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestStructureMergeIsTheObjectListWithoutStructures(t *testing.T) {
	objects := []terrain.StaticPlacement{
		structObject(3, 0), structObject(1, 1), structObject(2, 1), structObject(0, 9),
	}
	for _, structures := range [][]terrain.StructurePlacement{nil, {}} {
		order := terrain.PlaneOrder(structures, objects)
		if len(order) != len(objects) {
			t.Fatalf("%d refs, want the %d objects", len(order), len(objects))
		}
		for i, ref := range order {
			if ref.Structure || ref.Index != i {
				t.Fatalf("ref %d = %+v, want the object at index %d", i, ref, i)
			}
		}
	}

	// And with no objects it is the structure list in its own order.
	set := structSet(map[byte]*terrain.StructureClass{1: structClass(2, 1, 2)})
	structures, _, _ := terrain.StructurePlacements(structGrid(8, 8, structAt(1, 1, 1)), set, nil, 0)
	order := terrain.PlaneOrder(structures, nil)
	if len(order) != len(structures) {
		t.Fatalf("%d refs, want the %d entries", len(order), len(structures))
	}
	for i, ref := range order {
		if !ref.Structure || ref.Index != i {
			t.Fatalf("ref %d = %+v, want the structure entry at index %d", i, ref, i)
		}
	}
}

// The merge is pure and total: any two lists yield an order, nothing is mutated,
// and every drawable appears exactly once whatever the rows hold.
func TestStructureMergeIsPureAndTotal(t *testing.T) {
	set := structSet(map[byte]*terrain.StructureClass{
		1: structClass(2, 2, 4),
		2: func() *terrain.StructureClass { c := structClass(1, 1, 1); c.Flat = true; return c }(),
	})
	structures, _, _ := terrain.StructurePlacements(structGrid(10, 10,
		structAt(0, 0, 1), structAt(5, 5, 2), structAt(3, 7, 1)), set, nil, 0)
	objects := []terrain.StaticPlacement{
		structObject(0, 0), structObject(9, 0), structObject(4, 5), structObject(2, 9),
	}

	before := append([]terrain.StructurePlacement(nil), structures...)
	one := terrain.PlaneOrder(structures, objects)
	two := terrain.PlaneOrder(structures, objects)
	if !reflect.DeepEqual(one, two) {
		t.Error("two merges of one input disagree")
	}
	if !reflect.DeepEqual(before, structures) {
		t.Error("the merge mutated the structure list")
	}
	if len(one) != len(structures)+len(objects) {
		t.Fatalf("%d refs, want %d", len(one), len(structures)+len(objects))
	}
	// Every object index appears exactly once, in ascending order: the object
	// layer's own list is consumed as the builder made it, never reordered.
	last := -1
	for _, ref := range one {
		if ref.Structure {
			continue
		}
		if ref.Index != last+1 {
			t.Fatalf("object refs came out as %d after %d; the object list was reordered", ref.Index, last)
		}
		last = ref.Index
	}
	if last != len(objects)-1 {
		t.Fatalf("%d object refs, want %d", last+1, len(objects))
	}
}

// --- the ruin diagnostic (AC-4's ruin arm) ---

// AC-4's ruin arm. The block is addressed FROM THE END OF THE SHEET, a class with
// Indestructible set returns to base, and a sheet with no ruin block redraws its
// intact art.
func TestStructureRuinBlockIsAddressedFromTheEnd(t *testing.T) {
	const tw, fh, phases = 2, 3, 3
	const mask = "x-x-x-"
	cells := tw * fh
	c := structAnimClass(tw, 1, fh, phases, mask, []int{1, 2, 3})

	// cells*2 + phases*live = 12 + 9 = 21 frames, so the ruin base is 21-6 = 15
	// — which is NOT 2*cells, and the two are separable only because the
	// animation block's length is not the base block's.
	if len(c.Frames) != cells*2+phases*3 {
		t.Fatalf("the fixture sheet holds %d frames, want %d", len(c.Frames), cells*2+phases*3)
	}
	ruinBase := len(c.Frames) - cells
	if ruinBase == 2*cells {
		t.Fatal("the fixture cannot separate a ruin base taken from the end from one taken as 2*TW*FH")
	}

	for counter := uint32(0); counter < 8; counter++ {
		for i := 0; i < cells; i++ {
			if got, want := terrain.SelectStructureFrame(c, i, counter, true, true), ruinBase+i; got != want {
				t.Fatalf("counter %d grid %d: ruined selection %d, want %d — the block is addressed "+
					"from the END of the file", counter, i, got, want)
			}
		}
	}

	t.Run("a ruined structure does not animate", func(t *testing.T) {
		// The arms are ORDERED, not conjoined: the ruin arm is taken first, so a
		// live mask cell at a non-zero phase still draws its ruin frame.
		if got, want := terrain.SelectStructureFrame(c, 0, 0, true, true), ruinBase; got != want {
			t.Errorf("a live cell at phase %d selected %d while ruined, want %d",
				c.Timeline[0], got, want)
		}
	})

	t.Run("Indestructible returns a destroyed selection to base", func(t *testing.T) {
		hard := structAnimClass(tw, 1, fh, phases, mask, []int{1, 2, 3})
		hard.Indestructible = true
		for i := 0; i < cells; i++ {
			// With the animation switch off, the answer is the base frame; with it
			// on, the ordinary animation arm. Neither is the ruin block.
			if got := terrain.SelectStructureFrame(hard, i, 0, false, true); got != i {
				t.Fatalf("grid %d: an Indestructible class selected %d while ruined, want its base %d",
					i, got, i)
			}
			if got := terrain.SelectStructureFrame(hard, i, 1, true, true); got >= ruinBase {
				t.Fatalf("grid %d: an Indestructible class reached the ruin block at %d", i, got)
			}
		}
	})

	t.Run("a sheet with no ruin block redraws its intact art", func(t *testing.T) {
		bare := structClass(tw, 1, fh) // exactly the base block, nothing after it
		for i := 0; i < cells; i++ {
			if got := terrain.SelectStructureFrame(bare, i, 3, true, true); got != i {
				t.Fatalf("grid %d: a sheet of the base block alone selected %d while ruined, want %d",
					i, got, i)
			}
		}
		// And a sheet SHORT of even its base block selects nothing outside itself.
		short := structClass(tw, 1, fh)
		short.Frames = short.Frames[:2]
		for i := 0; i < cells; i++ {
			got := terrain.SelectStructureFrame(short, i, 3, true, true)
			if got < 0 || (got >= len(short.Frames) && got != i) {
				t.Fatalf("grid %d: a short sheet selected %d (P-4)", i, got)
			}
		}
	})
}

// The ruin pass walks EVERY entry, not the animated subset: every drawable
// structure is destroyed at once under the diagnostic, and a class that does not
// animate has no entry in that subset at all.
func TestAnimateStructuresRuinsEveryEntry(t *testing.T) {
	// A class with NO cycle, so the animated subset is empty and a pass that
	// walked only it would change nothing.
	c := structClass(2, 1, 3)
	c.Frames = append(c.Frames, structFrames(6)...) // base block twice: a ruin block
	set := structSet(map[byte]*terrain.StructureClass{1: c})
	places, _, animated := terrain.StructurePlacements(structGrid(8, 8, structAt(1, 1, 1)), set, nil, 0)

	if len(animated) != 0 {
		t.Fatalf("%d animated entries, want none — the fixture class has no cycle", len(animated))
	}
	out := terrain.AnimateStructures(nil, places, animated, 0, true, true)
	if len(out) != len(places) {
		t.Fatalf("the ruin pass returned %d entries, want %d", len(out), len(places))
	}
	ruinBase := len(c.Frames) - c.GridCells()
	for i := range out {
		if out[i].Cell != places[i].Cell || out[i].TopLeft != places[i].TopLeft ||
			out[i].GridIndex != places[i].GridIndex {
			t.Fatalf("entry %d moved under the ruin switch", i)
		}
		if want := c.Frames[ruinBase+out[i].GridIndex]; out[i].Frame != want {
			t.Fatalf("entry %d drew its intact frame under the ruin switch", i)
		}
	}

	t.Run("the switch off leaves the built list itself", func(t *testing.T) {
		out := terrain.AnimateStructures(nil, places, animated, 0, true, false)
		if len(out) != len(places) || &out[0] != &places[0] {
			t.Error("the pass copied the list with neither switch able to change anything")
		}
	})

	t.Run("Indestructible survives the pass", func(t *testing.T) {
		hard := structClass(2, 1, 3)
		hard.Frames = append(hard.Frames, structFrames(6)...)
		hard.Indestructible = true
		set := structSet(map[byte]*terrain.StructureClass{1: hard})
		places, _, animated := terrain.StructurePlacements(structGrid(8, 8, structAt(1, 1, 1)), set, nil, 0)
		out := terrain.AnimateStructures(nil, places, animated, 0, true, true)
		for i := range out {
			if out[i].Frame != places[i].Frame {
				t.Fatalf("entry %d of an Indestructible class changed under the ruin switch", i)
			}
		}
	})

	t.Run("the census does not move", func(t *testing.T) {
		// The switch changes the picture and no count: it is read at the draw and
		// the build never sees it.
		_, before, _ := terrain.StructurePlacements(structGrid(8, 8, structAt(1, 1, 1)), set, nil, 0)
		out := terrain.AnimateStructures(nil, places, animated, 5, true, true)
		if len(out) != before.Cells.Frames {
			t.Errorf("%d entries under the ruin switch, want the build's %d", len(out), before.Cells.Frames)
		}
	})
}

// --- the census (AC-9's census clause, AC-10's shape) ---

func TestStructureCensusIsComplete(t *testing.T) {
	classes := map[byte]*terrain.StructureClass{
		1: structClass(3, 2, 5),
		2: structClass(2, 2, 2),
		3: structClass(4, 3, 7),
		4: {TileWidth: 1, TileHeight: 1, FullHeight: 1},                          // artless
		5: {TileWidth: 1, TileHeight: 1, FullHeight: 1, VariableSize: true},      // a bridge
		6: {TileWidth: 0, TileHeight: 0, FullHeight: 0, Frames: structFrames(1)}, // no rectangle
	}
	set := structSet(classes)
	// The extents of the drawable classes, for the second identity's own sum.
	rect := map[byte][2]int{1: {3, 2}, 2: {2, 2}, 3: {4, 3}}

	seed := uint32(0x243f6a88)
	next := func(n int) int {
		seed ^= seed << 13
		seed ^= seed >> 17
		seed ^= seed << 5
		return int(seed>>8) % n
	}
	for trial := 0; trial < 64; trial++ {
		w, h := 1+next(12), 1+next(12)
		recs := make([]terrain.StructureRecord, 0, 8)
		for i := 0; i < next(9); i++ {
			// Anchors reach two cells PAST the map on either axis, so a
			// rectangle that overruns the edge is the ordinary case here.
			recs = append(recs, structAt(next(w+2), next(h+2), uint32(1+next(9))))
		}
		g := structGrid(w, h, recs...)
		places, counts, _ := terrain.StructurePlacements(g, set, nil, 0)

		p := counts.Placements
		if got := p.Drawn + p.NoClass + p.Undrawable + p.VariableSize; got != len(recs) {
			t.Fatalf("trial %d (%dx%d): counters sum to %d, want the %d records: %+v",
				trial, w, h, got, len(recs), p)
		}

		// The cells of every DRAWN placement's rectangle, summed independently.
		wantCells := 0
		for _, r := range recs {
			c := set.Classes[byte(r.Key)]
			if c == nil || c.VariableSize {
				continue
			}
			e, ok := rect[byte(r.Key)]
			if !ok {
				continue
			}
			wantCells += e[0] * e[1]
		}
		if got := counts.Cells.Strips + counts.Cells.Outside; got != wantCells {
			t.Fatalf("trial %d (%dx%d): strips %d + outside %d = %d, want the drawn rectangles' %d cells",
				trial, w, h, counts.Cells.Strips, counts.Cells.Outside, got, wantCells)
		}

		// Frames counts ENTRIES and Overhang is a subset of them, never a fifth
		// counter and never a cell.
		if counts.Cells.Frames != len(places) {
			t.Fatalf("trial %d: %d frames counted against %d entries", trial, counts.Cells.Frames, len(places))
		}
		if counts.Cells.Overhang > counts.Cells.Frames {
			t.Fatalf("trial %d: %d overhang frames of %d frames", trial, counts.Cells.Overhang, counts.Cells.Frames)
		}
		// An out-of-extent cell contributes no entry: every entry stands inside
		// the map.
		for _, e := range places {
			if e.Cell.X < 0 || e.Cell.X >= w || e.Cell.Y < 0 || e.Cell.Y >= h {
				t.Fatalf("trial %d: an entry stands at %v, outside the %dx%d map", trial, e.Cell, w, h)
			}
		}
	}
}

// A drawn placement wholly inside the map contributes TileWidth*FullHeight frames
// — one per grid index — of which TileWidth*(FullHeight-TileHeight) are overhang.
// The overhang is counted as a FRAME and never as a cell of its own: it stacks
// above the back row rather than standing on ground.
func TestStructureOverhangIsCountedAsAFrame(t *testing.T) {
	for _, tc := range []struct{ tw, th, fh int }{{3, 2, 5}, {2, 2, 2}, {4, 3, 7}, {1, 1, 9}} {
		set := structSet(map[byte]*terrain.StructureClass{1: structClass(tc.tw, tc.th, tc.fh)})
		_, counts, _ := terrain.StructurePlacements(structGrid(16, 16, structAt(2, 2, 1)), set, nil, 0)

		if got, want := counts.Cells.Strips, tc.tw*tc.th; got != want {
			t.Errorf("%dx%d/%d: %d strips, want the rectangle's %d cells", tc.tw, tc.th, tc.fh, got, want)
		}
		if got, want := counts.Cells.Frames, tc.tw*tc.fh; got != want {
			t.Errorf("%dx%d/%d: %d frames, want the grid's %d", tc.tw, tc.th, tc.fh, got, want)
		}
		if got, want := counts.Cells.Overhang, tc.tw*(tc.fh-tc.th); got != want {
			t.Errorf("%dx%d/%d: %d overhang frames, want %d", tc.tw, tc.th, tc.fh, got, want)
		}
	}
}

func TestStructureCensusLinesNameEveryCounter(t *testing.T) {
	counts := terrain.StructureCounts{
		Placements: terrain.StructurePlacementCounts{Drawn: 137, NoClass: 0, Undrawable: 0, VariableSize: 2},
		Cells:      terrain.StructureCellCounts{Strips: 402, Frames: 913, Overhang: 511, Outside: 6},
	}
	placements, cells := counts.CensusLines()
	if want := "structures: 137 drawn, 0 no class, 0 undrawable, 2 variable-size"; placements != want {
		t.Errorf("placements line = %q, want %q", placements, want)
	}
	if want := "cells: 402 strips, 913 frames (511 overhang), 6 outside the map"; cells != want {
		t.Errorf("cells line = %q, want %q", cells, want)
	}

	// A zero census still names all eight counters.
	zeroP, zeroC := terrain.StructureCounts{}.CensusLines()
	for _, tc := range []struct {
		line  string
		words []string
	}{
		{zeroP, []string{"drawn", "no class", "undrawable", "variable-size"}},
		{zeroC, []string{"strips", "frames", "overhang", "outside"}},
	} {
		for _, w := range tc.words {
			if !strings.Contains(tc.line, w) {
				t.Errorf("a zero census printed %q, which does not name %q", tc.line, w)
			}
		}
	}
}

// AC-10's first observation, as the harness asks for it: which class keys carry
// VariableSize, ascending, out of the bundle itself.
func TestStructureVariableSizeIDs(t *testing.T) {
	set := structSet(map[byte]*terrain.StructureClass{
		1:  structClass(1, 1, 1),
		33: {TileWidth: 1, TileHeight: 1, FullHeight: 1, VariableSize: true},
		37: {TileWidth: 1, TileHeight: 1, FullHeight: 1, VariableSize: true},
	})
	got := set.VariableSizeIDs()
	want := []int{33, 37}
	if len(got) != len(want) {
		t.Fatalf("VariableSizeIDs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("VariableSizeIDs() = %v, want %v", got, want)
		}
	}
	if ids := (*terrain.StructureSet)(nil).VariableSizeIDs(); ids != nil {
		t.Errorf("a nil bundle reported %v, want none", ids)
	}
}

// depthCells renders a three-way merged order as one readable string per ref,
// exactly as planeCells does for the two-way one.
func depthCells(order []terrain.DepthRef, structures []terrain.StructurePlacement,
	objects, entities []terrain.StaticPlacement) []string {
	out := make([]string, 0, len(order))
	for _, ref := range order {
		switch ref.Kind {
		case terrain.PlaneStructure:
			p := structures[ref.Index]
			out = append(out, fmt.Sprintf("S(%d,%d)", p.Cell.X, p.Cell.Y))
		case terrain.PlaneEntity:
			p := entities[ref.Index]
			out = append(out, fmt.Sprintf("E(%d,%d)", p.Cell.X, p.Cell.Y))
		default:
			p := objects[ref.Index]
			out = append(out, fmt.Sprintf("O(%d,%d)", p.Cell.X, p.Cell.Y))
		}
	}
	return out
}

func TestDepthOrderMergesEntitiesByRow(t *testing.T) {
	tall := structClass(1, 1, 3)
	flat := structClass(1, 1, 1)
	flat.Flat = true
	set := structSet(map[byte]*terrain.StructureClass{1: tall, 2: flat})

	g := structGrid(8, 8,
		structAt(4, 2, 1), // the tall one, row 2
		structAt(4, 7, 2), // the flat one, on the last row of the map
	)
	structures, _, _ := terrain.StructurePlacements(g, set, nil, 0)
	objects := []terrain.StaticPlacement{structObject(1, 0), structObject(0, 5)}
	entities := []terrain.StaticPlacement{
		structObject(3, 5), // last of the three by row, first by id
		structObject(2, 2), // the structure's own row: behind it in id, in FRONT on screen
		structObject(7, 1), // between the two objects
	}

	order := terrain.DepthOrder(nil, terrain.PlaneOrder(structures, objects), structures, objects, entities, nil)
	got := depthCells(order, structures, objects, entities)
	want := []string{
		"S(4,7)", // the flat pass, first whatever its row
		"O(1,0)",
		"E(7,1)",
		"S(4,2)", "S(4,2)", "S(4,2)", // the tall structure's three strip entries
		"E(2,2)", // same row as the structure: drawn LAST, so in front of it
		"O(0,5)",
		"E(3,5)", // same row as the object: in front of it too
	}
	if len(got) != len(want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}

	t.Run("every drawable appears exactly once", func(t *testing.T) {
		seen := map[terrain.DepthRef]bool{}
		for _, ref := range order {
			if seen[ref] {
				t.Fatalf("%+v appears twice", ref)
			}
			seen[ref] = true
		}
		if len(seen) != len(structures)+len(objects)+len(entities) {
			t.Fatalf("%d refs, want one per drawable", len(seen))
		}
	})

	t.Run("the caller's entity slice is not reordered", func(t *testing.T) {
		want := []image.Point{{X: 3, Y: 5}, {X: 2, Y: 2}, {X: 7, Y: 1}}
		for i, w := range want {
			if entities[i].Cell != w {
				t.Fatalf("entity %d is now at %v, want the caller's own %v — the sort runs over an "+
					"index permutation and must not write through the caller's slice", i, entities[i].Cell, w)
			}
		}
	})
}

func TestDepthOrderKeepsIdOrderWithinARow(t *testing.T) {
	entities := []terrain.StaticPlacement{structObject(9, 4), structObject(1, 4), structObject(5, 4)}
	order := terrain.DepthOrder(nil, nil, nil, nil, entities, nil)
	got := depthCells(order, nil, nil, entities)
	want := []string{"E(9,4)", "E(1,4)", "E(5,4)"}
	for i := range want {
		if i >= len(got) || got[i] != want[i] {
			t.Fatalf("order = %v, want %v — columns are compared nowhere and the sort is stable", got, want)
		}
	}
}

func TestDepthOrderWithoutEntitiesIsThePlaneOrder(t *testing.T) {
	set := structSet(map[byte]*terrain.StructureClass{1: structClass(1, 1, 1)})
	structures, _, _ := terrain.StructurePlacements(structGrid(12, 12, structAt(1, 4, 1)), set, nil, 0)
	objects := []terrain.StaticPlacement{structObject(9, 2), structObject(0, 5)}
	order := terrain.PlaneOrder(structures, objects)

	for _, entities := range [][]terrain.StaticPlacement{nil, {}} {
		got := depthCells(terrain.DepthOrder(nil, order, structures, objects, entities, nil), structures, objects, entities)
		want := planeCells(order, structures, objects)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("with %d entities the merged order is %v, want the plane order %v", len(entities), got, want)
		}
	}
}
