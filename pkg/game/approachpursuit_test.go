package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

func TestCurrentSAVKeepsAcquisitionPursuitAndFollowingTicks(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	if err := f.App("acquisition SAV").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	actors := []sim.Entity{
		{ID: 1, Owner: sim.SelfSlot, X: 10, Y: 10, HP: 100, MaxHP: 100, Reach: 1, ScanRange: 8, Facing: 192},
		{ID: 2, Owner: 2, X: 11, Y: 10, HP: 100, MaxHP: 100, Reach: 1, ScanRange: 8, Facing: 64},
	}
	for i := range actors {
		actors[i].TokenSize, actors[i].TypeID, actors[i].Speed = 1, 1, 30
		actors[i].HealthRegenPeriod, actors[i].ManaRegenPeriod, actors[i].Capacity = 1, 1, data.UnitCapacity()
	}
	var rel sim.Relations
	rel.Set(sim.SelfSlot, 2, 1)
	rel.Set(2, sim.SelfSlot, 2)
	w, err := sim.NewRelatedWorld(1, f.live.world.Bounds(), sim.ModeCanonical, sim.Terrain{}, actors, nil, rel)
	if err != nil {
		t.Fatal(err)
	}
	f.live.world, f.live.mission.state.World = w, w
	sim.Step(w, []sim.Command{sim.GroupDefend(1, 1, 7)})
	for tick := 0; tick < 16 && !w.Entities()[0].AcquirePursuit; tick++ {
		sim.Step(w, nil)
	}
	if !w.Entities()[0].AcquirePursuit {
		t.Fatal("Defend subject did not acquire")
	}
	for _, acquired := range []bool{true, false} {
		if !acquired {
			sim.Step(w, []sim.Command{sim.Attack(1, 2)})
		}
		raw, _, actions := saveCurrentEffect(t, f)
		if actions.Actions.Actors[0].AcquirePursuit != acquired {
			t.Fatal("SAV current actions lost pursuit arm")
		}
		cold := openCurrentEffectSave(t, f, raw)
		if cold.live.world.Entities()[0].AcquirePursuit != acquired {
			t.Fatal("cold SAV LOAD changed pursuit arm")
		}
		assertCurrentWorldEqual(t, w, cold.live.world, "pursuit cold LOAD")
		move := sim.MoveTo(2, sim.CellPoint{X: 20, Y: 15})
		sim.Step(w, []sim.Command{move})
		sim.Step(cold.live.world, []sim.Command{move})
		for tick := 0; tick < 32; tick++ {
			sim.Step(w, nil)
			sim.Step(cold.live.world, nil)
			if w.Hash() != cold.live.world.Hash() {
				t.Fatalf("SAV pursuit continuation differs at %d", tick)
			}
		}
	}
}
