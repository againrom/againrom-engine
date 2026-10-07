package ui

// The whole loop, driven by the keys (AC-2, AC-3, AC-5, success criterion
// SC-4).
//
// WHAT STANDS IN FOR THE WORLD HERE is a second terrain.Ticker built at the
// map-load period, exactly as pkg/game builds the world's own, with a seam that
// re-rates it on a cadence call and PUSHES WHAT IT IS HOLDING on every tick.
// Those two bodies are the two production statements this tier cannot see, and
// they are the whole of the fake; everything between the key and the box is the
// shipped path.
//
// Every assertion below compares the box against seam.clock — the far side's
// own — and never against the front-end's ladder. Note what that does and does
// not settle: driving the KEYS cannot separate the box-follows-the-clock and
// box-follows-the-ladder hypotheses, because the two agree at every rate a key
// can select, clamps included. What the keys witness is coverage of the ladder,
// the ceiling and the floor. The separation is the second test below, on this
// side, and its counterpart in pkg/game, on the other.

import (
	"testing"

	"againrom/pkg/render/terrain"
)

// readoutSeam is the far side of the map seam for these tests: a world's clock,
// its stop, and the viewer it pushes to.
type readoutSeam struct {
	clock   *terrain.Ticker
	stopped bool
	unpaced bool
	ticks   uint64
	v       *Viewer
}

// tick is the paced advance's stand-in, and it pushes the readout the way that
// advance does: FROM THE CLOCK, on every call, including one over a stopped
// world.
func (s *readoutSeam) tick() {
	if !s.stopped {
		s.ticks++
	}
	s.v.SetReadout(Readout{PeriodUS: s.clock.Period(), Stopped: s.stopped, Tick: s.ticks})
}

// setCadence is mapWorld.setCadenceMode's stand-in: the period is ADOPTED on
// every call, and the stop and owner-loop selector are switches beside it.
func (s *readoutSeam) setCadence(periodUS int, stopped, unpaced, _ bool) {
	s.clock.SetPeriod(periodUS)
	s.stopped = stopped
	s.unpaced = unpaced
}

func readoutKeyApp(t *testing.T) (*App, *Viewer, *readoutSeam, int) {
	t.Helper()
	var seam *readoutSeam
	load := func(int) (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		v, err := NewViewer("keys", grid(60, 60), &terrain.Tileset{})
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		v.SetFont(panelFont())
		s := &readoutSeam{
			clock: terrain.NewTicker(terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex)),
			v:     v,
		}
		seam = s
		return v, s.tick, nil, s.setCadence, nil, nil, nil, nil, nil, nil, nil
	}

	a := newTestApp(t, appRows(3), load)
	a.flow.screen = ScreenPicker
	a.step(appInput{Enter: true}, cadenceAt(0))
	if a.Screen() != ScreenMap {
		t.Fatalf("setup: screen = %v, want ScreenMap", a.Screen())
	}
	a.step(cadenceInput(false, false, false), cadenceAt(1))
	return a, a.flow.viewer, seam, 1
}

// statedRate is the rate the readout's own cadence row states, parsed back out
// of the text the box would draw — so what is asserted is what a player reads,
// not an intermediate value.
func statedRate(t *testing.T, v *Viewer) (rate int, stopped bool) {
	t.Helper()
	s := v.readoutSubjectOf(60)
	txt, ok := readoutText(s, PanelFieldCadence)
	if !ok {
		t.Fatal("the readout states no cadence")
	}
	stopped = len(txt) > len(readoutStopped) && txt[:len(readoutStopped)] == readoutStopped
	return terrain.RateOf(s.PeriodUS), stopped
}

