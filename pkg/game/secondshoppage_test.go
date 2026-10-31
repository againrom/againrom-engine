package game

import (
	"encoding/json"
	"fmt"
	"image"
	"slices"
	"testing"

	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// secondShopPage is c's shop page with gold in the purse, shown.
func secondShopPage(c *secondCampaign, gold int) *townScreen {
	s := &townScreen{shopController: shopController{sess: &CampaignSession{Town: &Town{second: c, gold: gold}}, in: &InstallResources{}}}
	s.bindBankShop()
	s.shop.shown = nil
	return s
}

func (s *townScreen) click(k ui.ShopControlKind, i int) ui.TownAction {
	return s.ShopClick(ui.ShopControl{Kind: k, Index: i})
}

// TestSecondShopPurchaseStopsAtTheBagBound refuses, before any gold moves,
// a purchase that would put the hero's bag with his offered units past what
// the SAVE producer writes; the campaign still saves.
func TestSecondShopPurchaseStopsAtTheBagBound(t *testing.T) {
	c := secondShopSample()
	*c.heroBag(), _ = secondShopTestTables.Units(secondshop.Item{Table: secondshop.Magic, Row: 70, Count: maxSecondBagUnits - 1, Price: 83})
	s := secondShopPage(c, 1000)
	s.click(ui.ShopControlPackCell, 1)
	s.click(ui.ShopControlShelfPick, 0)
	s.click(ui.ShopControlShelfCell, 0)
	s.click(ui.ShopControlButton, 1)
	if s.sess.Town.gold != 1000-125 || len(*c.heroBag()) != maxSecondBagUnits-1 || len(c.shop.selling()) != 1 {
		t.Fatalf("a purchase up to the bound: gold %d bag %d selling %v", s.sess.Town.gold, len(*c.heroBag()), c.shop.selling())
	}
	s.click(ui.ShopControlShelfCell, 0)
	s.click(ui.ShopControlButton, 1)
	if s.sess.Town.gold != 1000-125 || len(*c.heroBag()) != maxSecondBagUnits-1 || len(c.shop.buying()) != 1 {
		t.Fatalf("a purchase past the bound: gold %d bag %d buying %v", s.sess.Town.gold, len(*c.heroBag()), c.shop.buying())
	}
	if err := captureSecondCampaign(c).validate(); err != nil {
		t.Fatalf("SAVE refused after a refused purchase: %v", err)
	}
	s.click(ui.ShopControlButton, 0)
	if len(*c.heroBag()) != maxSecondBagUnits {
		t.Fatalf("undo: bag %d", len(*c.heroBag()))
	}
	if err := captureSecondCampaign(c).validate(); err != nil {
		t.Fatalf("SAVE refused a full bag: %v", err)
	}
}

// TestSecondShopOlderLargeStackLoads loads an older document whose one hero
// stack holds more units than its list's entry bound, and refuses one past
// the bag's bound.
func TestSecondShopOlderLargeStackLoads(t *testing.T) {
	doc := func(n int) []byte {
		raw, _ := json.Marshal(map[string]any{"Bank": make([]int32, 1024), "Current": currentSecondLocation{1, 10},
			"Available": []currentSecondLocation{{1, 10}}, "Shop": map[string]any{"Records": secondshop.NewGameRecords,
				"Hero": []secondHeroItem{{Kind: secondshop.Magic, Row: 70, Quantity: n, Price: 83}}}})
		return raw
	}
	var older currentSecondCampaign
	if err := json.Unmarshal(doc(maxSecondShopItems+1), &older); err != nil {
		t.Fatalf("an older stack of %d units is refused: %v", maxSecondShopItems+1, err)
	}
	c := older.restore()
	c.adoptLegacyHero(secondShopTestTables)
	if len(*c.heroBag()) != maxSecondShopItems+1 {
		t.Fatalf("bag %d units", len(*c.heroBag()))
	}
	raw, err := json.Marshal(captureSecondCampaign(c))
	if err == nil {
		err = json.Unmarshal(raw, new(currentSecondCampaign))
	}
	if err != nil {
		t.Fatalf("the loaded bag does not save: %v", err)
	}
	if err := json.Unmarshal(doc(maxSecondBagUnits+1), new(currentSecondCampaign)); err == nil {
		t.Fatal("an older stack past the bag's bound loads")
	}
}

// TestSecondShopPageKeepsUnstackableLotsApart stages two equal enchanted
// items and two units of one book through the page: each is its own table
// place, so a purse that pays for one buys one and leaves the next pending
// (R2-ENGINE-265, R2-ENGINE-266, DIV-3065); the bought books stay apart in
// the bag view and each offered book opens its own place.
func TestSecondShopPageKeepsUnstackableLotsApart(t *testing.T) {
	armour := secondshop.Item{Table: secondshop.Armor, Row: 3, Class: 1, Material: 2, Effects: []sim.ItemEffect{{Kind: 15, Operand: 4}}, Count: 1, Price: 901}
	c := secondShopSample()
	*c.heroBag() = nil
	c.shop.shelves[0] = secondShopItems(secondShopTestTables, []secondshop.Item{armour, armour})
	s := secondShopPage(c, 901)
	s.click(ui.ShopControlShelfPick, 0)
	s.click(ui.ShopControlShelfCell, 0)
	s.click(ui.ShopControlShelfCell, 1)
	if len(c.shop.Table()) != 2 {
		t.Fatalf("two enchanted items staged as %d places", len(c.shop.Table()))
	}
	s.click(ui.ShopControlButton, 1)
	if s.sess.Town.gold != 0 || len(*c.heroBag()) != 1 || secondShopUnits(c) != 1 {
		t.Fatalf("enchanted partial buy: gold %d bag %d pending %d", s.sess.Town.gold, len(*c.heroBag()), secondShopUnits(c))
	}

	c = secondShopSample()
	*c.heroBag() = nil
	s = secondShopPage(c, 1000)
	s.click(ui.ShopControlShelfPick, 3)
	s.click(ui.ShopControlShelfCell, 0)
	s.click(ui.ShopControlShelfCell, 0)
	if len(c.shop.Table()) != 2 {
		t.Fatalf("two units of a book staged as %d places", len(c.shop.Table()))
	}
	s.click(ui.ShopControlButton, 1)
	if s.sess.Town.gold != 0 || len(*c.heroBag()) != 1 || secondShopUnits(c) != 1 {
		t.Fatalf("book partial buy: gold %d bag %d pending %d", s.sess.Town.gold, len(*c.heroBag()), secondShopUnits(c))
	}
	s.sess.Town.gold = 1000
	s.click(ui.ShopControlButton, 1)
	v := s.ShopScreen()
	if len(*c.heroBag()) != 2 || v.Pack[1].Count != 1 || v.Pack[2].Count != 1 {
		t.Fatalf("two books in the bag view: bag %d cells %d %d", len(*c.heroBag()), v.Pack[1].Count, v.Pack[2].Count)
	}
	s.click(ui.ShopControlPackCell, 1)
	s.click(ui.ShopControlPackCell, 1)
	if got := c.shop.selling(); len(got) != 2 || got[0].Count != 1 || got[1].Count != 1 {
		t.Fatalf("two offered books: %v", got)
	}
}

// TestSecondShopPageTableHoldsFive fills the second game's five places:
// a sixth distinct lot is refused, a joining unit admitted, the wheel turns
// nothing, return and undo free places. The first game's holds five too.
func TestSecondShopPageTableHoldsFive(t *testing.T) {
	c := secondShopSample()
	c.shop.shelves[0] = []ShopItem{magicUnit(69, 2, 10)}
	for row := 70; row < 74; row++ {
		c.shop.shelves[0] = append(c.shop.shelves[0], magicUnit(row, 1, int64(row-59)))
	}
	*c.heroBag() = nil
	for row := 80; row < 83; row++ {
		units, _ := secondShopTestTables.Units(secondshop.Item{Table: secondshop.Magic, Row: row, Count: 1, Price: 20})
		*c.heroBag() = append(*c.heroBag(), units...)
	}
	s := secondShopPage(c, 1000)
	s.click(ui.ShopControlShelfPick, 0)
	s.click(ui.ShopControlShelfCell, 0)
	s.click(ui.ShopControlPackCell, s.packCell(0))
	s.click(ui.ShopControlPackCell, s.packCell(0))
	s.click(ui.ShopControlShelfCell, 1)
	s.click(ui.ShopControlShelfCell, 2)
	if got := c.shop.Table(); len(got) != ShopTablePlaces || len(*c.heroBag()) != 1 {
		t.Fatalf("five mixed lots: table %d places, bag %d", len(got), len(*c.heroBag()))
	}
	s.click(ui.ShopControlShelfCell, 3)
	s.click(ui.ShopControlPackCell, s.packCell(0))
	if got := c.shop.Table(); len(got) != ShopTablePlaces || c.shop.shelves[0][3].Count != 1 || len(*c.heroBag()) != 1 {
		t.Fatalf("a sixth distinct lot: table %d places, shelf entry %d, bag %d", len(got), c.shop.shelves[0][3].Count, len(*c.heroBag()))
	}
	s.click(ui.ShopControlShelfCell, 0)
	if got := c.shop.Table(); len(got) != ShopTablePlaces || got[0].ShopItem.Count != 2 || c.shop.shelves[0][0].Count != 0 {
		t.Fatalf("a joining unit at capacity: table %d places, first %d units", len(got), got[0].ShopItem.Count)
	}
	v := s.ShopScreen()
	at := v.Layout.TableCellRect(0).Min.Add(image.Pt(40, 40))
	if got := v.Layout.WheelAt(at); got != ui.ShopWheelNone {
		t.Fatalf("the wheel over the second game's table turns region %d", got)
	}
	for _, turn := range []int{+1, -1, +5} {
		s.ShopScroll(ui.ShopWheelTable, turn)
		if v = s.ShopScreen(); v.TableOffset != 0 {
			t.Fatalf("a turn of %d moved the strip to %d", turn, v.TableOffset)
		}
	}
	table := c.shop.Table()
	for i, cell := range v.Table {
		if !cell.Occupied() || int32(cell.Count) != table[i].ShopItem.Count {
			t.Fatalf("cell %d shows %d units, place %d holds %d", i, cell.Count, i, table[i].ShopItem.Count)
		}
	}
	mine := slices.IndexFunc(table, func(p ShopPlace) bool { return p.Mine })
	s.click(ui.ShopControlTableCell, mine)
	if got := c.shop.Table(); len(got) != ShopTablePlaces-1 || len(*c.heroBag()) != 2 {
		t.Fatalf("a returned place: table %d places, bag %d", len(got), len(*c.heroBag()))
	}
	s.click(ui.ShopControlPackCell, s.packCell(0))
	if got := c.shop.Table(); len(got) != ShopTablePlaces || len(*c.heroBag()) != 1 {
		t.Fatalf("the lot after a return: table %d places, bag %d", len(got), len(*c.heroBag()))
	}
	s.click(ui.ShopControlButton, 0)
	units := func(row int) (n int32) {
		for _, list := range c.shop.stock() {
			for _, it := range list {
				if it.Table == secondshop.Magic && it.Row == row {
					n += it.Count
				}
			}
		}
		return n
	}
	if len(c.shop.Table()) != 0 || len(*c.heroBag()) != 3 || units(69) != 2 || units(70) != 1 || units(71) != 1 {
		t.Fatalf("undo: table %d, bag %d, stock units %d %d %d", len(c.shop.Table()), len(*c.heroBag()), units(69), units(70), units(71))
	}
	s.click(ui.ShopControlShelfCell, 3)
	s.click(ui.ShopControlButton, 1)
	if len(c.shop.Table()) != 0 || s.sess.Town.gold != 1000-13 || len(*c.heroBag()) != 4 || c.shop.shelves[0][3].Count != 0 {
		t.Fatalf("the lot after undo, bought: table %d gold %d bag %d", len(c.shop.Table()), s.sess.Town.gold, len(*c.heroBag()))
	}

	f, first := shopRoom(t, nil)
	shelf := shopRoomShelves[roomWeapons].shelf
	f.Shop.shelves[shelf] = nil
	for i := range ShopTablePlaces + 1 {
		f.Shop.shelves[shelf] = append(f.Shop.shelves[shelf], ShopItem{Code: 0x0101, Price: int32(10 + i), Count: 1})
	}
	click(first, ui.ShopControlShelfPick, roomWeapons)
	for range ShopTablePlaces + 1 {
		click(first, ui.ShopControlShelfCell, 0)
	}
	first.ShopScroll(ui.ShopWheelTable, +1)
	if v := first.ShopScreen(); len(f.Shop.Table()) != ShopTablePlaces || len(f.Shop.Shelf(shelf)) != 1 || v.TableOffset != 0 || ui.ShopWheelAt(at) != ui.ShopWheelNone {
		t.Fatalf("first game: table %d, shelf %d, strip from %d", len(f.Shop.Table()), len(f.Shop.Shelf(shelf)), v.TableOffset)
	}
}

// TestSecondShopPageGoldFollowsTheBag draws the gold after the bag's items
// and a merchant's item on the table on the shelf's back, under the room's
// shop rules: a bought item takes the first cell ahead of the gold, a click
// on the gold offers nothing, and an item offered for sale keeps the bag's
// back (DIV-4090, DIV-4092).
func TestSecondShopPageGoldFollowsTheBag(t *testing.T) {
	c := secondShopSample()
	*c.heroBag() = nil
	s := secondShopPage(c, 1000)
	s.shop.page = func() *town.ShopSpec { return &town.ShopSpec{MoneyLast: true, TableShelfBack: true} }
	if v := s.ShopScreen(); !v.Pack[0].Money || v.Pack[1].Occupied() {
		t.Fatalf("empty bag: pack %+v", v.Pack[:2])
	}
	s.click(ui.ShopControlShelfPick, 0)
	s.click(ui.ShopControlShelfCell, 0)
	if v := s.ShopScreen(); v.Table[0].Back != ui.ShopBackAffordable {
		t.Fatalf("an item to buy draws back %d, want the shelf's %d", v.Table[0].Back, ui.ShopBackAffordable)
	}
	s.click(ui.ShopControlButton, 1)
	v := s.ShopScreen()
	if v.Pack[0].Money || !v.Pack[0].Occupied() || !v.Pack[1].Money || v.Pack[1].Count != 1000-125 {
		t.Fatalf("after a buy: pack %+v", v.Pack[:2])
	}
	s.click(ui.ShopControlPackCell, 1)
	if len(c.shop.Table()) != 0 {
		t.Fatalf("a click on the gold offered %v", c.shop.Table())
	}
	s.click(ui.ShopControlPackCell, 0)
	v = s.ShopScreen()
	if len(c.shop.selling()) != 1 || v.Table[0].Back != ui.ShopBackItem || !v.Pack[0].Money {
		t.Fatalf("offer: selling %v table back %d pack %+v", c.shop.selling(), v.Table[0].Back, v.Pack[0])
	}
	s.shop.page = nil
	if v := s.ShopScreen(); !v.Pack[0].Money {
		t.Fatalf("without the rules the gold is not first: %+v", v.Pack[0])
	}
}

// TestSecondShopRoomsCarryTheOriginalsArt checks the shop rooms' data: the
// pack strip, its arrows, the gold picture and backinv.bmp in every town;
// town 1's measured label offsets and its category layers, the subtree
// each category's region matches on the catalogue shots (DIV-4091).
func TestSecondShopRoomsCarryTheOriginalsArt(t *testing.T) {
	for id := 1; id <= 3; id++ {
		r := secondTownDescription(id).Room("shop")
		if r == nil || r.Shop == nil || !r.Shop.MoneyLast || !r.Shop.TableShelfBack {
			t.Fatalf("town %d: shop rules %+v", id, r)
		}
		keys := map[string]string{}
		for _, a := range r.Art {
			keys[a.Name] = a.Key
		}
		for name, key := range map[string]string{"pack": "graphics/interface/invframe.bmp", "pack-arrow": "graphics/interface/invarrow%d.bmp",
			"coin": "graphics/interface/money/money.16a", "back/1": "graphics/interface/backinv.bmp"} {
			if keys[name] != key {
				t.Errorf("town %d: %s is %q, want %q", id, name, keys[name], key)
			}
		}
	}
	r := secondTownDescription(1).Room("shop")
	if want := [4]town.Point{{1, 6}, {0, 4}, {1, 6}, {0, 6}}; r.Shop.LabelOffset != want {
		t.Errorf("town 1 label offsets %v, want %v", r.Shop.LabelOffset, want)
	}
	keys := map[string]string{}
	for _, a := range r.Scene.Art {
		keys[a.Name] = a.Key
	}
	at := map[string]town.Point{}
	for _, l := range r.Scene.Layers {
		at[l.Actor] = l.At
	}
	for i, want := range []struct {
		actor, key string
		at         town.Point
	}{{"rack-armour", "04", town.Point{353, 108}}, {"rack-weapons", "03", town.Point{197, 108}},
		{"rack-magic", "02", town.Point{313, 20}}, {"rack-books", "01", town.Point{201, 20}}} {
		if k := keys[fmt.Sprintf("rack/%d", i)]; k != "graphics/interface/shopanim/"+want.key+"/%d.bmp" || at[want.actor] != want.at {
			t.Errorf("category %d: art %q at %v, want subtree %s at %v", i, k, at[want.actor], want.key, want.at)
		}
	}
}
