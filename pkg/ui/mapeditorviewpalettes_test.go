package ui

import "testing"

// A map with no Edits lists the unit and structure palettes from its
// Palettes host, and a press on a cell arms no placing.
func TestViewOnlyMapListsPalettesAndArmsNothing(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	e.Layout(640, 480)
	host, ok := e.doc.Edits.(InspectionNewRecords)
	if !ok {
		t.Fatal("fixture host lists no palettes")
	}
	e.doc.Palettes, e.doc.Edits = host, nil
	for _, key := range []string{"units", "structures"} {
		chromeClick(e, barPoint(t, e, key))
		cell, ok := e.FindControl(ControlQuery{Role: ControlPlaceGridCell, N: 0})
		if !ok {
			t.Fatalf("%s: no palette cell on a viewed map", key)
		}
		chromeClick(e, center(cell))
		if _, running := e.addRunning(); running || e.pick == pickPlace {
			t.Fatalf("%s: a viewed map armed a placing", key)
		}
	}
	if len(calls.lines) != 0 {
		t.Fatalf("a press reached the host: %v", calls.lines)
	}
}
