package sim

import (
	"bytes"
	"testing"
)

func TestSourceBookLiteralWireAndInvalidStates(t *testing.T) {
	// Literal wire expectation, independent of the encoder and struct layout.
	want := make([]byte, 113)
	copy(want, []byte{2, 0x12, 2, 0x34, 0x80})
	copy(want[109:], []byte{0xfe, 0xff, 0xcd, 0xab})
	book := Spellbook{State: BookPresent}
	book.Slots[0], book.Slots[27] = BookSpell{0x12, 2, 0x8034}, BookSpell{0xfe, 0xff, 0xabcd}
	known := uint32(1<<1 | 1<<28)
	encoded := make([]byte, spellbookRecordLen)
	encodeSpellbook(encoded, book)
	if !bytes.Equal(encoded, want) {
		t.Fatal("book wire layout changed")
	}
	back, err := decodeSpellbook(want, known)
	if err != nil || back != book {
		t.Fatal("literal book decode", err)
	}
	for _, state := range []byte{0, 1, 3, 255} {
		broken := append([]byte(nil), want...)
		broken[0] = state
		if _, err := decodeSpellbook(broken, known); err == nil {
			t.Fatalf("invalid state%d admitted", state)
		}
	}
	if _, err := decodeSpellbook(want, 1<<1); err == nil {
		t.Fatal("unlearned instance admitted")
	}
	if _, err := decodeSpellbook(want, known|1); err == nil {
		t.Fatal("slot zero admitted")
	}
}

func TestSourceBookAIUsesInstanceCostAndExactZeroDefensive(t *testing.T) {
	rule := SpellRule{ID: 1, ManaCost: 99, School: 1, MaxRange: 1, DamageMin: 10, DamageMax: 10, Damaging: true, TargetsUnit: true, Defensive: true}
	for _, defensive := range []uint8{0, 1, 2} {
		mage := sourceMage1101(1, 1, BookSpell{5, defensive, 3})
		mage.Owner, mage.Mind, mage.Mana = 2, 30, 3
		target := spEnt(2, 5, 2)
		target.Owner = SelfSlot
		w := hlWorld(t, 31, acEnemies(t), []SpellRule{rule}, mage, target)
		if got := w.aiCast(0, nil); got != (defensive == 0) {
			t.Fatalf("AI defensive%d admitted%v", defensive, got)
		}
		if defensive == 0 {
			spRunUnbidden(w)
			if w.entities[0].Mana != 0 || w.entities[1].HP != 90 {
				t.Fatal("AI ignored instance cost or range")
			}
		}
	}
}

func sourceMage1101(id EntityID, spell uint16, s BookSpell) Entity {
	e := spMage(id, 2, 2, 60, 100, 100, uint32(1)<<spell)
	e.Book.State = BookPresent
	e.Book.Slots[spell-1] = s
	return e
}

func TestSourceBookInstancesDrivePopupAdmissionAndRelease(t *testing.T) {
	rule := SpellRule{ID: 1, School: 1, ManaCost: 90, MaxRange: 1, TargetsUnit: true, Damaging: true, DamageMin: 2, DamageMax: 2}
	a := sourceMage1101(1, 1, BookSpell{Range: 3, ManaCost: 7})
	b := sourceMage1101(2, 1, BookSpell{Range: 2, ManaCost: 17})
	b.Y = 3
	victim := spEnt(3, 5, 2)
	w := spWorld(t, 1101, []SpellRule{rule}, a, b, victim)
	if got := SpellCharacteristicsFor(Rules{}, a, rule); got.Range != 3 || got.ManaCost != 7 {
		t.Fatalf("instance popup %+v", got)
	}
	if reason := w.BookSpellRefusal(2, 3, 1); reason != "target out of range" {
		t.Fatalf("second actor range: %q", reason)
	}
	if events := spRunCast(w, spCast(1, 3, 1)); len(events) != 1 {
		t.Fatalf("saved range cast: %v", events)
	}
	if got := spAt(t, w, 1); got.Mana != 93 || got.Book.Slots[0] != (BookSpell{Range: 2, ManaCost: 7}) {
		t.Fatalf("cost or application refresh %+v", got.Book)
	}
	if spAt(t, w, 2).Book != b.Book || w.Spells()[0] != rule || spAt(t, w, 3).HP != 96 {
		t.Fatal("cast changed another instance/table or scratch damage")
	}
}

