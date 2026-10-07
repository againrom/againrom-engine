package text

import (
	"image"
	"image/color"
	"slices"
	"testing"
)

var (
	textColor = color.RGBA{R: 200, G: 100, B: 50, A: 255}
	backColor = color.RGBA{R: 9, G: 9, B: 9, A: 255}
)

func fill(r image.Rectangle, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(r)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// levelFont holds one record at 'A' whose four columns carry, in order: level
// 15, level 8, level 0 painted, and nothing painted at all.
func levelFont() *Font {
	f := &Font{Spacing: 2, Glyphs: make([]Glyph, 224)}
	for i := range f.Glyphs {
		f.Glyphs[i] = blank(4, 2, 2)
	}
	f.Glyphs[0] = blank(4, 2, 0)
	f.Glyphs['A'-FirstChar] = inked(4, 2, 2, func(x, y int) (uint8, bool) {
		switch x {
		case 0:
			return 15, true
		case 1:
			return 8, true
		case 2:
			return 0, true
		}
		return 0, false
	})
	return f
}

// AC-7: the ramp, level by level, over a background that says which pixels were
// written at all.
func TestDrawLevels(t *testing.T) {
	f := levelFont()
	dst := fill(image.Rect(0, 0, 8, 4), backColor)
	f.Draw(dst, "A", 0, 0, textColor)

	// 200*8/15 = 106, 100*8/15 = 53, 50*8/15 = 26 — truncating.
	want := []color.RGBA{
		{R: 200, G: 100, B: 50, A: 255}, // level 15 reproduces the colour exactly
		{R: 106, G: 53, B: 26, A: 255},  // level 8
		{R: 0, G: 0, B: 0, A: 255},      // level 0 IS a painted pixel
		backColor,                       // unpainted: the background, byte for byte
	}
	for x, w := range want {
		for y := 0; y < 2; y++ {
			if got := dst.RGBAAt(x, y); got != w {
				t.Fatalf("pixel (%d,%d) = %v, want %v", x, y, got, w)
			}
		}
	}

	// Everything past the glyph's own cell is untouched.
	for x := 4; x < 8; x++ {
		for y := 0; y < 4; y++ {
			if got := dst.RGBAAt(x, y); got != backColor {
				t.Fatalf("pixel (%d,%d) outside the cell = %v, want the background", x, y, got)
			}
		}
	}

	// Shade is the ramp itself, and level 0 is black rather than transparent.
	if got := Shade(textColor, 0); got != (color.RGBA{A: 255}) {
		t.Fatalf("Shade(level 0) = %v, want an opaque black", got)
	}
	if got := Shade(textColor, MaxLevel); got != textColor {
		t.Fatalf("Shade(level %d) = %v, want the colour unchanged", MaxLevel, got)
	}
	if got := Shade(textColor, 200); got != textColor {
		t.Fatalf("Shade(level 200) = %v, want it clamped to the colour", got)
	}

	// The alpha is carried through unscaled at every level.
	for lv := uint8(0); lv <= MaxLevel; lv++ {
		if got := Shade(color.RGBA{R: 255, G: 255, B: 255, A: 128}, lv); got.A != 128 {
			t.Fatalf("Shade(level %d) alpha = %d, want 128", lv, got.A)
		}
	}
}

// DrawFlat writes the colour itself on every painted pixel, level 0 included,
// leaves an unpainted pixel and the cell's surroundings alone, places glyphs
// exactly where Draw does, and clips and tolerates a nil destination.
func TestDrawFlatIgnoresLevelsAndPlacesLikeDraw(t *testing.T) {
	f := levelFont()
	flat := color.RGBA{R: 8, G: 8, B: 8, A: 255}
	dst := fill(image.Rect(0, 0, 12, 4), backColor)
	f.DrawFlat(dst, "AA", 1, 1, flat)
	shaded := fill(image.Rect(0, 0, 12, 4), backColor)
	f.Draw(shaded, "AA", 1, 1, textColor)
	for y := 0; y < 4; y++ {
		for x := 0; x < 12; x++ {
			want := backColor
			if shaded.RGBAAt(x, y) != backColor {
				want = flat
			}
			if got := dst.RGBAAt(x, y); got != want {
				t.Fatalf("pixel (%d,%d) = %v, want %v", x, y, got, want)
			}
		}
	}
	if dst.RGBAAt(1, 1) != flat || dst.RGBAAt(2, 1) != flat || dst.RGBAAt(3, 1) != flat {
		t.Fatal("levels 15, 8 and painted 0 did not all take the flat colour")
	}
	if dst.RGBAAt(4, 1) != backColor {
		t.Fatal("the unpainted column was written")
	}
	f.DrawFlat(fill(image.Rect(0, 0, 1, 1), backColor), "AA", -20, -20, flat)
	f.DrawFlat(nil, "AA", 0, 0, flat)
}

// A capture window records every DrawFlat glyph as a flat call: its colour
// unshaded, the cells it replaced, and the bounds of the image it painted. A
// skip-raster window records the call and paints nothing.
func TestDrawFlatRecordsFlatGlyphs(t *testing.T) {
	f := levelFont()
	flat := color.RGBA{R: 8, G: 8, B: 8, A: 255}
	ResetCapture()
	t.Cleanup(ResetCapture)
	SetCapture(false)
	dst := fill(image.Rect(0, 0, 12, 4), backColor)
	f.DrawFlat(dst, "AA", 1, 1, flat)
	calls := Captured()
	if len(calls) != 2 {
		t.Fatalf("DrawFlat recorded %d glyphs in a capture window, want 2", len(calls))
	}
	for i, c := range calls {
		if !c.Flat || c.Color != flat || c.Clip != dst.Bounds() {
			t.Fatalf("call %d = flat %v colour %v clip %v, want a flat %v call clipped to %v", i, c.Flat, c.Color, c.Clip, flat, dst.Bounds())
		}
		if len(c.Under) != len(c.Glyph.Pixels) {
			t.Fatalf("call %d recorded %d replaced cells, want %d", i, len(c.Under), len(c.Glyph.Pixels))
		}
		for n, p := range c.Glyph.Pixels {
			if p.Painted && c.Under[n] != backColor {
				t.Fatalf("call %d cell %d replaced %v, want %v", i, n, c.Under[n], backColor)
			}
			if p.Painted && c.CellColor(p.Level) != flat {
				t.Fatalf("call %d cell %d shades to %v, want the flat colour", i, n, c.CellColor(p.Level))
			}
		}
	}
	ResetCapture()
	SetCapture(true)
	skipped := fill(image.Rect(0, 0, 12, 4), backColor)
	f.DrawFlat(skipped, "AA", 1, 1, flat)
	if got := CapturedLen(); got != 2 {
		t.Fatalf("skip-raster DrawFlat recorded %d glyphs, want 2", got)
	}
	if !slices.Equal(skipped.Pix, fill(image.Rect(0, 0, 12, 4), backColor).Pix) {
		t.Fatal("skip-raster DrawFlat painted a pixel")
	}
}

// AC-8: drawing off every edge clips instead of panicking, and the part that
// lands is identical to the same draw into an image with room for all of it.
func TestDrawClips(t *testing.T) {
	f := levelFont()
	const s = "ABBA A"
	w, h := f.Measure(s)

	big := fill(image.Rect(-64, -64, w+64, h+64), backColor)
	f.Draw(big, s, 0, 0, textColor)

	small := image.Rect(0, 0, 3, 1)
	for _, at := range []image.Point{
		{X: -20, Y: -20}, {X: -2, Y: 0}, {X: 0, Y: -1},
		{X: w, Y: 0}, {X: 0, Y: h}, {X: w + 50, Y: h + 50},
		{X: 1, Y: 0}, {X: 0, Y: 0},
	} {
		dst := fill(small, backColor)
		f.Draw(dst, s, at.X, at.Y, textColor)
		for y := small.Min.Y; y < small.Max.Y; y++ {
			for x := small.Min.X; x < small.Max.X; x++ {
				// The same pixel of the same layout, drawn where there was room.
				// Outside the reference image nothing was drawn either, so the
				// expectation there is the untouched background.
				ref := backColor
				if (image.Point{X: x - at.X, Y: y - at.Y}).In(big.Bounds()) {
					ref = big.RGBAAt(x-at.X, y-at.Y)
				}
				if got := dst.RGBAAt(x, y); got != ref {
					t.Fatalf("at %v pixel (%d,%d) = %v, want %v", at, x, y, got, ref)
				}
			}
		}
	}

	// A destination whose bounds do not start at the origin, and a nil one.
	off := fill(image.Rect(100, 50, 140, 70), backColor)
	f.Draw(off, s, 100, 50, textColor)
	if off.RGBAAt(100, 50) != (color.RGBA{R: 200, G: 100, B: 50, A: 255}) {
		t.Fatalf("a non-origin destination did not take the first pixel: %v", off.RGBAAt(100, 50))
	}
	f.Draw(nil, s, 0, 0, textColor)
}

// A glyph whose pixel slice is shorter than its declared cell paints what it has
// and never reads past it: a hand-built font cannot make the blit panic.
func TestDrawShortPixelSlice(t *testing.T) {
	f := &Font{Spacing: 2, Glyphs: make([]Glyph, 224)}
	for i := range f.Glyphs {
		f.Glyphs[i] = blank(4, 4, 2)
	}
	f.Glyphs[0] = blank(4, 4, 0)
	f.Glyphs['A'-FirstChar] = Glyph{Width: 4, Height: 4, Advance: 2, Pixels: []Pixel{
		{Level: 15, Painted: true}, {Level: 15, Painted: true},
	}}
	dst := fill(image.Rect(0, 0, 8, 8), backColor)
	f.Draw(dst, "AA", 0, 0, textColor)
	if _, h := f.Measure("AA"); h != 4 {
		t.Fatalf("short-slice font measured height %d, want 4", h)
	}
	if dst.RGBAAt(0, 1) != backColor {
		t.Fatal("a row the pixel slice does not reach was painted")
	}
}
