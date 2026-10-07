package spr16

import (
	"encoding/binary"
	"fmt"
)

// .16 RLE control byte: op = b >> 6 and n = b & 0x3F, a 6-bit count.
const (
	opShiftG   = 6
	countMaskG = 0x3F

	opLiteralG = 0b00 // the next n bytes carry up to two 4-bit pixels each
	opRowsG    = 0b01 // blank rows: the cursor advances n*width
	// 0b10 (skip) and its 0b11 alias are decoded in one shared arm — the
	// opposite direction from the .16a grammar, whose 0b11 is blank rows.
)

// PixelG is one decoded .16 grid cell. A transparent cell is the zero value
// (Painted == false); Value is meaningful only when Painted is true and is the
// raw 4-bit glyph value 0–15, preserved with no reinterpretation — a painted
// value 0 is distinct from a transparent pixel, and its meaning is the
// consumer's concern.
type PixelG struct {
	Value   uint8 // raw 4-bit value: one nibble of a literal byte
	Painted bool
}

// FrameG is one decoded .16 image: a Width x Height grid in row-major order.
// Pixels has exactly Width*Height elements, empty and non-nil for a zero-area
// frame.
type FrameG struct {
	Width  int
	Height int
	Pixels []PixelG
}

// DecodeG parses a .16 glyph-atlas sprite stream. The format never has a
// palette, so frames start at offset 0 and the result is the frame list alone.
// DecodeG takes the raw nonnegative trailer as its frame count (SPR16A-071).
// Bit31-set values refuse: the original skips indexing if allocation returns,
// but native allocation and later usability are Unknown (DIV-1250). It
// decodes exactly frameCount frame records, each RLE block onto its
// width x height grid; bytes between the last counted record and the trailer
// are ignored. Malformed input yields a non-nil error and a nil frame list —
// never a panic or a partial result.
func DecodeG(data []byte) ([]FrameG, error) {
	if len(data) >= trailerSize {
		raw := binary.LittleEndian.Uint32(data[len(data)-trailerSize:])
		if int32(raw) < 0 {
			return nil, fmt.Errorf("spr16: unsupported negative .16 trailer %#08x", raw)
		}
	}
	recs, err := records(data, 0)
	if err != nil {
		return nil, err
	}
	frames := make([]FrameG, 0, len(recs))
	for _, rec := range recs {
		frame, err := decodeBlockG(rec)
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// decodeBlockG decodes one .16 RLE block onto its width x height grid. The
// grid starts fully transparent and the program is consumed to the end of its
// own block — an unfinished grid stays transparent — with each op's count
// announced to the cursor before the op acts. A literal of n bytes paints low
// nibble then high per byte; the one exception is the run's final byte with
// high nibble 0 — a pad, nothing emitted — while a mid-run zero high nibble
// paints value 0 and advances. A literal whose operands would cross the
// block's end refuses at the read, before any cursor move.
func decodeBlockG(rec record) (FrameG, error) {
	w, h := int(rec.width), int(rec.height)
	pixels := make([]PixelG, w*h)
	cur := cursor{w: w, h: h}
	block := rec.block
	for bi := 0; bi < len(block); {
		b := block[bi]
		bi++
		n := int(b & countMaskG)
		if err := cur.announce(n); err != nil {
			return FrameG{}, err
		}
		switch b >> opShiftG {
		case opLiteralG:
			if bi+n > len(block) {
				return FrameG{}, fmt.Errorf("spr16: literal run of %d bytes overruns its %d-byte block", n, len(block))
			}
			for j := 0; j < n; j++ {
				pb := block[bi]
				bi++
				i, err := cur.paint()
				if err != nil {
					return FrameG{}, err
				}
				pixels[i] = PixelG{Value: pb & 0x0F, Painted: true}
				// The pad rule fires only on the run's final byte: a zero
				// high nibble there emits nothing, anywhere else it paints
				// a value-0 pixel and advances.
				if j == n-1 && pb>>4 == 0 {
					continue
				}
				i, err = cur.paint()
				if err != nil {
					return FrameG{}, err
				}
				pixels[i] = PixelG{Value: pb >> 4, Painted: true}
			}
		case opRowsG:
			if err := cur.rows(n); err != nil {
				return FrameG{}, err
			}
		default: // skip (0b10) and its 0b11 alias: one shared skip arm.
			if err := cur.skip(n); err != nil {
				return FrameG{}, err
			}
		}
	}
	return FrameG{Width: w, Height: h, Pixels: pixels}, nil
}
