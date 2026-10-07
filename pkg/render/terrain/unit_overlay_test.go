package terrain_test

// Tests for the placed-units diagnostic overlay geometry of pkg/render/terrain
// (work item 0009-units-overlay).
//
// SEPARATE CONTEXT.
//
//	round(n/32) = floor((n+16)/32)                  (n >= 0)
//	half        = floor(cellpx/2)
//	(cx,cy)     = (ax*cellpx+half, ay*cellpx+half)  the centre pixel
//	r           = max(1, round(4*cellpx/32))        arm radius
//	t           = max(1, round(1*cellpx/32))        arm thickness
//	lo          = floor(t/2)                        centred-strip low offset
//	horizontal arm = [cx-r,  cx+r+1) x [cy-lo, cy-lo+t)
//	vertical   arm = [cx-lo, cx-lo+t) x [cy-r,  cy+r+1)
//
// The unit constants (radius 4, thickness 1) differ from the object overlay's
// (radius 6, thickness 3), so the helpers below are unit-named twins of
// 0008's object helpers rather than shared code: specUnitArmsRaw, specUnitCross
// and drawUnitsAndCheck. The scale-free helpers of overlay_test.go
// (specRound, specMapRect, sameRects, noPanicOverlay, sinkRects, markerBG) are
// reused verbatim; overlay_test.go itself is the frozen 0008 characterization
// pin and is not modified here.
//
// Every fixture is synthetic; nothing here reads a game install.

import (
	"image"
	"image/color"
	"math/bits"
	"testing"

	"againrom/pkg/render/terrain"
)

func specUnitArmsRaw(col, row, cellpx int) [2]image.Rectangle {
	half := cellpx / 2
	cx, cy := col*cellpx+half, row*cellpx+half
	r := max(1, specRound(4*cellpx))
	t := max(1, specRound(1*cellpx))
	lo := t / 2
	return [2]image.Rectangle{
		image.Rect(cx-r, cy-lo, cx+r+1, cy-lo+t), // horizontal
		image.Rect(cx-lo, cy-r, cx-lo+t, cy+r+1), // vertical
	}
}

func specUnitCross(col, row, cols, rows, cellpx int) []image.Rectangle {
	if cellpx < 1 || col < 0 || row < 0 || col >= cols || row >= rows {
		return nil
	}
	clip := specMapRect(cols, rows, cellpx)
	out := make([]image.Rectangle, 0, 2)
	for _, arm := range specUnitArmsRaw(col, row, cellpx) {
		if c := arm.Intersect(clip); !c.Empty() {
			out = append(out, c)
		}
	}
	return out
}

