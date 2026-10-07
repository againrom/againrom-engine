package sim

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"testing"
)

// TestEveryWindowOffsetHasACloserPredecessor is AC-1, and it is the property the
// whole march rests on: if one offset's predecessor were no closer to the centre
// than the offset itself, that offset's chain would not terminate and its value
// would be read before it was written.
func TestEveryWindowOffsetHasACloserPredecessor(t *testing.T) {
	t.Parallel()

	n := 0
	for dx := int32(-sightHalf); dx <= sightHalf; dx++ {
		for dy := int32(-sightHalf); dy <= sightHalf; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			n++
			at := sightOffset(dx, dy)
			px, py := dx+int32(sightTables.stepX[at]), dy+int32(sightTables.stepY[at])
			if got, want := cheb(px, py), cheb(dx, dy); got >= want {
				t.Fatalf("offset (%d,%d) at Chebyshev %d steps to (%d,%d) at %d",
					dx, dy, want, px, py, got)
			}
		}
	}
	if n != 1680 {
		t.Errorf("the window holds %d non-centre offsets, want 1680", n)
	}
	// The two the builder repairs by hand step straight to the centre.
	for _, dx := range []int32{1, -1} {
		at := sightOffset(dx, 0)
		if sightTables.stepX[at] != int8(-dx) || sightTables.stepY[at] != 0 {
			t.Errorf("offset (%d,0) steps (%d,%d), want (%d,0)",
				dx, sightTables.stepX[at], sightTables.stepY[at], -dx)
		}
	}
}

// TestWithoutTheBuilderRepairExactlyTwoOffsetsFail is the other half of AC-1: the
// four trailing literal stores are not redundant, and re-executing the builder
// without them leaves exactly the two cells the law repairs and no others.
//
// It is written as a re-execution rather than as a comment because "exactly two"
// is the discriminator — a reading that got the zone boundary or the mirroring
// order wrong would leave some other number.
func TestWithoutTheBuilderRepairExactlyTwoOffsetsFail(t *testing.T) {
	t.Parallel()

	var sx, sy [sightArea]int8
	for i := int32(0); i <= sightHalf; i++ {
		for j := int32(0); j <= sightHalf; j++ {
			zx, zy := sightZoneStep(i, j)
			for _, q := range [4][2]int32{{1, 1}, {-1, 1}, {1, -1}, {-1, -1}} {
				at := sightOffset(q[0]*i, q[1]*j)
				sx[at], sy[at] = int8(zx*q[0]), int8(zy*q[1])
			}
		}
	}
	var bad [][2]int32
	for dx := int32(-sightHalf); dx <= sightHalf; dx++ {
		for dy := int32(-sightHalf); dy <= sightHalf; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			at := sightOffset(dx, dy)
			if cheb(dx+int32(sx[at]), dy+int32(sy[at])) >= cheb(dx, dy) {
				bad = append(bad, [2]int32{dx, dy})
			}
		}
	}
	if len(bad) != 2 || bad[0] != [2]int32{-1, 0} || bad[1] != [2]int32{1, 0} {
		t.Errorf("without the repair the offsets that fail are %v, want exactly (-1,0) and (1,0)", bad)
	}
}

// TestTheStepCostIsTheRaysMeanStep is AC-2. The axis step is one whole unit of
// the fixed point, the diagonal step is that times the square root of two, and
// nothing in the window falls outside the two.
func TestTheStepCostIsTheRaysMeanStep(t *testing.T) {
	t.Parallel()

	lo, hi := int32(1<<30), int32(0)
	for dx := int32(-sightHalf); dx <= sightHalf; dx++ {
		for dy := int32(-sightHalf); dy <= sightHalf; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			c := sightTables.cost[sightOffset(dx, dy)]
			if c < lo {
				lo = c
			}
			if c > hi {
				hi = c
			}
			switch {
			case dx == 0 || dy == 0:
				if c != sightStep {
					t.Fatalf("axis offset (%d,%d) costs %d, want %d", dx, dy, c, sightStep)
				}
			case dx == dy || dx == -dy:
				if c != 181 {
					t.Fatalf("diagonal offset (%d,%d) costs %d, want 181", dx, dy, c)
				}
			}
		}
	}
	if lo != sightStep || hi != 181 {
		t.Errorf("the window's step costs run %d..%d, want %d..181", lo, hi, sightStep)
	}
}

