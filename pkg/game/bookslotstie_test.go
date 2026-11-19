package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
)

// The SAV format's slot count, the rules' book bound and the book's bounds
// are one count each, and Slot reaches every id inside them.
func TestBookBoundsTie(t *testing.T) {
	format, ruleMax, wire := sav.BookSlots, rules.SpellBookMax, sim.BookWireSlots
	ids, first := rules.SpellFormulaIDs, rules.SpellIDs
	if format != ruleMax || format != wire || sim.BookIDs != ids || first != wire {
		t.Fatalf("sav %d, rules %d, wire %d, ids %d/%d, first %d", format, ruleMax, wire, sim.BookIDs, ids, first)
	}
	var b sim.Spellbook
	for id := uint16(1); int(id) <= sim.BookIDs; id++ {
		if b.Slot(id) == nil {
			t.Fatalf("Slot(%d) is nil inside the book bound", id)
		}
	}
	if b.Slot(uint16(sim.BookIDs+1)) != nil || b.Slot(0) != nil {
		t.Fatal("Slot accepts an id outside the book bound")
	}
}