func TestUnitAnchorCellAndOffMap(t *testing.T) {
	const cols, rows = 8, 6
	const cellpx = terrain.CellSize // 32, the native scale

	if terrain.CellSize != 32 {
		t.Fatalf("CellSize = %d, want 32; the FR-6 native derivation assumes 32", terrain.CellSize)
	}

	cases := []struct {
		name  string
		x, y  uint32
		col   int
		row   int
		inMap bool
	}{
		// 0x180/256 = 1.500, 0x27F/256 = 2.496 -> cell (1,2)
		{"1.500,2.496 -> (1,2)", 0x0000_0180, 0x0000_027F, 1, 2, true},
		// 0x101/256 = 1.004, 0x201/256 = 2.004 -> cell (1,2) again: a SECOND unit
		// on the same cell, reached through a different fractional low byte.
		{"1.004,2.004 -> (1,2) coincident", 0x0000_0101, 0x0000_0201, 1, 2, true},
		// 0xFF/256 = 0.996, 0x01/256 = 0.004 -> cell (0,0), the map corner
		{"0.996,0.004 -> (0,0) corner", 0x0000_00FF, 0x0000_0001, 0, 0, true},
		// 0x7FE/256 = 7.992, 0x57F/256 = 5.496 -> cell (7,5), the last cell
		{"7.992,5.496 -> (7,5) last cell", 0x0000_07FE, 0x0000_057F, 7, 5, true},
		// 0x801/256 = 8.004 -> col 8, one past cols=8
		{"8.004,1.500 -> col 8 off-map", 0x0000_0801, 0x0000_0180, 8, 1, false},
		// 0x601/256 = 6.004 -> row 6, one past rows=6
		{"2.504,6.004 -> row 6 off-map", 0x0000_0281, 0x0000_0601, 2, 6, false},
		// the widest u32: 0xFFFFFFFF>>8 = 0xFFFFFF on both axes
		{"max u32 -> (0xFFFFFF,0xFFFFFF) off-map", 0xFFFF_FFFF, 0xFFFF_FFFF, 0xFFFFFF, 0xFFFFFF, false},
	}

	for _, tc := range cases {
		// DD3: AnchorCell is exactly the >>8 shift, nothing else, and it is the
		// same function the object overlay uses.
		gotCol, gotRow := terrain.AnchorCell(tc.x, tc.y)
		if gotCol != int(tc.x>>8) || gotRow != int(tc.y>>8) {
			t.Errorf("%s: AnchorCell(%#x,%#x) = (%d,%d), want (%d,%d) = (X>>8,Y>>8) per FR-1/DD3",
				tc.name, tc.x, tc.y, gotCol, gotRow, int(tc.x>>8), int(tc.y>>8))
			continue
		}
		if gotCol != tc.col || gotRow != tc.row {
			t.Errorf("%s: hand-derived cell drift: AnchorCell gave (%d,%d), the table says (%d,%d)",
				tc.name, gotCol, gotRow, tc.col, tc.row)
			continue
		}

		var got []image.Rectangle
		noPanicOverlay(t, "UnitMarkerRects", func() {
			got = terrain.UnitMarkerRects(gotCol, gotRow, cols, rows, cellpx)
		})

		if len(got) > 2 {
			t.Errorf("%s: got %d rectangles, want at most 2 (P-1: anchor cross only): %v",
				tc.name, len(got), got)
		}

		if !tc.inMap {
			if got != nil {
				t.Errorf("%s: an off-map anchor (%d,%d) in a %dx%d map must contribute nothing (AC-1, FR-3), got %v",
					tc.name, gotCol, gotRow, cols, rows, got)
			}
			continue
		}

		want := specUnitCross(gotCol, gotRow, cols, rows, cellpx)
		if len(want) != 2 {
			t.Fatalf("%s: oracle drift: an in-map anchor must yield 2 arms, oracle gave %v", tc.name, want)
		}
		if !sameRects(got, want) {
			t.Errorf("%s: UnitMarkerRects(%d,%d,%d,%d,%d) = %v, want the FR-6 unit cross %v",
				tc.name, gotCol, gotRow, cols, rows, cellpx, got, want)
		}
	}

	// AC-1: two units resolving to the same anchor cell each contribute a
	// marker and the markers are IDENTICAL, so the visible result is one
	// marker. The two coordinate pairs below differ in both fractional low
	// bytes.
	firstCol, firstRow := terrain.AnchorCell(0x0000_0180, 0x0000_027F)
	secondCol, secondRow := terrain.AnchorCell(0x0000_0101, 0x0000_0201)
	if firstCol != secondCol || firstRow != secondRow {
		t.Fatalf("fixture: the two coincident anchors resolved to (%d,%d) and (%d,%d); they must share a cell",
			firstCol, firstRow, secondCol, secondRow)
	}
	firstGeom := terrain.UnitMarkerRects(firstCol, firstRow, cols, rows, cellpx)
	secondGeom := terrain.UnitMarkerRects(secondCol, secondRow, cols, rows, cellpx)
	if !sameRects(firstGeom, secondGeom) {
		t.Errorf("AC-1: two units on cell (%d,%d) gave %v and %v; coincident units must yield identical geometry",
			firstCol, firstRow, firstGeom, secondGeom)
	}

	// AC-1: the fractional low byte changes no output. The integral cell is held
	// at (3,4) while both low bytes sweep the extremes of the /256 fraction.
	const lowCol, lowRow = 3, 4
	base := specUnitCross(lowCol, lowRow, cols, rows, cellpx)
	for _, lowX := range []uint32{0x00, 0x01, 0x7F, 0x80, 0xFF} {
		for _, lowY := range []uint32{0x00, 0x01, 0x7F, 0x80, 0xFF} {
			x := uint32(lowCol)<<8 | lowX
			y := uint32(lowRow)<<8 | lowY
			c, r := terrain.AnchorCell(x, y)
			if c != lowCol || r != lowRow {
				t.Fatalf("AnchorCell(%#x,%#x) = (%d,%d), want the fixed cell (%d,%d): the low byte must be truncated away",
					x, y, c, r, lowCol, lowRow)
			}
			if got := terrain.UnitMarkerRects(c, r, cols, rows, cellpx); !sameRects(got, base) {
				t.Errorf("AC-1: low bytes (%#02x,%#02x) gave %v, want %v; the fractional /256 part must change no output",
					lowX, lowY, got, base)
			}
		}
	}

	spot := []image.Rectangle{
		image.Rect(44, 80, 53, 81),
		image.Rect(48, 76, 49, 85),
	}
	if oracle := specUnitCross(1, 2, cols, rows, cellpx); !sameRects(oracle, spot) {
		t.Fatalf("derivation drift: oracle gave %v for cell (1,2) at cellpx=32, hand-computed FR-6 says %v",
			oracle, spot)
	}
	if got := terrain.UnitMarkerRects(1, 2, cols, rows, cellpx); !sameRects(got, spot) {
		t.Errorf("cell (1,2) at cellpx=32: got %v, want the hand-computed FR-6 unit cross %v (FR-6, AC-2)", got, spot)
	}
}

