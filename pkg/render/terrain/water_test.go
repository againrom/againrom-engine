package terrain_test

import (
	"math/rand"
	"testing"

	"againrom/pkg/render/terrain"
)

// waterWord builds a type1 tile word with the given strip group, blend column
// and sub-cell, per the TERR-IDX-003 split g=(w&0x1fff)>>6, b=(w>>4)&3,
// sub=w&0xf. Built here from the documented field positions rather than taken
// from any game file.
func word(group, blend, sub int) uint16 {
	return uint16(group<<6) | uint16(blend<<4) | uint16(sub)
}

// wantPhase recomputes the spec's formula independently of the implementation.
func wantPhase(group, col, row int, ctr uint32) int {
	return (group + (col+1)*row + int(ctr>>2)) & 3
}

// TestWaterGroupClassification - AC-1 / SC-1: exactly groups 8..11 are water.
func TestWaterGroupClassification(t *testing.T) {
	for g := 0; g < 128; g++ {
		want := g >= 8 && g <= 11
		if got := terrain.IsWaterGroup(g); got != want {
			t.Fatalf("IsWaterGroup(%d) = %v, want %v", g, got, want)
		}
	}
	// Negative and far out-of-range groups are not water and do not panic.
	for _, g := range []int{-1, -12, 1 << 20} {
		if terrain.IsWaterGroup(g) {
			t.Fatalf("IsWaterGroup(%d) = true, want false", g)
		}
	}
}

func TestWaterPhaseFormula(t *testing.T) {
	cases := []struct {
		g, col, row int
		ctr         uint32
		want        int
	}{
		// (8 + 1*0 + 0) & 3 = 0
		{g: 8, col: 0, row: 0, ctr: 0, want: 0},
		// (9 + 1*0 + 0) & 3 = 1
		{g: 9, col: 0, row: 0, ctr: 0, want: 1},
		// (8 + 4*3 + 0) & 3 = 20 & 3 = 0
		{g: 8, col: 3, row: 3, ctr: 0, want: 0},
		// (8 + 3*2 + 0) & 3 = 14 & 3 = 2
		{g: 8, col: 2, row: 2, ctr: 0, want: 2},
		// the counter enters as ctr>>2: 7>>2 = 1
		{g: 8, col: 0, row: 0, ctr: 7, want: 1},
		// 8>>2 = 2
		{g: 8, col: 0, row: 0, ctr: 8, want: 2},
		// a full cycle of 16 ticks returns to the start
		{g: 8, col: 0, row: 0, ctr: 16, want: 0},
		// the far corner of the largest shipped map
		{g: 11, col: 255, row: 255, ctr: 0, want: wantPhase(11, 255, 255, 0)},
	}
	for _, c := range cases {
		if got := terrain.WaterPhase(c.g, c.col, c.row, c.ctr); got != c.want {
			t.Fatalf("WaterPhase(%d,%d,%d,%d) = %d, want %d", c.g, c.col, c.row, c.ctr, got, c.want)
		}
	}

	// Totality: negative coordinates, an extreme counter, and a group outside
	// the water range all still yield a phase in 0..3.
	hostile := []struct {
		g, col, row int
		ctr         uint32
	}{
		{-5, -7, -9, 0},
		{8, -1, 0, 0},
		{8, 0, -1, ^uint32(0)},
		{11, 1 << 20, 1 << 10, ^uint32(0)},
		{0, 0, 0, ^uint32(0)},
	}
	for _, h := range hostile {
		got := terrain.WaterPhase(h.g, h.col, h.row, h.ctr)
		if got < 0 || got >= terrain.WaterPhases {
			t.Fatalf("WaterPhase(%d,%d,%d,%d) = %d, want 0..3", h.g, h.col, h.row, h.ctr, got)
		}
	}
}

