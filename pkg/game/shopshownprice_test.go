package game

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/itempkt"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type shownPriceRows struct{ secondShopTestCollection }

func (shownPriceRows) EntryName(int) string      { return "" }
func (shownPriceRows) EntryStrings(int) []string { return nil }

func TestShelfCellUsesRecordPriceWithoutChangingTradeValue(t *testing.T) {
	for _, effects := range [][]sim.ItemEffect{nil, {{Kind: 5, Operand: 1}}} {
		it := secondshop.Item{Table: secondshop.Armor, Row: 3, Material: 1, Count: 1, Price: 50, Effects: effects}
		table := secondShopCellTable()
		table.Armors = shownPriceRows{secondShopTestCollection{3: {0, 0, 0, 0, 6}}}
		table.ItemRecords = map[uint16]itempkt.Record{0x1603: {Flags: 2, Price: 51, HasPrice: true}}
		s, view := secondShopCellPage(t, table, 0, []secondshop.Item{it})
		want := int32(51)
		if len(effects) > 0 {
			want = 50
		}
		if cell := view.Shelf[0]; cell.Price != want || cell.PlaquePrice != want {
			t.Fatalf("cell price %d plaque %d want %d", cell.Price, cell.PlaquePrice, want)
		}
		if got := s.state().shop.shelves[0][0].Price; got != 50 {
			t.Fatalf("trade price changed to %d", got)
		}
	}
}

func TestShownShelfClickAndUndoKeepServerOrder(t *testing.T) {
	c := secondShopSample()
	c.shop.records[1][0].Mask |= 1 << 29
	var server []secondshop.Item
	for i := 1; i <= 3; i++ {
		server = append(server, secondshop.Item{Table: secondshop.Armor, Row: 3, Material: 1, Count: 1, Price: 50,
			Effects: []sim.ItemEffect{{Kind: 5, Operand: uint32(i)}}})
	}
	c.shop.shelves[0] = secondShopItems(secondShopTestTables, server)
	s := secondShopPage(c, 1000)
	s.click(ui.ShopControlShelfPick, 0)
	if got := c.shop.shelf()[0].Effects[0].Operand; got != 2 {
		t.Fatalf("first shown effect %d, want 2", got)
	}
	before := c.shop.stock()
	s.click(ui.ShopControlShelfCell, 0)
	if got := c.shop.buying(); len(got) != 1 || got[0].Effects[0].Operand != 2 {
		t.Fatalf("clicked item %v", got)
	}
	if c.shop.shelves[0][0].Count != 1 || c.shop.shelves[0][1].Count != 0 {
		t.Fatal("removed the wrong server entry")
	}
	s.click(ui.ShopControlButton, 0)
	if !reflect.DeepEqual(before, c.shop.stock()) {
		t.Fatal("Undo changed server order or quantities")
	}
	clone := cloneShopMutation(&c.shop.Shop)
	if !reflect.DeepEqual(clone.shelves[0], c.shop.shelves[0]) {
		t.Fatal("mutation clone changed server order")
	}
}
