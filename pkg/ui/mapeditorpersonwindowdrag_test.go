package ui

import (
	"image"
	"testing"
)

// A person window dragged past any edge still draws, face and doll cut.
func TestEditorPersonWindowDragPastEveryEdgeStillDraws(t *testing.T) {
	for _, d := range []image.Point{{0, 7}, {7, 0}, {-7, 0}, {0, -7}, {7, 7}, {-7, 7}, {7, -7}, {-7, -7}} {
		t.Run(d.String(), func(t *testing.T) {
			e, host := popupEditor(t)
			_ = e.Open("popups")
			e.doc.Edits = &personPortraitProbe{&personSettingsProbe{&settingsPreviewProbe{popupHost: host}}}
			e.Layout(800, 600)
			e.Select(0, false)
			if _, _, ok := e.objectSpec(); !ok || !e.nativeUnitWindow() {
				t.Fatal("the person window is not open")
			}
			for i := 0; i < 150; i++ {
				dragTitle(t, e, d)
				if pic := e.ChromeImage(); pic == nil {
					t.Fatal("no picture")
				}
			}
			if r, ok := e.FindControl(ControlQuery{Role: ControlTitle}); !ok || !r.Overlaps(e.popupBounds()) {
				t.Fatalf("title row %v left the editor %v", r, e.popupBounds())
			}
		})
	}
}
