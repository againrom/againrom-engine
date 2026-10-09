package bmp

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"testing"

	"againrom/internal/synth"
)

// --- the format contract, transcribed from the Windows BMP specification ---
//
// A BMP stream opens with a 14-byte BITMAPFILEHEADER ("BM", the file size, two
// reserved words and bfOffBits) followed by the DIB header. The only DIB header
// the shipped art uses is the 40-byte BITMAPINFOHEADER: biSize, biWidth and
// biHeight as signed 32-bit, biPlanes and biBitCount as 16-bit, biCompression,
// biSizeImage, two pels-per-metre fields, biClrUsed and biClrImportant.
//
// For an indexed depth the colour table follows the DIB header as BGRX quads,
// and biClrUsed == 0 means the full 2^biBitCount entries — so an 8-bpp file
// carries 256 quads and its pixel array can begin no earlier than
// 14 + 40 + 1024 = 1078. A 24-bpp file has no colour table, so its pixel array
// can begin no earlier than 54. bfOffBits, not any computed header size, is
// what locates the pixel array.
//
// Pixel rows are packed to whole bytes (three B,G,R bytes per pixel at 24 bpp,
// one index byte per pixel at 8 bpp) and each row is then padded to a 4-byte
// boundary; the padding is not pixel data. A positive biHeight means the rows
// are stored bottom-up (the first stored row is the bottom display row), a
// negative one that they are stored top-down.
const (
	btOffMagic       = 0x00 // 'B','M'
	btOffBfOffBits   = 0x0a // uint32: byte offset of the pixel array
	btOffDIBSize     = 0x0e // uint32: DIB header size; 40 == BITMAPINFOHEADER
	btOffWidth       = 0x12 // int32:  biWidth
	btOffHeight      = 0x16 // int32:  biHeight (>0 bottom-up, <0 top-down)
	btOffPlanes      = 0x1a // uint16: biPlanes, always 1
	btOffBitCount    = 0x1c // uint16: biBitCount
	btOffCompression = 0x1e // uint32: biCompression, 0 == BI_RGB
	btOffClrUsed     = 0x2e // uint32: biClrUsed, 0 == 2^biBitCount when indexed

	btFileHeader = 14
	btDIBHeader  = 40
	btHeaderLen  = btFileHeader + btDIBHeader // 54
	btPaletteLen = 256 * 4                    // a full 8-bpp colour table
	btPixels8    = btHeaderLen + btPaletteLen // 1078: earliest legal 8-bpp bfOffBits
	btPixels24   = btHeaderLen                // 54:   earliest legal 24-bpp bfOffBits
)

// --- expectations, computed here and nowhere else ---

// btWantColor is the colour of source pixel (x, y). Every pixel of a fixture is
// distinct and every row differs from every other, so a transposed, mirrored or
// stride-shifted read cannot accidentally agree with it.
func btWantColor(x, y int) color.RGBA {
	return color.RGBA{
		R: uint8(0x11 + 0x20*x),
		G: uint8(0x8f - 0x11*y),
		B: uint8(0x03 + 0x40*y + x),
		A: 0xff, // 24-bpp BMP carries no alpha channel: every decoded pixel is opaque
	}
}

// btWantIndex is the raw palette index of source pixel (x, y). The values are
// spread over 0x80…0xc2 so none of them collides with the padding sentinel the
// padding test writes.
func btWantIndex(x, y int) uint8 { return uint8(0x80 + 0x10*x + y) }

func btRGBASource(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, btWantColor(x, y))
		}
	}
	return img
}

// btPalettedSource carries btWantIndex at every pixel as a *raw* index. The
// palette is a caller's choice precisely because it must not influence what a
// reader reports.
func btPalettedSource(w, h int, pal color.Palette) *image.Paletted {
	img := image.NewPaletted(image.Rect(0, 0, w, h), pal)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Pix[img.PixOffset(x, y)] = btWantIndex(x, y)
		}
	}
	return img
}

// --- stream surgery ---

func btMutate(src []byte, f func(b []byte)) []byte {
	out := append([]byte(nil), src...)
	f(out)
	return out
}

func btPut32(b []byte, off int, v uint32) { binary.LittleEndian.PutUint32(b[off:off+4], v) }
func btPut16(b []byte, off int, v uint16) { binary.LittleEndian.PutUint16(b[off:off+2], v) }

