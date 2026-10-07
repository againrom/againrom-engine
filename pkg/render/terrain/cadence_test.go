package terrain_test

import (
	"testing"

	"againrom/pkg/render/terrain"
)

// theLadder is the whole ladder, by hand: seventeen periods in microseconds,
// slowest first, and whether each is one of the game's own nine settings.
//
// The three at the top and the five at the bottom are OURS — the halvings below
// the shipped table's slowest speed and the doublings above its fastest. The
// nine between them are the game's, and 62000 rather than 62500 at rung 7 is the
// whole point of the story: that is the shipped 1000/16 = 62 ms widened, the
// truncation the game itself performs.
var theLadder = []struct {
	rung    int
	us      int
	rate    int
	shipped int // the speed index, or -1 for a rung of our extension
}{
	{0, 1_000_000, 1, -1},
	{1, 500_000, 2, -1},
	{2, 250_000, 4, -1},
	{3, 125_000, 8, 0},
	{4, 100_000, 10, 1},
	{5, 83_000, 12, 2},
	{6, 71_000, 14, 3},
	{7, 62_000, 16, 4},
	{8, 50_000, 20, 5},
	{9, 41_000, 24, 6},
	{10, 35_000, 28, 7},
	{11, 31_000, 32, 8},
	{12, 15_625, 64, -1},
	{13, 7_812, 128, -1},
	{14, 3_906, 256, -1},
	{15, 1_953, 512, -1},
	{16, 976, 1024, -1},
}

// TestTheLadderIsTheShippedTableWithTwoDisclosedExtensions — 0061 SC-1
// (AC-1, AC-2): every rung, its period, and which side of the shipped set it
// is on.
//
// The bounds are asserted from the table's own length rather than restated, so a
// rung added or removed anywhere fails here rather than shifting the meaning of
// every number below it.
func TestTheLadderIsTheShippedTableWithTwoDisclosedExtensions(t *testing.T) {
	if got, want := terrain.CadenceRungMax-terrain.CadenceRungMin+1, len(theLadder); got != want {
		t.Fatalf("the ladder has %d rungs, want the %d this file is written over", got, want)
	}
	if terrain.CadenceShippedLo != 3 || terrain.CadenceShippedHi != 11 {
		t.Fatalf("the shipped span is rungs %d..%d, want 3..11",
			terrain.CadenceShippedLo, terrain.CadenceShippedHi)
	}

	for _, c := range theLadder {
		if got := terrain.CadencePeriod(c.rung); got != c.us {
			t.Errorf("rung %d is %d us, want %d", c.rung, got, c.us)
		}
		// The rate line the readout already draws must still read this period
		// back as the whole number of ticks a second the rung names.
		if got := terrain.RateOf(c.us); got != c.rate {
			t.Errorf("rung %d (%d us) reads back as %d ticks a second, want %d", c.rung, c.us, got, c.rate)
		}

		idx, ok := terrain.SpeedIndexOf(c.us)
		if c.shipped < 0 {
			if ok {
				t.Errorf("rung %d (%d us) is reported as the game's speed %d; it is OUR extension",
					c.rung, c.us, idx)
			}
			continue
		}
		if !ok || idx != c.shipped {
			t.Errorf("rung %d (%d us) is reported as speed (%d, %v), want (%d, true)",
				c.rung, c.us, idx, ok, c.shipped)
		}
		// ...and the shipped rungs are the game's own periods, not recomputed.
		if want := terrain.SpeedIndexPeriod(c.shipped); c.us != want {
			t.Errorf("rung %d is %d us, want speed index %d's own %d us", c.rung, c.us, c.shipped, want)
		}
	}
}

