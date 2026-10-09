package bmp

import (
	"encoding/binary"
	"image"
	"image/color"
	"testing"
)

// build assembles a synthetic 24-bit bitmap: the two headers this decoder
// accepts, then rows BOTTOM FIRST, exactly as the format stores them. Every
// fixture in this file is built here — nothing reads a game install
// (AGENTS.md rule 2).
//
// rows is top-down, one []Color per row, so a fixture reads the way the decoded
// picture is expected to and the flip is performed by the builder rather than
// by the reader of the test.
func build(t *testing.T, rows [][]Color, imgSize uint32, tail int) []byte {
	t.Helper()
	h := len(rows)
	w := 0
	if h > 0 {
		w = len(rows[0])
	}
	stride := (w*3 + 3) &^ 3

	b := make([]byte, HeaderLen+stride*h+tail)
	b[0], b[1] = 'B', 'M'
	binary.LittleEndian.PutUint32(b[2:], uint32(len(b)))
	binary.LittleEndian.PutUint32(b[10:], HeaderLen)
	binary.LittleEndian.PutUint32(b[14:], infoHeaderLen)
	binary.LittleEndian.PutUint32(b[18:], uint32(w))
	binary.LittleEndian.PutUint32(b[22:], uint32(h))
	binary.LittleEndian.PutUint16(b[26:], 1)
	binary.LittleEndian.PutUint16(b[28:], BitsPerPixel)
	binary.LittleEndian.PutUint32(b[34:], imgSize)

	for y, row := range rows {
		off := HeaderLen + (h-1-y)*stride
		for x, c := range row {
			p := off + x*3
			b[p], b[p+1], b[p+2] = c.B, c.G, c.R
		}
	}
	return b
}

func TestDecodeReadsTheRowsTopDownAndTheChannelsInOrder(t *testing.T) {
	red := Color{R: 0xff}
	green := Color{G: 0x80}
	blue := Color{B: 0x40}
	grey := Color{R: 9, G: 9, B: 9}

	rows := [][]Color{
		{red, green},
		{blue, grey},
	}
	im, err := Decode(build(t, rows, 0, 0))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if im.Width != 2 || im.Height != 2 {
		t.Fatalf("decoded %dx%d, want 2x2", im.Width, im.Height)
	}
	for y, row := range rows {
		for x, want := range row {
			if got := im.At(x, y); got != want {
				t.Errorf("pixel (%d,%d) = %+v, want %+v — the row order or the "+
					"channel order is wrong", x, y, got, want)
			}
		}
	}
}

// The two shipped payload shapes, which differ and must both read: 85 of 86
// portrait nodes carry `imgSize=0` and two bytes past their own pixels, and one
// carries a correct imgSize and no tail (`SPR256-PICT-043`).
func TestDecodeIgnoresImgSizeAndAnyTail(t *testing.T) {
	rows := [][]Color{{{R: 1}, {G: 2}, {B: 3}}}

	for _, tc := range []struct {
		name    string
		imgSize uint32
		tail    int
	}{
		{"imgSize 0 and two trailing bytes, the shape 85 of 86 nodes have", 0, 2},
		{"a correct imgSize and no tail, the shape one node has", 9, 0},
		{"a wrong imgSize, which nothing reads", 12345, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			im, err := Decode(build(t, rows, tc.imgSize, tc.tail))
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if im.Width != 3 || im.Height != 1 || im.At(2, 0) != (Color{B: 3}) {
				t.Errorf("decoded %dx%d with last pixel %+v", im.Width, im.Height, im.At(2, 0))
			}
		})
	}
}

// A width whose row is NOT already a multiple of four: the padding branch, which
// no shipped node exercises and which would otherwise read every row after the
// first three bytes out of step.
func TestDecodeHonoursTheRowPadding(t *testing.T) {
	a, b := Color{R: 0xaa}, Color{B: 0xbb}
	im, err := Decode(build(t, [][]Color{{a, a, a}, {b, b, b}}, 0, 0))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	for x := 0; x < 3; x++ {
		if im.At(x, 0) != a || im.At(x, 1) != b {
			t.Fatalf("row padding is not honoured: (%d,0)=%+v (%d,1)=%+v", x, im.At(x, 0), x, im.At(x, 1))
		}
	}
}

func TestDecodeRefusesEveryOtherShape(t *testing.T) {
	ok := build(t, [][]Color{{{R: 1}}}, 0, 0)

	bend := func(f func([]byte)) []byte {
		b := append([]byte(nil), ok...)
		f(b)
		return b
	}

	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"shorter than the header", ok[:HeaderLen-1]},
		{"not a bitmap at all", bend(func(b []byte) { b[0] = 'Z' })},
		{"a header version with a longer info block", bend(func(b []byte) {
			binary.LittleEndian.PutUint32(b[14:], 108)
		})},
		{"pixels past the end of the stream", bend(func(b []byte) {
			binary.LittleEndian.PutUint32(b[10:], 122)
		})},
		{"pixels inside the header", bend(func(b []byte) {
			binary.LittleEndian.PutUint32(b[10:], 50)
		})},
		{"no width", bend(func(b []byte) { binary.LittleEndian.PutUint32(b[18:], 0) })},
		{"two colour planes", bend(func(b []byte) { binary.LittleEndian.PutUint16(b[26:], 2) })},
		{"eight bits per pixel", bend(func(b []byte) { binary.LittleEndian.PutUint16(b[28:], 8) })},
		{"run-length compression", bend(func(b []byte) { binary.LittleEndian.PutUint32(b[30:], 1) })},
		{"overflowing dimensions", bend(func(b []byte) {
			binary.LittleEndian.PutUint32(b[18:], 0x7fffffff)
			binary.LittleEndian.PutUint32(b[22:], 0x7fffffff)
		})},
		{"a pixel run cut short", ok[:len(ok)-1]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if im, err := Decode(tc.data); err == nil {
				t.Errorf("decoded %dx%d, want a refusal", im.Width, im.Height)
			}
		})
	}
}

