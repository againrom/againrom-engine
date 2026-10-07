package game

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestRetreatCycleQueuesParameterThreePersistsAndWaitsForTheTick(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	store := OptionsStore{Path: path}
	if err := store.SetWimpyMode(wimpyModeHigh); err != nil {
		t.Fatalf("seed WimpyMode: %v", err)
	}
	f := &FrontEnd{PersistenceContext: PersistenceContext{Options: store}}
	f.LoadOptions()

	m := worldFixtureMap()
	local := mapload.FromALM(m).Entities()[0]
	local.Owner = sim.SelfSlot
	local.MaxHP, local.HP, local.Withdraw, local.Wimpy = 101, 101, 77, 66
	mw := retreatTestMapWorld(t, m, local)
	before := mw.world.Entities()
	// Prove loading the process option did not project High onto the world.
	if before[0].Withdraw != 77 || before[0].Wimpy != 66 {
		t.Fatalf("LoadOptions changed saved thresholds to %d/%d, want authored 77/66", before[0].Withdraw, before[0].Wimpy)
	}

	f.cycleRetreat(mw) // persisted High -> Off
	want := []sim.Command{{
		Kind: sim.KindPlayerParameter, Player: sim.SelfSlot,
		X: int32(sim.PlayerParameterRetreat), Y: sim.RetreatModeOff,
	}}
	if !reflect.DeepEqual(mw.pending, want) {
		t.Fatalf("pending = %+v, want %+v", mw.pending, want)
	}
	if got := mw.world.Entities()[0]; got.Withdraw != before[0].Withdraw || got.Wimpy != before[0].Wimpy {
		t.Fatalf("press mutated thresholds before Step: before %+v after %+v", before[0], got)
	}
	if got, err := (OptionsStore{Path: path}).WimpyMode(); err != nil || got != wimpyModeOff {
		t.Fatalf("persisted WimpyMode = %d, %v; want Off/%d", got, err, wimpyModeOff)
	}
	if applied := mw.tick(); applied != 1 {
		t.Fatalf("tick applied %d commands, want 1", applied)
	}
	if got := mw.world.Entities()[0]; got.Withdraw != 0 || got.Wimpy != 0 {
		t.Fatalf("executed Off thresholds = %d/%d, want 0/0", got.Withdraw, got.Wimpy)
	}
}

func TestRetreatCycleRunsOffLowHighOffAndLeavesForeignActorsUntouched(t *testing.T) {
	f := &FrontEnd{}
	m := worldFixtureMap()
	base := mapload.FromALM(m)
	local := base.Entities()[0]
	local.Owner = sim.SelfSlot
	local.MaxHP, local.HP, local.Withdraw, local.Wimpy = 101, 101, 77, 66
	foreign := local
	foreign.ID, foreign.Owner = local.ID+100, sim.SelfSlot+1
	foreign.MaxHP, foreign.HP, foreign.Withdraw, foreign.Wimpy = 333, 333, 55, 44
	mw := retreatTestMapWorld(t, m, local, foreign)

	for _, tc := range []struct {
		name        string
		wantMode    int
		wantRetreat int32
	}{
		{"Off to Low", wimpyModeLow, 10},
		{"Low to High", wimpyModeHigh, 30},
		{"High to Off", wimpyModeOff, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f.cycleRetreat(mw)
			if applied := mw.tick(); applied != 1 {
				t.Fatalf("tick applied %d commands, want 1", applied)
			}
			got := mw.world.Entities()
			if got[0].Withdraw != tc.wantRetreat || got[0].Wimpy != 0 {
				t.Errorf("local thresholds = %d/%d, want %d/0", got[0].Withdraw, got[0].Wimpy, tc.wantRetreat)
			}
			if got[1].Withdraw != 55 || got[1].Wimpy != 44 {
				t.Errorf("foreign thresholds = %d/%d, want authored 55/44", got[1].Withdraw, got[1].Wimpy)
			}
			if f.wimpyMode != tc.wantMode {
				t.Errorf("cached mode = %d, want %d", f.wimpyMode, tc.wantMode)
			}
		})
	}
}

func retreatTestMapWorld(t *testing.T, m *alm.Map, entities ...sim.Entity) *mapWorld {
	t.Helper()
	b := mapload.FromALM(m).Bounds()
	w, err := sim.NewWorld(1, b, sim.ModeCanonical,
		make([]byte, int(b.Width)*int(b.Height)), entities)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	return newMapWorld(w, nil, nil, worldFixtureViewer(t, m))
}

func TestRetreatCommandsIssuedWhileStoppedRemainOrderedUntilResume(t *testing.T) {
	const periodUS = 62_500
	base := time.Unix(1_700_000_000, 0)
	period := time.Duration(periodUS) * time.Microsecond
	f := &FrontEnd{}
	m := worldFixtureMap()
	mw := newMapWorld(mapload.FromALM(m), nil, nil, worldFixtureViewer(t, m))
	mw.setCadence(periodUS, false)
	if n, applied := mw.paceTo(base); n != 0 || applied != 0 {
		t.Fatalf("baseline ran %d ticks/%d commands", n, applied)
	}

	mw.setCadence(periodUS, true)
	f.cycleRetreat(mw) // Low
	f.cycleRetreat(mw) // High
	f.cycleRetreat(mw) // Off
	if n, applied := mw.paceTo(base.Add(period)); n != 0 || applied != 0 {
		t.Fatalf("stopped frame ran %d ticks/%d commands", n, applied)
	}
	if got := []int32{mw.pending[0].Y, mw.pending[1].Y, mw.pending[2].Y}; !reflect.DeepEqual(got, []int32{sim.RetreatModeLow, sim.RetreatModeHigh, sim.RetreatModeOff}) {
		t.Fatalf("stopped cycle values = %v, want [Low High Off]", got)
	}

	mw.setCadence(periodUS, false)
	if n, applied := mw.paceTo(base.Add(2 * period)); n != 1 || applied != 3 {
		t.Fatalf("resumed frame ran %d ticks/%d commands, want 1/3", n, applied)
	}
	if len(mw.pending) != 0 {
		t.Errorf("resumed queue holds %d commands, want empty", len(mw.pending))
	}
}

func TestNextWimpyModeFallsBackSafely(t *testing.T) {
	for _, tc := range []struct{ in, want int }{
		{wimpyModeOff, wimpyModeLow},
		{wimpyModeLow, wimpyModeHigh},
		{wimpyModeHigh, wimpyModeOff},
		{-1, wimpyModeLow},
		{99, wimpyModeLow},
	} {
		if got := nextWimpyMode(tc.in); got != tc.want {
			t.Errorf("nextWimpyMode(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
