package terrain_test

import (
	"testing"

	"againrom/pkg/render/terrain"
)

// TestRateOfRoundTripsEveryRateInRange — 0060 SC-3: the whole domain,
// walked, not argued.
//
// RateOf(RatePeriod(r)) == ClampRate(r) is NOT a general fact about truncating
// division composed with itself: floor(N/floor(N/r)) can exceed r once r grows
// past sqrt(N), and the readout's rate line rests on it holding for every rate
// this project can reach. sqrt(1000000) is 1000 and the ceiling is 1024, so the
// range ends 24 rates INSIDE the region where the identity stops being
// automatic. That is close enough that the only honest check is the exhaustive
// one, and it costs 1024 divisions.
func TestRateOfRoundTripsEveryRateInRange(t *testing.T) {
	for r := terrain.RateMin; r <= terrain.RateMax; r++ {
		if got := terrain.RateOf(terrain.RatePeriod(r)); got != r {
			t.Errorf("RateOf(RatePeriod(%d)) = %d, want %d (period %d us)",
				r, got, r, terrain.RatePeriod(r))
		}
	}
}

// TestRateOfIsTotalPastBothEnds — 0060 SC-3: an input outside the range is
// brought inside rather than refused, on the same clamp its forward twin uses,
// and a period of zero or below is the shortest a clock will hold rather than a
// division by zero.
func TestRateOfIsTotalPastBothEnds(t *testing.T) {
	for _, c := range []struct{ rate, want int }{
		{-5, terrain.RateMin}, {0, terrain.RateMin},
		{2048, terrain.RateMax}, {5000, terrain.RateMax}, {1 << 20, terrain.RateMax},
	} {
		if got := terrain.RateOf(terrain.RatePeriod(c.rate)); got != c.want {
			t.Errorf("RateOf(RatePeriod(%d)) = %d, want the clamped %d", c.rate, got, c.want)
		}
	}

	// A period no RatePeriod produced. Zero and negative are the two a caller
	// computing one some other way can arrive at, and neither may panic.
	for _, p := range []int{-1, 0, 1} {
		if got := terrain.RateOf(p); got != terrain.RateMax {
			t.Errorf("RateOf(%d us) = %d, want the ceiling %d", p, got, terrain.RateMax)
		}
	}
	if got := terrain.RateOf(1 << 30); got != terrain.RateMin {
		t.Errorf("RateOf(a period of 1073 seconds) = %d, want the floor %d", got, terrain.RateMin)
	}
}

func TestRateOfReadsTheGamesOwnSpeedPeriods(t *testing.T) {
	for i := terrain.SpeedIndexMin; i <= terrain.SpeedIndexMax; i++ {
		want := terrain.TicksPerSecond(i)
		if got := terrain.RateOf(terrain.SpeedIndexPeriod(i)); got != want {
			t.Errorf("RateOf(SpeedIndexPeriod(%d) = %d us) = %d, want the table's %d",
				i, terrain.SpeedIndexPeriod(i), got, want)
		}
	}

	// The two periods that share a stated rate and are different numbers. This
	// is the fact the readout draws BOTH quantities for: a rate line alone
	// cannot tell these two clocks apart.
	if terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex) == terrain.RatePeriod(16) {
		t.Fatal("the game's map-load period and our rate model's 16/s period are the same number; " +
			"the readout's period line was justified by their being different")
	}
	if a, b := terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex), terrain.RatePeriod(16); a != 62_000 || b != 62_500 {
		t.Errorf("the two periods are %d and %d us, want 62000 (the game's 62 ms) and 62500 (1000000/16)", a, b)
	}
}
