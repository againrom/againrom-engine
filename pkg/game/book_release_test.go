package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func relocateReleaseMissionWorld(t *testing.T, m *Mission, table *mapload.Table,
	source *sim.World, id sim.EntityID, x, y int32) *sim.World {
	t.Helper()
	ents := source.Entities()
	found := false
	for i := range ents {
		if ents[i].ID == id {
			ents[i].X, ents[i].Y = x, y
			found = true
		}
	}
	if !found {
		t.Fatalf("mission world has no party entity %d", id)
	}
	out, err := sim.NewStructuredWorld(mapload.Seed, source.Bounds(), sim.ModeCanonical,
		mapload.Planes(m.Map, table), ents, source.Script(), source.Relations(), source.Sacks(),
		source.Stock(), source.Spells(), source.Ghost(), source.Structures())
	if err != nil {
		t.Fatalf("relocate mission world to (%d,%d): %v", x, y, err)
	}
	if err := out.DeclareItemWeights(source.ItemWeights()); err != nil {
		t.Fatalf("restore installed item weights: %v", err)
	}
	return out
}

func carriedBookIndex(w *sim.World, id sim.EntityID, spell uint16) (int, bool) {
	stacks, _ := w.CarriedStacks(id)
	for i, stack := range stacks {
		if got, ok := stack.Instance().BookSpell(); ok && got == spell {
			return i, true
		}
	}
	return 0, false
}

