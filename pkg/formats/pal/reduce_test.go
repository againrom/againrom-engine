package pal

import (
	"image"
	"image/color"
	"testing"
)

// A picture of 1024 colours reduces to 256 entries, each pixel's entry near
// its colour, the same table twice; a picture of three colours keeps them.
func TestReduceBringsAPictureToAtMostTheTable(t *testing.T) {
	pic := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			pic.SetRGBA(x, y, color.RGBA{uint8(x * 8), uint8(y * 8), uint8((x + y) * 4), 0xff})
		}
	}
	out, err := Reduce(pic, EntryCount)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Palette) != EntryCount {
		t.Fatalf("%d entries", len(out.Palette))
	}
	again, _ := Reduce(pic, EntryCount)
	for i := range out.Palette {
		if out.Palette[i] != again.Palette[i] {
			t.Fatalf("entry %d differs between two reductions", i)
		}
	}
	for y := range 32 {
		for x := range 32 {
			c, want := out.At(x, y).(color.RGBA), pic.RGBAAt(x, y)
			if d := max(diff(c.R, want.R), diff(c.G, want.G), diff(c.B, want.B)); d > 24 {
				t.Fatalf("(%d,%d) is %v, want near %v", x, y, c, want)
			}
		}
	}
	few := image.NewRGBA(image.Rect(0, 0, 3, 1))
	cs := []color.RGBA{{10, 20, 30, 0xff}, {40, 50, 60, 0xff}, {70, 80, 90, 0xff}}
	for x, c := range cs {
		few.SetRGBA(x, 0, c)
	}
	exact, err := Reduce(few, EntryCount)
	if err != nil || len(exact.Palette) != 3 {
		t.Fatalf("%v, %v", exact, err)
	}
	for x, c := range cs {
		if exact.At(x, 0) != c {
			t.Errorf("pixel %d is %v, want %v", x, exact.At(x, 0), c)
		}
	}
}

func diff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}
