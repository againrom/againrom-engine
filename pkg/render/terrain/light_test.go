package terrain_test

// Tests for the per-vertex level-grid / light API of pkg/render/terrain.
//
// Every expected value below is DERIVED FROM THE SPEC FORMULA
// (docs/0007-terrain-lighting/spec.md, TERR-LIGHT-013 with the TERR-LIGHT-028
// gradient), never read from the implementation. Per vertex (x,y):
//
//	tanT   = tan|theta| ; stepH = 32.0 / cos|theta|      heights read SIGNED
//	dh1    = H[x -+ tanT, y+1] - H[x, y]                 forward,  ONE row
//	dh2    = H[x, y] - H[x -+ tanT, y-1]                 backward, ONE row
//	slopeI = atan2(dhI, stepH)
//	axisI  = clamp(L - R*sin(pi/6 - slopeI), 0, 95)      per half, BEFORE averaging
//	level  = ftol(0.5*(axis1 + axis2))                   truncate toward zero
//
// with R = Range and L = (Range>>1) + Ambient + 0x20. The fractional column is
// H[x,y] - (H[x,y]-H[x-1,y])*tanT for theta >= 0 (and the +x mirror for theta<0,
// and for the backward half on row y == 1).
//
// Each test hand-derives its two deltas from that definition for the fixture it
// builds, then runs them through specLevel below, which is the spec's back half.
// The deltas are the discriminating part: a perpendicular-axis or two-cell model
// produces different ones on the same fixture.
//
// This file tests ONLY the light / level-grid API. It never references the
// shading-transform functions (ShadeChannel / ShadeRGBA / InterpRow).

import (
	"math"
	"testing"

	terrain "againrom/pkg/render/terrain"
)

// lvlAt indexes the row-major W*H level grid returned by LevelGrid.
func lvlAt(g []uint8, w, r, c int) uint8 { return g[r*w+c] }

// flatHeightsGrid builds a w*h height grid whose vertices are all v.
func flatHeightsGrid(w, h int, v uint8) []uint8 {
	out := make([]uint8, w*h)
	for i := range out {
		out[i] = v
	}
	return out
}

// rowsGrid builds a w*h grid whose height depends only on the row.
func rowsGrid(w int, rows []uint8) []uint8 {
	out := make([]uint8, w*len(rows))
	for r, v := range rows {
		for c := 0; c < w; c++ {
			out[r*w+c] = v
		}
	}
	return out
}

// specStepH and specTanT are the spec's two theta terms.
func specStepH(theta float64) float64 { return 32.0 / math.Cos(math.Abs(theta)) }
func specTanT(theta float64) float64  { return math.Tan(math.Abs(theta)) }

// specAxis is the spec's per-half level: clamp(L - R*sin(pi/6 - atan2(dh,stepH)), 0, 95).
func specAxis(dh float64, lt terrain.Light) float64 {
	L := float64(int(lt.Range)>>1 + int(lt.Ambient) + 0x20)
	R := float64(lt.Range)
	v := L - R*math.Sin(math.Pi/6-math.Atan2(dh, specStepH(lt.Theta)))
	if v < 0 {
		return 0
	}
	if v > 95 {
		return 95
	}
	return v
}

// specLevel is the spec's back half: clamp each delta to a level, then average
// the two ALREADY-CLAMPED levels and truncate.
func specLevel(dh1, dh2 float64, lt terrain.Light) int {
	return int(0.5 * (specAxis(dh1, lt) + specAxis(dh2, lt)))
}

// noPanicLight runs fn and fails the test if it panics.
func noPanicLight(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("%s panicked: %v", name, rec)
		}
	}()
	fn()
}

