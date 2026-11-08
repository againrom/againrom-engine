package game

import (
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// TestSecondGamePickupAndGrantLinesNameTheItem: a carried book, magic item
// and equipment item of the second game are named by the composer, not by
// their code.
func TestSecondGamePickupAndGrantLinesNameTheItem(t *testing.T) {
	table := readUnder(base.GameROM2, secondSurfaceTable(true))
	words := ui.AuthoredWords()
	cases := []struct {
		what string
		code uint16
		name string
	}{
		{"book", 0x0e80 | 1, "Fire Arrow"},
		{"magic item", 0x0e01, "Potion"},
		{"equipment", 1<<12 | 3<<8 | 1<<5 | 0, "Cloak Steel Fine"},
	}
	for _, c := range cases {
		if card := itemInfoLines(data.ItemCode(c.code), table); len(card) == 0 || card[0] != c.name {
			t.Errorf("%s card = %q, want name %q first", c.what, card, c.name)
		}
		items := []namedStack{{stack: sim.PlainStack(c.code, 1), name: "Hero"}}
		wantPick := pickupItemLine(&words, c.name, 1, "Hero")
		if got := pickupLinesForItems(items, table, &words); len(got) != 1 || got[0] != wantPick {
			t.Errorf("%s pickup = %q, want %q", c.what, got, wantPick)
		}
		wantGrant := grantItemLine(&words, c.name, 1, "Hero")
		if got := grantLinesForItems(items, table, &words); len(got) != 1 || got[0] != wantGrant {
			t.Errorf("%s grant = %q, want %q", c.what, got, wantGrant)
		}
	}
}
