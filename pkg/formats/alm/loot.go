package alm

// The type-8 leaf grammar: the map's own authored loot.
//
// alm.go decodes the type-8 record as a raw body only (LootSection) and says
// so in its own comment: the payload carries no count of its own, so a
// reader cannot even tell how many records it holds without the type-0
// metadata word that sits beside it.
//
// Nothing here resolves an item code to anything. class and index are a bit
// split and nothing more; what a class or an index NAMES is a data-tier
// question this leaf does not answer.

import (
	"encoding/binary"
	"fmt"
)

const (
	lootHeadShort   = 16 // owner/X/Y, no gold (FormatVersion < lootWideHeadVersion)
	lootHeadWide    = 20 // owner/X/Y/gold (FormatVersion >= lootWideHeadVersion)
	lootElementSize = 10 // one item entry inside a record

	lootWideHeadVersion = 989 //

	lootClassShift       = 8
	lootClassMask        = 0x0f
	lootIndexMask        = 0x1f
	lootIndexMaskClass14 = 0xff
	lootClass14          = 14
)

// LootElement is one 10-byte item entry inside a loot record (spec.md
// "An element").
//
// Code is the element's leading word, carried whole; ItemCode reads its low
// 16 bits, which is the item code proper. Field04 is the next word, read only
// by the original on the stock arm and given no meaning here. TileMarkerIndex
// is a 1-based index into the map's own type-9 enchantment list — 0 names
// none. This leaf preserves the link; mapload resolves it only where the
// linked record has the item-effect shape.
type LootElement struct {
	Code            uint32 // +0x00, whole; ItemCode() takes the low 16 bits
	Field04         uint16 // +0x04, whole, unread by this build
	TileMarkerIndex uint32 // +0x06, 1-based index into TileMarkers; 0 names none
}

// ItemCode returns the element's item code: the low 16 bits of its leading
// word (spec.md "An item code").
func (e LootElement) ItemCode() uint16 { return uint16(e.Code) }

// LootRecord is one entry of the type-8 section: a head and its elements
// (spec.md "A loot record").
//
// Owner, X, Y and Gold are carried at the file's own u32 width and range-
// checked nowhere, following Unit.Owner's precedent in this reader. Gold is
// 0 on a short (16-byte) head — the file never wrote a fifth word there, so
// there is nothing to read rather than a zero this reader invents.
type LootRecord struct {
	Owner    uint32 // +0x04, 0 puts the record on the ground
	X        uint32 // +0x08, fixed point
	Y        uint32 // +0x0c, fixed point
	Gold     uint32 // +0x10, present only on the 20-byte head; 0 on the short head
	Elements []LootElement
}

// Ground reports whether the record's owner word is 0 — a ground record,
// making a sack, as opposed to a stock record naming an existing actor
// (spec.md "A ground record"/"a stock record").
func (r LootRecord) Ground() bool { return r.Owner == 0 }

func (r LootRecord) CellX() int32 { return int32(r.X) >> 8 }
func (r LootRecord) CellY() int32 { return int32(r.Y) >> 8 }

// Loot is a map's decoded type-8 payload: every loot record the section
// authors, in file order.
type Loot struct {
	Records []LootRecord
}

func ItemClass(code uint16) uint8 {
	return uint8((code >> lootClassShift) & lootClassMask)
}

func ItemIndex(code uint16) uint8 {
	if ItemClass(code) == lootClass14 {
		return uint8(code & lootIndexMaskClass14)
	}
	return uint8(code & lootIndexMask)
}

// Loot decodes the type-8 payload into its records.
//
// It is a METHOD ON THE DECODED MAP and not a second entry point, exactly as
// Script() is: a caller cannot ask for the leaf grammar of bytes this
// package never accepted as a map.
func (m *Map) Loot() (Loot, error) {
	b := m.LootSection.Body

	headSize := lootHeadShort
	if m.FormatVersion >= lootWideHeadVersion {
		headSize = lootHeadWide
	}

	n := int64(m.Meta.Word2C)
	var records []LootRecord
	off := 0
	for i := int64(0); i < n; i++ {
		if len(b)-off < headSize {
			return Loot{}, fmt.Errorf("alm: type8 record %d head does not fit in payload at offset %d (%d byte(s) left, want %d)",
				i, off, len(b)-off, headSize)
		}
		head := b[off : off+headSize]
		rec := LootRecord{
			Owner: binary.LittleEndian.Uint32(head[0x04:0x08]),
			X:     binary.LittleEndian.Uint32(head[0x08:0x0c]),
			Y:     binary.LittleEndian.Uint32(head[0x0c:0x10]),
		}
		if headSize == lootHeadWide {
			rec.Gold = binary.LittleEndian.Uint32(head[0x10:0x14])
		}
		elemN := int64(binary.LittleEndian.Uint32(head[0x00:0x04]))
		off += headSize

		span := elemN * lootElementSize
		if span > int64(len(b)-off) {
			return Loot{}, fmt.Errorf("alm: type8 record %d declares %d element(s) at offset %d, which are %d byte(s), and %d remain",
				i, elemN, off, span, len(b)-off)
		}
		elems := make([]LootElement, elemN)
		for k := range elems {
			e := b[off+int(k)*lootElementSize : off+(int(k)+1)*lootElementSize]
			elems[k] = LootElement{
				Code:            binary.LittleEndian.Uint32(e[0x00:0x04]),
				Field04:         binary.LittleEndian.Uint16(e[0x04:0x06]),
				TileMarkerIndex: binary.LittleEndian.Uint32(e[0x06:0x0a]),
			}
		}
		off += int(span)
		rec.Elements = elems
		records = append(records, rec)
	}

	if off != len(b) {
		return Loot{}, fmt.Errorf("alm: type8 walk consumed %d of %d payload bytes", off, len(b))
	}
	return Loot{Records: records}, nil
}
