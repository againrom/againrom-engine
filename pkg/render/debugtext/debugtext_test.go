package debugtext

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"testing"
)

func dependencyAtlas(t *testing.T) *image.RGBA {
	t.Helper()
	path := os.Getenv("AGAINROM_DEBUG_ATLAS")
	if path == "" {
		t.Skip("set AGAINROM_DEBUG_ATLAS to the pinned dependency ebitenutil/text.png")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	decoded, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds() != image.Rect(0, 0, 192, 128) {
		t.Fatalf("dependency atlas bounds = %v", decoded.Bounds())
	}
	pic := image.NewRGBA(decoded.Bounds())
	draw.Draw(pic, pic.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	t.Logf("reference dependency atlas: %s", path)
	return pic
}

func TestSourceMatchesDependencyAtlas(t *testing.T) {
	want := dependencyAtlas(t)
	got := cachedSource().atlas
	if !bytes.Equal(got.Pix, want.Pix) {
		for y := 0; y < want.Rect.Dy(); y++ {
			for x := 0; x < want.Rect.Dx(); x++ {
				if got.RGBAAt(x, y) != want.RGBAAt(x, y) {
					t.Fatalf("atlas differs at (%d,%d), rune U+%04X: got %v, want %v", x, y, y/16*32+x/6, got.RGBAAt(x, y), want.RGBAAt(x, y))
				}
			}
		}
	}
}

func TestMasksRetainExactInkAndShadow(t *testing.T) {
	ref := dependencyAtlas(t)
	for r := rune(0); r < 256; r++ {
		if r == '\n' {
			continue
		}
		visited := false
		Walk(string(r), -1, 0, func(p Placement) {
			visited = true
			if p.X != 0 || p.Y != 0 {
				t.Fatalf("rune U+%04X placed at (%d,%d)", r, p.X, p.Y)
			}
			if p.Face.Width != 6 || p.Face.Height != 16 || p.Face.Advance != 6 ||
				p.Shadow.Width != 6 || p.Shadow.Height != 16 || p.Shadow.Advance != 6 ||
				p.Face.CoverageLevels || p.Shadow.CoverageLevels {
				t.Fatalf("rune U+%04X mask metrics or coverage changed", r)
			}
			for n := 0; n < 6*16; n++ {
				face, shadow := p.Face.Pixels[n], p.Shadow.Pixels[n]
				if face.Painted && shadow.Painted {
					t.Fatalf("rune U+%04X masks overlap at cell %d", r, n)
				}
				var got color.RGBA
				if face.Painted {
					got = color.RGBA{255, 255, 255, 255}
					if face.Level != 15 {
						t.Fatalf("face intensity = %d", face.Level)
					}
				}
				if shadow.Painted {
					got = color.RGBA{A: 128}
					if shadow.Level != 15 {
						t.Fatalf("shadow intensity = %d", shadow.Level)
					}
				}
				want := ref.RGBAAt(int(r)%32*6+n%6, int(r)/32*16+n/6)
				if got != want {
					t.Fatalf("rune U+%04X mask cell %d = %v, want %v", r, n, got, want)
				}
			}
		})
		if !visited {
			t.Fatalf("rune U+%04X was not visited", r)
		}
	}
}

func referenceDraw(dst *image.RGBA, atlas *image.RGBA, s string, x, y int) {
	px, py := x+1, y
	for _, r := range s {
		if r == '\n' {
			px, py = x+1, py+16
			continue
		}
		if r < 256 {
			origin := image.Pt(int(r)%32*6, int(r)/32*16)
			draw.Draw(dst, image.Rect(px, py, px+6, py+16), atlas, origin, draw.Over)
		}
		px += 6
	}
}

func TestDrawMatchesDependencyRaster(t *testing.T) {
	atlas := dependencyAtlas(t)
	cases := []struct {
		name string
		text string
		x, y int
		clip image.Rectangle
	}{
		{"latin", "A-z 09!?\u00e9\u00ff", 1, 2, image.Rectangle{}},
		{"lines", " A\n\nB \nC", 4, -4, image.Rectangle{}},
		{"unavailable", "A\u0100B界C\xffD", -1, 1, image.Rectangle{}},
		{"blank", " \u0080 \u009f \u00a0", -1, 0, image.Rectangle{}},
		{"empty", "", 5, 2, image.Rectangle{}},
		{"clipped", "Clip\n\u00e9 A\nB", -5, -3, image.Rect(-3, -2, 17, 19)},
	}
	backgrounds := []color.RGBA{{}, {83, 117, 201, 255}, {34, 55, 82, 128}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, bg := range backgrounds {
				got := image.NewRGBA(image.Rect(-8, -6, 93, 53))
				want := image.NewRGBA(got.Rect)
				draw.Draw(got, got.Rect, image.NewUniform(bg), image.Point{}, draw.Src)
				copy(want.Pix, got.Pix)
				gotDst, wantDst := got, want
				if !tc.clip.Empty() {
					gotDst = got.SubImage(tc.clip).(*image.RGBA)
					wantDst = want.SubImage(tc.clip).(*image.RGBA)
				}
				Draw(gotDst, tc.text, tc.x, tc.y)
				referenceDraw(wantDst, atlas, tc.text, tc.x, tc.y)
				if !bytes.Equal(got.Pix, want.Pix) {
					t.Fatalf("raster differs over %v, clip %v", bg, tc.clip)
				}
				Draw(gotDst, " A\n\u00e9", 0, 6)
				referenceDraw(wantDst, atlas, " A\n\u00e9", 0, 6)
				if !bytes.Equal(got.Pix, want.Pix) {
					t.Fatalf("overlapping composition differs over %v", bg)
				}
			}
		})
	}
	Draw(nil, "nil destination", 0, 0)
}

func TestWalkPreservesPlacementAndCache(t *testing.T) {
	var first, replay []Placement
	s := "A \nB\u00e9\u0100C\xff"
	Walk(s, 3, 5, func(p Placement) { first = append(first, p) })
	Walk(s, 3, 5, func(p Placement) { replay = append(replay, p) })
	want := []image.Point{{4, 5}, {10, 5}, {4, 21}, {10, 21}, {22, 21}}
	if len(first) != len(want) || len(replay) != len(want) {
		t.Fatalf("placements = %d and %d, want %d", len(first), len(replay), len(want))
	}
	for i, p := range first {
		if image.Pt(p.X, p.Y) != want[i] || p != replay[i] {
			t.Fatalf("placement %d = %+v, replay %+v, want origin %v and identical glyph pointers", i, p, replay[i], want[i])
		}
	}
	for _, mask := range first[1].Face.Pixels {
		if mask.Painted {
			t.Fatal("space paints face pixels")
		}
	}
	for _, mask := range first[1].Shadow.Pixels {
		if mask.Painted {
			t.Fatal("space paints shadow pixels")
		}
	}
	Walk("ignored", 0, 0, nil)
}
