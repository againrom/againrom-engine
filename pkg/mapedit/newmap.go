package mapedit

import (
	"encoding/binary"
	"fmt"

	"againrom/pkg/formats/alm"
)

// NewMap starts an edit session on a new map. The map is built by the one
// constructor in pkg/formats/alm and opened through New like any other
// stream, so the session has no state of its own beyond the bytes.
func NewMap(spec alm.NewMapSpec) (*Editor, error) {
	doc, err := alm.NewMap(spec)
	if err != nil {
		return nil, fmt.Errorf("mapedit: %w", err)
	}
	return New(doc.Write())
}

// Fill words of a new type-6 record that are not zero. +0x20 to +0x26 are four
// u16 whose absent value is 0xFFFF (ALM-TAILU16-082); the health override at
// +0x20 reads -1 as "derive from the class" (UNIT-PLACE-034).
const unitAbsentU16 = 0xFFFF

var unitAbsentWords = [...]int{alm.UnitHP, alm.UnitHP + 2, alm.UnitMana, alm.UnitMana + 2}

// NewUnitRecord builds one type-6 record for a unit of the given class key and
// secondary key at the cell words x and y, owned by roster slot owner. Every override byte is
// absent (zero, or 0xFFFF for the four u16), the definition id is 0 so the
// class key resolves, the stock link is 0 and the group id is 0. The unit id
// is the next free one, by the rule a clone follows (DIV-3209). The record is
// not placed; PlaceUnit places it.
func (e *Editor) NewUnitRecord(x, y uint32, class int16, secondary uint16, owner uint32) ([]byte, error) {
	w, err := e.walkSection(KindUnit, "build a unit")
	if err != nil {
		return nil, err
	}
	rec := make([]byte, alm.UnitSize)
	le := binary.LittleEndian
	le.PutUint32(rec[alm.UnitX:], x)
	le.PutUint32(rec[alm.UnitY:], y)
	le.PutUint16(rec[alm.UnitClass:], uint16(class))
	le.PutUint16(rec[alm.UnitSubID:], secondary)
	le.PutUint32(rec[alm.UnitOwner:], owner)
	for _, off := range unitAbsentWords {
		le.PutUint16(rec[off:], unitAbsentU16)
	}
	if err := renumberUnit(e, w, rec); err != nil {
		return nil, fmt.Errorf("mapedit: cannot build a unit: %w", err)
	}
	if owner < 1 {
		return nil, fmt.Errorf("mapedit: cannot build a unit: owner slot %d", owner)
	}
	return rec, nil
}
