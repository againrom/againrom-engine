package ui

import (
	"image"
	"time"
)

// The editor's one tip (R10): a (?) mark's help shows at once; the whole
// text of a cut row, a row's own tip, a grid head or cell, a canvas box, a
// toolbar button and a picture choice's row show once the pointer has
// rested editorTipDelay on them. layoutTip lays every one out and
// drawPopupTip paints it.

// editorTipDelay is how long the pointer rests on an element before its tip
// shows, in milliseconds: one of the game's tooltip delays.
const editorTipDelay = 300

// editorTip is the tip clock and the windows the last chrome drew, in draw
// order: the tip reads the foremost one under the pointer.
type editorTip struct {
	clock   tooltipController
	windows []tipWindow
	// top is the first line a scrolled tip shows, for the tip whose key
	// is topKey.
	top    int
	topKey string
	wheel  float64
}

// scrollTip consumes the wheel over a shown, paged tip's anchor.
func (e *MapEditor) scrollTip(p image.Point, in wheelSource) bool {
	tip, ok := e.shownTipAt(p)
	if !ok || tip.Rows <= 0 {
		return false
	}
	key := tip.key()
	if key != e.tip.topKey {
		e.tip.wheel = 0
	}
	e.tip.top, e.tip.topKey = tip.scrolled(tip.Top, editorWheelRows(&e.tip.wheel, in)).Top, key
	return true
}

// tipWindow is a window the chrome drew: its spec and its layout.
type tipWindow struct {
	spec   popupSpec
	layout popupLayout
}

// gridTipAt is the tip of the list popup's grid cell under p, and the cell's
// corner; false over no cell that names itself or where another window
// covers the popup.
func (e *MapEditor) gridTipAt(p image.Point) (string, image.Point, bool) {
	cell, r, ok := e.gridCellAt(p)
	return cell.Tip, r.Min, ok
}