// TestTheLadderIsStrictlyDecreasingAndEndsAtTheRateBounds — 0061 SC-1
// (AC-2).
//
// Strictly decreasing is what makes the inverse exact, so it is asserted rather
// than assumed. The two ends being RateMin's and RateMax's own periods is the
// claim that this ladder EXTENDS the range 0041 chose instead of inventing a new
// one — nothing beyond the shipped table's ends is reachable here that was not
// reachable before.
func TestTheLadderIsStrictlyDecreasingAndEndsAtTheRateBounds(t *testing.T) {
	for r := terrain.CadenceRungMin; r < terrain.CadenceRungMax; r++ {
		if a, b := terrain.CadencePeriod(r), terrain.CadencePeriod(r+1); a <= b {
			t.Fatalf("rung %d is %d us and rung %d is %d us; the ladder must strictly quicken", r, a, r+1, b)
		}
	}
	if got, want := terrain.CadencePeriod(terrain.CadenceRungMin), terrain.RatePeriod(terrain.RateMin); got != want {
		t.Errorf("the ladder's slow end is %d us, want RateMin's %d us", got, want)
	}
	if got, want := terrain.CadencePeriod(terrain.CadenceRungMax), terrain.RatePeriod(terrain.RateMax); got != want {
		t.Errorf("the ladder's fast end is %d us, want RateMax's %d us", got, want)
	}
	// The extension is the doubling 0041's ladder had, CONTINUED past the ends of
	// the shipped set rather than started from a rate of ours: each span is
	// checked across its join with the table, so an extension that began at some
	// other rate fails here. It is asserted on the RATE and not on the period —
	// our own periods are a truncated microsecond quotient, so 1000000/128 is
	// 7812 and twice that is 15624 rather than rung 12's 15625.
	for r := terrain.CadenceRungMin; r < terrain.CadenceShippedLo; r++ {
		a, b := terrain.RateOf(terrain.CadencePeriod(r)), terrain.RateOf(terrain.CadencePeriod(r+1))
		if 2*a != b {
			t.Errorf("below the table, rung %d runs at %d/s and rung %d at %d/s, want a doubling", r, a, r+1, b)
		}
	}
	for r := terrain.CadenceShippedHi; r < terrain.CadenceRungMax; r++ {
		a, b := terrain.RateOf(terrain.CadencePeriod(r)), terrain.RateOf(terrain.CadencePeriod(r+1))
		if 2*a != b {
			t.Errorf("above the table, rung %d runs at %d/s and rung %d at %d/s, want a doubling", r, a, r+1, b)
		}
	}
}

// TestCadenceRungRoundTripsEveryRung — 0061 SC-2 (AC-3): the whole domain,
// not a sample. Seventeen rungs is small enough to walk, and the property is
// not a general fact about the two functions — it holds because the ladder
// is strictly decreasing — so it is measured.
func TestCadenceRungRoundTripsEveryRung(t *testing.T) {
	for r := terrain.CadenceRungMin; r <= terrain.CadenceRungMax; r++ {
		if got := terrain.CadenceRung(terrain.CadencePeriod(r)); got != r {
			t.Errorf("CadenceRung(CadencePeriod(%d) = %d us) = %d, want %d",
				r, terrain.CadencePeriod(r), got, r)
		}
	}
}

func TestCadenceRungIsTotalPastBothEnds(t *testing.T) {
	for _, c := range []struct{ rung, want int }{
		{-1, terrain.CadenceRungMin}, {-1 << 20, terrain.CadenceRungMin},
		{terrain.CadenceRungMax + 1, terrain.CadenceRungMax}, {1 << 20, terrain.CadenceRungMax},
	} {
		if got := terrain.CadencePeriod(c.rung); got != terrain.CadencePeriod(c.want) {
			t.Errorf("CadencePeriod(%d) = %d us, want the clamped rung %d's %d us",
				c.rung, got, c.want, terrain.CadencePeriod(c.want))
		}
	}
	// Slower than the whole ladder, and faster than all of it. Zero and negative
	// are the two a caller can reach by arithmetic rather than by choice.
	for _, p := range []int{1 << 30, 2_000_000, 1_000_001} {
		if got := terrain.CadenceRung(p); got != terrain.CadenceRungMin {
			t.Errorf("CadenceRung(%d us) = %d, want the slow end %d", p, got, terrain.CadenceRungMin)
		}
	}
	for _, p := range []int{975, 1, 0, -7} {
		if got := terrain.CadenceRung(p); got != terrain.CadenceRungMax {
			t.Errorf("CadenceRung(%d us) = %d, want the fast end %d", p, got, terrain.CadenceRungMax)
		}
	}
	// A period BETWEEN two rungs answers with the first rung at least as fast as
	// it, and is not thereby claimed to be that rung: 62500 is our old rate
	// model's 16/s, and it is on no rung at all.
	if got := terrain.CadenceRung(62_500); got != 7 {
		t.Errorf("CadenceRung(62500 us) = %d, want 7", got)
	}
	if _, ok := terrain.SpeedIndexOf(62_500); ok {
		t.Error("62500 us is reported as one of the game's own speeds; it is our rate model's 16/s and no rung")
	}
}

