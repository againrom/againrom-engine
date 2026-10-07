package bmp

import (
	"encoding/binary"
	"fmt"
)

// The header geometry, named rather than spelled into the reader as offsets.
// These are the two structures a `dataOff=54` file has and nothing else: a
// 14-byte file header and a 40-byte BITMAPINFOHEADER immediately after it.
const (
	fileHeaderLen = 14
	infoHeaderLen = 40

	// HeaderLen is where the pixel run begins in every shipped node
	// (`SPR256-PICT-043`: `dataOff=54` on all of them). It is not assumed —
	// the file's own dataOff field is read and compared against it — but it is
	// the only value this decoder accepts, because a different one means a
	// header version this package does not carry.
	HeaderLen = fileHeaderLen + infoHeaderLen

	// BitsPerPixel is the one depth this corpus ships. Three bytes a pixel,
	// stored blue-green-red, with no palette and no alpha anywhere.
	BitsPerPixel = 24
)

// Color is one pixel. The file stores blue, green and red in that order and
// this type carries them the other way round, which is the one reordering this
// package performs on a pixel.
type Color struct {
	R, G, B uint8
}

// Image is a decoded bitmap: its extent and one Color per pixel, row-major,
// THE TOP ROW FIRST.
//
// THE ROW ORDER IS FLIPPED AND THAT IS THE SECOND OF THIS PACKAGE'S TWO
// CONVERSIONS. A Windows bitmap with a positive height stores its rows bottom
// to top, which every shipped node is (`h=240`, `h=85`); a consumer that wants
// a picture wants it the other way, and doing it here means it is done once
// rather than by each of the two callers this format has.
type Image struct {
	Width, Height int
	Pix           []Color
}

// At is the pixel at (x, y), origin top-left, and the zero Color for a
// coordinate outside the image — the same total reading spr256's own frame
// accessor gives, so a caller cutting a sub-rectangle out of an atlas cannot
// index past the end of one.
func (im *Image) At(x, y int) Color {
	if im == nil || x < 0 || y < 0 || x >= im.Width || y >= im.Height {
		return Color{}
	}
	return im.Pix[y*im.Width+x]
}

// Decode reads one 24-bit uncompressed Windows bitmap.
//
// IT REFUSES EVERY OTHER SHAPE BY NAME, and the errors say which field was
// wrong rather than "bad bitmap": this decoder exists for a corpus whose whole
// header set was measured, so an unexpected value is evidence that the corpus
// widened and the reader should say exactly how.
//
// A file longer than its own pixel run is ACCEPTED, tail ignored — see the
// package doc for the two bytes that make that the ordinary case rather than
// the exception.
func Decode(data []byte) (*Image, error) {
	if len(data) < HeaderLen {
		return nil, fmt.Errorf("bmp: %d bytes is shorter than the %d-byte header", len(data), HeaderLen)
	}
	if data[0] != 'B' || data[1] != 'M' {
		return nil, fmt.Errorf("bmp: magic %q, want \"BM\"", data[0:2])
	}

	u16 := func(off int) int { return int(binary.LittleEndian.Uint16(data[off:])) }
	u32 := func(off int) uint32 { return binary.LittleEndian.Uint32(data[off:]) }
	i32 := func(off int) int { return int(int32(binary.LittleEndian.Uint32(data[off:]))) }

	if off := u32(10); off != HeaderLen {
		return nil, fmt.Errorf("bmp: pixel data at %d, want %d", off, HeaderLen)
	}
	if n := u32(14); n != infoHeaderLen {
		return nil, fmt.Errorf("bmp: info header is %d bytes, want %d", n, infoHeaderLen)
	}

	w, h := i32(18), i32(22)
	if w <= 0 {
		return nil, fmt.Errorf("bmp: width %d", w)
	}
	// A NEGATIVE HEIGHT IS THE TOP-DOWN FORM AND IS REFUSED RATHER THAN
	// HANDLED. No shipped node carries one, so the flip below would be the
	// only branch here with nothing to exercise it; a reader that silently
	// accepted it would be one whose row order is untested in half its domain.
	if h <= 0 {
		return nil, fmt.Errorf("bmp: height %d - a top-down bitmap is not a shape this corpus ships", h)
	}
	if planes := u16(26); planes != 1 {
		return nil, fmt.Errorf("bmp: %d colour planes, want 1", planes)
	}
	if bpp := u16(28); bpp != BitsPerPixel {
		return nil, fmt.Errorf("bmp: %d bits per pixel, want %d", bpp, BitsPerPixel)
	}
	if c := u32(30); c != 0 {
		return nil, fmt.Errorf("bmp: compression %d, want 0 (none)", c)
	}
	if n := u32(46); n != 0 {
		return nil, fmt.Errorf("bmp: %d palette entries, want 0", n)
	}

	// THE ROW STRIDE IS PADDED TO FOUR BYTES, which is the format's rule and
	// not this corpus's: at 24 bits a row of w pixels is 3w bytes and the next
	// row starts at the next multiple of four. Both shipped widths — 160 and
	// 480 — make 3w a multiple of four already, so the padding is zero on
	// everything that ships and is computed anyway, because a width that did
	// not would be read three bytes out of step per row with nothing to say so.
	stride := (w*3 + 3) &^ 3
	need := HeaderLen + stride*h
	if len(data) < need {
		return nil, fmt.Errorf("bmp: %dx%d needs %d bytes, have %d", w, h, need, len(data))
	}

	im := &Image{Width: w, Height: h, Pix: make([]Color, w*h)}
	for y := 0; y < h; y++ {
		// The flip: file row 0 is the picture's BOTTOM row.
		row := HeaderLen + (h-1-y)*stride
		out := y * w
		for x := 0; x < w; x++ {
			p := row + x*3
			im.Pix[out+x] = Color{R: data[p+2], G: data[p+1], B: data[p]}
		}
	}
	return im, nil
}
