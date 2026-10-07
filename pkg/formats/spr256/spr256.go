package spr256

import (
	"encoding/binary"
	"fmt"
)

// Layout constants for the ROM1 .256 sprite (little-endian). See the package doc
// and docs/0002-sprites-256/spec.md for the byte-level format.
const (
	trailerSize     = 4    // [31-bit frameCount][bit31 = has-palette]
	paletteSize     = 1024 // 256 entries x 4 bytes [B, G, R, reserved]
	paletteEntries  = 256
	frameHeaderSize = 12 // [u32 width][u32 height][u32 dataSize]

	frameCountMask = 0x7FFFFFFF // trailer & this == frame count
	hasPaletteBit  = 0x80000000 // trailer & this == has-palette flag

	// RLE control byte: [2-bit class | 6-bit count].
	rleClassMask   = 0xC0
	rleCountMask   = 0x3F
	rleLiteral     = 0x00 // next N bytes are opaque palette indices
	rleBlankRows   = 0x40 // N fully-transparent rows (row boundary only)
	rleTransparent = 0x80 // N transparent pixels (0xC0 aliases this)
)

// maxInt is the largest value representable by an int on this platform. A frame
// pixel count exceeding it cannot be allocated and is rejected before any make,
// so a crafted near-2^32 width/height pair is an atomic error, not a panic.
const maxInt = int(^uint(0) >> 1)

// Color is one palette entry as an RGB triple. The on-disk order is BGR with a
// reserved 4th byte; Decode reorders it to RGB and drops the reserved byte.
type Color struct {
	R, G, B uint8
}

// Pixel is one decoded grid cell. A transparent cell is the zero value
// (Opaque == false); Index is meaningful only when Opaque is true. Transparency
// is structural — carried by the RLE opcodes, not by palette index 0 — so an
// opaque pixel may legitimately carry index 0.
type Pixel struct {
	Index  uint8
	Opaque bool
}

// Frame is one decoded image: a Width x Height grid in row-major order. Pixels
// has exactly Width*Height elements (empty for a zero-area frame).
type Frame struct {
	Width  int
	Height int
	Pixels []Pixel
}

// Sprite is a decoded .256 stream: its frames in stream order, and its palette
// when the stream carries one. HasPalette reflects the trailer's bit 31; Palette
// holds 256 RGB colors when HasPalette is true and is nil otherwise. Palette
// entry 0 is the reserved transparent/background key, returned verbatim.
type Sprite struct {
	HasPalette bool
	Palette    []Color
	Frames     []Frame
}

// Decode parses a .256 sprite stream and returns its frames and palette. It
// reads the 4-byte trailer, takes frameCount = trailer & 0x7FFFFFFF and
// hasPalette = trailer & 0x80000000, reads the leading 1024-byte palette iff
// hasPalette, and reads exactly frameCount frame records from the frame region
// (offset 1024 with a palette, else 0), decoding each RLE block onto its
// width x height grid. Malformed input yields a non-nil error and a nil *Sprite,
// never a panic or a partial result.
func Decode(data []byte) (*Sprite, error) {
	if len(data) < trailerSize {
		return nil, fmt.Errorf("spr256: stream too small: %d bytes", len(data))
	}

	// The trailer is the last 4 bytes (the loader seeks to it first). Its low 31
	// bits are the frame count; bit 31 is the has-palette flag.
	trailer := binary.LittleEndian.Uint32(data[len(data)-trailerSize:])
	frameCount := int(trailer & frameCountMask)
	hasPalette := trailer&hasPaletteBit != 0

	// Frame region: [framesStart, trailerStart). The palette, when present,
	// occupies the leading 1024 bytes.
	framesStart := 0
	var palette []Color
	if hasPalette {
		if len(data) < paletteSize+trailerSize {
			return nil, fmt.Errorf("spr256: has-palette stream too small: %d bytes", len(data))
		}
		palette = decodePalette(data[:paletteSize])
		framesStart = paletteSize
	}
	trailerStart := len(data) - trailerSize

	// Read exactly frameCount records. A standard sprite's last record ends at
	// trailerStart; a Bucket-B sprite leaves an appended section between the last
	// frame and the trailer, which is intentionally left unread.
	frames := make([]Frame, 0, frameCount)
	cursor := framesStart
	for i := 0; i < frameCount; i++ {
		if int64(cursor)+frameHeaderSize > int64(trailerStart) {
			return nil, fmt.Errorf("spr256: frame %d header does not fit before the trailer", i)
		}
		width := binary.LittleEndian.Uint32(data[cursor : cursor+4])
		height := binary.LittleEndian.Uint32(data[cursor+4 : cursor+8])
		dataSize := binary.LittleEndian.Uint32(data[cursor+8 : cursor+12])

		blockStart := cursor + frameHeaderSize
		if int64(blockStart)+int64(dataSize) > int64(trailerStart) {
			return nil, fmt.Errorf("spr256: frame %d block (%d bytes) does not fit before the trailer", i, dataSize)
		}
		block := data[blockStart : blockStart+int(dataSize)]

		frame, err := decodeFrame(width, height, block)
		if err != nil {
			return nil, fmt.Errorf("spr256: frame %d: %w", i, err)
		}
		frames = append(frames, frame)
		cursor = blockStart + int(dataSize)
	}

	return &Sprite{HasPalette: hasPalette, Palette: palette, Frames: frames}, nil
}