// TestResolveAnimatedWater - AC-2 / SC-3: a water word keeps its blend column
// and sub-cell and selects the strip group 8+phase.
func TestResolveAnimatedWater(t *testing.T) {
	for g := 8; g <= 11; g++ {
		for b := 0; b < 4; b++ {
			for _, sub := range []int{0, 5, 15} {
				for _, p := range []struct {
					col, row int
					ctr      uint32
				}{{0, 0, 0}, {3, 7, 9}, {255, 255, 4321}} {
					ref := terrain.ResolveAnimated(word(g, b, sub), p.col, p.row, p.ctr)
					phase := wantPhase(g, p.col, p.row, p.ctr)
					wantSlot := (8+phase)*4 + b
					if ref.Slot != wantSlot {
						t.Fatalf("g=%d b=%d at (%d,%d) ctr=%d: Slot = %d, want %d",
							g, b, p.col, p.row, p.ctr, ref.Slot, wantSlot)
					}
					if ref.Sub != sub {
						t.Fatalf("Sub = %d, want %d (the sub-cell must survive the override)", ref.Sub, sub)
					}
					if !ref.Water {
						t.Fatalf("g=%d: Water = false, want true", g)
					}
				}
			}
		}
	}
}

func TestWaterVariantReachability(t *testing.T) {
	var seen [16]bool
	for g := 8; g <= 11; g++ {
		for b := 0; b < 4; b++ {
			w := word(g, b, 0)
			// Sweep positions and counters wide enough to hit every phase.
			for row := 0; row < 8; row++ {
				for col := 0; col < 8; col++ {
					for ctr := uint32(0); ctr < 16; ctr++ {
						ref := terrain.ResolveAnimated(w, col, row, ctr)
						if ref.Slot < 0 || ref.Slot >= terrain.SlotCount {
							t.Fatalf("Slot = %d, want inside 0..%d", ref.Slot, terrain.SlotCount-1)
						}
						// tile3 occupies slots 32..47: V = slot - 32.
						v := ref.Slot - 32
						if v < 0 || v > 15 {
							t.Fatalf("g=%d b=%d at (%d,%d) ctr=%d: tile3 variant %d, want 0..15",
								g, b, col, row, ctr, v)
						}
						seen[v] = true
					}
				}
			}
		}
	}
	for v, ok := range seen {
		if !ok {
			t.Fatalf("tile3 variant %d never reached; the cycle must cover all 16", v)
		}
	}
}

func TestResolveAnimatedPassesNonWaterThrough(t *testing.T) {
	positions := []struct {
		col, row int
		ctr      uint32
	}{{0, 0, 0}, {1, 0, 3}, {0, 1, 4}, {17, 23, 999}, {-4, -6, ^uint32(0)}}

	for w := 0; w < 1<<16; w++ {
		tw := uint16(w)
		static := terrain.Resolve(tw)
		if static.Water {
			continue
		}
		for _, p := range positions {
			if got := terrain.ResolveAnimated(tw, p.col, p.row, p.ctr); got != static {
				t.Fatalf("word %#04x at (%d,%d) ctr=%d: animated %+v != static %+v",
					tw, p.col, p.row, p.ctr, got, static)
			}
		}
	}
}

func TestWaterCycleCadence(t *testing.T) {
	const g, b = 8, 2
	w := word(g, b, 0)
	const col, row = 6, 5

	slotAt := func(ctr uint32) int { return terrain.ResolveAnimated(w, col, row, ctr).Slot }

	base := slotAt(0)
	for ctr := uint32(0); ctr < 64; ctr++ {
		got := slotAt(ctr)

		// The low two counter bits are inert: the image only depends on ctr>>2.
		if want := slotAt(ctr &^ 3); got != want {
			t.Fatalf("ctr=%d: slot %d, want %d — the low 2 bits must not affect the image", ctr, got, want)
		}
		// Period 16 ticks.
		if want := slotAt(ctr % terrain.WaterCycleTicks); got != want {
			t.Fatalf("ctr=%d: slot %d, want %d — the cycle must repeat every %d ticks",
				ctr, got, want, terrain.WaterCycleTicks)
		}
		// The phase advances by one per variant, so the slot advances by one
		// blend row (4 slots) every 4 ticks, wrapping inside tile3.
		variant := int(ctr) / terrain.TicksPerVariant
		wantSlot := 32 + ((base-32)/4+variant)%terrain.WaterPhases*4 + b
		if got != wantSlot {
			t.Fatalf("ctr=%d: slot %d, want %d", ctr, got, wantSlot)
		}
	}

	// A variant boundary really is a change, and a non-boundary really is not.
	if slotAt(3) != slotAt(0) {
		t.Fatal("the image changed inside a 4-tick variant")
	}
	if slotAt(4) == slotAt(3) {
		t.Fatal("the image did not change at the 4-tick variant boundary")
	}
}

