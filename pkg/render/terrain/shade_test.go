package terrain_test

// Tests for the shading-transform / interpolation API of pkg/render/terrain.
//
// Every expected value below is DERIVED FROM THE SPEC FORMULA
// (docs/0007-terrain-lighting/spec.md, TERR-LIGHT-019). Per channel, in integers:
//
//	out = clamp( ((chan + tint) * (96 − level)) / 32, 0, 255 )   (truncate toward zero)
//
// Multiplier (96−level)/32: level 0 -> x3.0, level 64 -> x1.0 (identity),
// level 95 -> x1/32. Higher level = darker. Tint is added BEFORE the multiply.
// InterpRow is truncated bilinear interpolation of a cell's four corner levels.
//
// This file tests ONLY the shading / interpolation API. It never references the
// light / level-grid functions (LevelGrid / LightFromFields / Light).

import (
	"image/color"
	"testing"

	terrain "againrom/pkg/render/terrain"
)

// clampByte clamps an integer channel result to the display byte range [0,255].
func clampByte(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// noPanicShade runs fn and fails the test if it panics.
func noPanicShade(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("%s panicked: %v", name, rec)
		}
	}()
	fn()
}

// TestInterpRow covers AC-6: truncated bilinear interpolation of the four corner
// levels — exact at the four corners, the truncated blend inside, and constant
// when the corners are equal. corners: l00=top-left, l10=top-right,
// l01=bottom-left, l11=bottom-right; fx=x/cell, fy=y/cell.
//
// The bilinear value is
//
//	v = l00·(1−fx)(1−fy) + l10·fx(1−fy) + l01·(1−fx)fy + l11·fx·fy
//
// truncated toward zero. With corners 10,20,30,40 and cell=32:
//
//	(0,0)   -> 10          (top-left corner)
//	(32,0)  -> 20          (top-right corner)
//	(0,32)  -> 30          (bottom-left corner)
//	(32,32) -> 40          (bottom-right corner)
//	(16,16) -> (10+20+30+40)/4 = 25                          (exact)
//	(16,8)  -> .375·(10+20) + .125·(30+40) = 11.25+8.75 = 20 (exact)
//	(8,0)   -> 10·.75 + 20·.25 = 12.5 -> trunc 12            (top edge, fractional fx)
//	(24,0)  -> 10·.25 + 20·.75 = 17.5 -> trunc 17
//	equal corners (all 50) -> 50 everywhere
func TestInterpRow(t *testing.T) {
	cell := terrain.CellSize // 32
	cases := []struct {
		name               string
		l00, l10, l01, l11 uint8
		x, y               int
		want               int
	}{
		{"top-left corner", 10, 20, 30, 40, 0, 0, 10},
		{"top-right corner", 10, 20, 30, 40, cell, 0, 20},
		{"bottom-left corner", 10, 20, 30, 40, 0, cell, 30},
		{"bottom-right corner", 10, 20, 30, 40, cell, cell, 40},
		{"center exact", 10, 20, 30, 40, cell / 2, cell / 2, 25},
		{"fx.5 fy.25 exact", 10, 20, 30, 40, cell / 2, cell / 4, 20},
		{"top edge fx.25 trunc", 10, 20, 30, 40, cell / 4, 0, 12},
		{"top edge fx.75 trunc", 10, 20, 30, 40, 3 * cell / 4, 0, 17},
		{"equal corners interior", 50, 50, 50, 50, 7, 13, 50},
		{"equal corners at corner", 50, 50, 50, 50, cell, cell, 50},
	}
	for _, c := range cases {
		got := terrain.InterpRow(c.l00, c.l10, c.l01, c.l11, c.x, c.y, cell)
		if got != c.want {
			t.Errorf("%s: InterpRow(%d,%d,%d,%d, x=%d,y=%d, cell=%d) = %d, want %d",
				c.name, c.l00, c.l10, c.l01, c.l11, c.x, c.y, cell, got, c.want)
		}
	}
}