// TestTheCadenceKeysRoundTripExactly — 0061 SC-2 (AC-3). THE trap the
// story was opened on: `+` n times and `-` n times must come back to where
// they started.
//
// Exhaustive over EVERY start rung and every press count up to four past the
// ladder's whole length, and the expected value is the closed form rather than a
// re-run of the code under test. Two things are asserted, and the second is why
// the first can be stated without a caveat:
//
//   - for every start and every n whose run does not leave the ladder, the round
//     trip is the identity. This is what the owner reproduces.
//   - where it DOES leave the ladder, the ladder SATURATES: presses past an end
//     move nothing and are not remembered, so the return lands n rungs in from
//     that end. No clamped ladder can be invertible for a run that saturates —
//     the map is not injective — and the alternative, remembering the presses
//     that did nothing, makes a key press that visibly does nothing for a while
//     afterwards. So the behaviour is stated and pinned here rather than left to
//     be discovered at an end.
func TestTheCadenceKeysRoundTripExactly(t *testing.T) {
	span := terrain.CadenceRungMax - terrain.CadenceRungMin + 1
	for start := terrain.CadenceRungMin; start <= terrain.CadenceRungMax; start++ {
		for n := 0; n <= span+4; n++ {
			rung := start
			for i := 0; i < n; i++ {
				rung = terrain.ClampCadenceRung(rung + 1)
			}
			up := rung
			for i := 0; i < n; i++ {
				rung = terrain.ClampCadenceRung(rung - 1)
			}

			wantUp := start + n
			if wantUp > terrain.CadenceRungMax {
				wantUp = terrain.CadenceRungMax
			}
			wantBack := wantUp - n
			if wantBack < terrain.CadenceRungMin {
				wantBack = terrain.CadenceRungMin
			}
			if up != wantUp || rung != wantBack {
				t.Fatalf("from rung %d, %d presses of + reached %d then %d presses of - reached %d, "+
					"want %d then %d", start, n, up, n, rung, wantUp, wantBack)
			}
			if start+n <= terrain.CadenceRungMax && rung != start {
				t.Fatalf("from rung %d, %d presses each way came back to %d — a run that never left the "+
					"ladder must be the identity", start, n, rung)
			}
		}
	}
}

// TestTheMapLoadCadenceIsOnTheLadder — 0061 SC-1 (AC-1, AC-6): the defect
// this story was opened on, stated as a property.
//
// A map opens at the game's own speed index 4, and 62000 us must BE a rung — so
// the ladder can return to it — while never becoming 62500, which is what our
// rate model would have made of the same 16 ticks a second. The 62 ms truncation
// is the game's own and the 992 ms water cycle follows from it.
func TestTheMapLoadCadenceIsOnTheLadder(t *testing.T) {
	open := terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex)
	if open != 62_000 {
		t.Fatalf("the map-load period is %d us, want the game's own 62000", open)
	}
	rung := terrain.CadenceRung(open)
	if terrain.CadencePeriod(rung) != open {
		t.Fatalf("the map-load period %d us is not on the ladder: rung %d is %d us",
			open, rung, terrain.CadencePeriod(rung))
	}
	if idx, ok := terrain.SpeedIndexOf(open); !ok || idx != terrain.DefaultSpeedIndex {
		t.Errorf("the map-load period is reported as speed (%d, %v), want (%d, true)",
			idx, ok, terrain.DefaultSpeedIndex)
	}
	if terrain.CadencePeriod(rung) == terrain.RatePeriod(terrain.TicksPerSecond(terrain.DefaultSpeedIndex)) {
		t.Error("the map-load rung is our rate model's period for 16/s; the game's truncated 62 ms was lost")
	}
	// Every rung is reachable from it in both directions, which is the whole of
	// what "the opening cadence lies ON the ladder" buys.
	for r := terrain.CadenceRungMin; r <= terrain.CadenceRungMax; r++ {
		back := rung
		step := 1
		if r < rung {
			step = -1
		}
		for i := 0; i < span(rung, r); i++ {
			back = terrain.ClampCadenceRung(back + step)
		}
		if back != r {
			t.Fatalf("rung %d is not reachable from the map-load rung %d", r, rung)
		}
	}
}

func span(a, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}
