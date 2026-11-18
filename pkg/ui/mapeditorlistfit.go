package ui

import "againrom/pkg/render/text"

// listFitMeasure is what a list window's fit measures with: the body font,
// the width of a string in it, the row pitch and the window of the spec once
// its label column is set.
type listFitMeasure struct {
	Body    *text.Font
	Advance func(string) int
	Pitch   int
	Window  uiWindow
}

// listFitIn is a list window's inputs to listFit as data.
type listFitIn struct {
	// Counts says which rows' labels (with their (?)) make the label column.
	Counts func(popupRow) bool
	// Value is the width a counted row asks for after its label column and
	// the 4 pixels beside it.
	Value func(popupRow, listFitMeasure) int
	// Heads makes the grid's row heads part of the label column, and the
	// grid's head column as wide as that column.
	Heads bool
	// GridSlack is what the grid's own width adds to its head column and
	// cells.
	GridSlack int
	// Rest is the width the window's other parts ask for: its list, its
	// notes and its title.
	Rest func(label int, m listFitMeasure) int
}

// listFit gives the list window s its natural width: the label column as
// wide as its widest counted label with its (?), the grid as wide as its
// head column and cells, the counted rows, the window's other parts and the
// button row, whichever is widest. It is the one measure of that width.
func (e *MapEditor) listFit(s *popupSpec, in listFitIn) {
	s.Width = e.popupWide(popList, fitListWidth(s, e.bodyFont(), in))
}

// fitListWidth is listFit's width before a map-maker's size: it also sets
// the label column and the grid's head column of s. The mockup page of a
// description list asks it with a window of its own.
func fitListWidth(s *popupSpec, body *text.Font, in listFitIn) int {
	m := listFitMeasure{Body: body, Advance: func(t string) int {
		if body == nil {
			return 8 * len(t)
		}
		return body.Advance(t)
	}, Pitch: fontHeight(body, 15) + popupRowExtra}
	label := 0
	for _, r := range s.Rows {
		if in.Counts(r) {
			label = max(label, m.Advance(r.Label)+8+helpMarkW(r, m.Pitch))
		}
	}
	g := s.Grid
	if in.Heads && g != nil && len(g.RowHeads) > 0 {
		for _, h := range g.RowHeads {
			label = max(label, m.Advance(h)+8)
		}
		g.HeadW = label
	}
	if label > 0 {
		s.LabelWidth = label
	}
	m.Window = specWindow(*s)
	need := 0
	for _, r := range s.Rows {
		if in.Counts(r) {
			need = max(need, label+4+in.Value(r, m))
		}
	}
	if g != nil {
		cols := g.Cols
		if cols <= 0 {
			cols = min(len(g.Cells), gridLeastCols)
		}
		need = max(need, g.HeadW+cols*g.side()+widgetSpriteSize+in.GridSlack)
	}
	if in.Rest != nil {
		need = max(need, in.Rest(label, m))
	}
	return max(need+2*popupPad, buttonRowWidth(m.Window, body))
}
