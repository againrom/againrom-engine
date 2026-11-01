package ui

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

// The loops below are the dialogue portrait's pane painters as they were
// before the pane became frame specs; the tests hold the specs to them.

func referencePanePixels(dst *image.RGBA, pane, fill image.Rectangle, framed bool, ground, edge color.RGBA) {
	for y := fill.Min.Y; y < fill.Max.Y; y++ {
		for x := fill.Min.X; x < fill.Max.X; x++ {
			c := ground
			if !framed && (x == pane.Min.X || y == pane.Min.Y || x == pane.Max.X-1 || y == pane.Max.Y-1) {
				c = edge
			}
			dst.SetRGBA(x, y, c)
		}
	}
}

func referenceBorderPixels(dst *image.RGBA, border *image.RGBA) {
	for y := 0; y < min(border.Bounds().Dy(), dst.Bounds().Dy()); y++ {
		for x := 0; x < min(border.Bounds().Dx(), dst.Bounds().Dx()); x++ {
			if c := border.RGBAAt(border.Bounds().Min.X+x, border.Bounds().Min.Y+y); c.A != 0 {
				dst.SetRGBA(x, y, c)
			}
		}
	}
}

func TestPortraitPaneSpecsEqualTheirReference(t *testing.T) {
	ground, edge := color.RGBA{10, 20, 30, 255}, color.RGBA{200, 100, 50, 255}
	canvas := func() *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, 200, 260))
		for i := 0; i < len(img.Pix); i += 4 {
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = byte(i), 7, byte(i/8), 255
		}
		return img
	}
	pane := image.Rect(8, 6, 8+noticeSurfaceW, 6+noticeSurfaceH)
	for _, fill := range []image.Rectangle{pane, pane.Inset(5)} {
		for _, framed := range []bool{false, true} {
			want, got := canvas(), canvas()
			referencePanePixels(want, pane, fill, framed, ground, edge)
			if framed {
				drawFrame(got, frameSpec{Kind: frameFill, Rect: fill, Fill: ground})
			} else {
				drawFrame(got, panelFrame(pane, ground, edge))
			}
			if framed || fill == pane {
				if string(want.Pix) != string(got.Pix) {
					t.Errorf("fill %v framed %v differs from its reference", fill, framed)
				}
			}
		}
	}
	for _, size := range []image.Point{{30, 40}, {noticeSurfaceW, noticeSurfaceH}, {noticeSurfaceW + 9, noticeSurfaceH + 9}} {
		border := image.NewRGBA(image.Rect(3, 2, 3+size.X, 2+size.Y))
		for i := 0; i < len(border.Pix); i += 4 {
			border.Pix[i], border.Pix[i+1], border.Pix[i+2], border.Pix[i+3] = byte(i), 90, 20, byte(255*(i/4%3%2))
		}
		want, got := canvas(), canvas()
		referenceBorderPixels(want, border)
		b := border.Bounds()
		drawFrame(got, frameSpec{Kind: frameArt, Rect: image.Rect(0, 0, b.Dx(), b.Dy()), Art: &DialogFrame{Portrait: border}})
		if string(want.Pix) != string(got.Pix) {
			t.Errorf("border %v differs from its reference", size)
		}
	}
}

// referenceScrollCard is the world map card's art path as it was before the
// card became a frame spec.
func referenceScrollCard(dst *image.RGBA, r image.Rectangle, top, paper, bottom image.Image) {
	body := image.Rect(r.Min.X, r.Min.Y+12, r.Max.X, r.Max.Y-12)
	for y := body.Min.Y; y < body.Max.Y; y += paper.Bounds().Dy() {
		part := image.Rect(body.Min.X, y, body.Max.X, min(y+paper.Bounds().Dy(), body.Max.Y))
		draw.Draw(dst, part, paper, paper.Bounds().Min, draw.Over)
	}
	draw.Draw(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+12), top, top.Bounds().Min, draw.Over)
	draw.Draw(dst, image.Rect(r.Min.X, r.Max.Y-12, r.Max.X, r.Max.Y), bottom, bottom.Bounds().Min, draw.Over)
}

func TestWorldMapScrollCardEqualsItsReference(t *testing.T) {
	pic := func(w, h int, c color.RGBA) *image.RGBA {
		img := image.NewRGBA(image.Rect(2, 3, 2+w, 3+h))
		for i := 0; i < len(img.Pix); i += 4 {
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G+byte(i), c.B, byte(120+i%100)
		}
		return img
	}
	top, paper, bottom := pic(160, 12, color.RGBA{200, 10, 10, 255}), pic(160, 7, color.RGBA{10, 200, 10, 255}), pic(160, 12, color.RGBA{10, 10, 200, 255})
	for _, v := range []WorldMapView{
		{Scroll: [3]image.Image{top, paper, bottom}},
		{Pressed: [3]image.Image{top, bottom, paper}},
	} {
		index := 0
		a, b, c := v.Scroll[0], v.Scroll[1], v.Scroll[2]
		if v.Scroll[0] == nil {
			index = -1
			a, b, c = v.Pressed[0], v.Pressed[2], v.Pressed[1]
		}
		for slot := 0; slot < 3; slot++ {
			want, got := image.NewRGBA(image.Rect(0, 0, 640, 480)), image.NewRGBA(image.Rect(0, 0, 640, 480))
			referenceScrollCard(want, worldMapCardRect(v, index, slot), a, b, c)
			drawWorldMapCard(got, v, index, slot, "", "", 0)
			if string(want.Pix) != string(got.Pix) {
				t.Errorf("index %d slot %d: the card differs from its reference", index, slot)
			}
		}
	}
}

// TestFrameBorderWidthAndSides checks the border against a per-pixel rule.
func TestFrameBorderWidthAndSides(t *testing.T) {
	ground, edge := color.RGBA{1, 2, 3, 255}, color.RGBA{250, 5, 6, 255}
	r := image.Rect(4, 5, 30, 22)
	for _, w := range []int{0, 1, 2, 3} {
		for _, sides := range []frameSides{0, edgeTopBit, edgeBottomBit, edgeTopBit | edgeLeftBit} {
			img := image.NewRGBA(image.Rect(0, 0, 40, 30))
			s := panelFrame(r, ground, edge)
			s.Width, s.Sides = w, sides
			drawFrame(img, s)
			bw := max(w, 1)
			all := sides == 0
			for y := 0; y < 30; y++ {
				for x := 0; x < 40; x++ {
					want := color.RGBA{}
					if image.Pt(x, y).In(r) {
						want = ground
						if (all || sides&edgeTopBit != 0) && y < r.Min.Y+bw ||
							(all || sides&edgeBottomBit != 0) && y >= r.Max.Y-bw ||
							(all || sides&edgeLeftBit != 0) && x < r.Min.X+bw ||
							(all || sides&edgeRightBit != 0) && x >= r.Max.X-bw {
							want = edge
						}
					}
					if got := img.RGBAAt(x, y); got != want {
						t.Fatalf("width %d sides %b: pixel (%d,%d) = %v, want %v", w, sides, x, y, got, want)
					}
				}
			}
		}
	}
}