// TestAdjacentCellsRipple - AC-6 / SC-6: neighbouring water cells are offset
// against each other, which is what makes the surface ripple rather than pulse.
func TestAdjacentCellsRipple(t *testing.T) {
	const g = 8
	w := word(g, 0, 0)
	const col, row, ctr = 4, 3, 0

	here := terrain.WaterPhase(g, col, row, ctr)
	right := terrain.WaterPhase(g, col+1, row, ctr)
	down := terrain.WaterPhase(g, col, row+1, ctr)

	if here == right {
		t.Fatalf("(%d,%d) and its right neighbour share phase %d; the ripple term is missing", col, row, here)
	}
	if here == down {
		t.Fatalf("(%d,%d) and its lower neighbour share phase %d; the ripple term is missing", col, row, here)
	}

	// The offset is a property of position, not of the resolve path.
	if a, b := terrain.ResolveAnimated(w, col, row, ctr), terrain.ResolveAnimated(w, col+1, row, ctr); a.Slot == b.Slot {
		t.Fatalf("adjacent cells resolved to the same slot %d", a.Slot)
	}
}

func TestStaticEqualsPhaseZero(t *testing.T) {
	for w := 0; w < 1<<16; w++ {
		tw := uint16(w)
		static := terrain.Resolve(tw)
		if !static.Water {
			continue
		}
		// Choose a position/counter whose phase is 0 for this word's group:
		// row = 0 kills the position term, and ctr>>2 == (4-g&3)&3 cancels g.
		g := int(tw&0x1fff) >> 6
		ctr := uint32(((4 - g%4) % 4) * 4)
		got := terrain.ResolveAnimated(tw, 0, 0, ctr)
		if terrain.WaterPhase(g, 0, 0, ctr) != 0 {
			t.Fatalf("word %#04x: test setup failed to reach phase 0", tw)
		}
		if got != static {
			t.Fatalf("word %#04x: animated at phase 0 %+v != static %+v", tw, got, static)
		}
	}
}

// TestSpeedTableAndTickMillis - AC-7 / DD5 / SC-8: the decoded speed table, the
// clamping, and the game's integer 1000/tps.
func TestSpeedTableAndTickMillis(t *testing.T) {
	wantTPS := []int{8, 10, 12, 14, 16, 20, 24, 28, 32}
	for i, want := range wantTPS {
		if got := terrain.TicksPerSecond(i); got != want {
			t.Fatalf("TicksPerSecond(%d) = %d, want %d", i, got, want)
		}
		if got, w := terrain.TickMillis(i), 1000/want; got != w {
			t.Fatalf("TickMillis(%d) = %d, want %d", i, got, w)
		}
	}

	// Out-of-range indices clamp rather than panic or error.
	for _, c := range []struct{ idx, want int }{{-3, 8}, {-1, 8}, {9, 32}, {11, 32}, {1 << 20, 32}} {
		if got := terrain.TicksPerSecond(c.idx); got != c.want {
			t.Fatalf("TicksPerSecond(%d) = %d, want %d (clamped)", c.idx, got, c.want)
		}
	}

	// The map-load default: 16 tps, 62 ms per tick by the game's truncating
	// division (not 62.5), so a full 16-tick cycle is 992 ms.
	if got := terrain.TicksPerSecond(terrain.DefaultSpeedIndex); got != 16 {
		t.Fatalf("default tps = %d, want 16", got)
	}
	if got := terrain.TickMillis(terrain.DefaultSpeedIndex); got != 62 {
		t.Fatalf("default TickMillis = %d, want 62", got)
	}
	if got := terrain.TickMillis(terrain.DefaultSpeedIndex) * terrain.WaterCycleTicks; got != 992 {
		t.Fatalf("default cycle = %d ms, want 992", got)
	}
}

