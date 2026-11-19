package sim

import "testing"

// The first game's book refreshes rows 1 to 28 and the second game's rows 1
// to 29; a row past the game's last spell has no slot.
func TestRefreshAppliedBookWalksEveryIDOfBothGames(t *testing.T) {
	r, err := NewRules(RulesParams{SkillCap: 150})
	if err != nil {
		t.Fatal(err)
	}
	var spells []SpellRule
	for id := 1; id <= 31; id++ {
		spells = append(spells, SpellRule{ID: uint16(id), MaxRange: 5, ManaCost: 1})
	}
	for _, c := range []struct {
		d   ScriptDialect
		top int
	}{{ScriptROM1, 28}, {ScriptROM2, 29}} {
		game := newGameProfile(c.d)
		if got := BookSpellCount(game.spellGame); got != c.top {
			t.Fatalf("dialect %v book count %d, want %d", c.d, got, c.top)
		}
		for id := uint32(0); id <= 31; id++ {
			book := Spellbook{State: BookPresent}
			var known uint32
			for i := uint16(1); i <= BookIDs; i++ {
				book.Slot(i).Range = 255
				known |= uint32(1) << i
			}
			e := Entity{Book: book, KnownSpells: known}
			w := &World{rules: r, spells: spells, entities: []Entity{e}, game: game}
			w.refreshAppliedBook(0, id)
			got := w.entities[0].Book
			for i := uint16(1); i <= BookIDs; i++ {
				refreshed := got.Slot(i).Range != 255
				want := id == uint32(i) && int(i) <= c.top
				if refreshed != want {
					t.Fatalf("dialect %v refresh id %d: row %d refreshed %v, want %v", c.d, id, i, refreshed, want)
				}
			}
		}
	}
}
