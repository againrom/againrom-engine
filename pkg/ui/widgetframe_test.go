package ui

import (
	"image"
	"image/color"
	"testing"
)

func TestWindowFrameShadowFallsOnATransparentPanel(t *testing.T) {
	art := dialogueClaimFrame()
	dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
	g := newDialogGeometry(540, 420)
	drawFrame(dst, g.frame(art))
	body := g.Body()
	for _, p := range []image.Point{{body.Max.X + 2, body.Min.Y + 100}, {body.Min.X + 100, body.Max.Y + 2}, {body.Max.X + 2, body.Max.Y + 2}} {
		if got := dst.RGBAAt(p.X, p.Y); got != frameShadowTone {
			t.Errorf("shadow at %v = %v, want %v", p, got, frameShadowTone)
		}
	}
	for _, p := range []image.Point{{body.Max.X + 2, body.Min.Y + 2}, {body.Min.X + 2, body.Max.Y + 2}} {
		if got := dst.RGBAAt(p.X, p.Y); got != (color.RGBA{}) {
			t.Errorf("shadow reaches %v outside the offset pieces: %v", p, got)
		}
	}
}
