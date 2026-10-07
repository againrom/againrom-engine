package catmullrom

import (
	"image"
	"math"
	"testing"
)

var shippedScales = []float64{1.40625, 1.875, 2.25, 3.0}

func flat(w, h int, c [4]byte) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		copy(img.Pix[i:i+4], c[:])
	}
	return img
}

func placed(src *image.RGBA, s float64) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, int(math.Ceil(float64(src.Rect.Dx())*s)), int(math.Ceil(float64(src.Rect.Dy())*s))))
	Resample(dst, src, s, 0, 0)
	return dst
}

func checkPremultiplied(t *testing.T, name string, img *image.RGBA) {
	t.Helper()
	for i := 0; i < len(img.Pix); i += 4 {
		a := img.Pix[i+3]
		if img.Pix[i] > a || img.Pix[i+1] > a || img.Pix[i+2] > a {
			t.Fatalf("%s: pixel %d = %v has a colour channel above alpha", name, i/4, img.Pix[i:i+4])
		}
	}
}

func TestResampleScaleOneIsIdentity(t *testing.T) {
	src := image.NewRGBA(image.Rect(3, 5, 12, 11))
	for i := range src.Pix {
		src.Pix[i] = byte(i * 37)
	}
	for i := 0; i < len(src.Pix); i += 4 {
		a := src.Pix[i+3]
		src.Pix[i], src.Pix[i+1], src.Pix[i+2] = min(src.Pix[i], a), min(src.Pix[i+1], a), min(src.Pix[i+2], a)
	}
	dst := image.NewRGBA(image.Rect(0, 0, 20, 20))
	Resample(dst, src, 1, 4, 2)
	for y := 0; y < src.Rect.Dy(); y++ {
		for x := 0; x < src.Rect.Dx(); x++ {
			if got, want := dst.RGBAAt(4+x, 2+y), src.RGBAAt(src.Rect.Min.X+x, src.Rect.Min.Y+y); got != want {
				t.Fatalf("(%d,%d) = %v, want %v", x, y, got, want)
			}
		}
	}
}

func TestResampleFlatFieldStaysFlat(t *testing.T) {
	c := [4]byte{90, 140, 200, 255}
	for _, s := range shippedScales {
		dst := placed(flat(13, 9, c), s)
		_, x1 := Covered(13, s, 0)
		_, y1 := Covered(9, s, 0)
		for y := 0; y < y1; y++ {
			for x := 0; x < x1; x++ {
				if p := dst.Pix[dst.PixOffset(x, y):][:4]; [4]byte(p) != c {
					t.Fatalf("scale %v: (%d,%d) = %v, want %v", s, x, y, p, c)
				}
			}
		}
	}
}

func TestWeightsSumToOne(t *testing.T) {
	for k := 0; k < 1000; k++ {
		f := float64(k) / 1000
		w := Weights(f)
		if sum := w[0] + w[1] + w[2] + w[3]; math.Abs(sum-1) > 1e-12 {
			t.Fatalf("f=%v: weights %v sum to %v", f, w, sum)
		}
	}
}

func TestResampleOnePixelSourceIsFlatAndTapsStayInside(t *testing.T) {
	c := [4]byte{10, 20, 30, 40}
	for _, s := range shippedScales {
		dst := placed(flat(1, 1, c), s)
		_, n := Covered(1, s, 0)
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				if p := dst.Pix[dst.PixOffset(x, y):][:4]; [4]byte(p) != c {
					t.Fatalf("scale %v: (%d,%d) = %v, want %v", s, x, y, p, c)
				}
			}
		}
	}
	for _, n := range []int{1, 2, 7} {
		for k := -40; k < 40*n+40; k++ {
			idx, _ := Taps(float64(k)/40, n)
			for _, i := range idx {
				if i < 0 || i >= n {
					t.Fatalf("n=%d p=%v: tap %d leaves the source", n, float64(k)/40, i)
				}
			}
		}
	}
}

func TestResampleEdgesStayPremultipliedAndInRange(t *testing.T) {
	step := flat(8, 4, [4]byte{0, 0, 0, 255})
	for y := 0; y < 4; y++ {
		for x := 4; x < 8; x++ {
			copy(step.Pix[step.PixOffset(x, y):], []byte{255, 255, 255, 255})
		}
	}
	sprite := flat(8, 8, [4]byte{})
	for y := 2; y < 6; y++ {
		for x := 2; x < 6; x++ {
			copy(sprite.Pix[sprite.PixOffset(x, y):], []byte{255, 255, 255, 255})
		}
	}
	for _, s := range shippedScales {
		checkPremultiplied(t, "step", placed(step, s))
		checkPremultiplied(t, "sprite edge", placed(sprite, s))
	}
}

func TestResampleLinearRampStaysLinear(t *testing.T) {
	const n = 32
	src := image.NewRGBA(image.Rect(0, 0, n, 1))
	for x := 0; x < n; x++ {
		copy(src.Pix[4*x:], []byte{byte(8 * x), byte(8 * x), byte(8 * x), 255})
	}
	for _, s := range shippedScales {
		dst := placed(src, s)
		for x := int(3 * s); x < int((n-3)*s); x++ {
			want := 8 * ((float64(x)+0.5)/s - 0.5)
			if got := float64(dst.Pix[4*x]); math.Abs(got-want) > 0.5+1e-9 {
				t.Fatalf("scale %v: x=%d = %v, want %.3f", s, x, got, want)
			}
		}
	}
}