// TestInterpSpan covers the two-denominator interpolator: InterpSpan is
// InterpRow with x and y given separate denominators, so a level can ramp
// down a drawn span shorter or taller than the 32 rows of a square cell.
//
//	fx = x/spanX, fy = y/spanY
//	v  = trunc( (l00(1−fx) + l10·fx)(1−fy) + (l01(1−fx) + l11·fx)·fy )
//
// (a) With equal denominators it IS InterpRow, over a swept domain — which is
// also what makes the existing TestInterpRow a check on this function.
// (b) With spanY != spanX the vertical fraction is y/spanY, so the same y takes a
// different level on a short span than on a 32-row one: corners 0/0 over 64/64 at
// y = 4 give 32 on an 8-row span and 8 on a 32-row one. Interpolating over the
// wrong domain would collapse that difference.
// (c) A non-positive denominator on either axis returns the top-left corner,
// InterpRow's own degenerate answer.
//
// Expected values are computed from the formula above, not from the
// implementation, and no case sits on an integer boundary where truncation could
// go either way.
func TestInterpSpan(t *testing.T) {
	cell := terrain.CellSize // 32

	// (a) equal denominators reproduce InterpRow exactly.
	for _, corners := range [][4]uint8{{10, 20, 30, 40}, {95, 0, 47, 63}, {50, 50, 50, 50}} {
		for _, span := range []int{1, 7, 8, cell} {
			for y := 0; y <= span; y++ {
				for x := 0; x <= span; x++ {
					want := terrain.InterpRow(corners[0], corners[1], corners[2], corners[3], x, y, span)
					got := terrain.InterpSpan(corners[0], corners[1], corners[2], corners[3], x, y, span, span)
					if got != want {
						t.Fatalf("InterpSpan(%v, x=%d,y=%d, %d,%d) = %d, want InterpRow's %d",
							corners, x, y, span, span, got, want)
					}
				}
			}
		}
	}

	// (b) independent denominators. Corners 10 (TL), 20 (TR), 30 (BL), 40 (BR)
	// over 32 columns and a 7-row span:
	//
	//	(16,3): fx=1/2 -> top=15, bot=35; fy=3/7 -> 15+20·3/7 = 23.571 -> 23
	//	(0,3) : fx=0   -> top=10, bot=30; fy=3/7 -> 10+20·3/7 = 18.571 -> 18
	//	(32,6): fx=1   -> top=20, bot=40; fy=6/7 -> 20+20·6/7 = 37.142 -> 37
	//	(8,1) : fx=1/4 -> top=12.5, bot=32.5; fy=1/7 -> 15.357     -> 15
	for _, c := range []struct {
		name               string
		l00, l10, l01, l11 uint8
		x, y, spanX, spanY int
		want               int
	}{
		{"span 7: top-left corner", 10, 20, 30, 40, 0, 0, cell, 7, 10},
		{"span 7: top-right corner", 10, 20, 30, 40, cell, 0, cell, 7, 20},
		{"span 7: bottom-left corner", 10, 20, 30, 40, 0, 7, cell, 7, 30},
		{"span 7: bottom-right corner", 10, 20, 30, 40, cell, 7, cell, 7, 40},
		{"span 7: mid column, row 3", 10, 20, 30, 40, cell / 2, 3, cell, 7, 23},
		{"span 7: left column, row 3", 10, 20, 30, 40, 0, 3, cell, 7, 18},
		{"span 7: right column, row 6", 10, 20, 30, 40, cell, 6, cell, 7, 37},
		{"span 7: quarter column, row 1", 10, 20, 30, 40, cell / 4, 1, cell, 7, 15},
		{"span 8 at y=4 is the midpoint", 0, 0, 64, 64, 0, 4, cell, 8, 32},
		{"span 32 at y=4 is an eighth", 0, 0, 64, 64, 0, 4, cell, cell, 8},
		{"span 8 at y=7 is 7/8", 0, 0, 64, 64, 17, 7, cell, 8, 56},
		{"span 1: y=0 is the top edge", 10, 20, 30, 40, 0, 0, cell, 1, 10},
		{"narrow x span", 10, 20, 30, 40, 1, 0, 2, 7, 15},
		{"equal corners, any span", 50, 50, 50, 50, 5, 3, cell, 9, 50},
	} {
		got := terrain.InterpSpan(c.l00, c.l10, c.l01, c.l11, c.x, c.y, c.spanX, c.spanY)
		if got != c.want {
			t.Errorf("%s: InterpSpan(%d,%d,%d,%d, x=%d,y=%d, spanX=%d,spanY=%d) = %d, want %d",
				c.name, c.l00, c.l10, c.l01, c.l11, c.x, c.y, c.spanX, c.spanY, got, c.want)
		}
	}

	// The short span and the 32-row one must genuinely disagree, or (b) proves
	// nothing about which domain was used.
	short := terrain.InterpSpan(0, 0, 64, 64, 0, 4, cell, 8)
	tall := terrain.InterpSpan(0, 0, 64, 64, 0, 4, cell, cell)
	if short == tall {
		t.Fatalf("span 8 and span 32 agree at y=4 (%d); the vertical domain is not observed", short)
	}

	// (c) a non-positive denominator on either axis returns the top-left corner.
	for _, c := range []struct{ spanX, spanY int }{{0, 0}, {0, cell}, {cell, 0}, {-1, cell}, {cell, -4}} {
		if got := terrain.InterpSpan(11, 20, 30, 40, 3, 5, c.spanX, c.spanY); got != 11 {
			t.Errorf("InterpSpan spanX=%d spanY=%d = %d, want the top-left level 11",
				c.spanX, c.spanY, got)
		}
	}
}

