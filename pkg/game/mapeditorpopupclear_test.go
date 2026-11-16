package game

import (
	"image"
	"slices"
	"time"

	"againrom/pkg/ui"
)

// clearMoves are the least drags that uncover points, else edge drags.
func clearMoves(e *ui.MapEditor, pop image.Rectangle, points []image.Point) []image.Point {
	lo, hi := points[0], points[0]
	for _, p := range points {
		lo, hi = image.Pt(min(lo.X, p.X), min(lo.Y, p.Y)), image.Pt(max(hi.X, p.X), max(hi.Y, p.Y))
	}
	room := e.MapRect()
	for _, d := range []image.Point{{lo.X - pop.Max.X, 0}, {hi.X + 1 - pop.Min.X, 0}, {0, hi.Y + 1 - pop.Min.Y}, {0, lo.Y - pop.Max.Y}} {
		if r := pop.Add(d); r.In(room) {
			return []image.Point{d}
		}
	}
	return []image.Point{{-5000, 0}, {5000, 0}, {0, 5000}}
}

// clearPopup drags the front popup by its title bar, the editor's own move,
// off points while it covers one, so a press there reaches the map. A drag
// is kept when the popup reopens. One no drag clears stays where it is.
func clearPopup(e *ui.MapEditor, points ...image.Point) {
	pop, ok := e.PopupRect()
	if !ok || !slices.ContainsFunc(points, func(p image.Point) bool { return p.In(pop) }) {
		return
	}
	for _, d := range clearMoves(e, pop, points) {
		pop, ok := e.PopupRect()
		if !ok || !slices.ContainsFunc(points, func(p image.Point) bool { return p.In(pop) }) {
			return
		}
		title, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlTitle})
		if !ok || !title.Overlaps(pop) {
			return
		}
		p := title.Min.Add(image.Pt(10, 10))
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