// decodePalette reorders 256 on-disk [B, G, R, reserved] entries to RGB Colors.
// The input is exactly 1024 bytes (validated by the caller).
func decodePalette(raw []byte) []Color {
	pal := make([]Color, paletteEntries)
	for i := range pal {
		e := raw[i*4 : i*4+4]
		pal[i] = Color{R: e[2], G: e[1], B: e[0]} // reserved byte e[3] dropped
	}
	return pal
}

// decodeFrame decodes one RLE block onto a width x height grid. The cursor moves
// left->right and wraps to the next row when the column reaches width; the grid
// starts fully transparent (the Pixel zero value), so only literal runs write.
// The block must tile the grid exactly (every row filled to width, exactly
// height rows, no leftover byte); any deviation is an atomic error.
func decodeFrame(width, height uint32, block []byte) (Frame, error) {
	// width*height cannot overflow uint64 (two uint32 factors). A count that does
	// not fit an int cannot be allocated, so reject it before make (never panic).
	w := uint64(width)
	pixelCount := w * uint64(height)
	if pixelCount > uint64(maxInt) {
		return Frame{}, fmt.Errorf("frame %dx%d exceeds the addressable pixel limit", width, height)
	}
	grid := make([]Pixel, int(pixelCount))

	var pos, col, rows uint64 // linear cursor, current column, completed rows
	bi := 0
	for bi < len(block) {
		c := block[bi]
		bi++
		count := uint64(c & rleCountMask)
		switch c & rleClassMask {
		case rleLiteral:
			// The next count bytes are opaque palette indices, one pixel each.
			if bi+int(count) > len(block) {
				return Frame{}, fmt.Errorf("literal run of %d overruns its %d-byte block", count, len(block))
			}
			for k := uint64(0); k < count; k++ {
				if w == 0 || pos >= pixelCount {
					return Frame{}, fmt.Errorf("literal pixel advances the cursor past the %dx%d grid", width, height)
				}
				grid[pos] = Pixel{Index: block[bi], Opaque: true}
				bi++
				pos++
				col++
				if col == w {
					col = 0
					rows++
				}
			}
		case rleBlankRows:
			if count == 0 {
				continue // no-op
			}
			// N fully-transparent rows, emitted whole at a row boundary.
			if col != 0 {
				return Frame{}, fmt.Errorf("blank-row op not at a row boundary (column %d)", col)
			}
			if rows+count > uint64(height) {
				return Frame{}, fmt.Errorf("blank-row op overruns the %d-row grid", height)
			}
			rows += count
			pos += count * w // == rows*width; stays within pixelCount
		default:
			// 0x80 and 0xC0 alike: a transparent skip of N pixels (0xC0 is a
			// decode-time alias of 0x80 in the shipped loader's shared branch).
			for k := uint64(0); k < count; k++ {
				if w == 0 || pos >= pixelCount {
					return Frame{}, fmt.Errorf("transparent skip advances the cursor past the %dx%d grid", width, height)
				}
				pos++
				col++
				if col == w {
					col = 0
					rows++
				}
			}
		}
	}

	// Exact tiling: every row filled to width (col back to 0) and exactly height
	// rows produced. These force pos == pixelCount.
	if col != 0 || rows != uint64(height) {
		return Frame{}, fmt.Errorf("RLE block does not tile the %dx%d grid (%d rows, column %d)", width, height, rows, col)
	}

	return Frame{Width: int(width), Height: int(height), Pixels: grid}, nil
}
