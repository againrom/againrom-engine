package terrain_test

import (
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/formats/bmp"
	"againrom/pkg/render/terrain"
)

// mapSource is an in-memory EntrySource standing in for an opened archive.
// Lookup is case-insensitive with '\' folded to '/', matching how the .res
// reader normalizes paths.
type mapSource map[string][]byte

func (m mapSource) ReadFile(name string) ([]byte, error) {
	key := strings.ToLower(strings.ReplaceAll(name, "\\", "/"))
	b, ok := m[key]
	if !ok {
		return nil, fmt.Errorf("no entry %q", name)
	}
	return b, nil
}

// fillOf returns the palette index solidStrip painted into sub-cell k, chosen so
// each sub-cell of each strip is distinguishable.
func fillOf(base, k int) byte { return byte(base + k) }

// shippedSource builds the entry set a real install carries: tile1/2/3 with 16
// variants each (14 / 14 / 8 sub-cells), tile4-00..03 (14) and dirt.bmp (4).
// Every sub-cell gets a distinct palette index so a mis-slice is visible.
func shippedSource() mapSource {
	src := mapSource{}
	add := func(path string, cells, base int) {
		src[path] = solidStrip(cells, func(k int) byte { return fillOf(base, k) })
	}
	base := 1
	for g := 1; g <= 4; g++ {
		variants, cells := 16, 14
		if g == 3 {
			cells = 8 // water strips are 32x256
		}
		if g == 4 {
			variants = 4 // only tile4-00..03 ship
		}
		for v := 0; v < variants; v++ {
			add(terrain.TilePath(g, v), cells, base)
			base += 16
		}
	}
	add(terrain.DirtPath, 4, 200)
	return src
}

// --- tests ---

func TestSliceStripSubCellCounts(t *testing.T) {
	for _, tc := range []struct{ cells int }{{14}, {8}, {4}} {
		name := fmt.Sprintf("32x%d", tc.cells*terrain.CellSize)
		t.Run(name, func(t *testing.T) {
			img, err := bmp.DecodePaletted(solidStrip(tc.cells, func(k int) byte { return fillOf(10, k) }))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			strip, err := terrain.SliceStrip(img)
			if err != nil {
				t.Fatalf("slice: %v", err)
			}
			if got := len(strip.SubCells); got != tc.cells {
				t.Fatalf("sub-cells = %d, want %d", got, tc.cells)
			}
			for k := 0; k < tc.cells; k++ {
				cell := strip.SubCell(k)
				if cell == nil {
					t.Fatalf("sub-cell %d is nil", k)
				}
				if b := cell.Bounds(); b.Dx() != terrain.CellSize || b.Dy() != terrain.CellSize {
					t.Fatalf("sub-cell %d bounds = %v, want %dx%d", k, b, terrain.CellSize, terrain.CellSize)
				}
				want := rampColor(fillOf(10, k))
				for _, p := range [][2]int{{0, 0}, {31, 0}, {0, 31}, {31, 31}, {17, 9}} {
					if got := pixColor(cell, p[0], p[1]); got != want {
						t.Fatalf("sub-cell %d pixel (%d,%d) = %v, want %v — wrong block or wrong row order",
							k, p[0], p[1], got, want)
					}
				}
			}
			// Out-of-range indices are nil, not a panic.
			if strip.SubCell(-1) != nil || strip.SubCell(tc.cells) != nil {
				t.Fatal("out-of-range SubCell should be nil")
			}
		})
	}
}

func TestSliceStripRejectsBadGeometry(t *testing.T) {
	if _, err := terrain.SliceStrip(nil); err == nil {
		t.Fatal("nil image: expected an error")
	}
	for _, tc := range []struct{ w, h int }{
		{16, 128}, // too narrow
		{64, 128}, // too wide
		{32, 20},  // not a whole cell
		{32, 100}, // 3 cells + 4 px
	} {
		img, err := bmp.DecodePaletted(buildBMP8(tc.w, tc.h, rampPalette(), make([]byte, tc.w*tc.h)))
		if err != nil {
			t.Fatalf("%dx%d decode: %v", tc.w, tc.h, err)
		}
		if strip, err := terrain.SliceStrip(img); err == nil {
			t.Fatalf("%dx%d: expected an error, got %d sub-cells", tc.w, tc.h, len(strip.SubCells))
		}
	}
}

func TestTilePathAndSlotIndex(t *testing.T) {
	for _, tc := range []struct {
		g, v int
		path string
		slot int
	}{
		{1, 0, "graphics/terrain/tile1-00.bmp", 0},
		{1, 9, "graphics/terrain/tile1-09.bmp", 9},
		{1, 15, "graphics/terrain/tile1-15.bmp", 15},
		{2, 0, "graphics/terrain/tile2-00.bmp", 16},
		{3, 0, "graphics/terrain/tile3-00.bmp", 32},
		{4, 3, "graphics/terrain/tile4-03.bmp", 51},
		{8, 15, "graphics/terrain/tile8-15.bmp", 127},
	} {
		if got := terrain.TilePath(tc.g, tc.v); got != tc.path {
			t.Errorf("TilePath(%d,%d) = %q, want %q", tc.g, tc.v, got, tc.path)
		}
		if got := terrain.SlotIndex(tc.g, tc.v); got != tc.slot {
			t.Errorf("SlotIndex(%d,%d) = %d, want %d", tc.g, tc.v, got, tc.slot)
		}
	}
}

