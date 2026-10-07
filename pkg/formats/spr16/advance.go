package spr16

import (
	"encoding/binary"
	"fmt"
)

// advanceEntrySize is one sidecar entry: a u32 LE.
const advanceEntrySize = 4

// Advances decodes a font's advance-table sidecar — the second node of the two a
// font is made of, <base>.dat beside <base>.16 or <base>.16a. The stream is one
// u32 little-endian value per atlas record, in record order, with no header, no
// trailer and no padding: a 224-record atlas has an 896-byte sidecar and a
// 64-record one a 256-byte sidecar.
//
// Each value is how far the pen moves for that glyph, and it is NOT the record's
// width — the record is a fixed cell and the proportional metric lives here.
// The two nodes are separate files with no cross-reference between them, so
// nothing in this stream says which atlas it belongs to and nothing here can
// check it: the sidecar has no self-describing length, and its one validity
// condition — that its entry count equals the atlas's record count — is a
// property of the PAIR, checked by whatever loads both.
//
// The caps are the container's own (see records): a table longer than
// maxFrameCount cannot belong to any atlas this package will decode, and an
// entry over maxDimension cannot be an advance inside a frame this package will
// accept. Both refuse rather than clamp, so a breach is visible; the caps also
// keep every returned value inside int arithmetic that cannot overflow when a
// consumer sums it over a string. Malformed input yields a non-nil error and no
// slice — never a panic and never a partial result.
func Advances(data []byte) ([]int, error) {
	if len(data)%advanceEntrySize != 0 {
		return nil, fmt.Errorf("spr16: advance table of %d bytes is not a whole number of %d-byte entries",
			len(data), advanceEntrySize)
	}
	count := len(data) / advanceEntrySize
	if count > maxFrameCount {
		return nil, fmt.Errorf("spr16: advance table of %d entries exceeds the %d cap", count, maxFrameCount)
	}
	out := make([]int, count)
	for i := range out {
		v := binary.LittleEndian.Uint32(data[i*advanceEntrySize:])
		if v > maxDimension {
			return nil, fmt.Errorf("spr16: advance %d of entry %d exceeds the %d cap", v, i, maxDimension)
		}
		out[i] = int(v)
	}
	return out, nil
}
