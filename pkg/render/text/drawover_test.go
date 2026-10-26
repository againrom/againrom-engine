package text

import (
	"image"
	"testing"
)

// DrawOver mixes a coverage glyph's pixels into the destination: level 15 is
// the ink, level 8 lies between the destination and the ink, level 0 leaves
// the destination, and an unpainted pixel is not touched. A shade glyph draws
// as Draw draws it.
func TestDrawOverMixesCoverageGlyphs(t *testing.T) {
	f := levelFont()
	g := &f.Glyphs['A'-FirstChar]
	g.CoverageLevels = true
	dst := fill(image.Rect(0, 0, 8, 4), backColor)
	f.DrawOver(dst, "A", 0, 0, textColor)
	want := []struct{ R, G, B uint8 }{
		{200, 100, 50},
		{(9*7 + 200*8 + 7) / 15, (9*7 + 100*8 + 7) / 15, (9*7 + 50*8 + 7) / 15},
		{9, 9, 9},
		{9, 9, 9},
	}
	for x, w := range want {
		got := dst.RGBAAt(x, 0)
		if got.R != w.R || got.G != w.G || got.B != w.B || got.A != 255 {
			t.Errorf("column %d = %v, want %v", x, got, w)
		}
	}
	g.CoverageLevels = false
	over, plain := fill(image.Rect(0, 0, 8, 4), backColor), fill(image.Rect(0, 0, 8, 4), backColor)
	f.DrawOver(over, "A", 0, 0, textColor)
	f.Draw(plain, "A", 0, 0, textColor)
	for x := 0; x < 4; x++ {
		if over.RGBAAt(x, 0) != plain.RGBAAt(x, 0) {
			t.Errorf("shade glyph column %d: DrawOver %v, Draw %v", x, over.RGBAAt(x, 0), plain.RGBAAt(x, 0))
		}
	}
}
