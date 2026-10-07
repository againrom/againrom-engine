package sim

import "testing"

func TestCurrentBookModeFollowsOrdinaryValues(t *testing.T) {
	book := Spellbook{State: BookPresent}
	book.Slots[0] = BookSpell{Range: 19, Defensive: 2, ManaCost: 12345}
	anchor := BookValueAnchor(book, 2)
	policy := (Entity{KnownSpells: 2}).Values()
	policy.LegacyBookWire = &anchor
	for _, tc := range []struct {
		name string
		edit func(*Entity)
	}{
		{"unchanged", func(e *Entity) {}},
		{"range", func(e *Entity) { e.Book.Slots[0].Range = 73 }},
		{"defensive", func(e *Entity) { e.Book.Slots[0].Defensive = 0x83 }},
		{"cost", func(e *Entity) { e.Book.Slots[0].ManaCost = 54321 }},
		{"membership", func(e *Entity) { e.Book.Slots[2], e.Book.Slots[0], e.KnownSpells = e.Book.Slots[0], BookSpell{}, 8 }},
		{"empty present", func(e *Entity) { e.Book, e.KnownSpells = Spellbook{State: BookPresent}, 0 }},
		{"absent", func(e *Entity) { e.Book, e.KnownSpells = Spellbook{State: BookAbsent}, 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := Entity{Book: book, KnownSpells: 2}
			tc.edit(&e)
			wantBook, wantKnown := e.Book, e.KnownSpells
			if tc.name == "unchanged" {
				wantBook = Spellbook{}
			}
			if err := e.restoreValues(policy); err != nil || e.Book != wantBook || e.KnownSpells != wantKnown {
				t.Fatal("legacy mode erased ordinary values", err, e.Book, e.KnownSpells)
			}
		})
	}
	e := Entity{Book: book, KnownSpells: 2}
	policy.LegacyBookWire = nil
	if err := e.restoreValues(policy); err != nil || e.Book != book {
		t.Fatal("unanchored old metadata erased ordinary values", err, e.Book)
	}
	for _, mode := range []BookState{BookLegacy, BookPresent, BookState(9)} {
		invalid := [32]byte{}
		if _, _, err := RestoreBookMode(book, 2, mode, &invalid, 0); err == nil {
			t.Fatal("invalid anchor policy accepted", mode)
		}
	}
}

func TestCurrentBookModeRetainsNativeMembershipAndConsumers(t *testing.T) {
	extra := uint32(1 | 1<<29 | 1<<31)
	ordinary := Spellbook{State: BookPresent}
	ordinary.Slots[0] = BookSpell{Range: 19, Defensive: 2, ManaCost: 12345}
	anchor := BookValueAnchor(ordinary, 2)
	for _, absent := range []bool{false, true} {
		book, known := ordinary, uint32(2)
		book.Slots[0] = BookSpell{Range: 73, Defensive: 0x83, ManaCost: 65533}
		wantMode := BookNativePresent
		if absent {
			book, known, wantMode = Spellbook{State: BookAbsent}, 0, BookNativeAbsent
		}
		got, learned, err := RestoreBookMode(book, known, BookLegacy, &anchor, extra)
		if err != nil || got.State != wantMode || got.Slots != book.Slots || learned != known|extra {
			t.Fatal("ordinary values and native membership cannot coexist", absent, err, got, learned)
		}
		e := Entity{ID: 7, X: 1, Y: 1, HP: 10, MaxHP: 10, Mana: 20, MaxMana: 20, Book: got, KnownSpells: learned}
		low, ok := BookRuleFor(e, SpellRule{ID: 1, MaxRange: 3, ManaCost: 99})
		if ok == absent || !absent && (low.MaxRange != 73 || low.ManaCost != -3 || !low.Defensive || low.bookDefensive != 0x83 || !low.bookInstance) {
			t.Fatal("ordinary instance admission changed", absent, low, ok)
		}
		high, ok := BookRuleFor(e, SpellRule{ID: 31, MaxRange: 17, ManaCost: 9})
		if !ok || high.MaxRange != 17 || high.ManaCost != 9 || high.bookInstance {
			t.Fatal("native-only spell lost its table-backed rule", high, ok)
		}
		if !absent {
			debitBook(&e, low)
			if e.Mana != 23 {
				t.Fatal("mixed book did not consume signed ordinary cost", e.Mana)
			}
		}
		w, err := NewWorld(3, Bounds{Width: 4, Height: 4}, ModeCanonical, nil, []Entity{e})
		if err != nil {
			t.Fatal(err)
		}
		before := w.Hash()
		for cycle := 0; cycle < 2; cycle++ {
			raw, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var cold World
			if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != before {
				t.Fatal("native book mode changed binary values", cycle, err)
			}
			w = &cold
		}
		if err := w.ImportOriginalActorSpellbooks([]OriginalActorSpellbook{{ID: 7, Book: got, KnownSpells: learned}}); err == nil || w.Hash() != before {
			t.Fatal("original-only admission accepted native-only mode", err)
		}
		LearnBookSpell(&e, 3, []SpellRule{{ID: 3, MaxRange: 5, ManaCost: 7}})
		if (e.KnownSpells&8 != 0) == absent || e.KnownSpells&extra != extra {
			t.Fatal("native membership changed ordinary teach presence", e.Book, e.KnownSpells)
		}
	}
}

func TestCurrentBookMalformedSecondActorDoesNotAdoptFirstActor(t *testing.T) {
	book := Spellbook{State: BookPresent}
	book.Slots[0] = BookSpell{Range: 17, ManaCost: 91}
	w, err := NewWorld(7, Bounds{Width: 8, Height: 8}, ModeCanonical, nil, []Entity{
		{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 10, KnownSpells: 2, Book: book},
		{ID: 2, X: 2, Y: 2, HP: 10, MaxHP: 10, KnownSpells: 2, Book: book},
	})
	if err != nil {
		t.Fatal(err)
	}
	before := w.Hash()
	for _, fault := range []string{"zero anchor", "anchor without legacy", "extra ordinary bit"} {
		values := map[EntityID]ActorValues{}
		for _, e := range w.entities {
			v := e.Values()
			anchor := BookValueAnchor(e.Book, e.KnownSpells)
			v.LegacyBook, v.LegacyBookWire = true, &anchor
			values[e.ID] = v
		}
		bad := values[2]
		switch fault {
		case "zero anchor":
			bad.LegacyBookWire = new([32]byte)
		case "anchor without legacy":
			bad.LegacyBook = false
		case "extra ordinary bit":
			bad.ExtraSpells = 2
		}
		values[2] = bad
		if err := w.RestoreCurrentContinuation(nil, values, w.Actions(), nil); err == nil || w.Hash() != before {
			t.Fatal("malformed second book committed earlier actor values", fault, err)
		}
	}
}