func TestShadeChannelTransform(t *testing.T) {
	channels := []uint8{0, 1, 50, 100, 128, 200, 255}
	levels := []int{0, 1, 32, 46, 64, 80, 95}
	for _, ch := range channels {
		for _, level := range levels {
			// Oracle: the exact integer spec formula (truncating), tint 0.
			exp := clampByte(int(ch) * (96 - level) / 32)
			got := terrain.ShadeChannel(ch, 0, level)
			if got != exp {
				t.Errorf("ShadeChannel(%d,0,%d) = %d, want %d", ch, level, got, exp)
			}
			// Determinism.
			if again := terrain.ShadeChannel(ch, 0, level); again != got {
				t.Errorf("ShadeChannel(%d,0,%d) non-deterministic: %d then %d", ch, level, got, again)
			}
		}
	}

	// Explicit spot checks straight from the spec.
	spots := []struct {
		ch    uint8
		level int
		want  uint8
	}{
		{200, 64, 200}, // (200·32)/32 = 200 (identity row)
		{255, 64, 255}, // identity
		{100, 0, 255},  // (100·96)/32 = 300 -> clamp 255
		{50, 0, 150},   // (50·96)/32 = 150
		{255, 0, 255},  // 255·3 = 765 -> clamp 255
		{200, 95, 6},   // (200·1)/32 = 6 (200/32 = 6.25)
		{100, 95, 3},   // 100/32 = 3.125 -> 3
		{0, 0, 0},      // zero channel stays zero
	}
	for _, s := range spots {
		if got := terrain.ShadeChannel(s.ch, 0, s.level); got != s.want {
			t.Errorf("spot ShadeChannel(%d,0,%d) = %d, want %d", s.ch, s.level, got, s.want)
		}
	}
}

func TestTintChannelIsShadeChannelAtItsIdentityLevel(t *testing.T) {
	values := []uint8{0, 1, 50, 100, 128, 200, 255}
	for _, ch := range values {
		for _, tint := range values {
			want := terrain.ShadeChannel(ch, tint, 64)
			got := terrain.TintChannel(ch, tint)
			if got != want {
				t.Errorf("TintChannel(%d,%d) = %d, want ShadeChannel(%d,%d,64) = %d", ch, tint, got, ch, tint, want)
			}
		}
	}

	// Spot checks against clamp(ch+tint,0,255) directly, an oracle independent
	// of ShadeChannel rather than a second comparison against it.
	spots := []struct{ ch, tint, want uint8 }{
		{200, 100, 255}, // 300 clamps to 255
		{10, 20, 30},
		{255, 255, 255},
		{0, 0, 0},
		{0, 255, 255},
	}
	for _, s := range spots {
		if got := terrain.TintChannel(s.ch, s.tint); got != s.want {
			t.Errorf("TintChannel(%d,%d) = %d, want %d", s.ch, s.tint, got, s.want)
		}
	}
}

