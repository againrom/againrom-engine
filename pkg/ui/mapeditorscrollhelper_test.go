package ui

// scrollGridRows scrolls the shown grid laid out as l by rows through its
// scroll area.
func scrollGridRows(e *MapEditor, l popupLayout, rows int) {
	g, _, _ := e.placeGridSpec()
	if a, ok := e.gridArea(l, g, e.placeTop(), &e.listWheel); ok {
		e.scrollWheelRows(a, rows)
	}
}