// TestTheIntegerStepCostIsThePublishedExpression is SC-1. The law computes the
// cost on the FPU and truncates; this package may hold no floating-point value
// in the code that ships, so the transcription is an integer identity. Whether
// the identity holds is a measurement over the whole window, taken here — in a
// test, where a float is allowed — rather than an argument in a comment.
func TestTheIntegerStepCostIsThePublishedExpression(t *testing.T) {
	t.Parallel()

	for i := int32(0); i <= sightHalf; i++ {
		for j := int32(0); j <= sightHalf; j++ {
			m := i
			if j > m {
				m = j
			}
			if m == 0 {
				continue
			}
			want := int32(float64(sightStep) * math.Sqrt(float64(i*i+j*j)) / float64(m))
			if got := sightStepCost(i, j, sightShift); got != want {
				t.Fatalf("cost(%d,%d) = %d, the published expression gives %d", i, j, got, want)
			}
		}
	}
}

// TestTheIntegerSquareRootIsExact pins isqrt at and around every square in the
// range the cost table reaches, which is where an off-by-one would put a cost one
// unit out and move the far edge of a whole ray.
func TestTheIntegerSquareRootIsExact(t *testing.T) {
	t.Parallel()

	for _, tc := range [][2]int32{{-1, 0}, {0, 0}} {
		if got := isqrt(tc[0]); got != tc[1] {
			t.Errorf("isqrt(%d) = %d, want %d", tc[0], got, tc[1])
		}
	}
	// From 1 up: the square itself, one past it, and one short of it, which is
	// where the previous root must still be the answer.
	for v := int32(1); v <= 3700; v++ {
		s := v * v
		for _, tc := range [][2]int32{{s, v}, {s + 1, v}, {s - 1, v - 1}} {
			if got := isqrt(tc[0]); got != tc[1] {
				t.Fatalf("isqrt(%d) = %d, want %d", tc[0], got, tc[1])
			}
		}
	}
}

