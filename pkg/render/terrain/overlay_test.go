package terrain_test

// Tests for the placed-objects diagnostic overlay geometry of
// pkg/render/terrain (work item 0008-structures-overlay).
//
// SEPARATE CONTEXT.
//
//	round(n/32) = floor((n+16)/32)                  (n >= 0)
//	half        = floor(cellpx/2)
//	(cx,cy)     = (ax*cellpx+half, ay*cellpx+half)  the centre pixel
//	r           = max(1, round(6*cellpx/32))        arm radius
//	t           = max(1, round(3*cellpx/32))        arm thickness
//	lo          = floor(t/2)                        centred-strip low offset
//	horizontal arm = [cx-r,  cx+r+1) x [cy-lo, cy-lo+t)
//	vertical   arm = [cx-lo, cx-lo+t) x [cy-r,  cy+r+1)
//
// Every fixture is synthetic; nothing here reads a game install.

import (
	"image"
	"image/color"
	"math/bits"
	"testing"

	"againrom/pkg/render/terrain"
)

// sinkRects keeps ObjectMarkerRects' result escaping so the allocation
// measurement in TestObjectMarkerRectsExtremes is not optimised away.
var sinkRects []image.Rectangle

func specRound(n int) int { return (n + 16) / 32 }

func specArmsRaw(col, row, cellpx int) [2]image.Rectangle {
	half := cellpx / 2
	cx, cy := col*cellpx+half, row*cellpx+half
	r := max(1, specRound(6*cellpx))
	t := max(1, specRound(3*cellpx))
	lo := t / 2
	return [2]image.Rectangle{
		image.Rect(cx-r, cy-lo, cx+r+1, cy-lo+t), // horizontal
		image.Rect(cx-lo, cy-r, cx-lo+t, cy+r+1), // vertical
	}
}

func specMapRect(cols, rows, cellpx int) image.Rectangle {
	return image.Rect(0, 0, cols*cellpx, rows*cellpx)
}

func specCross(col, row, cols, rows, cellpx int) []image.Rectangle {
	if cellpx < 1 || col < 0 || row < 0 || col >= cols || row >= rows {
		return nil
	}
	clip := specMapRect(cols, rows, cellpx)
	out := make([]image.Rectangle, 0, 2)
	for _, arm := range specArmsRaw(col, row, cellpx) {
		if c := arm.Intersect(clip); !c.Empty() {
			out = append(out, c)
		}
	}
	return out
}

// sameRects reports whether two rectangle slices are element-wise identical.
func sameRects(a, b []image.Rectangle) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func noPanicOverlay(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("%s panicked: %v (FR-2 requires no geometry, never a panic)", name, rec)
		}
	}()
	fn()
}

func TestAnchorCellAndOffMap(t *testing.T) {
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
		// DD3: AnchorCell is exactly the >>8 shift, nothing else.
		gotCol, gotRow := terrain.AnchorCell(tc.x, tc.y)
		if gotCol != int(tc.x>>8) || gotRow != int(tc.y>>8) {
			t.Errorf("%s: AnchorCell(%#x,%#x) = (%d,%d), want (%d,%d) = (X>>8,Y>>8) per DD3",
				tc.name, tc.x, tc.y, gotCol, gotRow, int(tc.x>>8), int(tc.y>>8))
			continue
		}
		if gotCol != tc.col || gotRow != tc.row {
			t.Errorf("%s: hand-derived cell drift: AnchorCell gave (%d,%d), the table says (%d,%d)",
				tc.name, gotCol, gotRow, tc.col, tc.row)
			continue
		}

		var got []image.Rectangle
		noPanicOverlay(t, "ObjectMarkerRects", func() {
			got = terrain.ObjectMarkerRects(gotCol, gotRow, cols, rows, cellpx)
		})

		if len(got) > 2 {
			t.Errorf("%s: got %d rectangles, want at most 2 (P-1: anchor cross only, no footprint): %v",
				tc.name, len(got), got)
		}

		if !tc.inMap {
			if got != nil {
				t.Errorf("%s: an off-map anchor (%d,%d) in a %dx%d map must contribute nothing (AC-1, FR-3), got %v",
					tc.name, gotCol, gotRow, cols, rows, got)
			}
			continue
		}

		want := specCross(gotCol, gotRow, cols, rows, cellpx)
		if len(want) != 2 {
			t.Fatalf("%s: oracle drift: an in-map anchor must yield 2 arms, oracle gave %v", tc.name, want)
		}
		if !sameRects(got, want) {
			t.Errorf("%s: ObjectMarkerRects(%d,%d,%d,%d,%d) = %v, want the FR-6 cross %v",
				tc.name, gotCol, gotRow, cols, rows, cellpx, got, want)
		}
	}

	spot := []image.Rectangle{
		image.Rect(42, 79, 55, 82),
		image.Rect(47, 74, 50, 87),
	}
	if oracle := specCross(1, 2, cols, rows, cellpx); !sameRects(oracle, spot) {
		t.Fatalf("derivation drift: oracle gave %v for cell (1,2) at cellpx=32, hand-computed FR-6 says %v",
			oracle, spot)
	}
	if got := terrain.ObjectMarkerRects(1, 2, cols, rows, cellpx); !sameRects(got, spot) {
		t.Errorf("cell (1,2) at cellpx=32: got %v, want the hand-computed FR-6 cross %v", got, spot)
	}
}

