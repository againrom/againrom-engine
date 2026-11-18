package game

import (
	"image"
	"math"
	"testing"

	"againrom/pkg/ui"
)

// With Smoothing on, an installed body's sibling of another size is centred on
// it at every window size; with Smoothing off no sibling is drawn.
func TestReleaseMissionSiblingIsCentredOnItsBodyAtEveryWindowSize(t *testing.T) {
	f := releaseFront(t)
	app := cardCaptionMission(t, f, f.DirectNewGame(120))
	live := f.live
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	live.push()
	for _, win := range []image.Point{{1024, 768}, {1280, 720}, {1920, 1080}, {2560, 1440}, {1920, 1200}} {
		for _, smoothing := range []bool{true, false} {
			app.SetFrameSmoothing(smoothing)
			app.Layout(win.X, win.Y)
			inspectionCentre(live, 133, 78)
			if _, _, err := app.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartFrame, Scaled: true}); err != nil {
				t.Fatal(err)
			}
			draws, err := live.view.HeadlessArtDraws()
			if err != nil {
				t.Fatal(err)
			}
			pairs, differ := 0, 0
			for i := 0; i+1 < len(draws); i++ {
				a, b := draws[i], draws[i+1]
				if a.Kind != "sprite" || b.Kind != "sprite" || a.Frame == nil || a.Frame.Boundary == nil || b.Frame != a.Frame.Boundary {
					continue
				}
				pairs++
				fa, fb := a.Frame, b.Frame
				if fa.Width == fb.Width && fa.Height == fb.Height {
					continue
				}
				differ++
				zoom := math.Abs(a.Geometry.Element(0, 0))
				ax, ay := a.Geometry.Element(0, 2), a.Geometry.Element(1, 2)
				bx, by := b.Geometry.Element(0, 2), b.Geometry.Element(1, 2)
				if a.Geometry.Element(0, 0) < 0 {
					ax -= float64(fa.Width) * zoom
					bx -= float64(fb.Width) * zoom
				}
				dx, dy := (bx-ax)/zoom, (by-ay)/zoom
				if want := float64(fa.Width/2 - fb.Width/2); math.Abs(dx-want) > 1e-6 {
					t.Fatalf("%v: %dx%d body, %dx%d sibling: sibling starts %v art pixels right of the body, want %v", win, fa.Width, fa.Height, fb.Width, fb.Height, dx, want)
				}
				if want := float64(fa.Height/2 - fb.Height/2); math.Abs(dy-want) > 1e-6 {
					t.Fatalf("%v: %dx%d body, %dx%d sibling: sibling starts %v art pixels below the body, want %v", win, fa.Width, fa.Height, fb.Width, fb.Height, dy, want)
				}
			}
			if smoothing && differ == 0 {
				t.Fatalf("%v: no body drawn with a sibling of another size (%d pairs)", win, pairs)
			}
			if !smoothing && pairs != 0 {
				t.Fatalf("%v: Smoothing off drew %d siblings", win, pairs)
			}
		}
	}
}
