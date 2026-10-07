package refraction

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"math"
	"testing"
)

func coordinates(rect image.Rectangle) *image.RGBA {
	img := image.NewRGBA(rect)
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetRGBA(x, y, color.RGBA{uint8(x), uint8(y), uint8(x ^ y), 255})
		}
	}
	return img
}

func clone(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Rect)
	draw.Draw(dst, dst.Rect, src, src.Rect.Min, draw.Src)
	return dst
}

func oracle(dst *image.RGBA, mapping *Map, center image.Point, clip image.Rectangle, snapshot bool) {
	src := dst
	if snapshot {
		src = clone(dst)
	}
	for x := mapping.Size - 1; x >= 0; x-- {
		for y := mapping.Size - 1; y >= 0; y-- {
			p := mapping.Offsets[x*mapping.Size+y]
			for _, c := range [][4]int{{x, y, p.X, p.Y}, {x, -y, p.X, -p.Y}, {-x, y, -p.X, p.Y}, {-x, -y, -p.X, -p.Y}} {
				d, s := center.Add(image.Pt(c[0], c[1])), center.Add(image.Pt(c[2], c[3]))
				if d.In(clip) && s.In(clip) {
					dst.SetRGBA(d.X, d.Y, src.RGBAAt(s.X, s.Y))
				}
			}
		}
	}
}

func TestPicture7MapAndCopiesMatchThePublishedContract(t *testing.T) {
	m := Picture7Map()
	if m.Size != 19 {
		t.Fatal("map size", m.Size)
	}
	for x := range 19 {
		for y := range 19 {
			want := image.Pt(x, y)
			k := int(math.Floor(math.Hypot(float64(x), float64(y)) + 0.5))
			if k < 19 {
				f := math.Sqrt(float64(k*k+16)) / 20
				if k == 0 {
					f = 1
				}
				want = image.Pt(int(math.Floor(float64(x)*f+0.5)), int(math.Floor(float64(y)*f+0.5)))
			}
			if got := m.Offsets[x*19+y]; got != want {
				t.Fatalf("map (%d,%d)=%v want %v", x, y, got, want)
			}
		}
	}
	src := coordinates(image.Rect(0, 0, 75, 61))
	dst, expected := clone(src), clone(src)
	center := image.Pt(37, 30)
	oracle(expected, m, center, src.Rect, false)
	Apply(dst, m, center, src.Rect)
	if !bytes.Equal(dst.Pix, expected.Pix) {
		t.Fatal("output differs from ordered in-place word-copy oracle")
	}
	if n := len(Resolve(m, center, src.Rect)); n != 1369 {
		t.Fatal("written support", n)
	}
	changed := 0
	for y := range 61 {
		for x := range 75 {
			if dst.RGBAAt(x, y) != src.RGBAAt(x, y) {
				changed++
				if x < 19 || x > 55 || y < 12 || y > 48 {
					t.Fatal("pixel changed outside support", x, y)
				}
			}
		}
	}
	if changed != 1084 {
		t.Fatal("changed source pixels", changed)
	}
}

func TestResolvedCopiesKeepInPlaceDependenciesAndSeparateClipping(t *testing.T) {
	for _, m := range []*Map{Picture7Map(), {Size: 2, Offsets: []image.Point{{0, 0}, {1, 0}, {0, 0}, {1, 0}}}} {
		for _, center := range []image.Point{{37, 30}, {0, 0}, {74, 60}, {-5, 30}, {75, 30}, {37, -2}} {
			for _, clip := range []image.Rectangle{image.Rect(0, 0, 75, 61), image.Rect(25, 20, 51, 42), {}} {
				src := coordinates(image.Rect(0, 0, 75, 61))
				dst, expected := clone(src), clone(src)
				oracle(expected, m, center, clip, false)
				Apply(dst, m, center, clip)
				if !bytes.Equal(dst.Pix, expected.Pix) {
					t.Fatalf("size %d center %v clip %v differs from in-place oracle", m.Size, center, clip)
				}
			}
		}
	}
	m := &Map{Size: 2, Offsets: []image.Point{{0, 0}, {1, 0}, {0, 0}, {1, 0}}}
	a, b := coordinates(image.Rect(0, 0, 9, 9)), coordinates(image.Rect(0, 0, 9, 9))
	oracle(a, m, image.Pt(4, 4), a.Rect, false)
	oracle(b, m, image.Pt(4, 4), b.Rect, true)
	if bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("dependency control does not distinguish a snapshot")
	}
}

func TestConstantBufferIsUnchangedAndRepeatedCopyIsNotIdempotent(t *testing.T) {
	m := Picture7Map()
	constant := image.NewRGBA(image.Rect(0, 0, 75, 61))
	for y := 0; y < 61; y++ {
		for x := 0; x < 75; x++ {
			constant.SetRGBA(x, y, color.RGBA{65, 77, 90, 128})
		}
	}
	before := clone(constant)
	Apply(constant, m, image.Pt(37, 30), constant.Rect)
	if !bytes.Equal(before.Pix, constant.Pix) {
		t.Fatal("callback painted new colours on a constant buffer")
	}
	varied := coordinates(constant.Rect)
	Apply(varied, m, image.Pt(37, 30), varied.Rect)
	once := clone(varied)
	Apply(varied, m, image.Pt(37, 30), varied.Rect)
	if bytes.Equal(once.Pix, varied.Pix) {
		t.Fatal("repeated-application control did not change")
	}
}
