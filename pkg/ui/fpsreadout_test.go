package ui

import (
	"bytes"
	"image"
	"image/color"
	"math"
	"testing"

	"againrom/pkg/render/text"
)

func TestFPSReadoutSamplingAndCapture(t *testing.T) {
	_, v := mkOnMap(t)
	v.SetFont(solidFont15())
	text.ResetCapture()
	t.Cleanup(text.ResetCapture)
	for i := 0; i <= 50; i++ {
		rate, pic, _, shown := v.FPSReadout(5000 + int64(i)*20)
		if rate != 0 || math.Signbit(rate) || pic != nil || shown {
			t.Fatalf("hidden priming/boundary draw %d: %v %v %v", i, rate, pic, shown)
		}
	}
	want := 52.0 * 1000 / 1040
	if rate, _, _, shown := v.FPSReadout(6040); rate != want || shown {
		t.Fatalf("hidden sample = %v shown %v, want %v hidden", rate, shown, want)
	}
	v.ToggleFPS()
	v.SetTextSmoothing(false)
	rate, off, at, shown := v.FPSReadout(6040)
	if rate != want || !shown || off == nil || len(text.Captured()) != 0 {
		t.Fatal("toggle reset or smoothing-off captured", rate, shown, text.Captured())
	}
	v.SetTextSmoothing(true)
	rate, on, _, shown := v.FPSReadout(6040)
	calls := text.Captured()
	if rate != want || !shown || !bytes.Equal(off.Pix, on.Pix) || len(calls) != 2*len(FPSLine(50)) {
		t.Fatal("capture changed raster or lost cached glyphs", rate, shown, len(calls))
	}
	n := len(calls) / 2
	for i := 0; i < n; i++ {
		s, ink := calls[i], calls[n+i]
		if !s.Flat || ink.Flat || s.X != ink.X+1 || s.Y != ink.Y+1 ||
			s.Color != (color.RGBA{8, 8, 8, 255}) || ink.Color != (color.RGBA{255, 255, 255, 255}) ||
			ink.Clip != on.Bounds().Add(at) {
			t.Fatal("shadow order/offset/ink/clip", i, s, ink)
		}
	}
	v.ToggleFPS()
	if rate, pic, _, shown := v.FPSReadout(6040); rate != want || pic != nil || shown {
		t.Fatal("hide reset sample", rate, shown)
	}
}

func TestFPSReadoutRampAndClipping(t *testing.T) {
	_, v := mkOnMap(t)
	f := &text.Font{Glyphs: make([]text.Glyph, 224)}
	g := text.Glyph{Width: 18, Height: 1, Advance: 18, Pixels: make([]text.Pixel, 18)}
	for i := 0; i < 16; i++ {
		g.Pixels[i] = text.Pixel{Level: uint8(i), Painted: true}
	}
	g.Pixels[17] = text.Pixel{Painted: true}
	f.Glyphs[int('0')-32] = g
	v.SetFont(f)
	v.ToggleFPS()
	v.SetTextSmoothing(false)
	pic, _, shown := v.fpsPresent(0)
	if !shown {
		t.Fatal("no synthetic ramp control")
	}
	x := 82 - 36
	for i := 0; i < 16; i++ {
		c := uint8(17 * i)
		if got := pic.RGBAAt(x+i, 0); got != (color.RGBA{c, c, c, 255}) {
			t.Fatalf("level %d = %v", i, got)
		}
		if got := pic.RGBAAt(x+i+1, 1); got != (color.RGBA{8, 8, 8, 255}) {
			t.Fatalf("flat shadow level %d = %v", i, got)
		}
	}
	if pic.RGBAAt(x+16, 0) != (color.RGBA{8, 8, 8, 255}) || pic.RGBAAt(x+17, 0) != (color.RGBA{0, 0, 0, 255}) {
		t.Fatal("transparent and painted zero collapsed")
	}
	wide, _, ok := v.fpsPresent(1000000000000000)
	if !ok || wide.Bounds() != image.Rect(0, 0, 90, 24) {
		t.Fatal("long line escaped its fixed box")
	}
	v.cam.ViewW = 119
	if _, _, ok := v.fpsPresent(0); ok {
		t.Fatal("negative-origin box was presented")
	}
}