// TestTickerAccumulates - AC-8 / SC-9: ticks fire only on whole period
// boundaries with the remainder carried; a large elapsed fires its whole
// quotient at once; a negative elapsed does nothing.
//
// 0041 T1 moves the clock's unit from milliseconds to microseconds, so every
// elapsed below is the same span written three orders of magnitude finer.
// The 62 ms tick it is written over is unchanged: the index path widens the
// game's own quotient rather than recomputing one, which is what
// TestSpeedTableAndTickMillis above pins and this file does not restate.
func TestTickerAccumulates(t *testing.T) {
	tk := terrain.NewTicker(terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex))
	if got := tk.Period(); got != 62_000 {
		t.Fatalf("Period = %d us, want 62000 — the decoded 62 ms widened, not 1000000/16", got)
	}
	if got := tk.Count(); got != 0 {
		t.Fatalf("initial Count = %d, want 0", got)
	}

	// Twenty 10 ms steps = 200 ms = 3 whole ticks (186 ms) with 14 ms carried.
	total := 0
	for i := 0; i < 20; i++ {
		total += tk.AdvanceMicros(10_000)
	}
	if total != 3 {
		t.Fatalf("20x10ms fired %d ticks, want 3", total)
	}
	if got := tk.Count(); got != 3 {
		t.Fatalf("Count = %d, want 3", got)
	}

	// A single large elapsed fires its whole quotient at once: 14 carried + 500
	// = 514 ms = 8 ticks, 18 ms carried.
	if got := tk.AdvanceMicros(500_000); got != 8 {
		t.Fatalf("AdvanceMicros(500000) fired %d ticks, want 8", got)
	}
	if got := tk.Count(); got != 11 {
		t.Fatalf("Count = %d, want 11", got)
	}

	// A negative or zero elapsed fires nothing and does not rewind.
	for _, e := range []int{0, -1, -1_000_000} {
		if got := tk.AdvanceMicros(e); got != 0 {
			t.Fatalf("AdvanceMicros(%d) fired %d ticks, want 0", e, got)
		}
	}
	if got := tk.Count(); got != 11 {
		t.Fatalf("Count = %d after non-positive advances, want 11", got)
	}

	// A re-rate keeps the carried remainder and the counter: index 8 is 32 tps
	// -> 31 ms -> 31000 us, and the 18 ms already carried stay carried, so the
	// tick fires 13 ms later rather than a whole period later.
	tk.SetPeriod(terrain.SpeedIndexPeriod(8))
	if got := tk.Period(); got != 31_000 {
		t.Fatalf("Period after re-rating to index 8 = %d us, want 31000", got)
	}
	if got := tk.Count(); got != 11 {
		t.Fatalf("Count changed on a re-rate: %d, want 11", got)
	}
	if got := tk.AdvanceMicros(12_999); got != 0 {
		t.Fatalf("12999 us after the re-rate fired %d ticks, want 0 — 18000 were carried, so 31000 is not yet reached", got)
	}
	if got := tk.AdvanceMicros(1); got != 1 {
		t.Fatalf("the microsecond crossing 31000 fired %d ticks, want 1 — the remainder must survive a re-rate", got)
	}

	// A period below one microsecond is brought up rather than dividing by zero.
	tk.SetPeriod(0)
	if got := tk.Period(); got != 1 {
		t.Fatalf("Period after SetPeriod(0) = %d, want 1", got)
	}
}

func TestTickerConservesTime(t *testing.T) {
	rng := rand.New(rand.NewSource(0x5EA))
	periods := []struct {
		label  string
		period int
	}{
		{"speed index 0", terrain.SpeedIndexPeriod(0)},
		{"speed index 4", terrain.SpeedIndexPeriod(4)},
		{"speed index 8", terrain.SpeedIndexPeriod(8)},
		{"rate 1", terrain.RatePeriod(1)},
		{"rate 256", terrain.RatePeriod(256)},
		{"rate 1024", terrain.RatePeriod(terrain.RateMax)},
	}
	for _, p := range periods {
		tk := terrain.NewTicker(p.period)
		sum, fired := 0, 0
		for i := 0; i < 500; i++ {
			e := rng.Intn(200_000)
			sum += e
			fired += tk.AdvanceMicros(e)
		}
		if want := sum / p.period; fired != want {
			t.Fatalf("%s: fired %d ticks over %d us at %d us/tick, want %d", p.label, fired, sum, p.period, want)
		}
		if got := tk.Count(); got != uint32(fired) {
			t.Fatalf("%s: Count = %d, want %d", p.label, got, fired)
		}
	}
}

