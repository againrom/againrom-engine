package game

import (
	"testing"
	"time"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The readout push (AC-4, AC-6, AC-7). Everything here is synthetic; nothing
// reads a game install.

var readoutBase = time.Unix(1_700_000_000, 0)

// readoutWorld is a map world over a hand-built world with no schedule, beside
// the viewer that draws it.
const readoutW, readoutH = 16, 16

func readoutWorld(t *testing.T, ents ...sim.Entity) (*mapWorld, *ui.Viewer) {
	t.Helper()
	w, err := sim.NewWorld(1, sim.Bounds{Width: readoutW, Height: readoutH},
		sim.ModeCanonical, make([]byte, readoutW*readoutH), ents)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	v, err := ui.NewViewer("readout", terrain.Grid{
		Width: readoutW, Height: readoutH, Tiles: make([]uint16, readoutW*readoutH),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	return newMapWorld(w, nil, nil, v), v
}

// TestReadoutStatesTheClockAndNotTheRequestedCadence — 0060 SC-2 (AC-4);
// 0061 SC-4: the readout equals the clock at every rung of the cadence
// ladder and at periods that are on no rung at all.
//
// WHAT THIS ONE DOES NOT PROVE, stated because it was assumed and then measured
// under 0060 and is WEAKER now: it does not discriminate a shadow of the
// REQUEST, and since 0061 it cannot even in principle on this path. setCadence
// adopts the period it is handed verbatim, so a copy kept beside the clock is
// the identical number for every input. What this test pins is that the ladder
// is covered end to end and that an off-ladder period is passed through rather
// than snapped; the discrimination is TestReadoutFollowsTheClockWhenNothingAsked
// below, which was checked by mutation rather than by argument.
func TestReadoutStatesTheClockAndNotTheRequestedCadence(t *testing.T) {
	mw, v := readoutWorld(t)
	mw.paceTo(readoutBase) // the baseline call

	asked := make([]int, 0, terrain.CadenceRungMax+4)
	for r := terrain.CadenceRungMin; r <= terrain.CadenceRungMax; r++ {
		asked = append(asked, terrain.CadencePeriod(r))
	}
	// Periods no rung produces, including our old rate model's own 16/s: the
	// clock holds what it is given and the box states that, whether or not the
	// ladder can name it.
	asked = append(asked, 62_500, 3_906, 40_000, 1)

	for _, periodUS := range asked {
		mw.setCadence(periodUS, false)
		mw.paceTo(readoutBase.Add(time.Millisecond))

		got := v.ReadoutState()
		if got.PeriodUS != mw.clock.Period() {
			t.Errorf("asked %d us: the readout states %d us, the clock holds %d us",
				periodUS, got.PeriodUS, mw.clock.Period())
		}
		if got.PeriodUS != periodUS {
			t.Errorf("asked %d us: the readout states %d us — the seam must adopt a period, not recompute it",
				periodUS, got.PeriodUS)
		}
		if rate, want := terrain.RateOf(got.PeriodUS), terrain.RateOf(periodUS); rate != want {
			t.Errorf("asked %d us: the readout states %d ticks a second, want %d", periodUS, rate, want)
		}
	}
}

// TestReadoutStatesTheStopOnTheFrameItIsSet — 0060 SC-4 (AC-5).
//
// The stopped path of the paced advance returns BEFORE the tick loop, so a push
// placed after that loop would never fire for a stopped world and the box would
// never learn about Space. This is the assertion that pins the deferred push:
// delete the defer and put the call after the loop, and this fails.
func TestReadoutStatesTheStopOnTheFrameItIsSet(t *testing.T) {
	mw, v := readoutWorld(t)
	mw.paceTo(readoutBase)

	mw.setCadence(terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex), true)
	mw.paceTo(readoutBase.Add(time.Second))
	if got := v.ReadoutState(); !got.Stopped {
		t.Error("the readout does not state the stop on the frame it was set")
	}

	// Many frames over a stopped world: the stop holds and so does everything
	// the world knows, because no tick fired.
	first := v.ReadoutState()
	for i := 2; i < 20; i++ {
		mw.paceTo(readoutBase.Add(time.Duration(i) * time.Second))
	}
	if got := v.ReadoutState(); got != first {
		t.Errorf("a stopped world's readout moved from %+v to %+v", first, got)
	}

	mw.setCadence(terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex), false)
	mw.paceTo(readoutBase.Add(30 * time.Second))
	if got := v.ReadoutState(); got.Stopped {
		t.Error("the readout still states the stop after it was cleared")
	}
	// The rate is unchanged across both, which is why the stop is stated beside
	// it rather than as a rate of zero.
	if got := terrain.RateOf(v.ReadoutState().PeriodUS); got != 16 {
		t.Errorf("the rate came back as %d across a stop, want 16", got)
	}
}

// TestReadoutStatesTheWorldsOwnTickAndDigest — 0060 SC-5 (AC-6, AC-7).
//
// Both are asked of the world at the moment they are stated, so they cannot lag
// it by a frame or run ahead of it. The digest is compared against the world's
// own Hash rather than against a pinned constant: what is being tested is that
// the readout reports THIS world, not that this world hashes to some number.
func TestReadoutStatesTheWorldsOwnTickAndDigest(t *testing.T) {
	mw, v := readoutWorld(t, sim.Entity{ID: 1, X: 2, Y: 3, HP: 10, MaxHP: 10})

	// The tick-0 push: a map opened and not yet advanced already states the
	// state it opened at, rather than a box of zeros.
	if got := v.ReadoutState(); got.Tick != mw.world.Tick() || got.Digest != mw.world.Hash() {
		t.Errorf("the tick-0 push states tick %d digest %#x, want %d and %#x",
			got.Tick, got.Digest, mw.world.Tick(), mw.world.Hash())
	}
	if got := v.ReadoutState().PeriodUS; got != terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex) {
		t.Errorf("a freshly opened map states %d us, want the map-load period %d us",
			got, terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex))
	}

	mw.paceTo(readoutBase)
	for i := 1; i <= 40; i++ {
		mw.paceTo(readoutBase.Add(time.Duration(i) * 100 * time.Millisecond))
		got := v.ReadoutState()
		if got.Tick != mw.world.Tick() {
			t.Fatalf("frame %d: the readout states tick %d, the world holds %d",
				i, got.Tick, mw.world.Tick())
		}
		if got.Digest != mw.world.Hash() {
			t.Fatalf("frame %d: the readout states digest %#x, the world hashes %#x",
				i, got.Digest, mw.world.Hash())
		}
	}
	if mw.world.Tick() == 0 {
		t.Fatal("no tick fired; this test would have passed over a world that never moved")
	}
}

