package sim

// The rate law as arithmetic, with no world anywhere in this file.
//
// Every case here drives rateOf and transitOf directly, which is the point of
// their signatures: the two terrain terms this tree has no plane for are
// PARAMETERS, so they can be exercised at values the production caller cannot
// supply. A test that could only reach the law through a world would be able to
// witness neither of them, and the fork on the movement domain would be
// invisible as well, since at the mean cost a world here produces the two arms
// compute one number.
//
// This file names a floating-point type, deliberately and in exactly one test.
// The determinism scan reads pkg/sim's PRODUCTION sources — it skips _test.go —
// and the import rule holds this package's tests to the standard library, which
// math is. That is what lets the diagonal's exactness be PROVED against the
// original's own shipped constant rather than asserted about it.

import (
	"math"
	"testing"
)

// TestTheWorkedExampleReproduces is AC-1: the one case the decoded law states
// end to end, and the cheapest thing in this file to get wrong.
//
// Speed 16, both cost bytes 8, level ground: the multiplier takes it to 128, the
// tilt does nothing, the mean cost is 8, and the divide brings it back to 16 —
// so a straight cell takes sixteen ticks. Sixteen ticks is one of the
// original's full ticks, which is the sense in which its default unit crosses
// one cell a second.
func TestTheWorkedExampleReproduces(t *testing.T) {
	v := rateOf(DomainGround, 16, 8, 8, 0, 0)
	if v != 16 {
		t.Fatalf("rate = %d, want 16", v)
	}
	if got := transitOf(v, false); got != 16 {
		t.Fatalf("straight transit = %d, want 16", got)
	}
}

// TestTheGroundArmIsTheOnlyOneThatReadsTheGround is AC-2.
//
// At the mean cost this tree's absent plane produces, all three domains agree —
// which is exactly why the other two blocks exist. Moving the mean cost or the
// ground under them must move the ground mover and nothing else; a fork wired
// the wrong way round would pass the first block and fail both others, and a
// fork deleted altogether would pass the first and fail both others too.
func TestTheGroundArmIsTheOnlyOneThatReadsTheGround(t *testing.T) {
	const speed = 20
	domains := []Domain{DomainGround, DomainGhost, DomainAir}

	for _, d := range domains {
		if v := rateOf(d, speed, 8, 8, 0, 0); v != speed {
			t.Errorf("domain %d at mean cost 8 on the level: rate = %d, want %d", d, v, speed)
		}
	}

	// Cheap ground and costly ground, and a slope in each direction. The ground
	// mover must move off its raw speed on all four; the other two must not
	// budge on any of them.
	cases := []struct {
		name             string
		costSrc, costDst uint8
		hSrc, hDst       uint8
		wantGround       int32
	}{
		{"cheap ground", 6, 6, 0, 0, 26},    // 160/6 = 26 (toward zero)
		{"costly ground", 16, 16, 0, 0, 10}, // 160/16
		{"downhill", 8, 8, 32, 0, 30},       // 160 + (160*32>>6) = 240, /8
		{"uphill", 8, 8, 0, 32, 10},         // 160 + (160*-32>>6) = 80, /8
	}
	for _, c := range cases {
		if v := rateOf(DomainGround, speed, c.costSrc, c.costDst, c.hSrc, c.hDst); v != c.wantGround {
			t.Errorf("%s: ground rate = %d, want %d", c.name, v, c.wantGround)
		}
		for _, d := range []Domain{DomainGhost, DomainAir} {
			if v := rateOf(d, speed, c.costSrc, c.costDst, c.hSrc, c.hDst); v != speed {
				t.Errorf("%s: domain %d rate = %d, want the raw speed %d", c.name, d, v, speed)
			}
		}
	}
}