func TestShadeIdentityRow(t *testing.T) {
	colors := []color.RGBA{
		{0, 0, 0, 0},
		{255, 255, 255, 255},
		{10, 128, 240, 55},
		{200, 100, 50, 0},
		{1, 2, 3, 4},
		{63, 127, 191, 200},
	}
	for _, c := range colors {
		got := terrain.ShadeRGBA(c, [3]uint8{0, 0, 0}, 64)
		want := color.RGBA{R: c.R, G: c.G, B: c.B, A: 0xff}
		if got != want {
			t.Errorf("ShadeRGBA(%v, {0,0,0}, 64) = %v, want %v (identity row, A forced 0xff)",
				c, got, want)
		}
	}
}

// TestShadeTintBeforeMultiply covers AC-11: the tint is added per channel BEFORE
// the multiply. Each case is constructed so the tint-after ordering would give a
// different value, so the ordering is genuinely observed.
//
//	before = ((ch+tint)·(96−level))/32          (correct)
//	after  = (ch·(96−level))/32 + tint          (wrong; only equal when multiplier == 1, level 64)
func TestShadeTintBeforeMultiply(t *testing.T) {
	cases := []struct {
		ch, tint uint8
		level    int
	}{
		{10, 20, 0},  // before=(30·96)/32=90 ; after=10·3+20=50
		{50, 30, 0},  // before=(80·96)/32=240; after=50·3+30=180
		{10, 20, 32}, // before=(30·64)/32=60 ; after=(10·64)/32+20=40
	}
	for _, c := range cases {
		before := clampByte((int(c.ch) + int(c.tint)) * (96 - c.level) / 32)
		after := int(clampByte(int(c.ch)*(96-c.level)/32)) + int(c.tint) // wrong ordering
		if int(before) == after {
			t.Fatalf("case ch=%d tint=%d level=%d is not discriminating (before==after==%d)",
				c.ch, c.tint, c.level, before)
		}
		if got := terrain.ShadeChannel(c.ch, c.tint, c.level); got != before {
			t.Errorf("ShadeChannel(%d,%d,%d) = %d, want %d (tint before multiply); after-multiply would be %d",
				c.ch, c.tint, c.level, got, before, after)
		}
	}
}

func TestShadeMonotonic(t *testing.T) {
	combos := []struct{ ch, tint uint8 }{
		{200, 0},
		{100, 50},
		{255, 255},
		{30, 0},
		{0, 0},
	}
	for _, cb := range combos {
		prev := 256 // sentinel above the max byte value
		for level := 0; level <= 95; level++ {
			out := int(terrain.ShadeChannel(cb.ch, cb.tint, level))
			if out < 0 || out > 255 {
				t.Errorf("ch=%d tint=%d level=%d: out=%d outside [0,255]", cb.ch, cb.tint, level, out)
			}
			if out > prev {
				t.Errorf("ch=%d tint=%d: non-monotonic at level %d: %d > %d (prev level)",
					cb.ch, cb.tint, level, out, prev)
			}
			prev = out
		}
	}
}

// TestShadeScaleFormula covers 0014 SC-4: ShadeScale is compared with ==
// against (96-level)/32 at EVERY level in [0,95], not only the four named
// spot values, so a hard-coded table of the interesting four cannot pass
// this; and an out-of-range level clamps into [0,95] first, through the same
// clamp ShadeChannel uses, rather than escaping the formula's range.
func TestShadeScaleFormula(t *testing.T) {
	for level := 0; level <= 95; level++ {
		want := float32(96-level) / 32
		if got := terrain.ShadeScale(level); got != want {
			t.Errorf("ShadeScale(%d) = %v, want %v ((96-%d)/32)", level, got, want, level)
		}
	}

	// The spec's own named spot values, as an additional explicit assertion.
	spots := []struct {
		level int
		want  float32
	}{
		{0, 3.0},
		{46, 1.5625},
		{64, 1.0},
		{95, float32(1) / 32},
	}
	for _, s := range spots {
		if got := terrain.ShadeScale(s.level); got != s.want {
			t.Errorf("ShadeScale(%d) = %v, want %v", s.level, got, s.want)
		}
	}

	// An out-of-range level clamps into [0,95] rather than being computed
	// straight through (96-(-1))/32 = 97/32, which is not a valid multiplier).
	for _, tc := range []struct{ level, clampsTo int }{
		{-1, 0}, {-1000, 0}, {96, 95}, {1000, 95},
	} {
		got := terrain.ShadeScale(tc.level)
		want := terrain.ShadeScale(tc.clampsTo)
		if got != want {
			t.Errorf("ShadeScale(%d) = %v, want the clamped ShadeScale(%d) = %v", tc.level, got, tc.clampsTo, want)
		}
		// Pinned against the formula directly too, so a clamp that agrees with
		// itself but not with (96-level)/32 at the boundary still fails.
		if straight := float32(96-tc.clampsTo) / 32; got != straight {
			t.Errorf("ShadeScale(%d) = %v, want %v (the formula at the clamped level %d)", tc.level, got, straight, tc.clampsTo)
		}
	}
}

