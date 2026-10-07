package synth

import (
	"encoding/binary"
	"fmt"
	"image/color"
)

// ---------------------------------------------------------------------------
// .256 sprite sheets (see docs/0002-sprites-256/spec.md, "Format definition")
// ---------------------------------------------------------------------------

// The .256 stream is
//
//	[ 1024 B palette ][ frame, frame, ... ][ 4 B trailer ]
//
// with every integer little-endian. The palette is present iff the trailer's
// bit 31 is set, and each frame record is
//
//	0x00 u32 width    0x08 u32 dataSize
//	0x04 u32 height   0x0c dataSize bytes of RLE program
//
// The trailer is [31-bit frameCount][bit 31 = has-palette]: frameCount equals
// the number of frame records between the palette and the trailer, and bit 31
// is what tells a reader whether the leading 1024 bytes are a palette at all.
//
// A palette entry is four bytes [B, G, R, reserved] — BGR, not RGB, and the
// reserved byte is 0 in every standard palette.
//
// The RLE program decodes a width x height grid. A cursor moves left to right
// and wraps at width; the background emits no pixel. Each control byte is
// [2-bit class | 6-bit count], class = c & 0xC0 and N = c & 0x3F:
//
//	0x00  literal:     the next N bytes are palette indices, one opaque pixel each
//	0x40  blank rows:  N fully-transparent rows, at a row boundary only
//	0x80  transparent: N transparent pixels
//	0xC0  an alias of 0x80, never emitted by shipped data
//
// Every row's tokens sum to exactly width, the program yields exactly height
// rows, and the block is consumed with no leftover byte.
const (
	spr256PaletteLen = 1024
	spr256EntryLen   = 4

	// spr256HasPalette is the trailer's bit 31.
	spr256HasPalette = 0x80000000

	// spr256MaxRun is the largest count a control byte's six bits hold, so a
	// longer run is split across several controls rather than overflowing into
	// the class bits.
	spr256MaxRun = 0x3f

	spr256OpLiteral     = 0x00
	spr256OpBlankRows   = 0x40
	spr256OpTransparent = 0x80
)

// Pixel256 is one pixel a caller hands the sheet builder: a palette index that
// is written only when Opaque is set, and a hole otherwise.
//
// TRANSPARENCY IS STRUCTURAL — it is carried by the RLE opcodes and by no
// reserved palette index — so {Index: 0, Opaque: true} is a legitimate pixel and
// this builder encodes it as a one-byte literal run holding 0. A builder that
// spelled transparency as index 0 could not express it, which is why the flag
// is a field of its own rather than a convention over Index.
//
// The zero value is a transparent pixel, so a Frame256 whose Pixels are left nil
// is a wholly transparent grid.
type Pixel256 struct {
	Index  uint8
	Opaque bool
}

// Frame256 is one frame of a synthetic sheet: a Width x Height grid in row-major
// order. Pixels holds exactly Width*Height elements, or is nil for a wholly
// transparent frame; any other length panics rather than being padded, since a
// short grid is a caller's arithmetic mistake and not a format the game ships.
//
// Frames in one sheet need NOT share a size (SPR256-FRAME-023: 131 of the
// install's 1384 sheets mix them, two of them object sheets whose classes select
// different frames by Index), so nothing here derives one frame's dimensions
// from another's.
//
// A Width or Height of 0 is a valid header describing an empty grid. A negative
// one panics: the record's fields are u32.
type Frame256 struct {
	Width  int
	Height int
	Pixels []Pixel256
}

// Sheet256Options parameterises a synthetic .256 sheet. The zero value builds a
// palette-bearing sheet of no frames — trailer 0x80000000 — which is the shape
// 1370 of the install's 1376 palette-bearing sheets carry, minus their content.
type Sheet256Options struct {
	// Palette holds up to 256 entries, written as [B, G, R, 0] each. Entries a
	// caller does not supply stay zero, and the alpha channel of a color.RGBA is
	// ignored: the format's 4th byte is reserved, not an alpha channel, and is 0
	// in every standard palette.
	Palette []color.RGBA

	// NoPalette omits the 1024-byte palette block and clears the trailer's bit
	// 31 — the 6 projectile-arrow sheets' variant, and the "palette-less sheet"
	// exclusion a consumer of the object layer must skip. It is an opt-OUT so
	// that the zero value builds the ordinary sheet.
	NoPalette bool

	// Frames are written in order, and that order is the frame order an Index
	// selects in.
	Frames []Frame256
}