func TestObjectMarkerRectsGeometry(t *testing.T) {
	cases := []struct {
		name       string
		col, row   int
		cols, rows int
		cellpx     int
		truncates  bool
		want       []image.Rectangle
	}{
		// ---- cellpx = 32 (native): r = round(192/32) = floor(208/32) = 6,
		//      t = round(96/32) = floor(112/32) = 3, lo = 1, half = 16.
		//      This is the spec's stated native pair [cx-6,cx+7)x[cy-1,cy+2),
		//      [cx-1,cx+2)x[cy-6,cy+7). Map is 4x3 -> [0,128)x[0,96).
		{
			// cx = 0*32+16 = 16, cy = 16
			"cellpx=32 cell(0,0)", 0, 0, 4, 3, 32, false,
			[]image.Rectangle{image.Rect(10, 15, 23, 18), image.Rect(15, 10, 18, 23)},
		},
		{
			// cx = 2*32+16 = 80, cy = 1*32+16 = 48
			"cellpx=32 cell(2,1)", 2, 1, 4, 3, 32, false,
			[]image.Rectangle{image.Rect(74, 47, 87, 50), image.Rect(79, 42, 82, 55)},
		},
		{
			// cx = 3*32+16 = 112, cy = 2*32+16 = 80 (the far corner cell)
			"cellpx=32 cell(3,2)", 3, 2, 4, 3, 32, false,
			[]image.Rectangle{image.Rect(106, 79, 119, 82), image.Rect(111, 74, 114, 87)},
		},

		// ---- cellpx = 16 (even): r = floor((96+16)/32) = 3,
		//      t = floor((48+16)/32) = 2, lo = 1, half = 8. Map 4x3 -> [0,64)x[0,48).
		{
			// cx = 8, cy = 8; H = [5,12)x[7,9), V = [7,9)x[5,12)
			"cellpx=16 cell(0,0)", 0, 0, 4, 3, 16, false,
			[]image.Rectangle{image.Rect(5, 7, 12, 9), image.Rect(7, 5, 9, 12)},
		},
		{
			// cx = 2*16+8 = 40, cy = 1*16+8 = 24
			"cellpx=16 cell(2,1)", 2, 1, 4, 3, 16, false,
			[]image.Rectangle{image.Rect(37, 23, 44, 25), image.Rect(39, 21, 41, 28)},
		},
		{
			// cx = 3*16+8 = 56, cy = 2*16+8 = 40
			"cellpx=16 cell(3,2)", 3, 2, 4, 3, 16, false,
			[]image.Rectangle{image.Rect(53, 39, 60, 41), image.Rect(55, 37, 57, 44)},
		},

		// ---- cellpx = 17 (odd, EVEN thickness): r = floor((102+16)/32) = 3,
		//      t = floor((51+16)/32) = floor(67/32) = 2, lo = 1, half = 8.
		//      t=2 is the R-3 low-side-biased strip; pinned explicitly below.
		//      Map 4x3 -> [0,68)x[0,51).
		{
			// cx = 8, cy = 8
			"cellpx=17 cell(0,0)", 0, 0, 4, 3, 17, false,
			[]image.Rectangle{image.Rect(5, 7, 12, 9), image.Rect(7, 5, 9, 12)},
		},
		{
			// cx = 2*17+8 = 42, cy = 1*17+8 = 25
			"cellpx=17 cell(2,1)", 2, 1, 4, 3, 17, false,
			[]image.Rectangle{image.Rect(39, 24, 46, 26), image.Rect(41, 22, 43, 29)},
		},
		{
			// cx = 3*17+8 = 59, cy = 2*17+8 = 42
			"cellpx=17 cell(3,2)", 3, 2, 4, 3, 17, false,
			[]image.Rectangle{image.Rect(56, 41, 63, 43), image.Rect(58, 39, 60, 46)},
		},

		// ---- cellpx = 15 (odd, thickness 1): r = floor((90+16)/32) = 3,
		//      t = floor((45+16)/32) = floor(61/32) = 1, lo = 0, half = 7.
		//      Map 4x3 -> [0,60)x[0,45).
		{
			// cx = 7, cy = 7; H = [4,11)x[7,8), V = [7,8)x[4,11)
			"cellpx=15 cell(0,0)", 0, 0, 4, 3, 15, false,
			[]image.Rectangle{image.Rect(4, 7, 11, 8), image.Rect(7, 4, 8, 11)},
		},
		{
			// cx = 2*15+7 = 37, cy = 1*15+7 = 22
			"cellpx=15 cell(2,1)", 2, 1, 4, 3, 15, false,
			[]image.Rectangle{image.Rect(34, 22, 41, 23), image.Rect(37, 19, 38, 26)},
		},
		{
			// cx = 3*15+7 = 52, cy = 2*15+7 = 37
			"cellpx=15 cell(3,2)", 3, 2, 4, 3, 15, false,
			[]image.Rectangle{image.Rect(49, 37, 56, 38), image.Rect(52, 34, 53, 41)},
		},

		// ---- Degenerate cellpx <= 2: the only reachable place the MAP-rect clip
		//      truncates (plan DD4 honesty note). r and t both floor to 0 and are
		//      raised to 1 by the max(1,...) guard.
		{
			// 1x1 map at cellpx=1: half=0 -> cx=cy=0; r=1, t=1, lo=0.
			// Raw H = [-1,2)x[0,1), raw V = [0,1)x[-1,2); the map is [0,1)x[0,1),
			// so BOTH arms overrun on all four sides and both truncate to the
			// single map pixel. Two arms survive (neither is empty).
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
		{
			// 2x2 map at cellpx=1, corner cell (0,0): cx=cy=0.
			// Raw H = [-1,2)x[0,1) truncates on the LOW side only -> [0,2)x[0,1);
			// raw V = [0,1)x[-1,2) truncates on the LOW side only -> [0,1)x[0,2).
			"cellpx=1 low-side clip, 2x2 map cell(0,0)", 0, 0, 2, 2, 1, true,
			[]image.Rectangle{image.Rect(0, 0, 2, 1), image.Rect(0, 0, 1, 2)},
		},
		{
			// 2x2 map at cellpx=1, far cell (1,1): cx=cy=1.
			// Raw H = [0,3)x[1,2) truncates on the HIGH side only -> [0,2)x[1,2);
			// raw V = [1,2)x[0,3) truncates on the HIGH side only -> [1,2)x[0,2).
			"cellpx=1 high-side clip, 2x2 map cell(1,1)", 1, 1, 2, 2, 1, true,
			[]image.Rectangle{image.Rect(0, 1, 2, 2), image.Rect(1, 0, 2, 2)},
		},
	}

	for _, tc := range cases {
		if oracle := specCross(tc.col, tc.row, tc.cols, tc.rows, tc.cellpx); !sameRects(oracle, tc.want) {
			t.Fatalf("%s: derivation drift: FR-6 oracle says %v, the hand-computed table says %v",
				tc.name, oracle, tc.want)
		}

		var got []image.Rectangle
		noPanicOverlay(t, tc.name, func() {
			got = terrain.ObjectMarkerRects(tc.col, tc.row, tc.cols, tc.rows, tc.cellpx)
		})
		if !sameRects(got, tc.want) {
			t.Errorf("%s: ObjectMarkerRects(%d,%d,%d,%d,%d) = %v, want the FR-6 rectangles %v (AC-2)",
				tc.name, tc.col, tc.row, tc.cols, tc.rows, tc.cellpx, got, tc.want)
			continue
		}

		raw := specArmsRaw(tc.col, tc.row, tc.cellpx)
		clip := specMapRect(tc.cols, tc.rows, tc.cellpx)

		if !tc.truncates {
			// Fixture guard for the DD4 honesty note: at these scales the whole
			// cross fits strictly inside its own cell, so the map-rect clip must
			// have changed nothing at all.
			for i, arm := range raw {
				if got[i] != arm {
					t.Errorf("%s: arm %d = %v but the unclipped FR-6 arm is %v; this scale's cross fits inside its cell, so the map clip must be a no-op",
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

	const bx, by = 42, 25
	arms := terrain.ObjectMarkerRects(2, 1, 4, 3, 17)
	if len(arms) != 2 {
		t.Fatalf("cellpx=17 cell(2,1): got %d arms, want 2", len(arms))
	}
	if arms[0].Min.Y != by-1 || arms[0].Max.Y != by+1 {
		t.Errorf("R-3: horizontal arm at cellpx=17 spans y [%d,%d), want [%d,%d) - an even t=2 strip is centred low, covering cy-1 and cy but not cy+1",
			arms[0].Min.Y, arms[0].Max.Y, by-1, by+1)
	}
	if arms[1].Min.X != bx-1 || arms[1].Max.X != bx+1 {
		t.Errorf("R-3: vertical arm at cellpx=17 spans x [%d,%d), want [%d,%d) - an even t=2 strip is centred low, covering cx-1 and cx but not cx+1",
			arms[1].Min.X, arms[1].Max.X, bx-1, bx+1)
	}
}

func TestObjectMarkerRectsExtremes(t *testing.T) {
	for _, cellpx := range []int{0, -1, -32, -(1 << 30)} {
		var got []image.Rectangle
		noPanicOverlay(t, "ObjectMarkerRects with an invalid cellpx", func() {
			got = terrain.ObjectMarkerRects(1, 1, 4, 4, cellpx)
		})
		if got != nil {
			t.Errorf("cellpx = %d is invalid and must yield no geometry (FR-2), got %v", cellpx, got)
		}
	}

	// (b) AC-3: off-map extreme magnitudes -> no geometry, no panic. 0xFFFFFF is
	//     the widest cell AnchorCell can produce (0xFFFFFFFF>>8).
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
		noPanicOverlay(t, "ObjectMarkerRects off-map "+tc.name, func() {
			got = terrain.ObjectMarkerRects(tc.col, tc.row, tc.cols, tc.rows, tc.cellpx)
		})
		if got != nil {
			t.Errorf("%s: anchor (%d,%d) is outside the %dx%d map and must contribute nothing (AC-1, FR-3), got %v",
				tc.name, tc.col, tc.row, tc.cols, tc.rows, got)
		}
		if n := testing.AllocsPerRun(100, func() {
			sinkRects = terrain.ObjectMarkerRects(tc.col, tc.row, tc.cols, tc.rows, tc.cellpx)
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
	noPanicOverlay(t, "ObjectMarkerRects at extreme in-map magnitude", func() {
		big = terrain.ObjectMarkerRects(bigCol, bigRow, bigCols, bigRows, bigCellpx)
	})
	if len(big) > 2 {
		t.Errorf("extreme in-map anchor: got %d rectangles, want at most 2 (P-1)", len(big))
	}
	// Derivation: half = 2048, cx = cy = (2^24-1)*4096 + 2048 = 68719474688; r
	// = floor((6*4096+16)/32) = floor(24592/32) = 768; t =
	// floor((3*4096+16)/32) = floor(12304/32) = 384; lo = 192.
	wantBig := specCross(bigCol, bigRow, bigCols, bigRows, bigCellpx)
	if len(wantBig) != 2 || wantBig[0] != image.Rect(68719473920, 68719474496, 68719475457, 68719474880) {
		t.Fatalf("derivation drift: FR-6 oracle gave %v for the extreme in-map anchor", wantBig)
	}
	if !sameRects(big, wantBig) {
		t.Errorf("extreme in-map anchor: got %v, want the FR-6 cross %v (AC-3: arithmetic clipping, no wrap)",
			big, wantBig)
	}

	const runs = 200
	small := testing.AllocsPerRun(runs, func() {
		sinkRects = terrain.ObjectMarkerRects(1, 1, 4, 4, 32)
	})
	huge := testing.AllocsPerRun(runs, func() {
		sinkRects = terrain.ObjectMarkerRects(bigCol, bigRow, bigCols, bigRows, bigCellpx)
	})
	// Logged so the comparison below is visibly not two zeros: a fixed non-zero
	// count is fine, magnitude-dependence is not.
	t.Logf("allocations per call: small-magnitude anchor %.0f, huge-magnitude anchor %.0f", small, huge)
	if small != huge {
		t.Errorf("P-2: allocations must not depend on coordinate magnitude: cell (1,1) in a 4x4 map at cellpx=32 allocates %.0f, cell (%d,%d) in a %dx%d map at cellpx=%d allocates %.0f",
			small, bigCol, bigRow, bigCols, bigRows, bigCellpx, huge)
	}
}

// markerBG is the synthetic background the draw tests paint first; it must
// differ from MarkerColor so every marked pixel is unambiguous.
var markerBG = color.RGBA{0x11, 0x22, 0x33, 0xFF}

// drawAndCheck paints markerBG over an imgW x imgH RGBA anchored at the origin
// (DD5's precondition), draws the cells, and then asserts that EXACTLY the
// spec-derived pixels hold MarkerColor and every other pixel still holds the
// background. It returns the marked-pixel count and the drawn image.
func drawAndCheck(t *testing.T, what string, imgW, imgH int, cells []image.Point, cols, rows, cellpx int) (int, *image.RGBA) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	for y := 0; y < imgH; y++ {
		for x := 0; x < imgW; x++ {
			img.SetRGBA(x, y, markerBG)
		}
	}

	want := make([]bool, imgW*imgH)
	for _, c := range cells {
		for _, arm := range specCross(c.X, c.Y, cols, rows, cellpx) {
			cl := arm.Intersect(img.Bounds())
			for y := cl.Min.Y; y < cl.Max.Y; y++ {
				for x := cl.Min.X; x < cl.Max.X; x++ {
					want[y*imgW+x] = true
				}
			}
		}
	}

	noPanicOverlay(t, what+": DrawObjectMarkers", func() {
		terrain.DrawObjectMarkers(img, cells, cols, rows, cellpx)
	})

	marked := 0
	for y := 0; y < imgH; y++ {
		for x := 0; x < imgW; x++ {
			got := img.RGBAAt(x, y)
			if want[y*imgW+x] {
				marked++
				if got != terrain.MarkerColor {
					t.Fatalf("%s: pixel (%d,%d) = %v, want MarkerColor %v - it lies inside an FR-6 arm clipped to the map rect and the image bounds (SC-4, AC-2)",
						what, x, y, got, terrain.MarkerColor)
				}
				continue
			}
			if got != markerBG {
				t.Fatalf("%s: pixel (%d,%d) = %v, want the untouched background %v - it lies outside every clipped FR-6 arm, and the overlay must touch nothing else (SC-4, P-1/P-3)",
					what, x, y, got, markerBG)
			}
		}
	}
	return marked, img
}

func TestDrawObjectMarkers(t *testing.T) {
	if want := (color.RGBA{R: 0xFF, G: 0xD0, B: 0x00, A: 0xFF}); terrain.MarkerColor != want {
		t.Fatalf("FR-6: MarkerColor = %v, want the opaque yellow #FFD000 %v", terrain.MarkerColor, want)
	}
	if terrain.MarkerColor == markerBG {
		t.Fatal("fixture: the background must differ from MarkerColor")
	}

	// A native cross covers |H| + |V| - |H and V| = 13*3 + 3*13 - 3*3 = 69 pixels.
	const nativeCross = 69

	const cols, rows, cellpx = 5, 4, 32
	mapW, mapH := cols*cellpx, rows*cellpx // 160 x 128

	inMap := []image.Point{{X: 0, Y: 0}, {X: 2, Y: 1}, {X: 4, Y: 3}}
	offMap := []image.Point{{X: cols, Y: 0}, {X: 0, Y: rows}, {X: 99, Y: 99}, {X: -1, Y: -1}}

	t.Run("map extent", func(t *testing.T) {
		cells := append(append([]image.Point(nil), inMap...), offMap...)
		marked, _ := drawAndCheck(t, "map extent", mapW, mapH, cells, cols, rows, cellpx)
		if want := len(inMap) * nativeCross; marked != want {
			t.Errorf("marked %d pixels, want %d = %d in-map crosses x %d pixels (13x3 arm + 3x13 arm - 3x3 overlap); the off-map cells must contribute nothing (FR-3)",
				marked, want, len(inMap), nativeCross)
		}
	})

	t.Run("off-map cells only", func(t *testing.T) {
		marked, _ := drawAndCheck(t, "off-map only", mapW, mapH, offMap, cols, rows, cellpx)
		if marked != 0 {
			t.Errorf("marked %d pixels, want 0: every anchor in this list is off-map (AC-1, FR-3)", marked)
		}
	})

	t.Run("nil cells", func(t *testing.T) {
		marked, _ := drawAndCheck(t, "nil cells", mapW, mapH, nil, cols, rows, cellpx)
		if marked != 0 {
			t.Errorf("marked %d pixels, want 0 for an empty cell list (P-3)", marked)
		}
	})

	t.Run("undersized image at the origin", func(t *testing.T) {
		const imgW, imgH = 80, 50
		cells := []image.Point{{X: 0, Y: 0}, {X: 2, Y: 1}, {X: 4, Y: 3}, {X: 7, Y: 0}}
		marked, img := drawAndCheck(t, "undersized image", imgW, imgH, cells, cols, rows, cellpx)
		if want := nativeCross + 23; marked != want {
			t.Errorf("marked %d pixels, want %d = 69 (the whole cross at cell (0,0)) + 23 (cell (2,1) truncated to the image bounds); cell (4,3) and the off-map cell draw nothing",
				marked, want)
		}

		for y := 47; y < 50; y++ {
			if got := img.RGBAAt(imgW-1, y); got != terrain.MarkerColor {
				t.Errorf("pixel (%d,%d) = %v, want MarkerColor: the arm [74,87) is truncated to the image edge, not inset from it (FR-3: no synthetic border)",
					imgW-1, y, got)
			}
		}
		// The vertical arm of cell (2,1) is truncated at the bottom edge the
		// same way: its last in-bounds row must be drawn.
		if got := img.RGBAAt(79, imgH-1); got != terrain.MarkerColor {
			t.Errorf("pixel (79,%d) = %v, want MarkerColor: the vertical arm [42,55) is truncated to the image edge, not inset from it (FR-3)",
				imgH-1, got)
		}
		// And the columns just outside the clipped arm inside the image stay
		// background, so the clip did not smear the marker along the boundary.
		for y := 47; y < 50; y++ {
			if got := img.RGBAAt(73, y); got != markerBG {
				t.Errorf("pixel (73,%d) = %v, want the background %v: the arm starts at x=74 (FR-6)", y, got, markerBG)
			}
		}
	})
}

// The native glyph dimensions the two overlays are defined at, named here so
// one translated oracle can serve both kinds.
const (
	specObjectRadius, specObjectThickness = 6, 3
	specUnitRadius, specUnitThickness     = 4, 1
)

// specArmsRawAt returns the two UNCLIPPED arms (horizontal, then vertical) for
// anchor cell (col,row) at cellpx on a canvas translated by offsetY, for a glyph
// of the given native radius and thickness. Straight transcription of the formula
// above; it calls nothing from the package under test, and at offsetY = 0 it
// reduces to specArmsRaw (6,3) and specUnitArmsRaw (4,1).
func specArmsRawAt(col, row, cellpx, offsetY, radius, thickness int) [2]image.Rectangle {
	half := cellpx / 2
	cx := col*cellpx + half
	cy := row*cellpx + half + offsetY
	r := max(1, specRound(radius*cellpx))
	t := max(1, specRound(thickness*cellpx))
	lo := t / 2
	return [2]image.Rectangle{
		image.Rect(cx-r, cy-lo, cx+r+1, cy-lo+t), // horizontal
		image.Rect(cx-lo, cy-r, cx-lo+t, cy+r+1), // vertical
	}
}

// specMapRectAt is the map pixel rectangle the stage-1 clip uses on a translated
// canvas: specMapRect moved by the same offsetY as the glyph.
func specMapRectAt(cols, rows, cellpx, offsetY int) image.Rectangle {
	return image.Rect(0, offsetY, cols*cellpx, rows*cellpx+offsetY)
}

// specCrossAt is the independent oracle for the *At entry points: the translated
// arms, each intersected with the translated map rect and dropped when empty, or
// nil for an off-map anchor / an invalid scale. The off-map test is on the cell
// indices, so offsetY cannot admit or drop an anchor.
func specCrossAt(col, row, cols, rows, cellpx, offsetY, radius, thickness int) []image.Rectangle {
	if cellpx < 1 || col < 0 || row < 0 || col >= cols || row >= rows {
		return nil
	}
	clip := specMapRectAt(cols, rows, cellpx, offsetY)
	out := make([]image.Rectangle, 0, 2)
	for _, arm := range specArmsRawAt(col, row, cellpx, offsetY, radius, thickness) {
		if c := arm.Intersect(clip); !c.Empty() {
			out = append(out, c)
		}
	}
	return out
}

// backgroundImage is an imgW x imgH RGBA anchored at the origin (the overlays'
// precondition) painted markerBG, so every marked pixel below is unambiguous.
func backgroundImage(imgW, imgH int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	for y := 0; y < imgH; y++ {
		for x := 0; x < imgW; x++ {
			img.SetRGBA(x, y, markerBG)
		}
	}
	return img
}

// drawBothAt paints the background, draws the object overlay and then the unit
// overlay at offsetY, and asserts that EXACTLY the spec-derived pixels hold each
// marker colour and every other pixel still holds the background. It returns the
// marked-pixel count and the drawn image.
//
// The oracle is specCrossAt for each glyph, clipped to img.Bounds() (the stage-2
// clip, which does not move) — never a call into the package's own rect builders.
// It is painted in the same order the overlays are drawn, objects then units, so
// the terrain -> objects -> units order is expressed as an order rather than as a
// precedence rule, and every shared pixel is asserted cyan rather than
// spot-checked.
func drawBothAt(t *testing.T, what string, imgW, imgH int, cells []image.Point, cols, rows, cellpx, offsetY int) (int, *image.RGBA) {
	t.Helper()

	img := backgroundImage(imgW, imgH)

	want := make([]color.RGBA, imgW*imgH)
	for i := range want {
		want[i] = markerBG
	}
	paint := func(radius, thickness int, c color.RGBA) {
		for _, cell := range cells {
			for _, arm := range specCrossAt(cell.X, cell.Y, cols, rows, cellpx, offsetY, radius, thickness) {
				cl := arm.Intersect(img.Bounds())
				for y := cl.Min.Y; y < cl.Max.Y; y++ {
					for x := cl.Min.X; x < cl.Max.X; x++ {
						want[y*imgW+x] = c
					}
				}
			}
		}
	}
	paint(specObjectRadius, specObjectThickness, terrain.MarkerColor)
	paint(specUnitRadius, specUnitThickness, terrain.UnitMarkerColor)

	noPanicOverlay(t, what+": DrawObjectMarkersAt", func() {
		terrain.DrawObjectMarkersAt(img, cells, cols, rows, cellpx, offsetY)
	})
	noPanicOverlay(t, what+": DrawUnitMarkersAt", func() {
		terrain.DrawUnitMarkersAt(img, cells, cols, rows, cellpx, offsetY)
	})

	marked := 0
	for y := 0; y < imgH; y++ {
		for x := 0; x < imgW; x++ {
			w := want[y*imgW+x]
			if w != markerBG {
				marked++
			}
			if got := img.RGBAAt(x, y); got != w {
				t.Fatalf("%s (offsetY %d): pixel (%d,%d) = %v, want %v — the translated FR-6 arms clipped to the moved map rect and then to the image bounds, units over objects (0012 FR-11, SC-13)",
					what, offsetY, x, y, got, w)
			}
		}
	}
	return marked, img
}

func TestMarkersAtOrigin(t *testing.T) {
	const cols, rows = 5, 4

	// (a) Derivation-drift guard: at offsetY = 0 the translated oracle must
	//     reproduce the frozen 0008 and 0009 oracles exactly. Without this, what
	//     follows could be testing a second geometry that merely agrees with
	//     itself.
	for _, c := range []image.Point{{X: 0, Y: 0}, {X: 2, Y: 1}, {X: 4, Y: 3}} {
		for _, cellpx := range []int{32, 64} {
			if got, want := specCrossAt(c.X, c.Y, cols, rows, cellpx, 0, specObjectRadius, specObjectThickness),
				specCross(c.X, c.Y, cols, rows, cellpx); !sameRects(got, want) {
				t.Fatalf("oracle drift at cell (%d,%d), cellpx=%d: the offset-0 object cross is %v, the frozen 0008 oracle says %v",
					c.X, c.Y, cellpx, got, want)
			}
			if got, want := specCrossAt(c.X, c.Y, cols, rows, cellpx, 0, specUnitRadius, specUnitThickness),
				specUnitCross(c.X, c.Y, cols, rows, cellpx); !sameRects(got, want) {
				t.Fatalf("oracle drift at cell (%d,%d), cellpx=%d: the offset-0 unit cross is %v, the frozen 0009 oracle says %v",
					c.X, c.Y, cellpx, got, want)
			}
		}
	}

	inMap := []image.Point{{X: 0, Y: 0}, {X: 2, Y: 1}, {X: 4, Y: 3}}
	// Two of these are chosen so that the TRANSLATED position of an off-map
	// anchor lands inside the canvas: (1,-1) at a positive offset and (1,rows) at
	// a negative one. A pixel-space rejection would draw them; the cell-index
	// test does not.
	offMap := []image.Point{{X: cols, Y: 0}, {X: 1, Y: -1}, {X: 1, Y: rows}, {X: 99, Y: 99}, {X: -1, Y: -1}}
	cells := append(append([]image.Point(nil), inMap...), offMap...)

	t.Run("the existing entry points are offsetY 0", func(t *testing.T) {
		const cellpx = 32
		w, h := cols*cellpx, rows*cellpx

		viaOld := backgroundImage(w, h)
		terrain.DrawObjectMarkers(viaOld, cells, cols, rows, cellpx)
		terrain.DrawUnitMarkers(viaOld, cells, cols, rows, cellpx)

		viaAt := backgroundImage(w, h)
		terrain.DrawObjectMarkersAt(viaAt, cells, cols, rows, cellpx, 0)
		terrain.DrawUnitMarkersAt(viaAt, cells, cols, rows, cellpx, 0)

		for i := range viaOld.Pix {
			if viaOld.Pix[i] != viaAt.Pix[i] {
				t.Fatalf("DD-8: DrawObjectMarkers/DrawUnitMarkers must be the offsetY = 0 case of the *At entry points, but the two images differ at Pix[%d] (%d vs %d)",
					i, viaOld.Pix[i], viaAt.Pix[i])
			}
		}
	})

	canvases := []struct {
		name         string
		cellpx       int
		offsetY      int
		imgH         int
		fullyVisible bool
	}{
		{
			// OriginY = -40 at scale 1: the canvas begins 40 native rows ABOVE
			// the lattice's row 0, every cross moves DOWN by 40, and a canvas 40
			// rows taller holds all of them.
			"native scale, offsetY +40 (OriginY -40, the ordinary case)", 32, +40, rows*32 + 40, true,
		},
		{
			// OriginY = +24 at scale 1: the canvas begins 24 native rows BELOW
			// the lattice's row 0, so cell row 0's crosses (native y 10..22) are
			// carried above output row 0 and the img.Bounds() clip drops them.
			// The canvas is 12 rows longer than the lattice's translated bottom,
			// which is where the off-map (1,rows) anchor's glyph would land.
			"native scale, offsetY -24 (OriginY +24)", 32, -24, rows*32 + 12, false,
		},
		{
			// offsetY is in OUTPUT pixels, not native rows: at scale 2 a MinV of
			// -20 gives -OriginY*scale = +40, an offset the cell size does not
			// divide, so a lattice built in cell units cannot fake it.
			"scale 2, offsetY +40 (OriginY -20 at scale 2)", 64, +40, rows*64 + 40, true,
		},
	}

	for _, cv := range canvases {
		refW, refH := cols*cv.cellpx, rows*cv.cellpx
		refMarked, ref := drawBothAt(t, cv.name+", reference at offset 0", refW, refH, cells, cols, rows, cv.cellpx, 0)
		if refMarked == 0 {
			t.Fatalf("%s: the offset-0 reference marked no pixel, so the case can say nothing", cv.name)
		}

		// Fixture guard: at least one off-map anchor must land its unclipped
		// glyph inside the translated canvas, or "an off-map anchor draws
		// nothing" would hold for some reason other than the cell-index test.
		canvas := image.Rect(0, 0, refW, cv.imgH)
		reachable := 0
		for _, c := range offMap {
			for _, arm := range specArmsRawAt(c.X, c.Y, cv.cellpx, cv.offsetY, specObjectRadius, specObjectThickness) {
				if !arm.Intersect(canvas).Empty() {
					reachable++
				}
			}
		}
		if reachable == 0 {
			t.Fatalf("%s: fixture: no off-map anchor's glyph reaches the canvas %v at offsetY %d, so the off-map drop would pass vacuously",
				cv.name, canvas, cv.offsetY)
		}

		marked, got := drawBothAt(t, cv.name, refW, cv.imgH, cells, cols, rows, cv.cellpx, cv.offsetY)

		for y := 0; y < cv.imgH; y++ {
			for x := 0; x < refW; x++ {
				want := markerBG
				if sy := y - cv.offsetY; sy >= 0 && sy < refH {
					want = ref.RGBAAt(x, sy)
				}
				if g := got.RGBAAt(x, y); g != want {
					t.Fatalf("%s: pixel (%d,%d) = %v, want %v — the pixel at (%d,%d) of the same overlay at offset 0; the lattice must be translated by offsetY = %d and by nothing else (FR-11, SC-13)",
						cv.name, x, y, g, want, x, y-cv.offsetY, cv.offsetY)
				}
			}
		}

		switch {
		case cv.fullyVisible && marked != refMarked:
			t.Errorf("%s: marked %d pixels, want the offset-0 count %d — this canvas is tall enough to hold the whole translated lattice, so the translation must lose nothing",
				cv.name, marked, refMarked)
		case !cv.fullyVisible && marked >= refMarked:
			t.Errorf("%s: marked %d pixels, want fewer than the offset-0 count %d — a positive OriginY carries cell row 0's crosses above output row 0, where the img.Bounds() clip must drop them",
				cv.name, marked, refMarked)
		}
	}

	// (c) The composition order at a non-zero origin. Every drawBothAt call above
	//     already EXPECTS cyan on each shared pixel and yellow on the object-only
	//     ones, at each offset, so the order is asserted rather than spot-checked;
	//     what is missing is the control that makes that assertion non-vacuous.
	//     The unit cross is a strict subset of the object cross when both are
	//     drawn at the SAME offset, so a reversed order does not dim the unit
	//     marker, it erases it — and "no cyan at all" is what a
	//     wrong order looks like.
	t.Run("composition order at a non-zero origin", func(t *testing.T) {
		const cellpx, offsetY = 32, 40
		w, h := cols*cellpx, rows*cellpx+offsetY
		one := []image.Point{{X: 2, Y: 1}}

		correct, _ := drawBothAt(t, "objects then units at an origin", w, h, one, cols, rows, cellpx, offsetY)
		if correct == 0 {
			t.Fatalf("fixture: cell %v marked nothing at offsetY %d", one[0], offsetY)
		}

		// The control. Drawn units-then-objects, the result must be byte-identical
		// to the OBJECT overlay alone: the object cross covers every pixel the
		// unit cross does, so a reversed order does not dim the unit marker, it
		// erases it. Stating it as an image comparison says both halves at once
		// — the crosses still nest at a non-zero origin, which needs the two
		// overlays to take the SAME offset, and a wrong order leaves no cyan at
		// all rather than a partly covered marker.
		reversed := backgroundImage(w, h)
		noPanicOverlay(t, "units then objects at an origin", func() {
			terrain.DrawUnitMarkersAt(reversed, one, cols, rows, cellpx, offsetY)
			terrain.DrawObjectMarkersAt(reversed, one, cols, rows, cellpx, offsetY)
		})
		objectsOnly := backgroundImage(w, h)
		terrain.DrawObjectMarkersAt(objectsOnly, one, cols, rows, cellpx, offsetY)
		for i := range objectsOnly.Pix {
			if reversed.Pix[i] != objectsOnly.Pix[i] {
				t.Fatalf("at offsetY %d the units-then-objects order differs from the object overlay alone at Pix[%d] (%d vs %d): at one shared offset the unit cross is a strict subset of the object cross, so the reversed order must erase the unit marker completely — the control that makes the order assertion above meaningful (0009 FR-4, R-4)",
					offsetY, i, reversed.Pix[i], objectsOnly.Pix[i])
			}
		}
	})
}