// TestTheGroundArmsSaturationsAndSubstitutions is AC-3: every place the ground
// arm refuses to be linear.
func TestTheGroundArmsSaturationsAndSubstitutions(t *testing.T) {
	const speed = 16 // 128 after the multiplier

	// The tilt saturates at plus and minus slopeLimit, so a steeper drop or
	// climb than that changes nothing further. A difference of 100 and one of 32
	// must give the same answer, and so must -100 and -32; a difference past 127
	// wraps in the byte, which is the plane's own arithmetic and is measured
	// rather than avoided.
	if a, b := rateOf(DomainGround, speed, 8, 8, 100, 0), rateOf(DomainGround, speed, 8, 8, 32, 0); a != b {
		t.Errorf("downhill saturation: rate at +100 is %d, at +32 is %d, want equal", a, b)
	}
	if a, b := rateOf(DomainGround, speed, 8, 8, 0, 100), rateOf(DomainGround, speed, 8, 8, 0, 32); a != b {
		t.Errorf("uphill saturation: rate at -100 is %d, at -32 is %d, want equal", a, b)
	}

	// Uphill reduces and downhill increases, and the shift is arithmetic: the
	// two must fall either side of the level rate rather than both above it,
	// which is what a logical shift on the negative half would give.
	level := rateOf(DomainGround, speed, 8, 8, 0, 0)
	down := rateOf(DomainGround, speed, 8, 8, 32, 0)
	up := rateOf(DomainGround, speed, 8, 8, 0, 32)
	if !(up < level && level < down) {
		t.Errorf("uphill %d, level %d, downhill %d: want strictly increasing", up, level, down)
	}

	// THE SHIFT IS ARITHMETIC AND NOT A DIVIDE, which the ordering above cannot
	// see: the two agree wherever the quotient is exact, and every whole
	// direction step at this speed is exact. Here it is not — a rate of 8 tilted
	// by −1 gives −8 over 64, which an arithmetic shift floors to −1 and a
	// truncating divide takes to 0 — and the mean cost is 1, so the difference
	// survives the divide and the clamp instead of being swallowed by them.
	if v := rateOf(DomainGround, 1, 2, 0, 0, 1); v != 7 {
		t.Errorf("an inexact uphill tilt: rate = %d, want 7 — a truncating divide gives 8", v)
	}

	// The cost add is a BYTE add, and the case has to survive the divide to say
	// so. 200 + 100 wraps to 44, whose half is 22, and 160 over 22 is 7; added
	// wide it is 150, and 160 over that is 1 — two answers the clamp keeps apart.
	// (200 + 200 would wrap to 72 and come out 1 either way, which is a case that
	// measures nothing.)
	if v := rateOf(DomainGround, 20, 200, 100, 0, 0); v != 7 {
		t.Errorf("wrapping cost add: rate = %d, want 7 — an add wider than a byte gives 1", v)
	}

	// A mean of zero takes the substitute, which is what every cell in this tree
	// does: with no cost plane both bytes are 0 and the rate is the speed.
	if v := rateOf(DomainGround, speed, 0, 0, 0, 0); v != speed {
		t.Errorf("zero mean cost: rate = %d, want the substitute's %d", v, speed)
	}
	// And it is the MEAN that is tested, not the bytes: 1 and 0 halve to 0 too.
	if v := rateOf(DomainGround, speed, 1, 0, 0, 0); v != speed {
		t.Errorf("mean of 1 and 0: rate = %d, want the substitute's %d", v, speed)
	}

	// The clamp binds at both ends, and it binds ON its own boundary rather than
	// one past it: at a mean cost of 8 a speed of rateCeil comes out exactly at
	// the ceiling and one of rateCeil+1 exactly one over, so a comparison off by
	// one is a failure here and nowhere else.
	if v := rateOf(DomainGround, rateCeil, 8, 8, 0, 0); v != rateCeil {
		t.Errorf("a rate exactly at the ceiling came out %d, want %d", v, rateCeil)
	}
	if v := rateOf(DomainGround, rateCeil+1, 8, 8, 0, 0); v != rateCeil {
		t.Errorf("a rate one past the ceiling came out %d, want the ceiling %d", v, rateCeil)
	}
	if v := rateOf(DomainGround, rateFloor, 8, 8, 0, 0); v != rateFloor {
		t.Errorf("a rate exactly at the floor came out %d, want %d", v, rateFloor)
	}

	// And nothing outside it is reachable.
	if v := rateOf(DomainGround, 0, 8, 8, 0, 0); v != rateFloor {
		t.Errorf("speed 0: rate = %d, want the floor %d", v, rateFloor)
	}
	if v := rateOf(DomainGround, -50, 8, 8, 0, 0); v != rateFloor {
		t.Errorf("negative speed: rate = %d, want the floor %d", v, rateFloor)
	}
	if v := rateOf(DomainGround, 1000, 8, 8, 0, 0); v != rateCeil {
		t.Errorf("speed 1000: rate = %d, want the ceiling %d", v, rateCeil)
	}
	if v := rateOf(DomainAir, 1000, 0, 0, 0, 0); v != rateCeil {
		t.Errorf("air at speed 1000: rate = %d, want the ceiling %d", v, rateCeil)
	}
}