func TestUnitMarkerRectsGeometry(t *testing.T) {
	cases := []struct {
		name       string
		col, row   int
		cols, rows int
		cellpx     int
		truncates  bool
		want       []image.Rectangle
	}{
		// ---- cellpx = 32 (native): r = round(128/32) = floor(144/32) = 4,
		//      t = round(32/32) = floor(48/32) = 1, lo = 0, half = 16.
		//      This is the spec's stated native pair [cx-4,cx+5)x[cy,cy+1),
		//      [cx,cx+1)x[cy-4,cy+5). Map is 4x3 -> [0,128)x[0,96).
		{
			// cx = 0*32+16 = 16, cy = 16 (the near corner cell)
			"cellpx=32 cell(0,0) corner", 0, 0, 4, 3, 32, false,
			[]image.Rectangle{image.Rect(12, 16, 21, 17), image.Rect(16, 12, 17, 21)},
		},
		{
			// cx = 2*32+16 = 80, cy = 1*32+16 = 48
			"cellpx=32 cell(2,1)", 2, 1, 4, 3, 32, false,
			[]image.Rectangle{image.Rect(76, 48, 85, 49), image.Rect(80, 44, 81, 53)},
		},
		{
			// cx = 3*32+16 = 112, cy = 2*32+16 = 80 (the far corner cell)
			"cellpx=32 cell(3,2) far corner", 3, 2, 4, 3, 32, false,
			[]image.Rectangle{image.Rect(108, 80, 117, 81), image.Rect(112, 76, 113, 85)},
		},

		// ---- cellpx = 15 (odd): r = floor((60+16)/32) = floor(76/32) = 2,
		//      t = floor((15+16)/32) = floor(31/32) = 0 -> raised to 1 by the
		//      max(1,...) guard, lo = 0, half = 7. Map 4x3 -> [0,60)x[0,45).
		{
			// cx = 7, cy = 7
			"cellpx=15 cell(0,0)", 0, 0, 4, 3, 15, false,
			[]image.Rectangle{image.Rect(5, 7, 10, 8), image.Rect(7, 5, 8, 10)},
		},
		{
			// cx = 2*15+7 = 37, cy = 1*15+7 = 22
			"cellpx=15 cell(2,1)", 2, 1, 4, 3, 15, false,
			[]image.Rectangle{image.Rect(35, 22, 40, 23), image.Rect(37, 20, 38, 25)},
		},
		{
			// cx = 3*15+7 = 52, cy = 2*15+7 = 37
			"cellpx=15 cell(3,2)", 3, 2, 4, 3, 15, false,
			[]image.Rectangle{image.Rect(50, 37, 55, 38), image.Rect(52, 35, 53, 40)},
		},

		// ---- cellpx = 16 (even): r = floor((64+16)/32) = floor(80/32) = 2,
		//      t = floor((16+16)/32) = 1, lo = 0, half = 8.
		//      Map 4x3 -> [0,64)x[0,48).
		{
			// cx = 8, cy = 8
			"cellpx=16 cell(0,0)", 0, 0, 4, 3, 16, false,
			[]image.Rectangle{image.Rect(6, 8, 11, 9), image.Rect(8, 6, 9, 11)},
		},
		{
			// cx = 2*16+8 = 40, cy = 1*16+8 = 24
			"cellpx=16 cell(2,1)", 2, 1, 4, 3, 16, false,
			[]image.Rectangle{image.Rect(38, 24, 43, 25), image.Rect(40, 22, 41, 27)},
		},
		{
			// cx = 3*16+8 = 56, cy = 2*16+8 = 40
			"cellpx=16 cell(3,2)", 3, 2, 4, 3, 16, false,
			[]image.Rectangle{image.Rect(54, 40, 59, 41), image.Rect(56, 38, 57, 43)},
		},

		// ---- cellpx = 17 (odd): r = floor((68+16)/32) = floor(84/32) = 2,
		//      t = floor((17+16)/32) = floor(33/32) = 1, lo = 0, half = 8.
		//      NOTE the asymmetry with the object overlay (R-6): at cellpx=17 the
		//      OBJECT thickness is already even, the UNIT thickness is still 1.
		//      Map 4x3 -> [0,68)x[0,51).
		{
			// cx = 8, cy = 8
			"cellpx=17 cell(0,0)", 0, 0, 4, 3, 17, false,
			[]image.Rectangle{image.Rect(6, 8, 11, 9), image.Rect(8, 6, 9, 11)},
		},
		{
			// cx = 2*17+8 = 42, cy = 1*17+8 = 25
			"cellpx=17 cell(2,1)", 2, 1, 4, 3, 17, false,
			[]image.Rectangle{image.Rect(40, 25, 45, 26), image.Rect(42, 23, 43, 28)},
		},
		{
			// cx = 3*17+8 = 59, cy = 2*17+8 = 42
			"cellpx=17 cell(3,2)", 3, 2, 4, 3, 17, false,
			[]image.Rectangle{image.Rect(57, 42, 62, 43), image.Rect(59, 40, 60, 45)},
		},

		{
			// cx = 24, cy = 24; H = [18,31)x[23,25), V = [23,25)x[18,31)
			"cellpx=48 cell(0,0)", 0, 0, 4, 3, 48, false,
			[]image.Rectangle{image.Rect(18, 23, 31, 25), image.Rect(23, 18, 25, 31)},
		},
		{
			// cx = 2*48+24 = 120, cy = 1*48+24 = 72
			"cellpx=48 cell(2,1)", 2, 1, 4, 3, 48, false,
			[]image.Rectangle{image.Rect(114, 71, 127, 73), image.Rect(119, 66, 121, 79)},
		},
		{
			// cx = 3*48+24 = 168, cy = 2*48+24 = 120
			"cellpx=48 cell(3,2)", 3, 2, 4, 3, 48, false,
			[]image.Rectangle{image.Rect(162, 119, 175, 121), image.Rect(167, 114, 169, 127)},
		},

		// ---- cellpx = 64 (= CellSize*2, the first non-native scale a shipped
		//      terraintool invocation can emit): r = floor((256+16)/32) =
		//      floor(272/32) = 8, t = floor((64+16)/32) = floor(80/32) = 2,
		//      lo = 1, half = 32. Map 4x3 -> [0,256)x[0,192).
		{
			// cx = 32, cy = 32
			"cellpx=64 cell(0,0)", 0, 0, 4, 3, 64, false,
			[]image.Rectangle{image.Rect(24, 31, 41, 33), image.Rect(31, 24, 33, 41)},
		},
		{
			// cx = 2*64+32 = 160, cy = 1*64+32 = 96
			"cellpx=64 cell(2,1)", 2, 1, 4, 3, 64, false,
			[]image.Rectangle{image.Rect(152, 95, 169, 97), image.Rect(159, 88, 161, 105)},
		},
		{
			// cx = 3*64+32 = 224, cy = 2*64+32 = 160
			"cellpx=64 cell(3,2)", 3, 2, 4, 3, 64, false,
			[]image.Rectangle{image.Rect(216, 159, 233, 161), image.Rect(223, 152, 225, 169)},
		},

		// ---- cellpx = 96 (= CellSize*3): r = floor((384+16)/32) =
		//      floor(400/32) = 12, t = floor((96+16)/32) = floor(112/32) = 3,
		//      lo = 1, half = 48. An ODD thickness with a non-zero lo, proving
		//      the centring term is not simply "1 at every large scale" (DD4).
		//      Map 4x3 -> [0,384)x[0,288).
		{
			// cx = 48, cy = 48
			"cellpx=96 cell(0,0)", 0, 0, 4, 3, 96, false,
			[]image.Rectangle{image.Rect(36, 47, 61, 50), image.Rect(47, 36, 50, 61)},
		},
		{
			// cx = 2*96+48 = 240, cy = 1*96+48 = 144
			"cellpx=96 cell(2,1)", 2, 1, 4, 3, 96, false,
			[]image.Rectangle{image.Rect(228, 143, 253, 146), image.Rect(239, 132, 242, 157)},
		},
		{
			// cx = 3*96+48 = 336, cy = 2*96+48 = 240
			"cellpx=96 cell(3,2)", 3, 2, 4, 3, 96, false,
			[]image.Rectangle{image.Rect(324, 239, 349, 242), image.Rect(335, 228, 338, 253)},
		},

		// ---- The clip case (DD4): the unit cross fits strictly inside its own
		//      cell from cellpx = 3 upward, so the MAP-rect clip is observable
		//      only at cellpx in {1,2}. r and t both floor to 0 there and are
		//      raised to 1 by the max(1,...) guard.
		{
			// 1x1 map at cellpx=1: half=0 -> cx=cy=0; r=1, t=1, lo=0.
			// Raw H = [-1,2)x[0,1), raw V = [0,1)x[-1,2); the map is [0,1)x[0,1),
			// so BOTH arms overrun on every side and both truncate to the single
			// map pixel. Two arms survive (neither is empty).
			"cellpx=1 on a 1x1 map", 0, 0, 1, 1, 1, true,
			[]image.Rectangle{image.Rect(0, 0, 1, 1), image.Rect(0, 0, 1, 1)},
		},
		{
			// 1x1 map at cellpx=2: half=1 -> cx=cy=1; r=1, t=1, lo=0.
			// Raw H = [0,3)x[1,2) truncates to [0,2)x[1,2);
			// raw V = [1,2)x[0,3) truncates to [1,2)x[0,2).
			"cellpx=2 on a 1x1 map", 0, 0, 1, 1, 2, true,
			[]image.Rectangle{image.Rect(0, 1, 2, 2), image.Rect(1, 0, 2, 2)},
		},
	}

	for _, tc := range cases {
		if oracle := specUnitCross(tc.col, tc.row, tc.cols, tc.rows, tc.cellpx); !sameRects(oracle, tc.want) {
			t.Fatalf("%s: derivation drift: FR-6 oracle says %v, the hand-computed table says %v",
				tc.name, oracle, tc.want)
		}

		var got []image.Rectangle
		noPanicOverlay(t, tc.name, func() {
			got = terrain.UnitMarkerRects(tc.col, tc.row, tc.cols, tc.rows, tc.cellpx)
		})
		if !sameRects(got, tc.want) {
			t.Errorf("%s: UnitMarkerRects(%d,%d,%d,%d,%d) = %v, want the FR-6 rectangles %v (AC-2)",
				tc.name, tc.col, tc.row, tc.cols, tc.rows, tc.cellpx, got, tc.want)
			continue
		}

		raw := specUnitArmsRaw(tc.col, tc.row, tc.cellpx)
		clip := specMapRect(tc.cols, tc.rows, tc.cellpx)

		if !tc.truncates {
			// Fixture guard for the DD4 note: at these scales the whole cross
			// fits strictly inside its own cell, so the map-rect clip must have
			// changed nothing at all - asserted, not assumed.
			if len(got) != 2 {
				t.Fatalf("%s: got %d arms, want 2 (nothing is clipped away at this scale)", tc.name, len(got))
			}
			for i, arm := range raw {
				if !arm.In(clip) {
					t.Fatalf("%s: fixture error: the unclipped FR-6 arm %d %v is not contained in the map rect %v, so this case cannot pin a no-op clip",
						tc.name, i, arm, clip)
				}
				if got[i] != arm {
					t.Errorf("%s: arm %d = %v but the unclipped FR-6 arm is %v; this scale's cross fits inside its cell, so the map clip must be a no-op (FR-3)",
						tc.name, i, got[i], arm)
				}
			}
			continue
		}

		if len(got) != len(raw) {
			t.Fatalf("%s: got %d arms, want %d (neither arm is empty after the clip here)",
				tc.name, len(got), len(raw))
		}
		for i, arm := range raw {
			trunc := image.Rect(
				max(arm.Min.X, clip.Min.X), max(arm.Min.Y, clip.Min.Y),
				min(arm.Max.X, clip.Max.X), min(arm.Max.Y, clip.Max.Y),
			)
			if got[i] != trunc {
				t.Errorf("%s: arm %d = %v, want %v = max/min truncation of the unclipped FR-6 arm %v against the map rect %v (FR-3: no synthetic border)",
					tc.name, i, got[i], trunc, arm, clip)
			}
			if got[i] == arm {
				t.Errorf("%s: arm %d = %v was not truncated at all, but the unclipped FR-6 arm %v overruns the map rect %v; this case is meant to exercise the FR-3 clip",
					tc.name, i, got[i], arm, clip)
			}
		}
	}

	// SC-2, the centring term observed rather than assumed. An implementation
	// that dropped the -floor(t/2) term would place it at [c, c+t) instead,
	// which is only distinguishable once t >= 2, i.e. once cellpx >= 48 (DD4,
	// R-6).
	centring := []struct {
		cellpx   int
		col, row int
		cx, cy   int
		t, lo    int
	}{
		{48, 2, 1, 120, 72, 2, 1},  // t = floor((48+16)/32) = 2
		{64, 2, 1, 160, 96, 2, 1},  // t = floor((64+16)/32) = 2  (-scale 2)
		{96, 2, 1, 240, 144, 3, 1}, // t = floor((96+16)/32) = 3
	}
	for _, tc := range centring {
		arms := terrain.UnitMarkerRects(tc.col, tc.row, 4, 3, tc.cellpx)
		if len(arms) != 2 {
			t.Fatalf("cellpx=%d cell(%d,%d): got %d arms, want 2", tc.cellpx, tc.col, tc.row, len(arms))
		}
		wantLoY, wantHiY := tc.cy-tc.lo, tc.cy-tc.lo+tc.t
		if arms[0].Min.Y != wantLoY || arms[0].Max.Y != wantHiY {
			t.Errorf("FR-6 centring: cellpx=%d horizontal arm spans y [%d,%d), want [%d,%d) = [cy-floor(t/2), cy-floor(t/2)+t) with cy=%d, t=%d; dropping the -floor(t/2) term would give [%d,%d)",
				tc.cellpx, arms[0].Min.Y, arms[0].Max.Y, wantLoY, wantHiY, tc.cy, tc.t, tc.cy, tc.cy+tc.t)
		}
		wantLoX, wantHiX := tc.cx-tc.lo, tc.cx-tc.lo+tc.t
		if arms[1].Min.X != wantLoX || arms[1].Max.X != wantHiX {
			t.Errorf("FR-6 centring: cellpx=%d vertical arm spans x [%d,%d), want [%d,%d) = [cx-floor(t/2), cx-floor(t/2)+t) with cx=%d, t=%d; dropping the -floor(t/2) term would give [%d,%d)",
				tc.cellpx, arms[1].Min.X, arms[1].Max.X, wantLoX, wantHiX, tc.cx, tc.t, tc.cx, tc.cx+tc.t)
		}
	}

	// SC-2, the corner-anchor no-op clip, asserted outright: at cellpx = 32 the
	// unit cross has r = 4 inside a 32-pixel cell whose centre sits at offset
	// 16, so even the corner cells reach only 12 pixels from the cell edge.
	const ccols, crows, ccellpx = 4, 3, 32
	cclip := specMapRect(ccols, crows, ccellpx)
	for _, c := range []image.Point{{X: 0, Y: 0}, {X: ccols - 1, Y: 0}, {X: 0, Y: crows - 1}, {X: ccols - 1, Y: crows - 1}} {
		raw := specUnitArmsRaw(c.X, c.Y, ccellpx)
		got := terrain.UnitMarkerRects(c.X, c.Y, ccols, crows, ccellpx)
		if len(got) != 2 {
			t.Fatalf("corner cell (%d,%d) at cellpx=32: got %d arms, want 2", c.X, c.Y, len(got))
		}
		for i, arm := range raw {
			if !arm.In(cclip) {
				t.Fatalf("fixture error: corner cell (%d,%d) arm %d %v escapes the map rect %v at cellpx=32; the SC-2 no-truncation claim is wrong",
					c.X, c.Y, i, arm, cclip)
			}
			if got[i] != arm {
				t.Errorf("SC-2: corner cell (%d,%d) at cellpx=32: arm %d = %v, want the unclipped FR-6 arm %v; the cross fits inside its own cell here, so the map-rect clip must truncate nothing",
					c.X, c.Y, i, got[i], arm)
			}
		}
	}
}