// TestLevelGridFlatIsFortySix covers AC-1: a flat grid under the default light
// levels EVERY vertex to exactly 46 (both deltas are zero everywhere, ring
// included), and the default light is the engine's cycle-off configuration.
//
// Derivation: DefaultDaytime has Ambient=0x0e, Range=0x20, so R=32 and
// L=(0x20>>1)+0x0e+0x20 = 16+14+32 = 62. A flat vertex has dh1=dh2=0 -> slope=0,
// so each half is L - R*sin(pi/6) = 62 - 32*0.5 = 46, and
// level = ftol(0.5*(46+46)) = 46 -- at ANY theta.
func TestLevelGridFlatIsFortySix(t *testing.T) {
	// Guard: the derivation above relies on the documented default bytes.
	if terrain.DefaultDaytime.Ambient != 0x0e || terrain.DefaultDaytime.Range != 0x20 {
		t.Fatalf("DefaultDaytime bytes drifted: ambient=%#x range=%#x, want 0x0e/0x20",
			terrain.DefaultDaytime.Ambient, terrain.DefaultDaytime.Range)
	}
	// The engine's cycle-off arm stores the literal 0x3fe921fb4d12d84a =
	// 0.78539815 (= 3.1415926/4), NOT math.Pi/4 (TERR-LIGHT-030). Assert the
	// literal exactly, and assert it DIFFERS from math.Pi/4, so a silent swap
	// back to the constant fails here.
	if terrain.DefaultDaytime.Theta != 0.78539815 {
		t.Fatalf("DefaultDaytime.Theta = %v, want the literal 0.78539815", terrain.DefaultDaytime.Theta)
	}
	if terrain.DefaultDaytime.Theta == math.Pi/4 {
		t.Fatalf("DefaultDaytime.Theta is math.Pi/4; the engine stores a different double")
	}
	if math.Float64bits(terrain.DefaultDaytime.Theta) != 0x3fe921fb4d12d84a {
		t.Fatalf("DefaultDaytime.Theta bits = %#x, want 0x3fe921fb4d12d84a",
			math.Float64bits(terrain.DefaultDaytime.Theta))
	}
	if terrain.DefaultDaytime.SkyTint != [3]uint8{0, 0, 0} {
		t.Fatalf("DefaultDaytime.SkyTint = %v, want {0,0,0}", terrain.DefaultDaytime.SkyTint)
	}

	const w, h = 5, 5
	for _, height := range []uint8{0, 100, 127} {
		heights := flatHeightsGrid(w, h, height)
		g := terrain.LevelGrid(heights, w, h, terrain.DefaultDaytime)
		if len(g) != w*h {
			t.Fatalf("len(grid) = %d, want %d", len(g), w*h)
		}
		// Only differences matter, so the flat value is 46 at any height level, on
		// the outer ring as well as the interior.
		for i, v := range g {
			if v != 46 {
				t.Errorf("flat height %d: vertex[%d] = %d, want 46", height, i, v)
			}
		}
	}

	// Flat is 46 at any theta, because both deltas are zero.
	for _, th := range []float64{0, 0.3, math.Pi / 4, 1.4, -1.0} {
		lt := terrain.DefaultDaytime
		lt.Theta = th
		g := terrain.LevelGrid(flatHeightsGrid(w, h, 60), w, h, lt)
		for i, v := range g {
			if v != 46 {
				t.Errorf("theta=%v: flat vertex[%d] = %d, want 46", th, i, v)
			}
		}
	}
}

// TestLevelGridFlatIntensity covers AC-2: a flat grid at several (ambient,range)
// pairs levels every vertex to int(L - R*sin(pi/6)) computed in float64 (NOT the
// integer L-(R>>1); they diverge for odd range). Range values are chosen so the
// truncation lands off an integer boundary (a .5 midpoint) and is genuinely
// observed.
func TestLevelGridFlatIntensity(t *testing.T) {
	pairs := []struct {
		amb, rng uint8
		wantSpot int // hand-derived expected, cross-checked against the float formula
	}{
		// daytime: L=62, R=32 -> int(62 - 32*0.5) = int(46.0) = 46.
		{0x0e, 0x20, 46},
		// odd range: L=(0x21>>1)+0+0x20 = 16+0+32 = 48, R=33.
		// int(48 - 33*0.5) = int(48 - 16.5) = int(31.5) = 31.  (integer L-(R>>1) = 48-16 = 32; diverges.)
		{0x00, 0x21, 31},
		// odd range: L=(0x1f>>1)+5+0x20 = 15+5+32 = 52, R=31.
		// int(52 - 31*0.5) = int(52 - 15.5) = int(36.5) = 36.  (integer L-(R>>1) = 52-15 = 37; diverges.)
		{0x05, 0x1f, 36},
	}

	const w, h = 5, 5
	heights := flatHeightsGrid(w, h, 120)

	for _, p := range pairs {
		L := int(p.rng>>1) + int(p.amb) + 0x20 // integer combination of the light bytes
		R := int(p.rng)
		// Flat -> both deltas 0 -> sin(pi/6 - 0). Same float64 expression the formula uses.
		exp := int(float64(L) - float64(R)*math.Sin(math.Pi/6))
		if exp != p.wantSpot {
			t.Fatalf("derivation drift for amb=%#x rng=%#x: formula=%d wantSpot=%d",
				p.amb, p.rng, exp, p.wantSpot)
		}
		lt := terrain.Light{Theta: terrain.DefaultTheta, Ambient: p.amb, Range: p.rng}
		g := terrain.LevelGrid(heights, w, h, lt)
		for r := 1; r <= h-2; r++ {
			for c := 1; c <= w-2; c++ {
				if got := lvlAt(g, w, r, c); got != uint8(exp) {
					t.Errorf("amb=%#x rng=%#x interior (%d,%d) = %d, want %d",
						p.amb, p.rng, r, c, got, exp)
				}
			}
		}
	}
}

