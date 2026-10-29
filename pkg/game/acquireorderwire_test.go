package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// actorOrderInner is byte 8 of the order record U158 of entity id in a SAV.
func actorOrderInner(t *testing.T, raw []byte, id sim.EntityID) byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		t.Fatal("action supplement", err)
	}
	for _, b := range a.Bindings {
		if !b.Structure && !b.Missing && b.ID == id {
			return mustRawField(t, &doc.Objects[b.Object-1], "U158")[8]
		}
	}
	t.Fatalf("no record binds entity %d", id)
	return 0
}

// An acquisition pursuit is written as order inner 6, the byte the
// original's acquisition writers store (AI-PURSUE-040); the World keeps its
// pursuit 5 with the marker, which the action supplement restores. A cold
// LOAD keeps the World, and its SAVE writes 6 again. An explicit attack ends
// the acquisition, and the next SAVE no longer writes 6.
func TestSaveWritesAnAcquisitionPursuitAsInnerSix(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	if err := f.App("acquisition order wire").OpenMission(f.MissionOpener(10)); err != nil {
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
	raw, _, _ := saveCurrentEffect(t, f)
	if got := actorOrderInner(t, raw, 1); got != 6 {
		t.Fatalf("acquisition SAVE writes inner %d, want 6", got)
	}
	cold := openCurrentEffectSave(t, f, raw)
	assertCurrentWorldEqual(t, w, cold.live.world, "acquisition cold LOAD")
	again, _, _ := saveCurrentEffect(t, cold)
	if got := actorOrderInner(t, again, 1); got != 6 {
		t.Fatalf("SAVE after cold LOAD writes inner %d, want 6", got)
	}
	sim.Step(w, []sim.Command{sim.Attack(1, 2)})
	if w.Entities()[0].AcquirePursuit {
		t.Fatal("explicit attack kept the acquisition marker")
	}
	raw, _, _ = saveCurrentEffect(t, f)
	if got := actorOrderInner(t, raw, 1); got == 6 {
		t.Fatal("explicit attack SAVE writes inner 6")
	}
}