func TestUnitMarkerRectsExtremes(t *testing.T) {
	for _, cellpx := range []int{0, -1, -32, -(1 << 30)} {
		var got []image.Rectangle
		noPanicOverlay(t, "UnitMarkerRects with an invalid cellpx", func() {
			got = terrain.UnitMarkerRects(1, 1, 4, 4, cellpx)
		})
		if got != nil {
			t.Errorf("cellpx = %d is invalid and must yield no geometry (FR-2, AC-3), got %v", cellpx, got)
		}
	}

	offMap := []struct {
		name       string
		col, row   int
		cols, rows int
		cellpx     int
	}{
		{"widest AnchorCell output vs a 4x4 map", 0xFFFFFF, 0xFFFFFF, 4, 4, 32},
		{"huge col, in-range row", 0xFFFFFF, 2, 4, 4, 32},
		{"huge row, in-range col", 1, 1 << 30, 4, 4, 32},
		{"negative col", -(1 << 30), 1, 4, 4, 32},
		{"negative row", 1, -1, 4, 4, 32},
	}
	for _, tc := range offMap {
		var got []image.Rectangle
		noPanicOverlay(t, "UnitMarkerRects off-map "+tc.name, func() {
			got = terrain.UnitMarkerRects(tc.col, tc.row, tc.cols, tc.rows, tc.cellpx)
		})
		if got != nil {
			t.Errorf("%s: anchor (%d,%d) is outside the %dx%d map and must contribute nothing, never wrapping into a visible marker (AC-1, AC-3, FR-3), got %v",
				tc.name, tc.col, tc.row, tc.cols, tc.rows, got)
		}
		if n := testing.AllocsPerRun(100, func() {
			sinkRects = terrain.UnitMarkerRects(tc.col, tc.row, tc.cols, tc.rows, tc.cellpx)
		}); n != 0 {
			t.Errorf("%s: an off-map anchor returns before any geometry is built, so it must not allocate (SC-3, P-2); got %.0f allocation(s)",
				tc.name, n)
		}
	}

	if bits.UintSize < 64 {
		t.Skip("the in-map extreme-magnitude case needs a 64-bit int")
	}
	const (
		bigCols   = 1 << 24
		bigRows   = 1 << 24
		bigCol    = bigCols - 1 // the widest AnchorCell output, 0xFFFFFF
		bigRow    = bigRows - 1
		bigCellpx = 1 << 12 // 4096 = CellSize*128, a deliberately large scale
	)
	var big []image.Rectangle
	noPanicOverlay(t, "UnitMarkerRects at extreme in-map magnitude", func() {
		big = terrain.UnitMarkerRects(bigCol, bigRow, bigCols, bigRows, bigCellpx)
	})
	if len(big) > 2 {
		t.Errorf("extreme in-map anchor: got %d rectangles, want at most 2 (P-1)", len(big))
	}
	// Derivation: half = 2048, cx = cy = (2^24-1)*4096 + 2048 = 68719474688; r
	// = floor((4*4096+16)/32) = floor(16400/32) = 512; t = floor((4096+16)/32)
	// = floor(4112/32) = 128; lo = 64.
	wantBig := specUnitCross(bigCol, bigRow, bigCols, bigRows, bigCellpx)
	if len(wantBig) != 2 || wantBig[0] != image.Rect(68719474176, 68719474624, 68719475201, 68719474752) {
		t.Fatalf("derivation drift: FR-6 oracle gave %v for the extreme in-map anchor", wantBig)
	}
	if !sameRects(big, wantBig) {
		t.Errorf("extreme in-map anchor: got %v, want the FR-6 unit cross %v (AC-3: arithmetic clipping, no wrap)",
			big, wantBig)
	}

	const runs = 200
	small := testing.AllocsPerRun(runs, func() {
		sinkRects = terrain.UnitMarkerRects(1, 1, 4, 4, 32)
	})
	huge := testing.AllocsPerRun(runs, func() {
		sinkRects = terrain.UnitMarkerRects(bigCol, bigRow, bigCols, bigRows, bigCellpx)
	})
	// Logged so the comparison below is visibly not two zeros: a fixed non-zero
	// count is fine, magnitude-dependence is not.
	t.Logf("allocations per call: small-magnitude anchor %.0f, huge-magnitude anchor %.0f", small, huge)
	if small != huge {
		t.Errorf("P-2: allocations must not depend on coordinate magnitude: cell (1,1) in a 4x4 map at cellpx=32 allocates %.0f, cell (%d,%d) in a %dx%d map at cellpx=%d allocates %.0f",
			small, bigCol, bigRow, bigCols, bigRows, bigCellpx, huge)
	}
}

