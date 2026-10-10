package ui

import (
	"image"
	"image/color"
	"math/rand"
	"testing"
)

// The card scans read pixels through Pix rows. These per-pixel forms are the
// reference they must answer as.
func referenceCardBackground(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	fill := src.RGBAAt(b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2)
	dst := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := src.RGBAAt(x, y)
			if c == fill {
				c = characterCardFill
			}
			dst.SetRGBA(x, y, c)
		}
	}
	return dst
}

func referenceCardExtent(l PanelLayout, bg *image.RGBA, y, h int, accept func(color.RGBA) bool) (left, right int) {
	left, right = 0, l.Size.X
	for ; h > 0; y, h = y+1, h-1 {
		lo, hi := l.Size.X, 0
		for x := 0; x < l.Size.X; x++ {
			if accept(bg.RGBAAt(bg.Bounds().Min.X+x, bg.Bounds().Min.Y+y)) {
				lo, hi = min(lo, x), max(hi, x+1)
			}
		}
		left, right = max(left, lo), min(right, hi)
	}
	return left + 1, right - 1
}

func TestCardPixelScansAnswerAsPerPixelReads(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	palette := []color.RGBA{characterCardFill, {30, 30, 30, 255}, {148, 89, 0, 255}, {40, 40, 20, 255}, {0, 0, 0, 0}, {33, 44, 33, 254}}
	for n := 0; n < 200; n++ {
		r := image.Rect(rng.Intn(9)-4, rng.Intn(9)-4, 0, 0)
		r.Max = r.Min.Add(image.Pt(1+rng.Intn(40), 1+rng.Intn(30)))
		img := image.NewRGBA(r)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				img.SetRGBA(x, y, palette[rng.Intn(len(palette))])
			}
		}
		if n%3 == 0 {
			img = img.SubImage(image.Rect(r.Min.X+1, r.Min.Y+1, r.Max.X, r.Max.Y)).(*image.RGBA)
			if img.Bounds().Empty() {
				continue
			}
		}
		got, want := characterCardBackground(img), referenceCardBackground(img)
		if got.Bounds() != want.Bounds() {
			t.Fatalf("case %d: bounds %v, want %v", n, got.Bounds(), want.Bounds())
		}
		for y := want.Bounds().Min.Y; y < want.Bounds().Max.Y; y++ {
			for x := want.Bounds().Min.X; x < want.Bounds().Max.X; x++ {
				if got.RGBAAt(x, y) != want.RGBAAt(x, y) {
					t.Fatalf("case %d: pixel (%d,%d) %v, want %v", n, x, y, got.RGBAAt(x, y), want.RGBAAt(x, y))
				}
			}
		}
		fill := func(c color.RGBA) bool { return c == characterCardFill }
		for k := 0; k < 20; k++ {
			l := PanelLayout{Size: image.Pt(rng.Intn(60)-5, 10)}
			y, h := rng.Intn(40)-5, rng.Intn(6)
			a, b := characterCardRowExtent(l, img, y, h)
			c, d := referenceCardExtent(l, img, y, h, fill)
			if a != c || b != d {
				t.Fatalf("case %d: row extent (%d,%d), want (%d,%d)", n, a, b, c, d)
			}
			a, b = characterCardWritableExtent(l, img, y, h)
			c, d = referenceCardExtent(l, img, y, h, characterCardWritable)
			if a != c || b != d {
				t.Fatalf("case %d: writable extent (%d,%d), want (%d,%d)", n, a, b, c, d)
			}
		}
	}
}
