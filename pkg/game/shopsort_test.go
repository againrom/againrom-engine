package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// sortRow is one row of a synthetic collection carrying the three cells the
// shelf order reads: the wear-rule column, an armour's Slot and a weapon's
// AttackType.
//
// THE ATTACK-TYPE COLUMN IS SPELLED HERE AS A NUMBER because pkg/data does not
// export it, and sortTable below asserts that data.AttackTypeFromCode reads
// back what each row meant. A column that moved would fail that assertion
// rather than quietly filing every weapon under one kind.
type sortRow struct {
	name string
	slot int32
	atk  int32
	suit int32
}

const sortAttackTypeColumn = 5

type sortCollection []sortRow

func (c sortCollection) Len() int                    { return len(c) }
func (c sortCollection) EntryName(i int) string      { return c[i].name }
func (c sortCollection) EntryStrings(i int) []string { return nil }

func (c sortCollection) EntryParams(i int) []int32 {
	if c[i].name == "" {
		return nil
	}
	p := make([]int32, data.SutableForColumn+1)
	for j := range p {
		p[j] = -1
	}
	p[4] = c[i].slot
	p[sortAttackTypeColumn] = c[i].atk
	p[data.SutableForColumn] = c[i].suit
	return p
}

// The two sutableFor cells the fixture uses, and the one an item usable by
// both classes carries.
const (
	sortFighter = 1
	sortMage    = 2
	sortBoth    = 3
)

// sortTable is a table holding one row per kind the order names.
func sortTable(t *testing.T) *mapload.Table {
	t.Helper()
	tbl := &mapload.Table{
		Weapons: sortCollection{
			{},
			{name: "sword", atk: data.SkillBlade, suit: sortFighter},
			{name: "axe", atk: data.SkillAxe, suit: sortFighter},
			{name: "mace", atk: data.SkillBludgen, suit: sortFighter},
			{name: "spear", atk: data.SkillPike, suit: sortFighter},
			{name: "bow", atk: data.SkillShoot, suit: sortFighter},
			{name: "thrower", atk: 11, suit: sortFighter},
			{name: "staff", atk: data.SkillBludgen, suit: sortMage},
		},
		Shields: sortCollection{
			{},
			{name: "targe", slot: 2, suit: sortFighter},
		},
		Armors: sortCollection{
			{},
			{name: "helm", slot: 6, suit: sortFighter},
			{name: "mail", slot: 7, suit: sortFighter},
			{name: "cuirass", slot: 8, suit: sortFighter},
			{name: "bracers", slot: 9, suit: sortFighter},
			{name: "gauntlets", slot: 10, suit: sortFighter},
			{name: "boots", slot: 12, suit: sortFighter},
			{name: "hat", slot: 6, suit: sortMage},
			{name: "ring", slot: 4, suit: sortBoth},
			{name: "amulet", slot: 5, suit: sortBoth},
		},
	}
	weapons := tbl.Weapons.(sortCollection)
	for row := 1; row < len(weapons); row++ {
		code := data.ComposeItemCode(0, 1, 0, row)
		got, ok := data.AttackTypeFromCode(code, tbl.Weapons)
		if !ok || got != weapons[row].atk {
			t.Fatalf("fixture row %d (%s): AttackTypeFromCode = %d, %v, want %d",
				row, weapons[row].name, got, ok, weapons[row].atk)
		}
	}
	return tbl
}

// weaponCode and shieldCode compose the code a shop candidate of each
// collection carries: field B is the class and field D is the row.
func weaponCode(row int) data.ItemCode { return data.ComposeItemCode(0, 1, 0, row) }
func shieldCode(row int) data.ItemCode { return data.ComposeItemCode(0, 2, 0, row) }

// armourCode is the same for an armour, whose field B is the row's own Slot
// (data.ArmorShopClass).
func armourCode(t *testing.T, tbl *mapload.Table, row int) data.ItemCode {
	t.Helper()
	slot := tbl.Armors.(sortCollection)[row].slot
	return data.ComposeItemCode(0, int(slot), 0, row)
}

