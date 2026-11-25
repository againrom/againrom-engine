package game

import (
	"testing"

	"againrom/pkg/formats/alm"
)

// A type-9 record with the A&8 flag reads as a unit's own spell, a trap
// record does not.
func TestEnchantmentViewMarksUnitBookRecords(t *testing.T) {
	x := &MapInspector{}
	book := x.inspectionEnchantment(0, alm.Enchantment{A: 8, X: 3, Y: 4, SpellRaw: 4}, nil, 0, castView{})
	trap := x.inspectionEnchantment(1, alm.Enchantment{A: 0, X: 3, Y: 4, SpellRaw: 4, Elements: make([]alm.EnchantmentElement, 2)}, nil, 0, castView{})
	if !book.UnitBook || trap.UnitBook {
		t.Fatalf("book record UnitBook %v, trap record %v", book.UnitBook, trap.UnitBook)
	}
}
