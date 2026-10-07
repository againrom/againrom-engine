package spr16

import (
	"encoding/binary"
	"fmt"
)

// Layout constants for the shared .16a/.16 container (little-endian). See the
// package doc and docs/0021-16bit-sprites/spec.md for the byte-level format.
const (
	trailerSize     = 4  // raw u32; .16 and .16a apply distinct count contracts
	frameHeaderSize = 12 // [u32 width][u32 height][u32 dataSize]

	frameCountMask = 0x7FFFFFFF // .16a count; DecodeG admits only nonnegative raw words
)

// Sanity caps (the spec's Constraints A numbers, verbatim). Every real sheet
// decodes with wide margin; a value over its cap is an error, never a frame
// silently skipped, so a corrupt header cannot demand an absurd allocation and
// a breach is visible. They also keep every cursor and walk product below
// overflow-free int arithmetic.
const (
	maxDimension  = 2048    // width and height, each
	maxFrameCount = 4096    // trailer & frameCountMask
	maxDataSize   = 1 << 24 // one frame's RLE block, bytes
	maxStreamSize = 1 << 26 // the whole stream, bytes
)

// record is one frame record as the container walk returns it: the header's
// dimensions and its undecoded RLE block, a sub-slice of the input stream.
type record struct {
	width, height uint32
	block         []byte
}

// records walks the container region shared by both formats. The low31 mask
// is the .16a count contract; DecodeG checks the .16 signed raw word first,
// so masking changes no admitted .16 count. It reads count records strictly
// sequentially from start (1024 past a declared palette, else 0). Each header
// and block must fit before the trailer's first byte; bytes between the last
// counted record and the trailer are never touched. Malformed input yields a
// non-nil error and nil records.
func records(data []byte, start int) ([]record, error) {
	if len(data) > maxStreamSize {
		return nil, fmt.Errorf("spr16: %d-byte stream exceeds the %d-byte cap", len(data), maxStreamSize)
	}
	if len(data) < start+trailerSize {
		return nil, fmt.Errorf("spr16: stream too small: %d bytes", len(data))
	}

	trailer := binary.LittleEndian.Uint32(data[len(data)-trailerSize:])
	count := int(trailer & frameCountMask)
	if count > maxFrameCount {
		return nil, fmt.Errorf("spr16: frame count %d exceeds the %d cap", count, maxFrameCount)
	}

	trailerStart := len(data) - trailerSize
	recs := make([]record, 0, count)
	pos := start
	for i := 0; i < count; i++ {
		if pos+frameHeaderSize > trailerStart {
			return nil, fmt.Errorf("spr16: frame %d header does not fit before the trailer", i)
		}
		width := binary.LittleEndian.Uint32(data[pos : pos+4])
		height := binary.LittleEndian.Uint32(data[pos+4 : pos+8])
		dataSize := binary.LittleEndian.Uint32(data[pos+8 : pos+12])
		if width > maxDimension || height > maxDimension {
			return nil, fmt.Errorf("spr16: frame %d is %dx%d, over the %d dimension cap", i, width, height, maxDimension)
		}
		if dataSize > maxDataSize {
			return nil, fmt.Errorf("spr16: frame %d block of %d bytes exceeds the %d-byte cap", i, dataSize, maxDataSize)
		}
		blockStart := pos + frameHeaderSize
		if blockStart+int(dataSize) > trailerStart {
			return nil, fmt.Errorf("spr16: frame %d block (%d bytes) does not fit before the trailer", i, dataSize)
		}
		recs = append(recs, record{width: width, height: height, block: data[blockStart : blockStart+int(dataSize)]})
		pos = blockStart + int(dataSize)
	}
	return recs, nil
}

// cursor owns the grid discipline both pixel grammars cite: position pos walks
// a w x h grid in linear order — pixel (pos mod w, pos div w), row 0 the top
// row. Any op may leave pos exactly at w*h; one that would move or paint
// beyond it is malformed. The caps above keep every product here
// overflow-free.
type cursor struct {
	w, h int
	pos  int
}

// announce enforces the completion rule on one op's count before the op acts:
// once pos equals w*h (a zero-area grid is born complete) a non-zero count
// refuses — a zero-width blank-row op included, though it would move nothing —
// and count-0 ops pass anywhere.
func (c *cursor) announce(n int) error {
	if n != 0 && c.pos == c.w*c.h {
		return fmt.Errorf("spr16: op with count %d after the %dx%d grid completed", n, c.w, c.h)
	}
	return nil
}

// skip advances the cursor n pixels, refusing a move past w*h.
func (c *cursor) skip(n int) error {
	if c.pos+n > c.w*c.h {
		return fmt.Errorf("spr16: skip of %d moves past the %dx%d grid", n, c.w, c.h)
	}
	c.pos += n
	return nil
}

// rows advances the cursor n blank rows (n*w pixels) — plain cursor arithmetic
// from any position, mid-row included — refusing a move past w*h.
func (c *cursor) rows(n int) error {
	if c.pos+n*c.w > c.w*c.h {
		return fmt.Errorf("spr16: blank-row op of %d rows moves past the %dx%d grid", n, c.w, c.h)
	}
	c.pos += n * c.w
	return nil
}

// paint yields the linear index of the pixel a literal paints, then advances,
// refusing a paint at or past w*h.
func (c *cursor) paint() (int, error) {
	if c.pos >= c.w*c.h {
		return 0, fmt.Errorf("spr16: literal paints past the %dx%d grid", c.w, c.h)
	}
	i := c.pos
	c.pos++
	return i, nil
}
