package sim

import "testing"

// Both games refresh book rows 1 to 28 only; spell 29 has no slot.
func TestRefreshAppliedBookWalksEveryIDOfBothGames(t *testing.T) {
	r, err := NewRules(RulesParams{SkillCap: 150})
	if err != nil {
		t.Fatal(err)
	}
	var spells []SpellRule
	for id := 1; id <= 31; id++ {
		spells = append(spells, SpellRule{ID: uint16(id), MaxRange: 5, ManaCost: 1})
	}
	for _, d := range []ScriptDialect{ScriptROM1, ScriptROM2} {
		game := newGameProfile(d)
		if got := BookSpellCount(game.spellGame); got != 28 {
			t.Fatalf("dialect %v book count %d, want 28", d, got)
		}
		for id := uint32(0); id <= 31; id++ {
			var slots [28]BookSpell
			for i := range slots {
				slots[i].Range = 255
			}
			var known uint32
			for i := uint32(1); i <= 31; i++ {
				known |= uint32(1) << i
			}
			e := Entity{Book: Spellbook{State: BookPresent, Slots: slots}, KnownSpells: known}
			w := &World{rules: r, spells: spells, entities: []Entity{e}, game: game}
			w.refreshAppliedBook(0, id)
			got := w.entities[0].Book.Slots
			for i := 1; i <= 28; i++ {
				refreshed := got[i-1].Range != 255
				want := id == uint32(i)
				if refreshed != want {
					t.Fatalf("dialect %v refresh id %d: row %d refreshed %v, want %v", d, id, i, refreshed, want)
				}
			}
		}
	}
}