// TestTheKeysMoveWhatTheBoxStates — 0060 SC-4 (AC-2, AC-3, AC-5).
//
// The ladder is driven all the way to the ceiling and all the way to the floor.
// Past either end the clamp holds the clock still — which is the state that
// makes a working key and a broken key draw the same picture, and the state this
// story exists to make legible — and the box must state the clock at every step.
func TestTheKeysMoveWhatTheBoxStates(t *testing.T) {
	a, v, seam, at := readoutKeyApp(t)

	// A freshly opened map already states its own cadence: the tick-0 push's
	// counterpart, here the first driven frame's.
	if rate, stopped := statedRate(t, v); rate != 16 || stopped {
		t.Fatalf("an opened map states %d/s stopped=%v, want 16/s running", rate, stopped)
	}
	if got := v.ReadoutState().PeriodUS; got != terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex) {
		t.Errorf("an opened map states %d us, want the game's own %d us",
			got, terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex))
	}

	press := func(pause, faster, slower bool) {
		at++
		a.step(cadenceInput(pause, faster, slower), cadenceAt(at))
	}

	// One rung a press to the ceiling, and then five presses past it.
	born := terrain.CadenceRung(terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex))
	rung := born
	for i := 0; i < terrain.CadenceRungMax-born+5; i++ {
		press(false, true, false)
		rung = terrain.ClampCadenceRung(rung + 1)
		rate, stopped := statedRate(t, v)
		if stopped {
			t.Fatalf("press %d: the box states a stop nobody asked for", i)
		}
		if want := terrain.RateOf(terrain.CadencePeriod(rung)); rate != want {
			t.Fatalf("press %d: the box states %d/s, want %d", i, rate, want)
		}
		if got := v.ReadoutState().PeriodUS; got != seam.clock.Period() {
			t.Fatalf("press %d: the box states %d us, the world's clock holds %d us",
				i, got, seam.clock.Period())
		}
	}

	// At the ceiling the box and the LADDER agree only because the ladder clamps
	// too. What is asserted is the clock; that the ladder happens to agree is
	// read here to show the two are separable at all.
	if rung != terrain.CadenceRungMax {
		t.Fatalf("the presses reached rung %d, want the ceiling %d", rung, terrain.CadenceRungMax)
	}
	if a.flow.rung != terrain.CadenceRungMax {
		t.Fatalf("the ladder stands at rung %d, want the ceiling %d", a.flow.rung, terrain.CadenceRungMax)
	}

	// All the way down and past the floor.
	for i := 0; i < terrain.CadenceRungMax+5; i++ {
		press(false, false, true)
		rung = terrain.ClampCadenceRung(rung - 1)
		if rate, _ := statedRate(t, v); rate != terrain.RateOf(terrain.CadencePeriod(rung)) {
			t.Fatalf("press %d down: the box states %d/s, want %d", i, rate,
				terrain.RateOf(terrain.CadencePeriod(rung)))
		}
		if got := v.ReadoutState().PeriodUS; got != seam.clock.Period() {
			t.Fatalf("press %d down: the box states %d us, the clock holds %d us",
				i, got, seam.clock.Period())
		}
	}
	if rung != terrain.CadenceRungMin {
		t.Fatalf("the presses down reached rung %d, want the floor %d", rung, terrain.CadenceRungMin)
	}
	floorRate := terrain.RateOf(terrain.CadencePeriod(terrain.CadenceRungMin))

	// Space, and Space again. The stop shows; the cadence under it does not move.
	press(true, false, false)
	rate, stopped := statedRate(t, v)
	if !stopped || rate != floorRate {
		t.Errorf("after Space the box states %d/s stopped=%v, want %d/s stopped", rate, stopped, floorRate)
	}
	press(true, false, false)
	rate, stopped = statedRate(t, v)
	if stopped || rate != floorRate {
		t.Errorf("after a second Space the box states %d/s stopped=%v, want it running", rate, stopped)
	}
}

// TestTheDrawPathHasNoSecondSourceOfTheRate — 0060 SC-4: what this TIER
// can prove, and stated as exactly that and no more.
//
// The clock is re-rated behind the front-end, so the pushed value and the
// ladder genuinely disagree — the ladder still stands on the map-load rung while
// the world runs at 512 ticks a second — and the box must state 512. That
// establishes the DRAW path reads what was pushed and consults nothing else; it
// cannot establish what production pushes, because the pusher here is this
// file's stand-in. The clock is the source in production, and that half is
// witnessed in pkg/game, by mutation.
//
// The structural half is stronger than either: the ladder is an unexported field
// of the flow and a viewer holds no pointer to one, so there is no expression a
// viewer could write that reaches it.
func TestTheDrawPathHasNoSecondSourceOfTheRate(t *testing.T) {
	a, v, seam, at := readoutKeyApp(t)

	seam.setCadence(terrain.RatePeriod(512), false, false, false)
	at++
	a.step(cadenceInput(false, false, false), cadenceAt(at))

	born := terrain.CadenceRung(terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex))
	if a.flow.rung != born {
		t.Fatalf("the ladder moved to rung %d; this test needs it left behind on %d", a.flow.rung, born)
	}
	ladderRate := terrain.RateOf(terrain.CadencePeriod(a.flow.rung))
	rate, _ := statedRate(t, v)
	if rate == ladderRate {
		t.Fatalf("the box states the ladder's %d/s — it is reading the key presses, not the clock", rate)
	}
	if rate != 512 {
		t.Errorf("the box states %d/s, want the clock's 512", rate)
	}
}
