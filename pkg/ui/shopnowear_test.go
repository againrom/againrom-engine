package ui

import (
	"image"
	"testing"
)

// A NoWear layout keeps its doll out of the drags: no doll press, and no
// release over the panel takes an item.
func TestShopNoWearDollTakesNoDrag(t *testing.T) {
	l := FirstShopLayout()
	v := ShopScreenView{Layout: &l, SlotMask: shopTestMask()}
	marked := shopFigureRect.Min.Add(image.Pt(50, 100))
	if c, ok := shopGridControlAt(v, marked); !ok || c.Kind != ShopControlDoll {
		t.Fatalf("an ordinary doll pixel = %v %v", c, ok)
	}
	if !shopPaneTakesItemAt(v, image.Pt(560, 400)) {
		t.Fatal("an ordinary panel takes no item")
	}
	l.NoWear = true
	if c, ok := shopGridControlAt(v, marked); ok && c.Kind == ShopControlDoll {
		t.Fatalf("a NoWear doll pixel = %v", c)
	}
	if shopPaneTakesItemAt(v, image.Pt(560, 400)) {
		t.Fatal("a NoWear panel takes an item")
	}
}

// BookOffset moves the open book and its cells together.
func TestShopBookOffsetMovesTheBook(t *testing.T) {
	l := FirstShopLayout()
	v := ShopScreenView{Layout: &l, Book: true, Spells: []SpellEntry{{ID: 1}}}
	first := bookCellRects(shopTableRegion, shopSpellColumns())[0]
	if _, i, ok := shopSpellEntryAt(v, first.Min); !ok || i != 0 {
		t.Fatalf("first cell at %v = %d %v", first.Min, i, ok)
	}
	l.BookOffset = image.Pt(0, 2)
	if _, i, ok := shopSpellEntryAt(v, first.Min.Add(image.Pt(0, 2))); !ok || i != 0 {
		t.Fatalf("moved first cell = %d %v", i, ok)
	}
	if _, _, ok := shopSpellEntryAt(v, first.Min); ok {
		t.Fatal("the cell's old top row still answers")
	}
}
