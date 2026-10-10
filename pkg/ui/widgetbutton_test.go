package ui

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"againrom/pkg/render/backdrop"
)

// rgb565 expands a packed RGB565 word as the painters' quantizer does.
func rgb565(w uint16) color.RGBA {
	return color.RGBA{uint8(int(w>>11) * 255 / 31), uint8(int(w>>5&63) * 255 / 63), uint8(int(w&31) * 255 / 31), 255}
}

// The DIALOGUE-065 bevel words, RGB565: light 10791, dark 97.
var (
	wantBevelLight = rgb565(10791)
	wantBevelDark  = rgb565(97)
)

// bevelPixels lists DLG-BUTTON-039's two colour sets for half-open r: set 1
// (dark at rest) and set 2 (light at rest).
func bevelPixels(r image.Rectangle) (one, two []image.Point) {
	l, t, rr, b := r.Min.X, r.Min.Y, r.Max.X-1, r.Max.Y-1
	for y := t + 2; y <= b-2; y++ {
		one = append(one, image.Pt(rr, y))
	}
	for y := t + 1; y <= b-1; y++ {
		one = append(one, image.Pt(rr-1, y))
	}
	for x := l + 2; x <= rr-2; x++ {
		one = append(one, image.Pt(x, b))
	}
	for x := l + 1; x <= rr-1; x++ {
		one = append(one, image.Pt(x, b-1))
	}
	one = append(one, image.Pt(rr-2, b-2))
	for x := l + 2; x <= rr-2; x++ {
		two = append(two, image.Pt(x, t))
	}
	for y := t + 2; y <= b-2; y++ {
		two = append(two, image.Pt(l, y))
	}
	two = append(two, image.Pt(l+1, t+1))
	return one, two
}

func buttonTestCanvas() *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, 160, 80))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.RGBA{96, 128, 160, 255}}, image.Point{}, draw.Src)
	return dst
}

// TestPushButtonStatesMatchThePainter checks each state against MENU-115,
// DLG-BUTTON-039 and DIALOGUE-065: the bevel colour sets, the label ink of
// both ramps, the shadow offset 2 or 4 with a fixed anchor, and the level-3
// remap of a disabled button.
func TestPushButtonStatesMatchThePainter(t *testing.T) {
	r := image.Rect(20, 10, 120, 40)
	font := solidFont15()
	// One 5x15 glyph at the anchor (L+(R'-L)/2+1-advance/2, T+(B'-T)/2-8).
	x0, y0 := 20+(119-20)/2+1-2, 10+(39-10)/2-8
	shadow := rgb565(2113)
	grey, gold, brown := rgb565(0xd69a), rgb565(48361), rgb565(37568)
	for _, tc := range []struct {
		name       string
		b          pushButton
		ink        color.RGBA
		sunk       bool
		shadowStep int
	}{
		{"menu normal", pushButton{}, grey, false, 2},
		{"menu hover", pushButton{Hover: true, Inside: true}, gold, false, 2},
		{"menu focus", pushButton{Focus: true}, gold, false, 2},
		{"menu pressed inside", pushButton{Hover: true, Pressed: true, Inside: true}, gold, true, 4},
		{"menu pressed outside", pushButton{Pressed: true}, grey, false, 2},
		{"menu disabled hover", pushButton{Hover: true, Inside: true, Disabled: true}, grey, false, 2},
		{"dialogue normal", pushButton{Ramp: pushButtonDialogue}, gold, false, 2},
		{"dialogue hover", pushButton{Ramp: pushButtonDialogue, Hover: true, Inside: true}, brown, false, 2},
		{"dialogue focus", pushButton{Ramp: pushButtonDialogue, Focus: true}, gold, false, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dst, plain := buttonTestCanvas(), buttonTestCanvas()
			b := tc.b
			b.Rect, b.Label = r, "A"
			drawPushButton(dst, font, b)
			one, two := bevelPixels(r)
			c1, c2 := wantBevelDark, wantBevelLight
			if tc.sunk {
				c1, c2 = c2, c1
			}
			want := map[image.Point]color.RGBA{}
			for _, p := range one {
				want[p] = c1
			}
			for _, p := range two {
				want[p] = c2
			}
			want[image.Pt(x0, y0)] = tc.ink
			want[image.Pt(x0+4, y0+14)] = tc.ink
			want[image.Pt(x0+4+tc.shadowStep, y0+14+tc.shadowStep)] = shadow
			want[image.Pt(x0+5, y0)] = plain.RGBAAt(x0+5, y0)
			if tc.shadowStep == 4 {
				want[image.Pt(x0+6, y0+16)] = shadow
				want[image.Pt(x0+4+2+1, y0+14+2+1)] = shadow
			} else {
				want[image.Pt(x0+4+4, y0+14+4)] = plain.RGBAAt(x0+8, y0+18)
			}
			if b.Disabled {
				l := mustLevel(t, 3)
				for p, c := range want {
					want[p] = l.Color(c)
				}
			}
			for p, c := range want {
				if got := dst.RGBAAt(p.X, p.Y); got != c {
					t.Fatalf("%v = %v, want %v", p, got, c)
				}
			}
			if got := dst.RGBAAt(2, 2); got != plain.RGBAAt(2, 2) {
				t.Fatal("the painter drew outside its rectangle")
			}
		})
	}
}

func mustLevel(t *testing.T, level uint8) *backdrop.Lookup {
	t.Helper()
	l, err := backdrop.NewLevel(backdrop.RGB565, backdrop.Full, level)
	if err != nil {
		t.Fatal(err)
	}
	return l
}