// Sheet256RawFrame is one frame record written verbatim: the three header words
// and the block that follows them, with no relation between DataSize and Data
// enforced and no check that Data decodes to a Width x Height grid.
type Sheet256RawFrame struct {
	Width, Height, DataSize uint32
	Data                    []byte
}

// Sheet256Raw lays out palette | frame records | trailer with no validation of
// any kind: the escape hatch for the fixtures a well-formed sheet cannot
// express — a dataSize that overruns the trailer, a frameCount disagreeing with
// the records, an RLE block whose rows do not close, a palette flag lying about
// the block in front of it.
//
// palette is written verbatim and a nil one writes nothing, INDEPENDENTLY of
// the trailer's bit 31: a caller wanting the ordinary 1024-byte block passes
// Palette256's output, and one wanting the two to disagree passes whatever it
// likes. Use Palette256 rather than a hand-rolled block, so the entry layout
// stays in one function.
//
// Every field goes out at the offset the format contract gives it, which is why
// Sheet256 is written over this rather than beside it: exactly one function in
// this package knows the byte layout.
func Sheet256Raw(palette []byte, frames []Sheet256RawFrame, trailer uint32) []byte {
	out := append([]byte(nil), palette...)
	for _, f := range frames {
		out = binary.LittleEndian.AppendUint32(out, f.Width)
		out = binary.LittleEndian.AppendUint32(out, f.Height)
		out = binary.LittleEndian.AppendUint32(out, f.DataSize)
		out = append(out, f.Data...)
	}
	return binary.LittleEndian.AppendUint32(out, trailer)
}

// Sheet256 assembles a well-formed .256 sheet: the optional palette, one frame
// record per frame with its own RLE program, and the trailer carrying the frame
// count and the has-palette flag that describes what was actually written.
//
// The trailer's two halves are derived rather than taken: the count is the
// number of records emitted and the flag follows NoPalette, so the stream can
// never claim a palette it does not carry. A fixture that needs it to lie is a
// Sheet256Raw fixture.
func Sheet256(o Sheet256Options) []byte {
	var palette []byte
	trailer := uint32(len(o.Frames))
	if !o.NoPalette {
		palette = Palette256(o.Palette)
		trailer |= spr256HasPalette
	}

	frames := make([]Sheet256RawFrame, len(o.Frames))
	for i, f := range o.Frames {
		if f.Width < 0 || f.Height < 0 {
			panic(fmt.Sprintf("synth: frame %d is %dx%d; the record's width and height are u32", i, f.Width, f.Height))
		}
		if f.Pixels != nil && len(f.Pixels) != f.Width*f.Height {
			panic(fmt.Sprintf("synth: frame %d carries %d pixels, want %d for a %dx%d grid",
				i, len(f.Pixels), f.Width*f.Height, f.Width, f.Height))
		}
		data := rle256(f)
		frames[i] = Sheet256RawFrame{
			Width:    uint32(f.Width),
			Height:   uint32(f.Height),
			DataSize: uint32(len(data)),
			Data:     data,
		}
	}
	return Sheet256Raw(palette, frames, trailer)
}

