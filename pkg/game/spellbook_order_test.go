package game

import (
	"slices"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The map book follows the cell rows; the known-only list keeps table order and every known spell.
func TestBookRoutesOrder(t *testing.T) {
	rules := []sim.SpellRule{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}}
	rows := []uint16{3, 1, 2}
	e := sim.Entity{Mind: 30, Book: sim.Spellbook{State: sim.BookPresent}}
	for _, id := range []uint16{1, 2, 3, 4} {
		sim.LearnBookSpell(&e, id, rules)
	}
	ids := func(book []ui.SpellEntry) []uint32 {
		var out []uint32
		for _, s := range book {
			out = append(out, s.ID)
		}
		return out
	}
	selected, fixed := selectedSpellbook(testFirstGame, sim.Rules{}, []sim.Entity{e}, rules, nil, testSpellPopupWords(), nil, rows)
	if want := []uint32{3, 1, 2}; !fixed || !slices.Equal(ids(selected), want) {
		t.Fatalf("selected book order %v fixed %v, want %v fixed true", ids(selected), fixed, want)
	}
	known := spellbookOf(testFirstGame, sim.Rules{}, e, rules, nil, testSpellPopupWords(), nil, rows)
	if want := []uint32{1, 2, 3, 4}; !slices.Equal(ids(known), want) {
		t.Fatalf("known-only list %v, want %v", ids(known), want)
	}
}
