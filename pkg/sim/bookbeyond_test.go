package sim

import "testing"

func beyondWorld(t *testing.T, book Spellbook, known uint32) *World {
	t.Helper()
	mage := Entity{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10, Mana: 20, MaxMana: 20, Book: book, KnownSpells: known}
	w, err := NewSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, nil, []Entity{mage}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// A World whose books hold no spell past the entity record's slots writes the
// plain form, so every first-game World keeps its bytes and hash; a book that
// holds one writes the beyond-slot section and reads it back.
func TestBeyondSlotBookSectionIsWrittenOnlyForASpellPastTheSlots(t *testing.T) {
	plain := Spellbook{State: BookPresent}
	plain.Slots[1] = BookSpell{Range: 6, ManaCost: 4}
	w := beyondWorld(t, plain, 1<<2)
	raw, err := w.MarshalBinary()
	if err != nil || raw[0] == beyondBookFormVersion {
		t.Fatalf("a book of the first 28 spells wrote form %d, %v", raw[0], err)
	}
	held := plain
	*held.Slot(29) = BookSpell{Range: 5, Defensive: 0, ManaCost: 10}
	w = beyondWorld(t, held, 1<<2|1<<29)
	raw, err = w.MarshalBinary()
	if err != nil || raw[0] != beyondBookFormVersion {
		t.Fatalf("a book holding spell 29 wrote form %d, %v", raw[0], err)
	}
	var back World
	if err := back.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if back.entities[0].Book != held || back.Hash() != w.Hash() {
		t.Fatal("the section did not carry spell 29's slot")
	}
	section := len(raw) - 9 - beyondBookRecordLen
	for name, corrupt := range map[string]func([]byte){
		"an entity past the World": func(b []byte) { b[section+3] = 0x7f },
		"an empty slot":            func(b []byte) { clear(b[section+4 : section+beyondBookRecordLen]) },
	} {
		bad := append([]byte(nil), raw...)
		corrupt(bad)
		var broken World
		if broken.UnmarshalBinary(bad) == nil {
			t.Fatalf("a section with %s decoded", name)
		}
	}
}
