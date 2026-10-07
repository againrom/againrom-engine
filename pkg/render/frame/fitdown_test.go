package frame

import (
	"image"
	"testing"
)

func TestFitDownKeepsNativePixelsAndFitsSmallWindows(t *testing.T) {
	for _, tc := range []struct {
		size  image.Point
		scale float64
	}{
		{image.Pt(1280, 960), 1},
		{image.Pt(641, 481), 1},
		{image.Pt(640, 480), 1},
		{image.Pt(320, 240), 0.5},
		{image.Pt(640, 240), 0.5},
	} {
		p := FitDown(640, 480, tc.size.X, tc.size.Y)
		if !p.Valid() || p.Scale() != tc.scale {
			t.Fatalf("size=%v scale=%g want=%g", tc.size, p.Scale(), tc.scale)
		}
		for _, point := range []image.Point{{0, 0}, {100, 80}, {638, 478}} {
			x, y, ok := p.FrameToWindow(point)
			if !ok {
				t.Fatal("mapped pixel missing", tc.size, point)
			}
			if got, ok := p.WindowToFrame(x, y); !ok || got != point {
				t.Fatal("pointer round trip", tc.size, point, got, ok)
			}
		}
	}
	if FitDown(640, 480, 0, 480).Valid() {
		t.Fatal("zero window accepted")
	}
}
