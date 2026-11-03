package game

import (
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/sim"
)

// itemUnitCell is one compared value: an item stack and, for a shop page
// lot, the city graph node it is bound to.
type itemUnitCell struct {
	stack sim.ItemStack
	city  sim.SavedObjectID
}

func (c itemUnitCell) lot() ShopItem {
	it := shopItemFromInstance(c.stack.Instance(), int32(c.stack.Count))
	it.CityID = c.city
	return it
}

// itemUnitBases are the synthetic items each crossing compares: a plain
// stackable item, a potion, an enchanted weapon holding its Spell, an item
// with a native record, and one bound to a saved object.
func itemUnitBases() []struct {
	name string
	cell itemUnitCell
} {
	plain := sim.ItemStack{Code: 0x1234, Kind: 1, Price: 100, Count: 2, Weight: 3, WeightPresent: true}
	potion := sim.ItemStack{Code: 0x0e05, Kind: 3, Price: 50, Count: 3, Effects: []sim.ItemEffect{{Kind: 7, Operand: 9}}}
	weapon := sim.ItemStack{Code: 0x2108, Kind: 2, Price: 400, Count: 1, Effects: []sim.ItemEffect{{Kind: 4, Operand: 2}, {Kind: 5, Mode: 1, Operand: 3}}}
	weapon.SourceEquipment.Class = sim.SourceWeapon
	weapon.SourceEquipment.Attack[3] = 6
	weapon.SourceEquipment.Spell = sim.SourceItemSpell{Present: true, ID: 4, Range: 5, ManaCost: 12}
	native := sim.ItemStack{Code: 0x3302, Kind: 1, Price: 70, Count: 1, NativeRecord: &sim.NativeItemRecord{Class: 1, F45: 2, Token: sim.SavedObjectToken{Identity: 11, T08: 3}}}
	object := sim.ItemStack{ObjectID: 9, Code: 0x1234, Kind: 1, Price: 100, Count: 1, Effects: []sim.ItemEffect{}}
	return []struct {
		name string
		cell itemUnitCell
	}{
		{"plain", itemUnitCell{plain, 0}}, {"potion", itemUnitCell{potion, 4}}, {"weapon", itemUnitCell{weapon, 0}},
		{"native", itemUnitCell{native, 0}}, {"object", itemUnitCell{object, 6}},
	}
}

// itemUnitVariants each change exactly one field of a cell. A variant that
// cannot apply to a base (a token on an item with no native record) leaves
// it equal.
var itemUnitVariants = []struct {
	name   string
	change func(*itemUnitCell)
}{
	{"same", func(*itemUnitCell) {}},
	{"object", func(c *itemUnitCell) { c.stack.ObjectID++ }},
	{"code", func(c *itemUnitCell) { c.stack.Code++ }},
	{"kind", func(c *itemUnitCell) { c.stack.Kind++ }},
	{"kind3", func(c *itemUnitCell) {
		if c.stack.Kind == 3 {
			c.stack.Kind = 1
		} else {
			c.stack.Kind = 3
		}
	}},
	{"price", func(c *itemUnitCell) { c.stack.Price += 5 }},
	{"count", func(c *itemUnitCell) { c.stack.Count++ }},
	{"weight", func(c *itemUnitCell) { c.stack.Weight++ }},
	{"weightpresent", func(c *itemUnitCell) { c.stack.WeightPresent = !c.stack.WeightPresent }},
	{"equipclass", func(c *itemUnitCell) { c.stack.SourceEquipment.Class ^= sim.SourceWeapon }},
	{"attack3", func(c *itemUnitCell) { c.stack.SourceEquipment.Attack[3]++ }},
	{"attack22", func(c *itemUnitCell) { c.stack.SourceEquipment.Attack[22]++ }},
	{"spell", func(c *itemUnitCell) { c.stack.SourceEquipment.Spell.ManaCost++ }},
	{"effects", func(c *itemUnitCell) {
		if len(c.stack.Effects) == 0 {
			c.stack.Effects = []sim.ItemEffect{{Kind: 4, Operand: 1}}
		} else {
			c.stack.Effects[0].Operand++
		}
	}},
	{"effectsnil", func(c *itemUnitCell) {
		switch {
		case c.stack.Effects == nil:
			c.stack.Effects = []sim.ItemEffect{}
		case len(c.stack.Effects) == 0:
			c.stack.Effects = nil
		}
	}},
	{"native", func(c *itemUnitCell) {
		if c.stack.NativeRecord == nil {
			c.stack.NativeRecord = &sim.NativeItemRecord{Class: 1}
		} else {
			c.stack.NativeRecord.F45++
		}
	}},
	{"nativetoken", func(c *itemUnitCell) {
		if c.stack.NativeRecord != nil {
			c.stack.NativeRecord.Token.Identity++
		}
	}},
	{"city", func(c *itemUnitCell) { c.city++ }},
}

