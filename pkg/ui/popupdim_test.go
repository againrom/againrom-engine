package ui

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

// popupDimKinds opens each popup kind a mission frame can show, on a viewer
// whose frame is larger than its map viewport, and names the dim colour that
// kind is authored with.
func popupDimKinds(t *testing.T) []struct {
	name string
	v    *Viewer
	want color.RGBA
} {
	t.Helper()
	var kinds []struct {
		name string
		v    *Viewer
		want color.RGBA
	}
	add := func(name string, want color.RGBA, open func(v *Viewer)) {
		v := backdropViewer(t)
		open(v)
		kinds = append(kinds, struct {
			name string
			v    *Viewer
			want color.RGBA
		}{name, v, want})
	}
	add("outcome notice", AuthoredNoticeBackdrop(), func(v *Viewer) { v.SetNotice("words", NoticeOutcome) })
	add("custom notice", color.RGBA{A: 0x30}, func(v *Viewer) {
		v.SetNoticeBackdrop(color.RGBA{A: 0x30})
		v.SetNotice("words", NoticeDialogue)
	})
	add("mission menu", AuthoredNoticeBackdrop(), func(v *Viewer) { v.menuUp = true })
	add("mission menu with a profile wash", color.RGBA{A: 64}, func(v *Viewer) {
		v.menuBackdrop = color.RGBA{A: 64}
		v.menuUp = true
	})
	add("documents window", AuthoredNoticeBackdrop(), func(v *Viewer) { v.docUp = true })
	return kinds
}

// A popup's dim covers every pixel of the presented frame: the map viewport,
// the right column and the bottom band alike. The window a popup draws is
// painted after the dim, so every pixel of the frame is outside it as far as
// the dim is concerned.
func TestPopupDimCoversTheWholePresentedFrame(t *testing.T) {
	for _, k := range popupDimKinds(t) {
		t.Run(k.name, func(t *testing.T) {
			v := k.v
			fw, fh := v.FrameSize().X, v.FrameSize().Y
			if v.cam.ViewW >= fw {
				t.Fatalf("fixture: the viewport %dx%d is not smaller than the frame %dx%d", v.cam.ViewW, v.cam.ViewH, fw, fh)
			}
			r, c, ok := v.noticeBackdropOf()
			if !ok {
				t.Fatal("a popup is open and no dim was composed")
			}
			if c != k.want {
				t.Errorf("dim colour %v, want %v", c, k.want)
			}
			base := color.RGBA{R: 200, G: 160, B: 120, A: 255}
			before := image.NewRGBA(image.Rect(0, 0, fw, fh))
			draw.Draw(before, before.Bounds(), image.NewUniform(base), image.Point{}, draw.Src)
			after := image.NewRGBA(before.Bounds())
			copy(after.Pix, before.Pix)
			draw.Draw(after, r, image.NewUniform(c), image.Point{}, draw.Over)
			undimmed := 0
			var first image.Point
			for y := 0; y < fh; y++ {
				for x := 0; x < fw; x++ {
					a, b := after.RGBAAt(x, y), before.RGBAAt(x, y)
					if int(a.R)+int(a.G)+int(a.B) >= int(b.R)+int(b.G)+int(b.B) {
						if undimmed == 0 {
							first = image.Pt(x, y)
						}
						undimmed++
					}
				}
			}
			if undimmed != 0 {
				t.Errorf("%d of %d pixels keep their undimmed value, the first at %v (dim rectangle %v, frame %dx%d)",
					undimmed, fw*fh, first, r, fw, fh)
			}
		})
	}
}

// A town menu, over any wide canvas the player's window gives, dims the whole
// canvas: its width and its height.
func TestPopupDimCoversTheWholeWideTownCanvas(t *testing.T) {
	for _, w := range []int{640, 800, 1024, 1920, 2560} {
		bounds := newWideFrameLayout(w, wideFrameCentered).bounds()
		p, ok := popupDim(bounds, AuthoredNoticeBackdrop())
		if !ok || p.rect != bounds || p.colour != AuthoredNoticeBackdrop() {
			t.Errorf("width %d: dim %v %v, want %v in the authored colour", w, p, ok, bounds)
		}
	}
}
