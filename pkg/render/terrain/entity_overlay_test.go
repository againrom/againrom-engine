package terrain_test

// Tests for the simulated-entity marker geometry of pkg/render/terrain (work
// item 0020-sim-render-snapshot, T1).
//
// SEPARATE CONTEXT. The glyph is a FILLED SQUARE, not a cross, 10 pixels
// across at the native cell size; for cell (ax,ay) at cellpx pixels per
// cell:
//
//	round(n/32) = floor((n+16)/32)                  (n >= 0)
//	half        = floor(cellpx/2)
//	(cx,cy)     = (ax*cellpx+half, ay*cellpx+half)  the centre pixel
//	s           = max(1, round(10*cellpx/32))       the side
//	lo          = floor(s/2)                        centred-strip low offset
//	square      = [cx-lo, cx-lo+s) x [cy-lo, cy-lo+s)
//
// The centred-strip rule is the one a cross's THICKNESS uses, applied to
// both axes instead of one. At cellpx = 32 it reduces to s = 10, lo = 5,
// i.e. [cx-5, cx+5) x [cy-5, cy+5) about (32*ax+16, 32*ay+16).
//
// The square is then intersected with the map pixel rect [0, cols*cellpx) x
// [0, rows*cellpx), and an off-map cell (col<0 || row<0 || col>=cols ||
// row>=rows) or an invalid cellpx < 1 yields no geometry at all — the same
// two rules the three cross builders obey, which since this story are SHARED
// helpers rather than a copy per glyph.
//
// The scale-free helpers of overlay_test.go (specRound, specMapRect, sameRects,
// noPanicOverlay, sinkRects) are reused verbatim; overlay_test.go,
// unit_overlay_test.go and static_marker_test.go are the frozen characterization
// pins of the three cross glyphs and are not modified by this story.
//
// Every fixture is synthetic; nothing here reads a game install.

import (
	"image"
	"image/color"
	"math/bits"
	"testing"

	"againrom/pkg/render/terrain"
)

func specEntitySquareRaw(col, row, cellpx int) image.Rectangle {
	half := cellpx / 2
	cx, cy := col*cellpx+half, row*cellpx+half
	s := max(1, specRound(10*cellpx))
	lo := s / 2
	return image.Rect(cx-lo, cy-lo, cx-lo+s, cy-lo+s)
}

func specEntitySquare(col, row, cols, rows, cellpx int) []image.Rectangle {
	if cellpx < 1 || col < 0 || row < 0 || col >= cols || row >= rows {
		return nil
	}
	c := specEntitySquareRaw(col, row, cellpx).Intersect(specMapRect(cols, rows, cellpx))
	if c.Empty() {
		return nil
	}
	return []image.Rectangle{c}
}

func specCellRect(col, row, cellpx int) image.Rectangle {
	return image.Rect(col*cellpx, row*cellpx, (col+1)*cellpx, (row+1)*cellpx)
}

// TestEntityMarkerRectsNativeGeometry covers SC-6 at the native cell size:
// one rectangle per in-map cell, 10 output pixels on both axes, centred on
// the cell's own centre.
//
// The literals are hand-computed from the header formula, so the oracle helper
// itself is pinned to arithmetic written out longhand rather than to a second
// reading of the same code.
func TestEntityMarkerRectsNativeGeometry(t *testing.T) {
	const cols, rows = 8, 6
	const cellpx = terrain.CellSize // 32, the native scale

	if terrain.CellSize != 32 {
		t.Fatalf("CellSize = %d, want 32; every literal in this file is stated over 32-pixel cells", terrain.CellSize)
	}

	// Cell (2,3) at cellpx=32: cx = 2*32+16 = 80, cy = 3*32+16 = 112; s = 10,
	// lo = 5, so the square is [80-5,80+5) x [112-5,112+5) = [75,85) x [107,117).
	// Cell (0,0), the map corner: cx = cy = 16, so [11,21) x [11,21).
	// Cell (7,5), the last cell: cx = 7*32+16 = 240, cy = 5*32+16 = 176, so
	// [235,245) x [171,181) — inside [0,256) x [0,192), hence unclipped.
	hand := []struct {
		col, row int
		want     image.Rectangle
	}{
		{2, 3, image.Rect(75, 107, 85, 117)},
		{0, 0, image.Rect(11, 11, 21, 21)},
		{7, 5, image.Rect(235, 171, 245, 181)},
	}
	for _, h := range hand {
		if oracle := specEntitySquareRaw(h.col, h.row, cellpx); oracle != h.want {
			t.Fatalf("derivation drift: oracle gave %v for cell (%d,%d) at cellpx=32, hand-computed FR-6 says %v",
				oracle, h.col, h.row, h.want)
		}
		var got []image.Rectangle
		noPanicOverlay(t, "EntityMarkerRects", func() {
			got = terrain.EntityMarkerRects(h.col, h.row, cols, rows, cellpx)
		})
		if !sameRects(got, []image.Rectangle{h.want}) {
			t.Errorf("EntityMarkerRects(%d,%d,%d,%d,%d) = %v, want the single FR-6 square %v",
				h.col, h.row, cols, rows, cellpx, got, h.want)
		}
	}

	// Every in-map cell: exactly one rectangle, 10 x 10, centred on the cell
	// centre, and equal to the oracle.
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			got := terrain.EntityMarkerRects(col, row, cols, rows, cellpx)
			if len(got) != 1 {
				t.Fatalf("cell (%d,%d): got %d rectangles, want exactly 1 — the entity glyph is one filled square, not a cross (FR-6)",
					col, row, len(got))
			}
			r := got[0]
			if r.Dx() != 10 || r.Dy() != 10 {
				t.Errorf("cell (%d,%d): square is %dx%d output pixels, want 10x10 at the native cell size (FR-6, SC-6)",
					col, row, r.Dx(), r.Dy())
			}
			cx, cy := col*cellpx+cellpx/2, row*cellpx+cellpx/2
			if r.Min.X != cx-5 || r.Min.Y != cy-5 {
				t.Errorf("cell (%d,%d): square starts at %v, want (%d,%d) — centred on the cell centre (%d,%d) by the centred-strip rule",
					col, row, r.Min, cx-5, cy-5, cx, cy)
			}
			if !sameRects(got, specEntitySquare(col, row, cols, rows, cellpx)) {
				t.Errorf("cell (%d,%d): got %v, want the oracle's %v",
					col, row, got, specEntitySquare(col, row, cols, rows, cellpx))
			}
		}
	}
}

