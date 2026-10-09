package bmp_test

import (
	"encoding/binary"
	"image"
	"image/color"
	"testing"

	"againrom/pkg/formats/bmp"
)

// bmpPixelOffset is where buildBMP8 puts the pixel data: the two headers and a
// full 256-entry palette, the bfOffBits = 1078 the shipped terrain tiles use.
const bmpPixelOffset = 14 + 40 + 256*4

// buildBMP8 assembles an uncompressed 8-bit bitmap from an RGB palette and a
// top-down index grid, rows stored bottom-up and padded to four bytes.
func buildBMP8(w, h int, palRGB [][3]byte, topDown []byte) []byte {
	stride := (w + 3) &^ 3
	out := make([]byte, bmpPixelOffset+stride*h)
	out[0x00], out[0x01] = 'B', 'M'
	binary.LittleEndian.PutUint32(out[0x02:], uint32(len(out)))
	binary.LittleEndian.PutUint32(out[0x0a:], uint32(bmpPixelOffset))
	binary.LittleEndian.PutUint32(out[0x0e:], 40)
	binary.LittleEndian.PutUint32(out[0x12:], uint32(int32(w)))
	binary.LittleEndian.PutUint32(out[0x16:], uint32(int32(h)))
	binary.LittleEndian.PutUint16(out[0x1a:], 1)
	binary.LittleEndian.PutUint16(out[0x1c:], 8)
	for i, c := range palRGB {
		e := out[54+i*4:]
		e[0], e[1], e[2] = c[2], c[1], c[0]
	}
	for y := 0; y < h; y++ {
		copy(out[bmpPixelOffset+(h-1-y)*stride:], topDown[y*w:(y+1)*w])
	}
	return out
}

func rampPalette() [][3]byte {
	pal := make([][3]byte, 256)
	for i := range pal {
		pal[i] = [3]byte{byte(i), byte(255 - i), byte(i*7 + 3)}
	}
	return pal
}

func rampColor(idx byte) color.RGBA {
	return color.RGBA{R: idx, G: 255 - idx, B: idx*7 + 3, A: 0xff}
}

func pixColor(img *image.Paletted, x, y int) color.RGBA {
	return img.Palette[img.ColorIndexAt(x, y)].(color.RGBA)
}

// i32bits reinterprets a signed 32-bit header field as the u32 that is stored.
// Written as a function so a negative value is not a constant conversion.
func i32bits(v int32) uint32 { return uint32(v) }

// decodeReject asserts that a byte stream is rejected atomically: an error
// and no image.
func decodeReject(t *testing.T, name string, data []byte) {
	t.Helper()
	img, err := bmp.DecodePaletted(data)
	if err == nil {
		t.Fatalf("%s: expected an error, got nil", name)
	}
	if img != nil {
		t.Fatalf("%s: expected a nil image on error, got %v", name, img.Bounds())
	}
}

// --- tests ---

func TestDecodePalettedPixels(t *testing.T) {
	const w, h = 3, 2
	topDown := []byte{1, 2, 3, 4, 5, 6} // row 0 (top) then row 1 (bottom)
	data := buildBMP8(w, h, rampPalette(), topDown)

	img, err := bmp.DecodePaletted(data)
	if err != nil {
		t.Fatalf("decode: unexpected error: %v", err)
	}
	if got := img.Bounds(); got.Dx() != w || got.Dy() != h {
		t.Fatalf("bounds = %v, want %dx%d", got, w, h)
	}
	if got, want := len(img.Pix), w*h; got != want {
		t.Fatalf("len(Pix) = %d, want %d (one index per pixel)", got, want)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := topDown[y*w+x]
			if got, want := pixColor(img, x, y), rampColor(idx); got != want {
				t.Fatalf("pixel (%d,%d) = %v, want %v (palette index %d)", x, y, got, want, idx)
			}
		}
	}

	// Top-down ordering is the discriminating check: the file stores row 1
	// first, so a decoder that skipped the flip would put index 4 at (0,0).
	if got := pixColor(img, 0, 0); got != rampColor(1) {
		t.Fatalf("top-left = %v, want the first top-down row (index 1), not the first stored row", got)
	}
}

