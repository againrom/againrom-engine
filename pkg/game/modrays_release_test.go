package game

import (
	"slices"
	"testing"

	"againrom/pkg/sim"
)

const sprayEnemies = 12

func raysModFront(t *testing.T, cap string) *FrontEnd {
	t.Helper()
	return targetsModFront(t, func(name func(int) string) string {
		row := "[[spell]]\ntarget = \"" + name(sim.PrismaticSpellID) + "\"\nmana = 1\n"
		if cap != "" {
			row += "rays = " + cap + "\n"
		}
		return row
	})
}

func raysMission(t *testing.T, f *FrontEnd) (sim.EntityID, []sim.EntityID) {
	t.Helper()
	casterID, allyID := targetsMission(t, f)
	mw := f.live
	caster, _ := mw.entity(casterID)
	rule, ok := mw.world.Spell(sim.PrismaticSpellID)
	if !ok {
		t.Fatal("no Prismatic Spray row")
	}
	book := caster.Book
	book.Slots[sim.PrismaticSpellID-1] = sim.BookSpell{Range: rule.MaxRange, ManaCost: uint16(rule.ManaCost)}
	if err := mw.world.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{
		ID: casterID, KnownSpells: caster.KnownSpells | 1<<sim.PrismaticSpellID, Book: book,
	}}); err != nil {
		t.Fatal(err)
	}
	relations := mw.world.Relations()
	var cells [][2]int32
	for r := int32(1); r <= 3; r++ {
		for dx := -r; dx <= r; dx++ {
			for dy := -r; dy <= r; dy++ {
				if max(dx, -dx, dy, -dy) == r {
					cells = append(cells, [2]int32{caster.X + dx, caster.Y + dy})
				}
			}
		}
	}
	var placed []sim.EntityID
	for _, e := range mw.world.Entities() {
		if len(placed) == sprayEnemies {
			break
		}
		if slices.Contains([]sim.EntityID{casterID, allyID}, e.ID) || !e.Alive() || e.OffMap || !relations.Hostile(sim.SelfSlot, e.Owner) {
			continue
		}
		for len(cells) > 0 {
			c := cells[0]
			cells = cells[1:]
			if mw.world.HeadlessPlace(e.ID, c[0], c[1]) == nil {
				placed = append(placed, e.ID)
				break
			}
		}
	}
	if len(placed) < sprayEnemies {
		t.Fatalf("only %d hostile units could be placed around the caster, want %d", len(placed), sprayEnemies)
	}
	return casterID, placed
}

func hitCount(mw *mapWorld, ids []sim.EntityID, before map[sim.EntityID]int32) int {
	n := 0
	for _, id := range ids {
		if e, ok := mw.entity(id); !ok || e.HP < before[id] {
			n++
		}
	}
	return n
}

func healthOf(mw *mapWorld, ids []sim.EntityID) map[sim.EntityID]int32 {
	out := map[sim.EntityID]int32{}
	for _, id := range ids {
		out[id] = hpOf(mw, id)
	}
	return out
}

func TestReleaseModRaysHitAndDrawEveryEnemyAndSurviveSaveAndLoad(t *testing.T) {
	f := raysModFront(t, "100")
	casterID, enemies := raysMission(t, f)
	mw := f.live
	if rule, _ := mw.world.Spell(sim.PrismaticSpellID); rule.Rays != 100 {
		t.Fatalf("mission world row %+v", rule)
	}
	before := healthOf(mw, enemies)
	caster, _ := mw.entity(casterID)
	mw.castAt(uint32(casterID), uint32(enemies[0]), sim.PrismaticSpellID)
	var inFlight []byte
	drawn := 0
	for tick := 0; hitCount(mw, enemies, before) < len(enemies); tick++ {
		if tick > 400 {
			t.Fatalf("only %d of %d enemies were hit", hitCount(mw, enemies, before), len(enemies))
		}
		if now, _ := mw.entity(casterID); inFlight == nil && now.Mana < caster.Mana && hitCount(mw, enemies, before) == 0 {
			inFlight = exportSave(t, f)
		}
		mw.tick()
		drawn = max(drawn, len(mw.bolts))
	}
	if drawn < len(enemies) {
		t.Errorf("%d ray figures drawn for %d enemies", drawn, len(enemies))
	}
	if inFlight == nil {
		t.Fatal("no SAVE was taken with the rays in flight")
	}

	g := raysModFront(t, "100")
	open, town, err := g.RestoreOriginal(inFlight)
	if err != nil || town {
		t.Fatalf("cold load: town %v err %v", town, err)
	}
	if err := g.App("cold load").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	late := g.live
	if rule, _ := late.world.Spell(sim.PrismaticSpellID); rule.Rays != 100 {
		t.Fatalf("restored row %+v", rule)
	}
	restored := healthOf(late, enemies)
	for tick := 0; hitCount(late, enemies, restored) < len(enemies); tick++ {
		if tick > 400 {
			t.Fatalf("after the cold LOAD only %d of %d enemies were hit", hitCount(late, enemies, restored), len(enemies))
		}
		late.tick()
	}

	plain := releaseFront(t)
	if _, _, err := plain.RestoreOriginal(inFlight); err == nil {
		t.Error("a game without the mod loaded the SAV written under it")
	}
}

func TestReleaseWithoutRaysTheSprayStopsAtTheOriginalCap(t *testing.T) {
	f := raysModFront(t, "")
	casterID, enemies := raysMission(t, f)
	mw := f.live
	mw.castAt(uint32(casterID), uint32(enemies[0]), sim.PrismaticSpellID)
	drawn := 0
	for tick := 0; tick < 12; tick++ {
		mw.tick()
		drawn = max(drawn, len(mw.bolts))
	}
	if drawn < 1 || drawn > 7 {
		t.Errorf("%d ray figures drawn with no cap, want 1 to 7", drawn)
	}
}
