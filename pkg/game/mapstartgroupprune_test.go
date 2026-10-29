package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// emptiedHumanGroupFront starts a mission no LOAD set: the human Player holds
// two Groups and another Player one, as the mission start builds them. The
// only member of the human Player's second Group is killed and leaves play,
// so that Group is empty.
func emptiedHumanGroupFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := currentPoolFixtureFront(t, 91, 92)
	if err := f.App("emptied human Group").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	actors := []sim.Entity{
		{ID: 1, Owner: sim.SelfSlot, X: 10, Y: 10, HP: 100, MaxHP: 100},
		{ID: 2, Owner: sim.SelfSlot, Group: 5, X: 12, Y: 10, HP: 100, MaxHP: 100},
		{ID: 3, Owner: 2, X: 30, Y: 30, HP: 100, MaxHP: 100},
	}
	for i := range actors {
		actors[i].TokenSize, actors[i].TypeID, actors[i].Speed = 1, 1, 30
		actors[i].HealthRegenPeriod, actors[i].ManaRegenPeriod, actors[i].Capacity = 1, 1, data.UnitCapacity()
	}
	w, err := sim.NewWorld(1, f.live.world.Bounds(), sim.ModeCanonical, nil, actors)
	if err != nil {
		t.Fatal(err)
	}
	f.live.world, f.live.mission.state.World = w, w
	if !w.GroupsConstructed() {
		t.Fatal("the fixture World holds no mission-start Group registry")
	}
	sim.Step(w, []sim.Command{sim.Kill(2)})
	for tick := 0; tick < 40000; tick++ {
		if _, held := w.Entity(2); !held {
			break
		}
		sim.Step(w, nil)
	}
	if _, held := w.Entity(2); held || emptyHumanGroups(w) != 1 {
		t.Fatal("the killed member did not leave play and empty its Group")
	}
	return f
}

// emptyHumanGroups counts the Groups of the human Player that hold no member.
func emptyHumanGroups(w *sim.World) int {
	groups, _, _ := w.SavedGroups()
	n := 0
	for _, g := range groups {
		if g.Owner.Owner == sim.SelfSlot && len(g.Members) == 0 {
			n++
		}
	}
	return n
}

// A SAVE of a mission no LOAD set removes the human Player's empty Group from
// the live list and does not write it, as the original's SAVE removes the
// empty Groups of a Player whose +28 is zero (SAV-1251). The cold LOAD of
// that SAV holds the pruned Group lists, and its SAVE writes them again.
func TestMapStartSaveDropsAnEmptiedHumanGroup(t *testing.T) {
	f := emptiedHumanGroupFront(t)
	raw := bindingOrderSave(t, f, "emptied")
	if n := emptyHumanGroups(f.live.world); n != 0 {
		t.Fatalf("the live World keeps %d empty human Groups after SAVE", n)
	}
	cold := coldCurrentScript(t, f, raw)
	if n := emptyHumanGroups(cold.live.world); n != 0 {
		t.Fatalf("the SAV holds %d empty human Groups", n)
	}
	live, loaded := groupListsForDialog(f), groupListsForDialog(cold)
	if live != loaded {
		t.Fatalf("the cold LOAD holds Groups %s, the SAVE left %s", loaded, live)
	}
	again := coldCurrentScript(t, cold, bindingOrderSave(t, cold, "emptied again"))
	if got := groupListsForDialog(again); got != live {
		t.Fatalf("the SAV after cold LOAD holds Groups %s, want %s", got, live)
	}
}

// An autosave writes the pruned lists too, and leaves the live World and its
// hash as they were: a timer changes no hashed state.
func TestAutosaveDropsAnEmptiedHumanGroupAndKeepsTheWorld(t *testing.T) {
	f := emptiedHumanGroupFront(t)
	before := f.live.world.Hash()
	var raw []byte
	if err := f.captureSave(captureTimer, true, func(s Snapshot, _ string) error {
		var err error
		raw, err = f.ExportCurrentSave(s, "autosave")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if f.live.world.Hash() != before || emptyHumanGroups(f.live.world) != 1 {
		t.Fatal("the autosave changed the live World")
	}
	if n := emptyHumanGroups(coldCurrentScript(t, f, raw).live.world); n != 0 {
		t.Fatalf("the autosave holds %d empty human Groups", n)
	}
}
