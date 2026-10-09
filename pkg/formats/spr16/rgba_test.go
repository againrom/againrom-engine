package spr16

import (
	"image/color"
	"testing"
)

// Coverage is (level+1)/16 of a full byte, carried into alpha and multiplied
// into each channel: premultiplied.
func TestResolvePremultipliesByLevelPlusOne(t *testing.T) {
	pal := []Color{{R: 200, G: 100, B: 255}}
	for _, c := range []struct {
		level uint8
		want  color.RGBA
	}{
		{15, color.RGBA{R: 200, G: 100, B: 255, A: 255}},
		{7, color.RGBA{R: 99, G: 49, B: 127, A: 127}},
		{0, color.RGBA{R: 11, G: 5, B: 15, A: 15}},
	} {
		if got := Resolve(pal, pa(0, c.level)); got != c.want {
			t.Errorf("level %d = %+v, want %+v", c.level, got, c.want)
		}
	}
	if got := Resolve(pal, pa(9, 15)); got != (color.RGBA{A: 255}) {
		t.Errorf("index past the palette = %+v, want black at full coverage", got)
	}
}

func TestFrameARGBAKeepsUnpaintedCellsApartFromLevelZero(t *testing.T) {
	data := concat(paletteA(map[int][4]byte{2: {30, 20, 10, 0}}),
		frameRecord(2, 1, words(litA(1), pixA(2, 0), skipA(1))), trailerBytes(1, true))
	s := decodeAOK(t, "one frame", data, true)
	pic := s.Frames[0].RGBA(s.Palette)
	if got := pic.RGBAAt(0, 0); got != (color.RGBA{R: 0, G: 1, B: 1, A: 15}) {
		t.Fatalf("painted level 0 = %+v", got)
	}
	if got := pic.RGBAAt(1, 0); got != (color.RGBA{}) {
		t.Fatalf("unpainted = %+v, want the zero colour", got)
	}
	if n := len(s.Frames[0].Colors(s.Palette)); n != 2 {
		t.Fatalf("%d colours for a 2x1 frame", n)
	}
}
