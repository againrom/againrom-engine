package bmp

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"

	"againrom/pkg/formats/pal"
)

// The header geometry: a 14-byte file header and a 40-byte BITMAPINFOHEADER,
// the only DIB header accepted. An 8-bit colour table starts right after it.
const (
	fileHeaderLen = 14
	infoHeaderLen = 40

	// HeaderLen is where the colour table, or a 24-bit pixel run, begins.
	HeaderLen = fileHeaderLen + infoHeaderLen

	// BitsPerPixel is the depth Decode reads: three bytes a pixel, stored
	// blue-green-red, with no alpha.
	BitsPerPixel = 24

	// IndexedBitsPerPixel is the depth DecodePaletted reads: one palette index
	// a pixel.
	IndexedBitsPerPixel = 8

	maxPalette = 256
)

// Color is one 24-bit pixel: pal.Color, since the stored order is the palette
// layout's own blue-green-red.
type Color = pal.Color

// Image is a decoded 24-bit bitmap: its extent and one Color per pixel,
// row-major, THE TOP ROW FIRST, whatever the stored row order.
type Image struct {
	Width, Height int
	Pix           []Color
}

// At is the pixel at (x, y), origin top-left, and the zero Color for a
// coordinate outside the image, so a caller cutting a sub-rectangle out of an
// atlas cannot index past the end of one.
func (im *Image) At(x, y int) Color {
	if im == nil || x < 0 || y < 0 || x >= im.Width || y >= im.Height {
		return Color{}
	}
	return im.Pix[y*im.Width+x]
}

// RGBA is the image at full opacity: the format stores no alpha.
func (im *Image) RGBA() *image.RGBA {
	if im == nil || im.Width <= 0 || im.Height <= 0 {
		return nil
	}
	return im.SubRGBA(image.Rect(0, 0, im.Width, im.Height))
}

// SubRGBA is the rectangle r of the image at full opacity, on an image of r's
// size with its origin at zero. Pixels of r outside the image are opaque black.
func (im *Image) SubRGBA(r image.Rectangle) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	for y := 0; y < r.Dy(); y++ {
		for x := 0; x < r.Dx(); x++ {
			c := im.At(r.Min.X+x, r.Min.Y+y)
			o := y*pic.Stride + x*4
			pic.Pix[o], pic.Pix[o+1], pic.Pix[o+2], pic.Pix[o+3] = c.R, c.G, c.B, 0xff
		}
	}
	return pic
}

// header is a validated header: the accepted subset, range-checked against
// the stream.
type header struct {
	width, height int
	bottomUp      bool
	stride        int
	offBits       int
	palette       []Color
}

// row is the stored bytes of display row y.
func (h header) row(data []byte, y int) []byte {
	if h.bottomUp {
		y = h.height - 1 - y
	}
	return data[h.offBits+y*h.stride:]
}

