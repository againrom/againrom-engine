package textsmooth

import (
	"image"
	"image/color"
	"math"
	"testing"

	"againrom/pkg/render/text"
)

func almostEqual(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

// TestRemapKnownPoints pins owner decision method C: resized coverage passes
// through unchanged, clamped only where the kernel overshoots.
func TestRemapKnownPoints(t *testing.T) {
	cases := []struct {
		v, want float64
	}{
		{0, 0},
		{0.5, 0.5},
		{1, 1},
		{0.6, 0.6},
		{0.4, 0.4},
		{1.1, 1},
		{-0.1, 0},
	}
	for _, c := range cases {
		if got := Remap(c.v); !almostEqual(got, c.want, 1e-9) {
			t.Errorf("Remap(%v) = %v, want %v", c.v, got, c.want)
		}
	}
}

func TestAlphaCoverageKeepsPaintedZeroAndClampsTheRamp(t *testing.T) {
	g := &text.Glyph{Width: 4, Height: 1, CoverageLevels: true, Pixels: []text.Pixel{
		{Level: 0, Painted: true}, {Level: 15, Painted: true},
		{Level: 255, Painted: true}, {Level: 15},
	}}
	want := []float64{1.0 / 16, 1, 1, 0}
	for i, got := range Coverage(g) {
		if got != want[i] {
			t.Fatalf("cell %d coverage = %v, want %v", i, got, want[i])
		}
	}
}

// TestTargetRectTilesWithNoGapOrOverlap is round-1's own recommendation 4:
// two glyphs placed edge to edge natively must have their OUTPUT rectangles
// share an edge exactly, at a non-integer scale, because each edge is
// rounded independently rather than derived from a rounded width.
func TestTargetRectTilesWithNoGapOrOverlap(t *testing.T) {
	const scale = 1.875
	// glyph A: native x in [0,5); glyph B starts exactly where A ends, x in [5,11).
	a := TargetRect(0, 0, 5, 10, scale, 0, 0)
	b := TargetRect(5, 0, 6, 10, scale, 0, 0)
	if a.Max.X != b.Min.X {
		t.Errorf("adjacent glyphs must tile with no gap or overlap: A ends at %d, B starts at %d", a.Max.X, b.Min.X)
	}
}

// TestCompositeReproducesCallerColourAtFullCoverage checks the required
// property directly: a fully painted glyph (every pixel level MaxLevel),
// resized 1:1, remaps to full coverage and composites to EXACTLY the
// caller's own colour at full alpha — "same caller colour... just smoothed".
func TestCompositeReproducesCallerColourAtFullCoverage(t *testing.T) {
	g := &text.Glyph{Width: 2, Height: 2, Pixels: []text.Pixel{
		{Level: text.MaxLevel, Painted: true}, {Level: text.MaxLevel, Painted: true},
		{Level: text.MaxLevel, Painted: true}, {Level: text.MaxLevel, Painted: true},
	}}
	want := color.RGBA{R: 200, G: 40, B: 90, A: 255}
	calls := []text.DrawCall{{Glyph: g, X: 0, Y: 0, Color: want}}
	dst := image.NewRGBA(image.Rect(0, 0, 2, 2))
	Composite(dst, calls, 1, 0, 0)
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			got := dst.RGBAAt(x, y)
			if got != want {
				t.Errorf("pixel (%d,%d) = %+v, want %+v", x, y, got, want)
			}
		}
	}
}

// TestCompositeLeavesUnpaintedGlyphTransparent: an unpainted glyph draws
// nothing.
func TestCompositeLeavesUnpaintedGlyphTransparent(t *testing.T) {
	g := &text.Glyph{Width: 2, Height: 2, Pixels: []text.Pixel{
		{Level: 0, Painted: false}, {Level: 9, Painted: false},
		{Level: 0, Painted: false}, {Level: 15, Painted: false},
	}}
	calls := []text.DrawCall{{Glyph: g, X: 0, Y: 0, Color: color.RGBA{R: 255, A: 255}}}
	dst := image.NewRGBA(image.Rect(0, 0, 2, 2))
	Composite(dst, calls, 1, 0, 0)
	for _, p := range dst.Pix {
		if p != 0 {
			t.Fatalf("expected a fully transparent buffer, got a non-zero byte: %v", dst.Pix)
		}
	}
}

