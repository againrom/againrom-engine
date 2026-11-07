package game

import (
	"testing"

	"againrom/pkg/secondshop"
	"againrom/pkg/ui"
)

// shelfShown lists the counts of the drawn shelf's occupied cells and the
// cell index of each. An occupied cell is the one a hover, a click or a drag
// starts from.
func shelfShown(s *townScreen) (counts []uint32, cells []int) {
	for i, c := range s.ShopScreen().Shelf {
		if c.Occupied() {
			counts = append(counts, c.Count)
			cells = append(cells, i)
		}
	}
	return counts, cells
}

// cellHolding is the drawn shelf cell that shows count units, -1 for none.
func cellHolding(s *townScreen, count uint32) int {
	for i, c := range s.ShopScreen().Shelf {
		if c.Occupied() && c.Count == count {
			return i
		}
	}
	return -1
}

func shelfStock(c *secondCampaign, shelf ShopShelf) (lots int) {
	for _, it := range c.shop.Shelf(shelf) {
		if it.Count > 0 {
			lots++
		}
	}
	return lots
}

// A lot taken from a shelf of the second game's shop leaves the shelf at once
// in the picture and the cells a click or drag starts from, a unit of a stack
// leaves it one smaller, a lot put back returns, and after a purchase the
// shelf shows what the stock holds and every shown cell can be taken.
func TestSecondShopTakenLotLeavesNoGhost(t *testing.T) {
	c := secondShopSample()
	c.shop.shelves[3] = secondShopItems(secondShopTestTables, []secondshop.Item{
		{Table: secondshop.Book, Spell: 1, Count: 1, Price: 1000},
		{Table: secondshop.Book, Spell: 2, Count: 3, Price: 500},
	})
	s := secondShopPage(c, 100000)
	s.click(ui.ShopControlShelfPick, 3)
	if counts, _ := shelfShown(s); len(counts) != 2 {
		t.Fatalf("shelf shows %v, want two lots", counts)
	}

	// The one-unit lot is carried: the shelf holds only the stack.
	s.click(ui.ShopControlShelfCell, cellHolding(s, 1))
	counts, _ := shelfShown(s)
	if len(counts) != 1 || counts[0] != 3 {
		t.Fatalf("one-unit lot taken: shelf shows %v", counts)
	}
	// Dropping it back puts it back.
	s.click(ui.ShopControlTableCell, 0)
	if counts, _ = shelfShown(s); len(counts) != 2 {
		t.Fatalf("lot put back: shelf shows %v", counts)
	}

	// A unit of the stack leaves it one smaller.
	s.click(ui.ShopControlShelfCell, cellHolding(s, 3))
	if counts, _ = shelfShown(s); len(counts) != 2 || counts[0]+counts[1] != 3 {
		t.Fatalf("stack unit taken: shelf shows %v", counts)
	}

	// Buy what is carried; the shelf shows what the stock holds.
	s.click(ui.ShopControlButton, 1)
	counts, cells := shelfShown(s)
	if len(counts) != shelfStock(c, 3) {
		t.Fatalf("after the purchase the shelf shows %v, the stock holds %d lots", counts, shelfStock(c, 3))
	}
	for len(cells) > 0 {
		before := len(c.shop.Table())
		s.click(ui.ShopControlShelfCell, cells[0])
		if len(c.shop.Table()) == before {
			t.Fatalf("shown cell %d cannot be taken: shelf shows %v", cells[0], counts)
		}
		s.click(ui.ShopControlButton, 1)
		counts, cells = shelfShown(s)
		if len(counts) != shelfStock(c, 3) {
			t.Fatalf("after the purchase the shelf shows %v, the stock holds %d lots", counts, shelfStock(c, 3))
		}
	}
	if len(counts) != 0 {
		t.Fatalf("shelf still shows %v", counts)
	}
}