// A shelf lists its stock in groups and runs each group cheapest first (owner,
// `DIV-319`): fighter items, then mage items, then the rest; inside a class by
// kind; inside a kind by price. A dearer helm stands before a cheaper mail, and
// a mage hat and a ring cheaper than every fighter item stand after all of them,
// so a sort on price first, or on class and kind with the dearest first, fails.
func TestShelfRunsCheapestFirstInsideEachGroup(t *testing.T) {
	tbl := sortTable(t)

	// want is the answer, labelled so a failure names the element that moved.
	type named struct {
		label string
		item  ShopItem
	}
	want := []named{
		{"helm 100", ShopItem{Code: armourCode(t, tbl, 1), Price: 100, Count: 1}},
		{"helm 200", ShopItem{Code: armourCode(t, tbl, 1), Price: 200, Count: 1}},
		{"mail 90", ShopItem{Code: armourCode(t, tbl, 2), Price: 90, Count: 1}},
		{"cuirass 80", ShopItem{Code: armourCode(t, tbl, 3), Price: 80, Count: 1}},
		{"bracers 70", ShopItem{Code: armourCode(t, tbl, 4), Price: 70, Count: 1}},
		{"gauntlets 60", ShopItem{Code: armourCode(t, tbl, 5), Price: 60, Count: 1}},
		{"boots 50", ShopItem{Code: armourCode(t, tbl, 6), Price: 50, Count: 1}},
		{"shield 40", ShopItem{Code: shieldCode(1), Price: 40, Count: 1}},
		{"sword 39", ShopItem{Code: weaponCode(1), Price: 39, Count: 1}},
		{"axe 38", ShopItem{Code: weaponCode(2), Price: 38, Count: 1}},
		{"mace 37", ShopItem{Code: weaponCode(3), Price: 37, Count: 1}},
		{"spear 36", ShopItem{Code: weaponCode(4), Price: 36, Count: 1}},
		{"bow 35", ShopItem{Code: weaponCode(5), Price: 35, Count: 1}},
		{"thrower 34", ShopItem{Code: weaponCode(6), Price: 34, Count: 1}},
		{"mage hat 30", ShopItem{Code: armourCode(t, tbl, 7), Price: 30, Count: 1}},
		{"mage hat 900", ShopItem{Code: armourCode(t, tbl, 7), Price: 900, Count: 1}},
		{"mage staff 800", ShopItem{Code: weaponCode(7), Price: 800, Count: 1}},
		{"ring 20", ShopItem{Code: armourCode(t, tbl, 8), Price: 20, Count: 1}},
		{"ring 700", ShopItem{Code: armourCode(t, tbl, 8), Price: 700, Count: 1}},
		{"amulet 10", ShopItem{Code: armourCode(t, tbl, 9), Price: 10, Count: 1}},
	}

	// The input is the reverse of the answer, so a sort that did nothing fails
	// on the first element and a sort that left a group's prices as they came
	// fails inside it.
	items := make([]ShopItem, 0, len(want))
	for i := len(want) - 1; i >= 0; i-- {
		items = append(items, want[i].item)
	}

	shopSortShelf(items, tbl)

	label := func(item ShopItem) string {
		for _, w := range want {
			if w.item.Code == item.Code && w.item.Price == item.Price {
				return w.label
			}
		}
		return "an element the fixture never made"
	}
	for i := range want {
		if got := label(items[i]); got != want[i].label {
			t.Errorf("position %d holds %q, want %q", i, got, want[i].label)
		}
	}
}

// Elements equal on class, kind and price keep the order they already had,
// which is what makes a shelf reproducible from its seed.
func TestShelfSortIsStableOnEqualPrices(t *testing.T) {
	tbl := sortTable(t)
	items := []ShopItem{
		{Code: armourCode(t, tbl, 1), Price: 10, Count: 1},
		{Code: armourCode(t, tbl, 1), Price: 10, Count: 2},
		{Code: armourCode(t, tbl, 1), Price: 10, Count: 3},
	}
	shopSortShelf(items, tbl)
	for i, item := range items {
		if item.Count != int32(i+1) {
			t.Fatalf("position %d holds count %d, want %d: equal prices were reordered",
				i, item.Count, i+1)
		}
	}
}

// A shop with no item table leaves a shelf exactly as it is. The dearer
// element stands first, so a price sort would move it.
func TestShelfSortWithNoTableChangesNothing(t *testing.T) {
	items := []ShopItem{
		{Code: 0x0112, Price: 99, Count: 1},
		{Code: 0x0611, Price: 1, Count: 1},
	}
	before := append([]ShopItem(nil), items...)
	shopSortShelf(items, nil)
	for i := range items {
		if !shopItemStateEqual(items[i], before[i]) {
			t.Fatalf("position %d moved with no table: %+v, want %+v", i, items[i], before[i])
		}
	}
}

// An item returned to a shelf lands in its group rather than at the end: fighter
// items, then mage items, then the rest; inside a class by kind; inside a kind
// cheapest first and after an element it ties with on all three. The ring is
// the cheapest element and stands last, in the last class.
//
// THE SYNTHETIC POOL ADMITS NOTHING — sortCollection carries no material mask
// block — so Generate leaves three empty shelves and this measures
// returnToShelf alone. What Generate itself produces is measured against the
// install by cmd/shopdump, which no test may read (golden rule 2).
func TestReturnedItemLandsInOrder(t *testing.T) {
	tbl := sortTable(t)
	s := NewShop(1000)
	s.Generate(tbl, 3)
	if got := len(s.Shelf(ShelfArmour)); got != 0 {
		t.Fatalf("the fixture pool drew %d armour elements, want 0", got)
	}

	cheapHelm := ShopItem{Code: armourCode(t, tbl, 1), Price: 5, Count: 1}
	helm := ShopItem{Code: armourCode(t, tbl, 1), Price: 500, Count: 1}
	enchantedHelm := ShopItem{Code: armourCode(t, tbl, 1), Price: 500, Count: 1,
		Effects: []sim.ItemEffect{{Kind: 3, Operand: 4}}}
	cheapBoots := ShopItem{Code: armourCode(t, tbl, 6), Price: 5, Count: 1}
	hat := ShopItem{Code: armourCode(t, tbl, 7), Price: 500, Count: 1}
	dearHat := ShopItem{Code: armourCode(t, tbl, 7), Price: 900, Count: 1}
	ring := ShopItem{Code: armourCode(t, tbl, 8), Price: 1, Count: 1}

	for _, item := range []ShopItem{cheapHelm, dearHat, ring, hat, helm, cheapBoots, enchantedHelm} {
		s.returnToShelf(ShelfArmour, item)
	}

	got := s.Shelf(ShelfArmour)
	want := []ShopItem{cheapHelm, helm, enchantedHelm, cheapBoots, hat, dearHat, ring}
	if len(got) != len(want) {
		t.Fatalf("shelf holds %d, want %d", len(got), len(want))
	}
	for i := range want {
		if !shopItemStateEqual(got[i], want[i]) {
			t.Errorf("position %d holds code %#x at %d with effects %v, want code %#x at %d with effects %v",
				i, uint16(got[i].Code), got[i].Price, got[i].Effects,
				uint16(want[i].Code), want[i].Price, want[i].Effects)
		}
	}
}
