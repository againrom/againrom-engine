package menu

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
)

// Layout constants for the Windows BMP subset the menu bitmaps use: a 14-byte
// BITMAPFILEHEADER, a 40-byte BITMAPINFOHEADER, an optional 256-entry RGBQUAD
// palette at 8 bpp, and uncompressed pixel data at the offset the file header
// names. The container is the public Windows Bitmap standard and is decoded
// generically; which bitmaps the menu is made of is the decoded contract in
// menu.go.
const (
	bmpFileHeaderSize = 14
	bmpInfoHeaderSize = 40 // the only DIB header accepted
	bmpHeaderSize     = bmpFileHeaderSize + bmpInfoHeaderSize
	bmpCompressionRGB = 0 // BI_RGB; the only compression accepted
	bmpPaletteEntry   = 4 // one RGBQUAD: B, G, R, X
	bmpMaxPalette     = 256
)

// Header field offsets, all inside the first bmpHeaderSize bytes.
const (
	bmpOffMagic       = 0x00 // 2 B "BM"
	bmpOffBits        = 0x0a // u32 pixel-data offset
	bmpOffDIBSize     = 0x0e // u32 DIB header size
	bmpOffWidth       = 0x12 // i32
	bmpOffHeight      = 0x16 // i32; positive => rows stored bottom-up
	bmpOffBitCount    = 0x1c // u16 bits per pixel
	bmpOffCompression = 0x1e // u32
	bmpOffClrUsed     = 0x2e // u32 palette entries stored (0 => 256 at 8 bpp)
)

// bmpInfo is a validated BMP header: the accepted subset, already range-checked
// against the stream.
type bmpInfo struct {
	width, height int   // display dimensions, always positive
	bottomUp      bool  // rows stored bottom-to-top (a positive stored height)
	stride        int64 // stored bytes per row, padded to 4
	offBits       int64 // first pixel byte
	clrUsed       int   // palette entries present (0 at 24 bpp)
}

// parseBMPHeader validates the header of an uncompressed Windows BMP at the given
// depth and returns its geometry.
//
// Row order is the one decision in this package that the research's *inferred*
// mask row order bears on, so it is taken here and nowhere else: a **positive**
// stored height means the rows are stored bottom-to-top and are normalised to
// top-down display order; a **negative** stored height means they are already
// top-down. That is the public BMP standard, and every bitmap in the shipped
// main.res is positive-height. If the original engine's raw-index loader does not
// perform that flip, the menu's hit regions are mirrored about the horizontal
// midline — which the near-symmetric brooch layout would very nearly hide. Only
// the manual criterion against the real game can settle it; nothing here should
// try to guess.
//
// The pixel region is located through bfOffBits and only required to be present,
// never to end exactly at EOF: every one of the eighteen shipped menu bitmaps
// carries two bytes beyond the size its header implies, so a stricter check would
// reject the whole asset set.
func parseBMPHeader(data []byte, wantBPP int) (bmpInfo, error) {
	var info bmpInfo
	if len(data) < bmpHeaderSize {
		return info, fmt.Errorf("bmp stream too small: %d bytes (want at least %d)", len(data), bmpHeaderSize)
	}
	if data[bmpOffMagic] != 'B' || data[bmpOffMagic+1] != 'M' {
		return info, fmt.Errorf("bmp bad magic %#02x %#02x, want \"BM\"", data[0], data[1])
	}
	if dib := binary.LittleEndian.Uint32(data[bmpOffDIBSize:]); dib != bmpInfoHeaderSize {
		return info, fmt.Errorf("bmp DIB header size %d, want %d", dib, bmpInfoHeaderSize)
	}
	if bpp := binary.LittleEndian.Uint16(data[bmpOffBitCount:]); int(bpp) != wantBPP {
		return info, fmt.Errorf("bmp is %d bpp, want %d", bpp, wantBPP)
	}
	if comp := binary.LittleEndian.Uint32(data[bmpOffCompression:]); comp != bmpCompressionRGB {
		return info, fmt.Errorf("bmp compression %d, want %d (BI_RGB)", comp, bmpCompressionRGB)
	}

	width := int32(binary.LittleEndian.Uint32(data[bmpOffWidth:]))
	storedH := int32(binary.LittleEndian.Uint32(data[bmpOffHeight:]))
	if width <= 0 {
		return info, fmt.Errorf("bmp width %d is not positive", width)
	}
	if storedH == 0 {
		return info, fmt.Errorf("bmp height is zero")
	}
	info.width = int(width)
	info.bottomUp = storedH > 0
	if storedH > 0 {
		info.height = int(storedH)
	} else {
		info.height = int(-int64(storedH))
	}

	// Palette geometry. clrUsed == 0 means "all entries the depth allows"; at
	// 24 bpp there is no palette at all.
	paletteEnd := int64(bmpHeaderSize)
	if wantBPP == 8 {
		clrUsed := binary.LittleEndian.Uint32(data[bmpOffClrUsed:])
		if clrUsed == 0 {
			clrUsed = bmpMaxPalette
		}
		if clrUsed > bmpMaxPalette {
			return info, fmt.Errorf("bmp palette declares %d entries, want at most %d", clrUsed, bmpMaxPalette)
		}
		info.clrUsed = int(clrUsed)
		paletteEnd += int64(clrUsed) * bmpPaletteEntry
	}

	// All range arithmetic is 64-bit so a hostile header cannot overflow a bound
	// into looking valid.
	total := int64(len(data))
	if paletteEnd > total {
		return info, fmt.Errorf("bmp palette [%d, %d) overruns the %d-byte stream", bmpHeaderSize, paletteEnd, total)
	}
	info.offBits = int64(binary.LittleEndian.Uint32(data[bmpOffBits:]))
	if info.offBits < paletteEnd || info.offBits > total {
		return info, fmt.Errorf("bmp pixel offset %d outside [%d, %d]", info.offBits, paletteEnd, total)
	}

	bytesPerPixel := int64(wantBPP / 8)
	info.stride = (int64(info.width)*bytesPerPixel + 3) &^ 3
	need := info.stride * int64(info.height)
	if avail := total - info.offBits; need > avail {
		return info, fmt.Errorf("bmp pixel data needs %d bytes (%dx%d), %d present",
			need, info.width, info.height, avail)
	}
	return info, nil
}

