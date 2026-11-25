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

// Format selects the map grammar of an install.
type Format uint8

const (
	FirstFormat Format = iota
	SecondFormat
)

type dialect = Format

const (
	dialectROM1 = FirstFormat
	dialectROM2 = SecondFormat
)

func selectedFormat(formats []Format) Format {
	if len(formats) > 0 {
		return formats[0]
	}
	return FirstFormat
}

// FormatOf is the grammar the map's own header declares.
func FormatOf(data []byte) Format {
	if len(data) < FileFormatVersion+4 {
		return FirstFormat
	}
	if v := binary.LittleEndian.Uint32(data[FileFormatVersion:]); v >= rom2MinVersion && v <= rom2MaxVersion {
		return SecondFormat
	}
	return FirstFormat
}

// PayloadExtent is the byte length a loader of the format reads for a record
// whose header declares size. The second game's editor declares 644 for the
// type-0 record it writes with 660 bytes, and its loader reads 660.
func PayloadExtent(f Format, typeID, size uint32) uint32 {
	if f == SecondFormat && typeID == TypeMeta {
		return rom2MetaSize
	}
	return size
}

// OpenROM2 reads a ROM2 map.
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
	head := make([]byte, MetaSize)
	copy(head, p[:rom2CountsAt])
	copy(head[MetaName:], p[rom2NameAt:rom2NameAt+MetaNameLen])
	copy(head[MetaWord70:], p[rom2WordsAt:rom2WordsAt+8])
	copy(head[MetaDesc:], p[rom2DescAt:rom2DescAt+MetaDescBlockLen])
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

// rom2RewardMarkBit is bit 8 of a second-game type-6 record's dword 0x10, the
// bit the actor builder stores into actor +0x1a0.
const rom2RewardMarkBit = 8

// RewardMarkROM2 reports whether a second-game record carries bit 8 of dword
// 0x10, which the decode keeps in DefID. The first game reads that dword as a
// definition id, so only the second game's placement asks.
func (u Unit) RewardMarkROM2() bool { return u.DefID&rom2RewardMarkBit != 0 }

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

// StructureAction is one type-11c record: the type-4 id of the structure it
// names, its gate and the index of an action node (R2-ENGINE-512).
type StructureAction struct {
	ID, Gate, Position uint32
}

// StructureActions decodes the type-11c records of the type-11 extension, in
// file order, after its 11a and 11b records. ok is false when the map carries
// no type-11 record or its length disagrees with the type-0 counts.
func (m *Map) StructureActions() (out []StructureAction, ok bool) {
	p := m.Extension[1]
	n := m.Meta.Count11
	at := uint64(rom2Type11A)*uint64(n[0]) + uint64(rom2Type11B)*uint64(n[1])
	if len(p) == 0 || uint64(len(p)) != at+uint64(rom2Type11C)*uint64(n[2]) {
		return nil, false
	}
	le := binary.LittleEndian
	for k := uint64(0); k < uint64(n[2]); k++ {
		rec := p[at+k*rom2Type11C:]
		out = append(out, StructureAction{ID: le.Uint32(rec), Gate: le.Uint32(rec[4:]), Position: le.Uint32(rec[8:])})
	}
	return out, true
}

// OutpostEntry is one type-10 record: the group number it names, the reload
// percent, its flag bits and an action index (R2-ENGINE-510).
type OutpostEntry struct {
	Group   uint32
	Percent int32
	Flags   uint32
	Action  int32
}

// OutpostEntries decodes the type-10 records in file order. ok is false when
// the map carries no type-10 record.
func (m *Map) OutpostEntries() (out []OutpostEntry, ok bool) {
	p := m.Extension[0]
	if len(p) == 0 || len(p)%rom2Type10Size != 0 {
		return nil, false
	}
	le := binary.LittleEndian
	for at := 0; at < len(p); at += rom2Type10Size {
		rec := p[at:]
		out = append(out, OutpostEntry{Group: le.Uint32(rec), Percent: int32(le.Uint32(rec[4:])),
			Flags: le.Uint32(rec[8:]), Action: int32(le.Uint32(rec[12:]))})
	}
	return out, true
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

// The second game's record layout the map editor addresses, relative to the
// record (R2-ASSET-253 to R2-ASSET-255, R2-ASSET-313 to R2-ASSET-316).
const (
	SecondUnitSize      = rom2UnitSize
	SecondUnitClass     = 0x08 // i16 lookup index
	SecondUnitField0A   = 0x0a // u16 unread
	SecondUnitFlags     = 0x0c // u32 flags; bit 0x10 selects Humans arm
	SecondUnitFlags10   = 0x10 // u32 actor flags
	SecondUnitDef       = 0x14 // u32 serverID in Units or Humans
	SecondUnitOwner     = 0x18 // owner slot, low word only
	SecondUnitSack      = 0x1c // 1-based type-8 index, 0 for none
	SecondUnitField20   = 0x20 // u32 unread
	SecondUnitHP        = 0x24 // i16 current health (-1 default)
	SecondUnitHPMax     = 0x26 // i16 maximum health (-1 default)
	SecondUnitTag       = 0x28 // low word the unit id, high word the low word of x
	SecondUnitGroup     = 0x2c
	SecondUnitFlagHuman = 0x10

	// SecondObjectExtBit is the kind bit that gives a type-4 element its
	// 8-byte extension besides kind 0x21.
	SecondObjectExtBit = rom2ObjectExt

	// SecondMarkerHead is the fixed part of a type-9 element before its
	// entries of SecondMarkerEntry bytes; the entry count ends the head.
	SecondMarkerHead  = 26
	SecondMarkerEntry = 6
	SecondMarkerMode  = 12 // u16
	SecondMarkerKey   = 14 // u32
	SecondMarkerCount = 22 // u32

	// SecondTypeLists is the id of the record that holds the 11a, 11b and
	// 11c lists, whose entry widths follow; SecondMeta11Count is the first
	// of the three type-0 words that count them.
	SecondTypeLists   = 11
	SecondType11A     = rom2Type11A
	SecondType11B     = rom2Type11B
	SecondType11C     = rom2Type11C
	SecondMeta11Count = 0x34
	// SecondMetaScriptCount is the type-0 word that equals the sum of the
	// three type-7 counts (R2-ASSET-255).
	SecondMetaScriptCount = 0x28
	// SecondMetaLevel is the type-0 dword read into Word74, the map level
	// (R2-ENGINE-1842, Medium).
	SecondMetaLevel = 0x88
)
