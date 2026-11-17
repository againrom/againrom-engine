package ui

import (
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/witness"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
)

func crispEditorLine() []text.DrawCall {
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
	calls := make([]text.DrawCall, 6)
	for i := range calls {
		calls[i] = text.DrawCall{Glyph: g, X: i * 8, Color: color.RGBA{230, 200, 120, 255}}
	}
	return calls
}

// TestMapEditorTextOverlayIsCrisp checks that the editor's overlay draws each
// native cell as a solid block of device pixels, as the game does.
func TestMapEditorTextOverlayIsCrisp(t *testing.T) {
	e := NewMapEditor(nil, nil, nil)
	calls := crispEditorLine()
	native := image.NewRGBA(image.Rect(0, 0, 48, 8))
	var one textOverlay
	one.cache.Composite(native, calls, 1, 0, 0)
	for _, scale := range []float64{1.25, 1.5, 2} {
		w, h := int(math.Ceil(48*scale))+2, int(math.Ceil(8*scale))+2
		got := image.NewRGBA(image.Rect(0, 0, w, h))
		e.textOverlay.cache.Composite(got, calls, scale, 0, 0)
		checked := 0
		for _, c := range calls {
			// The glyph's cells stretch over its own target rectangle.
			r := textsmooth.TargetRect(c.X, c.Y, 5, 5, scale, 0, 0)
			dw, dh := r.Dx(), r.Dy()
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					px, py := x-r.Min.X, y-r.Min.Y
					cx, cy := px*5/dw, py*5/dh
					if (px+1)*5 > (cx+1)*dw || (py+1)*5 > (cy+1)*dh {
						continue
					}
					checked++
					if want := native.RGBAAt(c.X+cx, c.Y+cy); got.RGBAAt(x, y) != want {
						t.Fatalf("scale %v pixel %d,%d = %v, want its cell %d,%d colour %v", scale, x, y, got.RGBAAt(x, y), cx, cy, want)
					}
				}
			}
		}
		if checked == 0 {
			t.Fatalf("scale %v checked no pixel", scale)
		}
	}
}

// TestWriteEditorTextCrop writes one editor text line at scale 2 to
// AGAINROM_CROP_DIR as the file named by AGAINROM_CROP_NAME.
func TestWriteEditorTextCrop(t *testing.T) {
	dir, name := os.Getenv("AGAINROM_CROP_DIR"), os.Getenv("AGAINROM_CROP_NAME")
	if dir == "" || name == "" {
		t.Skip("AGAINROM_CROP_DIR or AGAINROM_CROP_NAME not set")
	}
	e := NewMapEditor(nil, nil, nil)
	dst := image.NewRGBA(image.Rect(0, 0, 100, 12))
	for i := range dst.Pix {
		dst.Pix[i] = 30
		if i%4 == 3 {
			dst.Pix[i] = 255
		}
	}
	e.textOverlay.cache.Composite(dst, crispEditorLine(), 2, 0, 0)
	witness.PNG(t, filepath.Join(dir, name), dst)
}
