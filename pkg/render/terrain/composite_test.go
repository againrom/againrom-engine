package terrain_test

import (
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
)

// Fill bases for the compositor fixture, one per strip, so a pixel identifies
// both which strip and which sub-cell it came from.
const (
	landBase  = 1
	waterBase = 100
	dirtBase  = 200
)

// compositeSource is a minimal tileset: one land strip (tile1-00, 14 cells),
// one water strip (tile3-00, 8 cells) and dirt (4 cells). Every other slot is
// absent, so a word naming one exercises the placeholder path.
func compositeSource() mapSource {
	return mapSource{
		terrain.TilePath(1, 0): solidStrip(14, func(k int) byte { return fillOf(landBase, k) }),
		terrain.TilePath(3, 0): solidStrip(8, func(k int) byte { return fillOf(waterBase, k) }),
		terrain.DirtPath:       solidStrip(4, func(k int) byte { return fillOf(dirtBase, k) }),
	}
}

// overlayExpect mirrors the decoded impassable composite (TERR-DIRT-017): a
// non-zero dirt index replaces the terrain pixel outright, a zero index leaves
// the terrain showing. No arithmetic.
func overlayExpect(tile color.RGBA, dirtIndex byte) color.RGBA {
	if dirtIndex == 0 {
		return tile
	}
	return rampColor(dirtIndex)
}

// cellUniform asserts every pixel of the scaled cell at (col,row) is want,
// which also pins the cell's origin at (col*32, row*32)*scale.
func cellUniform(t *testing.T, r *terrain.Render, col, row, scale int, want color.RGBA, what string) {
	t.Helper()
	size := terrain.CellSize * scale
	originX, originY := col*size, row*size
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if got := r.Image.RGBAAt(originX+x, originY+y); got != want {
				t.Fatalf("%s: cell (%d,%d) pixel (%d,%d) = %v, want %v", what, col, row, x, y, got, want)
			}
		}
	}
}

// TestCompositePlacesSubCells - AC-6: each cell shows its resolved sub-cell at
// (col*32, row*32), an impassable non-water cell is composited with dirt sub-cell
// (col+row*5)&3, and a cell whose slot is absent becomes a reported placeholder.
func TestCompositePlacesSubCells(t *testing.T) {
	ts := terrain.LoadTileset(compositeSource())

	// (0,0) land tile1-00 sub 3; (1,0) water tile3-00 sub 5;
	// (0,1) impassable land tile1-00 sub 2; (1,1) tile1-04, which is absent.
	grid := terrain.Grid{Width: 2, Height: 2, Tiles: []uint16{
		tileWord(0, 0, 3),
		tileWord(8, 0, 5),
		tileWord(0, 0, 2) | terrain.ImpassableBit,
		tileWord(1, 0, 0),
	}}

	r, err := terrain.Composite(ts, grid, 1)
	if err != nil {
		t.Fatalf("composite: %v", err)
	}
	if b := r.Image.Bounds(); b.Dx() != 2*terrain.CellSize || b.Dy() != 2*terrain.CellSize {
		t.Fatalf("image bounds = %v, want %dx%d", b, 2*terrain.CellSize, 2*terrain.CellSize)
	}
	if r.Placeholders != 1 {
		t.Fatalf("Placeholders = %d, want 1", r.Placeholders)
	}

	cellUniform(t, r, 0, 0, 1, rampColor(fillOf(landBase, 3)), "land")
	cellUniform(t, r, 1, 0, 1, rampColor(fillOf(waterBase, 5)), "water")
	cellUniform(t, r, 1, 1, 1, terrain.PlaceholderColor, "absent slot")

	// The impassable cell takes dirt sub-cell (0 + 1*5) & 3 == 1. Every dirt
	// sub-cell in this fixture is a non-zero solid index, so the dirt replaces
	// the terrain outright; picking any other dirt cell must differ, which is
	// what makes the selector observable.
	tile := rampColor(fillOf(landBase, 2))
	wantDirt := terrain.DirtSubCell(0, 1)
	if wantDirt != 1 {
		t.Fatalf("fixture assumption: DirtSubCell(0,1) = %d, want 1", wantDirt)
	}
	want := overlayExpect(tile, fillOf(dirtBase, wantDirt))
	if want == tile {
		t.Fatal("fixture assumption: the dirt sub-cell must be opaque here")
	}
	cellUniform(t, r, 0, 1, 1, want, "impassable + dirt")

	for k := 0; k < 4; k++ {
		if k == wantDirt {
			continue
		}
		if other := overlayExpect(tile, fillOf(dirtBase, k)); other == want {
			t.Fatalf("fixture is not discriminating: dirt cell %d overlays identically to %d", k, wantDirt)
		}
	}
}