// TestRatePeriodIsTheExactMicrosecondQuotient - 0041 SC-1 (AC-1): the rate
// model's period is 1000000/rate truncated, over a rate brought into 1..1024
// rather than refused.
//
// The three pinned quotients are written as literals with their arithmetic
// beside them, never as microsPerSecond/rate recomputed here: a test that
// re-derived the expression would agree with any dividend the implementation
// happened to hold, including the millisecond one this story replaces.
func TestRatePeriodIsTheExactMicrosecondQuotient(t *testing.T) {
	for _, c := range []struct{ rate, want int }{
		{1, 1_000_000}, // 1000000/1
		{256, 3_906},   // 1000000/256 = 3906.25
		{1024, 976},    // 1000000/1024 = 976.5625
	} {
		if got := terrain.RatePeriod(c.rate); got != c.want {
			t.Errorf("RatePeriod(%d) = %d us, want %d", c.rate, got, c.want)
		}
	}

	// The bounds are ours and the clamp is total: outside is brought inside,
	// never refused, and the period that comes back is the boundary rate's own.
	if terrain.RateMin != 1 || terrain.RateMax != 1024 {
		t.Fatalf("the rate range is %d..%d, want 1..1024", terrain.RateMin, terrain.RateMax)
	}
	for _, c := range []struct{ rate, want int }{
		{-5, terrain.RateMin}, {0, terrain.RateMin}, {5000, terrain.RateMax},
		{1 << 20, terrain.RateMax}, {1, 1}, {1024, 1024}, {17, 17},
	} {
		if got := terrain.ClampRate(c.rate); got != c.want {
			t.Errorf("ClampRate(%d) = %d, want %d", c.rate, got, c.want)
		}
		if got, want := terrain.RatePeriod(c.rate), terrain.RatePeriod(c.want); got != want {
			t.Errorf("RatePeriod(%d) = %d, want the clamped rate %d's %d", c.rate, got, c.want, want)
		}
	}
}

// TestRatePeriodKeepsEveryRateWithinATenthOfAPercent - 0041 SC-1 (AC-1): the
// ACCURACY of the rate model, counted over a driven elapsed schedule rather
// than read back off the period.
//
// Each rate is driven ten seconds of elapsed time in 5 ms slices — a schedule,
// so what is measured is what the accumulator did over many calls and not what
// one division returned — and the ticks it fired are compared against the rate
// the caller asked for. The two out-of-range values are driven the same way and
// must fire exactly what the boundary rates fired.
//
// 969 is in the table because it is the worst case in the whole range: 1031 us
// against a true 1031.99, 0.0929% fast. A rate model that missed the ceiling by
// more than a tenth of a percent would show up there first.
func TestRatePeriodKeepsEveryRateWithinATenthOfAPercent(t *testing.T) {
	const (
		sliceUS  = 5_000
		seconds  = 10
		slices   = seconds * 1_000_000 / sliceUS
		toleranc = 1000 // 0.1% expressed as a permille denominator
	)

	drive := func(rate int) int {
		tk := terrain.NewTicker(terrain.RatePeriod(rate))
		fired := 0
		for i := 0; i < slices; i++ {
			fired += tk.AdvanceMicros(sliceUS)
		}
		if got := tk.Count(); got != uint32(fired) {
			t.Fatalf("rate %d: Count = %d after %d ticks", rate, got, fired)
		}
		return fired
	}

	for _, rate := range []int{1, 2, 16, 17, 64, 128, 256, 257, 969, 1024} {
		fired := drive(rate)
		want := rate * seconds
		off := fired - want
		if off < 0 {
			off = -off
		}
		// |fired - want| * 1000 > want  <=>  the error exceeds 0.1%.
		if off*toleranc > want {
			t.Errorf("rate %d fired %d ticks over %d s, want %d within 0.1%% (off by %d, period %d us)",
				rate, fired, seconds, want, off, terrain.RatePeriod(rate))
		}
	}

	// AC-1's last clause: a value outside the range runs at the nearest one
	// inside it, measured by driving both rather than by comparing periods.
	if got, want := drive(-5), drive(terrain.RateMin); got != want {
		t.Errorf("rate -5 fired %d ticks over %d s, want rate %d's %d", got, seconds, terrain.RateMin, want)
	}
	if got, want := drive(5000), drive(terrain.RateMax); got != want {
		t.Errorf("rate 5000 fired %d ticks over %d s, want rate %d's %d", got, seconds, terrain.RateMax, want)
	}
}

