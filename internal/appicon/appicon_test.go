package appicon

import (
	"bytes"
	"image/png"
	"testing"
)

func TestEveryIconIsA256SquareWithAlphaAndAllSizes(t *testing.T) {
	for _, n := range []Name{Game, MapEditor, Starter} {
		cfg, err := png.DecodeConfig(bytes.NewReader(PNG(n)))
		if err != nil || cfg.Width != 256 || cfg.Height != 256 {
			t.Fatalf("%s: %v %dx%d", n, err, cfg.Width, cfg.Height)
		}
		imgs := Images(n)
		if len(imgs) != len(Sizes) {
			t.Fatalf("%s: %d images", n, len(imgs))
		}
		for i, im := range imgs {
			if b := im.Bounds(); b.Dx() != Sizes[i] || b.Dy() != Sizes[i] {
				t.Errorf("%s image %d is %v", n, i, b)
			}
		}
	}
}

func TestIconsDiffer(t *testing.T) {
	if bytes.Equal(PNG(Game), PNG(MapEditor)) || bytes.Equal(PNG(Game), PNG(Starter)) || bytes.Equal(PNG(MapEditor), PNG(Starter)) {
		t.Fatal("two programs share an icon")
	}
	if PNG("nope") != nil || Images("nope") != nil {
		t.Fatal("unknown name answered")
	}
}

func TestSmallerSizesAreScaledNotBlank(t *testing.T) {
	for _, n := range []Name{Game, MapEditor, Starter} {
		small := Images(n)[0]
		opaque := 0
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				if _, _, _, a := small.At(x, y).RGBA(); a > 0 {
					opaque++
				}
			}
		}
		if opaque < 32 {
			t.Errorf("%s 16x16 has %d visible pixels", n, opaque)
		}
	}
}
