package mapedit

import (
	"encoding/binary"
	"fmt"

	"againrom/pkg/formats/alm"
)

// The type-6 record: a fixed 70 bytes whose first two words are the
// fixed-point coordinates, the unit id a clone renumbers, and the type-0 word
// that says how many records there are. The fields a caller may set by name are
// the table in field.go; every other byte is carried.
const (
	unitTypeID = 6

	unitRecordSize = 70
	unitX          = 0x00
	unitY          = 0x04
	unitID         = 0x40
	unitLootLink   = 0x18 // 1-based index into the type-8 array, 0 for none

	metaCount6 = 0x24
)

// UnitCount returns the number of type-6 records in the map, and zero on a map
// that carries no type-6 record at all.
//
// It is read from #type6 at type-0 +0x24, not from the type-6 payload's length.
// With the record absent the count word is inert rather than authoritative: the
// engine's unit case never runs, so the figure names records nothing will read,
// and the decoded view has no units there either.
func (e *Editor) UnitCount() int { return e.Count(KindUnit) }

// UnitRecord returns the 70 bytes of the record at file-order index i as a fresh
// copy, or an error and no record if there is no such index. The copy is the
// point: the cheap way to a record this model will accept is to clone one that
// is already in the map, and a window into the buffer would edit the map
// without going through an edit.
func (e *Editor) UnitRecord(i int) ([]byte, error) { return e.Record(KindUnit, i) }

// PlaceUnit appends record, exactly 70 bytes copied verbatim, after the last
// type-6 record and returns its file-order index. A map that carries no type-6
// record is a rejection and not the first unit of a new roster.
func (e *Editor) PlaceUnit(record []byte) (int, error) { return e.Place(KindUnit, record) }

// CloneUnit is CloneRecord for a unit; renumberUnit gives the copy its unit id.
func (e *Editor) CloneUnit(i int, x, y uint32) (int, error) { return e.CloneRecord(KindUnit, i, x, y) }

// MoveUnit sets the X and Y words of unit i, leaving its other 62 bytes as
// they arrived.
func (e *Editor) MoveUnit(i int, x, y uint32) error { return e.Move(KindUnit, i, x, y) }

// DeleteUnit removes unit i, decrementing #type6 and shifting every later
// index down by one: the mirror of PlaceUnit.
func (e *Editor) DeleteUnit(i int) error { return e.Delete(KindUnit, i) }

// UnitFields returns a copy of the settable type-6 fields in window order.
func UnitFields() []UnitField { return append([]UnitField(nil), unitFields...) }

// UnitFieldByKey returns the type-6 field named key.
func UnitFieldByKey(key string) (UnitField, bool) { return fieldByKey(unitFields, key) }

// UnitFieldValue reads the field named key from unit i.
func (e *Editor) UnitFieldValue(i int, key string) (int64, error) {
	return e.FieldValue(KindUnit, i, key)
}

// SetUnitField writes v into the field named key of unit i as one edit.
func (e *Editor) SetUnitField(i int, key string, v int64) error {
	return e.SetField(KindUnit, i, key, v)
}

// renumberUnit sets a clone's unit id to one above the highest of every
// type-6 unit id and every unit id a Target_Unit parameter of the type-7
// script names, so a script id whose unit was deleted stays unclaimed. The
// group id and every other byte stay the source's. A map whose next id would
// pass 0xFFFF is refused: the u16 field has no next value (ALM-UNIT-048).
func renumberUnit(e *Editor, w walk, rec []byte) error {
	highest := uint32(0)
	for k := 0; k < w.n(); k++ {
		at, _ := w.at(k)
		highest = max(highest, uint32(binary.LittleEndian.Uint16(e.data[at+unitID:])))
	}
	scripted, err := e.highestScriptUnit()
	if err != nil {
		return err
	}
	highest = max(highest, scripted)
	if highest >= 0xFFFF {
		return fmt.Errorf("the highest unit id is already %d", highest)
	}
	binary.LittleEndian.PutUint16(rec[unitID:], uint16(highest+1))
	return nil
}

// Script parameter type code of a Target_Unit value, and the first value of the
// hero band, which names the party and no placed unit.
const (
	scriptUnitParam = 4
	scriptHeroBand  = 10001
)

// highestScriptUnit is the highest unit id a Target_Unit parameter of the type-7
// script names below the hero band, 0 when the map has no script or none.
func (e *Editor) highestScriptUnit() (uint32, error) {
	return e.highestScriptTarget(scriptUnitParam, scriptHeroBand)
}

func (e *Editor) highestScriptTarget(param uint32, limit uint64) (uint32, error) {
	m, err := e.Map()
	if err != nil {
		return 0, err
	}
	sc, err := m.Script()
	if err != nil {
		return 0, err
	}
	highest := uint32(0)
	for _, nodes := range [][]alm.ScriptNode{sc.Actions, sc.Conditions} {
		for _, n := range nodes {
			for k, t := range n.Type {
				if t == param && uint64(n.Value[k]) < limit {
					highest = max(highest, n.Value[k])
				}
			}
		}
	}
	return highest, nil
}

// HighestGroup is the highest group id a type-6 record holds or a
// Target_Group parameter of the type-7 script names, 0 when there is none: a
// group id above it is one no record and no script of the map uses.
func (e *Editor) HighestGroup() (uint64, error) {
	m, err := e.Map()
	if err != nil {
		return 0, err
	}
	highest, err := e.highestScriptTarget(paramGroup, 1<<32)
	if err != nil {
		return 0, err
	}
	out := uint64(highest)
	for _, u := range m.Units {
		out = max(out, uint64(u.GroupID))
	}
	return out, nil
}

// sizeWordPart is the part that moves a located record's own payloadSize word,
// in its 20-byte header, by delta bytes.
func (e *Editor) sizeWordPart(sp span, delta int) part {
	return e.wordPart(sp.headerOff+recPayloadSize, delta)
}

// wordPart builds the part that adds delta to the little-endian word at absolute
// offset at. Every caller adds or removes one record's or element's worth after
// establishing that the word is large enough to take it.
func (e *Editor) wordPart(at, delta int) part {
	var img [4]byte
	binary.LittleEndian.PutUint32(img[:], uint32(int64(e.u32(at))+int64(delta)))
	return replace(at, e.data[at:at+len(img)], img[:])
}
