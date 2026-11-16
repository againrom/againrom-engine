package ui

import (
	"image"
	"testing"
)

// The mockup gives the trigger window the bounds and the row count the editor
// gives its own, and stands it centred, where the editor's placement store
// first opens a window of its size at the same window size.
func TestMockupPlacesTheTriggerWindowAsTheEditorDoes(t *testing.T) {
	font := wordsFont("en")
	for _, size := range []image.Point{{640, 480}, {800, 600}, {1920, 1080}} {
		e, _ := wizardChoiceEditor(t)
		e.font = font
		e.SetBodyFont(font)
		e.Layout(size.X, size.Y)
		e.places[e.storeRow(popObject)] = popupStanding{}
		want, _, ok := e.objectSpec()
		if !ok {
			t.Fatal("the editor builds no trigger window")
		}
		m, err := NewEditorMockup(triggerWizardDesc().encode(), "en", size, font, font, nil)
		if err != nil {
			t.Fatal(err)
		}
		if m.paint.Bounds != want.Bounds || m.paint.MaxRows != want.MaxRows {
			t.Errorf("%v: mockup bounds %v rows %d; editor bounds %v rows %d",
				size, m.paint.Bounds, m.paint.MaxRows, want.Bounds, want.MaxRows)
		}
		r := layoutPopup(m.paint, font, font).Rect
		want.At = e.centreCorner(r.Size(), openPlace{})
		if m.paint.At != want.At || r.Min.Y < m.paint.Bounds.Min.Y {
			t.Errorf("%v: mockup window %v; a first-opened window of its size stands at %v", size, r, want.At)
		}
	}
}