// TestDecodePalettedRejectsOutsideSubset - AC-2: a depth other than 8, a
// compression other than BI_RGB, a truncated palette and truncated pixel data
// each reject with no image.
func TestDecodePalettedRejectsOutsideSubset(t *testing.T) {
	valid := buildBMP8(3, 2, rampPalette(), []byte{1, 2, 3, 4, 5, 6})

	badDepth := append([]byte(nil), valid...)
	binary.LittleEndian.PutUint16(badDepth[0x1c:], 24)
	decodeReject(t, "bpp != 8", badDepth)

	badComp := append([]byte(nil), valid...)
	binary.LittleEndian.PutUint32(badComp[0x1e:], 1) // BI_RLE8
	decodeReject(t, "compression != BI_RGB", badComp)

	// Cut the stream mid-palette: the declared 256 entries no longer fit.
	decodeReject(t, "truncated palette", append([]byte(nil), valid[:54+400]...))

	// Keep the palette but drop a pixel row.
	shortPixels := append([]byte(nil), valid[:len(valid)-4]...)
	decodeReject(t, "truncated pixel data", shortPixels)

	// A pixel offset pointing past EOF, and one landing inside the palette.
	badOff := append([]byte(nil), valid...)
	binary.LittleEndian.PutUint32(badOff[0x0a:], uint32(len(valid)+1))
	decodeReject(t, "pixel offset past EOF", badOff)

	intoPalette := append([]byte(nil), valid...)
	binary.LittleEndian.PutUint32(intoPalette[0x0a:], 60)
	decodeReject(t, "pixel offset inside the palette", intoPalette)

	// A palette larger than 8 bpp can index.
	bigPalette := append([]byte(nil), valid...)
	binary.LittleEndian.PutUint32(bigPalette[0x2e:], 257)
	decodeReject(t, "clrUsed > 256", bigPalette)
}

func TestDecodePalettedRejectsMalformedHeader(t *testing.T) {
	valid := buildBMP8(3, 2, rampPalette(), []byte{1, 2, 3, 4, 5, 6})

	decodeReject(t, "empty", nil)
	decodeReject(t, "two bytes", []byte{'B', 'M'})
	decodeReject(t, "one byte short of a header", append([]byte(nil), valid[:53]...))

	badMagic := append([]byte(nil), valid...)
	badMagic[0], badMagic[1] = 'M', 'B'
	decodeReject(t, "bad magic", badMagic)

	badDIB := append([]byte(nil), valid...)
	binary.LittleEndian.PutUint32(badDIB[0x0e:], 12) // BITMAPCOREHEADER
	decodeReject(t, "DIB size != 40", badDIB)

	zeroHeight := append([]byte(nil), valid...)
	binary.LittleEndian.PutUint32(zeroHeight[0x16:], 0)
	decodeReject(t, "height 0", zeroHeight)

	negWidth := append([]byte(nil), valid...)
	binary.LittleEndian.PutUint32(negWidth[0x12:], i32bits(-3))
	decodeReject(t, "negative width", negWidth)

	// A width/height pair whose product would overflow a naive 32-bit bound.
	huge := append([]byte(nil), valid...)
	binary.LittleEndian.PutUint32(huge[0x12:], 0x7fffffff)
	binary.LittleEndian.PutUint32(huge[0x16:], 0x7fffffff)
	decodeReject(t, "overflowing dimensions", huge)
}

func TestDecodePalettedNeverPanics(t *testing.T) {
	valid := buildBMP8(8, 4, rampPalette(), make([]byte, 8*4))

	try := func(name string, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("%s: decoder panicked: %v", name, r)
			}
		}()
		img, err := bmp.DecodePaletted(data)
		switch {
		case err != nil && img != nil:
			t.Fatalf("%s: error and a non-nil image", name)
		case err == nil && img == nil:
			t.Fatalf("%s: no error and a nil image", name)
		case err == nil:
			b := img.Bounds()
			if len(img.Pix) != b.Dx()*b.Dy() {
				t.Fatalf("%s: %d index bytes for a %v image", name, len(img.Pix), b)
			}
		}
	}

	for n := 0; n <= len(valid); n++ {
		try("prefix", valid[:n])
	}
	for off := 0; off < 54; off++ {
		for _, b := range []byte{0x00, 0x01, 0x7f, 0x80, 0xff} {
			m := append([]byte(nil), valid...)
			m[off] = b
			try("header mutation", m)
		}
	}
	for _, g := range [][]byte{
		{},
		{0x00},
		make([]byte, 54),
		make([]byte, 1078),
		[]byte("BM not really a bitmap at all, just some ASCII filler bytes."),
	} {
		try("garbage", g)
	}
}
