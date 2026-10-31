package game

import (
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
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
		if got, ok := stack.Instance().BookSpell(rules.SpellGameFirst); ok && got == spell {
			return i, true
		}
	}
	return 0, false
}

// The shipped Astral book teaches Teleport; from the river bank grave ground
// is in sight and takes the cast. Census of the shipped map on both installs:
// 4-connected passable components (main 7803 cells, grave 1777); of 2408 main
// cells within Chebyshev 34 of the grave, 381 see it. Bank (29,48), landing
// (28,36) is the pair nearest the trigger.
func TestReleaseMission80AstralBookTeachesTeleportAndReachesTheGrave(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	hero := data.Hero{Body: 50, Reaction: 50, Mind: 50, Spirit: 50}
	// Slot 5 is Astral for a mage. Mind 50 + skill 80 - 30 reaches the
	// simulation's power ceiling 100, giving Teleport range 34.
	hero.Skill[5] = 80
	party := []mapload.PartyMember{{
		ID: "mission80-mage", Class: 100, Mage: true, StartingHero: true, PlayerCharacter: true,
		Hero:    hero,
		Profile: data.Profile{HealthColumn: true, ManaColumn: true},
	}}
	a := f.App("mission 80 teleport")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpenerWith(80, party)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	if live == nil || len(live.mission.ids) != 1 {
		t.Fatalf("mission 80 opened with %+v, want one party id", live)
	}
	m := live.mission.state.Map
	heroID := live.mission.ids[0]
	if releaseEntity(t, live, heroID).KnownSpells&(uint32(1)<<26) != 0 {
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
	ownerItems, ok := live.world.CarriedItems(bookOwner)
	if !ok {
		t.Fatalf("mission 80 world has no unit-143 entity %d", bookOwner)
	}
	books := 0
	for _, item := range ownerItems {
		if spell, readable := item.BookSpell(rules.SpellGameFirst); readable && spell == 26 {
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
	live.world = relocateReleaseMissionWorld(t, live.mission.state, f.Table, live.world, heroID, 37, 129)
	for tick := 0; tick < 128; tick++ {
		if _, got := carriedBookIndex(live.world, heroID, 26); got {
			break
		}
		sim.Step(live.world, nil)
	}
	bookIndex, gotBook := carriedBookIndex(live.world, heroID, 26)
	if !gotBook {
		t.Fatalf("mission 80 Give All did not transfer unit 143's Teleport book")
	}
	if _, still := carriedBookIndex(live.world, bookOwner, 26); still {
		t.Fatal("mission 80 Give All duplicated the Teleport book")
	}
	live.enqueueEquip(bookIndex)
	if len(live.pending) != 1 || live.pending[0].Kind != sim.KindReadBook {
		t.Fatalf("shipped book use queued %+v, want one KindReadBook", live.pending)
	}
	live.tick()
	if got := releaseEntity(t, live, heroID).KnownSpells; got&(uint32(1)<<26) == 0 {
		t.Fatalf("KnownSpells after shipped book = %#x, want Teleport", got)
	}
	if _, remains := carriedBookIndex(live.world, heroID, 26); remains {
		t.Fatal("shipped Teleport book remained after reading")
	}

	const bankX, bankY, landX, landY, trigX, trigY = 29, 48, 28, 36, 14, 14
	planes := mapload.Planes(m, f.Table)
	for _, cell := range [][2]int{{bankX, bankY}, {landX, landY}, {trigX, trigY}} {
		if planes.Block[cell[1]*m.Width+cell[0]] != 0 {
			t.Fatalf("mission 80 witness cell (%d,%d) is blocked", cell[0], cell[1])
		}
	}
	live.world = relocateReleaseMissionWorld(t, live.mission.state, f.Table, live.world, heroID, bankX, bankY)
	live.fog.refresh(live.world, sim.SelfSlot)
	// The reading notice takes the selection keys; Enter closes it.
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	inspectionCentre(live, bankX, bankY)
	live.push()
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("e"); err != nil {
		t.Fatal(err)
	}
	live.push()
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.HeadlessSpellPoint(26); err != nil {
		if err := a.HeadlessKey("book"); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	sx, sy, err := a.HeadlessSpellPoint(26)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, sx, sy); err != nil {
			t.Fatal(err)
		}
	}
	if _, current, armed := live.view.QuickSpellState(); current != 26 || !armed {
		t.Fatalf("the book click selected %d armed %v, want Teleport armed", current, armed)
	}

	hover := func(col, row int) string {
		t.Helper()
		inspectionCentre(live, col, row)
		live.push()
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		x, y, err := a.HeadlessCellPoint(col, row)
		if err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessPointer("hover", x, y); err != nil {
			t.Fatal(err)
		}
		got, shown := a.HeadlessMapCursor()
		if !shown {
			t.Fatal("no map cursor")
		}
		return got
	}
	// Control: a grave-component cell the mage does not see answers move.
	const hiddenX, hiddenY = 14, 20
	if live.fog.project()[hiddenY*m.Width+hiddenX] == ui.FogVisible {
		t.Fatalf("control cell (%d,%d) is in sight from the bank", hiddenX, hiddenY)
	}
	if got := hover(hiddenX, hiddenY); got != "move" {
		t.Fatalf("Teleport over the unseen grave cell (%d,%d): cursor %q, want move", hiddenX, hiddenY, got)
	}
	if got := hover(landX, landY); got != "cast" {
		t.Fatalf("Teleport over the grave cell in sight (%d,%d): cursor %q, want cast", landX, landY, got)
	}
	x, y, err := a.HeadlessCellPoint(landX, landY)
	if err != nil {
		t.Fatal(err)
	}
	live.pending = nil
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	if len(live.pending) != 1 || live.pending[0].Kind != sim.KindCastAt {
		t.Fatalf("the click in sight queued %+v, want one KindCastAt", live.pending)
	}
	arrived := false
	for tick := 0; tick < 160 && !arrived; tick++ {
		live.tick()
		e := releaseEntity(t, live, heroID)
		arrived = e.X == landX && e.Y == landY
	}
	if !arrived {
		e := releaseEntity(t, live, heroID)
		t.Fatalf("mission 80 Teleport ended at (%d,%d), want landing (%d,%d)", e.X, e.Y, landX, landY)
	}
	live.pending = append(live.pending, sim.GroupMoveTo(heroID, sim.CellPoint{X: trigX, Y: trigY}, 1))
	for tick := 0; tick < 1500 && live.world.Outcome() != sim.OutcomeWon; tick++ {
		live.tick()
	}
	if live.world.Outcome() != sim.OutcomeWon {
		e := releaseEntity(t, live, heroID)
		t.Fatalf("mission 80 outcome at (%d,%d) = %v, want won", e.X, e.Y, live.world.Outcome())
	}
}
