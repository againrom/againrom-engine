package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The final mission's authored start takes script unit 188 off the map and an
// authored arrival returns it once four groups of three have died. The pass that
// regenerates health visits only actors on the map.
var offMapArrivalGroups = map[uint32]bool{19: true, 18: true, 1: true, 15: true, 28: true, 29: true, 14: true, 16: true, 33: true, 5: true, 13: true, 32: true}

func openFinalMissionDemon(t *testing.T) (*FrontEnd, sim.EntityID) {
	t.Helper()
	f := openCampaignMission(t, 150)
	id, ok := mapload.ScriptUnits(f.live.mission.state.Map, f.live.mission.party)[188]
	if !ok {
		t.Fatal("mission 150 carries no script unit 188")
	}
	return f, id
}

func finalMissionWound(t *testing.T, f *FrontEnd, id sim.EntityID, hp int32) {
	t.Helper()
	for i := 0; i < 20; i++ {
		f.live.tick()
	}
	e, _ := f.live.entity(id)
	if !e.OffMap {
		t.Fatal("the authored start did not take unit 188 off the map")
	}
	if err := f.live.world.HeadlessDamage(id, e.HP-hp); err != nil {
		t.Fatal(err)
	}
}

func finalMissionHold(t *testing.T, f *FrontEnd, id sim.EntityID, label string, ticks int, hp int32) {
	t.Helper()
	for i := 0; i < ticks; i++ {
		f.live.tick()
		e, _ := f.live.entity(id)
		if !e.OffMap || e.HP != hp {
			t.Fatalf("%s: tick %d unit 188 off-map %v HP %d, want off the map at %d", label, i, e.OffMap, e.HP, hp)
		}
	}
}

func finalMissionArrive(t *testing.T, f *FrontEnd, id sim.EntityID) sim.Entity {
	t.Helper()
	mw := f.live
	hero, _ := mw.entity(mw.mission.ids[0])
	for _, e := range mw.world.Entities() {
		if e.ID != id && offMapArrivalGroups[e.Group] && e.Owner != hero.Owner {
			_ = mw.world.HeadlessKill(e.ID)
		}
	}
	for i := 0; i < 200; i++ {
		mw.tick()
		if e, _ := mw.entity(id); !e.OffMap {
			return e
		}
	}
	t.Fatal("unit 188 never arrived")
	return sim.Entity{}
}

// A wounded unit 188 keeps its health while it is off the map, natively and
// after a cold LOAD of a save made while it is off the map, and arrives wounded.
// On the map it regenerates again.
func TestReleaseFinalMissionUnitOffMapKeepsItsHealthNativeAndColdLoad(t *testing.T) {
	const wound = 5
	f, id := openFinalMissionDemon(t)
	finalMissionWound(t, f, id, wound)
	finalMissionHold(t, f, id, "native", 700, wound)
	arrived := finalMissionArrive(t, f, id)
	if arrived.HP != wound {
		t.Fatalf("native arrival HP %d, want the wound %d it left with", arrived.HP, wound)
	}
	for i := 0; i < 200; i++ {
		f.live.tick()
	}
	if e, _ := f.live.entity(id); e.HP <= wound {
		t.Fatalf("on the map unit 188 stayed at %d; regeneration must resume", e.HP)
	}

	g, gid := openFinalMissionDemon(t)
	finalMissionWound(t, g, gid, wound)
	cold := terminalSaveReload(t, g, "cold LOAD")
	finalMissionHold(t, cold, gid, "cold", 700, wound)
	carried := finalMissionArrive(t, cold, gid)
	if carried.ID != arrived.ID || carried.HP != wound || carried.X != arrived.X || carried.Y != arrived.Y {
		t.Fatalf("cold arrival %d at (%d,%d) HP %d, native %d at (%d,%d) HP %d",
			carried.ID, carried.X, carried.Y, carried.HP, arrived.ID, arrived.X, arrived.Y, arrived.HP)
	}
}

// Script unit 188 binds the same actor natively and after a cold LOAD made
// before its arrival: the authored arrival returns that one actor and no other.
func TestReleaseFinalMissionScriptUnitBindsTheSameActorAfterColdLoad(t *testing.T) {
	f, id := openFinalMissionDemon(t)
	for i := 0; i < 12; i++ {
		f.live.tick()
	}
	cold := terminalSaveReload(t, f, "cold LOAD")
	offBefore := map[sim.EntityID]bool{}
	for _, e := range cold.live.world.Entities() {
		offBefore[e.ID] = e.OffMap
	}
	if !offBefore[id] {
		t.Fatalf("entity %d is on the map before the arrival", id)
	}
	native := finalMissionArrive(t, f, id)
	loaded := finalMissionArrive(t, cold, id)
	if native.ID != id || loaded.ID != id || native.X != loaded.X || native.Y != loaded.Y {
		t.Fatalf("native arrival %+v, cold arrival %+v", native, loaded)
	}
	for _, e := range cold.live.world.Entities() {
		if offBefore[e.ID] && !e.OffMap && e.ID != id {
			t.Errorf("entity %d came back with unit 188 after the LOAD", e.ID)
		}
	}
}