// itemUnitCrossings are the item crossings with each one's "same unit"
// expression, in its call shape: a held value first, the value it is
// compared with second.
var itemUnitCrossings = []struct {
	name string
	same func(a, b itemUnitCell) bool
}{
	{"mission-deal", func(a, b itemUnitCell) bool {
		return sim.SameUnit(a.stack.Instance(), b.stack.Instance(), sim.UnitDeal)
	}},
	{"shop-lot", func(a, b itemUnitCell) bool { return shopItemEqual(a.lot(), b.lot()) }},
	{"shop-lot-state", func(a, b itemUnitCell) bool { return shopItemStateEqual(a.lot(), b.lot()) }},
	{"shop-selection", func(a, b itemUnitCell) bool {
		return shopItemMatchesSelection(a.stack.Instance(), b.stack.Instance())
	}},
	{"city-sale-group", func(a, b itemUnitCell) bool { return citySaleGroups(a.stack.Instance(), b.stack.Instance()) }},
	{"city-table-put", func(a, b itemUnitCell) bool {
		x, y := a.lot(), b.lot()
		return sim.SameUnit(x.Instance(), y.Instance(), sim.UnitPriced)
	}},
	{"city-shelf-join", func(a, b itemUnitCell) bool {
		return sim.SameUnit(a.stack.Instance(), b.stack.Instance(), sim.UnitItem)
	}},
	{"city-pack-join", func(a, b itemUnitCell) bool {
		return sim.SameUnit(a.stack.Instance(), b.stack.Instance(), sim.UnitMerge)
	}},
	{"city-bound-stock", func(a, b itemUnitCell) bool {
		return sim.SameUnit(a.stack, sim.StackItem(b.stack.Instance(), b.stack.Count), sim.UnitStack)
	}},
	{"city-worn-slot", func(a, b itemUnitCell) bool {
		return sim.SameUnit(a.stack, sim.StackItem(b.stack.Instance(), 1), sim.UnitStack)
	}},
	{"city-pack-item", func(a, b itemUnitCell) bool { return sim.SameUnit(a.stack, b.stack, sim.UnitStack) }},
	{"city-pack-expansion", func(a, b itemUnitCell) bool {
		return sim.SameUnit(a.stack.Instance(), b.stack.Instance(), sim.UnitStack)
	}},
	{"city-shared-item", func(a, b itemUnitCell) bool { return sim.SameUnit(a.stack, b.stack, sim.UnitStack) }},
	{"hero-pack-follow", func(a, b itemUnitCell) bool { return sim.SameUnit(a.stack, b.stack, sim.UnitStack) }},
	{"sav-current-unit", func(a, b itemUnitCell) bool {
		return sim.SameUnit(a.stack, sim.StackItem(b.stack.Instance(), a.stack.Count), sim.UnitStack)
	}},
	{"sav-item-owner", func(a, b itemUnitCell) bool { return sim.SameUnit(a.stack, b.stack, sim.UnitStack) }},
	{"sav-item-alias", func(a, b itemUnitCell) bool { return sim.SameUnit(a.stack, b.stack, sim.UnitStack) }},
	{"sav-native-identity", func(a, b itemUnitCell) bool {
		return sim.SameUnit(a.stack, sim.StackItem(b.stack.Instance(), b.stack.Count), sim.UnitStack)
	}},
	{"world-inventory", func(a, b itemUnitCell) bool {
		return itemStacksEqual([]sim.ItemStack{a.stack}, []sim.ItemStack{b.stack})
	}},
	{"worn-change", func(a, b itemUnitCell) bool {
		return sim.SameUnit(a.stack.Instance(), b.stack.Instance(), sim.UnitWorn)
	}},
	{"bag-kept-unit", func(a, b itemUnitCell) bool {
		back, in := a.stack.Instance(), b.stack.Instance()
		return sim.SameUnit(back, in, sim.UnitForm)
	}},
}