// btI32 reinterprets a signed header field as the uint32 that is stored, as a
// function so a negative value is not a constant conversion.
func btI32(v int32) uint32 { return uint32(v) }

// btFillPadding overwrites every row-padding byte of a stream with sentinel.
// Row bytes and stride are computed from the standard's own rule, and the pixel
// array is located through bfOffBits. A reader that copies whole stride-length
// rows into the image, or that treats the padding as pixels, then leaks a
// recognisable value.
func btFillPadding(stream []byte, w, h, bpp int, sentinel byte) []byte {
	rowBytes := (w*bpp + 7) / 8
	stride := (rowBytes + 3) &^ 3
	off := int(binary.LittleEndian.Uint32(stream[btOffBfOffBits : btOffBfOffBits+4]))
	out := append([]byte(nil), stream...)
	for r := 0; r < h; r++ {
		for i := rowBytes; i < stride; i++ {
			out[off+r*stride+i] = sentinel
		}
	}
	return out
}

// --- assertions ---

func btCheck24(t *testing.T, name string, img *image.RGBA, w, h int) {
	t.Helper()
	if img == nil {
		t.Fatalf("%s: nil image", name)
	}
	b := img.Bounds()
	if b.Dx() != w || b.Dy() != h {
		t.Fatalf("%s: bounds %v is %dx%d, want %dx%d", name, b, b.Dx(), b.Dy(), w, h)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			got := img.RGBAAt(b.Min.X+x, b.Min.Y+y)
			if want := btWantColor(x, y); got != want {
				t.Fatalf("%s: pixel (%d,%d) = %+v, want %+v", name, x, y, got, want)
			}
		}
	}
}

func btCheck8(t *testing.T, name string, img *image.Paletted, w, h int) {
	t.Helper()
	if img == nil {
		t.Fatalf("%s: nil image", name)
	}
	b := img.Bounds()
	if b.Dx() != w || b.Dy() != h {
		t.Fatalf("%s: bounds %v is %dx%d, want %dx%d", name, b, b.Dx(), b.Dy(), w, h)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			got := img.Pix[img.PixOffset(b.Min.X+x, b.Min.Y+y)]
			if want := btWantIndex(x, y); got != want {
				t.Fatalf("%s: index at (%d,%d) = %#02x, want %#02x", name, x, y, got, want)
			}
		}
	}
}

// btReject24 asserts that DecodeRGBA refuses a stream atomically: a non-nil
// error, a nil image, and no panic.
func btReject24(t *testing.T, name string, data []byte) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("%s: DecodeRGBA panicked: %v", name, r)
		}
	}()
	img, err := DecodeRGBA(data)
	if err == nil {
		t.Errorf("%s: DecodeRGBA accepted the stream, want an error", name)
		return
	}
	if img != nil {
		t.Errorf("%s: DecodeRGBA reported an error and a non-nil image %v", name, img.Bounds())
	}
}

func btReject8(t *testing.T, name string, data []byte) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("%s: DecodePaletted panicked: %v", name, r)
		}
	}()
	img, err := DecodePaletted(data)
	if err == nil {
		t.Errorf("%s: DecodePaletted accepted the stream, want an error", name)
		return
	}
	if img != nil {
		t.Errorf("%s: DecodePaletted reported an error and a non-nil image %v", name, img.Bounds())
	}
}

func btDecode24(t *testing.T, name string, data []byte) *image.RGBA {
	t.Helper()
	img, err := DecodeRGBA(data)
	if err != nil {
		t.Fatalf("%s: DecodeRGBA: %v", name, err)
	}
	if img == nil {
		t.Fatalf("%s: DecodeRGBA returned a nil image and a nil error", name)
	}
	return img
}

func btDecode8(t *testing.T, name string, data []byte) *image.Paletted {
	t.Helper()
	img, err := DecodePaletted(data)
	if err != nil {
		t.Fatalf("%s: DecodePaletted: %v", name, err)
	}
	if img == nil {
		t.Fatalf("%s: DecodePaletted returned a nil image and a nil error", name)
	}
	return img
}

