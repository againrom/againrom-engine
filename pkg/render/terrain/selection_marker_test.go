package terrain_test

// Tests for the selected unit's highlight geometry of pkg/render/terrain (work
// item 0028-app-unit-command, T3).
//
// SEPARATE CONTEXT. The glyph is the CELL'S OWN FOOTPRINT, drawn HOLLOW —
// a border, not a cross and not a filled square. For cell (ax,ay) at cellpx
// pixels per cell:
//
//	round(n/32) = floor((n+16)/32)                  (n >= 0)
//	(x0,y0)     = (ax*cellpx, ay*cellpx)            the footprint's corner
//	(x1,y1)     = (x0+cellpx, y0+cellpx)
//	t           = max(1, round(2*cellpx/32))        the rim's thickness
//	top         = [x0,   x1) x [y0,   y0+t)         full width
//	bottom      = [x0,   x1) x [y1-t, y1)           full width
//	left        = [x0,   x0+t) x [y0+t, y1-t)       between the two
//	right       = [x1-t, x1)   x [y0+t, y1-t)       between the two
//
// in that order. At cellpx = 32 it reduces to t = 2, so cell (0,0) yields
// [0,32) x [0,2), [0,32) x [30,32), [0,2) x [2,30) and [30,32) x [2,30).
//
// The strips are then intersected with the map pixel rect [0, cols*cellpx) x
// [0, rows*cellpx), an empty one is dropped, and an off-map cell (col<0 ||
// row<0 || col>=cols || row>=rows) or an invalid cellpx < 1 yields no
// geometry at all — the same two shared rules the four glyph builders
// beside it obey.
//
// WHY HOLLOW IS THE WHOLE POINT. This is the one glyph drawn LAST, over the
// entity's own sprite. TestSelectionRimCoversNoOtherGlyph measures that
// rather than asserting it.
//
// The scale-free helpers of overlay_test.go (specRound, specMapRect, sameRects,
// noPanicOverlay, sinkRects) are reused verbatim; overlay_test.go,
// unit_overlay_test.go, static_marker_test.go and entity_overlay_test.go pin
// the four shipped glyphs and none of their geometry is modified by this story.
//
// Every fixture is synthetic; nothing here reads a game install.

import (
	"image"
	"image/color"
	"math/bits"
	"testing"

	"againrom/pkg/render/terrain"
)

func specRimThickness(cellpx int) int { return max(1, specRound(2*cellpx)) }

func specSelectionRimRaw(col, row, cellpx int) []image.Rectangle {
	x0, y0 := col*cellpx, row*cellpx
	x1, y1 := x0+cellpx, y0+cellpx
	t := specRimThickness(cellpx)

	topHi := min(y0+t, y1)
	botLo := max(y1-t, topHi)

	out := make([]image.Rectangle, 0, 4)
	for _, s := range []image.Rectangle{
		image.Rect(x0, y0, x1, topHi),
		image.Rect(x0, botLo, x1, y1),
		image.Rect(x0, topHi, x0+t, botLo),
		image.Rect(x1-t, topHi, x1, botLo),
	} {
		if !s.Empty() {
			out = append(out, s)
		}
	}
	return out
}

