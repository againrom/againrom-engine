package ui

import (
	"fmt"
	"image"
	"slices"
	"testing"
)

// The Trees window's Forest mode holds a Clear marks button in its button
// row. It is off while no kind is ticked; a press unticks every ticked kind
// in the grid at once and the window's note counts none.
func TestForestClearMarksUnticksEveryKind(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	chromeClick(e, barPoint(t, e, "trees"))
	if !e.forestOn() {
		t.Fatal("the Trees window opens out of Forest mode")
	}
	clear := func(ticked int) (image.Rectangle, bool) {
		t.Helper()
		s := mustListSpec(t, e)
		l := layoutPopup(s, e.font, e.bodyFont())
		if !slices.Equal(s.Buttons, []string{e.word("editor.brush.forest.clear")}) || len(l.Buttons) != 1 {
			t.Fatalf("the Forest window's buttons are %q", s.Buttons)
		}
		if want := fmt.Sprintf(e.word("editor.brush.forest.count"), ticked); !slices.Contains(s.Notes, want) {
			t.Fatalf("the Forest window's notes %q do not count %d ticked kinds", s.Notes, ticked)
		}
		return l.Buttons[0], s.Disabled[0]
	}
	if _, off := clear(0); !off {
		t.Fatal("the clear button is on with no kind ticked")
	}
	cell, ok := e.FindControl(ControlQuery{Role: ControlPlaceGridCell, N: 0})
	if !ok {
		t.Fatal("no forest grid cell")
	}
	chromeClick(e, center(cell))
	button, off := clear(1)
	if len(e.forestTicked()) != 1 || off {
		t.Fatalf("a grid press ticked %v; the button is off %v", e.forestTicked(), off)
	}
	chromeClick(e, center(button))
	s := mustListSpec(t, e)
	if _, off := clear(0); len(e.forestTicked()) != 0 || s.Grid == nil || s.Grid.Cells[0].Ticked || !off {
		t.Fatalf("the clear press left %v ticked (off %v)", e.forestTicked(), off)
	}
	chromeClick(e, center(cell))
	chromeClick(e, center(cell))
	if len(e.forestTicked()) != 0 {
		t.Fatalf("a second grid press left %v ticked", e.forestTicked())
	}
}