// TestReleaseMission80AstralBookTeachesTeleportAndReachesTheGrave binds the
// whole owner-required route to shipped content. The release gate runs it on
// both lawful installs: mission 80's authored unit 143 owns the one-spell
// Astral book, its compiled Give All trigger transfers that complete object,
// inventory use consumes it and teaches Teleport, then the production game
// seam admits an unseen landing on the disconnected grave component and the
// shipped victory trigger fires there.
func TestReleaseMission80AstralBookTeachesTeleportAndReachesTheGrave(t *testing.T) {
	f := releaseFront(t)
	m := releaseMissionMap(t, f, 80)
	party := []mapload.PartyMember{{
		ID: "mission80-mage", Class: 100, Mage: true, StartingHero: true, PlayerCharacter: true,
		Hero:    data.Hero{Body: 50, Reaction: 50, Mind: 50, Spirit: 50},
		Profile: data.Profile{HealthColumn: true, ManaColumn: true},
	}}
	// Slot 5 is Astral for a mage. Mind 50 + skill 80 - 30 reaches the
	// simulation's power ceiling 100, giving Teleport range 34.
	party[0].Hero.Skill[5] = 80
	ms, err := StartMissionFrom(m, "scenario80.alm", 80, f.Table, mapload.DifficultyNormal, party)
	if err != nil {
		t.Fatalf("StartMissionFrom(mission 80): %v", err)
	}
	if len(ms.Start.IDs) != 1 {
		t.Fatalf("mission 80 minted %d party ids, want one", len(ms.Start.IDs))
	}
	heroID := ms.Start.IDs[0]
	if releaseEntity(t, &mapWorld{world: ms.World}, heroID).KnownSpells&(uint32(1)<<26) != 0 {
		t.Fatal("fixture mage already knows Teleport before reading the shipped book")
	}

	var bookOwner sim.EntityID
	ownerFound := false
	for i, u := range m.Units {
		if u.UnitID == 143 {
			if ownerFound {
				t.Fatal("mission 80 has more than one unit 143")
			}
			bookOwner = sim.EntityID(i)
			ownerFound = true
			if u.X/256 != 36 || u.Y/256 != 129 {
				t.Fatalf("mission 80 unit 143 stands at (%d,%d), want (36,129)", u.X/256, u.Y/256)
			}
		}
	}
	if !ownerFound {
		t.Fatal("mission 80 has no unit 143")
	}
	ownerItems, ok := ms.World.CarriedItems(bookOwner)
	if !ok {
		t.Fatalf("mission 80 world has no unit-143 entity %d", bookOwner)
	}
	books := 0
	for _, item := range ownerItems {
		if spell, readable := item.BookSpell(); readable && spell == 26 {
			books++
			if item.Code != 0x0e17 || item.Kind != 5 {
				t.Fatalf("mission 80 Teleport book = %+v, want code 0x0e17 kind 5", item)
			}
		}
	}
	if books != 1 {
		t.Fatalf("unit 143 readable Teleport books = %d, want one; stock %+v", books, ownerItems)
	}

	// Put the same minted hero inside the shipped trigger's <=3 neighbourhood.
	// The world is rebuilt from the mission's own entities, program, terrain,
	// stock, spells and structures; only the hero cell changes.
	ms.World = relocateReleaseMissionWorld(t, ms, f.Table, ms.World, heroID, 37, 129)
	for tick := 0; tick < 128; tick++ {
		if _, got := carriedBookIndex(ms.World, heroID, 26); got {
			break
		}
		sim.Step(ms.World, nil)
	}
	bookIndex, gotBook := carriedBookIndex(ms.World, heroID, 26)
	if !gotBook {
		t.Fatalf("mission 80 Give All did not transfer unit 143's Teleport book")
	}
	if _, still := carriedBookIndex(ms.World, bookOwner, 26); still {
		t.Fatal("mission 80 Give All duplicated the Teleport book")
	}

	mw := openMission(ms, f.Table, f.Units, worldFixtureViewer(t, m),
		f.Archives.Containers, f.Faces, f.NPCFaces)
	mw.enqueueEquip(bookIndex)
	if len(mw.pending) != 1 || mw.pending[0].Kind != sim.KindReadBook {
		t.Fatalf("shipped book use queued %+v, want one KindReadBook", mw.pending)
	}
	mw.tick()
	if got := releaseEntity(t, mw, heroID).KnownSpells; got&(uint32(1)<<26) == 0 {
		t.Fatalf("KnownSpells after shipped book = %#x, want Teleport", got)
	}
	if _, remains := carriedBookIndex(mw.world, heroID, 26); remains {
		t.Fatal("shipped Teleport book remained after reading")
	}

	// Mission 80's grave component is disconnected. (27,46) is the nearest
	// passable main-component frontier found by the map census; (14,14) is a
	// passable cell inside the grave trigger's <3 neighbourhood of (16,12).
	const fromX, fromY, graveX, graveY = 27, 46, 14, 14
	planes := mapload.Planes(m, f.Table)
	for _, cell := range [][2]int{{fromX, fromY}, {graveX, graveY}} {
		if planes.Block[cell[1]*m.Width+cell[0]] != 0 {
			t.Fatalf("mission 80 witness cell (%d,%d) is blocked", cell[0], cell[1])
		}
	}
	ms.World = relocateReleaseMissionWorld(t, ms, f.Table, mw.world, heroID, fromX, fromY)
	mw = openMission(ms, f.Table, f.Units, worldFixtureViewer(t, m),
		f.Archives.Containers, f.Faces, f.NPCFaces)
	if mw.fog.exploredAt(graveX, graveY) {
		t.Fatal("mission 80 grave destination is already explored in the frontier fixture")
	}
	mw.attackOrCast(uint32(heroID), 0, 26, graveX, graveY, true)
	if len(mw.pending) != 1 || mw.pending[0].Kind != sim.KindCastAt {
		t.Fatalf("unseen mission 80 Teleport queued %+v, want one KindCastAt", mw.pending)
	}
	arrived := false
	for tick := 0; tick < 160; tick++ {
		mw.tick()
		e := releaseEntity(t, mw, heroID)
		arrived = arrived || (e.X == graveX && e.Y == graveY)
		if arrived && mw.world.Outcome() == sim.OutcomeWon {
			break
		}
	}
	e := releaseEntity(t, mw, heroID)
	if !arrived || e.X != graveX || e.Y != graveY {
		t.Fatalf("mission 80 Teleport ended at (%d,%d), want grave (%d,%d)", e.X, e.Y, graveX, graveY)
	}
	if mw.world.Outcome() != sim.OutcomeWon {
		t.Fatalf("mission 80 outcome at the grave = %v, want won", mw.world.Outcome())
	}
}