// TestDecodeBMP — T2: the two bitmap readers against the Windows BMP contract
// transcribed above. Dimensions are deliberately non-square and deliberately
// unaligned (3 px at 24 bpp needs 3 padding bytes per row, 5 px at 8 bpp needs
// 3), so a width/height transposition and a padding leak are both caught.
func TestDecodeBMP(t *testing.T) {
	const (
		w24, h24 = 3, 2 // 9 pixel bytes per row, stride 12
		w8, h8   = 5, 3 // 5 pixel bytes per row, stride 8
	)
	src24 := btRGBASource(w24, h24)
	src8 := btPalettedSource(w8, h8, synth.GrayRamp())
	valid24 := synth.BMP24(src24)
	valid8 := synth.BMP8(src8)

	// A fixture whose declared geometry is not what the test expects would make
	// every assertion below meaningless, so the declared fields are read back
	// from the stream first.
	t.Run("fixture geometry", func(t *testing.T) {
		for _, c := range []struct {
			name             string
			data             []byte
			w, h, bpp, off   int
			wantLen, wantDIB int
		}{
			{"24 bpp", valid24, w24, h24, 24, btPixels24, btPixels24 + 12*h24, btDIBHeader},
			{"8 bpp", valid8, w8, h8, 8, btPixels8, btPixels8 + 8*h8, btDIBHeader},
		} {
			if got := int(binary.LittleEndian.Uint32(c.data[btOffWidth:])); got != c.w {
				t.Errorf("%s: declared width %d, want %d", c.name, got, c.w)
			}
			if got := int(int32(binary.LittleEndian.Uint32(c.data[btOffHeight:]))); got != c.h {
				t.Errorf("%s: declared height %d, want %d (positive == bottom-up)", c.name, got, c.h)
			}
			if got := int(binary.LittleEndian.Uint16(c.data[btOffBitCount:])); got != c.bpp {
				t.Errorf("%s: declared bpp %d, want %d", c.name, got, c.bpp)
			}
			if got := int(binary.LittleEndian.Uint32(c.data[btOffBfOffBits:])); got != c.off {
				t.Errorf("%s: bfOffBits %d, want %d", c.name, got, c.off)
			}
			if got := int(binary.LittleEndian.Uint32(c.data[btOffDIBSize:])); got != c.wantDIB {
				t.Errorf("%s: DIB header size %d, want %d", c.name, got, c.wantDIB)
			}
			if len(c.data) != c.wantLen {
				t.Errorf("%s: stream is %d bytes, want %d (header + padded rows)", c.name, len(c.data), c.wantLen)
			}
		}
	})

	// A 24-bpp stream decodes to the declared dimensions with the expected
	// colour at every position, opaque everywhere.
	t.Run("24bpp pixels", func(t *testing.T) {
		btCheck24(t, "24 bpp", btDecode24(t, "24 bpp", valid24), w24, h24)
	})

	// An 8-bpp stream preserves the raw index at every position.
	t.Run("8bpp raw indices", func(t *testing.T) {
		img := btDecode8(t, "8 bpp", valid8)
		btCheck8(t, "8 bpp", img, w8, h8)
		if len(img.Palette) != 256 {
			t.Errorf("decoded palette has %d entries, want 256 (biClrUsed == 0 at 8 bpp)", len(img.Palette))
		}
	})

	// The discriminator: the same indices under a palette that carries no
	// distinguishable colour must decode to the same indices. A reader that
	// resolved indices through the palette would report the grey level under
	// GrayRamp (where index == grey level, so it looks right) and zero under
	// BlackPalette; it cannot satisfy both.
	t.Run("8bpp palette carries no meaning", func(t *testing.T) {
		grayStream := synth.BMP8(btPalettedSource(w8, h8, synth.GrayRamp()))
		blackStream := synth.BMP8(btPalettedSource(w8, h8, synth.BlackPalette()))
		if bytes.Equal(grayStream, blackStream) {
			t.Fatal("the two palettes produced identical streams: the check would be vacuous")
		}

		gray := btDecode8(t, "gray ramp", grayStream)
		black := btDecode8(t, "black palette", blackStream)
		btCheck8(t, "gray ramp", gray, w8, h8)
		btCheck8(t, "black palette", black, w8, h8)
		for y := 0; y < h8; y++ {
			for x := 0; x < w8; x++ {
				g := gray.Pix[gray.PixOffset(gray.Bounds().Min.X+x, gray.Bounds().Min.Y+y)]
				b := black.Pix[black.PixOffset(black.Bounds().Min.X+x, black.Bounds().Min.Y+y)]
				if g != b {
					t.Fatalf("index at (%d,%d): %#02x under the grey ramp, %#02x under the black palette", x, y, g, b)
				}
			}
		}
	})

	// Row order: positive height is bottom-up storage normalised to top-down
	// display rows, negative height is already top-down. The same pixels through
	// both must give the same top-down image.
	t.Run("row order 24bpp", func(t *testing.T) {
		bottomUp := synth.BMP24(src24)
		topDown := synth.BMP24TopDown(src24)
		if bytes.Equal(bottomUp, topDown) {
			t.Fatal("bottom-up and top-down streams are byte-identical: the row-order check would be vacuous")
		}
		if got := int32(binary.LittleEndian.Uint32(topDown[btOffHeight:])); got != -h24 {
			t.Fatalf("top-down stream declares height %d, want %d", got, -h24)
		}
		a := btDecode24(t, "positive height", bottomUp)
		b := btDecode24(t, "negative height", topDown)
		btCheck24(t, "positive height", a, w24, h24)
		btCheck24(t, "negative height", b, w24, h24)
		for y := 0; y < h24; y++ {
			for x := 0; x < w24; x++ {
				if p, q := a.RGBAAt(x, y), b.RGBAAt(x, y); p != q {
					t.Fatalf("pixel (%d,%d): %+v from the bottom-up stream, %+v from the top-down one", x, y, p, q)
				}
			}
		}
	})

	t.Run("row order 8bpp", func(t *testing.T) {
		bottomUp := synth.BMP8(src8)
		topDown := synth.BMP8TopDown(src8)
		if bytes.Equal(bottomUp, topDown) {
			t.Fatal("bottom-up and top-down streams are byte-identical: the row-order check would be vacuous")
		}
		if got := int32(binary.LittleEndian.Uint32(topDown[btOffHeight:])); got != -h8 {
			t.Fatalf("top-down stream declares height %d, want %d", got, -h8)
		}
		a := btDecode8(t, "positive height", bottomUp)
		b := btDecode8(t, "negative height", topDown)
		btCheck8(t, "positive height", a, w8, h8)
		btCheck8(t, "negative height", b, w8, h8)
		for y := 0; y < h8; y++ {
			for x := 0; x < w8; x++ {
				p := a.Pix[a.PixOffset(a.Bounds().Min.X+x, a.Bounds().Min.Y+y)]
				q := b.Pix[b.PixOffset(b.Bounds().Min.X+x, b.Bounds().Min.Y+y)]
				if p != q {
					t.Fatalf("index at (%d,%d): %#02x from the bottom-up stream, %#02x from the top-down one", x, y, p, q)
				}
			}
		}
	})

	// Every one of the eighteen shipped menu bitmaps carries exactly two bytes
	// beyond the size its header implies, so a reader that required the stream
	// length to equal the computed minimum would reject the whole asset set.
	t.Run("trailing bytes", func(t *testing.T) {
		btCheck24(t, "24 bpp + 2", btDecode24(t, "24 bpp + 2", synth.Trailing(valid24, 2)), w24, h24)
		btCheck8(t, "8 bpp + 2", btDecode8(t, "8 bpp + 2", synth.Trailing(valid8, 2)), w8, h8)
	})

	// Row padding is not pixel data at either depth.
	t.Run("row padding is not pixel data", func(t *testing.T) {
		const sentinel24, sentinel8 = 0xaa, 0x7f

		p24 := btFillPadding(valid24, w24, h24, 24, sentinel24)
		if bytes.Equal(p24, valid24) {
			t.Fatal("width 3 at 24 bpp must have 3 padding bytes per row; the fixture has none")
		}
		btCheck24(t, "padded 24 bpp", btDecode24(t, "padded 24 bpp", p24), w24, h24)

		p8 := btFillPadding(valid8, w8, h8, 8, sentinel8)
		if bytes.Equal(p8, valid8) {
			t.Fatal("width 5 at 8 bpp must have 3 padding bytes per row; the fixture has none")
		}
		img8 := btDecode8(t, "padded 8 bpp", p8)
		btCheck8(t, "padded 8 bpp", img8, w8, h8)
		for i, b := range img8.Pix {
			if b == sentinel8 {
				t.Fatalf("padding byte %#02x leaked into Pix[%d]", sentinel8, i)
			}
		}
	})

	// biClrUsed == 0 means "the maximum for the depth", i.e. the same 256-entry
	// table an explicit 256 names.
	t.Run("clrUsed 256 is the same table as 0", func(t *testing.T) {
		s := btMutate(valid8, func(b []byte) { btPut32(b, btOffClrUsed, 256) })
		btCheck8(t, "clrUsed 256", btDecode8(t, "clrUsed 256", s), w8, h8)
	})

	// Rejections: a nil image and a non-nil error, never a panic.
	t.Run("rejects", func(t *testing.T) {
		badMagic := func(b []byte) { b[btOffMagic], b[btOffMagic+1] = 'M', 'B' }

		for _, c := range []struct {
			name string
			data []byte
		}{
			{"nil", nil},
			{"empty", []byte{}},
			{"two bytes", []byte{'B', 'M'}},
			{"one byte short of the 54-byte header", valid24[:btHeaderLen-1]},
			{"bad magic", btMutate(valid24, badMagic)},
			{"DIB header size 12 (BITMAPCOREHEADER)", btMutate(valid24, func(b []byte) { btPut32(b, btOffDIBSize, 12) })},
			{"DIB header size 108 (BITMAPV4HEADER)", btMutate(valid24, func(b []byte) { btPut32(b, btOffDIBSize, 108) })},
			{"an 8-bpp stream", valid8},
			{"biBitCount 4", btMutate(valid24, func(b []byte) { btPut16(b, btOffBitCount, 4) })},
			{"biBitCount 32", btMutate(valid24, func(b []byte) { btPut16(b, btOffBitCount, 32) })},
			{"compression != BI_RGB", btMutate(valid24, func(b []byte) { btPut32(b, btOffCompression, 1) })},
			{"width 0", btMutate(valid24, func(b []byte) { btPut32(b, btOffWidth, 0) })},
			{"negative width", btMutate(valid24, func(b []byte) { btPut32(b, btOffWidth, btI32(-3)) })},
			{"height 0", btMutate(valid24, func(b []byte) { btPut32(b, btOffHeight, 0) })},
			{"pixel offset inside the header", btMutate(valid24, func(b []byte) { btPut32(b, btOffBfOffBits, btPixels24-4) })},
			{"pixel offset past EOF", btMutate(valid24, func(b []byte) { btPut32(b, btOffBfOffBits, uint32(len(valid24)+1)) })},
			{"truncated pixel data", valid24[:len(valid24)-4]},
		} {
			btReject24(t, "DecodeRGBA/"+c.name, c.data)
		}

		for _, c := range []struct {
			name string
			data []byte
		}{
			{"nil", nil},
			{"empty", []byte{}},
			{"two bytes", []byte{'B', 'M'}},
			{"one byte short of the 54-byte header", valid8[:btHeaderLen-1]},
			{"bad magic", btMutate(valid8, badMagic)},
			{"DIB header size 12 (BITMAPCOREHEADER)", btMutate(valid8, func(b []byte) { btPut32(b, btOffDIBSize, 12) })},
			{"DIB header size 108 (BITMAPV4HEADER)", btMutate(valid8, func(b []byte) { btPut32(b, btOffDIBSize, 108) })},
			{"a 24-bpp stream", valid24},
			{"biBitCount 4", btMutate(valid8, func(b []byte) { btPut16(b, btOffBitCount, 4) })},
			{"biBitCount 24", btMutate(valid8, func(b []byte) { btPut16(b, btOffBitCount, 24) })},
			{"compression != BI_RGB", btMutate(valid8, func(b []byte) { btPut32(b, btOffCompression, 1) })},
			{"width 0", btMutate(valid8, func(b []byte) { btPut32(b, btOffWidth, 0) })},
			{"negative width", btMutate(valid8, func(b []byte) { btPut32(b, btOffWidth, btI32(-5)) })},
			{"height 0", btMutate(valid8, func(b []byte) { btPut32(b, btOffHeight, 0) })},
			{"pixel offset at the palette start", btMutate(valid8, func(b []byte) { btPut32(b, btOffBfOffBits, btHeaderLen) })},
			{"pixel offset inside the palette", btMutate(valid8, func(b []byte) { btPut32(b, btOffBfOffBits, btPixels8-4) })},
			{"pixel offset past EOF", btMutate(valid8, func(b []byte) { btPut32(b, btOffBfOffBits, uint32(len(valid8)+1)) })},
			{"truncated pixel data", valid8[:len(valid8)-4]},
		} {
			btReject8(t, "DecodePaletted/"+c.name, c.data)
		}
	})
}
