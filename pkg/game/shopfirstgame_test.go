package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/ui"
)

// The first game's shop page is declared as data: the pack strip has its own
// picture, the gold pile follows the last stack and the shelf gems follow the
// grid.
func TestFirstGameShopPageRulesAreDeclared(t *testing.T) {
	_, s := shopRoom(t, nil)
	room := s.townDescription().Room("shop")
	if room == nil || room.Shop == nil || !room.Shop.MoneyLast || !room.Shop.ShelfGems {
		t.Fatalf("shop page rules = %+v, want money last and shelf gems", room)
	}
	found := false
	for _, a := range room.Art {
		found = found || a.Name == "pack"
	}
	if !found {
		t.Fatal("the shop page declares no picture for the pack strip, so an empty place shows no plate")
	}
}

// The gold pile is the last item of the strip: five stacks fill the strip and
// the gold waits behind them instead of pushing the fifth stack off.
func TestPackStripShowsEveryStackBeforeTheGold(t *testing.T) {
	_, s := shopRoom(t, []uint16{0x0101, 0x0122, 0x0155, 0x0102, 0x0103})
	v := s.ShopScreen()
	for i, c := range v.Pack {
		if c.Money || !c.Occupied() {
			t.Fatalf("pack place %d = %+v, want a stack", i, c)
		}
	}
	s.ShopScroll(ui.ShopWheelPack, 1)
	if v = s.ShopScreen(); !v.Pack[4].Money || v.Pack[4].Count != uint32(s.sess.Town.Gold()) {
		t.Fatalf("after one turn the last place is %+v, want the gold pile", v.Pack[4])
	}
}

// The shelf's gems: the top one is dark on the first page and the bottom one
// lit while a further row waits; they swap at the end.
func TestShelfGemsFollowTheGridsScrollPosition(t *testing.T) {
	f, s := shopRoom(t, nil)
	stock := make([]ShopItem, 0, 8)
	for i := 0; i < 8; i++ {
		stock = append(stock, ShopItem{Code: data.ItemCode(0x0201 + i), Price: int32(90 - 10*i), Count: 1})
	}
	f.Shop.shelves[ShelfArmour] = stock
	click(s, ui.ShopControlShelfPick, roomArmour)
	if v := s.ShopScreen(); !v.ShelfGems || v.ShelfBack || !v.ShelfForward {
		t.Fatalf("first page: gems %v back %v forward %v, want dark top and lit bottom", v.ShelfGems, v.ShelfBack, v.ShelfForward)
	}
	s.ShopScroll(ui.ShopWheelShelf, 3)
	if v := s.ShopScreen(); !v.ShelfBack || v.ShelfForward {
		t.Fatalf("last page: back %v forward %v, want lit top and dark bottom", v.ShelfBack, v.ShelfForward)
	}
}

// While the merchant's dialogue stands over the shop the shelf grid shows no
// stock, as the original's does.
func TestShelfShowsNoStockUnderTheMerchantDialogue(t *testing.T) {
	_, s := shopRoom(t, nil)
	click(s, ui.ShopControlShelfPick, roomArmour)
	if !s.ShopScreen().Shelf[0].Occupied() {
		t.Fatal("the opened shelf shows no stock without a dialogue")
	}
	s.room, s.dialogueBuilding = roomTalk, TownShop
	for i, c := range s.ShopScreen().Shelf {
		if c.Occupied() {
			t.Fatalf("shelf place %d shows stock under the dialogue", i)
		}
	}
}
