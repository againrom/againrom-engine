package ui

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"againrom/pkg/ui/widgetkit"
)

// The launcher's panel and the frame builder's panel are the same pixels.
func TestKitPanelIsTheFrameBuilderPanel(t *testing.T) {
	fill, border := color.RGBA{10, 20, 30, 255}, color.RGBA{200, 100, 50, 255}
	canvas := image.Rect(0, 0, 40, 30)
	for _, r := range []image.Rectangle{
		image.Rect(2, 2, 30, 20),
		image.Rect(-5, -5, 10, 10),
		image.Rect(20, 5, 60, 40),
		image.Rect(3, 3, 4, 4),
		image.Rect(5, 5, 5, 5),
		image.Rect(0, 0, 40, 30),
	} {
		framed, kit := image.NewRGBA(canvas), image.NewRGBA(canvas)
		drawFrame(framed, panelFrame(r, fill, border))
		widgetkit.DrawPanel(kit, r, fill, border)
		if !bytes.Equal(framed.Pix, kit.Pix) {
			t.Errorf("panel over %v: the kit and the frame builder differ", r)
		}
	}
}
