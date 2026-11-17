package textsmooth

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/witness"
	"againrom/pkg/render/text"
)

// crispGlyph is a 5x5 coverage glyph of mixed levels.
func crispGlyph() *text.Glyph {
	levels := []uint8{
		0, 15, 15, 15, 0,
		15, 8, 0, 8, 15,
		15, 3, 15, 3, 15,
		15, 0, 0, 0, 15,
		0, 15, 7, 15, 0,
	}
	g := &text.Glyph{Width: 5, Height: 5, CoverageLevels: true, Pixels: make([]text.Pixel, 25)}
	for n, l := range levels {
		g.Pixels[n] = text.Pixel{Level: l, Painted: l > 0}
	}
	return g
}

func crispDraw(g *text.Glyph, scale float64, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	Composite(dst, []text.DrawCall{{Glyph: g, Color: color.RGBA{200, 120, 40, 255}}}, scale, 0, 0)
	return dst
}

func TestCrispGlyphIsNearestBlockUpscaleAtIntegerScale(t *testing.T) {
	g := crispGlyph()
	native := crispDraw(g, 1, 5, 5)
	for _, s := range []int{1, 2, 3} {
		got := crispDraw(g, float64(s), 5*s, 5*s)
		for y := 0; y < 5*s; y++ {
			for x := 0; x < 5*s; x++ {
				if got.RGBAAt(x, y) != native.RGBAAt(x/s, y/s) {
					t.Fatalf("scale %d pixel %d,%d = %v, want native cell %v", s, x, y, got.RGBAAt(x, y), native.RGBAAt(x/s, y/s))
				}
			}
		}
	}
}

func TestCrispGlyphAtHalfScaleKeepsWholeCellsAndBoundsAlpha(t *testing.T) {
	g := crispGlyph()
	// Four cells at 1.5 are six window pixels.
	g.Width, g.Height = 4, 4
	g.Pixels = []text.Pixel{g.Pixels[0], g.Pixels[1], g.Pixels[2], g.Pixels[3],
		g.Pixels[5], g.Pixels[6], g.Pixels[7], g.Pixels[8],
		g.Pixels[10], g.Pixels[11], g.Pixels[12], g.Pixels[13],
		g.Pixels[15], g.Pixels[16], g.Pixels[17], g.Pixels[18]}
	native := crispDraw(g, 1, 4, 4)
	const w = 6
	got := crispDraw(g, 1.5, w, w)
	cells := func(d int) (int, int) { return 2 * d / 3, (2*d + 1) / 3 }
	for y := 0; y < w; y++ {
		for x := 0; x < w; x++ {
			x0, x1 := cells(x)
			y0, y1 := cells(y)
			p := got.RGBAAt(x, y)
			if x0 == x1 && y0 == y1 {
				if p != native.RGBAAt(x0, y0) {
					t.Fatalf("pixel %d,%d inside cell %d,%d = %v, want %v", x, y, x0, y0, p, native.RGBAAt(x0, y0))
				}
				continue
			}
			lo, hi := 255, 0
			for cy := y0; cy <= y1 && cy < 4; cy++ {
				for cx := x0; cx <= x1 && cx < 4; cx++ {
					a := int(native.RGBAAt(cx, cy).A)
					lo, hi = min(lo, a), max(hi, a)
				}
			}
			if a := int(p.A); a < lo-1 || a > hi+1 {
				t.Fatalf("pixel %d,%d alpha %d outside the overlapped cells' range %d..%d", x, y, a, lo, hi)
			}
		}
	}
}

// TestWriteTextCrops writes before (Bicubic) and after (Crisp) crops to AGAINROM_CROP_DIR.
func TestWriteTextCrops(t *testing.T) {
	dir := os.Getenv("AGAINROM_CROP_DIR")
	if dir == "" {
		t.Skip("AGAINROM_CROP_DIR not set")
	}
	g := crispGlyph()
	for _, sc := range []struct {
		name  string
		scale float64
	}{{"2", 2}, {"1.5", 1.5}} {
		for _, m := range []struct {
			name   string
			method Method
		}{{"before", Bicubic}, {"after", Crisp}} {
			const n = 6
			w, h := int(float64(5*n)*sc.scale)+4, int(5*sc.scale)+4
			dst := image.NewRGBA(image.Rect(0, 0, w, h))
			for i := range dst.Pix {
				dst.Pix[i] = 30
				if i%4 == 3 {
					dst.Pix[i] = 255
				}
			}
			calls := make([]text.DrawCall, n)
			for i := range calls {
				calls[i] = text.DrawCall{Glyph: g, X: i * 5, Color: color.RGBA{230, 200, 120, 255}}
			}
			composite(dst, calls, sc.scale, 2, 2, nil, m.method)
			witness.PNG(t, filepath.Join(dir, "crop-"+m.name+"-scale"+sc.name+".png"), dst)
		}
	}
}
