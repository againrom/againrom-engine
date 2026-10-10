package alm

import (
	"encoding/binary"
	"fmt"
	"slices"
)

const (
	rom2MinVersion = 1300
	rom2MaxVersion = 1600

	rom2MetaSize   = 660
	rom2UnitSize   = 48
	rom2ObjectExt  = 0x1000000
	rom2Type10Size = 16
	rom2Type11A    = 12
	rom2Type11B    = 84
	rom2Type11C    = 12
	rom2Type12Head = 28
	rom2Type12Elem = 28

	rom2CountsAt = 0x30
	rom2NameAt   = 0x44
	rom2WordsAt  = 0x84
	rom2ExtraAt  = 0x8c
	rom2DescAt   = 0x94
)

type dialect uint8

const (
	dialectROM1 dialect = iota
	dialectROM2
)

// OpenROM2 reads a ROM2 map. The grammar is the one the preserved versions
// share; other versions are refused.
func OpenROM2(data []byte) (*Map, error) { return open(data, dialectROM2) }

func checkVersion(v uint32, d dialect) error {
	if d == dialectROM2 {
		if v < rom2MinVersion || v > rom2MaxVersion {
			return fmt.Errorf("alm: formatVersion %d outside the ROM2 range %d to %d", v, rom2MinVersion, rom2MaxVersion)
		}
		return nil
	}
	if v == skipHdrVersion {
		return fmt.Errorf("alm: formatVersion %d is the record-header-skipping dialect, which this reader does not implement", v)
	}
	if v > maxFormatVersion {
		return fmt.Errorf("alm: formatVersion %d, want at most %d", v, maxFormatVersion)
	}
	return nil
}

func decodeMetaROM2(p []byte) (angle float32, info Info, meta Meta, err error) {
	if len(p) != rom2MetaSize {
		return 0, info, meta, fmt.Errorf("alm: type-0 payload is %d bytes, want %d", len(p), rom2MetaSize)
	}
	head := make([]byte, metaSize)
	copy(head, p[:rom2CountsAt])
	copy(head[metaName:], p[rom2NameAt:rom2NameAt+metaNameLen])
	copy(head[metaWord70:], p[rom2WordsAt:rom2WordsAt+8])
	copy(head[metaDesc:], p[rom2DescAt:rom2DescAt+metaBlockLen])
	angle, info, meta, err = decodeMetaPayload(head)
	if err != nil {
		return 0, info, meta, err
	}
	le := binary.LittleEndian
	meta.Count10 = le.Uint32(p[rom2CountsAt:])
	for i := range meta.Count11 {
		meta.Count11[i] = le.Uint32(p[rom2CountsAt+4+4*i:])
	}
	meta.Count12 = le.Uint32(p[rom2CountsAt+16:])
	meta.Extra = [2]uint32{le.Uint32(p[rom2ExtraAt:]), le.Uint32(p[rom2ExtraAt+4:])}
	return angle, info, meta, nil
}

func (m *Map) decodeExtension(f recordFrame) error {
	n10 := uint64(m.Meta.Count10)
	n11 := m.Meta.Count11
	n12 := uint64(m.Meta.Count12)
	want := [3]uint64{
		rom2Type10Size * n10,
		rom2Type11A*uint64(n11[0]) + rom2Type11B*uint64(n11[1]) + rom2Type11C*uint64(n11[2]),
		rom2Type12Head + rom2Type12Elem*n12,
	}
	for i := range want {
		if !f.extPresent[i] {
			continue
		}
		if uint64(len(f.ext[i])) != want[i] {
			return fmt.Errorf("alm: type%d payload is %d bytes, want %d", 10+i, len(f.ext[i]), want[i])
		}
		m.Extension[i] = cloneBytes(f.ext[i])
	}
	return nil
}

func (m *Map) decodeUnitsROM2(p []byte) error {
	n := int(m.Meta.Count6)
	if uint64(len(p)) != uint64(rom2UnitSize)*uint64(m.Meta.Count6) {
		return fmt.Errorf("alm: type6 payload is %d bytes, want %d (%d*#type6)", len(p), int64(rom2UnitSize)*int64(m.Meta.Count6), rom2UnitSize)
	}
	units := make([]Unit, n)
	le := binary.LittleEndian
	for i := range units {
		rec := p[i*rom2UnitSize : (i+1)*rom2UnitSize]
		hp := int16(le.Uint16(rec[0x24:0x26]))
		units[i] = Unit{
			X:            le.Uint32(rec[0x00:]),
			Y:            le.Uint32(rec[0x04:]),
			ClassID:      int16(le.Uint16(rec[0x08:])),
			ClassSubID:   le.Uint16(rec[0x0a:]),
			Flags:        le.Uint32(rec[0x0c:]),
			DefID:        le.Uint32(rec[0x10:]),
			ServerID:     le.Uint32(rec[0x14:]),
			Owner:        le.Uint32(rec[0x18:]),
			CurrentHP:    hp,
			HasCurrentHP: hp != -1,
			UnitID:       le.Uint16(rec[0x28:]),
			GroupID:      le.Uint32(rec[0x2c:]),
		}
	}
	m.Units = units
	m.AuthoredUnits = slices.Clone(units)
	return nil
}

// MusicArea is one type-12 record: centre and radius in tiles and four
// themes, each an index into the mission list or -1 for none (R2-ASSET-084).
type MusicArea struct {
	X, Y, Radius int32
	Themes       [4]int32
}

// MusicAreas decodes the type-12 extension: the head record, then the area
// records in file order. ok is false when the map carries no type-12 record
// or its length is not a whole number of records.
func (m *Map) MusicAreas() (head MusicArea, areas []MusicArea, ok bool) {
	p := m.Extension[2]
	if len(p) < rom2Type12Head || (len(p)-rom2Type12Head)%rom2Type12Elem != 0 {
		return MusicArea{}, nil, false
	}
	read := func(rec []byte) MusicArea {
		le := binary.LittleEndian
		a := MusicArea{X: int32(le.Uint32(rec)), Y: int32(le.Uint32(rec[4:])), Radius: int32(le.Uint32(rec[8:]))}
		for i := range a.Themes {
			a.Themes[i] = int32(le.Uint32(rec[12+4*i:]))
		}
		return a
	}
	head = read(p[:rom2Type12Head])
	for at := rom2Type12Head; at < len(p); at += rom2Type12Elem {
		areas = append(areas, read(p[at:at+rom2Type12Elem]))
	}
	return head, areas, true
}