// TestCornerLevelsMatchesCompositor covers 0014 SC-5: CornerLevels' read for
// an interior, right-edge, bottom-edge and corner tile agrees with what
// CompositeLit actually painted for that tile. CompositeLit -- the CPU
// compositor -- is invoked in this test rather than its result restated:
// InterpRow is exact at a cell's own top-left pixel (x=0,y=0), so any
// in-grid tile's own top-left pixel is ShadeRGBA(base, tint, level) for THAT
// tile's own vertex, undiluted by interpolation. Every one of a tile's four
// corners, clamped or not, names some in-grid tile's own top-left vertex, so
// reading that tile's top-left pixel back out of the rendered image is a
// black-box check on CornerLevels' clamp, not a restatement of it.
func TestCornerLevelsMatchesCompositor(t *testing.T) {
	ts := terrain.LoadTileset(compositeSource())

	const W, H = 4, 4
	heights := make([]uint8, W*H)
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			heights[y*W+x] = uint8(x*11 + y*23)
		}
	}
	tiles := make([]uint16, W*H)
	for i := range tiles {
		tiles[i] = tileWord(0, 0, 0) // the same land sub-cell everywhere: only
		// the shading can vary a pixel.
	}
	gr := terrain.Grid{Width: W, Height: H, Tiles: tiles}

	lit, err := terrain.CompositeLit(ts, gr, heights, terrain.DefaultDaytime, 1)
	if err != nil {
		t.Fatalf("CompositeLit: %v", err)
	}
	levels := terrain.LevelGrid(heights, W, H, terrain.DefaultDaytime)
	base := rampColor(fillOf(landBase, 0))
	tint := [3]uint8{0, 0, 0}

	cases := []struct {
		name     string
		col, row int
	}{
		{"interior", 1, 1},
		{"right edge", W - 1, 1},
		{"bottom edge", 1, H - 1},
		{"corner", W - 1, H - 1},
	}
	offsets := [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} // TL, TR, BL, BR
	names := [4]string{"TL", "TR", "BL", "BR"}

	seen := map[uint8]bool{}
	for _, c := range cases {
		cl := terrain.CornerLevels(levels, W, H, c.col, c.row)
		for k, off := range offsets {
			c2, r2 := c.col+off[0], c.row+off[1]
			if c2 > W-1 {
				c2 = W - 1
			}
			if r2 > H-1 {
				r2 = H - 1
			}
			pix := lit.Image.RGBAAt(c2*terrain.CellSize, r2*terrain.CellSize)
			want := terrain.ShadeRGBA(base, tint, int(cl[k]))
			if pix != want {
				t.Errorf("%s tile (%d,%d) corner %s: CornerLevels level %d shades to %v, but tile (%d,%d)'s own top-left pixel (the same vertex) is %v",
					c.name, c.col, c.row, names[k], cl[k], want, c2, r2, pix)
			}
			seen[cl[k]] = true
		}
	}
	if len(seen) < 2 {
		t.Fatalf("fixture is degenerate: only %d distinct level(s) seen across every tile's corners", len(seen))
	}
}

