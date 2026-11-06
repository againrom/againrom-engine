package ui

import (
	"image"
	"image/color"
	"testing"
)

// The person window draws the unit's figure once, at the game's own size,
// wherever the editor's room holds it: every pixel of the figure's face and
// body stands in the window once, no portrait cut from it beside it.
func TestEditorPersonFigureStandsAtTheGameSize(t *testing.T) {
	for _, size := range []image.Point{{1280, 1024}, {1920, 1080}} {
		for _, plain := range []bool{false, true} {
			e, host := popupEditor(t)
			_ = e.Open("popups")
			e.doc.Edits = &personPortraitProbe{&personSettingsProbe{&settingsPreviewProbe{popupHost: host}}}
			e.Layout(size.X, size.Y)
			e.Select(0, false)
			e.props.plain = plain
			s, _, ok := e.objectSpec()
			if !ok || e.obj.unit.picture == nil {
				t.Fatalf("%v plain %v: the person window has no figure", size, plain)
			}
			pic := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
			e.drawObjectPreview(pic, s)
			face := color.RGBA{R: byte(e.obj.unit.entity.HP), G: 80, A: 255}
			body := color.RGBA{B: 180, A: 255}
			faces, bodies := 0, 0
			for y := range size.Y {
				for x := range size.X {
					switch pic.RGBAAt(x, y) {
					case face:
						faces++
					case body:
						bodies++
					}
				}
			}
			// The probe's figure holds a 72x92 face and a 50x126 body.
			if faces != 72*92 || bodies != 50*126 {
				t.Errorf("%v plain %v: the window draws %d face and %d body pixels, want %d and %d", size, plain, faces, bodies, 72*92, 50*126)
			}
		}
	}
}