// TestLevelGridRidge covers AC-3: a ramp ACROSS THE ROWS (height varying with y,
// constant along x) pins the sign->level direction against the exact computed
// value.
//
// Along a row-constant field the lateral tanT shear contributes nothing (both
// lateral operands of a row are equal), so both halves take the same one-cell
// row difference and there is NO halving toward 46:
//
//	up-ramp   h(y) = 20*y : dh1 = h(y+1)-h(y) = +20, dh2 = h(y)-h(y-1) = +20
//	down-ramp h(y) = 80-20*y : both deltas = -20
//
// Daytime L=62, R=32, stepH = 32/cos(theta) ~ 45.2548:
//
//	axis(+20) = 62 - 32*sin(pi/6 - atan2( 20,45.25)) ~ 58.56 -> level 58 (> 46)
//	axis(-20) = 62 - 32*sin(pi/6 - atan2(-20,45.25)) ~ 36.17 -> level 36 (< 46)
func TestLevelGridRidge(t *testing.T) {
	const w, h = 5, 5
	lt := terrain.DefaultDaytime

	up := rowsGrid(w, []uint8{0, 20, 40, 60, 80})
	down := rowsGrid(w, []uint8{80, 60, 40, 20, 0})

	expUp := specLevel(20, 20, lt)
	expDown := specLevel(-20, -20, lt)
	if expUp <= 46 {
		t.Fatalf("derivation: up-slope level %d not > 46", expUp)
	}
	if expDown >= 46 {
		t.Fatalf("derivation: down-slope level %d not < 46", expDown)
	}

	gu := terrain.LevelGrid(up, w, h, lt)
	gd := terrain.LevelGrid(down, w, h, lt)

	// Rows 2 and 3 are interior and are not the y==1 row (whose backward half
	// takes the mirrored lateral arm; on a row-constant field it is the same
	// value, but the assertion is kept to the plainly-derived rows).
	for _, r := range []int{2, 3} {
		for _, c := range []int{1, 2, 3} {
			if got := lvlAt(gu, w, r, c); int(got) != expUp {
				t.Errorf("up-ramp (%d,%d) = %d, want %d", r, c, got, expUp)
			}
			if got := lvlAt(gd, w, r, c); int(got) != expDown {
				t.Errorf("down-ramp (%d,%d) = %d, want %d", r, c, got, expDown)
			}
		}
	}
	if lvlAt(gu, w, 2, 2) <= 46 {
		t.Errorf("up-ramp interior = %d, want strictly > 46", lvlAt(gu, w, 2, 2))
	}
	if lvlAt(gd, w, 2, 2) >= 46 {
		t.Errorf("down-ramp interior = %d, want strictly < 46", lvlAt(gd, w, 2, 2))
	}
}