// TestCornerLevelsTotal covers 0014 T1's done-when: a grid too short for its
// dimensions, a non-positive dimension, and a tile outside the grid each have a
// defined answer -- a clamped read or the zero value -- and never a panic or an
// out-of-range index.
func TestCornerLevelsTotal(t *testing.T) {
	// A 2x2 grid: (col,row) -> level is (0,0)=10 (0,1)=20 (1,0)=30 (1,1)=40 in
	// row-major levels[r*w+c].
	levels := []uint8{10, 30, 20, 40}

	t.Run("interior reads are unclamped", func(t *testing.T) {
		if got, want := terrain.CornerLevels(levels, 2, 2, 0, 0), ([4]uint8{10, 30, 20, 40}); got != want {
			t.Fatalf("CornerLevels(2x2, 0,0) = %v, want %v", got, want)
		}
	})

	t.Run("grid too short for its dimensions", func(t *testing.T) {
		short := []uint8{1, 2, 3} // one byte short of the 2x2 it claims
		if got, want := terrain.CornerLevels(short, 2, 2, 0, 0), ([4]uint8{}); got != want {
			t.Errorf("CornerLevels over a short grid = %v, want the zero value %v", got, want)
		}
	})

	t.Run("non-positive dimension", func(t *testing.T) {
		for _, tc := range []struct{ w, h int }{{0, 2}, {2, 0}, {-1, 2}, {2, -1}, {0, 0}} {
			if got, want := terrain.CornerLevels(levels, tc.w, tc.h, 0, 0), ([4]uint8{}); got != want {
				t.Errorf("CornerLevels(w=%d,h=%d) = %v, want the zero value %v", tc.w, tc.h, got, want)
			}
		}
	})

	t.Run("a tile outside the grid clamps rather than panicking", func(t *testing.T) {
		// The grid's own bottom-right tile: every +1 offset already clamps to
		// (1,1), so all four corners equal the single level there. This is
		// the in-grid "corner tile" case, not yet outside the grid.
		want := [4]uint8{40, 40, 40, 40}
		if got := terrain.CornerLevels(levels, 2, 2, 1, 1); got != want {
			t.Errorf("CornerLevels(2x2, col=1,row=1) = %v, want %v", got, want)
		}

		// A (col,row) itself past the grid clamps every corner into the
		// grid's own far corner instead of indexing out of range.
		if got := terrain.CornerLevels(levels, 2, 2, 5, 5); got != want {
			t.Errorf("CornerLevels(2x2, col=5,row=5) = %v, want %v (clamped to the grid's own corner)", got, want)
		}

		// A negative column clamps BOTH the tile's own column and its +1
		// column to 0 (col+1 is still <= 0), so TL and TR collapse together;
		// the row is untouched.
		if got, want := terrain.CornerLevels(levels, 2, 2, -1, 0), ([4]uint8{10, 10, 20, 20}); got != want {
			t.Errorf("CornerLevels(2x2, col=-1,row=0) = %v, want %v", got, want)
		}

		// A tile fully outside the grid, negative on both axes, collapses to
		// the single nearest grid vertex.
		if got, want := terrain.CornerLevels(levels, 2, 2, -3, -3), ([4]uint8{10, 10, 10, 10}); got != want {
			t.Errorf("CornerLevels(2x2, col=-3,row=-3) = %v, want %v", got, want)
		}
	})
}

// TestShadeCorpusRange covers AC-13: every level in 0..95 (the corpus band 30..74
// is a subset) produces a defined result with no panic / out-of-range, for a
// spread of channels — via both ShadeChannel and ShadeRGBA.
func TestShadeCorpusRange(t *testing.T) {
	channels := []uint8{0, 30, 74, 128, 200, 255}
	noPanicShade(t, "corpus levels 0..95", func() {
		for _, ch := range channels {
			for level := 0; level <= 95; level++ {
				// Result is a uint8 by type, so it is inherently within [0,255];
				// the point of AC-13 is that every corpus level is defined and
				// never panics / indexes out of range.
				_ = terrain.ShadeChannel(ch, 0, level)
				out := terrain.ShadeRGBA(color.RGBA{R: ch, G: ch, B: ch, A: 0}, [3]uint8{0, 0, 0}, level)
				if out.A != 0xff {
					t.Fatalf("ShadeRGBA level %d: A=%d, want 0xff", level, out.A)
				}
			}
		}
	})
}
