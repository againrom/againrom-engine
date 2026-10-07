package ui

import (
	"image"
	"image/color"
	"testing"
)

// The quick-spell digits keep the ink they had before SHOP-108 changed the
// pack readout. The literal is deliberate: it must not follow shopPriceInk.
func TestQuickSpellMarkInkIsTheOriginalGoldNotThePackReadoutInk(t *testing.T) {
	want := color.RGBA{R: 200, G: 174, B: 84, A: 0xff}
	if quickSpellMarkInk != want {
		t.Fatalf("quickSpellMarkInk = %v, want %v", quickSpellMarkInk, want)
	}
	for _, selected := range []bool{false, true} {
		dst := image.NewRGBA(image.Rect(0, 0, 24, 24))
		cell := image.Rect(8, 8, 20, 20)
		drawOriginalQuickSpellMark(dst, chargenTestFont(), cell, 0, selected)
		pad := 0
		if selected {
			pad = 2
		}
		if got := dst.RGBAAt(cell.Min.X+pad, cell.Min.Y+pad); got != want {
			t.Fatalf("selected=%v: digit pixel = %v, want %v", selected, got, want)
		}
	}
}
