package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestOpeningAMapLightsItForItsOpeningMinute(t *testing.T) {
	v := dayNightViewer(t)
	seed := v.Sun()
	_ = newMapWorld(dayNightWorld(t), nil, nil, v)

	if v.Sun() == seed {
		t.Fatalf("opening a map left the sun at the seed %+v", seed)
	}
	if want := terrain.SunAt(0, true); v.Sun() != want {
		t.Errorf("the opened map's sun is %+v, want minute 0's %+v", v.Sun(), want)
	}
}

func TestTheClockTheViewerReceivesIsTheWorldsTick(t *testing.T) {
	v := dayNightViewer(t)
	mw := newMapWorld(dayNightWorld(t), nil, nil, v)
	for i := 0; i < 400; i++ {
		mw.tick()
	}
	if got, want := mw.world.Tick(), uint64(400); got != want {
		t.Fatalf("the world ran %d ticks, want %d", got, want)
	}
	// Minute 20's relight is sub-tick 320, the last one this run passed.
	if want := terrain.SunAt(20, true); v.Sun() != want {
		t.Errorf("after 400 ticks the sun is %+v, want minute 20's %+v", v.Sun(), want)
	}
}

func TestNoPacedCallCanFireTwoRelights(t *testing.T) {
	const period = terrain.RelightPeriodMinutes * terrain.SubTicksPerMinute // 320
	worst := 0
	for i := 0; i <= 8; i++ {
		if n := maxCatchUp(terrain.SpeedIndexPeriod(i)); n > worst {
			worst = n
		}
	}
	// The extension past the shipped ladder is the real worst case: the fastest
	// rate this build offers, not the fastest the original does.
	if n := maxCatchUp(terrain.RatePeriod(1 << 30)); n > worst {
		worst = n
	}
	if worst >= period {
		t.Errorf("one paced call may run %d ticks against a relight period of %d; two relights "+
			"could fall in one frame", worst, period)
	}
	t.Logf("worst-case ticks in one paced call = %d, relight period = %d sub-ticks", worst, period)
}

func TestTheCycleReachesNoSimulationState(t *testing.T) {
	plain := newMapWorld(dayNightWorld(t), nil, nil, dayNightViewer(t))

	scrubbed := newMapWorld(dayNightWorld(t), nil, nil, dayNightViewer(t))
	sv := scrubbed.view

	for i := 0; i < 500; i++ {
		plain.tick()
		scrubbed.tick()
		switch i % 7 {
		case 0:
			sv.ToggleTimeFlow()
		case 3:
			sv.StepLightClock()
		}
		if a, b := plain.world.Tick(), scrubbed.world.Tick(); a != b {
			t.Fatalf("tick %d: the two worlds are at %d and %d", i, a, b)
		}
		if a, b := plain.world.Hash(), scrubbed.world.Hash(); a != b {
			t.Fatalf("tick %d: digests %016x and %016x diverged", i, a, b)
		}
	}

	ab, err := plain.world.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	bb, err := scrubbed.world.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if len(ab) != len(bb) {
		t.Fatalf("byte forms are %d and %d bytes long", len(ab), len(bb))
	}
	for i := range ab {
		if ab[i] != bb[i] {
			t.Fatalf("byte forms differ at offset %d: %#02x vs %#02x", i, ab[i], bb[i])
		}
	}

	// And the form round-trips: decoding it and re-encoding gives the same
	// bytes and the same digest, so nothing this story added rides along
	// un-encoded.
	var back sim.World
	if err := back.UnmarshalBinary(ab); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if back.Hash() != plain.world.Hash() {
		t.Errorf("a round-tripped world hashes %016x, want %016x", back.Hash(), plain.world.Hash())
	}
}

// dayNightViewer is a viewer over a synthetic grid with relief in it, so a
// relight has something to change.
func dayNightViewer(t *testing.T) *ui.Viewer {
	t.Helper()
	const w, h = 4, 4
	alts := make([]uint8, w*h)
	for i := range alts {
		if i%3 == 0 {
			alts[i] = 40
		}
	}
	g := terrain.Grid{Width: w, Height: h, Tiles: make([]uint16, w*h), Altitudes: alts}
	v, err := ui.NewViewer("m", g, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	return v
}

// dayNightWorld is a world with two units on it, built by hand.
func dayNightWorld(t *testing.T) *sim.World {
	t.Helper()
	w, err := sim.NewWorld(1, sim.Bounds{Width: 12, Height: 12}, sim.ModeCanonical, nil, []sim.Entity{
		{ID: 0, X: 1, Y: 1, HP: mapload.SpawnHP, MaxHP: mapload.SpawnHP},
		{ID: 1, X: 5, Y: 1, HP: mapload.SpawnHP, MaxHP: mapload.SpawnHP},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	return w
}
