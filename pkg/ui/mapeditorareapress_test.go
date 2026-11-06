package ui

import (
	"image"
	"testing"
)

// TestTriggerAreaSelectsOnlyByItsLabel: on the map a press on a trigger's
// label selects the trigger; a press on bare ground inside its area clears
// the selection as bare ground outside every area does; a press on a unit
// inside the area selects the unit, not the trigger.
func TestTriggerAreaSelectsOnlyByItsLabel(t *testing.T) {
	const structure, trigger, unit = 5, 7, 0
	e, _, _ := panelsEditor(t, 1920, 1080, "")
	e.doc.Script.Areas = []InspectionArea{{Trigger: 0, Rect: image.Rect(8, 6, 21, 15)}}
	ground := func(cell image.Point) image.Point {
		e.Fit()
		p := e.CellPoint(cell)
		if id := e.chromeHitBuilt(p); id != popNone {
			t.Fatalf("the ground point %v is under window %v", p, id)
		}
		if len(e.hitsAt(p)) != 0 {
			t.Fatalf("the ground point %v is on a record", p)
		}
		return p
	}
	inside, outside := image.Pt(19, 13), image.Pt(24, 20)
	label := func() image.Point {
		r, ok := e.AreaLabelRect(0)
		if !ok {
			t.Fatal("the trigger's label is not on the map")
		}
		return r.Min.Add(image.Pt(2, 2))
	}
	cleared := func(route string) {
		t.Helper()
		if e.Selected() != -1 || e.selectedTrigger() != -1 {
			t.Fatalf("%s left %d selected, trigger %d", route, e.Selected(), e.selectedTrigger())
		}
	}

	e.Select(structure, false)
	modesClick(e, label())
	if e.Selected() != trigger || e.selectedTrigger() != 0 {
		t.Fatalf("a press on the label selected %d, trigger %d", e.Selected(), e.selectedTrigger())
	}
	modesClick(e, ground(inside))
	cleared("a press inside the selected trigger's area")

	e.Select(structure, false)
	modesClick(e, ground(inside))
	cleared("a press inside the area with the structure selected")

	modesClick(e, ground(inside))
	cleared("a press inside the area with nothing selected")

	e.Select(structure, false)
	modesClick(e, ground(outside))
	cleared("a press outside every area")

	e.Select(structure, false)
	e.Fit()
	p := e.CellPoint(e.doc.Records[unit].Cell)
	if id := e.chromeHitBuilt(p); id != popNone {
		t.Fatalf("the unit's point %v is under window %v", p, id)
	}
	modesClick(e, p)
	if e.Selected() != unit || e.selectedTrigger() != -1 {
		t.Fatalf("a press on the unit inside the area selected %d, trigger %d", e.Selected(), e.selectedTrigger())
	}
}
