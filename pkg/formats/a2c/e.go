package a2c

import (
	"encoding/binary"
	"math/bits"
)

// eField is one of section E's sixteen fields: its offset, its width and
// the key bit that puts a pad byte before it (R2-SESSION-022, R2-SESSION-517).
type eField struct{ at, width, pad int }

var eFields = [16]eField{
	{0x00, 4, 0}, {0x04, 4, 1}, {0x08, 4, 2}, {0x0c, 4, 3}, {0x10, 4, 4},
	{0x14, 1, 5}, {0x15, 1, 6}, {0x16, 1, 7}, {0x17, 1, 8},
	{0x18, 4, 14}, {0x1c, 4, 13},
	{0x20, 4, 9}, {0x24, 4, 10}, {0x28, 4, 11}, {0x2c, 4, 12}, {0x30, 4, 13},
}

// eValue is field i's decoded value from its stored value s and the fields
// decoded before it: the reader's chain (R2-SESSION-517).
func eValue(i int, s uint32, d *[16]uint32) uint32 {
	switch i {
	case 0:
		return s ^ 0x1529251
	case 1:
		return s + d[0]*5 + 0x13141516
	case 2:
		return s + d[1]*7 + 0xabcdef
	case 3:
		return s ^ 0x17ff12aa
	case 4:
		return s + d[0]*3 - 0x21524542
	case 5:
		return uint32(uint8(s) + uint8(d[4]*0x11+d[0]*0x13))
	case 6:
		return uint32(uint8(s) + uint8(d[5])*3)
	case 7:
		return uint32(uint8(s) + uint8(d[5]) + uint8(d[6])*5)
	case 8:
		return uint32(uint8(s) + uint8(d[5])*7 + uint8(d[7])*9)
	case 9:
		return s - 0x10121974
	case 10:
		return s
	case 11:
		return s ^ 0xdadedade
	}
	return s - d[i-1]*0x771
}

// eStored is field i's stored value: the writer's chain, the exact inverse
// of eValue.
func eStored(i int, v uint32, d *[16]uint32) uint32 {
	switch i {
	case 0:
		return v ^ 0x1529251
	case 1:
		return v - d[0]*5 - 0x13141516
	case 2:
		return v - d[1]*7 - 0xabcdef
	case 3:
		return v ^ 0x17ff12aa
	case 4:
		return v - d[0]*3 + 0x21524542
	case 5:
		return uint32(uint8(v) - uint8(d[4]*0x11+d[0]*0x13))
	case 6:
		return uint32(uint8(v) - uint8(d[5])*3)
	case 7:
		return uint32(uint8(v) - uint8(d[5]) - uint8(d[6])*5)
	case 8:
		return uint32(uint8(v) - uint8(d[5])*7 - uint8(d[7])*9)
	case 9:
		return v + 0x10121974
	case 10:
		return v
	case 11:
		return v ^ 0xdadedade
	}
	return v + d[i-1]*0x771
}

func (e E) fields() [16]uint32 {
	var d [16]uint32
	copy(d[0:4], e.Player[:])
	d[4] = e.Money
	for i, b := range e.Base {
		d[5+i] = uint32(b)
	}
	d[9], d[10] = e.Spells, e.Unit1C
	copy(d[11:], e.SkillXP[:])
	return d
}

func eOf(d [16]uint32) E {
	var e E
	copy(e.Player[:], d[0:4])
	e.Money = d[4]
	for i := range e.Base {
		e.Base[i] = uint8(d[5+i])
	}
	e.Spells, e.Unit1C = d[9], d[10]
	copy(e.SkillXP[:], d[11:])
	return e
}

// plain is the decoded 0x34 bytes the checksum covers.
func ePlain(d *[16]uint32) []byte {
	p := make([]byte, ESize)
	for i, f := range eFields {
		if f.width == 1 {
			p[f.at] = uint8(d[i])
		} else {
			binary.LittleEndian.PutUint32(p[f.at:], d[i])
		}
	}
	return p
}

// readE reads section E's fields from b, skipping a pad byte before each
// field whose key bit is set; the header length is not used. It answers the
// fields and the bytes it read.
func readE(b []byte, key uint16, sum uint32) (E, int, error) {
	var d [16]uint32
	at := 0
	for i, f := range eFields {
		if key&(1<<f.pad) != 0 {
			at++
		}
		if len(b)-at < f.width {
			return E{}, 0, ErrBody
		}
		var s uint32
		if f.width == 1 {
			s = uint32(b[at])
		} else {
			s = binary.LittleEndian.Uint32(b[at:])
		}
		at += f.width
		d[i] = eValue(i, s, &d)
	}
	if Checksum(ePlain(&d)) != sum {
		return E{}, 0, ErrBody
	}
	return eOf(d), at, nil
}

// appendE is section E of e: its header, whose length is 0x34 plus the key's
// bit count (R2-SESSION-513), then each stored field after a zero pad byte
// where its key bit is set.
func appendE(out []byte, e E) []byte {
	d := e.fields()
	plain := ePlain(&d)
	key := sectionKey(plain)
	out = binary.LittleEndian.AppendUint32(out, TagE)
	out = binary.LittleEndian.AppendUint32(out, uint32(ESize+bits.OnesCount16(key)))
	out = binary.LittleEndian.AppendUint16(out, 0)
	out = binary.LittleEndian.AppendUint16(out, key)
	out = binary.LittleEndian.AppendUint32(out, Checksum(plain))
	for i, f := range eFields {
		if key&(1<<f.pad) != 0 {
			out = append(out, 0)
		}
		s := eStored(i, d[i], &d)
		if f.width == 1 {
			out = append(out, uint8(s))
		} else {
			out = binary.LittleEndian.AppendUint32(out, s)
		}
	}
	return out
}
