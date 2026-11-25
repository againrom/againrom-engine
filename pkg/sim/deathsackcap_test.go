package sim

import "testing"

// A unit carrying one item and a purse of 7 dies on a cell whose sack holds
// entries distinct items and 5 gold, under a build whose death merge refuses
// a sack of limit entries or more.
func deathOnFullSack(t *testing.T, entries, limit int) Sack {
	t.Helper()
	w := mustStockedWorld(t, 1,
		[]Entity{{ID: 1, X: 4, Y: 6, HP: 1, MaxHP: 10, Purse: 7}},
		[]Stock{{ID: 1, Items: []uint16{0x101}}})
	codes := make([]uint16, entries)
	for k := range codes {
		codes[k] = uint16(0x1000 + k)
	}
	if err := w.ReplaceGroundSacks([]Sack{makeSack(4, 6, 5, plainItems(codes))}); err != nil {
		t.Fatal(err)
	}
	w.SetBuildRules(BuildRules{DeathSackCap: limit})
	Step(w, []Command{TerminalKill(1)})
	sacks := w.Sacks()
	if len(sacks) != 1 {
		t.Fatalf("%d sacks, want 1", len(sacks))
	}
	if carried, _ := w.Carried(1); len(carried) != 0 {
		t.Fatalf("body still carries %v", carried)
	}
	return sacks[0]
}

// The Russian merge returns at once when the cell's sack holds 1024 entries
// or more: the bag's items are lost and the gold is still added. Below the
// bound, and in a build with none, the bag joins the sack (R2-ENGINE-1828).
func TestDeathBagMergesNothingIntoASackOfTheBuildsBound(t *testing.T) {
	for _, c := range []struct {
		name           string
		entries, limit int
		items          int
	}{
		{"full sack", 1024, 1024, 1024},
		{"one below the bound", 1023, 1024, 1024},
		{"no bound", 1024, 0, 1025},
	} {
		s := deathOnFullSack(t, c.entries, c.limit)
		if len(s.Items) != c.items || s.Gold != 12 {
			t.Errorf("%s: sack of %d items and %d gold, want %d and 12", c.name, len(s.Items), s.Gold, c.items)
		}
		if lost := s.Items[len(s.Items)-1] != 0x101; lost != (c.items == c.entries) {
			t.Errorf("%s: last item %#x", c.name, s.Items[len(s.Items)-1])
		}
	}
}