// cheb is the Chebyshev norm of an offset, spelled for the tests that reason
// about the window rather than about a world.
func cheb(dx, dy int32) int32 {
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

// sightBounds is wide enough for a march of the longest range this build's ring
// bound allows to run out in every direction without meeting an edge.
var sightBounds = Bounds{Width: 45, Height: 45}

// sightWorld is a world over b carrying terrain and no entities: these tests
// march from a cell, not from a unit.
func sightWorld(t *testing.T, b Bounds, ter Terrain) *World {
	t.Helper()
	w, err := NewTerrainWorld(1, b, ModeCanonical, ter, nil, nil)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	return w
}

// sightPlane is a plane of b's size with every cell at v, so a test names only
// the cells it means to be different.
func sightPlane(b Bounds, v byte) []byte {
	p := make([]byte, int(b.Width)*int(b.Height))
	for i := range p {
		p[i] = v
	}
	return p
}

// sightAt marches one observer and answers the stamp plus how many cells it lit.
// sightTestRange is the range the worlds below are laid out against, and it is
// THIS FILE'S OWN NUMBER. It stood at the package constant that seeded every
// march until 0091 made the range an entity's own; a test that took its distances
// from the value it was exercising would move with that value and pin nothing, so
// the fixture states one.
const sightTestRange = 5

func sightAt(w *World, from cell, scanRange int32) ([]byte, int) {
	stamp := make([]byte, len(w.grid))
	w.marchSight(aiSight, stamp, from, scanRange)
	n := 0
	for _, b := range stamp {
		if b != 0 {
			n++
		}
	}
	return stamp, n
}

// TestTheFlatGroundRegionIsThePublishedDisc is AC-3 and SC-2. The six counts and
// the far one are the round's own measurements of the routine this file
// transcribes, and they are the discriminator for the whole reading: the zone
// boundaries, the mirroring order, the truncation and the seed each move at
// least one of them.
func TestTheFlatGroundRegionIsThePublishedDisc(t *testing.T) {
	t.Parallel()

	w := sightWorld(t, sightBounds, Terrain{})
	centre := cell{x: 22, y: 22}
	for _, tc := range []struct {
		scanRange int32
		want      int
	}{{1, 9}, {2, 21}, {3, 45}, {4, 69}, {5, 105}, {6, 145}} {
		if _, got := sightAt(w, centre, tc.scanRange); got != tc.want {
			t.Errorf("a range-%d march lights %d cells, want %d", tc.scanRange, got, tc.want)
		}
	}
	// At 19 the ring bound is what stops the walk rather than the budget, so the
	// figure also pins that the outermost ring of the window is never visited.
	far := sightWorld(t, Bounds{Width: 81, Height: 81}, Terrain{})
	if _, got := sightAt(far, cell{x: 40, y: 40}, 19); got != 1253 {
		t.Errorf("a range-19 march lights %d cells, want 1253", got)
	}
}

// TestTheReachIsTheBudgetDividedByTheStep is AC-4: the flat-ground far edge of a
// ray is the budget divided by that ray's own step cost, which is what makes the
// region a disc rather than the square its window is.
func TestTheReachIsTheBudgetDividedByTheStep(t *testing.T) {
	t.Parallel()

	w := sightWorld(t, sightBounds, Terrain{})
	centre := cell{x: 22, y: 22}
	for r := int32(1); r <= 6; r++ {
		stamp, _ := sightAt(w, centre, r)
		axis := int32(0)
		for axis < sightHalf && w.sightShows(stamp, cell{x: centre.x + axis + 1, y: centre.y}) {
			axis++
		}
		diag := int32(0)
		for diag < sightHalf && w.sightShows(stamp, cell{x: centre.x + diag + 1, y: centre.y + diag + 1}) {
			diag++
		}
		if axis != r {
			t.Errorf("a range-%d march reaches %d cells along the axis, want %d", r, axis, r)
		}
		if want := sightSeed(r<<8, sightShift) / 181; diag != want {
			t.Errorf("a range-%d march reaches %d cells along the diagonal, want %d", r, diag, want)
		}
	}
}

// TestGroundBetweenTheObserverAndTheCellCostsSight is the march's half of AC-5.
//
// A raised cell is charged against the budget of every ray that passes through
// it, ONCE, at the cell itself — so a ridge does not cast a shadow of its own
// height, it shortens every ray that crosses it by that much. Two cells past a
// 100-high column the flat-ground ray is still alive and the ridged one is not.
func TestGroundBetweenTheObserverAndTheCellCostsSight(t *testing.T) {
	t.Parallel()

	h := sightPlane(sightBounds, 0)
	for y := int32(0); y < sightBounds.Height; y++ {
		h[int(y)*int(sightBounds.Width)+24] = 100
	}
	ridge := sightWorld(t, sightBounds, Terrain{Height: h})
	flat := sightWorld(t, sightBounds, Terrain{})
	centre := cell{x: 22, y: 22}
	beyond := cell{x: 27, y: 22}

	fs, _ := sightAt(flat, centre, sightTestRange)
	rs, _ := sightAt(ridge, centre, sightTestRange)
	if !flat.sightShows(fs, beyond) {
		t.Fatal("the flat-ground control does not see the cell the ridge is meant to hide")
	}
	if ridge.sightShows(rs, beyond) {
		t.Error("a cell behind a raised ridge is still visible")
	}
}

// TestWhereTheObserverStandsDecidesHowFarItSees is AC-6, and it is the
// consequence that separates a budget from a radius: the observer's own altitude
// is added at every step of a ray while a cell's height is charged once, so
// standing high reaches past the flat figure and standing low falls short of it.
func TestWhereTheObserverStandsDecidesHowFarItSees(t *testing.T) {
	t.Parallel()

	centre := cell{x: 22, y: 22}
	flat := sightWorld(t, sightBounds, Terrain{})
	_, base := sightAt(flat, centre, sightTestRange)

	high := sightPlane(sightBounds, 0)
	high[int(centre.y)*int(sightBounds.Width)+int(centre.x)] = 60
	hw := sightWorld(t, sightBounds, Terrain{Height: high})
	if _, got := sightAt(hw, centre, sightTestRange); got <= base {
		t.Errorf("an observer standing 60 above its surroundings lights %d cells, want more than %d",
			got, base)
	}

	low := sightPlane(sightBounds, 60)
	low[int(centre.y)*int(sightBounds.Width)+int(centre.x)] = 0
	lw := sightWorld(t, sightBounds, Terrain{Height: low})
	if _, got := sightAt(lw, centre, sightTestRange); got >= base {
		t.Errorf("an observer standing 60 below its surroundings lights %d cells, want fewer than %d",
			got, base)
	}
}

// TestABlockerIsNotPrunedAndTheMarginIsOneUnit is AC-7 and the whole of the seam
// the contract names.
//
// The law stores a blocked cell's value and walks on, so whether anything is ever
// re-lit behind a blocker is arithmetic: a cell can rise above its predecessor
// only when the observer's height less the cell's reaches the cheapest step,
// sightRelight. Every altitude byte of every shipped map is 0..127, so the
// largest rise available there is 127 — one short — and the region is star-shaped
// on all of them. One authored byte at 128 or above reads back negative and the
// margin is gone.
//
// The two worlds below differ in ONE byte and in nothing else, which is what
// makes this the seam rather than a story about occlusion: everything else about
// the terrain, the observer and the range is identical, and the byte that moves
// is the byte no shipped map contains.
func TestABlockerIsNotPrunedAndTheMarginIsOneUnit(t *testing.T) {
	t.Parallel()

	b := Bounds{Width: 25, Height: 25}
	centre := cell{x: 12, y: 12}
	blocked := cell{x: 14, y: 12}
	behind := cell{x: 15, y: 12}

	// Flat ground, an observer standing 30 above it, and one 127-high cell two
	// east of it. At a range of 2 that leaves the blocked cell three units short
	// of visible — shallow enough that a large enough rise behind it could pay
	// the next step, which is the case a build that pruned could never reach.
	plane := func(behindByte byte) []byte {
		h := sightPlane(b, 0)
		h[int(centre.y)*int(b.Width)+int(centre.x)] = 30
		h[int(blocked.y)*int(b.Width)+int(blocked.x)] = 127
		h[int(behind.y)*int(b.Width)+int(behind.x)] = behindByte
		return h
	}

	sw := sightWorld(t, b, Terrain{Height: plane(0)})
	ss, _ := sightAt(sw, centre, 2)
	if !sw.sightShows(ss, cell{x: 13, y: 12}) {
		t.Fatal("the cell before the blocker is dark, so this case tests nothing")
	}
	if sw.sightShows(ss, blocked) {
		t.Fatal("the blocker itself is visible, so this case does not test re-lighting")
	}
	if sw.sightShows(ss, behind) {
		t.Error("a cell behind a blocked one is lit on shipped altitudes")
	}

	// The same world with that one cell authored at 128. It reads back as -128,
	// the rise across it is 158, and the ray comes back to life.
	aw := sightWorld(t, b, Terrain{Height: plane(128)})
	as, _ := sightAt(aw, centre, 2)
	if aw.sightShows(as, blocked) {
		t.Fatal("the blocker moved: the two worlds differ in more than the one byte")
	}
	if !aw.sightShows(as, behind) {
		t.Error("a cell behind a blocked one is dark under an authored altitude of 128: " +
			"this build prunes where the law does not")
	}
	if sightRelight != 128 {
		t.Errorf("the re-lighting margin is %d, want the cheapest step cost 128", sightRelight)
	}

	// And the value a blocked cell keeps is its OWN, not the zero an unvisited
	// cell carries. With one more raised cell in front, the block is deep enough
	// that even a 158-unit rise cannot pay the next step — where a build that
	// stored only positive values would hand the next cell a zero and light it.
	deep := plane(128)
	deep[int(blocked.y)*int(b.Width)+int(blocked.x)-1] = 127
	dw := sightWorld(t, b, Terrain{Height: deep})
	ds, _ := sightAt(dw, centre, 2)
	if !dw.sightShows(ds, cell{x: 13, y: 12}) {
		t.Fatal("the deep case blocks one cell too early to test what it is for")
	}
	if dw.sightShows(ds, blocked) {
		t.Fatal("the deep case does not block where it is meant to")
	}
	if dw.sightShows(ds, behind) {
		t.Error("a deeply blocked cell handed its successor a zero instead of its own value")
	}
}

func TestTheWalkNeverLeavesTheNineteenthRing(t *testing.T) {
	t.Parallel()

	b := Bounds{Width: 81, Height: 81}
	centre := cell{x: 40, y: 40}
	w := sightWorld(t, b, Terrain{})
	stamp, _ := sightAt(w, centre, 25)
	// The bound is written out as 19 rather than taken from the constant: a test
	// that read the constant would move with it and pin nothing.
	if !w.sightShows(stamp, cell{x: centre.x + 19, y: centre.y}) {
		t.Fatal("the walk does not even reach the nineteenth ring, so the bound is not what stops it")
	}
	for dx := int32(-30); dx <= 30; dx++ {
		for dy := int32(-30); dy <= 30; dy++ {
			if cheb(dx, dy) <= 19 {
				continue
			}
			if w.sightShows(stamp, cell{x: centre.x + dx, y: centre.y + dy}) {
				t.Fatalf("(%d,%d) at Chebyshev %d is lit, past the ring bound 19",
					dx, dy, cheb(dx, dy))
			}
		}
	}
}

func TestAnAllBlockedRingEndsTheWalk(t *testing.T) {
	t.Parallel()

	bb := Bounds{Width: 25, Height: 25}
	centre := cell{x: 12, y: 12}
	pit := cell{x: 14, y: 12}
	build := func(open bool) *World {
		h := sightPlane(bb, 120)
		h[int(centre.y)*int(bb.Width)+int(centre.x)] = 30
		h[int(pit.y)*int(bb.Width)+int(pit.x)] = 128
		if open {
			h[int(centre.y+1)*int(bb.Width)+int(centre.x)] = 0
		}
		return sightWorld(t, bb, Terrain{Height: h})
	}
	closed := build(false)
	cs, _ := sightAt(closed, centre, 1)
	if closed.sightShows(cs, pit) {
		t.Error("a cell beyond an entirely blocked ring is lit: the walk did not stop")
	}
	open := build(true)
	os, _ := sightAt(open, centre, 1)
	if !open.sightShows(os, cell{x: centre.x, y: centre.y + 1}) {
		t.Fatal("the lowered cell is not visible, so the second world does not keep its ring alive")
	}
	if !open.sightShows(os, pit) {
		t.Error("the pit is dark even with the ring kept alive, so the first case proves nothing")
	}
}

func TestNoVisibleCellHasABlockedPredecessorOnShippedAltitudes(t *testing.T) {
	t.Parallel()

	b := Bounds{Width: 29, Height: 29}
	centre := cell{x: 14, y: 14}
	for _, obs := range []byte{0, 40, 127} {
		for _, wall := range []byte{0, 1, 63, 127} {
			for _, step := range []int32{1, 2, 3} {
				h := sightPlane(b, 0)
				for y := int32(0); y < b.Height; y++ {
					for x := int32(0); x < b.Width; x++ {
						if (x+y)%(step+1) == 0 {
							h[int(y)*int(b.Width)+int(x)] = wall
						}
					}
				}
				h[int(centre.y)*int(b.Width)+int(centre.x)] = obs
				w := sightWorld(t, b, Terrain{Height: h})
				stamp, _ := sightAt(w, centre, 6)
				for dx := int32(-sightHalf); dx <= sightHalf; dx++ {
					for dy := int32(-sightHalf); dy <= sightHalf; dy++ {
						if dx == 0 && dy == 0 {
							continue
						}
						if !w.sightShows(stamp, cell{x: centre.x + dx, y: centre.y + dy}) {
							continue
						}
						o := sightOffset(dx, dy)
						px := dx + int32(sightTables.stepX[o])
						py := dy + int32(sightTables.stepY[o])
						if !w.sightShows(stamp, cell{x: centre.x + px, y: centre.y + py}) {
							t.Fatalf("observer %d, wall %d, step %d: (%d,%d) is lit and its "+
								"predecessor (%d,%d) is not", obs, wall, step, dx, dy, px, py)
						}
					}
				}
			}
		}
	}
}

// TestTheInsetStopsTheWalkAndTheObserverStillSeesItself is AC-8. A cell the grid
// closes to an air mover is the border ring on every map this tree loads: the
// walk never lights one, never carries a value through one, and an observer
// standing in one still finds its own cell.
func TestTheInsetStopsTheWalkAndTheObserverStillSeesItself(t *testing.T) {
	t.Parallel()

	b := Bounds{Width: 25, Height: 25}
	centre := cell{x: 12, y: 12}
	g := make([]byte, int(b.Width)*int(b.Height))
	for y := int32(0); y < b.Height; y++ {
		g[int(y)*int(b.Width)+14] = 1 << 1
	}
	w := sightWorld(t, b, Terrain{Block: g})
	stamp, _ := sightAt(w, centre, sightTestRange)
	for _, c := range []cell{{x: 14, y: 12}, {x: 15, y: 12}, {x: 16, y: 12}} {
		if w.sightShows(stamp, c) {
			t.Errorf("cell (%d,%d) is lit through a cell closed to an air mover", c.x, c.y)
		}
	}
	if !w.sightShows(stamp, cell{x: 13, y: 12}) {
		t.Error("the cell in front of the closed column is dark")
	}

	on := cell{x: 14, y: 12}
	stamp2, _ := sightAt(w, on, sightTestRange)
	if !w.sightShows(stamp2, on) {
		t.Error("an observer standing on a closed cell cannot see its own cell")
	}
}

// TestNothingWrapsAtTheEdgeOfAWideMap is AC-9, and it is the one place this build
// deliberately does not reproduce the law. The law compares its playable
// rectangle as bytes while indexing the planes in 32 bits, so on a 256-wide map a
// true column of -9 wraps to 247 and lights a cell on the far side. Nothing here
// wraps, and no cell outside the world is ever lit.
func TestNothingWrapsAtTheEdgeOfAWideMap(t *testing.T) {
	t.Parallel()

	b := Bounds{Width: 256, Height: 41}
	w := sightWorld(t, b, Terrain{})
	stamp, n := sightAt(w, cell{x: 2, y: 20}, 19)
	for y := int32(0); y < b.Height; y++ {
		for x := int32(200); x < b.Width; x++ {
			if w.sightShows(stamp, cell{x: x, y: y}) {
				t.Fatalf("an observer at column 2 lit (%d,%d) on a 256-wide map", x, y)
			}
		}
	}
	if n < 100 {
		t.Errorf("the march near the edge lit only %d cells", n)
	}
}

func TestTheSeedIdentityHoldsAtEveryShift(t *testing.T) {
	t.Parallel()

	for shift := int32(1); shift <= 8; shift++ {
		for r := int32(0); r <= 20; r++ {
			got := sightSeed(r<<8, shift)
			want := int32(1)<<(shift-1) + r<<shift
			if got != want {
				t.Errorf("sightSeed(%d<<8, %d) = %d, want %d", r, shift, got, want)
			}
		}
	}
}

// TestTheCostTableIsBoundedAtShift7 is the T1 task's own bound on the general
// buildSightTables(shift), independent of the package-level sightTables var:
// every non-centre offset costs 128..181 at shift 7, exactly 128 on the axes
// and 181 on the diagonals — the same figures TestTheStepCostIsTheRaysMeanStep
// pins for sightTables, measured here on a table this test builds itself so
// the parameterised builder is witnessed directly and not only through the one
// value the package happens to keep.
func TestTheCostTableIsBoundedAtShift7(t *testing.T) {
	t.Parallel()

	tb := buildSightTables(7)
	for dx := int32(-sightHalf); dx <= sightHalf; dx++ {
		for dy := int32(-sightHalf); dy <= sightHalf; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			c := tb.cost[sightOffset(dx, dy)]
			if c < 128 || c > 181 {
				t.Fatalf("offset (%d,%d) costs %d, want 128..181", dx, dy, c)
			}
			switch {
			case dx == 0 || dy == 0:
				if c != 128 {
					t.Errorf("axis offset (%d,%d) costs %d, want 128", dx, dy, c)
				}
			case dx == dy || dx == -dy:
				if c != 181 {
					t.Errorf("diagonal offset (%d,%d) costs %d, want 181", dx, dy, c)
				}
			}
		}
	}
}

