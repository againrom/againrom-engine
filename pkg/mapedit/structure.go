package mapedit

import "encoding/binary"

// The type-4 record: X, Y and kind as u32 words, then +0x0c u16, +0x0e u32 and
// +0x12 u16, 20 bytes; a record whose kind is 0x21 carries 8 more bytes whose
// low bytes at +0x14 and +0x18 override the footprint (ALM-OBJ-019,
// ALM-OBJ-034, ALM-CLS-036). +0x0c, +0x0e and +0x12 are carried and offered
// by no field row.
const (
	structureTypeID   = 4
	structureBaseSize = 20
	structureExtSize  = 8
	structureExtKind  = 0x21
	structureKind     = 0x08

	metaCount4 = 0x20
)

func structureLength(_ *Editor, b []byte) int {
	if len(b) < structureBaseSize {
		return -1
	}
	if binary.LittleEndian.Uint32(b[structureKind:]) != structureExtKind {
		return structureBaseSize
	}
	return fixedLength(b, structureBaseSize+structureExtSize)
}

// reshapeStructure gives a record whose kind became 0x21 a zero extension and
// drops the extension of one whose kind left 0x21. Zero override bytes select
// the table footprint (TERR-STRUCT-090), so the added extension changes no
// footprint.
func reshapeStructure(rec []byte) []byte {
	ext := binary.LittleEndian.Uint32(rec[structureKind:]) == structureExtKind
	switch {
	case ext && len(rec) == structureBaseSize:
		return append(rec, make([]byte, structureExtSize)...)
	case !ext && len(rec) > structureBaseSize:
		return rec[:structureBaseSize]
	}
	return rec
}

var structureKindField = Field{Key: "kind", Label: "Kind (structures.reg ID)", Offset: structureKind, Width: 4, Min: 1, Max: 255, Check: CheckStructureKind}

var structureExtFields = []Field{
	{Key: "width", Label: "Footprint width (0 = table)", Offset: 0x14, Width: 1, Min: 0, Max: 255},
	{Key: "height", Label: "Footprint height (0 = table)", Offset: 0x18, Width: 1, Min: 0, Max: 255},
}

func structureFields(_ *Editor, rec []byte) []Field {
	if len(rec) == structureBaseSize {
		return []Field{structureKindField}
	}
	return append([]Field{structureKindField}, structureExtFields...)
}