// TestSpeedIndexPeriodWidensTheGamesOwnQuotient - 0041 SC-2 (AC-2): the nine
// decoded speeds keep the periods the game computes, carried into the
// clock's unit by a MULTIPLICATION of the shipped millisecond quotient.
//
// The discriminator is index 4. 62000 is the game's 62 ms widened; 62500 is
// what a microsecond division of 1000000/16 would give, and it is the rate
// model's own answer for 16 — so this asserts that the two paths, over the same
// nominal 16 ticks a second, deliberately DISAGREE.
func TestSpeedIndexPeriodWidensTheGamesOwnQuotient(t *testing.T) {
	for i := terrain.SpeedIndexMin; i <= terrain.SpeedIndexMax; i++ {
		if got, want := terrain.SpeedIndexPeriod(i), terrain.TickMillis(i)*1000; got != want {
			t.Errorf("SpeedIndexPeriod(%d) = %d us, want the shipped %d ms widened = %d",
				i, got, terrain.TickMillis(i), want)
		}
	}
	for _, idx := range []int{-3, -1, 9, 1 << 20} {
		if got, want := terrain.SpeedIndexPeriod(idx), terrain.SpeedIndexPeriod(terrain.ClampSpeedIndex(idx)); got != want {
			t.Errorf("SpeedIndexPeriod(%d) = %d, want the clamped index's %d", idx, got, want)
		}
	}

	if got := terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex); got != 62_000 {
		t.Fatalf("the map-load period is %d us, want 62000 — the game's 62 ms, not 62.5", got)
	}
	if got := terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex) * terrain.WaterCycleTicks; got != 992_000 {
		t.Fatalf("the map-load water cycle is %d us, want 992000", got)
	}
	if terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex) == terrain.RatePeriod(16) {
		t.Fatal("index 4 and rate 16 give the same period; the decoded truncation has been folded into the rate model")
	}

	// AC-2's cadence clause: a clock built at the map-load period fires its
	// first tick when 62000 us have elapsed and not a microsecond earlier.
	tk := terrain.NewTicker(terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex))
	if got := tk.AdvanceMicros(61_999); got != 0 {
		t.Errorf("61999 us at the map-load period fired %d ticks, want 0", got)
	}
	if got := tk.AdvanceMicros(1); got != 1 {
		t.Errorf("the microsecond crossing 62000 fired %d ticks, want 1", got)
	}
	if got := tk.AdvanceMicros(terrain.WaterCycleTicks*62_000 - 62_000); got != terrain.WaterCycleTicks-1 {
		t.Errorf("the rest of a 992000 us cycle fired %d ticks, want %d", got, terrain.WaterCycleTicks-1)
	}
	if got := tk.Count(); got != terrain.WaterCycleTicks {
		t.Errorf("a full cycle left Count = %d, want %d", got, terrain.WaterCycleTicks)
	}
}

