package secondshop

import (
	"slices"
	"testing"
)

func TestShownOrderUsesKindThenPriceInsteadOfServerRow(t *testing.T) {
	tb := sortTables()
	items := []Item{{Table: Armor, Row: 1, Price: 300, Count: 1},
		{Table: Shield, Row: 1, Price: 50, Count: 2},
		{Table: Armor, Row: 3, Price: 200, Count: 3}}
	tb.Sort(items)
	var got []int32
	for _, it := range items {
		got = append(got, it.Count)
	}
	if !slices.Equal(got, []int32{2, 3, 1}) {
		t.Fatalf("shown order %v, want [2 3 1]", got)
	}
}

func TestShownOrderPermutesTwoEqualEquipmentItems(t *testing.T) {
	tb := sortTables()
	items := []Item{{Table: Shield, Row: 1, Price: 50, Count: 1}, {Table: Shield, Row: 1, Price: 50, Count: 2}}
	tb.Sort(items)
	if items[0].Count != 2 || items[1].Count != 1 {
		t.Fatalf("equal order %v, want [2 1]", items)
	}
}
