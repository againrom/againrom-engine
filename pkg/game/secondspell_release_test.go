package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// secondSpellMission starts a second-game campaign map with the one-hero
// party ordinary New Game builds and returns its world, its table and the
// hero.
func secondSpellMission(t *testing.T, mission int) (*sim.World, *mapload.Table, sim.EntityID) {
	t.Helper()
	archives := secondGameRoot(t)
	defs, err := LoadDefinitionsFor(archives.Containers, archives.Game())
	if err != nil {
		t.Fatal(err)
	}
	address, _ := MissionMap(mission)
	raw, err := archives.Containers.ReadFile(address)
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.OpenROM2(raw)
	if err != nil {
		t.Fatal(err)
	}
	party := MissionPartyAs(false, defs.StartWeapon, defs.Bodies, defs.Table)
	ms, err := StartMissionFrom(m, "", mission, defs.Table, mapload.DifficultyNormal, party)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ms.World.Entities() {
		if e.Owner == sim.SelfSlot {
			return ms.World, defs.Table, e.ID
		}
	}
	t.Fatal("no hero placed")
	return nil, nil, 0
}

// secondSpellCaster is the first placed mage, not the hero, whose book holds
// spell.
func secondSpellCaster(t *testing.T, w *sim.World, hero sim.EntityID, spell uint16) sim.Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.ID != hero && e.MaxMana > 0 && e.KnownSpells&(1<<spell) != 0 {
			return e
		}
	}
	t.Fatalf("no placed mage knows spell %d", spell)
	return sim.Entity{}
}

