package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The number a shop grid prints for an item's stored price (SHOP-SCREEN-037,
// ITEM-PRICETAG-144): the merchant's side prints the price the client holds and
// the player's side (price+1)/2 with a truncating division. Every case below is
// a literal from the claim's own table of grid texts or from the arithmetic it
// states, not a value computed by the code under test.

// gridWeapon is an equipment code (class 1) and gridPotion a class 14 item
// with a positive price; the access item is data.QuestDocumentCode.
const (
	gridWeapon data.ItemCode = 0x0101
	gridPotion data.ItemCode = 0x0e06
)

func TestShopGridPriceIsTheFigureTheOriginalPrints(t *testing.T) {
	quest := data.QuestDocumentCode
	for _, tc := range []struct {
		name        string
		code        data.ItemCode
		price       int32
		shelf, mine int32
	}{
		{"999", gridWeapon, 999, 999, 500},
		{"1,250", gridWeapon, 1250, 1250, 625},
		{"99,999", gridWeapon, 99999, 99999, 50000},
		{"9,999,999", gridWeapon, 9999999, 9999999, 5000000},
		{"one coin", gridWeapon, 1, 1, 1},
		{"two coins", gridWeapon, 2, 2, 1},
		{"three coins", gridWeapon, 3, 3, 2},
		{"0", gridWeapon, 0, 0, 0},
		{"-1", gridWeapon, -1, -1, 0},
		{"-1,250", gridWeapon, -1250, -1250, -624},
		{"a class 14 item at 0", quest, 0, 0, 0},
		{"a class 14 item at the -1 sentinel", quest, -1, 0, 0},
		{"a class 14 item at -1250 keeps it", quest, -1250, -1250, -624},
		{"a class 14 item with a price", gridPotion, 50, 50, 25},
	} {
		if got := shopGridPrice(tc.code, tc.price, false); got != tc.shelf {
			t.Errorf("%s: the merchant's side prints %d, want %d", tc.name, got, tc.shelf)
		}
		if got := shopGridPrice(tc.code, tc.price, true); got != tc.mine {
			t.Errorf("%s: the player's side prints %d, want %d", tc.name, got, tc.mine)
		}
	}
}

// The shelf and the table hand the screen the printed number of an item priced 0
// or below, and every such cell is occupied: an equipment item at 0 and at -1
// keeps its price on the merchant's side, a class 14 item at -1 carries 0, and
// the player's places print 0 for both.
func TestShopViewCellsCarryThePrintedNumberOfAnUnpricedItem(t *testing.T) {
	f, s := shopRoom(t, nil)
	stock := []ShopItem{
		{Code: data.QuestDocumentCode, Price: -1, Count: 1},
		{Code: gridWeapon, Price: 0, Count: 1},
		{Code: gridWeapon, Price: -1, Count: 1},
		{Code: gridWeapon, Price: 1250, Count: 1},
	}
	f.Shop.shelves[ShelfArmour] = append([]ShopItem(nil), stock...)
	click(s, ui.ShopControlShelfPick, roomArmour)
	view := s.ShopScreen()
	for i, want := range []int32{0, 0, -1, 1250} {
		if c := view.Shelf[i]; !c.Occupied() || c.Money || c.Price != want {
			t.Errorf("shelf cell %d: occupied %v money %v prints %d, want an occupied item cell printing %d", i, c.Occupied(), c.Money, c.Price, want)
		}
	}

	for range stock {
		if !f.Shop.TakeFromShelf(ShelfArmour, 0, 1) {
			t.Fatal("could not stage a merchant's place")
		}
	}
	view = s.ShopScreen()
	for i, want := range []int32{0, 0, -1, 1250} {
		if c := view.Table[i]; !c.Occupied() || c.Mine || c.Price != want {
			t.Errorf("merchant table cell %d: occupied %v mine %v prints %d, want an occupied merchant cell printing %d", i, c.Occupied(), c.Mine, c.Price, want)
		}
	}

	f.Shop.ClearTable()
	for _, item := range stock {
		if !f.Shop.PutOnTable(item) {
			t.Fatal("could not stage a player's place")
		}
	}
	view = s.ShopScreen()
	for i, want := range []int32{0, 0, 0, 625} {
		if c := view.Table[i]; !c.Occupied() || !c.Mine || c.Price != want {
			t.Errorf("player's table cell %d: occupied %v mine %v prints %d, want an occupied player's cell printing %d", i, c.Occupied(), c.Mine, c.Price, want)
		}
	}
}