func specSelectionRim(col, row, cols, rows, cellpx int) []image.Rectangle {
	if cellpx < 1 || col < 0 || row < 0 || col >= cols || row >= rows {
		return nil
	}
	clip := specMapRect(cols, rows, cellpx)
	out := make([]image.Rectangle, 0, 4)
	for _, s := range specSelectionRimRaw(col, row, cellpx) {
		if c := s.Intersect(clip); !c.Empty() {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// selectionPixelSet is the set of pixels a rectangle list covers. It is how
// hollowness, disjointness and non-covering are measured below: a claim about
// which pixels a glyph owns is stated over pixels, not over rectangles, so no
// rearrangement of the same strips can satisfy it by accident.
func selectionPixelSet(rects []image.Rectangle) map[image.Point]bool {
	m := map[image.Point]bool{}
	for _, r := range rects {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				m[image.Pt(x, y)] = true
			}
		}
	}
	return m
}

// TestSelectionMarkerRectsNativeGeometry covers SC-4 at the native cell
// size: four rectangles per in-map cell, the footprint's border two pixels
// thick, in the order top, bottom, left, right.
//
// The literals are hand-computed from the header formula, so the oracle helper
// itself is pinned to arithmetic written out longhand rather than to a second
// reading of the same code.
func TestSelectionMarkerRectsNativeGeometry(t *testing.T) {
	const cols, rows = 8, 6
	const cellpx = terrain.CellSize // 32, the native scale

	if terrain.CellSize != 32 {
		t.Fatalf("CellSize = %d, want 32; every literal in this file is stated over 32-pixel cells", terrain.CellSize)
	}
	if got := specRimThickness(cellpx); got != 2 {
		t.Fatalf("the transcribed thickness at cellpx=32 is %d, want 2 — the test's own FR-2 reading is wrong", got)
	}

	// Cell (0,0), the map corner: footprint [0,32) x [0,32).
	// Cell (2,3): footprint [64,96) x [96,128).
	// Cell (7,5), the last cell: footprint [224,256) x [160,192) — flush against
	// the map rect [0,256) x [0,192) on both far edges, hence unclipped.
	hand := []struct {
		col, row int
		want     []image.Rectangle
	}{
		{0, 0, []image.Rectangle{
			image.Rect(0, 0, 32, 2),   // top
			image.Rect(0, 30, 32, 32), // bottom
			image.Rect(0, 2, 2, 30),   // left
			image.Rect(30, 2, 32, 30), // right
		}},
		{2, 3, []image.Rectangle{
			image.Rect(64, 96, 96, 98),
			image.Rect(64, 126, 96, 128),
			image.Rect(64, 98, 66, 126),
			image.Rect(94, 98, 96, 126),
		}},
		{7, 5, []image.Rectangle{
			image.Rect(224, 160, 256, 162),
			image.Rect(224, 190, 256, 192),
			image.Rect(224, 162, 226, 190),
			image.Rect(254, 162, 256, 190),
		}},
	}
	for _, h := range hand {
		if oracle := specSelectionRimRaw(h.col, h.row, cellpx); !sameRects(oracle, h.want) {
			t.Fatalf("derivation drift: oracle gave %v for cell (%d,%d) at cellpx=32, hand-computed FR-2 says %v",
				oracle, h.col, h.row, h.want)
		}
		var got []image.Rectangle
		noPanicOverlay(t, "SelectionMarkerRects", func() {
			got = terrain.SelectionMarkerRects(h.col, h.row, cols, rows, cellpx)
		})
		if !sameRects(got, h.want) {
			t.Errorf("SelectionMarkerRects(%d,%d,%d,%d,%d) = %v, want the FR-2 rim %v",
				h.col, h.row, cols, rows, cellpx, got, h.want)
		}
	}

	// Every in-map cell: exactly four strips, two of full cell width, two of the
	// rim's thickness, all equal to the oracle.
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			got := terrain.SelectionMarkerRects(col, row, cols, rows, cellpx)
			if len(got) != 4 {
				t.Fatalf("cell (%d,%d): got %d rectangles, want exactly 4 — the highlight is a hollow rim, "+
					"not a cross and not a filled square (FR-2, DD-8)", col, row, len(got))
			}
			if got[0].Dx() != cellpx || got[0].Dy() != 2 || got[1].Dx() != cellpx || got[1].Dy() != 2 {
				t.Errorf("cell (%d,%d): the two full-width strips are %dx%d and %dx%d, want %dx2 both",
					col, row, got[0].Dx(), got[0].Dy(), got[1].Dx(), got[1].Dy(), cellpx)
			}
			if got[2].Dx() != 2 || got[2].Dy() != cellpx-4 || got[3].Dx() != 2 || got[3].Dy() != cellpx-4 {
				t.Errorf("cell (%d,%d): the two side strips are %dx%d and %dx%d, want 2x%d both — they run "+
					"BETWEEN the full-width strips, not beside them",
					col, row, got[2].Dx(), got[2].Dy(), got[3].Dx(), got[3].Dy(), cellpx-4)
			}
			if !sameRects(got, specSelectionRim(col, row, cols, rows, cellpx)) {
				t.Errorf("cell (%d,%d): got %v, want the oracle's %v",
					col, row, got, specSelectionRim(col, row, cols, rows, cellpx))
			}
		}
	}
}

