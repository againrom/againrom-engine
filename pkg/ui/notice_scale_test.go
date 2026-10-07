package ui

import (
	"fmt"
	"image"
	"testing"
)

func TestNoticeKeepsNativePixelsAcrossMissionLayouts(t *testing.T) {
	for _, size := range []image.Point{{1024, 768}, {1920, 1080}, {2560, 1440}} {
		for _, panels := range []bool{false, true} {
			for _, zoom := range []float64{0.5, 1, 2} {
				t.Run(fmt.Sprintf("%dx%d/panels=%v/zoom=%g", size.X, size.Y, panels, zoom), func(t *testing.T) {
					v := noticeViewer(t)
					v.SetEntities(sbEntities())
					v.sel = selection{sbUnitID}
					v.SetInventorySubject(InventorySubject{ID: sbUnitID})
					v.SetSpellbookCatalog(sbUnitID, []SpellEntry{{ID: 1, Name: "spell"}})
					v.hudHidden[hudPanelPack], v.hudHidden[hudPanelBook] = !panels, !panels
					v.Layout(size.X, size.Y)
					v.cam.SetZoom(zoom)
					v.SetDialogue(Dialogue{Text: "A conversation.", Portrait: true, Speaks: true, Face: coordPicture()})
					pic, at, scale, ok := v.noticePresent()
					if !ok || pic == nil {
						t.Fatal("dialogue not presented")
					}
					if scale != 1 || pic.Bounds().Size() != image.Pt(488, 232) {
						t.Fatalf("dialogue is resampled inside mission canvas: size=%v scale=%g", pic.Bounds().Size(), scale)
					}
					l := v.noticeLayout()
					want := image.Pt((v.cam.ViewW-640)/2+l.Box.Min.X, (v.cam.ViewH-480)/2+l.Box.Min.Y)
					if at != want {
						t.Fatalf("dialogue origin=%v want=%v", at, want)
					}
					button := l.Button.Add(at)
					for _, point := range []struct {
						at  image.Point
						hit bool
					}{
						{button.Min.Add(button.Max).Div(2), true},
						{button.Min, true},
						{button.Max.Sub(image.Pt(1, 1)), true},
						{image.Pt(button.Min.X, button.Max.Y), false},
						{image.Pt(button.Min.X-2, (button.Min.Y+button.Max.Y)/2), false},
						{at.Add(image.Pt(4, 4)), false},
					} {
						x, y, mapped := v.place.FrameToWindow(point.at)
						if !mapped || v.noticeButtonAt(x, y) != point.hit {
							t.Fatalf("button mapping at %v: mapped=%v hit=%v want=%v", point.at, mapped, v.noticeButtonAt(x, y), point.hit)
						}
					}
				})
			}
		}
	}
}
