package sim

import "testing"

// An unbidden Bless never chooses a mage, the caster included; a warrior
// stays its target, and a player's order still casts it on any mage (owner,
// DIV-4830, DIV-4831). A mage is an actor with a mana pool.

func blessMageCompanion(id EntityID, x int32) Entity {
	e := spMage(id, x, 2, 30, 40, 40, 0)
	e.Owner = SelfSlot
	return e
}

// The level policy of the human player's mage: Average lists Bless then Haste
// per candidate, the caster first.
func TestAnAutomaticBlessSparesEveryMageAndTakesAWarrior(t *testing.T) {
	for _, tc := range []struct {
		name      string
		setting   AutoCastSetting
		known     uint32
		companion Entity
		spell     uint16
		target    EntityID
		cast      bool
	}{
		{"Bless passes over its caster to the warrior", AutoCastAverage | AutoCastOwn, acsMask(23), acsUnit(2, SelfSlot, 4, 100), 23, 2, true},
		{"Bless with only mages in sight casts nothing", AutoCastAverage | AutoCastOwn, acsMask(23), blessMageCompanion(2, 4), 0, 0, false},
		{"the caster takes Haste where Bless spares it", AutoCastAverage | AutoCastOwn, acsMask(23, 24), acsUnit(2, SelfSlot, 4, 100), 24, 1, true},
		{"Shield still lands on its caster", AutoCastMinimum | AutoCastOwn, acsMask(18, 23), acsUnit(2, SelfSlot, 4, 100), 18, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := acsWorld(t, Relations{}, tc.setting, tc.known, 0, tc.companion)
			spell, target, cast := acsLevel(w)
			if cast != tc.cast || spell != tc.spell || target != tc.target {
				t.Fatalf("cast %v spell %d at %d, want %v spell %d at %d", cast, spell, target, tc.cast, tc.spell, tc.target)
			}
		})
	}
}

// A heal is not spared: the hurt mage heals itself first.
func TestAnAutomaticHealStillTakesItsMageCaster(t *testing.T) {
	w := acsWorld(t, Relations{}, AutoCastAverage|AutoCastOwn, acsMask(6, 23), 0, acsUnit(2, SelfSlot, 4, 100))
	w.entities[0].HP = 40
	if spell, target, cast := acsLevel(w); !cast || spell != 6 || target != 1 {
		t.Fatalf("cast %v spell %d at %d, want Heal at the caster", cast, spell, target)
	}
}

// A Bless armed in the book is cast on its own: the target is chosen
// automatically, so it spares mages as the level policy does.
func TestAnArmedBlessSparesEveryMage(t *testing.T) {
	w := acsWorld(t, Relations{}, 0, acsMask(23), 23, acsUnit(2, SelfSlot, 4, 100))
	if spell, target, cast := acsFirst(w); !cast || spell != 23 || target != 2 {
		t.Fatalf("armed Bless: cast %v spell %d at %d, want 23 at the warrior 2", cast, spell, target)
	}
	w = acsWorld(t, Relations{}, 0, acsMask(23), 23, blessMageCompanion(2, 4))
	if spell, target, cast := acsFirst(w); cast {
		t.Fatalf("armed Bless with only mages: cast %d at %d", spell, target)
	}
}

// The player's order is not the automatic choice: it casts Bless on the
// caster and on another mage.
func TestAnOrderedBlessStillLandsOnAMage(t *testing.T) {
	for _, target := range []EntityID{1, 2} {
		w := acsWorld(t, Relations{}, 0, acsMask(23), 0, blessMageCompanion(2, 4))
		Step(w, []Command{Cast(1, target, SpellID(23))})
		k, ok := w.bookCastIndex(1)
		if !ok || w.bookCasts[k].Spell != 23 || w.bookCasts[k].Target != target {
			t.Fatalf("ordered Bless at mage %d began %v %+v", target, ok, w.bookCasts)
		}
	}
}

// blessSlotCreature is a computer creature whose one slot always draws Bless,
// its book holding Bless and Fire Arrow; mage gives it a mana pool.
func blessSlotCreature(t *testing.T, second, mage bool) *World {
	t.Helper()
	bless, arrow := uint32(23), uint32(1)
	rows := []SpellRule{hlArrow(), acsBuff(23)}
	if second {
		bless = 20
		rows = []SpellRule{secondRow(hlArrow()), secondRow(acsBuff(20))}
	}
	book, known := creatureBookOf(map[uint32]uint8{bless: 7, arrow: 7})
	book.Slots[bless-1].Defensive = 1
	var w *World
	if second {
		w = secondCreatureWorld(t, rows, creatureCertainSlot(bless), known, book)
	} else {
		w = creatureWorld(t, rows, creatureCertainSlot(bless), known, book)
	}
	if mage {
		e := &w.entities[indexOfEntity(w.entities, 1)]
		e.MaxMana, e.Mana = 50, 50
	}
	return w
}

// The computer's slot draw aims Bless at the caster. A mage drops the pick
// and its engage follows: here its book's Fire Arrow at the victim. A
// creature with no mana pool still blesses itself.
func TestAComputerMageDrawingBlessDoesNotBlessItself(t *testing.T) {
	for _, tc := range []struct {
		name         string
		second, mage bool
		spell        uint16
		target       EntityID
	}{
		{"first game mage", false, true, 1, 2},
		{"first game creature", false, false, 23, 1},
		{"second game mage", true, true, 1, 2},
		{"second game creature", true, false, 20, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := blessSlotCreature(t, tc.second, tc.mage)
			w.orderAttack(0, 2)
			k, ok := w.bookCastIndex(1)
			if !ok {
				t.Fatal("the creature began no cast")
			}
			if c := w.bookCasts[k]; c.Spell != tc.spell || c.Target != tc.target {
				t.Fatalf("cast spell %d at %d, want %d at %d", c.Spell, c.Target, tc.spell, tc.target)
			}
		})
	}
}