// TestLevelGridRowAxisGradient covers AC-14: the gradient axis is the ROW axis,
// and no perpendicular x difference is formed. Two fields of the same gradient
// magnitude (20 per cell) must level DIFFERENTLY:
//
//   - h = 20*y (varies with y): both halves take the one-cell row difference,
//     dh1 = dh2 = +20 -> level 58.
//   - h = 20*x (varies with x): every row is a ramp, so the row differences are
//     zero and the level moves only through the lateral shear --
//     dh1 = -20*tanT, dh2 = +20*tanT -> level 47.
//
// Under the superseded perpendicular two-axis model the two fields would level
// identically, so this discriminates the corrected gradient.
func TestLevelGridRowAxisGradient(t *testing.T) {
	const w, h = 5, 5
	lt := terrain.DefaultDaytime
	tanT := specTanT(lt.Theta)

	byRow := rowsGrid(w, []uint8{0, 20, 40, 60, 80})
	byCol := make([]uint8, w*h)
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			byCol[r*w+c] = uint8(20 * c)
		}
	}

	expRow := specLevel(20, 20, lt)
	expCol := specLevel(-20*tanT, 20*tanT, lt)
	if expRow == expCol {
		t.Fatalf("derivation: row (%d) and column (%d) fields must differ", expRow, expCol)
	}

	gr := terrain.LevelGrid(byRow, w, h, lt)
	gc := terrain.LevelGrid(byCol, w, h, lt)

	for _, r := range []int{2, 3} {
		for _, c := range []int{1, 2, 3} {
			if got := lvlAt(gr, w, r, c); int(got) != expRow {
				t.Errorf("row-varying field (%d,%d) = %d, want %d", r, c, got, expRow)
			}
			if got := lvlAt(gc, w, r, c); int(got) != expCol {
				t.Errorf("column-varying field (%d,%d) = %d, want %d", r, c, got, expCol)
			}
		}
	}
	if lvlAt(gr, w, 2, 2) == lvlAt(gc, w, 2, 2) {
		t.Errorf("row- and column-varying fields level identically (%d); the gradient is not axis-specific",
			lvlAt(gr, w, 2, 2))
	}
}

// TestLevelGridLateralShear covers AC-15: the lateral tan|theta| shear is applied
// inside the adjacent row. On a 0/127 checkerboard at the default theta
// (tanT ~ 1) the sheared sample lands on the diagonal neighbour, which shares the
// vertex's parity -- so both deltas vanish and every interior vertex levels to
// exactly 46.
//
// The same fixture read WITHOUT the shear gives dh1 = +127, dh2 = -127 (or the
// mirror), i.e. level 56 -- computed here so the assertion fails if the shear is
// dropped or mis-signed.
func TestLevelGridLateralShear(t *testing.T) {
	const w, h = 6, 6
	lt := terrain.DefaultDaytime

	checker := make([]uint8, w*h)
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			if (r+c)%2 == 1 {
				checker[r*w+c] = 127
			}
		}
	}

	noShear := specLevel(127, -127, lt)
	if noShear == 46 {
		t.Fatalf("derivation: a shear-free reading gives %d, which does not discriminate", noShear)
	}

	g := terrain.LevelGrid(checker, w, h, lt)
	for r := 1; r <= h-2; r++ {
		for c := 1; c <= w-2; c++ {
			if got := lvlAt(g, w, r, c); got != 46 {
				t.Errorf("checkerboard interior (%d,%d) = %d, want 46 (shear-free would give %d)",
					r, c, got, noShear)
			}
		}
	}
}

// TestLevelGridSignedHeights covers AC-16: height bytes are read SIGNED, as the
// engine's MOVSX does. Rows 0x80/0x90 are -128/-112 signed and 128/144 unsigned;
// the fixture is asymmetric so the two readings give different levels.
//
//	rows      = [0x00, 0x00, 0x80, 0x90, 0x90]
//	at y = 2, signed:   dh1 = -112 - (-128) = +16 ; dh2 = -128 - 0 = -128
//	          unsigned: dh1 =  144 -   128  = +16 ; dh2 =  128 - 0 = +128
func TestLevelGridSignedHeights(t *testing.T) {
	const w, h = 5, 5
	lt := terrain.DefaultDaytime

	heights := rowsGrid(w, []uint8{0x00, 0x00, 0x80, 0x90, 0x90})

	expSigned := specLevel(16, -128, lt)
	expUnsigned := specLevel(16, 128, lt)
	if expSigned == expUnsigned {
		t.Fatalf("derivation: signed (%d) and unsigned (%d) readings must differ", expSigned, expUnsigned)
	}

	g := terrain.LevelGrid(heights, w, h, lt)
	for _, c := range []int{1, 2, 3} {
		got := lvlAt(g, w, 2, c)
		if int(got) != expSigned {
			t.Errorf("vertex (2,%d) = %d, want %d (signed height reads)", c, got, expSigned)
		}
		if int(got) == expUnsigned {
			t.Errorf("vertex (2,%d) = %d, which is the UNSIGNED reading", c, got)
		}
	}
}

