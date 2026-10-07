package game

import (
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type shopBookSpellRow struct {
	name   string
	school int32
	price  int32
	short  bool
}

type shopBookSpells []shopBookSpellRow

func (c shopBookSpells) Len() int                  { return len(c) }
func (c shopBookSpells) EntryName(i int) string    { return c[i].name }
func (c shopBookSpells) EntryStrings(int) []string { return nil }
func (c shopBookSpells) EntryParams(i int) []int32 {
	n := 22
	if c[i].short {
		n = 21
	}
	p := make([]int32, n)
	p[2] = c[i].school
	if len(p) > 21 {
		p[21] = c[i].price
	}
	return p
}

func shopBookFixtureTable() *mapload.Table {
	spells := make(shopBookSpells, 33)
	spells[10] = shopBookSpellRow{name: "Fire Arrow", school: 1, price: 101}
	spells[2] = shopBookSpellRow{name: "Water Bolt", school: 2, price: 202}
	spells[3] = shopBookSpellRow{name: "Air Shield", school: 3, price: 303}
	spells[4] = shopBookSpellRow{name: "Earth Wall", school: 4, price: 404}
	spells[5] = shopBookSpellRow{name: "Astral Gate", school: 5, price: 505}
	spells[7] = shopBookSpellRow{name: "Second Fire", school: 1, price: 101}
	spells[11] = shopBookSpellRow{name: "No Book", school: 1, price: 0}
	spells[8] = shopBookSpellRow{name: "Too Dear", school: 1, price: 1001}
	spells[9] = shopBookSpellRow{name: "Short Row", school: 1, price: 99, short: true}
	spells[32] = shopBookSpellRow{name: "Outside Mask", school: 1, price: 99}
	names := data.ItemNames{}
	for _, code := range shopBookCodeBySchool[1:] {
		names[code] = "Книга"
	}
	return &mapload.Table{Spells: spells, Names: names}
}

func TestBookPoolUsesInstalledCostAndNonMonotonicSchoolCodes(t *testing.T) {
	table := shopBookFixtureTable()
	pool := shopBookPool(table, 1000)
	want := map[uint16]data.ItemCode{
		10: 0x0e15,
		2:  0x0e14,
		3:  0x0e13,
		4:  0x0e16,
		5:  0x0e17,
		7:  0x0e15,
	}
	if len(pool) != len(want) {
		t.Fatalf("book pool holds %d entries, want %d: %+v", len(pool), len(want), pool)
	}
	for _, item := range pool {
		spell, ok := item.Instance().BookSpell()
		if !ok {
			t.Fatalf("generated item is not a readable book: %+v", item)
		}
		if item.Code != want[spell] {
			t.Errorf("spell %d code = %#04x, want %#04x", spell, item.Code, want[spell])
		}
		price, _ := mapload.SpellBookCost(spell, table)
		if item.Kind != 5 || len(item.Effects) != 1 || item.Effects[0].Mode != 0 || item.Price != price {
			t.Errorf("spell %d book = %+v", spell, item)
		}
	}
}

func TestBookShelfGuaranteesEligibleSpellsAndKeepsCompleteIdentity(t *testing.T) {
	table := shopBookFixtureTable()
	shop := NewShop(1000)
	shop.Generate(table, 1053)
	books := shop.Shelf(ShelfBooks)
	units := 0
	for _, item := range books {
		units += int(item.Count)
	}
	if units < 6 || units > 14 {
		t.Fatalf("generated book shelf holds %d units, want guaranteed6 plus at most8 random", units)
	}

	pool := shopBookPool(table, 1000)
	first, second := pool[4], pool[5]
	if first.Code != second.Code || first.Price != second.Price {
		t.Fatal("fixture books do not share the school code and price")
	}
	folded := shopStackShelf([]ShopItem{first, second, first})
	if len(folded) != 2 {
		t.Fatalf("two spells sharing a school folded to %+v", folded)
	}
	for _, item := range folded {
		spell, _ := item.Instance().BookSpell()
		wantCount := int32(1)
		if spell == 7 {
			wantCount = 2
		}
		if item.Count != wantCount {
			t.Errorf("spell %d count = %d, want %d", spell, item.Count, wantCount)
		}
	}
}

func TestBookShelfGuaranteeRespectsInclusiveCeilingOnEveryRestock(t *testing.T) {
	for _, ceiling := range []int32{0, 100, 101, 404, 1000} {
		shop := NewShop(ceiling)
		for seed := int64(0); seed < 16; seed++ {
			shop.Generate(shopBookFixtureTable(), seed)
			seen := map[uint16]int32{}
			for _, item := range shop.Shelf(ShelfBooks) {
				spell, ok := item.Instance().BookSpell()
				if !ok || item.Price <= 0 || item.Price > ceiling {
					t.Fatalf("ceiling%d seed%d: invalid book %+v", ceiling, seed, item)
				}
				seen[spell] += item.Count
			}
			for spell, price := range map[uint16]int32{2: 202, 3: 303, 4: 404, 5: 505, 7: 101, 10: 101} {
				if (seen[spell] > 0) != (price <= ceiling) {
					t.Fatalf("ceiling%d seed%d: spell%d count%d at price%d", ceiling, seed, spell, seen[spell], price)
				}
			}
			for _, spell := range []uint16{8, 9, 11, 32} {
				if seen[spell] != 0 {
					t.Fatalf("ineligible spell%d entered stock", spell)
				}
			}
		}
	}
}

func TestSoldUnreadBooksReturnToFourthShelfWithoutLosingSpellIdentity(t *testing.T) {
	table := shopBookFixtureTable()
	pool := shopBookPool(table, 1000)
	shop := NewShop(1000)
	for _, item := range []ShopItem{pool[4], pool[5], pool[4]} {
		if !shop.PutOnTable(item) {
			t.Fatal("PutOnTable refused a book")
		}
	}
	if _, ok := shop.Sell(); !ok {
		t.Fatal("Sell refused priced unread books")
	}
	books := shop.Shelf(ShelfBooks)
	if len(books) != 2 || len(shop.Shelf(ShelfMagic)) != 0 {
		t.Fatalf("sold books: fourth=%+v magic=%+v", books, shop.Shelf(ShelfMagic))
	}
	counts := map[uint16]int32{}
	for _, item := range books {
		spell, readable := item.Instance().BookSpell()
		if !readable {
			t.Fatalf("returned book lost identity: %+v", item)
		}
		counts[spell] = item.Count
	}
	if counts[7] != 2 || counts[10] != 1 {
		t.Fatalf("returned book counts = %v, want spell 7 x2 and spell 10 x1", counts)
	}
}

func TestFourthSelectorUsesInstalledGenericBookNameAndShowsBooks(t *testing.T) {
	f, screen := shopRoom(t, nil)
	f.Table = shopBookFixtureTable()
	f.Shop = NewShop(1000)
	f.Shop.Generate(f.Table, 1053)

	action := click(screen, ui.ShopControlShelfPick, roomBooks)
	view := screen.ShopScreen()
	if view.Chosen != roomBooks || view.ShelfName != "Книга" || action.Msg != "" {
		t.Fatalf("fourth selector: chosen=%d name=%q message=%q", view.Chosen, view.ShelfName, action.Msg)
	}
	found := false
	items := f.Shop.Shelf(ShelfBooks)
	for i, cell := range view.Shelf {
		if !cell.Occupied() {
			continue
		}
		found = true
		spell, _ := items[i].Instance().BookSpell()
		spellName := f.Table.Spells.EntryName(int(spell))
		if len(cell.Info) != 1 || cell.Info[0] != itemName(items[i].Code, f.Table)+" "+ui.AuthoredWords().ItemSpellOf+" "+spellName {
			t.Fatalf("book cell info = %q, want the book's name alone and no price line", cell.Info)
		}
	}
	if !found {
		t.Fatal("fourth selector shows no generated book")
	}
}

func TestShopEquipGestureReadsOneBookIntoTheSelectedMage(t *testing.T) {
	book := shopBookPool(shopBookFixtureTable(), 1000)[0].Instance()
	spell, readable := book.BookSpell()
	if !readable {
		t.Fatal("fixture is not a readable book")
	}

	t.Run("pack and duplicate", func(t *testing.T) {
		f, screen := shopRoom(t, nil)
		f.Table = shopBookFixtureTable()
		f.Carried[0].Mage = true
		f.Carried[0].Carry.ItemInstances = []sim.ItemInstance{book, book}
		f.Carried[0].Carry.Items = []uint16{book.Code, book.Code}
		for remaining := 1; remaining >= 0; remaining-- {
			if action := screen.shopEquipFromPack(1); action.Msg != "learned" {
				t.Fatalf("pack read = %+v, want learned", action)
			}
			if got := f.Carried[0].KnownSpells; got != uint32(1)<<spell {
				t.Fatalf("KnownSpells = %#x, want only spell %d", got, spell)
			}
			if got := len(mapload.MemberCarriedItems(f.Carried[0], f.Table)); got != remaining {
				t.Fatalf("pack holds %d book(s), want %d", got, remaining)
			}
		}
	})

	t.Run("fighter refusal is mutation free", func(t *testing.T) {
		f, screen := shopRoom(t, nil)
		f.Table = shopBookFixtureTable()
		f.Carried[0].Carry.ItemInstances = []sim.ItemInstance{book}
		f.Carried[0].Carry.Items = []uint16{book.Code}
		if action := screen.shopEquipFromPack(1); action.Msg != "only a mage can read that" {
			t.Fatalf("fighter read = %+v", action)
		}
		if f.Carried[0].KnownSpells != 0 || len(mapload.MemberCarriedItems(f.Carried[0], f.Table)) != 1 {
			t.Fatalf("fighter refusal mutated member: %+v", f.Carried[0])
		}
	})

	t.Run("direct shelf purchase", func(t *testing.T) {
		f, screen := shopRoom(t, nil)
		f.Table = shopBookFixtureTable()
		f.Shop = NewShop(1000)
		f.Shop.Generate(f.Table, 1053)
		f.Carried[0].Mage = true
		f.Town.gold = 2000
		screen.shopChosen = roomBooks
		item := f.Shop.Shelf(ShelfBooks)[0]
		spell, _ := item.Instance().BookSpell()
		beforeUnits, beforeGold := shelfUnits(f.Shop, ShelfBooks), f.Town.Gold()
		if action := screen.shopEquipFromShelf(0); !strings.Contains(action.Msg, "bought and learned") {
			t.Fatalf("shelf read = %+v", action)
		}
		if f.Carried[0].KnownSpells != uint32(1)<<spell ||
			shelfUnits(f.Shop, ShelfBooks) != beforeUnits-1 || f.Town.Gold() != beforeGold-int(item.Price) {
			t.Fatalf("shelf result: spells=%#x units=%d gold=%d",
				f.Carried[0].KnownSpells, shelfUnits(f.Shop, ShelfBooks), f.Town.Gold())
		}
	})

	t.Run("merchant table purchase", func(t *testing.T) {
		f, screen := shopRoom(t, nil)
		f.Table = shopBookFixtureTable()
		f.Shop = NewShop(1000)
		f.Shop.Generate(f.Table, 1053)
		f.Carried[0].Mage = true
		f.Town.gold = 2000
		item := f.Shop.Shelf(ShelfBooks)[0]
		spell, _ := item.Instance().BookSpell()
		if !f.Shop.TakeFromShelf(ShelfBooks, 0, 1) {
			t.Fatal("TakeFromShelf refused book")
		}
		beforeGold := f.Town.Gold()
		if action := screen.shopEquipFromTable(0); !strings.Contains(action.Msg, "bought and learned") {
			t.Fatalf("table read = %+v", action)
		}
		if f.Carried[0].KnownSpells != uint32(1)<<spell || len(f.Shop.Table()) != 0 ||
			f.Town.Gold() != beforeGold-int(item.Price) {
			t.Fatalf("table result: spells=%#x places=%d gold=%d",
				f.Carried[0].KnownSpells, len(f.Shop.Table()), f.Town.Gold())
		}
	})
}
