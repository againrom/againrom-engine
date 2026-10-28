package ui

import (
	"strings"
	"testing"
)

// The Forest tool's presets row starts with a Clear marks button left of
// the density buttons. It is off while no kind is marked; a press unticks
// every marked kind in the grid and the list at once and the title counts
// none.
func TestForestClearMarksUnticksEveryKind(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	if err := e.Key("objects"); err != nil {
		t.Fatal(err)
	}
	r, ok := e.ListButtonRect(2, 1)
	if !ok {
		t.Fatal("no Forest button")
	}
	chromeClick(e, center(r))
	clear, ok := e.ListButtonRect(4, 0)
	sparse, ok2 := e.ListButtonRect(4, 1)
	if !ok || !ok2 || clear.Max.X > sparse.Min.X || clear.Min.Y != sparse.Min.Y {
		t.Fatalf("the clear button %v does not stand left of the density buttons %v", clear, sparse)
	}
	off := func() bool {
		s, _ := e.listSpec()
		row := s.Rows[4]
		return len(row.Disabled) > 0 && row.Disabled[0] && row.Buttons[0] == e.word("editor.brush.forest.clear")
	}
	if !off() {
		t.Fatal("the clear button is on with no kind marked")
	}
	cell, ok := e.PlaceGridCellRect(0)
	if !ok {
		t.Fatal("no forest grid cell")
	}
	chromeClick(e, center(cell))
	if len(e.forestTicked()) != 1 || off() || !strings.Contains(e.brushStatus(), "1") {
		t.Fatalf("a grid press marked %v; the button is off %v; status %q", e.forestTicked(), off(), e.brushStatus())
	}
	clear, _ = e.ListButtonRect(4, 0)
	chromeClick(e, center(clear))
	s, _ := e.listSpec()
	listed := false
	for _, row := range e.brushRows() {
		listed = listed || strings.HasPrefix(row, "[x] ")
	}
	if len(e.forestTicked()) != 0 || s.Grid == nil || s.Grid.Cells[0].Ticked || listed || !off() || !strings.HasPrefix(e.brushStatus(), "Forest: 0") {
		t.Fatalf("the clear press left %v marked (listed %v, off %v, status %q)", e.forestTicked(), listed, off(), e.brushStatus())
	}
	if part, _ := layoutPopup(s, e.font, e.bodyFont()).hit(center(clear)); part == popupRowButton {
		t.Fatal("the off button takes a press")
	}
}
