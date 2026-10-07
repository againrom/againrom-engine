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

// TestResizeConstantFieldStaysConstant checks the kernel's own weights are
// normalised: a source grid that is one uniform coverage value resizes, at
// every size this glyph pipeline actually uses (upscaling a small native
// cell by 1.875x-3x), to that same value everywhere, including right at a
// clamped edge — not just in the interior.
func TestResizeConstantFieldStaysConstant(t *testing.T) {
	for _, dims := range []struct{ sw, sh, dw, dh int }{
		{4, 6, 8, 12},
		{4, 6, 7, 11},  // 1.875x-ish, non-integer per axis
		{1, 1, 3, 3},   // degenerate 1x1 source
		{6, 8, 18, 24}, // 3x
	} {
		src := make([]float64, dims.sw*dims.sh)
		for i := range src {
			src[i] = 0.5
		}
		out := Resize(src, dims.sw, dims.sh, dims.dw, dims.dh)
		if len(out) != dims.dw*dims.dh {
			t.Fatalf("Resize(%+v): got %d values, want %d", dims, len(out), dims.dw*dims.dh)
		}
		for i, v := range out {
			if !almostEqual(v, 0.5, 1e-9) {
				t.Fatalf("Resize(%+v)[%d] = %v, want 0.5 (constant field must stay constant)", dims, i, v)
			}
		}
	}
}

// TestResizeRejectsBadSizes matches the documented nil-on-non-positive-or-
// short-source contract.
func TestResizeRejectsBadSizes(t *testing.T) {
	if Resize(nil, 0, 0, 4, 4) != nil {
		t.Error("Resize with srcW=0 must answer nil")
	}
	if Resize([]float64{1, 2}, 2, 2, 4, 4) != nil {
		t.Error("Resize with a source shorter than srcW*srcH must answer nil")
	}
	if Resize([]float64{1, 2, 3, 4}, 2, 2, 0, 4) != nil {
		t.Error("Resize with dstW=0 must answer nil")
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

// The two cubic pieces meet at 1 and the outer piece reaches 0 at 2. A wrong
// outer coefficient weighted distant samples and striped every upscaled glyph.
func TestKernelIsContinuousAndCompact(t *testing.T) {
	const eps = 1e-9
	if d := kernel(1-eps) - kernel(1+eps); d > 1e-6 || d < -1e-6 {
		t.Fatalf("kernel jumps at 1 by %v", d)
	}
	if v := kernel(2 - eps); v > 1e-6 || v < -1e-6 {
		t.Fatalf("kernel(2) = %v, want 0", v)
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