// TestSelectionRimIsHollowAndStaysInsideItsOwnCell covers SC-4's shape
// clause at every scale checked: the strips are pairwise disjoint, they
// cover exactly the footprint's border and leave its interior untouched,
// they never leave the cell, and the map-rect clip is therefore inert on an
// in-map cell.
//
// All four halves matter. Containment is what makes drawing it last safe for
// the NEIGHBOURS as well: a rim reaching past its own footprint could cover
// a glyph belonging to a cell that holds no selected unit at all, which no
// draw order could repair. Disjointness is what "four strips" means rather
// than "four rectangles that happen to union to a border".
//
// The two degenerate scales are asserted rather than skipped: at cellpx 2 the
// two full-width strips tile the footprint and the sides are empty; at cellpx 1
// the single strip IS the footprint. Neither is reachable from either renderer
// — the window builds every glyph at CellSize and only then scales by the
// camera's zoom, and the raster tool's -scale is at least 1 — but a builder that
// handed back an inverted rectangle there would be handing back one whose
// corners image.Rect had silently swapped, which is a different bug from a small
// glyph.
func TestSelectionRimIsHollowAndStaysInsideItsOwnCell(t *testing.T) {
	const cols, rows = 8, 6

	// 1 and 2 are the degenerate scales where the clamps bite; 3 is the first
	// scale carrying all four strips; 31/33 straddle the native 32; 4096 is
	// CellSize*128, the large scale the object tests use.
	scales := []int{1, 2, 3, 7, 16, 31, 32, 33, 48, 64, 96, 4096}
	corners := [][2]int{{0, 0}, {cols - 1, 0}, {0, rows - 1}, {cols - 1, rows - 1}}
	interior := [][2]int{{1, 1}, {3, 2}, {cols - 2, rows - 2}}

	for _, cellpx := range scales {
		t2 := 2 * specRimThickness(cellpx)
		wantStrips := 4
		switch {
		case t2 > cellpx:
			wantStrips = 1
		case t2 == cellpx:
			wantStrips = 2
		}

		for _, cell := range append(append([][2]int{}, corners...), interior...) {
			col, row := cell[0], cell[1]
			got := terrain.SelectionMarkerRects(col, row, cols, rows, cellpx)
			if len(got) != wantStrips {
				t.Fatalf("cellpx=%d cell (%d,%d): got %d rectangles, want %d", cellpx, col, row, len(got), wantStrips)
			}

			// (a) Wholly inside its own cell, strip by strip.
			cellRect := specCellRect(col, row, cellpx)
			for i, r := range got {
				if !r.In(cellRect) {
					t.Errorf("cellpx=%d cell (%d,%d): strip %d %v leaves its own cell %v — a neighbouring cell's "+
						"glyph could be covered by a pass that is drawn last (DD-8)", cellpx, col, row, i, r, cellRect)
				}
			}

			// (b) Pairwise disjoint, by rectangle intersection.
			for i := range got {
				for j := i + 1; j < len(got); j++ {
					if !got[i].Intersect(got[j]).Empty() {
						t.Errorf("cellpx=%d cell (%d,%d): strips %d %v and %d %v overlap; DD-8's four strips are DISJOINT",
							cellpx, col, row, i, got[i], j, got[j])
					}
				}
			}

			// (c) The clip is inert: the returned strips are the UNCLIPPED ones,
			//     even on a corner cell, where a glyph reaching outside its cell
			//     would be trimmed by the map rect.
			if raw := specSelectionRimRaw(col, row, cellpx); !sameRects(got, raw) {
				t.Errorf("cellpx=%d cell (%d,%d): got %v, want the unclipped rim %v — the map-rect clip must not "+
					"trim an in-map rim (DD-4's shared rule, inert here)", cellpx, col, row, got, raw)
			}
		}

		// (d) Hollow, stated over PIXELS, on one interior cell per scale. The
		//     large scales are skipped for this half alone: the claim is a
		//     rectangle identity and enumerating 16 million pixels to restate it
		//     would say nothing more.
		if cellpx > 64 {
			continue
		}
		const col, row = 3, 2
		x0, y0 := col*cellpx, row*cellpx
		th := specRimThickness(cellpx)
		covered := selectionPixelSet(terrain.SelectionMarkerRects(col, row, cols, rows, cellpx))
		for y := y0; y < y0+cellpx; y++ {
			for x := x0; x < x0+cellpx; x++ {
				onBorder := x < x0+th || x >= x0+cellpx-th || y < y0+th || y >= y0+cellpx-th
				if got := covered[image.Pt(x, y)]; got != onBorder {
					t.Fatalf("cellpx=%d: pixel (%d,%d) of cell (%d,%d) is covered=%v, want %v — the rim covers "+
						"exactly the footprint's %d-pixel border and leaves the interior, where the unit's art "+
						"stands, untouched (FR-2)", cellpx, x, y, col, row, got, onBorder, th)
				}
			}
		}
	}
}