// diagonalConstant is the eight bytes the original holds its diagonal factor in,
// read as the double they are. They are quoted here as a bit pattern rather than
// as a decimal so that this test compares against the SHIPPED value and not
// against a transcription of it that could have been rounded on the way in.
const diagonalConstant = 0x3FE69FBE76C8B439

// TestTheDiagonalRatioIsExactAgainstTheShippedConstant is AC-4's first half, and
// the whole justification for doing this arithmetic in integers.
//
// It runs to 999 and stops there on purpose. Beyond that the argument the
// equality rests on — that 707v/1000 is never an integer, so the exact product
// is never within 1/1000 of one — no longer holds, and at v = 1000 exactly it
// fails. The clamp puts every reachable rate sixteen times below the bound.
func TestTheDiagonalRatioIsExactAgainstTheShippedConstant(t *testing.T) {
	k := math.Float64frombits(diagonalConstant)
	// The ratio this package divides by and the double the original multiplies
	// by are the SAME value: the shipped bytes are exactly the double that
	// diagNumerator/diagDenominator rounds to. Asserted from the package's own
	// two constants, so moving either one fails here first.
	if want := math.Float64bits(float64(diagNumerator) / float64(diagDenominator)); want != diagonalConstant {
		t.Fatalf("the shipped constant is %#016x; %d/%d rounds to %#016x",
			uint64(diagonalConstant), diagNumerator, diagDenominator, want)
	}
	for v := int32(0); v <= 999; v++ {
		want := int32(math.Trunc(float64(v) * k))
		if got := diagonalStep(v); got != want {
			t.Fatalf("diagonalStep(%d) = %d, the shipped constant truncates to %d", v, got, want)
		}
	}
	// The clamp's own endpoints, spelled out, so that a change to either
	// constant is a failure here and not merely a change in coverage.
	if got := diagonalStep(rateCeil); got != 44 {
		t.Errorf("diagonalStep at the ceiling = %d, want 44", got)
	}
	if got := diagonalStep(rateFloor); got != 0 {
		t.Errorf("diagonalStep at the floor = %d, want 0", got)
	}
}

// TestTheTransitCollapsesIntoTwentySevenClasses is AC-4's second half.
//
// The published enumeration of the law's own domain says the straight transit
// takes 27 distinct values over the whole clamped rate range and that the
// fastest class is twelve rates wide. Both numbers are reproduced here rather
// than quoted: a rounding down instead of up, or a grid of any other size, moves
// them.
func TestTheTransitCollapsesIntoTwentySevenClasses(t *testing.T) {
	seen := map[int32]bool{}
	for v := int32(rateFloor); v <= rateCeil; v++ {
		seen[transitOf(v, false)] = true
	}
	if len(seen) != 27 {
		t.Errorf("straight transit takes %d distinct values over [%d,%d], want 27", len(seen), rateFloor, rateCeil)
	}
	for v := int32(52); v <= 63; v++ {
		if got := transitOf(v, false); got != 5 {
			t.Errorf("transitOf(%d) = %d, want 5 — rates 52..63 are one class", v, got)
		}
	}
	if got := transitOf(51, false); got != 6 {
		t.Errorf("transitOf(51) = %d, want 6 — the class below must differ", got)
	}
	// The two ends of the range, and the fact that the transit is monotone in
	// the rate: a faster mover never takes longer.
	if got := transitOf(rateFloor, false); got != maxTransit {
		t.Errorf("transitOf at the floor = %d, want %d", got, maxTransit)
	}
	prev := int32(maxTransit + 1)
	for v := int32(rateFloor); v <= rateCeil; v++ {
		got := transitOf(v, false)
		if got > prev {
			t.Fatalf("transitOf(%d) = %d rose above transitOf(%d) = %d", v, got, v-1, prev)
		}
		prev = got
	}
}

