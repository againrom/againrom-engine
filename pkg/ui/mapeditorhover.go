package ui

import (
	"image"
	"image/draw"
	"time"
)

// The editor's hover tooltip over a palette's grid cells: the game's one
// tooltip clock and hover box, in the editor's font and colours.

// editorTipDelay is how long the pointer rests on a grid cell before the
// cell's tooltip shows, in milliseconds: one of the game's tooltip delays.
const editorTipDelay = 300

// editorTip is the tooltip clock and the text of the cell it times.
type editorTip struct {
	clock tooltipController
	text  string
}

// editorTipStyle is the game's hover box in the editor's body font, paper
// ink, panel fill and gold bevel.
func (e *MapEditor) editorTipStyle() *hoverStyle {
	return &hoverStyle{font: e.bodyFont(), ink: editorPaper, box: &hoverBoxInk{fill: editorPanel, light: editorGold, dark: editorInk}}
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

// observeTip gives the tooltip clock a pointer sample: the clock restarts on
// a new cell and stops while a button is held or no cell is under p.
func (e *MapEditor) observeTip(p image.Point, now time.Time, held bool) {
	e.tip.clock.setDelay(editorTipDelay)
	text, at, ok := e.gridTipAt(p)
	e.tip.text = text
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

// drawTip draws the shown grid cell tooltip, its lower left corner at the
// pointer.
func (e *MapEditor) drawTip(pic *image.RGBA) {
	text := e.TipShown()
	if text == "" {
		return
	}
	t := tooltipTarget{kind: tooltipText, id: "editor", lines: []string{text}, font: e.bodyFont()}
	if img, at, ok := tooltipPicture(t, e.pointer, pic.Bounds(), nil, e.editorTipStyle()); ok {
		draw.Draw(pic, img.Bounds().Add(at), img, image.Point{}, draw.Over)
	}
}
