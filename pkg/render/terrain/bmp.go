package terrain

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
)

// Layout constants for the 8-bit Windows BMP subset the terrain tiles use: a
// 14-byte BITMAPFILEHEADER, a 40-byte BITMAPINFOHEADER, a 256-entry RGBQUAD
// palette and uncompressed 8-bit indices (TERR-LOC-001 records bfOffBits = 1078
// = 14 + 40 + 1024 on every terrain tile). The container itself is the public
// Windows Bitmap standard and is decoded generically. See the package doc and
// docs/0004-terrain-viewer/spec.md.
const (
	bmpFileHeaderSize = 14                                    // BITMAPFILEHEADER
	bmpInfoHeaderSize = 40                                    // BITMAPINFOHEADER; the only DIB header accepted
	bmpHeaderSize     = bmpFileHeaderSize + bmpInfoHeaderSize // 54; the palette starts here
	bmpBitsPerPixel   = 8                                     // the only depth accepted
	bmpCompressionRGB = 0                                     // BI_RGB (uncompressed); the only compression accepted
	bmpPaletteEntry   = 4                                     // one RGBQUAD: B, G, R, X
	bmpMaxPalette     = 256                                   // 8 bpp can index no more
)

// Header field offsets (file-header relative; all lie inside the first
// bmpHeaderSize bytes, so one length check covers every header read).
const (
	bmpOffMagic       = 0x00 // 2 B "BM"
	bmpOffBits        = 0x0a // u32 pixel-data offset
	bmpOffDIBSize     = 0x0e // u32 DIB header size
	bmpOffWidth       = 0x12 // i32
	bmpOffHeight      = 0x16 // i32; positive => rows stored bottom-to-top
	bmpOffBitCount    = 0x1c // u16 bits per pixel
	bmpOffCompression = 0x1e // u32
	bmpOffClrUsed     = 0x2e // u32 palette entries stored (0 => 256 at 8 bpp)
)

// DecodeBMP8 decodes an uncompressed 8-bit Windows BMP into a *paletted* image
// in top-down display order: palette entries are B,G,R,X with X ignored and
// alpha forced opaque, and the file's bottom-up rows (positive height) are read
// into top-down rows with each stored row padded to a 4-byte boundary.
//
// The palette *indices* are retained rather than resolved to colours, because
// two decoded game behaviours address a terrain pixel by its index rather than
// its colour: the impassable dirt overlay treats index 0 as transparent
// (TERR-DIRT-017), and the terrain blitters look each pixel up in a shading
// table addressed by (brightness level, palette index) (TERR-LIGHT-011).
// Resolving to RGBA here would discard the only value those paths key on.
//
// The accepted subset is validated up front — magic "BM", a 40-byte DIB
// header, 8 bits per pixel, BI_RGB compression, a positive width and height,
// and a palette and pixel region that both lie inside the stream. Anything
// else is rejected atomically: the result is a nil image and a non-nil
// error, never a partial image, a panic or an out-of-bounds read.
func DecodeBMP8(data []byte) (*image.Paletted, error) {
	if len(data) < bmpHeaderSize {
		return nil, fmt.Errorf("terrain: bmp stream too small: %d bytes (want at least %d)", len(data), bmpHeaderSize)
	}
	if data[bmpOffMagic] != 'B' || data[bmpOffMagic+1] != 'M' {
		return nil, fmt.Errorf("terrain: bmp bad magic %#02x %#02x, want \"BM\"", data[0], data[1])
	}
	if dib := binary.LittleEndian.Uint32(data[bmpOffDIBSize:]); dib != bmpInfoHeaderSize {
		return nil, fmt.Errorf("terrain: bmp DIB header size %d, want %d", dib, bmpInfoHeaderSize)
	}
	if bpp := binary.LittleEndian.Uint16(data[bmpOffBitCount:]); bpp != bmpBitsPerPixel {
		return nil, fmt.Errorf("terrain: bmp is %d bpp, want %d", bpp, bmpBitsPerPixel)
	}
	if comp := binary.LittleEndian.Uint32(data[bmpOffCompression:]); comp != bmpCompressionRGB {
		return nil, fmt.Errorf("terrain: bmp compression %d, want %d (BI_RGB)", comp, bmpCompressionRGB)
	}

	width := int32(binary.LittleEndian.Uint32(data[bmpOffWidth:]))
	height := int32(binary.LittleEndian.Uint32(data[bmpOffHeight:]))
	if width <= 0 {
		return nil, fmt.Errorf("terrain: bmp width %d is not positive", width)
	}
	// The documented subset stores rows bottom-up, which a positive height
	// signals. A zero or negative height is outside it and is rejected rather
	// than guessed at.
	if height <= 0 {
		return nil, fmt.Errorf("terrain: bmp height %d is not positive (top-down rows are outside the subset)", height)
	}

	// Palette geometry. clrUsed == 0 means "all entries the depth allows".
	clrUsed := binary.LittleEndian.Uint32(data[bmpOffClrUsed:])
	if clrUsed == 0 {
		clrUsed = bmpMaxPalette
	}
	if clrUsed > bmpMaxPalette {
		return nil, fmt.Errorf("terrain: bmp palette declares %d entries, want at most %d", clrUsed, bmpMaxPalette)
	}

	// All range arithmetic is 64-bit so a hostile header cannot overflow a
	// bound into looking valid.
	total := int64(len(data))
	paletteEnd := int64(bmpHeaderSize) + int64(clrUsed)*bmpPaletteEntry
	if paletteEnd > total {
		return nil, fmt.Errorf("terrain: bmp palette [%d, %d) overruns the %d-byte stream", bmpHeaderSize, paletteEnd, total)
	}
	offBits := int64(binary.LittleEndian.Uint32(data[bmpOffBits:]))
	if offBits < paletteEnd || offBits > total {
		return nil, fmt.Errorf("terrain: bmp pixel offset %d outside [%d, %d]", offBits, paletteEnd, total)
	}

	// Each stored row occupies width bytes padded up to a 4-byte boundary.
	// Requiring the whole pixel region to be present also bounds the image
	// allocation below by the input length.
	stride := (int64(width) + 3) &^ 3
	need := stride * int64(height)
	if avail := total - offBits; need > avail {
		return nil, fmt.Errorf("terrain: bmp pixel data needs %d bytes (%d x %d), %d present", need, stride, height, avail)
	}

	palette := make(color.Palette, clrUsed)
	for i := range palette {
		e := data[bmpHeaderSize+i*bmpPaletteEntry:]
		palette[i] = color.RGBA{R: e[2], G: e[1], B: e[0], A: 0xff}
	}

	w, h := int(width), int(height)
	img := image.NewPaletted(image.Rect(0, 0, w, h), palette)
	for y := 0; y < h; y++ {
		// Display row y is stored row h-1-y: the file runs bottom-to-top.
		row := data[offBits+int64(h-1-y)*stride:]
		dst := img.Pix[y*img.Stride:]
		for x := 0; x < w; x++ {
			idx := int(row[x])
			// An index past the stored palette would resolve to nothing; reject
			// rather than clamp, so a malformed file cannot silently render.
			if idx >= len(palette) {
				return nil, fmt.Errorf("terrain: bmp pixel (%d,%d) index %d outside the %d-entry palette", x, y, idx, len(palette))
			}
			dst[x] = byte(idx)
		}
	}
	return img, nil
}