func TestTheExportedSightMatchesTheFlatGroundDiscThroughAnEntity(t *testing.T) {
	t.Parallel()

	centre := cell{x: 22, y: 22}
	for _, tc := range []struct {
		r    uint8
		want int
	}{{1, 9}, {2, 21}, {3, 45}, {4, 69}, {5, 105}, {6, 145}} {
		w, err := NewTerrainWorld(1, sightBounds, ModeCanonical, Terrain{}, []Entity{
			{ID: 1, X: centre.x, Y: centre.y, Owner: 1, ScanRange: tc.r},
		}, nil)
		if err != nil {
			t.Fatalf("NewTerrainWorld: %v", err)
		}
		stamp := w.Sight(1)
		n := 0
		for _, v := range stamp {
			if v != 0 {
				n++
			}
		}
		if n != tc.want {
			t.Errorf("Sight(1) for a range-%d entity lights %d cell(s), want %d", tc.r, n, tc.want)
		}
	}
}

// borderGrid marks blockAir on every cell within inset of an edge, the shape
// map load actually writes (aiInset's own comment) — so a test that wants the
// AI reader's rectangle to behave as it does on a loaded map, rather than as
// the unrestricted grid a bare Terrain{} carries, has to build it explicitly.
func borderGrid(b Bounds, inset int32) []byte {
	g := make([]byte, int(b.Width)*int(b.Height))
	for y := int32(0); y < b.Height; y++ {
		for x := int32(0); x < b.Width; x++ {
			if x < inset || x >= b.Width-inset || y < inset || y >= b.Height-inset {
				g[int(y)*int(b.Width)+int(x)] = blockAir
			}
		}
	}
	return g
}

