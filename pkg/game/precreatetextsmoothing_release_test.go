package game

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// preCreateWindowPoint is the window pixel a native 640x480 frame point falls
// on in a w x h window.
func preCreateWindowPoint(w, h int, p image.Point) image.Point {
	x, y, _ := frame.Fit(640, 480, w, h).FrameToWindow(p)
	return image.Pt(x, y)
}

// visibleGlyphs counts the distinct captured glyphs that show at least one of
// their own cells in pix, and those among them that show some cells and not
// all. pix is the CPU composite of the same state, so this reads what the
// player sees and not what the overlay decided.
func visibleGlyphs(calls []text.DrawCall, pix *image.RGBA) (visible, partly int) {
	type key struct {
		g    *text.Glyph
		x, y int
		c    color.RGBA
	}
	seen := map[key]bool{}
	for _, call := range calls {
		g := call.Glyph
		if g == nil || g.Width <= 0 || seen[key{g, call.X, call.Y, call.Color}] {
			continue
		}
		seen[key{g, call.X, call.Y, call.Color}] = true
		shown, painted := 0, 0
		for i, p := range g.Pixels {
			if !p.Painted {
				continue
			}
			painted++
			at := image.Pt(call.X+i%g.Width, call.Y+i/g.Width)
			if at.In(pix.Bounds()) && pix.RGBAAt(at.X, at.Y) == text.Shade(call.Color, p.Level) {
				shown++
			}
		}
		if shown > 0 {
			visible++
			if shown < painted {
				partly++
			}
		}
	}
	return visible, partly
}

// The character creation page's text is smoothed wherever any of it shows.
// A glyph a tooltip covers only in part is still on the page, and the overlay
// must draw it: the prompt and the name field sit under the tooltip the name
// field raises, and a glyph left to the raster steps at window scale.
func TestReleasePreCreateTextIsSmoothedWhereverAnyOfItShows(t *testing.T) {
	for _, size := range []image.Point{{1366, 768}, {1920, 1080}, {2560, 1440}} {
		if s := frame.Fit(640, 480, size.X, size.Y).Scale(); s <= 1 {
			t.Fatalf("window %v fits the frame at scale %v, want above 1", size, s)
		}
		for _, state := range []string{"tip open", "tip closed", "pointer on the name field"} {
			t.Run(fmt.Sprintf("%dx%d/%s", size.X, size.Y, state), func(t *testing.T) {
				f := releaseFront(t)
				f.Options = OptionsStore{}
				f.SetDeterministicFrames(true)
				a := f.App("pre-create text smoothing")
				a.Layout(size.X, size.Y)
				t.Cleanup(a.StopAudio)
				c := preCreateNameOpen(t, f, a)
				if state != "tip open" {
					tip := c.TipPanel()
					if tip.Rect.Empty() {
						t.Fatal("the page opened without its tip panel")
					}
					r := ui.TipPanelCloseRect(tip.Rect)
					p := preCreateWindowPoint(size.X, size.Y, r.Min.Add(r.Size().Div(2)))
					for _, edge := range []string{"press", "release"} {
						if err := a.HeadlessPointer(edge, p.X, p.Y); err != nil {
							t.Fatal(err)
						}
					}
					if !c.TipPanel().Rect.Empty() {
						t.Fatal("the tip panel's Close left it open")
					}
				}
				if state == "pointer on the name field" {
					p := preCreateWindowPoint(size.X, size.Y, image.Pt(300, 328))
					if err := a.HeadlessPointer("hover", p.X, p.Y); err != nil {
						t.Fatal(err)
					}
					for range 30 {
						if err := a.HeadlessStep(); err != nil {
							t.Fatal(err)
						}
					}
					if st, _ := a.HeadlessTooltip(); !st.Visible {
						t.Fatal("the name field raised no tooltip")
					}
				}
				a.Draw(ebiten.NewImage(size.X, size.Y))
				pix, _, err := a.HeadlessFrame()
				if err != nil {
					t.Fatal(err)
				}
				captured, kept, fallbacks := a.TextSettle()
				visible, partly := visibleGlyphs(text.Captured(), pix)
				t.Logf("captured %d, kept %d, showing in the frame %d (%d in part), readback frames %d",
					captured, kept, visible, partly, fallbacks)
				if kept < visible {
					t.Errorf("the overlay smoothed %d glyphs and %d show in the frame: %d stay as the raster stepped them",
						kept, visible, visible-kept)
				}
				if fallbacks != 0 {
					t.Errorf("the frame needed %d readbacks; the pixel log should decide it", fallbacks)
				}
			})
		}
	}
}
