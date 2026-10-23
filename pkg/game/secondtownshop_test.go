package game

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/shopstock"
	"againrom/pkg/sim"
)

// secondShopTestCollection is a data.bin collection of parameter rows.
type secondShopTestCollection map[int][]int32

func (c secondShopTestCollection) Len() int                   { return 128 }
func (c secondShopTestCollection) EntryParams(i int) []int32  { return c[i] }
func (c secondShopTestCollection) EntryRaw(int) []byte        { return nil }
func (c secondShopTestCollection) EntryDoubles(int) []float64 { return nil }

// secondShopTestTables code armour row 3 as body part 6.
var secondShopTestTables = secondshop.Tables{Armors: secondShopTestCollection{3: {0, 0, 0, 0, 6}}}

// secondShopSample is a campaign in a mission whose shop holds a drawn
// stock and whose hero carries four potions.
func secondShopSample() *secondCampaign {
	c := &secondCampaign{current: partyPlace{1, 10}, available: []partyPlace{{1, 10}}, shop: newSecondShop()}
	c.shop.filled, c.shop.tables = 1, secondShopTestTables
	c.shop.records[3][3].Mask |= secondShopClass4
	c.shop.shelves[0] = secondShopItems(secondShopTestTables, []secondshop.Item{{Table: secondshop.Armor, Row: 3, Class: 1, Material: 2, Count: 3, Price: 125}})
	c.shop.shelves[3] = secondShopItems(secondShopTestTables, []secondshop.Item{{Table: secondshop.Book, Spell: 5, Count: 2, Price: 1000}})
	*c.heroBag(), _ = secondShopTestTables.Units(secondshop.Item{Table: secondshop.Magic, Row: 70, Count: 4, Price: 83})
	return c
}

// TestSecondShopDealSettlesGoldAndItems runs a purchase and a sale through
// the campaign's shop: gold moves by q*P and floor(q*P/2+0.5), items move
// between the shelf and the hero's bag, and an unaffordable purchase stays
// open.
func TestSecondShopDealSettlesGoldAndItems(t *testing.T) {
	c := secondShopSample()
	s, bag := &c.shop, c.heroBag()
	gold := 300
	if !s.take(0) || !s.take(0) || s.shelves[0][0].Count != 1 || len(s.buying()) != 1 || s.buying()[0].Count != 2 {
		t.Fatalf("take: stock %v buying %v", s.shelves[0], s.buying())
	}
	if !s.buy(bag, &gold) || gold != 50 || len(s.buying()) != 0 || len(c.heroItems()) != 2 || c.heroItems()[1].Count != 2 || len(*bag) != 6 {
		t.Fatalf("buy: gold %d buying %v hero %v", gold, s.buying(), c.heroItems())
	}
	if (*bag)[4].Code != 2<<12|6<<8|1<<5|3 || (*bag)[0].Code != 0x0e00|70 {
		t.Fatalf("bag codes %#04x %#04x", (*bag)[4].Code, (*bag)[0].Code)
	}
	s.take(0)
	if s.buy(bag, &gold) || gold != 50 || len(s.buying()) != 1 {
		t.Fatalf("unaffordable buy settled: gold %d buying %v", gold, s.buying())
	}
	if !s.offer(bag, 1) || !s.sell(bag, &gold) || gold != 50+125 || s.shelves[0][0].Count != 2 {
		t.Fatalf("sell: gold %d stock %v", gold, s.shelves[0])
	}
	s.leave(bag)
	if len(s.buying()) != 0 || s.shelves[0][0].Count != 3 || len(c.heroItems()) != 1 {
		t.Fatalf("leave: stock %v hero %v", s.shelves[0], c.heroItems())
	}
	// Loss control: an odd price credits floor(q*P/2+0.5), not the cell's
	// ((P+1)/2)*q.
	*bag, _ = secondShopTestTables.Units(secondshop.Item{Table: secondshop.Magic, Row: 70, Count: 3, Price: 83})
	gold = 0
	s.offer(bag, 0)
	s.sell(bag, &gold)
	if gold != 125 || len(*bag) != 0 {
		t.Fatalf("sale credit of 3 at 83 = %d, want 125; bag %d", gold, len(*bag))
	}
}

