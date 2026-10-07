package sav

import (
	"encoding/binary"
	"fmt"
)

// Building is a detached projection of a top-level Building (or its decoded
// subclass). SAV-BLDG-037 fixes the two raw health words and shape members.
// AuthoredID is the complete Token+08 dword, not the runtime ID at Token+04
// (SAV-TOKEN-034, SAV-ID-015). Consumers decide which map bindings they support.
type Building struct {
	Off                  int
	Class                string
	Identity, AuthoredID uint32
	Col, Row, Kind       uint8
	Health, MaxHealth    uint16
	Width, Height        uint8
	Blocking, Attach     uint32
	ArchiveIndex         uint16
	Position             [12]byte
	RuntimeID            uint32
	Token0C              uint8
	Token0E, Token18     uint16
	Token1C, Reference   uint32
	Base52               [22]byte
	Field46              uint16
	Field48              uint8
	Tavern9C, Shop70     uint32
	OutpostWords         [4]uint32 // +84, +88, +80, +8c in serializer order
	OutpostRecords       [][8]byte
}

// Buildings reads the counted document list, validates the complete archive
// endpoint, and returns no partial population on error. A no-world save differs
// from an empty saved list; neither implies that an ALM placement was destroyed.
func (f *File) Buildings() ([]Building, bool, error) {
	doc, present, err := f.exactDocument()
	if err != nil || !present {
		return nil, present, err
	}
	out := make([]Building, 0, len(doc.buildings))
	seen := make(map[uint32]int, len(doc.buildings))
	for i, record := range doc.buildings {
		identity := record.value("Identity")
		if prior, exists := seen[identity]; exists && prior == record.Off {
			continue // One archive object referenced repeatedly is one structure.
		}
		if _, exists := seen[identity]; identity == 0 || exists {
			return nil, true, fmt.Errorf("sav: Building %d has zero or repeated identity %#x", i, identity)
		}
		seen[identity] = record.Off
		// A key colliding with an actor, Player or nested Token cannot supply
		// an unambiguous typed Building pointer in the shared identity map.
		for _, other := range doc.objects {
			key := other.value("Identity")
			if key == 0 {
				key = other.value("This")
			}
			if other != record && key == identity {
				return nil, true, fmt.Errorf("sav: ambiguous Building identity %#x", identity)
			}
		}
		position := record.Raw["Block12"]
		if len(position) != 12 {
			return nil, true, fmt.Errorf("sav: Building %d has no complete Position", i)
		}
		b := Building{Off: record.Off, Class: record.Class,
			Identity: identity, AuthoredID: record.value("T08"),
			Col: position[0], Row: position[1], Kind: uint8(record.value("B40")),
			Health: uint16(record.value("B42")), MaxHealth: uint16(record.value("B44")),
			Width: uint8(record.value("B60")), Height: uint8(record.value("B61")),
			Blocking: record.value("B64"), Attach: record.value("B68"),
			ArchiveIndex: record.Index, RuntimeID: record.value("RuntimeID"),
			Token0C: uint8(record.value("T0C")), Token0E: uint16(record.value("T0E")),
			Token18: uint16(record.value("T18")), Token1C: record.value("T1C"), Reference: record.value("Reference"),
			Field46: uint16(record.value("B46")), Field48: uint8(record.value("B48")),
			Tavern9C: record.value("T9C"), Shop70: record.value("S70"),
			OutpostWords: [4]uint32{record.value("O84"), record.value("O88"), record.value("O80"), record.value("O8C")}}
		copy(b.Position[:], position)
		copy(b.Base52[:], record.Raw["B52"])
		// The raw +52 image overlaps later scalar stores. LOAD's last writes
		// are the runtime values, not an independent second shape/mask.
		b.Base52[14], b.Base52[15] = b.Width, b.Height
		binary.LittleEndian.PutUint32(b.Base52[18:], b.Blocking)
		raw := record.Raw["O6C"]
		if len(raw) != 0 {
			b.OutpostRecords = make([][8]byte, len(raw)/8)
		}
		for j := range b.OutpostRecords {
			copy(b.OutpostRecords[j][:], raw[j*8:])
		}
		out = append(out, b)
	}
	return out, true, nil
}

// StructureCell is the structure-related projection of a complete saved cell.
// BuildingKey is Token's file-local identity, not an MFC archive index or an
// authored identifier. Zero explicitly clears the slot. Duplicate cell keys
// retain archive order for the last-write overlay (SAV-CELLLOAD-109..111).
type StructureCell struct {
	Cell                         uint16
	BaselineCost, BaselineStatic uint8
	BuildingKey                  uint32
}

func (f *File) StructureCells() ([]StructureCell, bool, error) {
	doc, present, err := f.exactDocument()
	if err != nil || !present {
		return nil, present, err
	}
	out := make([]StructureCell, doc.world.CellRecCount)
	for i := range out {
		off := doc.world.CellRecDataOff + i*cellRecLen
		out[i] = StructureCell{Cell: u16(f.Body, off), BaselineCost: f.Body[off+2],
			BaselineStatic: f.Body[off+3], BuildingKey: u32(f.Body, off+2+12)}
	}
	return out, true, nil
}
