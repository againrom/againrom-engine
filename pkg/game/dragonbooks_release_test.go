package game

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// dragonClanMission starts mission 131 from the installed map. move, when
// nonzero, relocates that map unit beside the party drop so a fight starts at
// once. classBooks builds the world without the fixed creature books, the
// loss control.
func dragonClanMission(t *testing.T, move uint16, classBooks bool) (*sim.World, sim.EntityID) {
	t.Helper()
	f := releaseFront(t)
	defs, err := LoadDefinitionsFor(f.Archives.Containers, f.Archives.Game())
	if err != nil {
		t.Fatal(err)
	}
	address, _ := MissionMap(131)
	raw, err := f.Archives.Containers.ReadFile(address)
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	if move != 0 {
		moved := false
		for i := range m.Units {
			if m.Units[i].UnitID == move {
				m.Units[i].X, m.Units[i].Y = 15<<8|0x80, 12<<8|0x80
				moved = true
			}
		}
		if !moved {
			t.Fatalf("mission 131 has no unit %d", move)
		}
	}
	table := *defs.Table
	table.ClassCreatureBooks = classBooks
	party := MissionPartyAs(false, defs.StartWeapon, defs.Bodies, &table)
	ms, err := StartMissionFrom(m, "", 131, &table, mapload.DifficultyNormal, party)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ms.World.Entities() {
		if e.Owner == sim.SelfSlot {
			return ms.World, e.ID
		}
	}
	t.Fatal("no hero placed")
	return nil, 0
}

func spellMask(ids ...uint32) uint32 {
	var m uint32
	for _, id := range ids {
		m |= 1 << id
	}
	return m
}

// TestReleaseDragonClanDragonsHoldTheOriginalBooks: the three mission 131
// dragons carry the books and elemental skills the original's own mission
// start save holds for them (Dragon, Dragon.2, Dragon.3 by face). The loss
// control builds the same world without the fixed books and finds them empty.
func TestReleaseDragonClanDragonsHoldTheOriginalBooks(t *testing.T) {
	want := map[uint16]struct {
		known uint32
		skill [6]int32
	}{
		197: {spellMask(2, 8, 11, 14, 27), [6]int32{280, 50, 50, 50, 0, 50}},
		198: {spellMask(2, 3, 13), [6]int32{100, 30, 0, 30, 0, 0}},
		199: {spellMask(2, 3, 11, 13, 27), [6]int32{190, 50, 60, 50, 0, 20}},
	}
	for _, classBooks := range []bool{false, true} {
		w, _ := dragonClanMission(t, 0, classBooks)
		found := 0
		for _, e := range w.Entities() {
			if e.TypeID != 71 {
				continue
			}
			found++
			wnt, ok := want[e.MapUnitID]
			if !ok {
				t.Fatalf("unexpected dragon %d", e.MapUnitID)
			}
			if !e.Book.HasInstances() {
				t.Fatalf("dragon %d book %v", e.MapUnitID, e.Book.State)
			}
			if classBooks {
				if e.KnownSpells != 0 {
					t.Fatalf("loss control: dragon %d knows %b", e.MapUnitID, e.KnownSpells)
				}
				continue
			}
			if e.KnownSpells != wnt.known || [6]int32(e.Skill) != wnt.skill {
				t.Fatalf("dragon %d knows %b skills %v, want %b %v", e.MapUnitID, e.KnownSpells, e.Skill, wnt.known, wnt.skill)
			}
			for id := uint32(1); id <= 28; id++ {
				slot := e.Book.Slots[id-1]
				if e.KnownSpells&(1<<id) == 0 {
					if slot != (sim.BookSpell{}) {
						t.Fatalf("dragon %d slot %d filled", e.MapUnitID, id)
					}
					continue
				}
				rule := w.Spells()[id-1]
				if slot.ManaCost != uint16(rule.ManaCost) || slot.Range != rule.MaxRange {
					t.Fatalf("dragon %d spell %d entry %+v, rule cost %d range %d", e.MapUnitID, id, slot, rule.ManaCost, rule.MaxRange)
				}
			}
		}
		if found != 3 {
			t.Fatalf("mission 131 dragons=%d, want 3", found)
		}
	}
}

// TestReleaseDragonClanDragonCastsInAFight: dragon 197 placed beside the
// village casts a spell from its book in the fight; the loss control, without
// the fixed books, casts nothing over the same steps.
func TestReleaseDragonClanDragonCastsInAFight(t *testing.T) {
	const ticks = 3000
	run := func(classBooks bool) (uint16, int) {
		w, hero := dragonClanMission(t, 197, classBooks)
		var dragon sim.EntityID
		for _, e := range w.Entities() {
			if e.MapUnitID == 197 && e.TypeID == 71 {
				dragon = e.ID
			}
		}
		if dragon == 0 {
			t.Fatal("dragon 198 not placed")
		}
		sim.Step(w, []sim.Command{sim.Attack(hero, dragon)})
		for tick := 0; tick < ticks; tick++ {
			for _, ev := range sim.StepObserved(w, nil) {
				if ev.Caster == dragon && ev.Spell != 0 {
					return ev.Spell, tick
				}
			}
			if d, ok := w.Entity(dragon); !ok || d.HP <= 0 {
				return 0, tick
			}
		}
		return 0, ticks
	}
	spell, tick := run(false)
	if spell == 0 || spellMask(2, 8, 11, 14, 27)&(1<<spell) == 0 {
		t.Fatalf("dragon 197 cast %d by step %d, want one of its spells 2, 8, 11, 14, 27", spell, tick)
	}
	t.Logf("dragon 197 cast spell %d at step %d", spell, tick)
	if s, at := run(true); s != 0 {
		t.Fatalf("loss control: dragon without the fixed book cast %d at step %d", s, at)
	}
}