func TestSelectionRimCoversNoOtherGlyph(t *testing.T) {
	const cols, rows, col, row = 5, 4, 2, 1

	others := []struct {
		name  string
		build func(col, row, cols, rows, cellpx int) []image.Rectangle
	}{
		{"ObjectMarkerRects", terrain.ObjectMarkerRects},
		{"UnitMarkerRects", terrain.UnitMarkerRects},
		{"StaticMarkerRects", terrain.StaticMarkerRects},
		{"EntityMarkerRects", terrain.EntityMarkerRects},
	}

	for _, cellpx := range []int{terrain.CellSize, 48, 64, 96, 128} {
		rim := selectionPixelSet(terrain.SelectionMarkerRects(col, row, cols, rows, cellpx))
		if len(rim) == 0 {
			t.Fatalf("cellpx=%d: the rim covered no pixel at cell (%d,%d)", cellpx, col, row)
		}
		for _, o := range others {
			glyph := selectionPixelSet(o.build(col, row, cols, rows, cellpx))
			if len(glyph) == 0 {
				t.Fatalf("cellpx=%d: %s covered no pixel at cell (%d,%d), so the comparison would be vacuous",
					cellpx, o.name, col, row)
			}
			for p := range glyph {
				if rim[p] {
					t.Errorf("cellpx=%d: the rim owns pixel %v, which %s also owns — the highlight is drawn LAST "+
						"and must cover no glyph on its own cell (DD-8)", cellpx, p, o.name)
				}
			}
		}
	}
}

// TestSelectionMarkerRectsRejectsOffMapAndBadScale covers SC-4's rejection
// clause: no geometry for an off-map cell or for cellpx < 1, never a panic,
// and no allocation on the way out.
//
// The wrap case is the one that says the off-map test is load-bearing rather
// than redundant with the clip. A column whose product with cellpx overflows
// wraps back to a small pixel coordinate, so a builder that skipped the cell
// test and relied on the map-rect clip alone would return a rim sitting on a
// perfectly plausible in-map cell.
func TestSelectionMarkerRectsRejectsOffMapAndBadScale(t *testing.T) {
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
		noPanicOverlay(t, "SelectionMarkerRects "+tc.name, func() {
			got = terrain.SelectionMarkerRects(tc.col, tc.row, cols, rows, tc.cellpx)
		})
		if got != nil {
			t.Errorf("%s: SelectionMarkerRects(%d,%d,%d,%d,%d) = %v, want nil (FR-2: no geometry at all)",
				tc.name, tc.col, tc.row, cols, rows, tc.cellpx, got)
		}
		if n := testing.AllocsPerRun(100, func() {
			sinkRects = terrain.SelectionMarkerRects(tc.col, tc.row, cols, rows, tc.cellpx)
		}); n != 0 {
			t.Errorf("%s: a rejected cell returns before any geometry is built, so it must not allocate; got %.0f allocation(s)",
				tc.name, n)
		}
	}
}

// TestSelectionMarkerColorDiffersFromEveryOtherGlyph covers SC-4's colour
// clause: the highlight's colour equals none of the four glyph colours
// already in the frame, and is opaque like all of them.
//
// It is not a style rule. The highlight is drawn LAST, over the entity's own
// sprite and beside all three diagnostic crosses, so sharing a colour with any
// of them would make "this unit is selected" indistinguishable from a marker
// that was already there — and the pass order, which is the only automated
// guard the stories have, is observed BY COLOUR.
func TestSelectionMarkerColorDiffersFromEveryOtherGlyph(t *testing.T) {
	others := []struct {
		name string
		c    color.RGBA
	}{
		{"MarkerColor (placed objects, yellow)", terrain.MarkerColor},
		{"UnitMarkerColor (placed units, cyan)", terrain.UnitMarkerColor},
		{"StaticMarkerColor (static objects, red)", terrain.StaticMarkerColor},
		{"EntityMarkerColor (simulated entities, magenta)", terrain.EntityMarkerColor},
	}
	for _, o := range others {
		if terrain.SelectionMarkerColor == o.c {
			t.Errorf("SelectionMarkerColor = %v, the same colour as %s; FR-2 requires it to differ from all four",
				terrain.SelectionMarkerColor, o.name)
		}
	}
	if terrain.SelectionMarkerColor.A != 0xff {
		t.Errorf("SelectionMarkerColor alpha = %#x, want 0xff — every marker colour in this family is opaque",
			terrain.SelectionMarkerColor.A)
	}
}
