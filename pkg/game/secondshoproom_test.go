package game

import (
	"testing"

	"againrom/pkg/ui"
)

// TestSecondShopPageTradesThroughTheOneController drives the second game's
// shop through the one page's controls: a rack, a shelf cell, the quantity
// convention, a buy, a pack cell, a sale and the undo button. The deal,
// gold and the hero's bag move by the second game's rules.
func TestSecondShopPageTradesThroughTheOneController(t *testing.T) {
	c := secondShopSample()
	sess := &CampaignSession{Town: &Town{second: c, gold: 300}}
	sc := &secondCampaignScreen{session: sess, install: &InstallResources{}}
	sc.bindShop()
	sc.shop.shown = nil
	click := func(k ui.ShopControlKind, i int, shift bool) {
		sc.ShopClick(ui.ShopControl{Kind: k, Index: i, Shift: shift})
	}
	click(ui.ShopControlShelfPick, 0, false)
	click(ui.ShopControlShelfCell, 0, false)
	if got := c.shop.buying(); len(got) != 1 || got[0].Count != 1 || c.shop.shelves[0][0].Count != 2 {
		t.Fatalf("one unit: buying %v shelf %v", got, c.shop.shelves[0])
	}
	click(ui.ShopControlShelfCell, 0, true)
	if got := c.shop.buying(); len(got) != 1 || got[0].Count != 3 {
		t.Fatalf("whole lot: buying %v", got)
	}
	click(ui.ShopControlButton, 1, false)
	if sess.Town.gold != 300 || len(c.shop.buying()) != 1 || len(*c.heroBag()) != 4 {
		t.Fatalf("unaffordable buy: gold %d buying %v bag %d", sess.Town.gold, c.shop.buying(), len(*c.heroBag()))
	}
	click(ui.ShopControlButton, 0, false)
	if len(c.shop.Table()) != 0 || c.shop.shelves[0][0].Count != 3 {
		t.Fatalf("undo: table %v shelf %v", c.shop.Table(), c.shop.shelves[0])
	}
	click(ui.ShopControlShelfCell, 0, false)
	click(ui.ShopControlShelfCell, 0, false)
	click(ui.ShopControlButton, 1, false)
	if sess.Town.gold != 300-2*125 || len(c.shop.Table()) != 0 || len(*c.heroBag()) != 4+2 {
		t.Fatalf("buy: gold %d table %v bag %d", sess.Town.gold, c.shop.Table(), len(*c.heroBag()))
	}
	v := sc.ShopScreen()
	if v.Layout == nil || v.Layout.Character || v.Layout.Book || len(v.Pack[1].Text) == 0 || !v.Pack[0].Money {
		t.Fatalf("view: layout %v pack %+v", v.Layout, v.Pack[:2])
	}
	click(ui.ShopControlPackCell, 1, false)
	if got := c.shop.selling(); len(got) != 1 || got[0].Count != 1 || got[0].Price != 83 {
		t.Fatalf("offer: selling %v", got)
	}
	gold := sess.Town.gold
	click(ui.ShopControlButton, 2, false)
	if sess.Town.gold != gold+42 || len(c.shop.Table()) != 0 || len(*c.heroBag()) != 5 {
		t.Fatalf("sale: gold %d (from %d) table %v bag %d", sess.Town.gold, gold, c.shop.Table(), len(*c.heroBag()))
	}
}