// itemUnitEmptiness are the null-position tests of a saved pack.
var itemUnitEmptiness = []struct {
	name  string
	empty func(sim.ItemStack) bool
}{
	{"current-null-position", func(v sim.ItemStack) bool { return v.Empty() }},
	{"sav-unbound-null", func(v sim.ItemStack) bool { return v.Empty() }},
	{"sav-null-pack-position", func(v sim.ItemStack) bool { return v.Empty() }},
}

func cloneItemUnitCell(c itemUnitCell) itemUnitCell {
	c.stack = c.stack.Clone()
	return c
}

func itemUnitRecord() string {
	var b strings.Builder
	names := make([]string, len(itemUnitVariants))
	for i, v := range itemUnitVariants {
		names[i] = v.name
	}
	fmt.Fprintf(&b, "variants %s\n", strings.Join(names, " "))
	for _, crossing := range itemUnitCrossings {
		for _, base := range itemUnitBases() {
			row := make([]byte, len(itemUnitVariants))
			for i, v := range itemUnitVariants {
				other := cloneItemUnitCell(base.cell)
				v.change(&other)
				row[i] = '.'
				if crossing.same(cloneItemUnitCell(base.cell), other) {
					row[i] = '='
				}
			}
			fmt.Fprintf(&b, "%s %s %s\n", crossing.name, base.name, row)
		}
	}
	for _, test := range itemUnitEmptiness {
		row := make([]byte, len(itemUnitVariants))
		for i, v := range itemUnitVariants {
			var c itemUnitCell
			v.change(&c)
			row[i] = '.'
			if test.empty(c.stack) {
				row[i] = '='
			}
		}
		fmt.Fprintf(&b, "%s empty %s\n", test.name, row)
	}
	return b.String()
}

// TestItemUnitRecord holds, for every item crossing and every one-field
// change of a synthetic item, whether the crossing calls the two the same
// unit ('=') or apart ('.'), and which one-field values each null-position
// test calls empty.
func TestItemUnitRecord(t *testing.T) {
	if got := itemUnitRecord(); got != itemUnitRecordWant {
		t.Fatalf("item unit record changed:\n%s", got)
	}
}