// TestHiddenReadoutIsNotPushed — 0060 SC-6: the digest is a full encode of
// the world's byte form, and a front-end that hid the box should not pay for
// a number nobody can see.
//
// The assertion is that what the viewer holds does not MOVE while the box is
// hidden, which is the only thing observable from here — and it is the same
// thing: a push that happened would carry the new tick.
func TestHiddenReadoutIsNotPushed(t *testing.T) {
	mw, v := readoutWorld(t, sim.Entity{ID: 1, X: 2, Y: 3, HP: 10, MaxHP: 10})
	v.ShowReadout(false)
	mw.paceTo(readoutBase)

	before := v.ReadoutState()
	for i := 1; i <= 10; i++ {
		mw.paceTo(readoutBase.Add(time.Duration(i) * 100 * time.Millisecond))
	}
	if mw.world.Tick() == 0 {
		t.Fatal("no tick fired; the world must have moved for this to mean anything")
	}
	if got := v.ReadoutState(); got != before {
		t.Errorf("a hidden readout was pushed: %+v became %+v", before, got)
	}

	// Shown again, the very next frame catches it up.
	v.ShowReadout(true)
	mw.paceTo(readoutBase.Add(2 * time.Second))
	if got := v.ReadoutState(); got.Tick != mw.world.Tick() {
		t.Errorf("the frame after showing states tick %d, the world holds %d", got.Tick, mw.world.Tick())
	}
}

