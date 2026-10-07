package ui

import (
	"image"
	"image/color"
	"math/rand"
	"testing"
)

func referenceCompactPanelInset(background *image.RGBA) image.Point {
	b := background.Bounds()
	counts := map[color.RGBA]int{}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			counts[background.RGBAAt(x, y)]++
		}
	}
	var fill color.RGBA
	best := -1
	for c, n := range counts {
		if n > best {
			best, fill = n, c
		}
	}
	left, top := 0, 0
	for x := b.Min.X; x < b.Max.X; x++ {
		n := 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			if background.RGBAAt(x, y) == fill {
				n++
			}
		}
		if n*4 >= b.Dy() {
			left = x - b.Min.X
			break
		}
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		n := 0
		for x := b.Min.X; x < b.Max.X; x++ {
			if background.RGBAAt(x, y) == fill {
				n++
			}
		}
		if n*4 >= b.Dx() {
			top = y - b.Min.Y
			break
		}
	}
	return image.Pt(left+2, top+2)
}

func TestCompactPanelInsetMatchesPerPixelMeasure(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for trial := 0; trial < 40; trial++ {
		w, h := 20+rng.Intn(60), 20+rng.Intn(60)
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		fill := color.RGBA{R: 30, G: 40, B: 50, A: 255}
		frame := color.RGBA{R: 200, G: 10, B: 10, A: 255}
		l, tp := rng.Intn(8), rng.Intn(8)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				c := fill
				if x < l || y < tp {
					c = frame
				} else if rng.Intn(9) == 0 {
					c = color.RGBA{R: uint8(rng.Intn(256)), G: uint8(rng.Intn(256)), B: 7, A: 255}
				}
				img.SetRGBA(x, y, c)
			}
		}
		want := referenceCompactPanelInset(img)
		for pass := 0; pass < 2; pass++ {
			if got := compactPanelInset(img); got != want {
				t.Fatalf("trial %d pass %d: inset %v, per-pixel %v", trial, pass, got, want)
			}
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if x < l+3 || y < tp+3 {
					img.SetRGBA(x, y, frame)
				} else {
					img.SetRGBA(x, y, fill)
				}
			}
		}
		if got, want := compactPanelInset(img), referenceCompactPanelInset(img); got != want {
			t.Fatalf("trial %d edited: inset %v, per-pixel %v", trial, got, want)
		}
	}
	if got := compactPanelInset(nil); got != image.Pt(4, 3) {
		t.Fatalf("nil background inset %v", got)
	}
	sub := image.NewRGBA(image.Rect(0, 0, 40, 40)).SubImage(image.Rect(5, 5, 30, 30)).(*image.RGBA)
	if got, want := compactPanelInset(sub), referenceCompactPanelInset(sub); got != want {
		t.Fatalf("sub-image inset %v, per-pixel %v", got, want)
	}
}
