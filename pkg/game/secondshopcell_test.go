package game

import (
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/text"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondShopCellTable is a second-game table with spell 1, Fire Arrow, of the
// Fire school, its school's book template in MagicItems row 1 and its scroll
// in row 6, whose effect string casts it; the shipped names name both rows.
func secondShopCellTable() *mapload.Table {
	spells := make(dbCollection, 30)
	for id := range spells {
		spells[id] = dbEntry{params: make([]int32, 22)}
	}
	spells[1].name = "Fire Arrow"
	spells[1].params[2] = 1
	items := make(dbCollection, 40)
	items[1] = dbEntry{name: "Book_Fire", params: []int32{1000}}
	items[6] = dbEntry{name: "Scroll_Fire_Arrow", params: []int32{100}, strings: []string{"castSpell=Fire_Arrow:50"}}
	return &mapload.Table{Spells: spells, MagicItems: items,
		Names: data.ItemNames{0x0e01: "Book", 0x0e06: "Scroll", 0x6603: "Cloak"}}
}

// secondShopCellPage is the shop page of a campaign whose shelf k holds items,
// in an install with table, its rack k chosen.
func secondShopCellPage(t *testing.T, table *mapload.Table, k int, items []secondshop.Item) (*townScreen, ui.ShopScreenView) {
	t.Helper()
	c := secondShopSample()
	c.shop.shelves[k] = secondShopItems(secondShopTestTables, items)
	if len(c.shop.shelves[k]) != len(items) {
		t.Fatalf("shelf %d holds %d of %d items", k, len(c.shop.shelves[k]), len(items))
	}
	s := secondShopPage(c, 100000)
	s.in.Table = table
	s.click(ui.ShopControlShelfPick, k)
	return s, s.ShopScreen()
}

// A book's cell caption is its school's book name and its spell's, and a
// scroll's names the spell its effect string casts (R2-ENGINE-711,
// R2-ENGINE-712).
func TestSecondShopCellCaptionNamesTheSpell(t *testing.T) {
	_, v := secondShopCellPage(t, secondShopCellTable(), 3, []secondshop.Item{
		{Table: secondshop.Book, Spell: 1, Count: 2, Price: 1000},
		{Table: secondshop.Magic, Row: 6, Count: 1, Price: 100},
	})
	if got := v.Shelf[0].Info; len(got) == 0 || got[0] != "Book of Fire Arrow" {
		t.Errorf("book caption %q, want the first line %q", got, "Book of Fire Arrow")
	}
	if got := v.Shelf[1].Info; len(got) == 0 || got[0] != "Scroll of Fire Arrow" {
		t.Errorf("scroll caption %q, want the first line %q", got, "Scroll of Fire Arrow")
	}
}

// An enchanted cloak's caption lists the attribute it raises and casts
// nothing: clothing carries no castSpell (R2-ENGINE-711).
func TestSecondShopClothingCaptionListsItsAttribute(t *testing.T) {
	_, v := secondShopCellPage(t, secondShopCellTable(), 0, []secondshop.Item{
		{Table: secondshop.Armor, Row: 3, Class: 0, Material: 6, Count: 1, Price: 900,
			Effects: []sim.ItemEffect{{Kind: 5, Operand: 3}}},
	})
	got := v.Shelf[0].Info
	words := itemWords(nil)
	want := itemEffectInfoLine(sim.ItemEffect{Kind: 5, Operand: 3}, nil)
	if !slices.Contains(got, want) {
		t.Fatalf("cloak caption %q, want the attribute line %q", got, want)
	}
	for _, line := range got {
		if strings.Contains(line, words.ItemCasts) {
			t.Fatalf("cloak caption %q casts", got)
		}
	}
}

// The shop cell's price draws in the generation screen's text font, font2,
// not the install's tip font (R2-ENGINE-713); its plaque is chosen from the
// stored price's digit count.
func TestSecondShopPriceDrawsInTheTextFont(t *testing.T) {
	textFont, tipFont := &text.Font{}, &text.Font{}
	c := secondShopSample()
	s := secondShopPage(c, 1000)
	s.in.ChargenAssets = &ChargenAssets{Presentation: &ui.ChargenPresentation{Font: textFont, TipFont: tipFont}}
	s.click(ui.ShopControlShelfPick, 0)
	v := s.ShopScreen()
	if v.PriceFont != textFont {
		t.Fatalf("price font is not the text font (tip font: %v)", v.PriceFont == tipFont)
	}
	if cell := v.Shelf[0]; cell.Price != 125 || cell.PlaquePrice != 125 {
		t.Fatalf("shelf cell price %d plaque price %d, want 125", cell.Price, cell.PlaquePrice)
	}
}
