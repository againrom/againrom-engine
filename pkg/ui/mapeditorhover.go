package ui

import (
	"image"
	"image/draw"
	"time"
)

// The editor's hover tooltip over a palette's grid cells, the toolbar's
// buttons and the rows of an open speaker list: the game's one tooltip clock
// and hover box, in the editor's font and colours.

// editorTipDelay is how long the pointer rests on a grid cell before the
// cell's tooltip shows, in milliseconds: one of the game's tooltip delays.
const editorTipDelay = 300

// editorTip is the tooltip clock, the text of the cell it times and the
// picture the tip shows under the text, nil for none.
type editorTip struct {
	clock   tooltipController
	text    string
	picture image.Image
}

// editorTipStyle is the game's hover box in the editor's body font, paper
// ink, panel fill and gold bevel, with picture under the text.
func (e *MapEditor) editorTipStyle(picture image.Image) *hoverStyle {
	return hoverStyleFor(hoverStyleEditor, hoverParams{font: e.bodyFont(), picture: picture})
}

// gridTipAt is the tip of the list popup's grid cell under p, and the cell's
// corner; false over no cell that names itself or where another window
// covers the popup.
func (e *MapEditor) gridTipAt(p image.Point) (string, image.Point, bool) {
	if e.chromeHit(p) != popList {
		return "", image.Point{}, false
	}
	s, ok := e.listSpec()
	if !ok || s.Grid == nil {
		return "", image.Point{}, false
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	for i, r := range l.GridCells {
		if n := l.GridFirst + i; p.In(r) && n < len(s.Grid.Cells) && s.Grid.Cells[n].Tip != "" {
			return s.Grid.Cells[n].Tip, r.Min, true
		}
	}
	return "", image.Point{}, false
}

// choiceTipAt is the tip of the open list's row under p when the field shows
// its pictures in the tip: the choice's label, its picture at full size and
// the row's corner; false over no row or a choice with no picture.
func (e *MapEditor) choiceTipAt(p image.Point) (string, image.Image, image.Point, bool) {
	o := &e.obj
	d, ok := e.openDropdown()
	if !ok || o.open >= len(o.fields) || !o.fields[o.open].IconTip || e.chromeHit(p) != popObject {
		return "", nil, image.Point{}, false
	}
	f := o.fields[o.open]
	part, row := d.hit(e.bodyFont(), p)
	if part != dropdownRow || row >= len(f.Choices) {
		return "", nil, image.Point{}, false
	}
	pic := f.Icons[f.Choices[row].Value]
	if pic == nil {
		return "", nil, image.Point{}, false
	}
	top, _ := d.List.Visible()
	return f.Choices[row].Label, pic, d.listBox(e.bodyFont()).Row(row - top).Min, true
}

// observeTip gives the tooltip clock a pointer sample: the clock restarts on
// a new cell, toolbar button or list row and stops while a button is held or
// none is under p.
func (e *MapEditor) observeTip(p image.Point, now time.Time, held bool) {
	e.tip.clock.setDelay(editorTipDelay)
	if e.dragFades() {
		e.tip.text, e.tip.picture = "", nil
		e.tip.clock.observe(now, p, "", true)
		return
	}
	text, picture, at, ok := e.choiceTipAt(p)
	if !ok {
		text, at, ok = e.gridTipAt(p)
	}
	if !ok && e.chromeHit(p) == popToolbar {
		text, at, ok = e.toolTip(p)
	}
	e.tip.text, e.tip.picture = text, picture
	e.tip.clock.observe(now, at, text, held || !ok)
}

// TipShown is the text of the grid cell tooltip that shows now, empty for
// none.
func (e *MapEditor) TipShown() string {
	if e.tip.text == "" || !e.tip.clock.visible(e.tip.text) {
		return ""
	}
	return e.tip.text
}

// drawTip draws the shown tooltip, its lower left corner at the pointer.
func (e *MapEditor) drawTip(pic *image.RGBA) {
	text := e.TipShown()
	if text == "" {
		return
	}
	t := tooltipTarget{kind: tooltipText, id: "editor", lines: []string{text}, font: e.bodyFont()}
	if img, at, ok := tooltipPicture(t, e.pointer, pic.Bounds(), nil, e.editorTipStyle(e.tip.picture)); ok {
		draw.Draw(pic, img.Bounds().Add(at), img, image.Point{}, draw.Over)
	}
}