// TestCompositeDirtIsTransparentKeyed - TERR-DIRT-017: the dirt overlay is a
// transparent-keyed replace, not a blend. Where the dirt sub-cell holds palette
// index 0 the terrain shows through untouched; where it holds any other index
// the dirt colour replaces the terrain outright, with no arithmetic mixing the
// two.
func TestCompositeDirtIsTransparentKeyed(t *testing.T) {
	const landIdx, dirtIdx = 40, 210

	// A dirt strip whose sub-cell 1 is transparent (index 0) on its left half
	// and opaque (index dirtIdx) on its right half.
	src := mapSource{
		terrain.TilePath(1, 0): solidStrip(14, func(k int) byte { return landIdx }),
		terrain.DirtPath: halfStrip(4, func(k int, x int) byte {
			if x < terrain.CellSize/2 {
				return 0 // the transparent key
			}
			return dirtIdx
		}),
	}
	ts := terrain.LoadTileset(src)

	// One impassable land cell at (0,1), which selects dirt sub-cell 1.
	g := terrain.Grid{Width: 1, Height: 2, Tiles: []uint16{
		tileWord(0, 0, 0),
		tileWord(0, 0, 0) | terrain.ImpassableBit,
	}}
	r, err := terrain.Composite(ts, g, 1)
	if err != nil {
		t.Fatalf("Composite: err = %v", err)
	}

	land := rampColor(landIdx)
	dirt := rampColor(dirtIdx)
	if land == dirt {
		t.Fatal("fixture assumption: the land and dirt colours must differ")
	}

	for y := 0; y < terrain.CellSize; y++ {
		for x := 0; x < terrain.CellSize; x++ {
			got := r.Image.RGBAAt(x, terrain.CellSize+y)
			if x < terrain.CellSize/2 {
				// Index 0: the terrain must survive completely untouched. A
				// blend would have shifted it toward the dirt colour.
				if got != land {
					t.Fatalf("transparent half (%d,%d) = %v, want the terrain %v untouched", x, y, got, land)
				}
			} else if got != dirt {
				// Non-zero: the dirt replaces the terrain outright. A blend
				// would land between the two.
				t.Fatalf("opaque half (%d,%d) = %v, want the dirt %v outright", x, y, got, dirt)
			}
		}
	}

	// The passable cell above is never composited, whatever the dirt holds.
	cellUniform(t, r, 0, 0, 1, land, "passable land")
}

func TestCompositeScales(t *testing.T) {
	ts := terrain.LoadTileset(compositeSource())
	grid := terrain.Grid{Width: 2, Height: 1, Tiles: []uint16{
		tileWord(0, 0, 0),
		tileWord(0, 0, 7),
	}}

	for _, scale := range []int{1, 2, 3} {
		r, err := terrain.Composite(ts, grid, scale)
		if err != nil {
			t.Fatalf("scale %d: %v", scale, err)
		}
		wantW := 2 * terrain.CellSize * scale
		wantH := 1 * terrain.CellSize * scale
		if b := r.Image.Bounds(); b.Dx() != wantW || b.Dy() != wantH {
			t.Fatalf("scale %d: bounds = %v, want %dx%d", scale, b, wantW, wantH)
		}
		cellUniform(t, r, 0, 0, scale, rampColor(fillOf(landBase, 0)), "scaled cell 0")
		cellUniform(t, r, 1, 0, scale, rampColor(fillOf(landBase, 7)), "scaled cell 1")
	}
}

func TestCompositeWaterIgnoresImpassable(t *testing.T) {
	ts := terrain.LoadTileset(compositeSource())
	grid := terrain.Grid{Width: 1, Height: 1, Tiles: []uint16{
		tileWord(8, 0, 4) | terrain.ImpassableBit,
	}}

	r, err := terrain.Composite(ts, grid, 1)
	if err != nil {
		t.Fatalf("composite: %v", err)
	}
	if r.Placeholders != 0 {
		t.Fatalf("Placeholders = %d, want 0", r.Placeholders)
	}
	cellUniform(t, r, 0, 0, 1, rampColor(fillOf(waterBase, 4)), "impassable water")
}