// Palette256 lays out the 1024-byte palette block: 256 entries of
// [B, G, R, reserved], the reserved byte zero.
//
// THE ORDER IS BGR. It is not a convention this builder is free to choose: the
// order is pinned by a known-answer test on the shipped corpus — one file's
// palette is the canonical IBM VGA palette, which matches under BGR and does not
// match under RGB (SPR256-PAL-011). A test that only round-tripped this block
// through a decoder could not see the two swapped together, so synth_test.go
// asserts these three bytes at their offsets directly.
//
// Entries beyond len(colors) stay zero. Index 0 is the reserved
// transparent/background key of the shipped palettes and is written like any
// other entry: the reservation is a fact about the art, not about the block, and
// a pixel carrying index 0 opaquely is a case the object layer must handle.
func Palette256(colors []color.RGBA) []byte {
	if len(colors) > spr256PaletteLen/spr256EntryLen {
		panic(fmt.Sprintf("synth: palette has %d entries, want at most %d",
			len(colors), spr256PaletteLen/spr256EntryLen))
	}
	out := make([]byte, spr256PaletteLen)
	for i, c := range colors {
		e := out[i*spr256EntryLen:]
		e[0], e[1], e[2] = c.B, c.G, c.R
		// e[3], the entry's reserved byte, stays zero.
	}
	return out
}

// rle256 encodes one frame's grid as the RLE program the format defines.
//
// Two encoding choices are this builder's, both legal under the grammar and both
// visible in synth_test.go's hand-written expected bytes:
//
//   - a row that is wholly transparent joins its neighbours in ONE blank-rows
//     token (class 0x40), which is what the shipped art does and what makes the
//     third opcode reachable from a caller that supplies pixels alone;
//   - a row's trailing transparent pixels are emitted as a transparent skip
//     rather than dropped, because a row's tokens must sum to exactly width.
//
// A run longer than 63 is split across several controls: the count is six bits,
// and a builder that let it overflow would write a control byte whose class bits
// are the high bits of its own length.
//
// An empty grid — "width = 0 or height = 0 is a valid header describing an empty
// grid" — is encoded on the reading that its ROWS STILL HAVE TO CLOSE. A frame
// of no rows encodes as no bytes, since there is nothing to close; a frame of
// zero width and some rows encodes as one blank-rows token covering them,
// because the program must still yield exactly height rows and a row of no
// pixels is trivially a transparent one. The looser reading — no bytes for
// either dimension zero — reads the empty-grid sentence alone and drops the
// row-count clause beside it; both sentences are the same contract, so the
// stricter reading is the one taken.
func rle256(f Frame256) []byte {
	if f.Height <= 0 {
		return nil
	}

	var out []byte
	for row := 0; row < f.Height; {
		if !blankRow256(f, row) {
			out = append(out, rleRow256(f, row)...)
			row++
			continue
		}
		n := 1
		for row+n < f.Height && blankRow256(f, row+n) {
			n++
		}
		row += n
		for n > 0 {
			k := min(n, spr256MaxRun)
			out = append(out, spr256OpBlankRows|byte(k))
			n -= k
		}
	}
	return out
}

// rleRow256 encodes one row as alternating opaque and transparent runs, each
// split to fit a control byte's six-bit count.
func rleRow256(f Frame256, row int) []byte {
	var out []byte
	for col := 0; col < f.Width; {
		opaque := pixel256(f, row, col).Opaque
		n := 1
		for col+n < f.Width && pixel256(f, row, col+n).Opaque == opaque {
			n++
		}
		for n > 0 {
			k := min(n, spr256MaxRun)
			if opaque {
				out = append(out, spr256OpLiteral|byte(k))
				for i := 0; i < k; i++ {
					out = append(out, pixel256(f, row, col+i).Index)
				}
			} else {
				out = append(out, spr256OpTransparent|byte(k))
			}
			col += k
			n -= k
		}
	}
	return out
}

// blankRow256 reports whether every pixel of a row is transparent.
func blankRow256(f Frame256, row int) bool {
	for col := 0; col < f.Width; col++ {
		if pixel256(f, row, col).Opaque {
			return false
		}
	}
	return true
}

// pixel256 reads one grid cell, a nil Pixels being the wholly transparent grid.
func pixel256(f Frame256, row, col int) Pixel256 {
	if f.Pixels == nil {
		return Pixel256{}
	}
	return f.Pixels[row*f.Width+col]
}
