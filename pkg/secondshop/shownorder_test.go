package secondshop

import (
	"slices"
	"testing"

	"againrom/pkg/formats/itempkt"
	"againrom/pkg/sim"
)

func TestShownOrderSeparatesClientKeys(t *testing.T) {
	tb := sortTables()
	tb.Weapons = fixtureRows{n: 23, params: map[int][]int32{1: equipmentRow(1, 1, 1), 18: equipmentRow(1, 1, 1)}}
	items := []Item{
		{Table: Armor, Row: 4, Material: 1, Price: 1, Count: 1},
		{Table: Weapon, Row: 18, Material: 1, Price: 1, Count: 2},
		{Table: Shield, Row: 1, Material: 1, Price: 1, Count: 3},
		{Table: Weapon, Row: 1, Material: 1, Price: 1, Count: 4},
		{Table: Weapon, Row: 1, Material: 1, Class: 1, Price: 99, Count: 5},
	}
	tb.ItemRecords = map[uint16]itempkt.Record{}
	for _, it := range items {
		key, _ := tb.NameKey(it)
		tb.ItemRecords[key] = itempkt.Record{Flags: 2, Price: 9, HasPrice: true}
	}
	tb.ItemRecords[0x1604] = itempkt.Record{Flags: 0, Price: 1, HasPrice: true}
	tb.ItemRecords[0x1101] = itempkt.Record{Flags: 2, Price: 10, HasPrice: true}
	tb.Sort(items)
	var got []int32
	for _, it := range items {
		got = append(got, it.Count)
	}
	if !slices.Equal(got, []int32{5, 4, 2, 3, 1}) {
		t.Fatalf("shown order %v", got)
	}
}

func TestShownOrderEqualEquipmentUsesCRT(t *testing.T) {
	tb := sortTables()
	for _, n := range []int{2, 8, 9, 10, 12, 14, 17} {
		list := make([]Item, n)
		for i := range list {
			list[i] = Item{Table: Shield, Row: 1, Material: 1, Price: 50, Count: int32(i)}
		}
		tb.Sort(list)
		want := make([]int32, n)
		for i := range want {
			want[i] = int32(i)
		}
		if n <= 8 {
			want = append(want[1:], want[0])
		} else {
			want[0], want[n/2] = want[n/2], want[0]
		}
		for i := range list {
			if list[i].Count != want[i] {
				t.Fatalf("n %d: position %d got %d want %d", n, i, list[i].Count, want[i])
			}
		}
	}
}

func TestKnownServerOrderFeedsClientPermutation(t *testing.T) {
	tb := sortTables()
	items := []Item{{Table: Shield, Row: 1, Price: 50, Count: 1},
		{Table: Shield, Row: 1, Price: 50, Count: 2}, {Table: Shield, Row: 1, Price: 50, Count: 3}}
	p := tb.Profile(Record{}, func() int { return 0 })
	p.Categories[0].Finish(items, nil)
	if items[0].Count != 2 {
		t.Fatal("the profile skipped the server permutation")
	}
	tb.Sort(items)
	if items[0].Count != 3 || items[1].Count != 1 || items[2].Count != 2 {
		t.Fatalf("combined permutations %v", items)
	}
}

func TestShownOrderUnknownMagicTiesAreStable(t *testing.T) {
	tb := sortTables()
	list := []Item{{Table: Magic, Row: 69, Price: 100, Count: 1}, {Table: Magic, Row: 72, Price: 100, Count: 2}, {Table: Magic, Row: 6, Price: 100, Count: 3}}
	tb.Sort(list)
	for i := range list {
		if list[i].Count != int32(i+1) {
			t.Fatalf("unknown tie order %v", list)
		}
	}
}

func TestShownPriceKeepsTheServerPriceWithEffects(t *testing.T) {
	tb := sortTables()
	tb.ItemRecords = map[uint16]itempkt.Record{0x1201: {Flags: 2, Price: 51, HasPrice: true}}
	it := Item{Table: Shield, Row: 1, Material: 1, Price: 50}
	if got := tb.ShownPrice(it); got != 51 {
		t.Fatalf("plain price %d", got)
	}
	it.Effects = []sim.ItemEffect{{Kind: 5, Operand: 1}}
	if got := tb.ShownPrice(it); got != 50 {
		t.Fatalf("effect price %d", got)
	}
}

func TestShownBookFlagComesFromItsTemplateRecord(t *testing.T) {
	tb := sortTables()
	tb.TemplateCode = func(uint16) uint16 { return 0x0e03 }
	tb.ItemRecords = map[uint16]itempkt.Record{0x0e03: {Flags: 2}}
	if !tb.ShownFlag(Item{Table: Book, Spell: 1}) {
		t.Fatal("ignored the template flag")
	}
}

func TestShownMagicPriceUsesTheLoadedMagicValue(t *testing.T) {
	tb := sortTables()
	tb.MagicItems = fixtureRows{n: 70, params: map[int][]int32{69: {100}}}
	tb.ItemRecords = map[uint16]itempkt.Record{0x0e45: {Flags: 7, Price: 777, HasPrice: true}}
	if got := tb.ShownPrice(Item{Table: Magic, Row: 69, Price: 100}); got != 100 {
		t.Fatalf("magic price %d", got)
	}
}