// TestCompositeSkipsNilOrEmptyGlyph guards the loop's own nil/zero-size
// checks so a malformed capture cannot panic the presentation layer.
func TestCompositeSkipsNilOrEmptyGlyph(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 4, 4))
	calls := []text.DrawCall{
		{Glyph: nil, X: 0, Y: 0, Color: color.RGBA{A: 255}},
		{Glyph: &text.Glyph{Width: 0, Height: 3}, X: 0, Y: 0, Color: color.RGBA{A: 255}},
	}
	Composite(dst, calls, 1, 0, 0) // must not panic
	for _, p := range dst.Pix {
		if p != 0 {
			t.Fatal("a nil or zero-size glyph must draw nothing")
		}
	}
}

// A stroke shaded column by column (ROM1 fonts do) stays one opaque stroke
// in Draw's own shades; level is never read as transparency.
func TestCompositeKeepsShadedStrokeOpaque(t *testing.T) {
	g := &text.Glyph{Width: 2, Height: 1, Pixels: []text.Pixel{{Level: text.MaxLevel, Painted: true}, {Level: 2, Painted: true}}}
	c := color.RGBA{R: 240, G: 200, B: 80, A: 255}
	dst := image.NewRGBA(image.Rect(0, 0, 2, 1))
	Composite(dst, []text.DrawCall{{Glyph: g, Color: c}}, 1, 0, 0)
	bright, dark := dst.RGBAAt(0, 0), dst.RGBAAt(1, 0)
	if bright.A != 255 || dark.A != 255 {
		t.Fatalf("shaded stroke lost coverage: %+v %+v", bright, dark)
	}
	if bright.R <= dark.R || dark.R >= c.R {
		t.Fatalf("stroke shade not carried: %+v %+v", bright, dark)
	}
}

// TestSettleKeepsOnlyGlyphsVisibleInTheFrame: a glyph a later opaque picture
// covers (a tooltip over the statistics card) is dropped and left as the
// frame shows it; a visible glyph is kept once, however often it was
// captured, and its baked raster is erased back to what it covered.
func TestSettleKeepsOnlyGlyphsVisibleInTheFrame(t *testing.T) {
	cell := text.Glyph{Width: 2, Height: 2, Advance: 3, Pixels: []text.Pixel{
		{Level: text.MaxLevel, Painted: true}, {Level: text.MaxLevel, Painted: true},
		{Level: text.MaxLevel, Painted: true}, {Level: text.MaxLevel, Painted: true}}}
	font := &text.Font{Glyphs: make([]text.Glyph, 224)}
	for i := range font.Glyphs {
		font.Glyphs[i] = cell
	}
	back, ink, cover := color.RGBA{10, 20, 30, 255}, color.RGBA{200, 180, 120, 255}, color.RGBA{1, 2, 3, 255}
	frame := image.NewRGBA(image.Rect(0, 0, 12, 4))
	for i := 0; i < len(frame.Pix); i += 4 {
		copy(frame.Pix[i:], []uint8{back.R, back.G, back.B, back.A})
	}
	calls := text.Record(func() { font.Draw(frame, "AA", 0, 0, ink) })
	if len(calls) != 2 {
		t.Fatalf("recorded %d glyphs, want 2", len(calls))
	}
	second := image.Rect(calls[1].X, 0, calls[1].X+2, 2)
	for y := second.Min.Y; y < second.Max.Y; y++ {
		for x := second.Min.X; x < second.Max.X; x++ {
			frame.SetRGBA(x, y, cover)
		}
	}
	kept := Settle(frame, append(calls, calls[0]))
	if len(kept) != 1 || kept[0].X != calls[0].X {
		t.Fatalf("Settle kept %d glyphs, want only the uncovered first one", len(kept))
	}
	if got := frame.RGBAAt(calls[0].X, 0); got != back {
		t.Fatalf("visible glyph cell = %v after Settle, want the background %v", got, back)
	}
	if got := frame.RGBAAt(calls[1].X, 0); got != cover {
		t.Fatalf("covered glyph cell = %v after Settle, want the covering picture %v", got, cover)
	}
}