// TestEntityMarkerSquareStaysInsideItsOwnCell covers SC-6's scale clause: at
// every scale checked the square is no wider than its own cell, and the
// map-rect clip therefore leaves a corner cell untouched.
//
// Both halves matter. The containment is what makes the entity pass safe to
// draw UNDER the diagnostic crosses: a filled square that reached into a
// neighbouring cell could cover a cross belonging to a cell that holds no
// entity at all, which no draw order could repair.
func TestEntityMarkerSquareStaysInsideItsOwnCell(t *testing.T) {
	const cols, rows = 8, 6

	// 1 and 2 are the degenerate scales where max(1, ...) bites; 31/33 straddle
	// the native 32; 4096 is CellSize*128, the large scale the object tests use.
	scales := []int{1, 2, 3, 7, 16, 31, 32, 33, 48, 64, 96, 4096}
	corners := [][2]int{{0, 0}, {cols - 1, 0}, {0, rows - 1}, {cols - 1, rows - 1}}
	interior := [][2]int{{1, 1}, {3, 2}, {cols - 2, rows - 2}}

	for _, cellpx := range scales {
		for _, cell := range append(append([][2]int{}, corners...), interior...) {
			col, row := cell[0], cell[1]
			got := terrain.EntityMarkerRects(col, row, cols, rows, cellpx)
			if len(got) != 1 {
				t.Fatalf("cellpx=%d cell (%d,%d): got %d rectangles, want exactly 1", cellpx, col, row, len(got))
			}
			r := got[0]

			// No wider than its own cell, on both axes, and wholly inside it.
			if r.Dx() > cellpx || r.Dy() > cellpx {
				t.Errorf("cellpx=%d cell (%d,%d): square is %dx%d, wider than the %d-pixel cell it stands on (DD-4: s <= cellpx at every scale)",
					cellpx, col, row, r.Dx(), r.Dy(), cellpx)
			}
			cellRect := specCellRect(col, row, cellpx)
			if !r.In(cellRect) {
				t.Errorf("cellpx=%d cell (%d,%d): square %v leaves its own cell %v — a neighbouring cell's cross could be covered (DD-4, FR-7)",
					cellpx, col, row, r, cellRect)
			}

			// The clip is inert: the returned square is the UNCLIPPED one, even
			// on a corner cell, where a glyph reaching outside its cell would be
			// trimmed by the map rect.
			if raw := specEntitySquareRaw(col, row, cellpx); r != raw {
				t.Errorf("cellpx=%d cell (%d,%d): got %v, want the unclipped square %v — the map-rect clip must not trim an in-map entity square (DD-4)",
					cellpx, col, row, r, raw)
			}
		}
	}
}

