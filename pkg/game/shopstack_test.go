package game

import (
	"testing"

	"againrom/pkg/ui"
)

// The shelf stacking rule (owner, `DIV-322`): a shelf shows one cell per
// distinct item, carrying how many of it the merchant has.
//
// These are the tests that fail when the rule is removed. The rest of the
// shop's suite was adjusted to survive it, which is a different thing: a test
// that counts units instead of elements passes with or without the fold.

// TestGeneratedShelfFoldsItsRepeats is the rule at the generator: equal code
// and price arrive as one element carrying the count, and nothing is lost.
func TestGeneratedShelfFoldsItsRepeats(t *testing.T) {
	s := NewShop(1000)
	s.Generate(shopTable(), 5)

	for shelf := ShopShelf(0); shelf < numShopShelves; shelf++ {
		items := s.Shelf(shelf)
		if len(items) == 0 {
			continue
		}
		seen := map[[2]int32]int{}
		for _, item := range items {
			k := [2]int32{int32(item.Code), item.Price}
			seen[k]++
			if seen[k] > 1 {
				t.Errorf("shelf %v shows code %#x at price %d in %d separate cells, want one",
					shelf, item.Code, item.Price, seen[k])
			}
			if item.Count < 1 {
				t.Errorf("shelf %v holds a cell of %d units: %+v", shelf, item.Count, item)
			}
		}
	}

	// The fold really happened on the shelf this fixture repeats on, so a
	// build that never folded could not pass the loop above by drawing
	// distinct items only.
	var stacked int
	for _, item := range s.Shelf(ShelfWeapons) {
		if item.Count > 1 {
			stacked++
		}
	}
	if stacked == 0 {
		t.Error("no weapon cell holds more than one unit, so nothing was folded")
	}
}

// TestShopStackShelfKeepsEveryUnitAndTheFirstPosition is the fold itself,
// away from the generator: the unit total is preserved, the first occurrence
// keeps its place, and code and price must BOTH agree to merge.
func TestShopStackShelfKeepsEveryUnitAndTheFirstPosition(t *testing.T) {
	in := []ShopItem{
		{Code: 0x0101, Price: 40, Count: 1},
		{Code: 0x0202, Price: 90, Count: 2},
		{Code: 0x0101, Price: 40, Count: 3},
		{Code: 0x0101, Price: 41, Count: 1}, // same code, other price
		{Code: 0x0202, Price: 90, Count: 1},
	}
	out := shopStackShelf(in)

	if len(out) != 3 {
		t.Fatalf("stacking gave %d cells: %+v, want three", len(out), out)
	}
	if out[0].Code != 0x0101 || out[0].Price != 40 || out[0].Count != 4 {
		t.Errorf("cell 0 = %+v, want the first code folded to four units at its own price", out[0])
	}
	if out[1].Code != 0x0202 || out[1].Count != 3 {
		t.Errorf("cell 1 = %+v, want three units", out[1])
	}
	if out[2].Price != 41 || out[2].Count != 1 {
		t.Errorf("cell 2 = %+v, want the odd-priced lot kept apart", out[2])
	}

	var before, after int32
	for _, item := range in {
		before += item.Count
	}
	for _, item := range out {
		after += item.Count
	}
	if before != after {
		t.Errorf("stacking turned %d units into %d", before, after)
	}
}

// TestTheRoomJoinsASecondUnitToTheSamePlace is shopFindHisPlace's own arm:
// two default clicks on one shelf cell make ONE table place of two units,
// which is what keeps five clicks on a stack of six from filling the table.
func TestTheRoomJoinsASecondUnitToTheSamePlace(t *testing.T) {
	f, s := shopRoom(t, nil)
	f.Shop.shelves[ShelfArmour] = []ShopItem{{Code: 0x0201, Price: 30, Count: 6}}
	click(s, ui.ShopControlShelfPick, roomArmour)

	for i := 0; i < 6; i++ {
		if act := click(s, ui.ShopControlShelfCell, 0); act.Msg != "" {
			t.Fatalf("click %d said %q, want the table to take it", i+1, act.Msg)
		}
	}

	places := f.Shop.Table()
	if len(places) != 1 {
		t.Fatalf("six clicks opened %d places: %+v, want one joined place", len(places), places)
	}
	if places[0].Count != 6 || places[0].Mine || places[0].From != ShelfArmour {
		t.Errorf("place = %+v, want the merchant's six units off the armour shelf", places[0])
	}
	if got := shelfUnits(f.Shop, ShelfArmour); got != 0 {
		t.Errorf("the shelf holds %d units, want none — all six moved", got)
	}
}

// TestADefaultClickTakesOneUnitAndShiftTakesTheStack is the shift+click
// convention reaching the shelf (`DIV-047`, extended by `DIV-322`).
func TestADefaultClickTakesOneUnitAndShiftTakesTheStack(t *testing.T) {
	f, s := shopRoom(t, nil)
	f.Shop.shelves[ShelfArmour] = []ShopItem{{Code: 0x0201, Price: 30, Count: 4}}
	click(s, ui.ShopControlShelfPick, roomArmour)

	click(s, ui.ShopControlShelfCell, 0)
	if got := f.Shop.Table()[0].Count; got != 1 {
		t.Errorf("a default click moved %d units, want one", got)
	}
	if got := shelfUnits(f.Shop, ShelfArmour); got != 3 {
		t.Errorf("the shelf holds %d units after one default click, want three", got)
	}

	clickShift(s, ui.ShopControlShelfCell, 0)
	if got := f.Shop.Table()[0].Count; got != 4 {
		t.Errorf("shift+click left the place at %d units, want all four joined", got)
	}
	if got := len(f.Shop.Shelf(ShelfArmour)); got != 0 {
		t.Errorf("the shelf still shows %d cells, want none", got)
	}
}

// TestAUnitPutBackJoinsTheCellItCameFrom is returnToShelf's merge: a unit off
// a stack and back again leaves the shelf exactly as it was, in one cell and
// not in two. SHOP-DUP-028 establishes the original's return path merges too.
func TestAUnitPutBackJoinsTheCellItCameFrom(t *testing.T) {
	f, s := shopRoom(t, nil)
	f.Shop.shelves[ShelfArmour] = []ShopItem{{Code: 0x0201, Price: 30, Count: 5}}
	click(s, ui.ShopControlShelfPick, roomArmour)

	click(s, ui.ShopControlShelfCell, 0)
	click(s, ui.ShopControlButton, 0) // clear the table

	items := f.Shop.Shelf(ShelfArmour)
	if len(items) != 1 {
		t.Fatalf("the shelf shows %d cells after a round trip: %+v, want one", len(items), items)
	}
	if items[0].Count != 5 {
		t.Errorf("the cell holds %d units, want the five it started with", items[0].Count)
	}
}

// TestTheShelfCellShowsItsCount is the whole point reaching the screen: the
// composed cell carries the element's own count, so the renderer draws the
// number it already draws for a table place and a pack stack.
func TestTheShelfCellShowsItsCount(t *testing.T) {
	f, s := shopRoom(t, nil)
	f.Shop.shelves[ShelfArmour] = []ShopItem{
		{Code: 0x0201, Price: 30, Count: 7},
		{Code: 0x0202, Price: 20, Count: 1},
	}
	click(s, ui.ShopControlShelfPick, roomArmour)

	cells := s.ShopScreen().Shelf
	if got := cells[0].Count; got != 7 {
		t.Errorf("the first shelf cell shows count %d, want 7", got)
	}
	if got := cells[1].Count; got != 1 {
		t.Errorf("the second shelf cell shows count %d, want 1", got)
	}
}