// The two geometries this corpus actually ships, decoded end to end: a portrait
// node and the spell icon atlas (`SPR256-PICT-043`, `MAGIC-ICON-024`). The
// pixels are synthetic; only the extents are the shipped ones.
func TestDecodeTheTwoShippedGeometries(t *testing.T) {
	for _, tc := range []struct {
		name string
		w, h int
	}{
		{"a portrait node", 160, 240},
		{"the spell icon atlas", 480, 85},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows := make([][]Color, tc.h)
			for y := range rows {
				rows[y] = make([]Color, tc.w)
				rows[y][tc.w-1] = Color{R: uint8(y), G: uint8(y >> 8), B: 7}
			}
			im, err := Decode(build(t, rows, 0, 2))
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if im.Width != tc.w || im.Height != tc.h {
				t.Fatalf("decoded %dx%d, want %dx%d", im.Width, im.Height, tc.w, tc.h)
			}
			for y := 0; y < tc.h; y++ {
				want := Color{R: uint8(y), G: uint8(y >> 8), B: 7}
				if got := im.At(tc.w-1, y); got != want {
					t.Fatalf("row %d ends %+v, want %+v", y, got, want)
				}
			}
		})
	}
}

// At is TOTAL: no coordinate panics and everything outside the picture is the
// zero Color, which is what lets a cell cut out of an atlas be written without
// a bounds test at every read.
func TestAtIsTotal(t *testing.T) {
	im, err := Decode(build(t, [][]Color{{{R: 5}}}, 0, 0))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	for _, p := range [][2]int{{-1, 0}, {0, -1}, {1, 0}, {0, 1}, {99, 99}} {
		if got := (im.At(p[0], p[1])); got != (Color{}) {
			t.Errorf("At(%d,%d) = %+v, want the zero colour", p[0], p[1], got)
		}
	}
	var nilIm *Image
	if got := nilIm.At(0, 0); got != (Color{}) {
		t.Errorf("a nil image answered %+v", got)
	}
}

// The public format's own variants are read, not refused: a negative height
// stores rows top-down, the pixel run may start past the headers, and a 24-bit
// file may declare a colour table nothing reads.
func TestDecodeReadsTheFormatsOwnVariants(t *testing.T) {
	rows := [][]Color{{{R: 1}, {G: 2}}, {{B: 3}, {R: 4, G: 4}}}
	bottomUp := build(t, rows, 0, 0)

	topDown := append([]byte(nil), bottomUp[:HeaderLen]...)
	binary.LittleEndian.PutUint32(topDown[22:], ^uint32(1)) // height -2
	topDown = append(topDown, bottomUp[HeaderLen+8:]...)    // stored row 1 is the top
	topDown = append(topDown, bottomUp[HeaderLen:HeaderLen+8]...)

	later := append([]byte(nil), bottomUp[:HeaderLen]...)
	binary.LittleEndian.PutUint32(later[10:], HeaderLen+6)
	later = append(append(later, 9, 9, 9, 9, 9, 9), bottomUp[HeaderLen:]...)

	table := append([]byte(nil), bottomUp...)
	binary.LittleEndian.PutUint32(table[46:], 256)

	for name, data := range map[string][]byte{"top-down": topDown, "later pixels": later, "declared table": table} {
		im, err := Decode(data)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for y, row := range rows {
			for x, want := range row {
				if got := im.At(x, y); got != want {
					t.Fatalf("%s: pixel (%d,%d) = %+v, want %+v", name, x, y, got, want)
				}
			}
		}
	}
}

// RGBA and SubRGBA are the colour grid at full opacity; a cell outside the
// image is opaque black.
func TestRGBAIsOpaqueAndSubRGBACutsACell(t *testing.T) {
	im, err := Decode(build(t, [][]Color{{{R: 1}, {G: 2}}, {{B: 3}, {R: 4}}}, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	pic := im.RGBA()
	if got := pic.RGBAAt(1, 1); got != (color.RGBA{R: 4, A: 0xff}) {
		t.Fatalf("RGBA (1,1) = %+v", got)
	}
	cell := im.SubRGBA(image.Rect(1, 0, 3, 1))
	if cell.Bounds() != image.Rect(0, 0, 2, 1) {
		t.Fatalf("cell bounds %v", cell.Bounds())
	}
	if a, b := cell.RGBAAt(0, 0), cell.RGBAAt(1, 0); a != (color.RGBA{G: 2, A: 0xff}) || b != (color.RGBA{A: 0xff}) {
		t.Fatalf("cell = %+v %+v", a, b)
	}
	if got, err := DecodeRGBA(build(t, [][]Color{{{R: 7}}}, 0, 0)); err != nil || got.RGBAAt(0, 0) != (color.RGBA{R: 7, A: 0xff}) {
		t.Fatalf("DecodeRGBA = %v, %v", got, err)
	}
	var nilIm *Image
	if nilIm.RGBA() != nil {
		t.Fatal("a nil image converted")
	}
}
