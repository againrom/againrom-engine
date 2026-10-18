package game

import (
	"encoding/json"
	"image"
	"testing"

	"againrom/pkg/secondshop"
	"againrom/pkg/ui"
)

// secondShopPage is c's shop page with gold in the purse, shown.
func secondShopPage(c *secondCampaign, gold int) *secondCampaignScreen {
	s := &secondCampaignScreen{session: &CampaignSession{Town: &Town{second: c, gold: gold}}, install: &InstallResources{}}
	s.bindShop()
	s.shop.shown = nil
	return s
}

func (s *secondCampaignScreen) click(k ui.ShopControlKind, i int) ui.TownAction {
	return s.ShopClick(ui.ShopControl{Kind: k, Index: i})
}

// TestSecondShopPurchaseStopsAtTheBagBound refuses, before any gold moves,
// a purchase that would put the hero's bag with his offered units past what
// the SAVE producer writes; the campaign still saves.
func TestSecondShopPurchaseStopsAtTheBagBound(t *testing.T) {
	c := secondShopSample()
	*c.heroBag(), _ = secondShopTestTables.Units(secondshop.Item{Kind: secondshop.Magic, Row: 70, Quantity: maxSecondBagUnits - 1, Price: 83})
	s := secondShopPage(c, 1000)
	s.click(ui.ShopControlPackCell, 1)
	s.click(ui.ShopControlShelfPick, 0)
	s.click(ui.ShopControlShelfCell, 0)
	s.click(ui.ShopControlButton, 1)
	if s.session.Town.gold != 1000-125 || len(*c.heroBag()) != maxSecondBagUnits-1 || len(c.shop.selling()) != 1 {
		t.Fatalf("a purchase up to the bound: gold %d bag %d selling %v", s.session.Town.gold, len(*c.heroBag()), c.shop.selling())
	}
	s.click(ui.ShopControlShelfCell, 0)
	s.click(ui.ShopControlButton, 1)
	if s.session.Town.gold != 1000-125 || len(*c.heroBag()) != maxSecondBagUnits-1 || len(c.shop.buying()) != 1 {
		t.Fatalf("a purchase past the bound: gold %d bag %d buying %v", s.session.Town.gold, len(*c.heroBag()), c.shop.buying())
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
				"Hero": []secondshop.Item{{Kind: secondshop.Magic, Row: 70, Quantity: n, Price: 83}}}})
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
	armour := secondshop.Item{Kind: secondshop.Armor, Row: 3, Class: 1, Material: 2, Spell: 4, Level: 7, Quantity: 1, Price: 901}
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
	if s.session.Town.gold != 0 || len(*c.heroBag()) != 1 || secondShopUnits(c) != 1 {
		t.Fatalf("enchanted partial buy: gold %d bag %d pending %d", s.session.Town.gold, len(*c.heroBag()), secondShopUnits(c))
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
	if s.session.Town.gold != 0 || len(*c.heroBag()) != 1 || secondShopUnits(c) != 1 {
		t.Fatalf("book partial buy: gold %d bag %d pending %d", s.session.Town.gold, len(*c.heroBag()), secondShopUnits(c))
	}
	s.session.Town.gold = 1000
	s.click(ui.ShopControlButton, 1)
	v := s.ShopScreen()
	if len(*c.heroBag()) != 2 || v.Pack[1].Count != 1 || v.Pack[2].Count != 1 {
		t.Fatalf("two books in the bag view: bag %d cells %d %d", len(*c.heroBag()), v.Pack[1].Count, v.Pack[2].Count)
	}
	s.click(ui.ShopControlPackCell, 1)
	s.click(ui.ShopControlPackCell, 1)
	if got := c.shop.selling(); len(got) != 2 || got[0].Quantity != 1 || got[1].Quantity != 1 {
		t.Fatalf("two offered books: %v", got)
	}
}

// TestSecondShopPageTableScrolls stages six distinct items on the second
// game's unbounded table: the strip shows the newest place, the wheel over
// the table turns it one place at a time, every place is shown at some turn
// and the shown places cost what BUY asks, and the sixth place returns to
// its shelf by its own cell. The first game's table does not turn.
func TestSecondShopPageTableScrolls(t *testing.T) {
	c := secondShopSample()
	c.shop.shelves[0] = nil
	for i := 0; i < 6; i++ {
		c.shop.shelves[0] = append(c.shop.shelves[0], magicUnit(69+i, 1, int64(10+i)))
	}
	s := secondShopPage(c, 1000)
	s.click(ui.ShopControlShelfPick, 0)
	for i := 0; i < 6; i++ {
		s.click(ui.ShopControlShelfCell, i)
	}
	table := c.shop.Table()
	v := s.ShopScreen()
	if len(table) != 6 || v.TableOffset != 1 || v.Table[4].Count != 1 || v.Table[4].Price != table[5].ShopItem.Price {
		t.Fatalf("six places: table %d, strip from %d", len(table), v.TableOffset)
	}
	at := v.Layout.TableCellRect(0).Min.Add(image.Pt(40, 40))
	if got := v.Layout.WheelAt(at); got != ui.ShopWheelTable {
		t.Fatalf("the wheel over the second game's table turns region %d", got)
	}
	if got := ui.ShopWheelAt(at); got != ui.ShopWheelNone {
		t.Fatalf("the wheel over the first game's table turns region %d", got)
	}
	shown, cost := map[int]bool{}, int32(0)
	for _, turn := range []int{-1, -1, +1, +5} {
		s.ShopScroll(ui.ShopWheelTable, turn)
		v = s.ShopScreen()
		for i, cell := range v.Table {
			k := v.TableOffset + i
			if !cell.Occupied() || k >= len(table) || cell.Price != table[k].ShopItem.Price {
				t.Fatalf("strip from %d: cell %d shows %d, place %d", v.TableOffset, i, cell.Price, k)
			}
			if !shown[k] {
				shown[k], cost = true, cost+cell.Price*int32(cell.Count)
			}
		}
	}
	if len(shown) != 6 || cost != v.Buy || v.TableOffset != 1 {
		t.Fatalf("shown places %v cost %d, BUY %d, strip from %d", shown, cost, v.Buy, v.TableOffset)
	}
	sixth := table[5].ShopItem
	s.click(ui.ShopControlTableCell, 4)
	if got := c.shop.Table(); len(got) != 5 || !shopItemEqual(got[4].ShopItem, table[4].ShopItem) || len(c.shop.shelves[0]) != 6 || c.shop.shelves[0][5].Count != 1 || !shopItemEqual(c.shop.shelves[0][5], sixth) {
		t.Fatalf("the sixth place did not return to its shelf: table %d shelf %v", len(got), c.shop.shelves[0])
	}
}
