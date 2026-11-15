package mapload

import (
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
)

// Both games' books take spells 1 to 28; a cell or mask bit 29 is dropped.
func TestBookSpellCountWalksEveryIDOfBothGames(t *testing.T) {
	if got := SpellGameOf(nil); got != rules.SpellGameFirst {
		t.Fatalf("no table game %v, want first", got)
	}
	var rows []sim.SpellRule
	for id := 1; id <= 31; id++ {
		rows = append(rows, sim.SpellRule{ID: uint16(id)})
	}
	for _, spells := range []string{"", "rom1", "rom2"} {
		tab := &Table{builtSpellRules: rows}
		if spells != "" {
			tab.Edition = &base.Edition{Spells: spells}
		}
		if got := sim.BookSpellCount(SpellGameOf(tab)); got != 28 {
			t.Fatalf("%q book count %d, want 28", spells, got)
		}
		for id := int32(0); id <= 31; id++ {
			want := uint32(0)
			if id >= 1 && id <= 28 {
				want = uint32(1) << uint(id)
			}
			params := make([]int32, 64)
			params[48] = id
			if known, _, _ := unitSpellbook(params, tab, 0x46); known != want {
				t.Fatalf("%q cell read id %d: known %#x, want %#x", spells, id, known, want)
			}
			b := &CreatureBooks{KnownCell: 40}
			params = make([]int32, 64)
			params[40] = int32(uint32(1) << uint(id))
			var known uint32
			var book sim.Spellbook
			var skill [6]int32
			b.add(0x46, 1, params, tab, &known, &book, &skill)
			if known != want {
				t.Fatalf("%q mask insert id %d: known %#x, want %#x", spells, id, known, want)
			}
		}
	}
}