// TestSecondShopSaveRoundTrip writes the shop with an open deal: the SAV
// holds the records and the hero, through the party member producer, with
// the deal undone and no stock; the live deal stays open, a document with a
// stock in it loads with that stock dropped, and an old save without a shop
// restores the new game's records.
func TestSecondShopSaveRoundTrip(t *testing.T) {
	c := secondShopSample()
	c.shop.take(0)
	c.shop.offer(c.heroBag(), 0)
	captured := captureSecondCampaign(c)
	offered := 0
	for _, it := range c.shop.selling() {
		offered += int(it.Count)
	}
	if len(c.shop.buying()) != 1 || offered != 4 || len(*c.heroBag()) != 0 || c.shop.shelves[0][0].Count != 2 {
		t.Fatal("capture changed the live deal")
	}
	raw, err := json.Marshal(captured)
	if err != nil {
		t.Fatal(err)
	}
	if fields := secondShopWrittenFields(t, raw); !reflect.DeepEqual(fields, []string{"Records"}) {
		t.Fatalf("the written shop holds fields %v, want only Records: a SAV carries no stock", fields)
	}
	var decoded currentSecondCampaign
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	sample := secondShopSample()
	want := secondShop{records: sample.shop.records}
	restored := decoded.restore()
	if !reflect.DeepEqual(restored.shop, want) || !reflect.DeepEqual(restored.heroItems(), sample.heroItems()) {
		t.Fatalf("restored shop %+v hero %v, want %+v %v", restored.shop, restored.heroItems(), want, sample.heroItems())
	}
	// Loss control: the stock the live shop holds is not what a LOAD restores.
	if reflect.DeepEqual(sample.shop, want) {
		t.Fatal("loss control compares equal")
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	shop := doc["Shop"].(map[string]any)
	shop["Stock"] = []any{[]any{map[string]any{"Kind": 1, "Row": 3, "Quantity": 9, "Price": 125}}, nil, nil, nil}
	shop["Filled"] = 1
	written, _ := json.Marshal(doc)
	var earlier currentSecondCampaign
	if err := json.Unmarshal(written, &earlier); err != nil {
		t.Fatal("a document this engine wrote with a stock is refused:", err)
	}
	if got := earlier.restore().shop; !reflect.DeepEqual(got, want) {
		t.Fatalf("a written stock came back: %+v", got)
	}
	old, _ := json.Marshal(struct {
		Bank      []int32
		Current   currentSecondLocation
		Available []currentSecondLocation
	}{make([]int32, 1024), currentSecondLocation{1, 10}, []currentSecondLocation{{1, 10}}})
	var legacy currentSecondCampaign
	if err := json.Unmarshal(old, &legacy); err != nil {
		t.Fatal(err)
	}
	if got := legacy.restore().shop; !reflect.DeepEqual(got, newSecondShop()) {
		t.Fatalf("an old save's shop %+v", got)
	}
	if fresh := captureSecondCampaign(newSecondCampaign()); fresh.Shop != nil || fresh.Party != nil {
		t.Fatal("a new game's shop or party is written")
	}
	drawn := newSecondCampaign()
	drawn.shop.shelves, drawn.shop.filled = sample.shop.shelves, 1
	if captureSecondCampaign(drawn).Shop != nil {
		t.Fatal("a new game's drawn stock writes a shop")
	}
	for name, change := range map[string]func(d map[string]any){
		"code":    func(d map[string]any) { secondShopBagUnit(d)["Code"] = 0x0f00 },
		"price":   func(d map[string]any) { secondShopBagUnit(d)["Price"] = -1 },
		"members": func(d map[string]any) { d["Party"] = append(d["Party"].([]any), d["Party"].([]any)[0]) },
		"unknown": func(d map[string]any) { d["Shop"].(map[string]any)["Deal"] = 1 },
	} {
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		change(doc)
		bad, _ := json.Marshal(doc)
		var refused currentSecondCampaign
		if err := json.Unmarshal(bad, &refused); err == nil || strings.Contains(err.Error(), "bank") {
			t.Errorf("%s: a malformed document decoded: %v", name, err)
		}
	}
}

// secondShopBagUnit is the first bag unit of a written document.
func secondShopBagUnit(doc map[string]any) map[string]any {
	member := doc["Party"].([]any)[0].(map[string]any)["Member"].(map[string]any)
	return member["CarriedItems"].([]any)[0].(map[string]any)
}

// TestSecondShopOlderHeroListLoadsIntoBag reads a document whose shop holds
// the hero list an earlier engine wrote: the list waits for the install's
// tables, is written back unchanged while it waits, and then becomes bag
// units the page lists as before.
func TestSecondShopOlderHeroListLoadsIntoBag(t *testing.T) {
	older := []secondshop.Item{
		{Table: secondshop.Armor, Row: 3, Class: 1, Material: 2, Spell: 4, Level: 7, Count: 1, Price: 900},
		{Table: secondshop.Magic, Row: 70, Count: 2, Price: 83},
		{Table: secondshop.Book, Spell: 5, Count: 1, Price: 1000},
	}
	raw, _ := json.Marshal(map[string]any{"Bank": make([]int32, 1024), "Current": currentSecondLocation{1, 10},
		"Available": []currentSecondLocation{{1, 10}}, "Shop": map[string]any{"Records": secondshop.NewGameRecords, "Hero": secondHeroDocument(older)}})
	var doc currentSecondCampaign
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	c := doc.restore()
	if again := captureSecondCampaign(c); again.Shop == nil || !reflect.DeepEqual(again.Shop.Hero, secondHeroDocument(older)) || again.Party != nil {
		t.Fatalf("a waiting hero list was not written back: %+v", again.Shop)
	}
	c.adoptLegacyHero(secondShopTestTables)
	if got := c.heroItems(); !reflect.DeepEqual(got, older) || len(*c.heroBag()) != 4 {
		t.Fatalf("bag %v (%d units), want %v", got, len(*c.heroBag()), older)
	}
	after := captureSecondCampaign(c)
	if after.Shop != nil || len(after.Party) != 1 {
		t.Fatalf("after loading: shop %+v party %d", after.Shop, len(after.Party))
	}
	if e := (*c.heroBag())[0].Effects; len(e) != 1 || e[0].Kind != 41 || e[0].Operand != 4|7<<16 {
		t.Fatalf("enchantment effect %v", e)
	}
	if e := (*c.heroBag())[3].Effects; len(e) != 1 || e[0].Kind != 42 || e[0].Operand != 5 {
		t.Fatalf("book effect %v", e)
	}
}

// secondHeroDocument is items as an older document's hero list writes them.
func secondHeroDocument(items []secondshop.Item) []secondHeroItem {
	var out []secondHeroItem
	for _, it := range items {
		out = append(out, secondHeroItem{Kind: it.Table, Row: it.Row, Class: it.Class, Material: it.Material, Spell: it.Spell, Level: it.Level,
			Quantity: int(it.Count), Price: int64(it.Price)})
	}
	return out
}

// secondShopWrittenFields answers the sorted field names of the shop a
// written second campaign document holds.
func secondShopWrittenFields(t *testing.T, raw []byte) []string {
	t.Helper()
	var doc struct{ Shop map[string]json.RawMessage }
	if err := json.Unmarshal(raw, &doc); err != nil || doc.Shop == nil {
		t.Fatal("no written shop:", err)
	}
	var out []string
	for k := range doc.Shop {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// TestSecondShopModStockReachesShelf: a fixture mod's stocked armour joins
// the second game's stock by placement on a fill and is bought into the bag;
// loss control: the mod's unstocked item stays off every shelf.
func TestSecondShopModStockReachesShelf(t *testing.T) {
	const code = 2<<12 | 6<<8 | 1<<5 | 3
	tab := &mapload.Table{}
	tab.Mods.Items = []mapload.ModItem{
		{Mod: "fixture", Key: "plate", Code: code, Price: 77, Stock: true},
		{Mod: "fixture", Key: "cap", Code: 2<<12 | 6<<8 | 1<<5 | 4, Price: 55},
	}
	tables, _ := secondShopTables(tab)
	tables.Armors = secondShopTestTables.Armors
	c := &secondCampaign{shop: newSecondShop()}
	c.shop.enter(tables, 1, func() int { return 0 }, c.heroBag())
	shelf, at := -1, -1
	for k, list := range c.shop.stock() {
		for i, it := range list {
			if it.Table == secondshop.Armor && it.Price == 55 {
				t.Fatalf("unstocked mod item on shelf %d", k)
			}
			if it.Table == secondshop.Armor && it.Row == 3 && it.Price == 77 {
				shelf, at = k, i
			}
		}
	}
	if shelf < 0 {
		t.Fatalf("stocked mod item on no shelf: %v", c.shop.shelves)
	}
	gold := 100
	c.shop.category = shelf
	if !c.shop.take(at) || !c.shop.buy(c.heroBag(), &gold) || gold != 23 || len(*c.heroBag()) != 1 || (*c.heroBag())[0].Code != code {
		t.Fatalf("buy of the mod item: gold %d bag %v", gold, *c.heroBag())
	}
}

// heroItems is the hero's bag, one entry per stack.
func (c *secondCampaign) heroItems() []secondshop.Item {
	items, _ := secondshop.Stacks(*c.heroBag())
	return items
}

// visibleShelf answers the shown category's entries with units left.
func (s *secondShop) visibleShelf() []int {
	var out []int
	for i, it := range s.shelf() {
		if it.Count > 0 {
			out = append(out, i)
		}
	}
	return out
}

// TestSecondShopPutDropsAnEntryWithNoReadBack: a shelf entry whose bag unit
// reads back as no table item (armour row 3 coded as the shield part) leaves
// the shelf when an item is filed there, and every other entry keeps its
// place and count: the shelf a read-back of every entry gives.
func TestSecondShopPutDropsAnEntryWithNoReadBack(t *testing.T) {
	tables := secondshop.Tables{Armors: secondShopTestCollection{3: {0, 0, 0, 0, 2}, 4: {0, 0, 0, 0, 6}}}
	readBack := func(shelf []ShopItem, i ShopItem) []ShopItem {
		return secondShopItems(tables, shopstock.Insert(secondItems(shelf), secondshop.Stacking, secondItem(i)))
	}
	shelf := secondShopItems(tables, []secondshop.Item{
		{Table: secondshop.Armor, Row: 3, Count: 1, Price: 40},
		{Table: secondshop.Armor, Row: 4, Count: 2, Price: 50},
	})
	if len(shelf) != 2 || secondItem(shelf[0]).Table != 0 || secondItem(shelf[1]).Row != 4 {
		t.Fatalf("shelf %+v: want the unreadable row 3, then row 4", shelf)
	}
	s := newSecondShop()
	s.filled, s.tables = 1, tables
	unit, _ := secondShopItem(tables, secondshop.Item{Table: secondshop.Armor, Row: 4, Count: 1, Price: 50})
	if got, want := s.rules().put(shelf, unit), readBack(shelf, unit); !reflect.DeepEqual(got, want) || len(got) != 1 || got[0].Count != 3 {
		t.Fatalf("put %+v, want %+v", got, want)
	}
	s.shelves[0] = shelf
	if !s.take(1) || len(s.shelves[0]) != 2 || s.shelves[0][1].Count != 1 {
		t.Fatalf("take: shelf %+v", s.shelves[0])
	}
	before := slices.Clone(s.shelves[0])
	var bag []sim.ItemInstance
	s.undo(&bag)
	if want := readBack(before, unit); !reflect.DeepEqual(s.shelves[0], want) || len(want) != 1 || want[0].Count != 2 || len(bag) != 0 {
		t.Fatalf("undo: shelf %+v, want %+v; bag %v", s.shelves[0], want, bag)
	}
}