// The number a grid chooses an item's plaque from is the unit price the client
// holds (SHOP-SCREEN-037, ITEM-PRICETAG-144): the stored price, and 0 for a
// class 14 item at the -1 sentinel (SHOP-MISSION-019). It is the same on both
// sides of the deal.
func TestShopClientPriceIsThePriceTheClientHolds(t *testing.T) {
	quest := data.QuestDocumentCode
	for _, tc := range []struct {
		name        string
		code        data.ItemCode
		price, want int32
	}{
		{"a stored price", gridWeapon, 1250, 1250},
		{"10", gridWeapon, 10, 10},
		{"0", gridWeapon, 0, 0},
		{"-1 on equipment", gridWeapon, -1, -1},
		{"-1,250", gridWeapon, -1250, -1250},
		{"a class 14 item at the -1 sentinel", quest, -1, 0},
		{"a class 14 item at -1250 keeps it", quest, -1250, -1250},
		{"a class 14 item with a price", gridPotion, 50, 50},
	} {
		if got := shopClientPrice(tc.code, tc.price); got != tc.want {
			t.Errorf("%s: the plaque is chosen from %d, want %d", tc.name, got, tc.want)
		}
	}
}

// Each grid cell carries two numbers: the figure it prints and the stored price
// its plaque is chosen from. On the shelf and on the merchant's places they are
// equal; on the player's places and in the pack the figure is half the price
// with a truncating division, and the plaque still follows the whole price, so a
// price of 10 prints 5 on the second plaque and 100 prints 50 on the third. A
// class 14 item at the -1 sentinel carries 0 in both.
func TestShopViewCellsCarryTheStoredPriceTheirPlaqueIsChosenFrom(t *testing.T) {
	f, s := shopRoom(t, nil)
	stock := []ShopItem{
		{Code: data.QuestDocumentCode, Price: -1, Count: 1},
		{Code: gridWeapon, Price: 10, Count: 1},
		{Code: gridWeapon, Price: 100, Count: 1},
		{Code: gridWeapon, Price: 1250, Count: 1},
	}
	stored := []int32{0, 10, 100, 1250}
	half := []int32{0, 5, 50, 625}
	check := func(state string, cells []ui.ShopCell, mine bool, figures []int32) {
		t.Helper()
		for i, c := range cells {
			if !c.Occupied() || c.Money || c.Mine != mine || c.Price != figures[i] || c.PlaquePrice != stored[i] {
				t.Errorf("%s cell %d: occupied %v money %v mine %v prints %d, plaque chosen from %d; want an item cell, mine %v, printing %d from %d",
					state, i, c.Occupied(), c.Money, c.Mine, c.Price, c.PlaquePrice, mine, figures[i], stored[i])
			}
		}
	}

	f.Shop.shelves[ShelfArmour] = append([]ShopItem(nil), stock...)
	click(s, ui.ShopControlShelfPick, roomArmour)
	view := s.ShopScreen()
	check("shelf", view.Shelf[:len(stock)], false, stored)

	for range stock {
		if !f.Shop.TakeFromShelf(ShelfArmour, 0, 1) {
			t.Fatal("could not stage a merchant's place")
		}
	}
	view = s.ShopScreen()
	check("merchant's table", view.Table[:len(stock)], false, stored)

	f.Shop.ClearTable()
	for _, item := range stock {
		if !f.Shop.PutOnTable(item) {
			t.Fatal("could not stage a player's place")
		}
	}
	view = s.ShopScreen()
	check("player's table", view.Table[:len(stock)], true, half)

	pack := make([]sim.ItemInstance, len(stock))
	for i, item := range stock {
		pack[i] = sim.ItemInstance{Code: uint16(item.Code), Price: item.Price}
	}
	for len(pack) < 5 {
		pack = append(pack, sim.ItemInstance{Code: uint16(0x0f00 + len(pack))})
	}
	f.Carried[0].Carry.ItemInstances = pack
	s.packBase = 1
	view = s.ShopScreen()
	check("pack", view.Pack[:len(stock)], true, half)
}
