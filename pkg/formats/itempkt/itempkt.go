// Package itempkt decodes data/itemname.pkt records (R2-ASSET-207).
// The nine header bytes and record bytes +2, +3 and +5 have no decoded meaning.
// Records are bounded by the input, not the shipped count of 491. Tag 1 is a
// little-endian u32 price; all other tags carry one opaque byte. No flags
// beyond their raw bits or non-price attribute meanings are inferred.
package itempkt

import (
	"encoding/binary"
	"fmt"
)

type Record struct {
	Flags    byte
	Price    uint32
	HasPrice bool
}

func Parse(b []byte) (map[uint16]Record, error) {
	if len(b) < 9 {
		return nil, fmt.Errorf("itemname.pkt: short header")
	}
	out := make(map[uint16]Record)
	for at := 9; at < len(b); {
		if len(b)-at < 7 {
			return nil, fmt.Errorf("itemname.pkt: short record at %d", at)
		}
		n := int(b[at+6])
		if n > len(b)-at-7 {
			return nil, fmt.Errorf("itemname.pkt: short attributes at %d", at)
		}
		r := Record{Flags: b[at+4]}
		attrs := b[at+7 : at+7+n]
		for len(attrs) > 0 {
			size := 2
			if attrs[0] == 1 {
				size = 5
			}
			if len(attrs) < size {
				return nil, fmt.Errorf("itemname.pkt: short tag %d at %d", attrs[0], at)
			}
			if attrs[0] == 1 {
				r.Price, r.HasPrice = binary.LittleEndian.Uint32(attrs[1:5]), true
			}
			attrs = attrs[size:]
		}
		out[binary.LittleEndian.Uint16(b[at:at+2])] = r
		at += 7 + n
	}
	return out, nil
}