// TestTheTwoRectanglesAgreeAwayFromTheEdgeAndDisagreeInsideIt is AC-3. On a
// world whose grid carries the AI's own 8-cell border (borderGrid), the fog
// reader's computed 7-cell inset differs from it only at the single column
// or row between the two — 7 itself, and its mirror W-8 / H-8. A march
// only ever reaches that column when its own ring bound, 19, closes the gap
// to an observer's distance from the edge: at 27 the closest ring cell lands
// on column 8, which both readers admit; at 26 it lands on column 7, which
// only the fog reader does.
func TestTheTwoRectanglesAgreeAwayFromTheEdgeAndDisagreeInsideIt(t *testing.T) {
	t.Parallel()

	b := Bounds{Width: 100, Height: 60}
	w := sightWorld(t, b, Terrain{Block: borderGrid(b, 8)})

	stampsAgree := func(x int32) bool {
		from := cell{x: x, y: 30}
		ai := make([]byte, len(w.grid))
		fog := make([]byte, len(w.grid))
		w.marchSight(aiSight, ai, from, 19)
		w.marchSight(fogSight, fog, from, 19)
		return bytes.Equal(ai, fog)
	}

	if !stampsAgree(27) {
		t.Error("an observer 27 cells from every edge disagrees between the two readers, want agreement")
	}
	if stampsAgree(26) {
		t.Error("an observer 26 cells from the left edge agrees between the two readers, want a difference")
	}
}

