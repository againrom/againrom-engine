package spr256_test

import (
	"image/color"
	"testing"

	"againrom/pkg/formats/spr256"
)

// One 2x1 frame, opaque index 1 then a hole, under a palette whose entry 1 is
// stored B=3, G=2, R=1.
func rgbaSheet(hasPalette bool) []byte {
	block := []byte{0x01, 1, 0x81}
	parts := [][]byte{}
	if hasPalette {
		parts = append(parts, palette(map[int][4]byte{1: {3, 2, 1, 9}}))
	}
	return concat(append(parts, frameRecord(2, 1, block), trailer(1, hasPalette))...)
}

func TestFrameRGBAResolvesThroughTheSheetsOwnTable(t *testing.T) {
	s := decodeOK(t, "own palette", rgbaSheet(true))
	table, ok := s.Table()
	if !ok {
		t.Fatal("a sheet with a palette answered no table")
	}
	if table[1] != (color.RGBA{R: 1, G: 2, B: 3, A: 0xff}) {
		t.Fatalf("table[1] = %+v", table[1])
	}
	pic := s.Frames[0].RGBA(table)
	if pic.Bounds().Dx() != 2 || pic.Bounds().Dy() != 1 {
		t.Fatalf("bounds %v", pic.Bounds())
	}
	if got := pic.RGBAAt(0, 0); got != (color.RGBA{R: 1, G: 2, B: 3, A: 0xff}) {
		t.Fatalf("opaque pixel = %+v", got)
	}
	if got := pic.RGBAAt(1, 0); got != (color.RGBA{}) {
		t.Fatalf("hole = %+v, want transparent", got)
	}
}

// The table is a caller option: a sheet without a palette is drawn through a
// table its registry row names, and an opaque index 0 is a colour, not a hole.
func TestFrameColorsTakesAnotherTable(t *testing.T) {
	s := decodeOK(t, "no palette", rgbaSheet(false))
	if _, ok := s.Table(); ok {
		t.Fatal("a sheet without a palette answered a table")
	}
	var shared [256]color.RGBA
	shared[0] = color.RGBA{R: 50, A: 0xff}
	shared[1] = color.RGBA{G: 60, A: 0xff}
	got := s.Frames[0].Colors(&shared)
	if len(got) != 2 || got[0] != shared[1] || got[1] != (color.RGBA{}) {
		t.Fatalf("colors = %+v", got)
	}
	f := spr256.Frame{Width: 1, Height: 1, Pixels: []spr256.Pixel{{Index: 0, Opaque: true}}}
	if c := f.Colors(&shared)[0]; c != shared[0] {
		t.Fatalf("opaque index 0 = %+v, want entry 0", c)
	}
	var nilSprite *spr256.Sprite
	if _, ok := nilSprite.Table(); ok {
		t.Fatal("a nil sprite answered a table")
	}
}
