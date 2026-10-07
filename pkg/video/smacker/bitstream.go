// Ported from libsmacker 1.2.0 (Greg Kennedy, LGPL-2.1-or-later): the
// LSB-first bitstream reader (smk_bs_read_1 / smk_bs_read_8 in smacker.c).
// See doc.go for the package-wide attribution and THIRD_PARTY_NOTICES.md for
// the notice entry.
package smacker

import "io"

// bitReader ports libsmacker's smk_bit_t / smk_bs_read_1 / smk_bs_read_8. It
// reads least-significant-bit-first, matching the original bitstream: bit 0
// of the current byte is consumed before bit 1, and a byte read spanning a
// bit boundary combines the low bits of the current byte with the high bits
// of the next one exactly as smk_bs_read_8 does.
type bitReader struct {
	data []byte
	pos  int  // index of the byte currently being consumed
	bit  uint // next bit to read within data[pos], 0..7
}

func newBitReader(b []byte) *bitReader {
	return &bitReader{data: b}
}

// readBit ports smk_bs_read_1.
func (r *bitReader) readBit() (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	v := int(r.data[r.pos]>>r.bit) & 1
	if r.bit == 7 {
		r.pos++
		r.bit = 0
	} else {
		r.bit++
	}
	return v, nil
}

// readByte ports smk_bs_read_8, including its unaligned-read combination of
// the tail of the current byte with the head of the next one.
func (r *bitReader) readByte() (int, error) {
	need := r.pos
	if r.bit > 0 {
		need++
	}
	if need >= len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	if r.bit == 0 {
		v := int(r.data[r.pos])
		r.pos++
		return v, nil
	}
	v := int(r.data[r.pos]) >> r.bit
	r.pos++
	v |= (int(r.data[r.pos]) << (8 - r.bit)) & 0xFF
	return v, nil
}