// TestEntityMarkerRectsRejectsOffMapAndBadScale covers SC-6's rejection
// clause: no geometry for an off-map cell or for cellpx < 1, never a panic,
// and no allocation on the way out.
//
// The wrap case is the one that says the off-map test is load-bearing rather
// than redundant with the clip. A column whose product with cellpx overflows
// wraps back to a small pixel coordinate, so a builder that skipped the cell
// test and relied on the map-rect clip alone would return a square sitting on a
// perfectly plausible in-map cell.
func TestEntityMarkerRectsRejectsOffMapAndBadScale(t *testing.T) {
	const cols, rows = 8, 6

	// col*cellpx wraps to exactly 0 at cellpx = 32 = 2^5, on 32- and 64-bit
	// alike: 2^(bits-5) * 2^5 = 2^bits.
	const wrapCol = 1 << (bits.UintSize - 5)

	cases := []struct {
		name             string
		col, row, cellpx int
	}{
		{"col == cols, the first column off the right edge", cols, 1, 32},
		{"row == rows, the first row off the bottom edge", 3, rows, 32},
		{"negative col", -1, 1, 32},
		{"negative row", 3, -1, 32},
		{"far negative col", -(1 << 30), 1, 32},
		{"huge col", 0xFFFFFF, 2, 32},
		{"huge row", 1, 1 << 30, 32},
		{"col whose pixel product wraps back into the map", wrapCol, 1, 32},
		{"cellpx 0", 3, 2, 0},
		{"cellpx -1", 3, 2, -1},
		{"cellpx -32", 3, 2, -32},
		{"off-map and invalid scale together", cols, rows, 0},
	}

	for _, tc := range cases {
		var got []image.Rectangle
		noPanicOverlay(t, "EntityMarkerRects "+tc.name, func() {
			got = terrain.EntityMarkerRects(tc.col, tc.row, cols, rows, tc.cellpx)
		})
		if got != nil {
			t.Errorf("%s: EntityMarkerRects(%d,%d,%d,%d,%d) = %v, want nil (FR-6: no geometry at all)",
				tc.name, tc.col, tc.row, cols, rows, tc.cellpx, got)
		}
		if n := testing.AllocsPerRun(100, func() {
			sinkRects = terrain.EntityMarkerRects(tc.col, tc.row, cols, rows, tc.cellpx)
		}); n != 0 {
			t.Errorf("%s: a rejected cell returns before any geometry is built, so it must not allocate; got %.0f allocation(s)",
				tc.name, n)
		}
	}
}

func TestMarkerFamilyRejectionIsOneSharedRule(t *testing.T) {
	const cols, rows = 8, 6

	builders := []struct {
		name  string
		rects func(col, row, cols, rows, cellpx int) []image.Rectangle
	}{
		{"ObjectMarkerRects", terrain.ObjectMarkerRects},
		{"UnitMarkerRects", terrain.UnitMarkerRects},
		{"StaticMarkerRects", terrain.StaticMarkerRects},
		{"EntityMarkerRects", terrain.EntityMarkerRects},
		{"SelectionMarkerRects", terrain.SelectionMarkerRects},
	}
	boundaries := []struct {
		name             string
		col, row, cellpx int
	}{
		{"col == cols", cols, 2, 32},
		{"row == rows", 2, rows, 32},
		{"col == -1", -1, 2, 32},
		{"row == -1", 2, -1, 32},
		{"cellpx == 0", 2, 2, 0},
	}

	for _, b := range builders {
		for _, tc := range boundaries {
			if got := b.rects(tc.col, tc.row, cols, rows, tc.cellpx); got != nil {
				t.Errorf("%s at %s: got %v, want nil", b.name, tc.name, got)
			}
			if n := testing.AllocsPerRun(100, func() {
				sinkRects = b.rects(tc.col, tc.row, cols, rows, tc.cellpx)
			}); n != 0 {
				t.Errorf("%s at %s: the shared rejection must return before any geometry is built; got %.0f allocation(s)",
					b.name, tc.name, n)
			}
		}
	}
}

// TestEntityMarkerColorDiffersFromEveryDiagnostic covers SC-6's colour
// clause: the entity colour equals none of the three diagnostic marker
// colours.
//
// It is not a style rule. The entity square is drawn UNDER all three
// crosses, so a coincident pair reads as a cross standing on a block of
// another colour; sharing a colour with the cross on top of it would make
// the pair indistinguishable from an entity alone, which is the comparison
// the whole arrangement exists to make visible.
func TestEntityMarkerColorDiffersFromEveryDiagnostic(t *testing.T) {
	others := []struct {
		name string
		c    color.RGBA
	}{
		{"MarkerColor (placed objects, yellow)", terrain.MarkerColor},
		{"UnitMarkerColor (placed units, cyan)", terrain.UnitMarkerColor},
		{"StaticMarkerColor (static objects, red)", terrain.StaticMarkerColor},
	}
	for _, o := range others {
		if terrain.EntityMarkerColor == o.c {
			t.Errorf("EntityMarkerColor = %v, the same colour as %s; FR-6 requires it to differ from all three",
				terrain.EntityMarkerColor, o.name)
		}
	}
	if terrain.EntityMarkerColor.A != 0xff {
		t.Errorf("EntityMarkerColor alpha = %#x, want 0xff — every marker colour in this family is opaque",
			terrain.EntityMarkerColor.A)
	}
}