func TestLevelGridClampsAxes(t *testing.T) {
	const w, h = 5, 5
	override := terrain.Light{Theta: terrain.DefaultTheta, Ambient: 0, Range: 0xFF}

	if got := specAxis(40, override); got != 95 {
		t.Fatalf("derivation: axis(+40) = %v, want the high clamp 95", got)
	}
	if got := specAxis(-40, override); got != 0 {
		t.Fatalf("derivation: axis(-40) = %v, want the low clamp 0", got)
	}

	up := rowsGrid(w, []uint8{0, 40, 80, 120, 160 - 1})
	if got := lvlAt(terrain.LevelGrid(up, w, h, override), w, 2, 2); got != 95 {
		t.Errorf("up-ramp vertex (2,2) = %d, want 95 (both halves saturate high)", got)
	}

	down := rowsGrid(w, []uint8{120, 100, 80, 60, 40})
	if got := lvlAt(terrain.LevelGrid(down, w, h, override), w, 2, 2); got != 0 {
		t.Errorf("down-ramp vertex (2,2) = %d, want 0 (both halves saturate low)", got)
	}

	// Valley at row 2: dh1 = h(3)-h(2) = +40, dh2 = h(2)-h(1) = -40.
	valley := rowsGrid(w, []uint8{80, 80, 40, 80, 80})
	if got := lvlAt(terrain.LevelGrid(valley, w, h, override), w, 2, 2); got != 47 {
		t.Errorf("valley vertex (2,2) = %d, want 47 (halves clamp independently)", got)
	}

	// Hostile grid over several thetas: no level may ever leave [0,95]. The
	// bytes straddle 0x80, so this also exercises the signed reads.
	const hw, hh = 6, 6
	hostile := make([]uint8, hw*hh)
	for r := 0; r < hh; r++ {
		for c := 0; c < hw; c++ {
			if (r+c)%2 == 0 {
				hostile[r*hw+c] = 0x80
			} else {
				hostile[r*hw+c] = 0x7f
			}
		}
	}
	thetas := []float64{0, math.Pi / 6, terrain.DefaultTheta, math.Pi / 3, -math.Pi / 4, 1.4}
	bases := []terrain.Light{
		{Ambient: 0, Range: 0xFF},
		{Ambient: 0x0e, Range: 0x20},
	}
	for _, th := range thetas {
		for _, b := range bases {
			lt := b
			lt.Theta = th
			g := terrain.LevelGrid(hostile, hw, hh, lt)
			for i, v := range g {
				if v > 95 { // v is uint8, so v >= 0 is automatic
					t.Errorf("theta=%v amb=%#x rng=%#x: level[%d]=%d out of [0,95]",
						th, b.Ambient, b.Range, i, v)
				}
			}
		}
	}
}

func TestLevelGridBordersAndDegenerate(t *testing.T) {
	const w, h = 5, 5
	lt := terrain.DefaultDaytime
	ramp := rowsGrid(w, []uint8{0, 20, 40, 60, 80})

	expInterior := specLevel(20, 20, lt)
	expTop := specLevel(20, 0, lt)    // backward half collapses on row 0
	expBottom := specLevel(0, 20, lt) // forward half collapses on row H-1
	if expTop == expInterior || expBottom == expInterior {
		t.Fatalf("derivation: ring (%d/%d) and interior (%d) should differ", expTop, expBottom, expInterior)
	}

	var g []uint8
	noPanicLight(t, "LevelGrid(ramp 5x5)", func() {
		g = terrain.LevelGrid(ramp, w, h, lt)
	})
	if len(g) != w*h {
		t.Fatalf("ramp grid: len = %d, want %d", len(g), w*h)
	}
	for _, c := range []int{1, 2, 3} {
		if got := lvlAt(g, w, 2, c); int(got) != expInterior {
			t.Errorf("interior (2,%d) = %d, want %d", c, got, expInterior)
		}
		if got := lvlAt(g, w, 0, c); int(got) != expTop {
			t.Errorf("top-ring (0,%d) = %d, want %d (backward half clamped)", c, got, expTop)
		}
		if got := lvlAt(g, w, h-1, c); int(got) != expBottom {
			t.Errorf("bottom-ring (%d,%d) = %d, want %d (forward half clamped)", h-1, c, got, expBottom)
		}
	}

	// (b) Degenerate shapes: full W*H slice, correct length, no panic.
	//     1x1 and all-flat grids level to the flat daytime value 46.
	type shape struct {
		name    string
		heights []uint8
		w, h    int
		flat    bool // if true, all vertices must equal 46
	}
	shapes := []shape{
		{"1x1", []uint8{100}, 1, 1, true},
		{"1x5 flat", flatHeightsGrid(5, 1, 50), 5, 1, true},
		{"5x1 flat", flatHeightsGrid(1, 5, 50), 1, 5, true},
		{"1x5 varying", []uint8{0, 20, 40, 60, 80}, 5, 1, false},
		{"5x1 varying", []uint8{0, 20, 40, 60, 80}, 1, 5, false},
	}
	for _, s := range shapes {
		var out []uint8
		noPanicLight(t, "LevelGrid("+s.name+")", func() {
			out = terrain.LevelGrid(s.heights, s.w, s.h, lt)
		})
		if out == nil {
			t.Errorf("%s: got nil, want non-nil slice", s.name)
			continue
		}
		if len(out) != s.w*s.h {
			t.Errorf("%s: len = %d, want %d", s.name, len(out), s.w*s.h)
		}
		if s.flat {
			for i, v := range out {
				if v != 46 {
					t.Errorf("%s: flat vertex[%d] = %d, want 46", s.name, i, v)
				}
			}
		}
	}

	// The documented nil returns: w<1, h<1, or too-short heights slice.
	if got := terrain.LevelGrid(flatHeightsGrid(5, 5, 10), 0, 5, lt); got != nil {
		t.Errorf("w<1 should return nil, got len %d", len(got))
	}
	if got := terrain.LevelGrid(flatHeightsGrid(5, 5, 10), 5, 0, lt); got != nil {
		t.Errorf("h<1 should return nil, got len %d", len(got))
	}
	if got := terrain.LevelGrid([]uint8{1, 2, 3}, 5, 5, lt); got != nil {
		t.Errorf("len(heights) < w*h should return nil, got len %d", len(got))
	}
}

