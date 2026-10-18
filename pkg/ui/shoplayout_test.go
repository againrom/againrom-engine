package ui

import (
	"image"
	"testing"

	"againrom/pkg/render/text"
)

// The first layout is the page's constants: a nil Layout and an explicit first
// layout compose the same pixels and answer the same controls.
func TestFirstShopLayoutIsTheConstants(t *testing.T) {
	l := FirstShopLayout()
	for i := 0; i < shopShelfN; i++ {
		if got, want := l.ShelfCellRect(i), image.Rect(1+80*(i%2), 31+80*(i/2), 81+80*(i%2), 111+80*(i/2)); got != want {
			t.Errorf("shelf cell %d = %v, want %v", i, got, want)
		}
	}
	for i := 0; i < shopStripN; i++ {
		if got, want := l.TableCellRect(i), image.Rect(32+80*i, 303, 112+80*i, 383); got != want {
			t.Errorf("table cell %d = %v, want %v", i, got, want)
		}
		if got, want := l.PackCellRect(i), image.Rect(32+80*i, 395, 112+80*i, 475); got != want {
			t.Errorf("pack cell %d = %v, want %v", i, got, want)
		}
	}
	cell := ShopCell{Back: ShopBackItem, Count: 2, Price: 30, PlaquePrice: 30}
	v := ShopScreenView{Font: shopLayoutTestFont(), Chosen: -1}
	v.Shelf[0], v.Table[1], v.Pack[2] = cell, cell, cell
	explicit := v
	explicit.Layout = &l
	a := ComposeShopScreen(v, image.Pt(40, 60), true, nil, false)
	b := ComposeShopScreen(explicit, image.Pt(40, 60), true, nil, false)
	if string(a.Pix) != string(b.Pix) {
		t.Fatal("the explicit first layout composes other pixels than the view's default")
	}
	for y := 0; y < shopScreenH; y += 3 {
		for x := 0; x < shopScreenW; x += 3 {
			p := image.Pt(x, y)
			c1, ok1 := ShopControlAt(p)
			c2, ok2 := shopScreenControlAt(explicit, p)
			if c1 != c2 || ok1 != ok2 {
				t.Fatalf("control at %v: %v %v, first layout %v %v", p, c1, ok1, c2, ok2)
			}
		}
	}
}

// A layout without the character panel and the book answers no picker, mode,
// doll or book control, draws nothing in the panel's region and prints a text
// cell's name where the item has no picture.
func TestShopLayoutPanelsAndTextCells(t *testing.T) {
	l := FirstShopLayout()
	l.Character, l.Book = false, false
	font := shopLayoutTestFont()
	v := ShopScreenView{Layout: &l, Font: font, Chosen: -1, Book: true,
		Figure: image.NewRGBA(image.Rect(0, 0, 160, 240)), MemberCount: 1}
	v.Shelf[0] = ShopCell{Back: ShopBackItem, Count: 1, Price: 7, PlaquePrice: 7}
	for _, p := range []image.Point{shopPickerPrevRect.Min, shopPickerNextRect.Min, shopCharacterModeRect.Min} {
		if c, ok := shopScreenControlAt(v, p); ok {
			t.Errorf("control %v at %v on a layout without the character panel", c, p)
		}
	}
	if shopPaneTakesItemAt(v, image.Pt(560, 400)) {
		t.Error("the absent character panel takes an item")
	}
	if v.bookOpen() {
		t.Error("the book opens on a layout without it")
	}
	if c, ok := shopGridControlAt(v, l.TableCellRect(0).Min.Add(image.Pt(5, 5))); !ok || c.Kind != ShopControlTableCell {
		t.Errorf("table cell under the absent book = %v %v", c, ok)
	}
	plain := ComposeShopScreen(v, image.Point{}, false, nil, false)
	for y := shopFigureRect.Min.Y; y < shopFigureRect.Max.Y; y++ {
		for x := shopFigureRect.Min.X; x < shopFigureRect.Max.X; x++ {
			if plain.RGBAAt(x, y) != shopScreenFill {
				t.Fatalf("pixel %d,%d of the absent character panel is drawn", x, y)
			}
		}
	}
	named := v
	named.Shelf[0].Text = []string{"A", "A"}
	withText := ComposeShopScreen(named, image.Point{}, false, nil, false)
	r := l.ShelfCellRect(0)
	differs := false
	for y := r.Min.Y + 20; y < r.Max.Y-16 && !differs; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if withText.RGBAAt(x, y) != plain.RGBAAt(x, y) {
				differs = true
				break
			}
		}
	}
	if !differs {
		t.Error("a text cell draws no name")
	}
	pictured := named
	pictured.Shelf[0].Icon = image.NewRGBA(image.Rect(0, 0, 1, 1))
	withIcon := ComposeShopScreen(pictured, image.Point{}, false, nil, false)
	iconOnly := v
	iconOnly.Shelf[0].Icon = pictured.Shelf[0].Icon
	if string(withIcon.Pix) != string(ComposeShopScreen(iconOnly, image.Point{}, false, nil, false).Pix) {
		t.Error("a cell with a picture also prints its text")
	}
}

func shopLayoutTestFont() *text.Font {
	font := &text.Font{Glyphs: make([]text.Glyph, 224)}
	font.Glyphs['A'-text.FirstChar] = text.Glyph{Width: 3, Height: 3, Advance: 4,
		Pixels: []text.Pixel{
			{}, {}, {},
			{}, {Level: text.MaxLevel, Painted: true}, {},
			{}, {}, {},
		}}
	return font
}
