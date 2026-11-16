package ui

import (
	"image"
	"testing"
)

// The mockup places the trigger window at the corner, within the bounds and
// with the row count the editor gives its own, before it centres it, at the
// same window size.
func TestMockupPlacesTheTriggerWindowAsTheEditorDoes(t *testing.T) {
	font := wordsFont("en")
	for _, size := range []image.Point{{640, 480}, {800, 600}, {1920, 1080}} {
		e, _ := wizardChoiceEditor(t)
		e.font = font
		e.SetBodyFont(font)
		e.Layout(size.X, size.Y)
		e.places[e.storeRow(popObject)] = popupStanding{}
		want, _, ok := e.objectSpecBase()
		if !ok {
			t.Fatal("the editor builds no trigger window")
		}
		m, err := NewEditorMockup(triggerWizardDesc().encode(), "en", size, font, font, nil)
		if err != nil {
			t.Fatal(err)
		}
		if m.paint.At != want.At || m.paint.Bounds != want.Bounds || m.paint.MaxRows != want.MaxRows {
			t.Errorf("%v: mockup at %v bounds %v rows %d; editor at %v bounds %v rows %d",
				size, m.paint.At, m.paint.Bounds, m.paint.MaxRows, want.At, want.Bounds, want.MaxRows)
		}
		t.Logf("%v: at %v bounds %v rows %d", size, want.At, want.Bounds, want.MaxRows)
	}
}