// TestTickerReportsTheRemainderItCarries - 0047 SC-6's clock half: the
// accumulator reports how far through the current tick it stands, the
// quantity a frame's own position inside a tick is read off.
//
// Every expectation is arithmetic this test does itself вЂ” elapsed minus the
// whole ticks consumed вЂ” and never a second read of the accessor. The last arm
// is the state the drawing's clamp exists for: SetPeriod keeps the remainder, so
// a re-rate to a period SHORTER than the remainder already accumulated leaves a
// clock reporting more than a whole tick's worth, and it does so until the next
// elapsed span is fed in.
func TestTickerReportsTheRemainderItCarries(t *testing.T) {
	const period = 62_500

	tk := terrain.NewTicker(period)
	if got := tk.Remainder(); got != 0 {
		t.Errorf("a fresh ticker carries %d us, want 0", got)
	}

	// Across an advance: the part of the elapsed span no whole tick consumed.
	fed := 0
	for _, elapsed := range []int{10_000, 10_000, 50_000, 1, 200_000} {
		before := tk.Remainder()
		ticks := tk.AdvanceMicros(elapsed)
		fed += elapsed
		if got, want := tk.Remainder(), before+elapsed-ticks*period; got != want {
			t.Errorf("after %d us the remainder is %d, want %d (it fired %d tick(s))",
				elapsed, got, want, ticks)
		}
		if got := tk.Remainder(); got < 0 || got >= period {
			t.Errorf("the remainder is %d us against a %d us period; an advance leaves less than one tick",
				got, period)
		}
	}
	// Non-vacuity: the remainder must have been somewhere other than zero over
	// that run, or every comparison above holds for a clock that reports nothing.
	if fed%period == 0 {
		t.Fatalf("the fed spans total %d us, a whole number of ticks; pick spans that leave a fraction", fed)
	}
	if tk.Remainder() == 0 {
		t.Error("the run ends on a whole tick boundary, so the accessor was never seen carrying anything")
	}

	// Across a re-rate that SHORTENS the period below the remainder held: the
	// value survives the write and is reported unbounded by the new period.
	tk = terrain.NewTicker(period)
	tk.AdvanceMicros(period - 1)
	held := tk.Remainder()
	if held != period-1 {
		t.Fatalf("the clock carries %d us short of its first tick, want %d", held, period-1)
	}
	short := terrain.RatePeriod(terrain.RateMax)
	if short >= held {
		t.Fatalf("rate %d has period %d us, which is not shorter than the %d held; pick another rate",
			terrain.RateMax, short, held)
	}
	tk.SetPeriod(short)
	if got := tk.Remainder(); got != held {
		t.Errorf("the re-rate left %d us, want the %d it was carrying вЂ” a re-rate keeps the fraction of a "+
			"tick already elapsed", got, held)
	}
	if got := tk.Period(); got != short {
		t.Errorf("the re-rate left period %d, want %d", got, short)
	}
	if tk.Remainder() <= tk.Period() {
		t.Fatal("the remainder is not larger than the new period, so this arm does not reach the state a " +
			"consumer's clamp exists for")
	}

	// And the very next advance consumes it as the whole ticks it is worth.
	if got, want := tk.AdvanceMicros(1), (held+1)/short; got != want {
		t.Errorf("one microsecond past that re-rate fired %d ticks, want %d", got, want)
	}
	if got := tk.Remainder(); got >= short {
		t.Errorf("the remainder is still %d us against a %d us period", got, short)
	}
}

func TestTickerAdvanceOneAndResetPhasePreserveItsOtherState(t *testing.T) {
	tk := terrain.NewTicker(10)
	if got := tk.AdvanceMicros(7); got != 0 {
		t.Fatalf("7 us at a 10 us period advanced %d ticks, want 0", got)
	}
	tk.AdvanceOne()
	if tk.Count() != 1 || tk.Remainder() != 7 || tk.Period() != 10 {
		t.Fatalf("AdvanceOne left count %d remainder %d period %d, want 1, 7, 10",
			tk.Count(), tk.Remainder(), tk.Period())
	}
	tk.ResetPhase()
	if tk.Count() != 1 || tk.Remainder() != 0 || tk.Period() != 10 {
		t.Fatalf("ResetPhase left count %d remainder %d period %d, want 1, 0, 10",
			tk.Count(), tk.Remainder(), tk.Period())
	}
}
