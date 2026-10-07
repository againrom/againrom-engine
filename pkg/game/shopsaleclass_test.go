package game

import (
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

type staffSaleSuitabilityRows []int32

func (c staffSaleSuitabilityRows) Len() int                  { return len(c) }
func (c staffSaleSuitabilityRows) EntryName(int) string      { return "equipment" }
func (c staffSaleSuitabilityRows) EntryStrings(int) []string { return nil }
func (c staffSaleSuitabilityRows) EntryParams(i int) []int32 {
	p := make([]int32, data.SutableForColumn+1)
	p[2], p[4], p[data.SutableForColumn] = 5, 5, c[i]
	return p
}

func TestShopSaleUsesWeaponSuitabilityAndRetainsOtherShelves(t *testing.T) {
	table := shopTable()
	table.Weapons = staffSaleSuitabilityRows{1, 2, 3, 0, 6, -1}
	table.Armors, table.Shields = staffSaleSuitabilityRows{2}, staffSaleSuitabilityRows{2}
	missing := *table
	missing.Weapons = nil
	cast := []sim.ItemEffect{{Kind: 41, Operand: 65537}}
	bound := func(suitable int32) sim.SourceEquipment {
		return sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: 4,
			Definition: sim.SourceWeaponDefinition{Present: true, Suitable: suitable}}
	}
	for _, tc := range []struct {
		name   string
		code   uint16
		kind   uint8
		table  *mapload.Table
		effect []sim.ItemEffect
		source sim.SourceEquipment
		want   ShopShelf
	}{
		{name: "mage only staff", code: 0x0101, table: table, effect: cast, want: ShelfMagic},
		{name: "mage only without cast effect", code: 0x0101, table: table, want: ShelfMagic},
		{name: "warrior enchanted weapon", code: 0x0100, table: table, effect: cast, want: ShelfWeapons},
		{name: "dual suitability", code: 0x0102, table: table, effect: cast, want: ShelfWeapons},
		{name: "neither suitability", code: 0x0103, table: table, effect: cast, want: ShelfWeapons},
		{name: "mage bit with other bits", code: 0x0104, table: table, want: ShelfMagic},
		{name: "negative dual suitability", code: 0x0105, table: table, want: ShelfWeapons},
		{name: "mage only armour", code: 0x0500, table: table, want: ShelfArmour},
		{name: "mage only shield", code: 0x0200, table: table, want: ShelfArmour},
		{name: "readable book priority", code: 0x0101, kind: 5, table: table, effect: []sim.ItemEffect{{Kind: 42, Operand: 1}}, want: ShelfBooks},
		{name: "potion priority", code: 0x0101, kind: 3, table: table, want: ShelfBooks},
		{name: "scroll priority", code: 0x0101, kind: 4, table: table, want: ShelfBooks},
		{name: "nil table fallback", code: 0x0101, effect: cast, want: ShelfWeapons},
		{name: "missing collection fallback", code: 0x0101, table: &missing, effect: cast, want: ShelfWeapons},
		{name: "unknown row fallback", code: 0x013f, table: table, effect: cast, want: ShelfWeapons},
		{name: "bound fighter overrides mage appearance", code: 0x0101, table: table, source: bound(1), effect: cast, want: ShelfWeapons},
		{name: "bound mage overrides fighter appearance", code: 0x0100, table: table, source: bound(2), effect: cast, want: ShelfMagic},
		{name: "bound neither overrides mage appearance", code: 0x0101, table: table, source: bound(0), effect: cast, want: ShelfWeapons},
		{name: "bound mage without table", code: 0x0100, source: bound(2), want: ShelfMagic},
		{name: "bound mage armour keeps armour", code: 0x0500, table: table, source: bound(2), want: ShelfArmour},
	} {
		for _, graph := range []bool{false, true} {
			route := "ordinary"
			if graph {
				route = "city objects"
			}
			t.Run(tc.name+"/"+route, func(t *testing.T) {
				item := sim.ItemInstance{Code: tc.code, Kind: tc.kind, Price: 5, Effects: tc.effect, SourceEquipment: tc.source}
				shop := NewShop(1000)
				shop.tbl = tc.table
				paid, ok := int32(0), false
				if !graph {
					if !shop.PutOnTable(shopItemFromInstance(item, 3)) {
						t.Fatal("could not stage synthetic sale")
					}
					paid, ok = shop.Sell()
				} else {
					f, screen := shopRoom(t, nil)
					f.Shop = shop
					if tc.table != nil {
						f.Table = tc.table
					}
					f.Carried[0].ID = "hero"
					cityShopSetTestPack(t, f, 0, sim.StackItem(item, 3))
					cityShopGraph(t, f)
					if action := screen.shopFromPack(0, true); strings.Contains(action.Msg, "cannot") {
						t.Fatal("city sale staging", action)
					}
					if places := f.Shop.Table(); len(places) != 1 || !staffSaleValueEqual(places[0].Instance(), item) || places[0].Count != 3 {
						t.Fatal("city route changed the sale control value", places)
					}
					gold := f.Town.Gold()
					action := screen.shopSell()
					paid, ok = int32(f.Town.Gold()-gold), !strings.Contains(action.Msg, "cannot")
					shop = f.Shop
				}
				if !ok || paid != 8 || len(shop.Table()) != 0 {
					t.Fatalf("sale paid%d/ok%v/table%d, want8/true/0", paid, ok, len(shop.Table()))
				}
				for shelf := ShopShelf(0); shelf < numShopShelves; shelf++ {
					got := int32(0)
					for _, stock := range shop.Shelf(shelf) {
						if uint16(stock.Code) == item.Code {
							if !staffSaleValueEqual(stock.Instance(), item) {
								t.Error("sale changed synthetic full value", stock)
							}
							got += stock.Count
						}
					}
					want := int32(0)
					if shelf == tc.want {
						want = 3
					}
					if got != want {
						t.Errorf("sale quantity on%s=%d, want%d", shelf, got, want)
					}
				}
			})
		}
	}
}
