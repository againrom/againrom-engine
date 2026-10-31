package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondNewGameFight orders the hero of live at the nearest unit hostile to
// it and steps app until a blow lands on either side, at most 3,000 ticks.
// It returns the victim and the hit points both sides lost.
func secondNewGameFight(t *testing.T, app *ui.App, live *mapWorld, hero sim.EntityID) (victim sim.Entity, heroLost, victimLost int32) {
	t.Helper()
	h, _ := live.world.Entity(hero)
	relations := live.world.Relations()
	best := int64(-1)
	for _, e := range live.world.Entities() {
		if !e.Alive() || e.Owner == h.Owner || !relations.Hostile(h.Owner, e.Owner) {
			continue
		}
		dx, dy := int64(e.X-h.X), int64(e.Y-h.Y)
		if d := dx*dx + dy*dy; best < 0 || d < best {
			best, victim = d, e
		}
	}
	if best < 0 {
		t.Fatal("mission 10 holds no unit hostile to the hero")
	}
	live.strike(uint32(hero), uint32(victim.ID))
	for tick := 0; tick < 3000; tick++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		dismissSecondMissionDialogue(t, app, live)
		nh, _ := live.world.Entity(hero)
		nv, _ := live.world.Entity(victim.ID)
		if heroLost, victimLost = h.HP-nh.HP, victim.HP-nv.HP; heroLost > 0 || victimLost > 0 {
			return victim, heroLost, victimLost
		}
	}
	t.Fatalf("no blow landed in 3,000 ticks between the hero and unit %d", victim.ID)
	return
}

// A second-game new game enters mission 10 through the first town's talk
// and gates, the hero walks and fights the nearest hostile unit, and the
// save page writes a Bsg& mission file of mission 10, kept under
// AGAINROM_SECOND_SAVE_OUTPUT. A cold LOAD of the file on the same install
// resumes mission 10 with the World of the play that saved it.
func TestReleaseSecondNewGameMissionTenSaves(t *testing.T) {
	secondGameRoot(t)
	out := secondMissionSaveDirectory(t)
	f, app := secondMissionSettled(t)
	live := f.live
	hero := live.mission.ids[0]
	start, _ := live.world.Entity(hero)
	secondMissionMove(t, app, hero, int(start.X)+3, int(start.Y))
	for tick := 0; tick < 200; tick++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		dismissSecondMissionDialogue(t, app, live)
	}
	moved, _ := live.world.Entity(hero)
	if moved.X == start.X && moved.Y == start.Y {
		t.Fatalf("the hero did not move from %d/%d", start.X, start.Y)
	}
	victim, heroLost, victimLost := secondNewGameFight(t, app, live, hero)
	for tick := 0; tick < 100; tick++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		dismissSecondMissionDialogue(t, app, live)
	}
	if live.world.Outcome() != sim.OutcomeUndecided {
		t.Fatalf("mission 10 is decided (%v) before the SAVE", live.world.Outcome())
	}
	t.Logf("hero %d from %d/%d to %d/%d fought unit %d (type %d, map unit %d, owner %d): hero lost %d HP, unit lost %d HP; tick %d hash %016x",
		hero, start.X, start.Y, moved.X, moved.Y, victim.ID, victim.TypeID, victim.MapUnitID, victim.Owner, heroLost, victimLost, live.world.Tick(), live.world.Hash())
	raw := secondMissionNamedSave(t, f, app, out, "Againrom mission 10")
	checkSecondMissionFile(t, raw, 10)
	name := secondSaveFile(t, out, "Againrom mission 10")
	t.Logf("saved %s, %d bytes", filepath.Join(out, name), len(raw))
	cold, _ := secondMissionCold(t, out, name)
	assertCurrentWorldEqual(t, f.live.world, cold.live.world, "cold LOAD")
	if _, err := os.Stat(filepath.Join(out, name)); err != nil {
		t.Fatal(err)
	}
}
