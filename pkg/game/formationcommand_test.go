package game

import (
	"reflect"
	"testing"
	"time"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestFormationSeamQueuesTheDecodedPlayerCommandAndCyclesVisibleStates(t *testing.T) {
	m := worldFixtureMap()
	mw := newMapWorld(mapload.FromALM(m), nil, nil, worldFixtureViewer(t, m))
	if got := mw.world.FormationMode(sim.SelfSlot); got != 2 {
		t.Fatalf("opening formation mode = %d, want Auto/stored 2", got)
	}

	for _, tc := range []struct {
		name       string
		wantParam  int32
		wantStored uint8
	}{
		{"Auto to On", 2, 1},
		{"On to Off", 0, 0},
		{"Off to Auto", 1, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := mw.world.Hash()
			mw.cycleFormation()
			want := []sim.Command{{
				Kind: sim.KindPlayerParameter, Player: sim.SelfSlot,
				X: int32(sim.PlayerParameterFormation), Y: tc.wantParam,
			}}
			if !reflect.DeepEqual(mw.pending, want) {
				t.Fatalf("pending = %+v, want %+v", mw.pending, want)
			}
			if mw.world.Hash() != before {
				t.Fatal("issuing the player command mutated the world before an advance")
			}
			if len(mw.commanded) != 0 {
				t.Fatalf("formation setting marked entities commanded: %v", mw.commanded)
			}
			if applied := mw.tick(); applied != 1 {
				t.Fatalf("advance applied %d pending commands, want 1", applied)
			}
			if got := mw.world.FormationMode(sim.SelfSlot); got != tc.wantStored {
				t.Errorf("stored mode = %d, want %d", got, tc.wantStored)
			}
		})
	}
}

func TestFormationCommandIssuedWhileStoppedWaitsForTheFirstResumedTick(t *testing.T) {
	const periodUS = 62_500
	base := time.Unix(1_700_000_000, 0)
	period := time.Duration(periodUS) * time.Microsecond

	m := worldFixtureMap()
	mw := newMapWorld(mapload.FromALM(m), nil, nil, worldFixtureViewer(t, m))
	mw.setCadence(periodUS, false)
	if n, applied := mw.paceTo(base); n != 0 || applied != 0 {
		t.Fatalf("baseline ran %d ticks/%d commands", n, applied)
	}

	mw.setCadence(periodUS, true)
	before := mw.world.Hash()
	// Three distinct edges must remain three distinct cycle steps even though
	// no tick may mutate the canonical byte between them.
	mw.cycleFormation() // Auto -> On (authored 2)
	mw.cycleFormation() // On -> Off (authored 0)
	mw.cycleFormation() // Off -> Auto (authored 1)
	if n, applied := mw.paceTo(base.Add(period)); n != 0 || applied != 0 {
		t.Fatalf("stopped frame ran %d ticks/%d commands", n, applied)
	}
	if mw.world.Hash() != before || mw.world.FormationMode(sim.SelfSlot) != 2 || len(mw.pending) != 3 {
		t.Fatalf("stopped command leaked: mode %d pending %d hash moved %v",
			mw.world.FormationMode(sim.SelfSlot), len(mw.pending), mw.world.Hash() != before)
	}
	if got := []int32{mw.pending[0].Y, mw.pending[1].Y, mw.pending[2].Y}; !reflect.DeepEqual(got, []int32{2, 0, 1}) {
		t.Fatalf("stopped cycle authored values = %v, want [2 0 1]", got)
	}

	mw.setCadence(periodUS, false)
	if n, applied := mw.paceTo(base.Add(2 * period)); n != 1 || applied != 3 {
		t.Fatalf("resumed frame ran %d ticks/%d commands, want 1/3", n, applied)
	}
	if got := mw.world.FormationMode(sim.SelfSlot); got != 2 {
		t.Errorf("resumed mode = %d, want Auto/stored 2", got)
	}
	if len(mw.pending) != 0 {
		t.Errorf("resumed queue holds %d commands, want empty", len(mw.pending))
	}
}

func TestFormationCycleTreatsEveryForcedModeAsOn(t *testing.T) {
	for _, tc := range []struct {
		mode uint8
		want int32
	}{
		{0, 1},
		{2, 2},
		{1, 0},
		{3, 0},
		{255, 0},
	} {
		if got := nextFormationParameter(tc.mode); got != tc.want {
			t.Errorf("stored mode %d cycles with authored %d, want %d", tc.mode, got, tc.want)
		}
	}
}

func TestQueuedFormationCycleUsesTheCommandDefaultAsAuto(t *testing.T) {
	if got := nextQueuedFormationParameter(-1); got != 2 {
		t.Errorf("queued default cycles with authored %d, want On/authored 2", got)
	}
}
