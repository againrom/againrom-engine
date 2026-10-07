package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The placed mage of the campaign's "Down with orcs" mission is entity 0, the
// world's first id. Selecting him must open his own book, show his spells and
// cast from it like any other unit's.
const spellbookMission = 71

func spellbookMissionApp(t *testing.T) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	a := f.App("any unit spellbook")
	t.Cleanup(a.StopAudio)
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpener(spellbookMission)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8 && a.HeadlessNoticeOpen(); i++ {
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	return f, a
}

// spellbookShow selects id on the map and leaves his book on screen.
func spellbookShow(t *testing.T, a *ui.App, live *mapWorld, id sim.EntityID) {
	t.Helper()
	e, ok := live.entity(id)
	if !ok {
		t.Fatalf("entity %d is absent", id)
	}
	inspectionCentre(live, int(e.X), int(e.Y))
	live.push()
	if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	live.push()
	if err := a.HeadlessPointer("hover", -1, -1); err != nil {
		t.Fatal(err)
	}
	if got := a.HeadlessSelection(); len(got) != 1 || got[0] != uint32(id) {
		t.Fatalf("selection %v, want [%d]", got, id)
	}
	for i := 0; i < 2; i++ {
		if _, _, err := a.HeadlessSpellPoint(1); err == nil {
			return
		}
		if err := a.HeadlessKey("book"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReleaseSecondMageOfDownWithOrcsOpensHisBookAndCasts(t *testing.T) {
	f, a := spellbookMissionApp(t)
	live := f.live
	const mage = sim.EntityID(0)
	caster, ok := live.entity(mage)
	if !ok || caster.KnownSpells == 0 || caster.Owner != sim.SelfSlot {
		t.Fatalf("mission %d no longer places a player mage as entity 0: %+v", spellbookMission, caster)
	}
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	spellbookShow(t, a, live, mage)
	var pointSpell uint32
	for _, r := range live.world.Spells() {
		if caster.KnownSpells&(1<<r.ID) == 0 {
			continue
		}
		if _, _, err := a.HeadlessSpellPoint(uint32(r.ID)); err != nil {
			t.Fatalf("known spell %d is not in the open book: %v", r.ID, err)
		}
		if pointSpell == 0 && !r.TargetsUnit && r.Area {
			pointSpell = uint32(r.ID)
		}
	}
	if pointSpell == 0 {
		t.Fatal("the mage knows no point-target spell")
	}
	for id := uint16(1); id <= 28; id++ {
		if caster.KnownSpells&(1<<id) != 0 {
			continue
		}
		if x, y, err := a.HeadlessSpellPoint(uint32(id)); err == nil {
			// An unknown spell's cell is drawn but unavailable: a click arms nothing.
			for _, edge := range []string{"press", "release"} {
				if err := a.HeadlessPointer(edge, x, y); err != nil {
					t.Fatal(err)
				}
			}
			if _, selected, armed := live.view.QuickSpellState(); selected != 0 || armed {
				t.Fatalf("unknown spell %d armed", id)
			}
			break
		}
	}
	x, y, err := a.HeadlessSpellPoint(pointSpell)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	if _, selected, armed := live.view.QuickSpellState(); selected != pointSpell || !armed {
		t.Fatalf("book click selected %d armed %v, want %d armed", selected, armed, pointSpell)
	}
	targetX, targetY := caster.X+2, caster.Y
	inspectionCentre(live, int(targetX), int(targetY))
	live.push()
	spellbookClickCell(t, a, int(targetX), int(targetY))
	if len(live.pending) != 1 || live.pending[0].Kind != sim.KindCastAt || live.pending[0].Entity != mage ||
		live.pending[0].Spell != uint16(pointSpell) || live.pending[0].X != targetX || live.pending[0].Y != targetY {
		t.Fatalf("pointer produced %+v, want one cast of spell %d by entity 0 at %d,%d", live.pending, pointSpell, targetX, targetY)
	}
	for tick := 0; tick < 600; tick++ {
		live.tick()
		if got, _ := live.entity(mage); got.Mana < caster.Mana {
			return
		}
	}
	got, _ := live.entity(mage)
	t.Fatalf("the mage never paid for the cast: at %d,%d hp %d mana %d", got.X, got.Y, got.HP, got.Mana)
}

// spellbookReplaceEntities rebuilds the live world with edit applied to the
// named entities, the way a different mission start would have minted them.
func spellbookReplaceEntities(t *testing.T, f *FrontEnd, edit map[sim.EntityID]func(*sim.Entity)) {
	t.Helper()
	live := f.live
	source := live.world
	ents := source.Entities()
	for i := range ents {
		if fn, ok := edit[ents[i].ID]; ok {
			fn(&ents[i])
		}
	}
	w, err := sim.NewStructuredWorld(mapload.Seed, source.Bounds(), sim.ModeCanonical,
		mapload.Planes(live.mission.state.Map, f.Table), ents, source.Script(), source.Relations(), source.Sacks(),
		source.Stock(), source.Spells(), source.Ghost(), source.Structures())
	if err != nil {
		t.Fatal(err)
	}
	if err := w.DeclareItemWeights(source.ItemWeights()); err != nil {
		t.Fatal(err)
	}
	live.world = w
	live.push()
}

func spellbookClick(t *testing.T, a *ui.App, x, y int) {
	t.Helper()
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
}

// A controlled creature that holds a book opens it and arms a spell from it,
// while a controlled warrior with no book gets the same bar with every cell
// unavailable.
func TestReleaseControlledCreatureOpensItsBookAndWarriorShowsItEmpty(t *testing.T) {
	f, a := spellbookMissionApp(t)
	live := f.live
	const creature, warrior = sim.EntityID(0), sim.EntityID(62)
	if w, _ := live.entity(warrior); w.KnownSpells != 0 || w.Owner != sim.SelfSlot {
		t.Fatalf("warrior fixture is %+v", w)
	}
	spellbookShow(t, a, live, warrior)
	for _, id := range []uint32{1, 6, 12} {
		x, y, err := a.HeadlessSpellPoint(id)
		if err != nil {
			t.Fatalf("the warrior's book bar is missing spell cell %d: %v", id, err)
		}
		spellbookClick(t, a, x, y)
		if _, selected, armed := live.view.QuickSpellState(); selected != 0 || armed {
			t.Fatalf("warrior armed spell %d", selected)
		}
	}

	spellbookReplaceEntities(t, f, map[sim.EntityID]func(*sim.Entity){
		creature: func(e *sim.Entity) {
			e.Humanoid = false
			e.Mana, e.MaxMana = 60, 60
			e.Book = sim.Spellbook{State: sim.BookPresent}
			e.KnownSpells = 0
			for _, r := range live.world.Spells() {
				if r.ID == 1 || r.ID == 6 {
					sim.LearnBookSpell(e, r.ID, live.world.Spells())
				}
			}
			e.KnownSpells |= 1<<1 | 1<<6
		},
	})
	if c, ok := live.entity(creature); !ok || c.Humanoid || c.KnownSpells != 1<<1|1<<6 || c.Owner != sim.SelfSlot {
		t.Fatalf("creature fixture is %+v", c)
	}

	spellbookShow(t, a, live, creature)
	for _, id := range []uint32{1, 6} {
		x, y, err := a.HeadlessSpellPoint(id)
		if err != nil {
			t.Fatalf("the creature's spell %d is not in its book: %v", id, err)
		}
		spellbookClick(t, a, x, y)
		if _, selected, armed := live.view.QuickSpellState(); selected != id || !armed {
			t.Fatalf("creature book click selected %d armed %v, want %d armed", selected, armed, id)
		}
	}

}

// spellbookClickCell presses and releases on a visible ground pixel of the
// world cell (x, y).
func spellbookClickCell(t *testing.T, app *ui.App, x, y int) {
	t.Helper()
	for py := 100; py < 560; py += 4 {
		for px := 160; px < 750; px += 4 {
			if cx, cy, err := app.HeadlessDropCell(px, py); err == nil && cx == x && cy == y {
				spellbookClick(t, app, px, py)
				return
			}
		}
	}
	t.Fatalf("cell (%d,%d) has no visible ground pointer witness", x, y)
}
