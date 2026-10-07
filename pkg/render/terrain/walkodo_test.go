package terrain_test

import (
	"math"
	"testing"

	"againrom/pkg/render/terrain"
)

// engineShares is the decoded arm, spelled out: the delta still owed divided by
// the ticks still owed, truncating, subtracted, once per tick. It is the thing
// WalkAdvance's closed form claims to be, and it is written here as a loop over
// state precisely because the closed form is neither.
func engineShares(delta, span int) []int {
	if delta < 0 {
		delta = -delta
	}
	owed := delta * 256
	out := make([]int, span)
	for k := 0; k < span; k++ {
		s := owed / (span - k)
		out[k] = s
		owed -= s
	}
	return out
}

// TestWalkAdvanceIsTheEngineRecurrence covers SC-1: for every span the rate law
// can produce and then some, and for a straight, a diagonal and a multi-cell
// delta, the advance equals the euclidean length of the recurrence's own
// shares, tick for tick.
//
// The range runs past 256 — the longest crossing the rate law admits — so that
// the shares of one and of zero, which is where the diagonal degenerates, are
// exercised rather than assumed unreachable.
func TestWalkAdvanceIsTheEngineRecurrence(t *testing.T) {
	deltas := [][2]int{{1, 0}, {0, 1}, {1, 1}, {-1, 1}, {2, 0}, {2, 3}}
	for span := 1; span <= 512; span++ {
		for _, d := range deltas {
			sx := engineShares(d[0], span)
			sy := engineShares(d[1], span)
			for k := 0; k < span; k++ {
				want := int(math.Sqrt(float64(sx[k]*sx[k] + sy[k]*sy[k])))
				if got := terrain.WalkAdvance(d[0], d[1], span, k); got != want {
					t.Fatalf("WalkAdvance(%d, %d, %d, %d) = %d, the recurrence gives %d",
						d[0], d[1], span, k, got, want)
				}
			}
		}
	}
}

// TestWalkAdvanceStraightCellIsSixteenSteps covers SC-2 — the story's headline
// invariant. Over every span, one straight cell's shares sum to exactly one
// cell and the timeline advances by exactly sixteen steps: no speed term, no
// residue, no rounding.
//
// The 256 and the 16 are the contract's own numbers, not readings off the
// package.
func TestWalkAdvanceStraightCellIsSixteenSteps(t *testing.T) {
	for span := 1; span <= 512; span++ {
		for _, d := range [][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}} {
			total := 0
			for k := 0; k < span; k++ {
				total += terrain.WalkAdvance(d[0], d[1], span, k)
			}
			if total != 256 {
				t.Fatalf("straight %v over %d ticks travelled %d, want 256", d, span, total)
			}
			if got := terrain.WalkPhase(total); got != 16 {
				t.Fatalf("straight %v over %d ticks advanced %d steps, want 16", d, span, got)
			}
		}
	}
}

// TestWalkAdvanceDiagonalCosts covers SC-3: a diagonal cell is never cheaper
// than a straight one and never dearer than its own exact euclidean length,
// and at the slowest crossings the per-tick truncation closes the two together.
//
// 362 is trunc(256*sqrt2), the exact diagonal a crossing of one tick pays.
func TestWalkAdvanceDiagonalCosts(t *testing.T) {
	degenerate := 0
	for span := 1; span <= 512; span++ {
		total := 0
		for k := 0; k < span; k++ {
			total += terrain.WalkAdvance(1, 1, span, k)
		}
		if total < 256 || total > 362 {
			t.Fatalf("diagonal over %d ticks travelled %d, want within [256, 362]", span, total)
		}
		if total == 256 {
			degenerate++
		}
	}
	if degenerate == 0 {
		t.Error("no span makes the diagonal cost exactly a straight cell; AC-3 says the slowest ones do")
	}
}

// TestWalkAdvanceWorkedCells pins three crossings as hand-computed literals, so
// that a change to the split shows up as a number and not only as a property.
//
//	span 16, straight: sixteen shares of 16, total 256, phase 16.
//	span 16, diagonal: sixteen shares of (16,16); trunc(sqrt(512)) = 22 each,
//	                   total 352, phase 22.
//	span  3, straight: 256 = 3*85 + 1, so shares 85, 85, 86 — the surplus LAST.
func TestWalkAdvanceWorkedCells(t *testing.T) {
	for k := 0; k < 16; k++ {
		if got := terrain.WalkAdvance(1, 0, 16, k); got != 16 {
			t.Errorf("span 16 straight tick %d = %d, want 16", k, got)
		}
		if got := terrain.WalkAdvance(1, 1, 16, k); got != 22 {
			t.Errorf("span 16 diagonal tick %d = %d, want 22", k, got)
		}
	}
	want := [3]int{85, 85, 86}
	for k, w := range want {
		if got := terrain.WalkAdvance(1, 0, 3, k); got != w {
			t.Errorf("span 3 straight tick %d = %d, want %d", k, got, w)
		}
	}
}

func TestWalkPhaseRoundsDown(t *testing.T) {
	cases := []struct{ odo, want int }{
		{0, 0}, {15, 0}, {16, 1}, {31, 1}, {256, 16}, {362, 22},
		{-1, -1}, {-16, -1}, {-17, -2},
	}
	for _, c := range cases {
		if got := terrain.WalkPhase(c.odo); got != c.want {
			t.Errorf("WalkPhase(%d) = %d, want %d", c.odo, got, c.want)
		}
	}
}

func TestWalkAdvanceIsTotal(t *testing.T) {
	if got := terrain.WalkAdvance(0, 0, 16, 3); got != 0 {
		t.Errorf("a mover that did not move travelled %d, want 0", got)
	}
	// A span of zero or below is one tick, which pays the whole cell at once.
	for _, span := range []int{0, -1, -1000} {
		if got := terrain.WalkAdvance(1, 0, span, 0); got != 256 {
			t.Errorf("span %d = %d, want the whole cell 256", span, got)
		}
	}
	// A tick before the crossing takes its first share, one past its end the
	// last: 256 over 3 ticks is 85, 85, 86.
	if got := terrain.WalkAdvance(1, 0, 3, -9); got != 85 {
		t.Errorf("tick -9 = %d, want the first share 85", got)
	}
	if got := terrain.WalkAdvance(1, 0, 3, 99); got != 86 {
		t.Errorf("tick 99 = %d, want the last share 86", got)
	}
	// A delta far past any map neither panics nor overflows into a negative.
	if got := terrain.WalkAdvance(1<<40, 1<<40, 7, 3); got <= 0 {
		t.Errorf("an absurd delta answered %d, want a positive distance", got)
	}
}
