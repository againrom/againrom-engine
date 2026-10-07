package game

import (
	"testing"

	"againrom/pkg/sim"
)

// dyingCompanionWorld puts entity 100 inside its dying window: at hp, at the
// stage a death starts at, with dwell ticks still owed. dwell is the number of
// decayPass calls the window survives.
func dyingCompanionWorld(t *testing.T, hp int32, dwell uint16) *sim.World {
	t.Helper()
	return heroWorld(t, nil, nil, []sim.Entity{
		{ID: 99, Owner: sim.SelfSlot, HP: 10, MaxHP: 10},
		{ID: 100, Owner: sim.SelfSlot, HP: hp, MaxHP: 10, Decay: sim.DecayFallen, Dwell: dwell, DyingTime: int32(dwell)},
	})
}

func TestAFallenCharacterStillInsideHisDyingWindowDoesNotEndTheMission(t *testing.T) {
	w := dyingCompanionWorld(t, -1, 40)
	mw := partyWorld(t, w, townCompanionParty(), []sim.EntityID{99, 100})

	e, ok := mw.entity(100)
	if !ok || e.Alive() || !e.Dying() {
		t.Fatalf("setup: entity 100 is alive=%v dying=%v present=%v; the fixture cannot discriminate",
			e.Alive(), e.Dying(), ok)
	}

	mw.tick()
	if mw.mission.announced {
		t.Fatalf("announced = true with outcome %v while the fallen character is still dying, want no report",
			mw.mission.outcome)
	}
}

// A body already at -10 cannot be healed, so only its dying window defers the
// loss, and the loss comes when that window closes.
func TestAFinishedBodyLosesTheMissionWhenTheDyingWindowCloses(t *testing.T) {
	const dwell = 3
	w := dyingCompanionWorld(t, -10, dwell)
	mw := partyWorld(t, w, townCompanionParty(), []sim.EntityID{99, 100})

	// One tick per owed dwell. The window is still open on each of them,
	// which is what makes the announcement below attributable to its close
	// rather than to any tick that happened to run.
	for i := 0; i < dwell; i++ {
		if mw.mission.announced {
			t.Fatalf("announced on tick %d of %d, while the dying window was still open", i, dwell)
		}
		mw.tick()
	}

	if !mw.mission.announced || mw.mission.outcome != sim.OutcomeLost {
		t.Fatalf("announced=%v outcome=%v after the dying window closed, want true/OutcomeLost",
			mw.mission.announced, mw.mission.outcome)
	}
}