func TestLevelGridPure(t *testing.T) {
	const w, h = 4, 4
	heights := []uint8{
		10, 50, 90, 120,
		20, 60, 100, 110,
		30, 70, 105, 115,
		40, 80, 118, 125,
	}
	orig := append([]uint8(nil), heights...)

	g1 := terrain.LevelGrid(heights, w, h, terrain.DefaultDaytime)
	g2 := terrain.LevelGrid(heights, w, h, terrain.DefaultDaytime)

	if len(g1) != len(g2) {
		t.Fatalf("lengths differ: %d vs %d", len(g1), len(g2))
	}
	for i := range g1 {
		if g1[i] != g2[i] {
			t.Errorf("non-deterministic at %d: %d vs %d", i, g1[i], g2[i])
		}
	}
	if len(heights) != len(orig) {
		t.Fatalf("heights length changed: %d vs %d", len(heights), len(orig))
	}
	for i := range orig {
		if heights[i] != orig[i] {
			t.Errorf("input heights mutated at %d: %d, want %d", i, heights[i], orig[i])
		}
	}
}

// TestLightFromFields covers the AC-7 unit part: LightFromFields passes theta
// through as float64(angle) and takes ambient/range as the LOW BYTE of the u32s;
// SkyTint stays {0,0,0}.
func TestLightFromFields(t *testing.T) {
	cases := []struct {
		angle   float32
		ambient uint32
		rng     uint32
		wantAmb uint8
		wantRng uint8
	}{
		{0.5, 0x00AB, 0xFF12, 0xAB, 0x12},   // low byte of 0x00AB=0xAB, 0xFF12=0x12
		{-0.75, 0x1234, 0x0000, 0x34, 0x00}, // low byte of 0x1234=0x34
		{float32(math.Pi / 4), 0x000000FF, 0x000000C0, 0xFF, 0xC0},
	}
	for _, c := range cases {
		lt := terrain.LightFromFields(c.angle, c.ambient, c.rng)
		if lt.Theta != float64(c.angle) {
			t.Errorf("angle %v: Theta = %v, want %v", c.angle, lt.Theta, float64(c.angle))
		}
		if lt.Ambient != c.wantAmb {
			t.Errorf("ambient %#x: Ambient = %#x, want %#x", c.ambient, lt.Ambient, c.wantAmb)
		}
		if lt.Range != c.wantRng {
			t.Errorf("range %#x: Range = %#x, want %#x", c.rng, lt.Range, c.wantRng)
		}
		if lt.SkyTint != [3]uint8{0, 0, 0} {
			t.Errorf("SkyTint = %v, want {0,0,0}", lt.SkyTint)
		}
		// The angle must be a representative in-range sun angle.
		if lt.Theta < -math.Pi/2 || lt.Theta > math.Pi/2 {
			t.Errorf("test angle %v outside representative [-pi/2, pi/2]", lt.Theta)
		}
	}
}
