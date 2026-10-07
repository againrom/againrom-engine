package sim

import (
	"reflect"
	"testing"
)

func TestImportOriginalActorSpellbooksOnlyChangesBooksAndIsAtomic(t *testing.T) {
	a, b := spMage(0, 2, 2, 30, 50, 17, 1<<6), spMage(1, 5, 2, 30, 50, 17, 1<<6)
	w := spWorld(t, 1105, nil, a, b)
	before := append([]Entity(nil), w.Entities()...)
	book := Spellbook{State: BookPresent}
	book.Slots[0] = BookSpell{7, 255, 65535}
	good := OriginalActorSpellbook{ID: 0, KnownSpells: 1 << 1, Book: book}
	for name, bad := range map[string]OriginalActorSpellbook{
		"duplicate":            good,
		"missing":              {ID: 99, Book: Spellbook{State: BookAbsent}},
		"legacy":               {ID: 1},
		"bad presence":         {ID: 1, Book: Spellbook{State: 255}},
		"absent membership":    {ID: 1, KnownSpells: 1 << 1, Book: Spellbook{State: BookAbsent}},
		"unlearned parameters": {ID: 1, Book: book},
		"bad membership":       {ID: 1, KnownSpells: 1, Book: Spellbook{State: BookPresent}},
	} {
		t.Run(name, func(t *testing.T) {
			hash := w.Hash()
			if err := w.ImportOriginalActorSpellbooks([]OriginalActorSpellbook{good, bad}); err == nil || w.Hash() != hash {
				t.Fatalf("partial/accepted bad batch: %v", err)
			}
		})
	}
	for _, dead := range []bool{false, true} {
		w.entities[1].OffMap = !dead
		if dead {
			w.entities[1].HP = 0
		}
		hash := w.Hash()
		err := w.ImportOriginalActorSpellbooks([]OriginalActorSpellbook{good, {ID: 1, Book: Spellbook{State: BookAbsent}}})
		if !dead {
			if err != nil || !w.entities[1].OffMap || w.entities[1].Book.State != BookAbsent || w.entities[0].Book != good.Book {
				t.Fatal("current detached books were not imported", err)
			}
		} else if err == nil || w.Hash() != hash {
			t.Fatal("nonliving target changed batch", err)
		}
	}
	w.entities[1] = before[1]
	if err := w.ImportOriginalActorSpellbooks([]OriginalActorSpellbook{good, {ID: 1, Book: Spellbook{State: BookAbsent}}}); err != nil {
		t.Fatal(err)
	}
	before[0].KnownSpells, before[0].Book = good.KnownSpells, good.Book
	before[1].KnownSpells, before[1].Book = 0, Spellbook{State: BookAbsent}
	if !reflect.DeepEqual(before, w.Entities()) {
		t.Fatal("import changed unrelated state")
	}
}

func TestImportedNonPartyBookAIUsesRawDefensiveAndSignedCost(t *testing.T) {
	rule := SpellRule{ID: 1, ManaCost: 99, School: 1, MaxRange: 1, DamageMin: 10, DamageMax: 10, Damaging: true, TargetsUnit: true, Defensive: true}
	for _, defensive := range []byte{0, 1, 2, 255} {
		mage := spMage(0, 2, 2, 30, 50, 0, 0)
		mage.Owner = 2
		victim := spEnt(1, 5, 2)
		victim.Owner = SelfSlot
		w := hlWorld(t, 1105, acEnemies(t), []SpellRule{rule}, mage, victim)
		book := Spellbook{State: BookPresent}
		book.Slots[0] = BookSpell{7, defensive, 65535}
		if err := w.ImportOriginalActorSpellbooks([]OriginalActorSpellbook{{ID: 0, KnownSpells: 1 << 1, Book: book}}); err != nil {
			t.Fatal(err)
		}
		if got := w.aiCast(0, nil); got != (defensive == 0) {
			t.Fatalf("raw Defensive%d: AI ordered %t", defensive, got)
		}
		if defensive != 0 {
			continue
		}
		spRunUnbidden(w)
		if w.entities[0].Mana != 1 || w.entities[1].HP >= 100 {
			t.Fatal("imported AI cast lost signed cost or range")
		}
		if w.Spells()[0] != rule {
			t.Fatal("instance contaminated canonical spell table")
		}
	}
}

// A source block that is all zero leaves a creature constructed with class
// slots holding them; a nonzero source replaces them; an actor with no slots
// takes the source whole.
func TestImportOriginalCreatureSlotsKeepClassSlotsOverAZeroSource(t *testing.T) {
	class := [CreatureSpellSlots]CreatureSpell{{ID: 9, Threshold: 3270}}
	a, b := spMage(0, 2, 2, 30, 50, 17, 1<<6), spMage(1, 5, 2, 30, 50, 17, 1<<6)
	a.CreatureSpells, b.CreatureSpells = class, class
	w := spWorld(t, 1105, nil, a, b)
	other := [CreatureSpellSlots]CreatureSpell{{ID: 3, Threshold: 100}}
	book := Spellbook{State: BookAbsent}
	if err := w.ImportOriginalActorSpellbooks([]OriginalActorSpellbook{
		{ID: 0, Book: book, HasCreatureSpells: true},
		{ID: 1, Book: book, HasCreatureSpells: true, CreatureSpells: other},
	}); err != nil {
		t.Fatal(err)
	}
	if got := w.entities[0].CreatureSpells; got != class {
		t.Fatalf("a zero source replaced the class slots: %v", got)
	}
	if got := w.entities[1].CreatureSpells; got != other {
		t.Fatalf("a nonzero source did not replace the class slots: %v", got)
	}
}