// TestDrawnEntityCarriesTheSpeed — 0060 SC-5: the speed and the crossing
// cross the seam as the simulation's own, and the crossing is the RECORDED
// span rather than a length derived from the speed on either side.
func TestDrawnEntityCarriesTheSpeed(t *testing.T) {
	mw, _ := readoutWorld(t,
		sim.Entity{ID: 1, X: 2, Y: 3, HP: 10, MaxHP: 10, Speed: 35,
			Transit: 5, TransitTotal: 6},
		sim.Entity{ID: 2, X: 4, Y: 5, HP: 10, MaxHP: 10})

	draws := mw.entityDraws()
	if len(draws) != 2 {
		t.Fatalf("drew %d entities, want 2", len(draws))
	}
	if draws[0].Speed != 35 || draws[0].TransitSpan != 6 {
		t.Errorf("entity 1 crosses as speed %d span %d, want 35 and the recorded 6",
			draws[0].Speed, draws[0].TransitSpan)
	}
	// A mover with no speed is on no recorded crossing, and the two are carried
	// independently: nothing here derives one from the other.
	if draws[1].Speed != 0 || draws[1].TransitSpan != 0 {
		t.Errorf("entity 2 crosses as speed %d span %d, want 0 and 0",
			draws[1].Speed, draws[1].TransitSpan)
	}
}

// TestDrawnEntityCarriesTheGroupRate — 0060 SC-10: the group term crosses
// whole, beside the entity's own speed and never composed with it.
//
// The two numbers differ in the fixture, and they differ the way the live
// behaviour makes them differ: a fast unit ordered with a slow group. A seam that
// carried the composed rate would state 12 for the speed and nothing for the
// term, and this says so.
func TestDrawnEntityCarriesTheGroupRate(t *testing.T) {
	mw, _ := readoutWorld(t,
		sim.Entity{ID: 1, X: 2, Y: 3, HP: 10, MaxHP: 10, Speed: 40, GroupSpeed: 12},
		sim.Entity{ID: 2, X: 4, Y: 5, HP: 10, MaxHP: 10, Speed: 40})

	draws := mw.entityDraws()
	if len(draws) != 2 {
		t.Fatalf("drew %d entities, want 2", len(draws))
	}
	if draws[0].Speed != 40 || draws[0].GroupSpeed != 12 {
		t.Errorf("entity 1 crosses as speed %d group %d, want 40 and 12 — both, uncomposed",
			draws[0].Speed, draws[0].GroupSpeed)
	}
	if draws[1].Speed != 40 || draws[1].GroupSpeed != 0 {
		t.Errorf("entity 2 crosses as speed %d group %d, want 40 and no term",
			draws[1].Speed, draws[1].GroupSpeed)
	}
}

// TestReadoutFollowsTheClockWhenNothingAsked — 0060 SC-2. THE
// anti-shadow-counter test: the clock is moved by a path that never told
// anyone it was moving.
//
// This is the shape the contract forbids — a value MAINTAINED beside an
// authority — and it is the only shape that can be caught, because a value
// derived from the authority on the same statement is not a shadow at all. Any
// record kept at setCadence is stale here, so a readout reading one states the
// rate nobody is running at.
//
// MUTATION-CHECKED: storing the requested period in setCadence and reporting
// that stored copy makes this test fail and leaves every other test in this file
// green. It is the ONLY test in this file that does so since 0061 — the ladder's
// period and the clock's are now the same number on every path where nobody
// bypasses the seam, which is the defect that story fixed and is exactly why the
// bypass below is the discriminating fixture.
func TestReadoutFollowsTheClockWhenNothingAsked(t *testing.T) {
	mw, v := readoutWorld(t)
	mw.paceTo(readoutBase)

	// A cadence the front-end did ask for, so any record beside the clock is
	// populated and holds a plausible-looking value.
	mw.setCadence(terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex), false)
	mw.paceTo(readoutBase.Add(time.Millisecond))
	if got := terrain.RateOf(v.ReadoutState().PeriodUS); got != 16 {
		t.Fatalf("setup: the readout states %d/s, want 16", got)
	}

	// Now move the clock alone. Nothing else in this world is told.
	for _, rate := range []int{512, 4, 1024, 1} {
		mw.clock.SetPeriod(terrain.RatePeriod(rate))
		mw.paceTo(readoutBase.Add(2 * time.Millisecond))

		got := v.ReadoutState().PeriodUS
		if got == terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex) {
			t.Fatalf("re-rated to %d/s behind the front-end, the readout still states the last "+
				"REQUESTED 16/s — it is reading a counter beside the clock, not the clock", rate)
		}
		if want := mw.clock.Period(); got != want {
			t.Errorf("re-rated to %d/s, the readout states %d us, the clock holds %d us", rate, got, want)
		}
		if r := terrain.RateOf(got); r != rate {
			t.Errorf("re-rated to %d/s, the readout states %d/s", rate, r)
		}
	}
}

