package text

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

// Tinted is the arithmetic of image/draw's solid source-over fill, so a dim a
// composer drew with it and the tint the overlay derives agree on every cell.
func TestTintedMatchesDrawOver(t *testing.T) {
	tints := []color.RGBA{
		{A: 0x90},
		{A: 45},
		{R: 0, G: 7, B: 6, A: 220},
		{R: 10, G: 20, B: 30, A: 64},
		{R: 1, G: 1, B: 1, A: 1},
		{R: 128, G: 128, B: 128, A: 128},
	}
	for _, tint := range tints {
		for v := 0; v < 256; v++ {
			c := color.RGBA{R: uint8(v), G: uint8(255 - v), B: uint8(v / 2), A: 255}
			px := image.NewRGBA(image.Rect(0, 0, 1, 1))
			px.SetRGBA(0, 0, c)
			draw.Draw(px, px.Rect, &image.Uniform{C: tint}, image.Point{}, draw.Over)
			if got, want := Tinted(tint, c), px.RGBAAt(0, 0); got != want {
				t.Fatalf("Tinted(%v, %v) = %v, image/draw gives %v", tint, c, got, want)
			}
		}
	}
	if got := Tinted(color.RGBA{}, textColor); got != textColor {
		t.Fatalf("Tinted with no tint = %v, want the colour itself", got)
	}
}

// TintSince tints the glyphs captured after start whose painted cells all lie
// inside the dimmed rectangle, and only those.
func TestTintSinceTintsGlyphsInsideTheDim(t *testing.T) {
	f := levelFont()
	dst := fill(image.Rect(0, 0, 40, 4), backColor)
	ResetCapture()
	SetCapture(false)
	f.Draw(dst, "A", 0, 0, textColor)
	start := CapturedLen()
	f.Draw(dst, "A", 8, 0, textColor)
	f.Draw(dst, "A", 30, 0, textColor)
	dim := color.RGBA{A: 0x90}
	TintSince(start, image.Rect(8, 0, 20, 4), dim)
	StopCapture()
	got := Captured()
	if len(got) != 3 {
		t.Fatalf("captured %d glyphs, want 3", len(got))
	}
	for i, want := range []color.RGBA{{}, dim, {}} {
		if got[i].Tint != want {
			t.Errorf("glyph %d tint = %v, want %v", i, got[i].Tint, want)
		}
	}
	if shown, want := got[1].ShownColor(15), Tinted(dim, textColor); shown != want {
		t.Errorf("shown colour = %v, want %v", shown, want)
	}
	// An opaque or empty colour dims nothing, and outside a window nothing is
	// recorded.
	ResetCapture()
	SetCapture(false)
	f.Draw(dst, "A", 8, 0, textColor)
	TintSince(0, dst.Bounds(), color.RGBA{A: 255})
	TintSince(0, dst.Bounds(), color.RGBA{})
	StopCapture()
	if c := Captured()[0]; c.Tint != (color.RGBA{}) {
		t.Errorf("opaque or empty dim tinted a glyph: %v", c.Tint)
	}
	TintSince(0, dst.Bounds(), dim)
	if c := Captured()[0]; c.Tint != (color.RGBA{}) {
		t.Errorf("TintSince outside a window tinted a glyph: %v", c.Tint)
	}
	ResetCapture()
}