// shadedFont is a font whose glyphs carry every shade level, so a resample
// exercises partial coverage and the colour fields.
func shadedFont() *text.Font {
	font := &text.Font{Glyphs: make([]text.Glyph, 224)}
	for i := range font.Glyphs {
		g := text.Glyph{Width: 3, Height: 4, Advance: 4, Pixels: make([]text.Pixel, 12)}
		for k := range g.Pixels {
			level := (i + k) % (text.MaxLevel + 1)
			g.Pixels[k] = text.Pixel{Level: uint8(level), Painted: level > 0}
		}
		font.Glyphs[i] = g
	}
	return font
}

func TestCacheCompositeMatchesComposite(t *testing.T) {
	font := shadedFont()
	frame := image.NewRGBA(image.Rect(0, 0, 64, 32))
	calls := text.Record(func() {
		font.Draw(frame, "Hello, world", 1, 2, color.RGBA{200, 180, 120, 255})
		font.Draw(frame, "Hello", 3, 3, color.RGBA{40, 90, 160, 200})
		font.Draw(frame, "edge", 50, 28, color.RGBA{255, 255, 255, 255})
	})
	var cache Cache
	for _, scale := range []float64{2.25, 2.25, 3, 1.875} {
		want := image.NewRGBA(image.Rect(0, 0, 160, 80))
		Composite(want, calls, scale, 3.5, 1)
		got := image.NewRGBA(want.Rect)
		dirty := cache.Composite(got, calls, scale, 3.5, 1)
		if string(got.Pix) != string(want.Pix) {
			t.Fatalf("scale %v: the cached composite differs from Composite", scale)
		}
		for y := want.Rect.Min.Y; y < want.Rect.Max.Y; y++ {
			for x := want.Rect.Min.X; x < want.Rect.Max.X; x++ {
				if want.RGBAAt(x, y) != (color.RGBA{}) && !image.Pt(x, y).In(dirty) {
					t.Fatalf("scale %v: painted pixel %d,%d lies outside the reported rectangle %v", scale, x, y, dirty)
				}
			}
		}
	}
}

func TestDecideDefersOnlyGlyphsItCannotRuleOut(t *testing.T) {
	font := shadedFont()
	frame := image.NewRGBA(image.Rect(0, 0, 32, 8))
	for i := 3; i < len(frame.Pix); i += 4 {
		frame.Pix[i] = 255
	}
	calls := text.Record(func() { font.Draw(frame, "AB", 0, 0, color.RGBA{200, 180, 120, 255}) })
	second := calls[1].X
	unknownFrom := func(x0 int, other Verdict) func(x, y int, want color.RGBA) Verdict {
		return func(x, y int, want color.RGBA) Verdict {
			if x >= x0 {
				return other
			}
			return Matches
		}
	}
	if kept, certain := Decide(calls, frame.Rect, unknownFrom(second, Unknown)); certain || len(kept) != 1 {
		t.Fatalf("an undecidable second glyph: kept %d, certain %v; want the first kept and certain false", len(kept), certain)
	}
	if kept, certain := Decide(calls, frame.Rect, unknownFrom(second, Differs)); !certain || len(kept) != 1 {
		t.Fatalf("a covered second glyph: kept %d, certain %v; want the first kept and certain true", len(kept), certain)
	}
}