func TestCompositeWithoutDirt(t *testing.T) {
	src := compositeSource()
	delete(src, terrain.DirtPath)
	ts := terrain.LoadTileset(src)
	if ts.Dirt != nil {
		t.Fatal("fixture: dirt should be absent")
	}

	grid := terrain.Grid{Width: 1, Height: 1, Tiles: []uint16{tileWord(0, 0, 6) | terrain.ImpassableBit}}
	r, err := terrain.Composite(ts, grid, 1)
	if err != nil {
		t.Fatalf("composite: %v", err)
	}
	if r.Placeholders != 0 {
		t.Fatalf("Placeholders = %d, want 0 (the tile itself is present)", r.Placeholders)
	}
	cellUniform(t, r, 0, 0, 1, rampColor(fillOf(landBase, 6)), "impassable without dirt")
}

func TestCompositeRejectsBadArguments(t *testing.T) {
	ts := terrain.LoadTileset(compositeSource())
	ok := terrain.Grid{Width: 2, Height: 2, Tiles: make([]uint16, 4)}

	for _, tc := range []struct {
		name  string
		ts    *terrain.Tileset
		grid  terrain.Grid
		scale int
	}{
		{"nil tileset", nil, ok, 1},
		{"zero width", ts, terrain.Grid{Width: 0, Height: 2, Tiles: make([]uint16, 0)}, 1},
		{"negative height", ts, terrain.Grid{Width: 2, Height: -1, Tiles: make([]uint16, 4)}, 1},
		{"short grid", ts, terrain.Grid{Width: 2, Height: 2, Tiles: make([]uint16, 3)}, 1},
		{"long grid", ts, terrain.Grid{Width: 2, Height: 2, Tiles: make([]uint16, 5)}, 1},
		{"zero scale", ts, ok, 0},
		{"negative scale", ts, ok, -2},
		{"over the pixel cap", ts, terrain.Grid{Width: 1 << 14, Height: 1 << 14, Tiles: nil}, 1},
	} {
		r, err := terrain.Composite(tc.ts, tc.grid, tc.scale)
		if err == nil {
			t.Errorf("%s: expected an error, got a render", tc.name)
		}
		if r != nil {
			t.Errorf("%s: expected a nil render on error", tc.name)
		}
	}
}

func TestCompositeNeverPanics(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Composite panicked: %v", r)
		}
	}()

	empty := terrain.LoadTileset(mapSource{})
	full := terrain.LoadTileset(compositeSource())

	// Every cell of an empty tileset is a placeholder.
	grid := terrain.Grid{Width: 3, Height: 2, Tiles: make([]uint16, 6)}
	r, err := terrain.Composite(empty, grid, 2)
	if err != nil {
		t.Fatalf("empty tileset: %v", err)
	}
	if r.Placeholders != 6 {
		t.Fatalf("empty tileset: Placeholders = %d, want 6", r.Placeholders)
	}
	for row := 0; row < 2; row++ {
		for col := 0; col < 3; col++ {
			cellUniform(t, r, col, row, 2, terrain.PlaceholderColor, "empty tileset")
		}
	}

	// Off-corpus words: the maximum group (0x1fff >> 6 = 127) names slot 511,
	// far outside the 128-slot array, and sub-cells past every strip's height.
	words := []uint16{0xffff, 0x7fff, 0x1fff, 0xffc0, 0x3fff, 0x200f}
	off := terrain.Grid{Width: len(words), Height: 1, Tiles: words}
	for _, scale := range []int{1, 2, 4} {
		r, err := terrain.Composite(full, off, scale)
		if err != nil {
			t.Fatalf("off-range words at scale %d: %v", scale, err)
		}
		if r.Placeholders != len(words) {
			t.Fatalf("off-range words: Placeholders = %d, want %d", r.Placeholders, len(words))
		}
	}

	// A sub-cell just past a present strip's height is also a placeholder:
	// tile3-00 has 8 cells, so sub 8 must fall back rather than read on.
	shortSub := terrain.Grid{Width: 1, Height: 1, Tiles: []uint16{tileWord(8, 0, 8)}}
	r, err = terrain.Composite(full, shortSub, 1)
	if err != nil {
		t.Fatalf("short sub-cell: %v", err)
	}
	if r.Placeholders != 1 {
		t.Fatalf("short sub-cell: Placeholders = %d, want 1", r.Placeholders)
	}
	cellUniform(t, r, 0, 0, 1, terrain.PlaceholderColor, "sub-cell past the strip")
}