const itemUnitRecordWant = `variants same object code kind kind3 price count weight weightpresent equipclass attack3 attack22 spell effects effectsnil native nativetoken city
mission-deal plain ==...==....=..====
mission-deal potion ==...==....=..====
mission-deal weapon ==...==....=..====
mission-deal native ==...==....=..====
mission-deal object ==...==....=..====
shop-lot plain =..=..=....=..===.
shop-lot potion =.....=....=..===.
shop-lot weapon =.....=....=..===.
shop-lot native =..=..=....=..===.
shop-lot object =..=..=....=..===.
shop-lot-state plain =.............====
shop-lot-state potion =.............====
shop-lot-state weapon =.............====
shop-lot-state native =.............====
shop-lot-state object =.............====
shop-selection plain =..=..=....=..====
shop-selection potion =.....=....=..====
shop-selection weapon =.....=....=..====
shop-selection native =..=..=....=..====
shop-selection object =.================
city-sale-group plain ==.=..=....=..====
city-sale-group potion ==....=....=..====
city-sale-group weapon ==....=....=..====
city-sale-group native ==.=..=....=..====
city-sale-group object ==.=..=....=..====
city-table-put plain ==.=..=....=..====
city-table-put potion ==....=....=..====
city-table-put weapon ==....=....=..====
city-table-put native ==.=..=....=..====
city-table-put object ==.=..=....=..====
city-shelf-join plain ==.=.==....=..====
city-shelf-join potion ==....=....=..====
city-shelf-join weapon ==...==....=..====
city-shelf-join native ==.=.==....=..====
city-shelf-join object ==.=.==....=..====
city-pack-join plain ==....=....=..=.==
city-pack-join potion ==....=....=..=.==
city-pack-join weapon ==....=....=..=.==
city-pack-join native ==....=....=..=.==
city-pack-join object ==....=....=..=.==
city-bound-stock plain =.............=.==
city-bound-stock potion =.............=.==
city-bound-stock weapon =.............=.==
city-bound-stock native =.............=..=
city-bound-stock object =.............=.==
city-worn-slot plain ..................
city-worn-slot potion ..................
city-worn-slot weapon =.....=.......=.==
city-worn-slot native =.....=.......=..=
city-worn-slot object =.....=.......=.==
city-pack-item plain =.............=.==
city-pack-item potion =.............=.==
city-pack-item weapon =.............=.==
city-pack-item native =.............=..=
city-pack-item object =.............=.==
city-pack-expansion plain =.....=.......=.==
city-pack-expansion potion =.....=.......=.==
city-pack-expansion weapon =.....=.......=.==
city-pack-expansion native =.....=.......=..=
city-pack-expansion object =.....=.......=.==
city-shared-item plain =.............=.==
city-shared-item potion =.............=.==
city-shared-item weapon =.............=.==
city-shared-item native =.............=..=
city-shared-item object =.............=.==
hero-pack-follow plain =.............=.==
hero-pack-follow potion =.............=.==
hero-pack-follow weapon =.............=.==
hero-pack-follow native =.............=..=
hero-pack-follow object =.............=.==
sav-current-unit plain =.....=.......=.==
sav-current-unit potion =.....=.......=.==
sav-current-unit weapon =.....=.......=.==
sav-current-unit native =.....=.......=..=
sav-current-unit object =.....=.......=.==
sav-item-owner plain =.............=.==
sav-item-owner potion =.............=.==
sav-item-owner weapon =.............=.==
sav-item-owner native =.............=..=
sav-item-owner object =.............=.==
sav-item-alias plain =.............=.==
sav-item-alias potion =.............=.==
sav-item-alias weapon =.............=.==
sav-item-alias native =.............=..=
sav-item-alias object =.............=.==
sav-native-identity plain =.............=.==
sav-native-identity potion =.............=.==
sav-native-identity weapon =.............=.==
sav-native-identity native =.............=..=
sav-native-identity object =.............=.==
world-inventory plain =.............=.==
world-inventory potion =.............=.==
world-inventory weapon =.............=.==
world-inventory native =.............=..=
world-inventory object =.............=.==
worn-change plain ==.=.==....==.====
worn-change potion ==....=....==.====
worn-change weapon ==...==....==.====
worn-change native ==.=.==....==.====
worn-change object ==.=.==....==.====
bag-kept-unit plain ==....=======.====
bag-kept-unit potion ==....=======.====
bag-kept-unit weapon ==....=======.====
bag-kept-unit native ==....=======.====
bag-kept-unit object ==....=======.====
current-null-position empty =.............=.==
sav-unbound-null empty =.............=.==
sav-null-pack-position empty =.............=.==
`
