package synth

import "encoding/binary"

// The .16 byte-control grammar this builder emits: op = c >> 6, n = c & 0x3F.
// 00 literal n bytes (two 4-bit pixels each, LOW nibble first, a zero HIGH
// nibble on the run's LAST byte being a pad and not a pixel), 01 blank rows,
// 10 skip n pixels.
const (
	font16Literal   = 0x00
	font16BlankRows = 0x40
	font16Skip      = 0x80
	font16MaxCount  = 0x3F
)

// Font16Glyph describes one record of a synthetic .16 font atlas.
//
// Ink reports the 4-bit level at (x, y) and whether that pixel is painted at
// all; a nil Ink is a wholly blank record, which is what record 0 is in every
// shipped atlas. Advance is the value the sidecar carries for this record — the
// pen distance, which the format never ties to Width.
type Font16Glyph struct {
	Width   int
	Height  int
	Advance uint32
	Ink     func(x, y int) (level uint8, painted bool)
}

// Font16 builds the TWO nodes a font is made of: the `.16` atlas stream and its
// `<base>.dat` sidecar, in that order.
//
// The atlas is well-formed by construction — every record's program closes on
// its own cell with no slack, and the trailer carries the record count — so a
// fixture that needs a malformed one is a hand-written stream, not this.
//
// Levels are emitted as given, level 0 included: it is a painted pixel in this
// format, and a builder that quietly promoted it to 1 would make the one clause
// no shipped file exercises untestable.
func Font16(glyphs []Font16Glyph) (atlas, advances []byte) {
	for _, g := range glyphs {
		block := font16Block(g)
		atlas = binary.LittleEndian.AppendUint32(atlas, uint32(g.Width))
		atlas = binary.LittleEndian.AppendUint32(atlas, uint32(g.Height))
		atlas = binary.LittleEndian.AppendUint32(atlas, uint32(len(block)))
		atlas = append(atlas, block...)
		advances = binary.LittleEndian.AppendUint32(advances, g.Advance)
	}
	atlas = binary.LittleEndian.AppendUint32(atlas, uint32(len(glyphs)))
	return atlas, advances
}

// font16Block encodes one record's pixel program row by row: each row is cut
// into runs of painted and unpainted pixels, the unpainted ones emitted as skips
// and the painted ones as literals.
func font16Block(g Font16Glyph) []byte {
	if g.Width <= 0 || g.Height <= 0 {
		return []byte{font16BlankRows}
	}
	at := func(x, y int) (uint8, bool) {
		if g.Ink == nil {
			return 0, false
		}
		lv, on := g.Ink(x, y)
		return lv & 0x0F, on
	}

	blankRow := func(y int) bool {
		for x := 0; x < g.Width; x++ {
			if _, on := at(x, y); on {
				return false
			}
		}
		return true
	}

	var out []byte
	for y := 0; y < g.Height; y++ {
		// A run of wholly empty rows goes out as blank-rows ops, which is what
		// makes a blank record one control byte — the shape record 0 has in
		// every shipped atlas.
		if blankRow(y) {
			n := 1
			for y+n < g.Height && blankRow(y+n) {
				n++
			}
			y += n - 1
			for n > 0 {
				k := min(n, font16MaxCount)
				out = append(out, font16BlankRows|byte(k))
				n -= k
			}
			continue
		}
		for x := 0; x < g.Width; {
			_, painted := at(x, y)
			run := 1
			for x+run < g.Width {
				if _, p := at(x+run, y); p != painted {
					break
				}
				run++
			}
			if !painted {
				for run > 0 {
					n := min(run, font16MaxCount)
					out = append(out, font16Skip|byte(n))
					run -= n
					x += n
				}
				continue
			}
			for run > 0 {
				n := min(run, font16MaxCount)
				// An even-length run whose last pixel is level 0 would put a
				// zero in the final byte's HIGH nibble, where the grammar reads
				// a pad rather than a pixel. Shorten it by one and let the odd
				// tail carry that pixel in a low nibble of its own.
				if n%2 == 0 {
					if lv, _ := at(x+n-1, y); lv == 0 {
						n--
					}
				}
				out = append(out, font16Literal|byte((n+1)/2))
				for i := 0; i < n; i += 2 {
					lo, _ := at(x+i, y)
					var hi uint8
					if i+1 < n {
						hi, _ = at(x+i+1, y)
					}
					out = append(out, lo|hi<<4)
				}
				run -= n
				x += n
			}
		}
	}
	return out
}