// drawUnitsAndCheck paints markerBG over an imgW x imgH RGBA anchored at the
// origin (DD5's precondition), draws the cells with DrawUnitMarkers, and then
// asserts that EXACTLY the spec-derived pixels hold UnitMarkerColor and every
// other pixel still holds the background. It returns the marked-pixel count and
// the drawn image.
func drawUnitsAndCheck(t *testing.T, what string, imgW, imgH int, cells []image.Point, cols, rows, cellpx int) (int, *image.RGBA) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	for y := 0; y < imgH; y++ {
		for x := 0; x < imgW; x++ {
			img.SetRGBA(x, y, markerBG)
		}
	}

	want := make([]bool, imgW*imgH)
	for _, c := range cells {
		for _, arm := range specUnitCross(c.X, c.Y, cols, rows, cellpx) {
			cl := arm.Intersect(img.Bounds())
			for y := cl.Min.Y; y < cl.Max.Y; y++ {
				for x := cl.Min.X; x < cl.Max.X; x++ {
					want[y*imgW+x] = true
				}
			}
		}
	}

	noPanicOverlay(t, what+": DrawUnitMarkers", func() {
		terrain.DrawUnitMarkers(img, cells, cols, rows, cellpx)
	})

	marked := 0
	for y := 0; y < imgH; y++ {
		for x := 0; x < imgW; x++ {
			got := img.RGBAAt(x, y)
			if want[y*imgW+x] {
				marked++
				if got != terrain.UnitMarkerColor {
					t.Fatalf("%s: pixel (%d,%d) = %v, want UnitMarkerColor %v - it lies inside an FR-6 unit arm clipped to the map rect and the image bounds (SC-4, AC-2)",
						what, x, y, got, terrain.UnitMarkerColor)
				}
				continue
			}
			if got != markerBG {
				t.Fatalf("%s: pixel (%d,%d) = %v, want the untouched background %v - it lies outside every clipped FR-6 unit arm, and the overlay must touch nothing else (SC-4, P-1)",
					what, x, y, got, markerBG)
			}
		}
	}
	return marked, img
}

