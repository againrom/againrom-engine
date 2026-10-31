package game

import (
	"fmt"
	"image"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/secondshop"
	"againrom/pkg/ui"
)

// shopLiftNameInk counts the light pixels in a pack cell's body below its price plaque.
func shopLiftNameInk(pic *image.RGBA, cell image.Rectangle) int {
	body := image.Rect(cell.Min.X+2, cell.Min.Y+24, cell.Max.X-2, cell.Max.Y-2)
	n := 0
	for y := body.Min.Y; y < body.Max.Y; y++ {
		for x := body.Min.X; x < body.Max.X; x++ {
			if c := pic.RGBAAt(x, y); c.R >= 120 && c.G >= 120 && c.B >= 100 {
				n++
			}
		}
	}
	return n
}

// shopLiftDiff counts the pixels of r that differ between two frames and are
// outside every rectangle of except.
func shopLiftDiff(a, b *image.RGBA, r image.Rectangle, except ...image.Rectangle) int {
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			p := image.Pt(x, y)
			skip := false
			for _, e := range except {
				skip = skip || p.In(e)
			}
			if !skip && a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				n++
			}
		}
	}
	return n
}

// TestReleaseSecondShopLiftedItemLeavesNoName lifts a spell book and a plain
// shield out of the hero's pack in town 1's shop. The vacated slot keeps its
// back and its price figure and draws no name text where the picture was; the
// trade table is the frame before the lift, except the picture the pointer carries.
func TestReleaseSecondShopLiftedItemLeavesNoName(t *testing.T) {
	secondGameRoot(t)
	f, app, s, _ := secondShopTownOne(t)
	c := f.Town.second
	secondShopEnter(t, f, app, s)
	f.Town.gold = 100000
	books := 0
	for k := 0; k < 4 && books == 0; k++ {
		secondShopActivate(t, f, app, fmt.Sprintf("SHELF %d", k+1))
		for n, idx := range c.shop.visibleShelf() {
			if c.shop.shelf()[idx].Table == secondshop.Book {
				secondShopActivate(t, f, app, fmt.Sprintf("ITEM %d", n+1), "BUY")
				books++
				break
			}
		}
	}
	if books != 1 {
		t.Fatal("no spell book on any shelf of town 1")
	}
	secondShopActivate(t, f, app, "SHELF 1", "ITEM 1", "BUY")

	frame := func() *image.RGBA {
		pic, _, err := app.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		return pic
	}
	pointer := func(action string, x, y int) {
		if err := app.HeadlessPointer(action, x, y); err != nil {
			t.Fatal(err)
		}
	}
	// The trade table: its places are empty here, and the enchanted pack items' star trails move with every paint.
	strip := image.Rect(0, 300, 465, 392)
	for cellIndex, name := range []string{"spell book", "shield"} {
		view := s.ShopScreen()
		if cell := view.Pack[cellIndex]; cell.Icon == nil || len(cell.Text) == 0 {
			t.Fatalf("%s: pack cell %d has icon %v and name %q; want a picture and a name behind it", name, cellIndex, cell.Icon != nil, cell.Text)
		}
		before := frame()
		cell := ui.ShopPackCellRect(cellIndex)
		x, y, err := app.HeadlessShopPoint("pack", cellIndex)
		if err != nil {
			t.Fatal(err)
		}
		pointer("hover", x, y)
		pointer("press", x, y)
		to := image.Pt(x+20, y-60)
		for _, d := range []image.Point{{}, {X: 6, Y: -20}, to.Sub(image.Pt(x, y))} {
			pointer("move", x+d.X, y+d.Y)
		}
		lifted := frame()
		if got := shopLiftNameInk(lifted, cell); got != 0 {
			t.Errorf("%s: the vacated slot draws %d light pixels in its body, want no name", name, got)
		}
		carried := image.Rect(to.X-48, to.Y-48, to.X+48, to.Y+48)
		if n := shopLiftDiff(before, lifted, strip, cell, carried); n != 0 {
			t.Errorf("%s: %d pixels of the trade table outside the slot and the carried picture differ from the frame before the lift", name, n)
		}
		_, figure := ui.ShopPricePlacement(view.Art, view.PriceFont, view.Pack[cellIndex], cell)
		if figure.Empty() {
			t.Fatalf("%s: the price figure has no box", name)
		}
		if n := shopLiftDiff(before, lifted, figure); n != 0 {
			t.Errorf("%s: the price figure differs from the frame before the lift in %d pixels", name, n)
		}
		pointer("move", x, y)
		pointer("release", x, y)
		for range 3 {
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
		if n := shopLiftDiff(before, frame(), strip); n != 0 {
			t.Errorf("%s: %d pixels of the trade table differ after the item is put back", name, n)
		}
	}
}

// TestReleaseFirstShopLiftedItemKeepsItsPriceFigure is the first game's side of
// the same state: the vacated pack slot draws no name and keeps its price
// figure, which is what the second game's vacated slot does.
func TestReleaseFirstShopLiftedItemKeepsItsPriceFigure(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	r := packDragTownFixture(t, 1, false)
	index := 2 - r.s.packBase
	view := r.s.ShopScreen()
	if cell := view.Pack[index]; cell.Icon == nil || len(cell.Text) != 0 || cell.Count != 1 {
		t.Fatalf("pack cell %d has icon %v, name %q, count %d; want a single item with a picture and no name", index, cell.Icon != nil, cell.Text, cell.Count)
	}
	frame := func() *image.RGBA {
		pic, _, err := r.app.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		return pic
	}
	before := frame()
	from := r.point(t, 2)
	to := image.Pt(from.X+20, from.Y-60)
	for _, e := range []struct {
		action string
		at     image.Point
	}{{"press", from}, {"move", from}, {"move", image.Pt(from.X+6, from.Y-20)}, {"move", to}} {
		if err := r.app.HeadlessPointer(e.action, e.at.X, e.at.Y); err != nil {
			t.Fatal(err)
		}
	}
	lifted := frame()
	cell := ui.ShopPackCellRect(index)
	if shopLiftNameInk(before, cell) == 0 {
		t.Error("the resting cell shows no light pixel of its picture, so the lift proves nothing")
	}
	if got := shopLiftNameInk(lifted, cell); got != 0 {
		t.Errorf("the vacated slot draws %d light pixels in its body, want none", got)
	}
	_, figure := ui.ShopPricePlacement(view.Art, view.PriceFont, view.Pack[index], cell)
	if figure.Empty() {
		t.Fatal("the price figure has no box")
	}
	if n := shopLiftDiff(before, lifted, figure); n != 0 {
		t.Errorf("the price figure differs from the frame before the lift in %d pixels", n)
	}
}
