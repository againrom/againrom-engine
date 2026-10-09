package spr16

import (
	"encoding/binary"
	"fmt"

	"againrom/pkg/formats/pal"
)

// Palette layout for a declared .16a palette: 1024 leading bytes, 256 entries
// of on-disk [B, G, R, x]; the 4th byte is not decoded. Presence is the
// caller's declaration, never inferred from stream content, which is why it is
// a parameter of DecodeA rather than a stream fact.
const paletteSize = 1024

// .16a RLE control word: a u16 cw with op = cw >> 14 and n = cw & 0x3FFF, a
// 14-bit count.
const (
	opShiftA   = 14
	countMaskA = 0x3FFF

	opLiteralA = 0b00 // the next n u16 words paint one pixel each
	opRowsA    = 0b01 // blank rows: the cursor advances n*width
	opSkipA    = 0b10 // skip: the cursor advances n
	// 0b11 is decoded identically to opRowsA (blank rows), in the same arm.
)

// Color is one palette entry: pal.Color, the one reading of the
// [B, G, R, x] layout.
type Color = pal.Color

// PixelA is one decoded .16a grid cell. A transparent cell is the zero value
// (Painted == false); Index and Level are meaningful only when Painted is true
// and are preserved raw, assigned no colour or transparency meaning — a
// painted level 0 is distinct from a transparent pixel, and resolving the pair
// against the palette is the consumer's concern.
type PixelA struct {
	Index   uint8 // palette index: bits 1–8 of the literal pixel word
	Level   uint8 // 4-bit level: bits 9–12 of the literal pixel word
	Painted bool
}

// FrameA is one decoded .16a image: a Width x Height grid in row-major order.
// Pixels has exactly Width*Height elements, empty and non-nil for a zero-area
// frame.
type FrameA struct {
	Width  int
	Height int
	Pixels []PixelA
}

// SpriteA is a decoded .16a stream: its frames in stream order, and its
// 256-colour palette when the caller declared one — nil when not, which is how
// an absent palette is reported.
type SpriteA struct {
	Palette []Color
	Frames  []FrameA
}

// DecodeA parses a .16a sprite stream. palette is the caller's declaration
// that the stream leads with a 1024-byte palette; it is never inferred from
// stream content. DecodeA takes frameCount = trailer & 0x7FFFFFFF from the
// stream's last 4 bytes and decodes exactly frameCount frame records from
// offset 1024 when the palette is declared, else 0, each RLE block onto its
// width x height grid; bytes between the last counted record and the trailer
// are ignored. Malformed input yields a non-nil error and a nil *SpriteA — no
// frames, no palette — never a panic or a partial result.
func DecodeA(data []byte, palette bool) (*SpriteA, error) {
	start := 0
	if palette {
		start = paletteSize
	}
	recs, err := records(data, start)
	if err != nil {
		return nil, err
	}
	frames := make([]FrameA, 0, len(recs))
	for _, rec := range recs {
		frame, err := decodeBlockA(rec)
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	var colors []Color
	if palette {
		colors = pal.Entries(data[:paletteSize])
	}
	return &SpriteA{Palette: colors, Frames: frames}, nil
}

// decodeBlockA decodes one .16a RLE block onto its width x height grid. The
// grid starts fully transparent and the program is consumed to the end of its
// own block — an unfinished grid stays transparent — with each op's count
// announced to the cursor before the op acts. A control or literal operand
// that would cross the block's end refuses at the read, before any cursor
// move; the odd trailing byte where a word should start is the same refusal.
func decodeBlockA(rec record) (FrameA, error) {
	w, h := int(rec.width), int(rec.height)
	pixels := make([]PixelA, w*h)
	cur := cursor{w: w, h: h}
	block := rec.block
	for bi := 0; bi < len(block); {
		if bi+2 > len(block) {
			return FrameA{}, fmt.Errorf("spr16: control word truncated at byte %d of a %d-byte block", bi, len(block))
		}
		cw := binary.LittleEndian.Uint16(block[bi:])
		bi += 2
		n := int(cw & countMaskA)
		if err := cur.announce(n); err != nil {
			return FrameA{}, err
		}
		switch cw >> opShiftA {
		case opLiteralA:
			if bi+2*n > len(block) {
				return FrameA{}, fmt.Errorf("spr16: literal run of %d words overruns its %d-byte block", n, len(block))
			}
			for k := 0; k < n; k++ {
				ss := binary.LittleEndian.Uint16(block[bi:])
				bi += 2
				i, err := cur.paint()
				if err != nil {
					return FrameA{}, err
				}
				// Bits 0 and 13–15 are masked off unconditionally, never
				// validated; index and level are preserved raw.
				pixels[i] = PixelA{Index: uint8((ss >> 1) & 0xFF), Level: uint8((ss >> 9) & 0x0F), Painted: true}
			}
		case opSkipA:
			if err := cur.skip(n); err != nil {
				return FrameA{}, err
			}
		default: // opRowsA and its 0b11 alias: one shared blank-rows arm.
			if err := cur.rows(n); err != nil {
				return FrameA{}, err
			}
		}
	}
	return FrameA{Width: w, Height: h, Pixels: pixels}, nil
}