func secondSpellRule(t *testing.T, w *sim.World, id uint16) sim.SpellRule {
	t.Helper()
	for _, r := range w.Spells() {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("no rule for spell %d", id)
	return sim.SpellRule{}
}

// TestReleaseSecondGameSpellTableLoads: the second game's table holds 29
// spells past its blank trailing entries, and every row carries its arm.
func TestReleaseSecondGameSpellTableLoads(t *testing.T) {
	archives := secondGameRoot(t)
	defs, err := LoadDefinitionsFor(archives.Containers, archives.Game())
	if err != nil {
		t.Fatal(err)
	}
	if got := defs.Table.Spells.Len(); got != 35 {
		t.Fatalf("Spells collection holds %d entries, want 35 (entry 0, 29 rows, 5 blank)", got)
	}
	spells, err := data.LoadSpells(defs.Table.Spells)
	if err != nil || len(spells) != 29 {
		t.Fatalf("LoadSpells = %d rows, %v; want 29", len(spells), err)
	}
	rules := mapload.SpellRules(defs.Table)
	if len(rules) != 29 {
		t.Fatalf("SpellRules = %d rows, want 29", len(rules))
	}
	for i, r := range rules {
		if r.ID != uint16(i+1) || !r.Second || r.ArmID() != sim.SecondGameSpellArm(r.ID) {
			t.Errorf("row %d: id %d arm %d second %v", i+1, r.ID, r.ArmID(), r.Second)
		}
	}
	heal, drain, prismatic := rules[23], rules[25], rules[10]
	if !heal.Restorative || heal.Damaging || drain.Damaging || drain.Restorative || !prismatic.Damaging {
		t.Errorf("damage flags: Heal %v/%v, Drain Life %v/%v, Prismatic Spray %v", heal.Damaging, heal.Restorative, drain.Damaging, drain.Restorative, prismatic.Damaging)
	}
}

// TestReleaseSecondGameMissionTenMageHealsTheHero: the hero walks to mission
// 10's mage, who heals him with the second game's Heal (row 24) through the
// ordinary engage, paying the row's mana cost.
func TestReleaseSecondGameMissionTenMageHealsTheHero(t *testing.T) {
	w, _, hero := secondSpellMission(t, 10)
	mage := secondSpellCaster(t, w, hero, 24)
	heal := secondSpellRule(t, w, 24)
	if heal.ArmID() != 6 || !heal.Restorative {
		t.Fatalf("row 24 runs arm %d restorative=%v, want the Heal arm", heal.ArmID(), heal.Restorative)
	}
	sim.Step(w, []sim.Command{sim.MoveTo(hero, sim.CellPoint{X: mage.X - 2, Y: mage.Y})})
	paid := int32(-1)
	for tick := 0; tick < 2000; tick++ {
		before, _ := w.Entity(hero)
		caster, _ := w.Entity(mage.ID)
		events := sim.StepObserved(w, nil)
		if now, _ := w.Entity(mage.ID); paid < 0 && now.Mana < caster.Mana {
			paid = caster.Mana - now.Mana
		}
		for _, ev := range events {
			if ev.Caster != mage.ID || ev.Target != hero || ev.Spell != 24 {
				continue
			}
			after, _ := w.Entity(hero)
			if ev.HealthRestored <= 0 || after.HP <= before.HP {
				t.Fatalf("Heal restored %d; hero health %d -> %d", ev.HealthRestored, before.HP, after.HP)
			}
			if paid != heal.ManaCost {
				t.Fatalf("caster paid %d mana, want the row's %d", paid, heal.ManaCost)
			}
			t.Logf("tick %d: mage %d healed the hero by %d (%d -> %d) for %d mana", w.Tick(), mage.ID, ev.HealthRestored, before.HP, after.HP, paid)
			return
		}
	}
	t.Fatal("the mage never healed the hero")
}

// TestReleaseSecondGameMissionTwentyOneMageCastsIceMissile: the hero attacks
// mission 21's Ice Missile mage, who answers with the second game's new
// spell (row 5, no singular arm): it pays the row's cost and its damage lands.
func TestReleaseSecondGameMissionTwentyOneMageCastsIceMissile(t *testing.T) {
	w, _, hero := secondSpellMission(t, 21)
	mage := secondSpellCaster(t, w, hero, 5)
	ice := secondSpellRule(t, w, 5)
	if ice.ArmID() != sim.ArmNone || !ice.Damaging || !sim.SpellRuleLands(ice) {
		t.Fatalf("row 5 runs arm %d damaging=%v", ice.ArmID(), ice.Damaging)
	}
	sim.Step(w, []sim.Command{sim.Attack(hero, mage.ID)})
	released, paid := -1, int32(-1)
	var atRelease int32
	for tick := 0; tick < 2000; tick++ {
		caster, _ := w.Entity(mage.ID)
		events := sim.StepObserved(w, nil)
		if now, _ := w.Entity(mage.ID); paid < 0 && now.Mana < caster.Mana {
			paid = caster.Mana - now.Mana
		}
		for _, ev := range events {
			if released < 0 && ev.Caster == mage.ID && ev.Target == hero && ev.Spell == 5 {
				if paid != ice.ManaCost {
					t.Fatalf("caster paid %d mana, want the row's %d", paid, ice.ManaCost)
				}
				h, _ := w.Entity(hero)
				released, atRelease = tick, h.HP
			}
		}
		if released >= 0 {
			if h, _ := w.Entity(hero); h.HP < atRelease {
				t.Logf("Ice Missile released at step %d for %d mana; hero health %d -> %d at step %d", released, paid, atRelease, h.HP, tick)
				return
			}
			if tick-released > 40 {
				t.Fatal("the Ice Missile released and the hero lost no health")
			}
		}
	}
	t.Fatal("the mage never cast Ice Missile at the hero")
}

// TestReleaseSecondMissionTenHealSaveContinuation: after mission 10's mage
// has healed the hero, an ordinary named SAV and a cold LOAD keep the mage's
// spent mana, his book and the table's arms, and both worlds continue alike.
func TestReleaseSecondMissionTenHealSaveContinuation(t *testing.T) {
	secondGameRoot(t)
	out := secondMissionSaveDirectory(t)
	f, app := secondMissionSettled(t)
	hero := f.live.mission.ids[0]
	mage := secondSpellCaster(t, f.live.world, hero, 24)
	// The hero strikes the nearest creature, then walks to a cell inside the
	// mage's reach short of the meeting that ends the mission.
	victim := secondSpellNearestHostile(t, f.live.world, hero)
	f.live.strike(uint32(hero), uint32(victim))
	start := f.live.world.Tick()
	walked, healed := false, false
	for step := 0; step < 6000 && !healed; step++ {
		if !walked && f.live.world.Tick() >= start+400 {
			f.live.enqueue(uint32(hero), int(mage.X)-7, int(mage.Y))
			walked = true
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		dismissSecondMissionDialogue(t, app, f.live)
		e, _ := f.live.world.Entity(mage.ID)
		healed = e.Mana < e.MaxMana
	}
	if !healed || f.live.world.Outcome() != sim.OutcomeUndecided {
		h, _ := f.live.world.Entity(hero)
		t.Fatalf("the mage cast %v; tick %d hero at %d,%d hp %d outcome %v", healed, f.live.world.Tick(), h.X, h.Y, h.HP, f.live.world.Outcome())
	}
	spent, _ := f.live.world.Entity(mage.ID)
	secondMissionNamedSave(t, f, app, out, "Spell mission")
	cold, a := secondMissionColdAt(t, out, "Spell mission.sav", 10)
	assertCurrentWorldEqual(t, f.live.world, cold.live.world, "cold LOAD")
	loaded, _ := cold.live.world.Entity(mage.ID)
	if loaded.Mana != spent.Mana || loaded.KnownSpells != spent.KnownSpells {
		t.Fatalf("cold LOAD mage mana %d known %b, want %d %b", loaded.Mana, loaded.KnownSpells, spent.Mana, spent.KnownSpells)
	}
	for i, r := range cold.live.world.Spells() {
		if want := f.live.world.Spells()[i]; r != want || !r.Second {
			t.Fatalf("cold LOAD row %d = %+v, want %+v", r.ID, r, want)
		}
	}
	casts := 0
	for range 300 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		e, _ := f.live.world.Entity(mage.ID)
		if e.Mana < spent.Mana {
			casts++
		}
		spent = e
	}
	assertCurrentWorldEqual(t, f.live.world, cold.live.world, "post-load continuation")
	if f.live.world.Hash() != cold.live.world.Hash() {
		t.Fatal("post-load continuation hashes differ")
	}
	t.Logf("M10 Heal cast -> named SAV -> cold LOAD -> 300 identical steps holding %d further cast(s)", casts)
}

// secondSpellNearestHostile is the placed creature nearest the hero that
// carries a creature spell slot.
func secondSpellNearestHostile(t *testing.T, w *sim.World, hero sim.EntityID) sim.EntityID {
	t.Helper()
	h, _ := w.Entity(hero)
	best, dist := sim.EntityID(0), int32(-1)
	for _, e := range w.Entities() {
		if e.CreatureSpells[0].ID == 0 || !e.Alive() {
			continue
		}
		d := max(e.X-h.X, h.X-e.X) + max(e.Y-h.Y, h.Y-e.Y)
		if dist < 0 || d < dist {
			best, dist = e.ID, d
		}
	}
	if dist < 0 {
		t.Fatal("no creature carries a spell slot")
	}
	return best
}