// TestADiagonalCostsMoreThanAStraightStep is AC-4's third half and the one that
// catches the factor applied the wrong way up.
//
// A diagonal must never be quicker, and at the rate where the law's own divisor
// vanishes it must produce the longest transit rather than an expression with no
// value.
func TestADiagonalCostsMoreThanAStraightStep(t *testing.T) {
	for v := int32(rateFloor); v <= rateCeil; v++ {
		straight, diagonal := transitOf(v, false), transitOf(v, true)
		if diagonal < straight {
			t.Fatalf("rate %d: diagonal %d is quicker than straight %d", v, diagonal, straight)
		}
	}
	// The rate whose diagonal step truncates to nothing. The law divides the
	// grid by that step and so has no value here; ours takes the longest transit
	// the grid allows.
	if got := transitOf(1, true); got != maxTransit {
		t.Fatalf("the zero-step diagonal took %d ticks, want the longest transit %d", got, maxTransit)
	}
	// The worked example's own diagonal, so the number a reader can check by
	// hand is in the suite: rate 16 steps 11 a tick, and 256/11 rounds up to 24.
	if got := transitOf(16, true); got != 24 {
		t.Fatalf("the worked example's diagonal took %d ticks, want 24", got)
	}
}

// TestTheDiagonalStaysInsideItsPublishedBandExceptAtOneRate is the sharpest
// check in this file, because what it checks against is a number nothing here
// chose.
//
// The decoded law's own customisation limits state that over the SHIPPED speeds
// a diagonal transit falls between 0.97 and 1.15 times root-two times the
// straight one. That band is not a property of the arithmetic in general: over
// the whole of 8..35 exactly ONE rate falls outside it, and the extremes over the
// rest reproduce the two published bounds to two figures. So the band is a
// PREDICTION — the shipped alphabet cannot contain that one rate — and the
// developer run against a lawful install is where it is met.
//
// Nothing here reads an install. What is asserted is the arithmetic's own shape:
// which rate is the exception, that it is the only one, and that the extremes
// over the rest are the published pair. A wrong diagonal factor, a rounding down
// instead of up, or a grid of another size moves all three.
func TestTheDiagonalStaysInsideItsPublishedBandExceptAtOneRate(t *testing.T) {
	const lo, hi = 0.97, 1.15
	const exception = 23

	var outside []int32
	minR, maxR := math.Inf(1), math.Inf(-1)
	t.Logf("rate straight diagonal  ratio to root-two")
	for v := int32(8); v <= 35; v++ {
		s, d := transitOf(v, false), transitOf(v, true)
		r := float64(d) / (float64(s) * math.Sqrt2)
		t.Logf("%4d %8d %9d  %.4f", v, s, d, r)
		if r < lo || r > hi {
			outside = append(outside, v)
			continue
		}
		minR, maxR = math.Min(minR, r), math.Max(maxR, r)
	}

	if len(outside) != 1 || outside[0] != exception {
		t.Fatalf("the rates outside [%.2f, %.2f] over 8..35 are %v, want exactly [%d] — the published "+
			"band is a statement about the SHIPPED alphabet, and it holds only if that one is absent "+
			"from it", lo, hi, outside, exception)
	}
	if got := math.Round(minR*100) / 100; got != lo {
		t.Errorf("the smallest ratio over the rest is %.4f, which rounds to %.2f and not the published %.2f",
			minR, got, lo)
	}
	if got := math.Round(maxR*100) / 100; got != hi {
		t.Errorf("the largest ratio over the rest is %.4f, which rounds to %.2f and not the published %.2f",
			maxR, got, hi)
	}
}