func TestLoadTilesetPopulatesShippedSlots(t *testing.T) {
	ts := terrain.LoadTileset(shippedSource())

	if ts.Loaded != 52 {
		t.Fatalf("Loaded = %d, want 52", ts.Loaded)
	}
	for _, tc := range []struct{ g, v, cells int }{
		{1, 0, 14}, {1, 15, 14}, {2, 7, 14}, {3, 0, 8}, {3, 15, 8}, {4, 0, 14}, {4, 3, 14},
	} {
		strip := ts.Slot(terrain.SlotIndex(tc.g, tc.v))
		if strip == nil {
			t.Fatalf("tile%d-%02d: slot is nil", tc.g, tc.v)
		}
		if got := len(strip.SubCells); got != tc.cells {
			t.Fatalf("tile%d-%02d: %d sub-cells, want %d", tc.g, tc.v, got, tc.cells)
		}
	}
	if ts.Dirt == nil || len(ts.Dirt.SubCells) != 4 {
		t.Fatalf("dirt: want 4 sub-cells, got %v", ts.Dirt)
	}

	// Absent by construction on a real install.
	for _, tc := range []struct{ g, v int }{{4, 4}, {4, 15}, {5, 0}, {8, 15}} {
		if ts.Slot(terrain.SlotIndex(tc.g, tc.v)) != nil {
			t.Errorf("tile%d-%02d: slot should be nil", tc.g, tc.v)
		}
	}
	// 128 attempted slots + dirt, minus the 52 that loaded and the dirt that did.
	if got := len(ts.Missing); got != 128-52 {
		t.Fatalf("Missing = %d entries, want %d", got, 128-52)
	}
	for _, want := range []string{terrain.TilePath(4, 4), terrain.TilePath(5, 0)} {
		if !containsString(ts.Missing, want) {
			t.Errorf("Missing does not record %q", want)
		}
	}
}

// TestLoadTilesetRecordsAbsent - AC-7: an empty archive, non-BMP bytes and a
// short/odd strip each leave a recorded null slot rather than failing or
// panicking, and out-of-range slot lookups stay nil.
func TestLoadTilesetRecordsAbsent(t *testing.T) {
	empty := terrain.LoadTileset(mapSource{})
	if empty.Loaded != 0 || empty.Dirt != nil {
		t.Fatalf("empty archive: Loaded = %d, Dirt = %v, want 0/nil", empty.Loaded, empty.Dirt)
	}
	if got := len(empty.Missing); got != terrain.SlotCount+1 {
		t.Fatalf("empty archive: Missing = %d, want %d (128 tiles + dirt)", got, terrain.SlotCount+1)
	}
	for i := 0; i < terrain.SlotCount; i++ {
		if empty.Slot(i) != nil {
			t.Fatalf("empty archive: slot %d is not nil", i)
		}
	}

	// Present but undecodable: junk bytes, a truncated BMP, and a BMP whose
	// geometry is not a whole number of 32x32 cells.
	bad := mapSource{
		terrain.TilePath(1, 0): []byte("not a bitmap"),
		terrain.TilePath(1, 1): solidStrip(14, func(int) byte { return 1 })[:100],
		terrain.TilePath(1, 2): buildBMP8(32, 20, rampPalette(), make([]byte, 32*20)),
		terrain.TilePath(1, 3): buildBMP8(16, 128, rampPalette(), make([]byte, 16*128)),
		terrain.DirtPath:       []byte{},
	}
	ts := terrain.LoadTileset(bad)
	if ts.Loaded != 0 {
		t.Fatalf("undecodable entries: Loaded = %d, want 0", ts.Loaded)
	}
	if ts.Dirt != nil {
		t.Fatal("undecodable dirt should leave a nil slot")
	}
	for v := 0; v < 4; v++ {
		if ts.Slot(terrain.SlotIndex(1, v)) != nil {
			t.Errorf("tile1-%02d: undecodable entry should leave a nil slot", v)
		}
		if !containsString(ts.Missing, terrain.TilePath(1, v)) {
			t.Errorf("tile1-%02d: not recorded in Missing", v)
		}
	}

	// A nil source is survivable, and slot lookups are bounds-checked.
	nilSrc := terrain.LoadTileset(nil)
	if nilSrc.Loaded != 0 || len(nilSrc.Missing) == 0 {
		t.Fatal("nil source should load nothing and record the condition")
	}
	if nilSrc.Slot(-1) != nil || nilSrc.Slot(terrain.SlotCount) != nil || nilSrc.Slot(1<<20) != nil {
		t.Fatal("out-of-range Slot should be nil")
	}
	var nilSet *terrain.Tileset
	if nilSet.Slot(0) != nil {
		t.Fatal("Slot on a nil Tileset should be nil")
	}
	var nilStrip *terrain.Strip
	if nilStrip.SubCell(0) != nil {
		t.Fatal("SubCell on a nil Strip should be nil")
	}
}

func containsString(hay []string, needle string) bool {
	for _, s := range hay {
		if s == needle {
			return true
		}
	}
	return false
}