func TestSourceBookTeachingPresenceAndIdempotence(t *testing.T) {
	rule := SpellRule{ID: 26, ManaCost: 50, MaxRange: 5, Defensive: true}
	for _, state := range []BookState{BookLegacy, BookAbsent, BookPresent} {
		e := Entity{Book: Spellbook{State: state}}
		LearnBookSpell(&e, 26, []SpellRule{rule})
		if state == BookAbsent {
			if e.KnownSpells != 0 || e.Book.State != BookAbsent {
				t.Fatal("teaching allocated absent book")
			}
			continue
		}
		if e.KnownSpells != 1<<26 {
			t.Fatal("existing book not taught")
		}
		if state == BookPresent {
			if e.Book.Slots[25] != (BookSpell{5, 1, 50}) {
				t.Fatal("constructor parameters")
			}
			e.Book.Slots[25] = BookSpell{19, 2, 65535}
			LearnBookSpell(&e, 26, []SpellRule{rule})
			if e.Book.Slots[25] != (BookSpell{19, 2, 65535}) {
				t.Fatal("repeat teaching replaced instance")
			}
		}
	}
}

func TestSourceBookActualReadAndEquipmentTeaching(t *testing.T) {
	rule := SpellRule{ID: 26, ManaCost: 50, MaxRange: 5, Defensive: true}
	for _, state := range []BookState{BookAbsent, BookPresent} {
		e := spMage(7, 2, 2, 60, 100, 100, 0)
		e.Book.State = state
		w := spWorld(t, 1101, []SpellRule{rule}, e)
		w.carried[0] = []ItemStack{StackItem(testSpellBook(26), 2)}
		Step(w, []Command{{Kind: KindReadBook, Entity: 7, X: 0}})
		if len(w.carried[0]) != 1 || w.carried[0][0].Count != 1 {
			t.Fatal("not one book consumed")
		}
		if state == BookAbsent {
			if w.entities[0].KnownSpells != 0 {
				t.Fatal("read allocated first book")
			}
		} else {
			if w.entities[0].Book.Slots[25] != (BookSpell{5, 1, 50}) {
				t.Fatal("new Spell constructor missing")
			}
			w.entities[0].Book.Slots[25] = BookSpell{19, 2, 65535}
		}
		before := w.entities[0].Book
		Step(w, []Command{{Kind: KindReadBook, Entity: 7, X: 0}})
		applyEquipmentItemState(&w.entities[0], ItemInstance{}, testSpellBook(26), w.spells)
		if len(w.carried[0]) != 0 || w.entities[0].Book != before {
			t.Fatal("duplicate read/equip replaced instance")
		}
	}
}

func TestSourceBookRefreshZeroOrdinaryTeleportPreservesStoredFields(t *testing.T) {
	e := sourceMage1101(1, 1, BookSpell{199, 2, 65535})
	e.KnownSpells |= 1<<20 | 1<<26
	e.Book.Slots[19], e.Book.Slots[25] = BookSpell{200, 3, 32768}, BookSpell{201, 255, 32767}
	rules := []SpellRule{{ID: 1, School: 1, MaxRange: 5}, {ID: 20}, {ID: 26, MaxRange: 5}}
	RefreshBook(Rules{}, &e, rules)
	if e.Book.Slots[0] != (BookSpell{6, 2, 65535}) || e.Book.Slots[19] != (BookSpell{0, 3, 32768}) || e.Book.Slots[25] != (BookSpell{15, 255, 32767}) {
		t.Fatalf("refresh %+v", e.Book)
	}
}

func TestSourceBookSignedCostAndRawDefensiveConsumers(t *testing.T) {
	for _, tc := range []struct {
		cost        uint16
		mana, after int32
		admitted    bool
	}{
		{32767, 32766, 32766, false}, {32767, 32767, 0, true},
		{32768, 0, -32768, true}, {65535, 0, 1, true}, {1, 65535, 65535, false},
	} {
		e := sourceMage1101(1, 1, BookSpell{5, 2, tc.cost})
		e.Mana = tc.mana
		rule, _ := BookRuleFor(e, SpellRule{ID: 1})
		if bookAffords(e, rule) != tc.admitted {
			t.Fatalf("afford signed %#x at %d", tc.cost, tc.mana)
		}
		if tc.admitted {
			debitBook(&e, rule)
		}
		if e.Mana != tc.after {
			t.Fatalf("debit %#x: %d", tc.cost, e.Mana)
		}
	}
	for _, defensive := range []uint8{0, 1, 2, 255} {
		e := sourceMage1101(1, 1, BookSpell{5, defensive, 7})
		e.Owner, e.TypeID = 2, 0x17
		victim := spEnt(2, 3, 2)
		rule := SpellRule{ID: 1, MaxRange: 5, TargetsUnit: true, Damaging: true, DamageMin: 1, DamageMax: 1}
		w := spWorld(t, 1101, []SpellRule{rule}, e, victim)
		resolved, _ := BookRuleFor(e, rule)
		w.pointAttribution(0, 1, resolved)
		if w.entities[1].HasKillCredit != (defensive != 1) {
			t.Fatalf("point byte %d", defensive)
		}
		if resolved.Defensive != (defensive != 0) {
			t.Fatalf("AI byte %d", defensive)
		}
	}
}
