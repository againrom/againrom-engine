package game

import (
	"bytes"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// bookIconProof is what the continuous run hands the cold process: the book
// the hero bought and the spell it teaches.
type bookIconProof struct {
	Code  uint16
	Spell uint32
}

// schoolBookIcon is the picture a spell's book shows: the school's own Book
// template read through the inventory's loader, independent of the shop.
func schoolBookIcon(t *testing.T, f *FrontEnd, spell int) *image.RGBA {
	t.Helper()
	params := f.Table.Spells.EntryParams(spell)
	if len(params) <= 2 {
		t.Fatalf("spell %d has no school", spell)
	}
	code := shopSchoolTemplateCode(f.Table, "Book", params[2])
	if code == 0 {
		t.Fatalf("spell %d: school %d has no Book template", spell, params[2])
	}
	icon, err := loadItemIcon(f.Archives.Containers, graphicsPrefix+data.ItemIconPath(code))
	if err != nil || icon == nil {
		t.Fatalf("spell %d: template %04x icon: %v", spell, uint16(code), err)
	}
	return icon
}

func sameIcon(a, b *image.RGBA) bool {
	return a != nil && b != nil && a.Rect == b.Rect && bytes.Equal(a.Pix, b.Pix)
}

// bookCell is the cell of the grid that shows the book, found by the lines
// the page prints for it, or nil.
func bookCell(cells []ui.ShopCell, s *townScreen, book sim.ItemInstance) *ui.ShopCell {
	want := s.label(book)
	for i := range cells {
		if cells[i].Occupied() && slices.Equal(cells[i].Text, want) {
			return &cells[i]
		}
	}
	return nil
}

func secondShopBookInstance(t *testing.T, f *FrontEnd, spell int) sim.ItemInstance {
	t.Helper()
	tables, ok := secondShopTables(f.Table)
	if !ok {
		t.Fatal("no shop tables")
	}
	in, ok := tables.Instance(secondshop.Item{Table: secondshop.Book, Spell: spell, Count: 1})
	if !ok {
		t.Fatalf("no book for spell %d", spell)
	}
	return in
}

// TestReleaseSecondShopBookIcons shows the five books of town 1's opening
// stock on the shelf, on the table and in the pack with their school's
// picture, keeps two spells of one school apart by their teach effect, and has
// a separate process LOAD the SAV the purchase was saved in (DIV-3043).
func TestReleaseSecondShopBookIcons(t *testing.T) {
	secondGameRoot(t)
	if path := os.Getenv("AGAINROM_SECOND_BOOKICON_INPUT"); path != "" {
		secondBookIconCold(t, path)
		return
	}
	f, app := secondTownNew(t, false, false)
	secondShopActivate(t, f, app, "SHOP")
	c := f.Town.second
	s := f.TownScreen().(*townScreen)
	f.Town.gold = 100000

	category := -1
	var books []secondshop.Item
	for k, list := range c.shop.stock() {
		for _, it := range list {
			if it.Table == secondshop.Book {
				category = k
				books = append(books, it)
			}
		}
	}
	if len(books) != 5 {
		t.Fatalf("town 1 opens with %d books, want 5", len(books))
	}
	secondShopActivate(t, f, app, []string{"SHELF 1", "SHELF 2", "SHELF 3", "SHELF 4"}[category])

	// The shelf: every book of the opening stock has its school's picture.
	for _, it := range books {
		book := secondShopBookInstance(t, f, it.Spell)
		want := schoolBookIcon(t, f, it.Spell)
		if got := s.shopItemIcon(book); !sameIcon(got, want) {
			t.Errorf("spell %d code %04x: icon %v, want the school's", it.Spell, book.Code, got != nil)
		}
		entry := slices.IndexFunc(c.shop.shelf(), func(x secondshop.Item) bool { return x.Table == secondshop.Book && x.Spell == it.Spell })
		var cell *ui.ShopCell
		for top := 0; top <= entry && cell == nil; top++ {
			s.ShopScroll(ui.ShopWheelShelf, top)
			v := s.ShopScreen()
			cell = bookCell(v.Shelf[:], s, book)
		}
		if cell == nil || !sameIcon(cell.Icon, want) {
			t.Errorf("spell %d: shelf cell %v, want the school's picture", it.Spell, cell)
		}
	}

	// Two spells of one school: one picture, two identities.
	first := books[0]
	a := secondShopBookInstance(t, f, first.Spell)
	school := f.Table.Spells.EntryParams(first.Spell)[2]
	other := 0
	for spell := 1; spell < 32 && spell < f.Table.Spells.Len() && other == 0; spell++ {
		if params := f.Table.Spells.EntryParams(spell); spell != first.Spell && len(params) > 2 && params[2] == school {
			if _, ok := mapload.SpellBookCost(uint16(spell), f.Table); ok {
				other = spell
			}
		}
	}
	if other == 0 {
		t.Fatalf("no second spell in school %d", school)
	}
	b := secondShopBookInstance(t, f, other)
	if a.Code == b.Code || slices.Equal(a.Effects, b.Effects) || !sameIcon(s.shopItemIcon(a), s.shopItemIcon(b)) || s.shopItemIcon(a) == nil {
		t.Fatalf("spells %d and %d: codes %04x/%04x effects %v/%v, icons equal %v", first.Spell, other, a.Code, b.Code, a.Effects, b.Effects, sameIcon(s.shopItemIcon(a), s.shopItemIcon(b)))
	}

	// The table and the pack: buy the first book.
	visible := c.shop.visibleShelf()
	at := slices.Index(visible, slices.IndexFunc(c.shop.shelf(), func(it secondshop.Item) bool { return it.Table == secondshop.Book && it.Spell == first.Spell }))
	if at < 0 {
		t.Fatal("the first book is not on the shown shelf")
	}
	*c.heroBag() = nil
	s.ShopScroll(ui.ShopWheelShelf, visible[at]/2)
	secondShopActivate(t, f, app, "ITEM "+strconv.Itoa(at+1))
	v := s.ShopScreen()
	want := schoolBookIcon(t, f, first.Spell)
	if cell := bookCell(v.Table[:], s, a); cell == nil || !sameIcon(cell.Icon, want) {
		t.Fatalf("table cell for spell %d: %v", first.Spell, cell)
	}
	secondShopActivate(t, f, app, "BUY")
	bag := *c.heroBag()
	if len(bag) != 1 || bag[0].Code != a.Code || len(bag[0].Effects) != 1 || bag[0].Effects[0].Operand != uint32(first.Spell) {
		t.Fatalf("bag after the buy: %v", bag)
	}
	v = s.ShopScreen()
	if cell := bookCell(v.Pack[:], s, bag[0]); cell == nil || !sameIcon(cell.Icon, want) {
		t.Fatalf("pack cell for spell %d: %v", first.Spell, cell)
	}

	// SAVE and a cold LOAD in another process.
	dir := t.TempDir()
	secondTownNamedSave(t, f, app, dir, "Book")
	proof, err := json.Marshal(bookIconProof{Code: bag[0].Code, Spell: bag[0].Effects[0].Operand})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, secondSaveFile(t, dir, "Book"))
	if err := os.WriteFile(path+".json", proof, 0600); err != nil {
		t.Fatal(err)
	}
	runSpellWitnessChild(t, path, "AGAINROM_SECOND_BOOKICON_INPUT")
}

// secondBookIconCold is the separate process: the LOAD keeps the book's code
// and spell, and the page shows it with its school's picture.
func secondBookIconCold(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var proof bookIconProof
	if err := json.Unmarshal(raw, &proof); err != nil {
		t.Fatal(err)
	}
	f, _ := secondTownCold(t, filepath.Dir(path), filepath.Base(path))
	c := f.Town.second
	bag := *c.heroBag()
	if len(bag) != 1 || bag[0].Code != proof.Code || len(bag[0].Effects) != 1 || bag[0].Effects[0].Operand != proof.Spell {
		t.Fatalf("cold LOAD: bag %v, want code %04x spell %d", bag, proof.Code, proof.Spell)
	}
	s := f.TownScreen().(*townScreen)
	want := schoolBookIcon(t, f, int(proof.Spell))
	v := s.ShopScreen()
	if cell := bookCell(v.Pack[:], s, bag[0]); cell == nil || !sameIcon(cell.Icon, want) {
		t.Fatalf("cold LOAD: pack cell for spell %d: %v", proof.Spell, cell)
	}
}