// storedRow maps a top-down display row to the row index holding it in the
// stream.
func (i bmpInfo) storedRow(y int) int64 {
	if i.bottomUp {
		return int64(i.height - 1 - y)
	}
	return int64(y)
}

// decodeBMP24 decodes an uncompressed 24-bpp Windows BMP into an *image.RGBA in
// top-down display order, with alpha forced opaque.
//
// Opaque is not a simplification: the research establishes the menu overlays as
// opaque 24-bit rectangles and leaves any blend the original may apply outside
// what it has established, so an alpha channel here would be invented rather than
// decoded.
//
// A stream outside the accepted subset is rejected atomically: nil image, non-nil
// error, never a partial image, a panic or an out-of-bounds read.
func decodeBMP24(data []byte) (*image.RGBA, error) {
	info, err := parseBMPHeader(data, 24)
	if err != nil {
		return nil, err
	}
	img := image.NewRGBA(image.Rect(0, 0, info.width, info.height))
	for y := 0; y < info.height; y++ {
		src := data[info.offBits+info.storedRow(y)*info.stride:]
		dst := img.Pix[y*img.Stride:]
		for x := 0; x < info.width; x++ {
			s := src[x*3:]
			d := dst[x*4:]
			d[0], d[1], d[2], d[3] = s[2], s[1], s[0], 0xff // stored B,G,R
		}
	}
	return img, nil
}

// decodeBMP8 decodes an uncompressed 8-bpp Windows BMP into an *image.Paletted in
// top-down display order, keeping the raw palette indices.
//
// Keeping indices rather than resolving them to colours is what the hit mask
// requires, not an optimisation: the mask's palette is an identity grayscale
// ramp, so the byte value *is* the semantic and any reading that goes through
// palette colours is wrong by construction. The palette is still decoded and
// carried, so a caller that legitimately wants colours has them — but nothing in
// this package's hit path consults it.
func decodeBMP8(data []byte) (*image.Paletted, error) {
	info, err := parseBMPHeader(data, 8)
	if err != nil {
		return nil, err
	}

	palette := make(color.Palette, info.clrUsed)
	for i := range palette {
		e := data[bmpHeaderSize+i*bmpPaletteEntry:]
		palette[i] = color.RGBA{R: e[2], G: e[1], B: e[0], A: 0xff}
	}

	img := image.NewPaletted(image.Rect(0, 0, info.width, info.height), palette)
	for y := 0; y < info.height; y++ {
		src := data[info.offBits+info.storedRow(y)*info.stride:]
		dst := img.Pix[y*img.Stride:]
		for x := 0; x < info.width; x++ {
			idx := int(src[x])
			// An index past the stored palette would resolve to nothing; reject
			// rather than clamp, so a malformed file cannot silently render.
			if idx >= len(palette) {
				return nil, fmt.Errorf("bmp pixel (%d,%d) index %d outside the %d-entry palette",
					x, y, idx, len(palette))
			}
			dst[x] = byte(idx)
		}
	}
	return img, nil
}
