package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// A unit pursuer ordered onto a structure exports a current SAV that loads.
func TestCurrentSAVAfterAPursuerTurnsToAStructure(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	if err := f.App("structure order SAV").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	actors := []sim.Entity{
		{ID: 1, Owner: sim.SelfSlot, X: 2, Y: 10, HP: 100, MaxHP: 100, Reach: 1, Facing: 64},
		{ID: 2, Owner: 2, X: 20, Y: 10, HP: 100, MaxHP: 100, Reach: 1, Facing: 192},
	}
	for i := range actors {
		actors[i].TokenSize, actors[i].TypeID = 1, 1
		actors[i].HealthRegenPeriod, actors[i].ManaRegenPeriod, actors[i].Capacity = 1, 1, data.UnitCapacity()
	}
	s := sim.Structure{ID: 3, Col: 25, Row: 10, Width: 1, Height: 1, Attach: 1, Blocking: 1, Field42: 100, MaxHealth: 100}
	w, err := sim.NewStructuredWorld(1, f.live.world.Bounds(), sim.ModeCanonical, sim.Terrain{}, actors, nil, sim.Relations{}, nil, nil, nil, sim.GhostTemplate{}, []sim.Structure{s})
	if err != nil {
		t.Fatal(err)
	}
	f.live.world, f.live.mission.state.World = w, w
	sim.Step(w, []sim.Command{sim.Attack(1, 2)})
	if !w.Entities()[0].Pursuit.Held {
		t.Fatal("no held pursuit after the unit order")
	}
	sim.Step(w, []sim.Command{sim.AttackStructure(1, 3)})
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, "Structure order")
	if err != nil {
		t.Fatalf("current SAV export after the structure order: %v", err)
	}
	openCurrentEffectSave(t, f, raw)
}