func TestDrawUnitMarkers(t *testing.T) {
	if want := (color.RGBA{R: 0x00, G: 0xE5, B: 0xFF, A: 0xFF}); terrain.UnitMarkerColor != want {
		t.Fatalf("FR-6: UnitMarkerColor = %v, want the opaque cyan #00E5FF %v", terrain.UnitMarkerColor, want)
	}
	if terrain.UnitMarkerColor == markerBG {
		t.Fatal("fixture: the background must differ from UnitMarkerColor")
	}
	if terrain.UnitMarkerColor == terrain.MarkerColor {
		t.Fatal("FR-6: the unit marker colour must differ from the object marker colour, or the FR-4 stacking test proves nothing")
	}

	// A native unit cross covers |H| + |V| - |H and V| = 9*1 + 1*9 - 1*1 = 17 px.
	const nativeCross = 17

	const cols, rows, cellpx = 5, 4, 32
	mapW, mapH := cols*cellpx, rows*cellpx // 160 x 128

	inMap := []image.Point{{X: 0, Y: 0}, {X: 2, Y: 1}, {X: 4, Y: 3}}
	offMap := []image.Point{{X: cols, Y: 0}, {X: 0, Y: rows}, {X: 99, Y: 99}, {X: -1, Y: -1}}

	t.Run("map extent", func(t *testing.T) {
		cells := append(append([]image.Point(nil), inMap...), offMap...)
		marked, _ := drawUnitsAndCheck(t, "map extent", mapW, mapH, cells, cols, rows, cellpx)
		if want := len(inMap) * nativeCross; marked != want {
			t.Errorf("marked %d pixels, want %d = %d in-map crosses x %d pixels (9x1 arm + 1x9 arm - 1x1 overlap); the off-map cells must contribute nothing (FR-3)",
				marked, want, len(inMap), nativeCross)
		}
	})

	t.Run("off-map cells only", func(t *testing.T) {
		marked, _ := drawUnitsAndCheck(t, "off-map only", mapW, mapH, offMap, cols, rows, cellpx)
		if marked != 0 {
			t.Errorf("marked %d pixels, want 0: every anchor in this list is off-map (AC-1, FR-3)", marked)
		}
	})

	t.Run("coincident units", func(t *testing.T) {
		one := []image.Point{{X: 2, Y: 1}}
		two := []image.Point{{X: 2, Y: 1}, {X: 2, Y: 1}}
		markedOne, imgOne := drawUnitsAndCheck(t, "one unit", mapW, mapH, one, cols, rows, cellpx)
		markedTwo, imgTwo := drawUnitsAndCheck(t, "two coincident units", mapW, mapH, two, cols, rows, cellpx)
		if markedOne != nativeCross || markedTwo != nativeCross {
			t.Errorf("one unit marked %d pixels and two coincident units marked %d, want %d each (FR-3: the visible result is one marker)",
				markedOne, markedTwo, nativeCross)
		}
		for i := range imgOne.Pix {
			if imgOne.Pix[i] != imgTwo.Pix[i] {
				t.Fatalf("FR-3: drawing two coincident units produced a different image than drawing one; the markers are identical and opaque, so the result must be byte-identical (first difference at Pix[%d])", i)
			}
		}
	})

	t.Run("empty cell list", func(t *testing.T) {
		marked, _ := drawUnitsAndCheck(t, "nil cells", mapW, mapH, nil, cols, rows, cellpx)
		if marked != 0 {
			t.Errorf("marked %d pixels, want 0 for an empty cell list (FR-2)", marked)
		}
		marked, _ = drawUnitsAndCheck(t, "empty cells", mapW, mapH, []image.Point{}, cols, rows, cellpx)
		if marked != 0 {
			t.Errorf("marked %d pixels, want 0 for an empty cell list (FR-2)", marked)
		}
	})

	t.Run("no-op targets", func(t *testing.T) {
		noPanicOverlay(t, "DrawUnitMarkers on a nil image", func() {
			terrain.DrawUnitMarkers(nil, inMap, cols, rows, cellpx)
		})
		empty := image.NewRGBA(image.Rect(0, 0, 0, 0))
		noPanicOverlay(t, "DrawUnitMarkers on an empty target rectangle", func() {
			terrain.DrawUnitMarkers(empty, inMap, cols, rows, cellpx)
		})
		if len(empty.Pix) != 0 {
			t.Errorf("fixture: an empty target rectangle must have no pixels, got %d bytes", len(empty.Pix))
		}
		// AC-3: an invalid scale draws nothing either.
		marked, _ := drawUnitsAndCheck(t, "invalid scale", 16, 16, inMap, cols, rows, 0)
		if marked != 0 {
			t.Errorf("marked %d pixels at cellpx=0, want 0 (FR-2: an invalid scale yields no geometry)", marked)
		}
	})

	t.Run("undersized image at the origin", func(t *testing.T) {
		const imgW, imgH = 82, 50
		cells := []image.Point{{X: 0, Y: 0}, {X: 2, Y: 1}, {X: 4, Y: 3}, {X: 7, Y: 0}}
		marked, img := drawUnitsAndCheck(t, "undersized image", imgW, imgH, cells, cols, rows, cellpx)
		if want := nativeCross + 11; marked != want {
			t.Errorf("marked %d pixels, want %d = 17 (the whole cross at cell (0,0)) + 11 (cell (2,1) truncated to the image bounds); cell (4,3) and the off-map cell draw nothing",
				marked, want)
		}

		if got := img.RGBAAt(imgW-1, 48); got != terrain.UnitMarkerColor {
			t.Errorf("pixel (%d,48) = %v, want UnitMarkerColor: the arm [76,85) is truncated to the image edge, not inset from it (FR-3: no synthetic border)",
				imgW-1, got)
		}
		// The vertical arm of cell (2,1) is truncated at the bottom edge the same
		// way: its last in-bounds row must be drawn.
		if got := img.RGBAAt(80, imgH-1); got != terrain.UnitMarkerColor {
			t.Errorf("pixel (80,%d) = %v, want UnitMarkerColor: the vertical arm [44,53) is truncated to the image edge, not inset from it (FR-3)",
				imgH-1, got)
		}
		// And the column just outside the clipped arm inside the image stays
		// background, so the clip did not smear the marker along the boundary.
		if got := img.RGBAAt(75, 48); got != markerBG {
			t.Errorf("pixel (75,48) = %v, want the background %v: the horizontal arm starts at x=76 (FR-6)", got, markerBG)
		}
	})

	t.Run("FR-4 stacking order", func(t *testing.T) {
		const col, row = 2, 1
		cells := []image.Point{{X: col, Y: row}}

		unitArms := specUnitCross(col, row, cols, rows, cellpx)
		objectArms := specCross(col, row, cols, rows, cellpx)
		if len(unitArms) != 2 || len(objectArms) != 2 {
			t.Fatalf("fixture: want 2 unit arms and 2 object arms at cell (%d,%d), got %v and %v", col, row, unitArms, objectArms)
		}

		mask := func(arms []image.Rectangle) []bool {
			m := make([]bool, mapW*mapH)
			for _, a := range arms {
				for y := a.Min.Y; y < a.Max.Y; y++ {
					for x := a.Min.X; x < a.Max.X; x++ {
						m[y*mapW+x] = true
					}
				}
			}
			return m
		}
		unitMask, objectMask := mask(unitArms), mask(objectArms)

		// Fixture guard for the strict-subset property the whole test rests on.
		unitPx, objectPx := 0, 0
		for i := range unitMask {
			if unitMask[i] {
				unitPx++
				if !objectMask[i] {
					t.Fatalf("fixture: the unit cross is not a subset of the object cross (pixel index %d); the FR-4 order witness only works because it is", i)
				}
			}
			if objectMask[i] {
				objectPx++
			}
		}
		if unitPx != nativeCross || objectPx <= unitPx {
			t.Fatalf("fixture: unit cross %d px, object cross %d px; want %d and strictly more", unitPx, objectPx, nativeCross)
		}

		newBG := func() *image.RGBA {
			img := image.NewRGBA(image.Rect(0, 0, mapW, mapH))
			for y := 0; y < mapH; y++ {
				for x := 0; x < mapW; x++ {
					img.SetRGBA(x, y, markerBG)
				}
			}
			return img
		}

		correct := newBG()
		noPanicOverlay(t, "objects then units", func() {
			terrain.DrawObjectMarkers(correct, cells, cols, rows, cellpx)
			terrain.DrawUnitMarkers(correct, cells, cols, rows, cellpx)
		})
		for y := 0; y < mapH; y++ {
			for x := 0; x < mapW; x++ {
				i := y*mapW + x
				got := correct.RGBAAt(x, y)
				switch {
				case unitMask[i]:
					if got != terrain.UnitMarkerColor {
						t.Fatalf("FR-4: pixel (%d,%d) = %v, want UnitMarkerColor %v - with the draw order terrain -> objects -> units, every pixel of the unit cross must be cyan",
							x, y, got, terrain.UnitMarkerColor)
					}
				case objectMask[i]:
					if got != terrain.MarkerColor {
						t.Fatalf("FR-4: pixel (%d,%d) = %v, want MarkerColor %v - the unit cross does not cover this object pixel, so the object marker must survive underneath",
							x, y, got, terrain.MarkerColor)
					}
				default:
					if got != markerBG {
						t.Fatalf("FR-4: pixel (%d,%d) = %v, want the untouched background %v - it lies outside both crosses", x, y, got, markerBG)
					}
				}
			}
		}

		reversed := newBG()
		noPanicOverlay(t, "units then objects", func() {
			terrain.DrawUnitMarkers(reversed, cells, cols, rows, cellpx)
			terrain.DrawObjectMarkers(reversed, cells, cols, rows, cellpx)
		})
		cyan := 0
		for y := 0; y < mapH; y++ {
			for x := 0; x < mapW; x++ {
				i := y*mapW + x
				got := reversed.RGBAAt(x, y)
				if got == terrain.UnitMarkerColor {
					cyan++
				}
				if objectMask[i] {
					if got != terrain.MarkerColor {
						t.Fatalf("pixel (%d,%d) = %v, want MarkerColor %v - the object pass ran second here and its store is unconditional and opaque",
							x, y, got, terrain.MarkerColor)
					}
					continue
				}
				if got != markerBG {
					t.Fatalf("pixel (%d,%d) = %v, want the untouched background %v", x, y, got, markerBG)
				}
			}
		}
		if cyan != 0 {
			t.Errorf("the units-then-objects order left %d UnitMarkerColor pixel(s), want 0: the unit cross is a strict subset of the object cross, so a reversed order must erase it completely - this is the control that makes the correct-order assertion above meaningful (FR-4, R-4)",
				cyan)
		}
	})
}