// sightGoldenWorld is the fixed synthetic world
// TestTheAIReaderStampIsUnchangedByThisStory marches: relief under one member,
// a loaded-map-shaped 8-cell border, and three members of different ranges —
// everything the AI reader's march touches. It is named so the golden test
// alone builds it, since nothing else in this file needs a world this
// specific.
func sightGoldenWorld(t *testing.T) (*World, []int) {
	t.Helper()

	b := Bounds{Width: 45, Height: 45}
	h := sightPlane(b, 0)
	for y := int32(0); y < b.Height; y++ {
		h[int(y)*int(b.Width)+30] = 80
	}
	h[22*int(b.Width)+15] = 40
	ents := []Entity{
		{ID: 1, X: 15, Y: 22, ScanRange: 6},
		{ID: 2, X: 25, Y: 10, ScanRange: 3},
		{ID: 3, X: 20, Y: 35, ScanRange: 9},
	}
	w, err := NewTerrainWorld(1, b, ModeCanonical, Terrain{Height: h, Block: borderGrid(b, 8)}, ents, nil)
	if err != nil {
		t.Fatalf("NewTerrainWorld: %v", err)
	}
	return w, []int{0, 1, 2}
}

func TestTheAIReaderStampIsUnchangedByThisStory(t *testing.T) {
	t.Parallel()

	w, members := sightGoldenWorld(t)
	stamp := w.groupSight(aiSight, members)
	n := 0
	for _, v := range stamp {
		if v != 0 {
			n++
		}
	}
	sum := sha256.Sum256(stamp)
	const wantHex = "9e0b5778d752c4c0d87253c13c06088f60c71969e01498f2a34dd5624ba66bb9"
	const wantN = 435
	if got := hex.EncodeToString(sum[:]); got != wantHex {
		t.Errorf("the AI reader's stamp hashes to %s (lit=%d), want %s (lit=%d) — captured before T1",
			got, n, wantHex, wantN)
	}
	if n != wantN {
		t.Errorf("the AI reader's stamp lights %d cell(s), want %d — captured before T1", n, wantN)
	}
}
