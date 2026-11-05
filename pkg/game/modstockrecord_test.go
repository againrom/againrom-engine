package game

import (
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
)

// modStockRecordTable is a merged mod item table with a stocked armour, a
// stocked shield, a stocked weapon, an unstocked armour and a stocked
// clothing layer, each at its own price. Armour filler rows 4 to 27 are named
// and made in two shapes and three materials, so the plain armour pool is large enough
// for a restock to leave a mod item out.
func modStockRecordTable(t *testing.T) *mapload.Table {
	t.Helper()
	tab := fixtureItemTable(28, 5, 5)
	tab.Spells = nil
	for i, c := 4, tab.Armors.(*fixtureCollection); i < c.Len(); i++ {
		c.names[i], c.raw[i] = fmt.Sprintf("Filler %d", i), data.ItemMaskRaw([data.ShopShapes]uint16{7, 7})
	}
	armour, shield := testRow("a", "body", "Soft Mail"), testRow("s", "shield", "Buckler")
	plain, layer := testRow("u", "head", "Cap"), layerRow("g", "body", "Soft Mail", mod.LayerUnder, "body")
	armour.Stock, shield.Stock, layer.Stock = true, true, true
	shield.Price, plain.Price, layer.Price = 90, 60, 45
	if err := itemFront(tab).SetModItems(mod.ItemData{Rows: []mod.ItemRow{armour, shield, plain, layer}}); err != nil {
		t.Fatal(err)
	}
	tab.Mods.Items = append(tab.Mods.Items, mapload.ModItem{Mod: "m", Key: "w", Code: uint16(data.ComposeItemCode(0, 1, 0, 4)), Row: 4, Price: 150, Stock: true})
	return tab
}

// modStockRecord is what both games' shops place of the table's mod items at
// town arrival and after one, two and five restocks, and the layer lookup's
// answer for each code.
func modStockRecord(t *testing.T) []string {
	tab := modStockRecordTable(t)
	mods := make(map[uint16]bool)
	for _, m := range tab.Mods.Items {
		mods[m.Code] = true
	}
	var out []string
	record := func(game string, restocks int, shelves [][]ShopItem) {
		for k, shelf := range shelves {
			for i, it := range shelf {
				if mods[uint16(it.Code)] {
					out = append(out, fmt.Sprintf("%s r%d shelf %d #%d of %d: code %#04x price %d units %d kind %d", game, restocks, k, i, len(shelf), uint16(it.Code), it.Price, it.Count, it.Kind))
				}
			}
		}
	}
	marks := map[int]bool{0: true, 1: true, 2: true, 5: true}
	for _, seed := range []int64{7, 8} {
		for _, restockSeed := range []int64{11, 12, 13} {
			s := NewShop(5000)
			s.Generate(tab, seed)
			for n := 0; n <= 5; n++ {
				if n > 0 {
					s.Restock(tab, restockSeed)
				}
				if marks[n] {
					var shelves [][]ShopItem
					for k := ShopShelf(0); k < shopShelfCount; k++ {
						shelves = append(shelves, s.Shelf(k))
					}
					record(fmt.Sprintf("first seed %d/%d", seed, restockSeed), n, shelves)
				}
			}
		}
	}

	tables, _ := secondShopTables(tab)
	for _, it := range tables.Stocked {
		out = append(out, fmt.Sprintf("second stocked: table %d row %d class %d material %d price %d count %d effects %d", it.Table, it.Row, it.Class, it.Material, it.Price, it.Count, len(it.Effects)))
	}
	armours, shields := secondShopTestCollection{3: {0, 0, 0, 0, 6}}, secondShopTestCollection{}
	for _, m := range tab.Mods.Items {
		switch part := int32(m.Code>>8) & 0xf; {
		case part == 2:
			shields[int(m.Code&0x1f)] = []int32{0, 0, 0, 0, part}
		case part > 2:
			armours[int(m.Code&0x1f)] = []int32{0, 0, 0, 0, part}
		}
	}
	tables.Armors, tables.Shields = armours, shields
	for location := 1; location <= 3; location++ {
		state := uint32(location)
		rnd := func() int {
			state = state*1103515245 + 12345
			return int(state>>16) & 0x7fff
		}
		c := &secondCampaign{shop: newSecondShop()}
		for n := 0; n <= 5; n++ {
			c.shop.filled = 0
			c.shop.enter(tables, location, rnd, c.heroBag())
			if marks[n] {
				record(fmt.Sprintf("second location %d", location), n, c.shop.shelves[:])
			}
		}
	}

	for _, m := range tab.Mods.Items {
		got, ok := mapload.LayerItem(tab, m.Code)
		out = append(out, fmt.Sprintf("layer %#04x: %v key %q layer %d anchor %d", m.Code, ok, got.Key, got.Layer, got.Anchor))
	}
	for _, code := range []uint16{0, 0x0a0f} {
		_, ok := mapload.LayerItem(tab, code)
		out = append(out, fmt.Sprintf("layer %#04x: %v", code, ok))
	}
	return out
}