// gridCellAt is the grid cell under p that names itself, of the list popup,
// the mod window or a field prompt, and its rectangle. A cell with TipPictures carries them
// as one TipPicture: a sheet of them, each at the cell's picture size.
func (e *MapEditor) gridCellAt(p image.Point) (popupGridCell, image.Rectangle, bool) {
	var s popupSpec
	ok := false
	switch e.chromeHit(p) {
	case popList:
		s, ok = e.listSpec()
	case popObject:
		if e.modWindow {
			s, _, ok = e.modSpec()
		}
	case popPrompt:
		s, ok = e.promptSpec()
	}
	if !ok || s.Grid == nil {
		return popupGridCell{}, image.Rectangle{}, false
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	for i, r := range l.GridCells {
		if n := l.gridIndex(i); p.In(r) && n < len(s.Grid.Cells) && s.Grid.Cells[n].Tip != "" {
			cell := s.Grid.Cells[n]
			if len(cell.TipPictures) > 0 {
				cell.TipPicture = e.fitTipSheet(cell.Tip, cell.TipPictures, s.Grid.pictureSide(), e.popupBounds())
			}
			return cell, r, true
		}
	}
	return popupGridCell{}, image.Rectangle{}, false
}

// fitTipSheet is the tip sheet of pics at side with the tip's text whole
// inside bounds: at side while the tip fits, else at the largest side that
// makes it fit.
func (e *MapEditor) fitTipSheet(label string, pics []*image.RGBA, side int, bounds image.Rectangle) *image.RGBA {
	sheet := tipSheet(pics, side)
	size := sheet.Bounds().Size()
	if body := e.bodyFont(); body != nil {
		size = layoutTip(bounds, body, image.Rectangle{Min: bounds.Min}, label, sheet, tipShape{Fit: true}).Rect.Size()
	}
	if size.X <= bounds.Dx() && size.Y <= bounds.Dy() {
		return sheet
	}
	cols := min(len(pics), tipSheetColumns)
	rows := (len(pics) + cols - 1) / cols
	over := size.Sub(sheet.Bounds().Size())
	side = min((bounds.Dx()-over.X+2)/cols-2, (bounds.Dy()-over.Y+2)/rows-2)
	return tipSheet(pics, max(side, 1))
}

// tipSheetColumns is how many pictures a row of a tip sheet holds.
const tipSheetColumns = 8

// tipSheet is pics side by side in rows of tipSheetColumns, each drawn in a
// square side pixels wide as a grid cell draws its picture, 2 pixels apart.
func tipSheet(pics []*image.RGBA, side int) *image.RGBA {
	cols := min(len(pics), tipSheetColumns)
	rows := (len(pics) + cols - 1) / cols
	sheet := image.NewRGBA(image.Rect(0, 0, cols*(side+2)-2, rows*(side+2)-2))
	for i, pic := range pics {
		at := image.Pt(i%cols*(side+2), i/cols*(side+2))
		cell := image.Rectangle{Min: at, Max: at.Add(image.Pt(side, side))}
		drawEditorPicture(sheet, editorPicture{Pic: pic, Box: cell, Clip: cell})
	}
	return sheet
}

// choiceTipAt is the tip of the open list's row under p when the field shows
// its pictures in the tip: the choice's label, its picture at full size and
// the row; false over no row or a choice with no picture.
func (e *MapEditor) choiceTipAt(p image.Point) (string, image.Image, image.Rectangle, bool) {
	o := &e.obj
	d, ok := e.openDropdown()
	if !ok || o.open >= len(o.fields) || !o.fields[o.open].IconTip || e.chromeHit(p) != popObject {
		return "", nil, image.Rectangle{}, false
	}
	f := o.fields[o.open]
	row, onRow := d.rowAt(e.bodyFont(), p)
	if !onRow || row >= len(f.Choices) {
		return "", nil, image.Rectangle{}, false
	}
	pic := f.Icons[f.Choices[row].Value]
	if pic == nil {
		return "", nil, image.Rectangle{}, false
	}
	top, _ := d.List.Visible()
	return f.Choices[row].Label, pic, d.listBox(e.bodyFont()).Row(row - top), true
}

// tipAt is the one tip under p: a picture choice's row of the open list, a
// grid cell of the list or mod window, a toolbar button, or what the
// foremost window the last chrome drew under p explains (popupTipAt);
// false for none, over an open menu or drop-down, or while a map drag
// fades the chrome.
func (e *MapEditor) tipAt(p image.Point) (popupTip, bool) {
	body := e.bodyFont()
	if body == nil || e.dragFades() {
		return popupTip{}, false
	}
	bounds := image.Rect(0, 0, e.window.X, e.window.Y)
	if label, pic, r, ok := e.choiceTipAt(p); ok {
		return hoverTip(bounds, body, r, label, pic), true
	}
	if e.placeOwnerDropdownHit(p) {
		return popupTip{}, false
	}
	if cell, r, ok := e.gridCellAt(p); ok {
		var pic image.Image
		if cell.TipPicture != nil {
			pic = cell.TipPicture
		}
		return hoverTip(bounds, body, r, cell.Tip, pic), true
	}
	switch id := e.chromeHit(p); {
	case id == popMapTabs:
		_, tabs := e.mapTabStrip()
		return popupTipAt(bounds, e.font, body, popupSpec{}, popupLayout{TabControls: &tabs}, p, true)
	case id == popStatus:
		s := e.statusSpec()
		return popupTipAt(bounds, e.font, body, s, layoutPopup(s, e.font, body), p, true)
	case id == popToolbar:
		if tip, ok := popupTipAt(bounds, e.font, body, popupSpec{}, e.toolbarLayout(), p, true); ok {
			return tip, true
		}
	case id == popMenu && e.searchTyping():
		d := e.menuDropdown(e.searchEntry(), true)
		if d.over(body, p) {
			l := popupLayout{HasList: true, List: d.listBox(body)}
			return popupTipAt(bounds, body, body, popupSpec{List: d.List}, l, p, true)
		}
		return popupTipAt(bounds, e.font, body, popupSpec{}, e.searchLayout(), p, true)
	case id == popMenu || e.openDropdownHit(p) || e.promptDropdownHit(p):
		return popupTip{}, false
	}
	for i := len(e.tip.windows) - 1; i >= 0; i-- {
		w := e.tip.windows[i]
		if part, _ := w.layout.hit(p); part != popupMiss {
			return popupTipAt(bounds, e.font, body, w.spec, w.layout, p, true)
		}
	}
	return popupTip{}, false
}

// observeTip gives the tip clock a pointer sample: the clock restarts on
// another element and stops while a button is held, none is under p or the
// tip is a (?) mark's help, which needs no wait.
func (e *MapEditor) observeTip(p image.Point, now time.Time, held bool) {
	e.tip.clock.setDelay(editorTipDelay)
	tip, ok := e.tipAt(p)
	e.tip.clock.observe(now, tip.Anchor.Min, tip.key(), held || !ok || tip.Help)
}

// shownTip is the tip that shows now: a (?) mark's help at once, any other
// once the pointer has rested editorTipDelay on its element.
func (e *MapEditor) shownTip() (popupTip, bool) {
	if !e.pointerOK {
		return popupTip{}, false
	}
	return e.shownTipAt(e.pointer)
}

func (e *MapEditor) shownTipAt(p image.Point) (popupTip, bool) {
	tip, ok := e.tipAt(p)
	key := tip.key()
	if !ok || key == "" || !(tip.Help || e.tip.clock.visible(key)) {
		return popupTip{}, false
	}
	if key == e.tip.topKey {
		tip = tip.scrolled(e.tip.top, 0)
	}
	return tip, true
}

// TipShown is the text of the tip that shows now, empty for none.
func (e *MapEditor) TipShown() string {
	tip, _ := e.shownTip()
	return tip.Text
}

// drawTip draws the shown tip over everything else in the editor.
func (e *MapEditor) drawTip(pic *image.RGBA) {
	if tip, ok := e.shownTip(); ok {
		drawPopupTip(pic, e.bodyFont(), tip, editorPopupInk)
	}
}