// TestAFreshlyOpenedMapStatesTheGamesOwnTick — 0060 SC-2; 0061 SC-4
// (AC-6): a map opens at the GAME's own tick and is never rounded to ours.
//
// ITS CLAIM CHANGED WITH 0061 AND IS RESTATED RATHER THAN LEFT TO BE READ THE
// OLD WAY. Under 0060 this was the one place the clock and the key ladder
// disagreed in production — the clock opened at the shipped 62 ms widened,
// 62000 us, while the ladder opened at a rate of 16, whose period in our own
// model is 62500 — and the test discriminated a readout fed by the ladder. 0061
// removed that disagreement by putting the ladder through the game's own
// settings, so it no longer discriminates a ladder-fed box; that duty is
// TestReadoutFollowsTheClockWhenNothingAsked's alone.
//
// What it pins NOW is the other half, and it is the half the owner cares about:
// 62000 is the game's own integer division of 1000 by 16 and must not become the
// round 62500, because the 992 ms water cycle — the one independently confirmed
// quantity in this area — follows from that truncation.
func TestAFreshlyOpenedMapStatesTheGamesOwnTick(t *testing.T) {
	_, v := readoutWorld(t)

	game := terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex)
	ours := terrain.RatePeriod(terrain.TicksPerSecond(terrain.DefaultSpeedIndex))
	if game == ours {
		t.Fatal("the two periods are equal; this test discriminates nothing")
	}

	got := v.ReadoutState().PeriodUS
	if got == ours {
		t.Fatalf("an opened map states %d us — the period OUR rate model makes of 16 ticks a second, not "+
			"the %d us the game's own arithmetic gives", ours, game)
	}
	if got != game {
		t.Errorf("an opened map states %d us, want the game's own %d us", got, game)
	}
	// Both still state 16 ticks a second, which is exactly why the period is
	// drawn beside the rate: a rate-only box cannot tell these two apart.
	if a, b := terrain.RateOf(game), terrain.RateOf(ours); a != 16 || b != 16 {
		t.Errorf("the two periods state %d/s and %d/s; the fixture assumed both were 16", a, b)
	}
	// And the opening period is a RUNG, which is what makes it reachable again
	// after any number of presses (AC-1).
	if idx, ok := terrain.SpeedIndexOf(got); !ok || idx != terrain.DefaultSpeedIndex {
		t.Errorf("an opened map states %d us, which the ladder reports as speed (%d, %v) — the cadence a "+
			"map opens at must lie ON the ladder", got, idx, ok)
	}
}

// The readout states the saved group record's rate term.
func TestDrawnEntityCarriesTheSavedGroupRate(t *testing.T) {
	mw, _ := readoutWorld(t,
		sim.Entity{ID: 1, Owner: 1, Group: 19, X: 2, Y: 3, HP: 10, MaxHP: 10, Speed: 40},
		sim.Entity{ID: 2, Owner: 1, Group: 20, X: 4, Y: 5, HP: 10, MaxHP: 10, Speed: 40})
	g := sim.SavedGroup{ID: 71, Selector: 19}
	g.Members = []sim.SavedGroupMember{{Archive: 1, Entity: 1, Bound: true}}
	g.AI[0x44] = 7
	other := sim.SavedGroup{ID: 72, Selector: 20}
	if err := mw.world.ImportSavedGroups([]sim.SavedGroup{g, other}, nil); err != nil {
		t.Fatal(err)
	}
	if rate, _ := mw.world.RateSpeed(1); rate != 7 {
		t.Fatalf("the simulation's own rate for entity 1 is %d, want the record's 7", rate)
	}

	draws := mw.entityDraws()
	if draws[0].GroupSpeed != 7 {
		t.Errorf("entity 1 reads group rate %d, want the saved group record's 7", draws[0].GroupSpeed)
	}
	if draws[1].GroupSpeed != 0 {
		t.Errorf("entity 2 reads group rate %d, want none", draws[1].GroupSpeed)
	}
}
