package game

import (
	"image"
	"slices"
	"time"

	"againrom/pkg/ui"
)

// clearPopup drags the front popup by its title bar, the editor's own move,
// to the window's left, then right, then bottom edge while it covers one of
// points, so a press there reaches the map. Popups open in the middle of the
// window, over the cells a test presses; a drag is kept when the popup
// reopens. A popup no drag clears stays, and the press lands on it.
func clearPopup(e *ui.MapEditor, points ...image.Point) {
	for _, d := range []image.Point{{-5000, 0}, {5000, 0}, {0, 5000}} {
		pop, ok := e.PopupRect()
		if !ok || !slices.ContainsFunc(points, func(p image.Point) bool { return p.In(pop) }) {
			return
		}
		title, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlTitle})
		if !ok || !title.Overlaps(pop) {
			return
		}
		p := title.Min.Add(image.Pt(10, 4))
		q := p.Add(d)
		e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Time{})
		e.Pointer(ui.Input{CursorX: q.X, CursorY: q.Y, PrimaryDown: true}, time.Time{})
		e.Pointer(ui.Input{CursorX: q.X, CursorY: q.Y}, time.Time{})
	}
}

// clearCells is clearPopup for the window points of cells.
func clearCells(e *ui.MapEditor, cells ...image.Point) {
	points := make([]image.Point, len(cells))
	for i, c := range cells {
		points[i] = e.CellPoint(c)
	}
	clearPopup(e, points...)
}
