package terrain_test

import (
	"encoding/binary"
	"image"
	"image/color"
)

// --- synthetic fixture builders (no game bytes anywhere) ---

// bmpPixelOffset is where buildBMP8 puts the pixel data: 14-byte file header +
// 40-byte DIB header + a full 256-entry palette, i.e. the bfOffBits = 1078 the
// real terrain tiles use.
const bmpPixelOffset = 14 + 40 + 256*4

// buildBMP8 assembles a byte-exact uncompressed 8-bit Windows BMP from an RGB
// palette and a top-down index grid. Rows are written bottom-up and padded to a
// 4-byte boundary, exactly as the format requires, so the decoder's row
// ordering and padding handling are both exercised by construction.
func buildBMP8(w, h int, palRGB [][3]byte, topDown []byte) []byte {
	stride := (w + 3) &^ 3
	out := make([]byte, bmpPixelOffset+stride*h)

	out[0x00], out[0x01] = 'B', 'M'
	binary.LittleEndian.PutUint32(out[0x02:], uint32(len(out)))       // bfSize
	binary.LittleEndian.PutUint32(out[0x0a:], uint32(bmpPixelOffset)) // bfOffBits
	binary.LittleEndian.PutUint32(out[0x0e:], 40)                     // DIB header size
	binary.LittleEndian.PutUint32(out[0x12:], uint32(int32(w)))       // width
	binary.LittleEndian.PutUint32(out[0x16:], uint32(int32(h)))       // height (positive => bottom-up)
	binary.LittleEndian.PutUint16(out[0x1a:], 1)                      // planes
	binary.LittleEndian.PutUint16(out[0x1c:], 8)                      // bits per pixel
	binary.LittleEndian.PutUint32(out[0x1e:], 0)                      // BI_RGB
	binary.LittleEndian.PutUint32(out[0x2e:], 0)                      // clrUsed 0 => 256

	// Palette entries are stored B, G, R, X.
	for i, c := range palRGB {
		e := out[54+i*4:]
		e[0], e[1], e[2] = c[2], c[1], c[0]
	}
	for y := 0; y < h; y++ {
		copy(out[bmpPixelOffset+(h-1-y)*stride:], topDown[y*w:(y+1)*w])
	}
	return out
}

// rampPalette returns a 256-entry palette whose entry i is a distinct RGB
// triple, so a decoded pixel identifies its source index unambiguously.
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

// solidStrip builds a 32 x (32*cells) tile strip in which sub-cell k is filled
// with palette index fill(k) — the shape of every terrain tile file.
func solidStrip(cells int, fill func(k int) byte) []byte {
	const size = 32
	h := size * cells
	idx := make([]byte, size*h)
	for k := 0; k < cells; k++ {
		for y := k * size; y < (k+1)*size; y++ {
			for x := 0; x < size; x++ {
				idx[y*size+x] = fill(k)
			}
		}
	}
	return buildBMP8(size, h, rampPalette(), idx)
}

// halfStrip builds a strip whose fill varies across the row, so a fixture can
// mix transparent (index 0) and opaque pixels inside one sub-cell.
func halfStrip(cells int, fill func(k, x int) byte) []byte {
	const size = 32
	h := size * cells
	idx := make([]byte, size*h)
	for k := 0; k < cells; k++ {
		for y := k * size; y < (k+1)*size; y++ {
			for x := 0; x < size; x++ {
				idx[y*size+x] = fill(k, x)
			}
		}
	}
	return buildBMP8(size, h, rampPalette(), idx)
}

// pixColor resolves a paletted pixel to the RGBA the compositor would draw.
// Tests read decoded tiles by palette index now, so this keeps the assertions
// written in colours.
func pixColor(img *image.Paletted, x, y int) color.RGBA {
	return rampColor(img.ColorIndexAt(x, y))
}