// parse validates the header of an uncompressed Windows bitmap at depth bpp.
//
// The pixel run is located through the file's own offset, its length is
// computed from the width, height and 4-byte row stride and never read from
// the header, and a stream longer than the run is accepted, its tail ignored:
// shipped bitmaps carry two bytes past their pixels (see doc.go). A positive
// height stores rows bottom-up and a negative one top-down; both are the public
// format's rule. At 8 bits a colour table of clrUsed entries (0 meaning 256)
// follows the header; at 24 bits there is none to read.
func parse(data []byte, bpp int) (header, error) {
	var h header
	if len(data) < HeaderLen {
		return h, fmt.Errorf("bmp: %d bytes is shorter than the %d-byte header", len(data), HeaderLen)
	}
	if data[0] != 'B' || data[1] != 'M' {
		return h, fmt.Errorf("bmp: magic %q, want \"BM\"", data[0:2])
	}
	u16 := func(off int) int { return int(binary.LittleEndian.Uint16(data[off:])) }
	u32 := func(off int) uint32 { return binary.LittleEndian.Uint32(data[off:]) }
	if n := u32(14); n != infoHeaderLen {
		return h, fmt.Errorf("bmp: info header is %d bytes, want %d", n, infoHeaderLen)
	}
	if planes := u16(26); planes != 1 {
		return h, fmt.Errorf("bmp: %d colour planes, want 1", planes)
	}
	if got := u16(28); got != bpp {
		return h, fmt.Errorf("bmp: %d bits per pixel, want %d", got, bpp)
	}
	if c := u32(30); c != 0 {
		return h, fmt.Errorf("bmp: compression %d, want 0 (none)", c)
	}
	w, storedH := int64(int32(u32(18))), int64(int32(u32(22)))
	if w <= 0 {
		return h, fmt.Errorf("bmp: width %d", w)
	}
	if storedH == 0 {
		return h, fmt.Errorf("bmp: height 0")
	}
	h.bottomUp = storedH > 0
	if storedH < 0 {
		storedH = -storedH
	}

	total := int64(len(data))
	tableEnd := int64(HeaderLen)
	if bpp == IndexedBitsPerPixel {
		n := u32(46)
		if n == 0 {
			n = maxPalette
		}
		if n > maxPalette {
			return h, fmt.Errorf("bmp: palette declares %d entries, want at most %d", n, maxPalette)
		}
		tableEnd += int64(n) * pal.EntrySize
		if tableEnd > total {
			return h, fmt.Errorf("bmp: palette [%d, %d) overruns the %d-byte stream", HeaderLen, tableEnd, total)
		}
		h.palette = pal.Entries(data[HeaderLen:tableEnd])
	}
	off := int64(u32(10))
	if off < tableEnd || off > total {
		return h, fmt.Errorf("bmp: pixel data at %d, outside [%d, %d]", off, tableEnd, total)
	}
	stride := (w*int64(bpp/8) + 3) &^ 3
	// Compared by division, so no product of two header fields can overflow
	// into a length that looks present.
	if avail := total - off; storedH > avail/stride {
		return h, fmt.Errorf("bmp: %dx%d needs %d-byte rows, %d bytes present", w, storedH, stride, avail)
	}
	h.width, h.height, h.stride, h.offBits = int(w), int(storedH), int(stride), int(off)
	return h, nil
}

// Decode reads one uncompressed 24-bit Windows bitmap into its colour grid.
// Every other shape is refused by name, never decoded into a partial image.
func Decode(data []byte) (*Image, error) {
	h, err := parse(data, BitsPerPixel)
	if err != nil {
		return nil, err
	}
	im := &Image{Width: h.width, Height: h.height, Pix: make([]Color, h.width*h.height)}
	for y := 0; y < h.height; y++ {
		row := h.row(data, y)
		copy(im.Pix[y*h.width:(y+1)*h.width], pal.Pixels(row[:h.width*3]))
	}
	return im, nil
}

// DecodeRGBA is Decode at full opacity.
func DecodeRGBA(data []byte) (*image.RGBA, error) {
	im, err := Decode(data)
	if err != nil {
		return nil, err
	}
	return im.RGBA(), nil
}

// DecodePaletted reads one uncompressed 8-bit Windows bitmap and keeps its
// palette indices: a hit mask and a terrain tile are addressed by index, not
// colour. The colour table rides along at full opacity. A pixel naming an
// entry past the stored table is refused rather than clamped.
func DecodePaletted(data []byte) (*image.Paletted, error) {
	h, err := parse(data, IndexedBitsPerPixel)
	if err != nil {
		return nil, err
	}
	palette := make(color.Palette, len(h.palette))
	for i, c := range h.palette {
		palette[i] = c.Opaque()
	}
	img := image.NewPaletted(image.Rect(0, 0, h.width, h.height), palette)
	for y := 0; y < h.height; y++ {
		row := h.row(data, y)
		dst := img.Pix[y*img.Stride:]
		for x := 0; x < h.width; x++ {
			if int(row[x]) >= len(palette) {
				return nil, fmt.Errorf("bmp: pixel (%d,%d) index %d outside the %d-entry palette", x, y, row[x], len(palette))
			}
			dst[x] = row[x]
		}
	}
	return img, nil
}
