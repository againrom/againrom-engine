package ui

import (
	"fmt"
	"image"
	"strconv"
)

// InspectionBook is a spell book a page of W shows under its grid, drawn by
// the game's book painter from Art: Cells in book order, Chosen's cell lit,
// each cell's tip. Grow is the tip of the + at each end of the book, which
// takes no press. Drops are the page's drag and drop rules; Remove is the
// value W's Remove button sends the field's key, "" for a button that does
// nothing on the page.
type InspectionBook struct {
	Art    *BottomHUDArt
	Cells  []SpellEntry
	Chosen uint32
	Tips   []string
	Grow   string
	Drops  []InspectionDrop
	Remove string
}

// DropPlace is where a drag starts or ends on a page with a book.
type DropPlace uint8

const (
	// DropGrid is a cell of the page's grid as a start, the whole grid as
	// an end.
	DropGrid DropPlace = iota + 1
	// DropBook is a cell of the book.
	DropBook
)

// InspectionDrop is one drag and drop rule of a page with a book: a press
// carried from From onto To sends the book field's key "Word from to". A
// grid cell is named by its choice's value, a book cell by its number from
// 1, the grid as an end by 0. Empty lands only on an empty book cell.
type InspectionDrop struct {
	From, To DropPlace
	Word     string
	Empty    bool
}

// dropParts is each DropPlace's part of a window.
var dropParts = map[DropPlace]popupPart{DropGrid: popupGridPart, DropBook: popupCanvasBox}

// bookGrow is the width of the + at each end of a book and its gap to the
// end column.
const bookGrow, bookGrowGap = 20, 6

// modBook is book b as a canvas width pixels wide and its drop rules: the
// painter draws the book over the canvas's whole width, each cell is a box
// that takes a press, its tip naming its spell, and the + stands outside
// each end column. Without art each cell is a button.
func (e *MapEditor) modBook(b *InspectionBook, width int) (*popupCanvas, []popupDrop) {
	view := max(bookBarW+2*(bookGrow+bookGrowGap), width-2*popupPad)
	bar := image.Rect(0, 0, view-2, bookBarH)
	c := &popupCanvas{Sheet: image.Pt(view, bookBarH+2), Height: bookBarH + 2}
	if b.Art != nil {
		c.Picture = composeOriginalSpellBar(b.Art, b.Cells, b.Chosen, bar, 0)
		c.Picture.Rect = c.Picture.Rect.Add(image.Pt(1, 1))
	}
	cells := bookCellRects(bar, bookColumns)
	for i, r := range cells[:min(len(cells), len(b.Cells))] {
		box := canvasBox{Area: r.Add(image.Pt(1, 1)), Bare: c.Picture != nil, Empty: b.Cells[i].ID == 0, Held: b.Cells[i].Icon}
		if i < len(b.Tips) {
			box.Tip = b.Tips[i]
		}
		c.Boxes = append(c.Boxes, box)
	}
	if b.Grow != "" && len(cells) > 0 {
		top, bottom := cells[0].Min.Y+1, cells[len(cells)-1].Max.Y+1
		left, right := cells[0].Min.X+1-bookGrowGap-bookGrow, cells[bookColumns-1].Max.X+1+bookGrowGap
		for _, x := range []int{left, right} {
			c.Boxes = append(c.Boxes, canvasBox{Area: image.Rect(x, top, x+bookGrow, bottom), Lines: []string{"+"}, Inert: true, Tip: b.Grow})
		}
	}
	var drops []popupDrop
	for _, d := range b.Drops {
		drops = append(drops, popupDrop{From: dropParts[d.From], To: dropParts[d.To], Empty: d.Empty, Word: d.Word})
	}
	return c, drops
}

// modBookField is the shown page's field with a book; false for none.
func (e *MapEditor) modBookField() (InspectionField, bool) {
	for _, f := range e.obj.fields {
		if f.Book != nil {
			return f, true
		}
	}
	return InspectionField{}, false
}

// modCarryClick is a press on a grid cell or a book cell that did not
// carry: the cell's choice for the grid's field, "choose N" for the book's.
func (e *MapEditor) modCarryClick(s popupSpec, part popupPart, n int) {
	if part == popupCanvasBox {
		if f, ok := e.modBookField(); ok {
			_ = e.modAction(f.Key, "choose "+strconv.Itoa(n+1))
		}
		return
	}
	e.modGridPress(n)
}

// modGridPress picks choice n of the page's grid field.
func (e *MapEditor) modGridPress(n int) {
	for _, f := range e.obj.fields {
		if f.Grid && n >= 0 && n < len(f.Choices) {
			_ = e.modAction(f.Key, strconv.FormatInt(f.Choices[n].Value, 10))
		}
	}
}

// modCarryDrop sends the book field's key the drop of carry c by rule d
// onto box to, -1 for the grid.
func (e *MapEditor) modCarryDrop(s popupSpec, d popupDrop, c popupCarry, to int) {
	f, ok := e.modBookField()
	if !ok {
		return
	}
	place := func(part popupPart, n int) string {
		switch {
		case part == popupCanvasBox:
			return strconv.Itoa(n + 1)
		case n < 0:
			return "0"
		}
		for _, g := range e.obj.fields {
			if g.Grid && n < len(g.Choices) {
				return strconv.FormatInt(g.Choices[n].Value, 10)
			}
		}
		return "0"
	}
	_ = e.modAction(f.Key, fmt.Sprintf("%s %s %s", d.Word, place(c.from, c.index), place(d.To, to)))
}