// TestModStockRecordBothShops holds both games' shelves of the mod items and
// the layer lookup's answers to the record taken before the shops shared one
// stock reader: the same items at the same prices, units and places.
func TestModStockRecordBothShops(t *testing.T) {
	got := strings.Join(modStockRecord(t), "\n")
	if got != modStockRecordWant {
		t.Fatalf("mod stock record changed:\n%s\nwant:\n%s", got, modStockRecordWant)
	}
}

const modStockRecordWant = `first seed 7/11 r0 shelf 0 #0 of 77: code 0x0104 price 150 units 1 kind 1
first seed 7/11 r0 shelf 0 #2 of 77: code 0x0205 price 90 units 1 kind 1
first seed 7/11 r0 shelf 0 #75 of 77: code 0x071e price 45 units 1 kind 1
first seed 7/11 r0 shelf 0 #76 of 77: code 0x071c price 120 units 1 kind 1
first seed 7/11 r1 shelf 0 #0 of 71: code 0x0104 price 150 units 1 kind 1
first seed 7/11 r1 shelf 0 #1 of 71: code 0x0205 price 90 units 1 kind 1
first seed 7/11 r1 shelf 0 #70 of 71: code 0x071c price 120 units 1 kind 1
first seed 7/11 r2 shelf 0 #0 of 81: code 0x0104 price 150 units 1 kind 1
first seed 7/11 r2 shelf 0 #2 of 81: code 0x0205 price 90 units 1 kind 1
first seed 7/11 r2 shelf 0 #78 of 81: code 0x071e price 45 units 1 kind 1
first seed 7/11 r2 shelf 0 #79 of 81: code 0x071c price 120 units 1 kind 1
first seed 7/11 r5 shelf 0 #73 of 74: code 0x071e price 45 units 1 kind 1
first seed 7/12 r0 shelf 0 #0 of 77: code 0x0104 price 150 units 1 kind 1
first seed 7/12 r0 shelf 0 #2 of 77: code 0x0205 price 90 units 1 kind 1
first seed 7/12 r0 shelf 0 #75 of 77: code 0x071e price 45 units 1 kind 1
first seed 7/12 r0 shelf 0 #76 of 77: code 0x071c price 120 units 1 kind 1
first seed 7/12 r1 shelf 0 #0 of 74: code 0x0104 price 150 units 1 kind 1
first seed 7/12 r1 shelf 0 #72 of 74: code 0x071e price 45 units 1 kind 1
first seed 7/12 r2 shelf 0 #73 of 74: code 0x071e price 45 units 1 kind 1
first seed 7/12 r5 shelf 0 #0 of 81: code 0x0104 price 150 units 1 kind 1
first seed 7/12 r5 shelf 0 #2 of 81: code 0x0205 price 90 units 1 kind 1
first seed 7/12 r5 shelf 0 #78 of 81: code 0x071e price 45 units 1 kind 1
first seed 7/12 r5 shelf 0 #79 of 81: code 0x071c price 120 units 1 kind 1
first seed 7/13 r0 shelf 0 #0 of 77: code 0x0104 price 150 units 1 kind 1
first seed 7/13 r0 shelf 0 #2 of 77: code 0x0205 price 90 units 1 kind 1
first seed 7/13 r0 shelf 0 #75 of 77: code 0x071e price 45 units 1 kind 1
first seed 7/13 r0 shelf 0 #76 of 77: code 0x071c price 120 units 1 kind 1
first seed 7/13 r1 shelf 0 #1 of 79: code 0x0205 price 90 units 1 kind 1
first seed 7/13 r1 shelf 0 #78 of 79: code 0x071c price 120 units 1 kind 1
first seed 7/13 r2 shelf 0 #0 of 71: code 0x0104 price 150 units 1 kind 1
first seed 7/13 r2 shelf 0 #70 of 71: code 0x071c price 120 units 1 kind 1
first seed 7/13 r5 shelf 0 #73 of 75: code 0x071e price 45 units 1 kind 1
first seed 8/11 r0 shelf 0 #0 of 76: code 0x0104 price 150 units 1 kind 1
first seed 8/11 r0 shelf 0 #1 of 76: code 0x0205 price 90 units 1 kind 1
first seed 8/11 r0 shelf 0 #74 of 76: code 0x071e price 45 units 1 kind 1
first seed 8/11 r0 shelf 0 #75 of 76: code 0x071c price 120 units 1 kind 1
first seed 8/11 r1 shelf 0 #0 of 71: code 0x0104 price 150 units 1 kind 1
first seed 8/11 r1 shelf 0 #1 of 71: code 0x0205 price 90 units 1 kind 1
first seed 8/11 r1 shelf 0 #70 of 71: code 0x071c price 120 units 1 kind 1
first seed 8/11 r2 shelf 0 #0 of 81: code 0x0104 price 150 units 1 kind 1
first seed 8/11 r2 shelf 0 #2 of 81: code 0x0205 price 90 units 1 kind 1
first seed 8/11 r2 shelf 0 #78 of 81: code 0x071e price 45 units 1 kind 1
first seed 8/11 r2 shelf 0 #79 of 81: code 0x071c price 120 units 1 kind 1
first seed 8/11 r5 shelf 0 #73 of 74: code 0x071e price 45 units 1 kind 1
first seed 8/12 r0 shelf 0 #0 of 76: code 0x0104 price 150 units 1 kind 1
first seed 8/12 r0 shelf 0 #1 of 76: code 0x0205 price 90 units 1 kind 1
first seed 8/12 r0 shelf 0 #74 of 76: code 0x071e price 45 units 1 kind 1
first seed 8/12 r0 shelf 0 #75 of 76: code 0x071c price 120 units 1 kind 1
first seed 8/12 r1 shelf 0 #0 of 74: code 0x0104 price 150 units 1 kind 1
first seed 8/12 r1 shelf 0 #72 of 74: code 0x071e price 45 units 1 kind 1
first seed 8/12 r2 shelf 0 #73 of 74: code 0x071e price 45 units 1 kind 1
first seed 8/12 r5 shelf 0 #0 of 81: code 0x0104 price 150 units 1 kind 1
first seed 8/12 r5 shelf 0 #2 of 81: code 0x0205 price 90 units 1 kind 1
first seed 8/12 r5 shelf 0 #78 of 81: code 0x071e price 45 units 1 kind 1
first seed 8/12 r5 shelf 0 #79 of 81: code 0x071c price 120 units 1 kind 1
first seed 8/13 r0 shelf 0 #0 of 76: code 0x0104 price 150 units 1 kind 1
first seed 8/13 r0 shelf 0 #1 of 76: code 0x0205 price 90 units 1 kind 1
first seed 8/13 r0 shelf 0 #74 of 76: code 0x071e price 45 units 1 kind 1
first seed 8/13 r0 shelf 0 #75 of 76: code 0x071c price 120 units 1 kind 1
first seed 8/13 r1 shelf 0 #1 of 79: code 0x0205 price 90 units 1 kind 1
first seed 8/13 r1 shelf 0 #78 of 79: code 0x071c price 120 units 1 kind 1
first seed 8/13 r2 shelf 0 #0 of 71: code 0x0104 price 150 units 1 kind 1
first seed 8/13 r2 shelf 0 #70 of 71: code 0x071c price 120 units 1 kind 1
first seed 8/13 r5 shelf 0 #73 of 75: code 0x071e price 45 units 1 kind 1
second stocked: table 1 row 28 class 0 material 0 price 120 count 1 effects 0
second stocked: table 2 row 5 class 0 material 0 price 90 count 1 effects 0
second stocked: table 1 row 30 class 0 material 0 price 45 count 1 effects 0
second location 1 r0 shelf 0 #0 of 3: code 0x071c price 120 units 1 kind 1
second location 1 r0 shelf 0 #1 of 3: code 0x0205 price 90 units 1 kind 0
second location 1 r0 shelf 0 #2 of 3: code 0x071e price 45 units 1 kind 1
second location 1 r1 shelf 0 #0 of 3: code 0x071c price 120 units 1 kind 1
second location 1 r1 shelf 0 #1 of 3: code 0x0205 price 90 units 1 kind 0
second location 1 r1 shelf 0 #2 of 3: code 0x071e price 45 units 1 kind 1
second location 1 r2 shelf 0 #0 of 3: code 0x071c price 120 units 1 kind 1
second location 1 r2 shelf 0 #1 of 3: code 0x0205 price 90 units 1 kind 0
second location 1 r2 shelf 0 #2 of 3: code 0x071e price 45 units 1 kind 1
second location 1 r5 shelf 0 #0 of 3: code 0x071c price 120 units 1 kind 1
second location 1 r5 shelf 0 #1 of 3: code 0x0205 price 90 units 1 kind 0
second location 1 r5 shelf 0 #2 of 3: code 0x071e price 45 units 1 kind 1
second location 2 r0 shelf 0 #0 of 3: code 0x071c price 120 units 1 kind 1
second location 2 r0 shelf 0 #1 of 3: code 0x0205 price 90 units 1 kind 0
second location 2 r0 shelf 0 #2 of 3: code 0x071e price 45 units 1 kind 1
second location 2 r1 shelf 0 #0 of 3: code 0x071c price 120 units 1 kind 1
second location 2 r1 shelf 0 #1 of 3: code 0x0205 price 90 units 1 kind 0
second location 2 r1 shelf 0 #2 of 3: code 0x071e price 45 units 1 kind 1
second location 2 r2 shelf 0 #0 of 3: code 0x071c price 120 units 1 kind 1
second location 2 r2 shelf 0 #1 of 3: code 0x0205 price 90 units 1 kind 0
second location 2 r2 shelf 0 #2 of 3: code 0x071e price 45 units 1 kind 1
second location 2 r5 shelf 0 #0 of 3: code 0x071c price 120 units 1 kind 1
second location 2 r5 shelf 0 #1 of 3: code 0x0205 price 90 units 1 kind 0
second location 2 r5 shelf 0 #2 of 3: code 0x071e price 45 units 1 kind 1
second location 3 r0 shelf 3 #0 of 3: code 0x071c price 120 units 1 kind 1
second location 3 r0 shelf 3 #1 of 3: code 0x0205 price 90 units 1 kind 0
second location 3 r0 shelf 3 #2 of 3: code 0x071e price 45 units 1 kind 1
second location 3 r1 shelf 3 #0 of 3: code 0x071c price 120 units 1 kind 1
second location 3 r1 shelf 3 #1 of 3: code 0x0205 price 90 units 1 kind 0
second location 3 r1 shelf 3 #2 of 3: code 0x071e price 45 units 1 kind 1
second location 3 r2 shelf 3 #0 of 3: code 0x071c price 120 units 1 kind 1
second location 3 r2 shelf 3 #1 of 3: code 0x0205 price 90 units 1 kind 0
second location 3 r2 shelf 3 #2 of 3: code 0x071e price 45 units 1 kind 1
second location 3 r5 shelf 3 #0 of 3: code 0x071c price 120 units 1 kind 1
second location 3 r5 shelf 3 #1 of 3: code 0x0205 price 90 units 1 kind 0
second location 3 r5 shelf 3 #2 of 3: code 0x071e price 45 units 1 kind 1
layer 0x071c: false key "" layer 0 anchor 0
layer 0x0205: false key "" layer 0 anchor 0
layer 0x061d: false key "" layer 0 anchor 0
layer 0x071e: true key "g" layer -1 anchor 7
layer 0x0104: false key "" layer 0 anchor 0
layer 0x0000: false
layer 0x0a0f: false`
