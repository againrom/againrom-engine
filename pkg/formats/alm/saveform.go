package alm

import (
	"encoding/binary"
)

// The unit fields the save form touches (ALM-UNIT-040): the owner slot, the
// loot link written as the loot index plus one, and the script unit id.
const (
	unitOwnerOff = 0x14
	unitLootOff  = 0x18
	unitIDOff    = 0x40

	// saveFormVersion is the file version the original editor writes
	// (ALM-SAVE-255). The layout rules below hold from it up; a file below it
	// is returned as it is.
	saveFormVersion = 990

	// firstSlot is the player an unmatched owner is bound to (ALM-SAVE-256).
	firstSlot = 1

	// orphanUnitID is the unit id a loot record with no owner unit moves to
	// (ALM-SAVE-255).
	orphanUnitID = 1
)

// SaveForm returns the bytes the original editor's save writes for the map in
// data; every other byte rides the input, so a map the editor wrote itself
// comes back equal. A file below version 990 is returned as it is.
//
//   - A unit whose owner matches no type-5 record (ids 1 to the record count)
//     is owned by player 1 (ALM-SAVE-256).
//   - A unit's loot link outside 1 to the loot count is written as 0, and a
//     loot record a unit links takes that unit's id as owner (ALM-SAVE-255).
//   - A loot record with an owner takes the owner unit's position; one whose
//     owner is no unit moves to the unit with id 1 (ALM-SAVE-255, DIV-3252).
func SaveForm(data []byte) ([]byte, error) {
	doc, err := OpenDocument(data)
	if err != nil {
		return nil, err
	}
	out := doc.Write()
	if doc.Version() < saveFormVersion {
		return out, nil
	}
	m, err := doc.Map()
	if err != nil {
		return nil, err
	}
	pay := func(id uint32) []byte {
		var found []byte
		for i, tid := range doc.RecordTypeIDs() {
			if tid == id {
				sp := doc.spans[i]
				start := sp.headerOff + recordHeaderSize
				found = out[start : start+sp.payloadSize]
			}
		}
		return found
	}
	units, lootBytes := pay(6), pay(8)
	le := binary.LittleEndian
	nUnits := len(m.Units)
	players := len(m.Groups)

	var heads []int
	if lootBytes != nil {
		heads, err = lootHeads(m, lootBytes)
		if err != nil {
			return nil, err
		}
	}
	unitOff := func(i int) int { return i * unitRecordSize }

	for i := 0; i < nUnits; i++ {
		u := units[unitOff(i):]
		if players > 0 {
			if o := le.Uint32(u[unitOwnerOff:]); o < 1 || o > uint32(players) {
				le.PutUint32(u[unitOwnerOff:], firstSlot)
			}
		}
		link := le.Uint32(u[unitLootOff:])
		if link < 1 || link > uint32(len(heads)) {
			le.PutUint32(u[unitLootOff:], 0)
			continue
		}
		le.PutUint32(lootBytes[heads[link-1]+lootOwnerOff:], uint32(le.Uint16(u[unitIDOff:])))
	}

	unitByID := func(id uint32) int {
		for i := 0; i < nUnits && id <= 0xffff; i++ {
			if uint32(le.Uint16(units[unitOff(i)+unitIDOff:])) == id {
				return i
			}
		}
		return -1
	}
	for j, h := range heads {
		owner := le.Uint32(lootBytes[h+lootOwnerOff:])
		if owner == 0 {
			continue
		}
		at := unitByID(owner)
		if at < 0 {
			if at = unitByID(orphanUnitID); at < 0 {
				continue
			}
			le.PutUint32(lootBytes[h+lootOwnerOff:], orphanUnitID)
			le.PutUint32(units[unitOff(at)+unitLootOff:], uint32(j+1))
		}
		copy(lootBytes[h+lootXOff:h+lootXOff+8], units[unitOff(at):unitOff(at)+8])
	}
	return out, nil
}

// Type-8 head offsets the save form uses (the Loot reader's layout).
const (
	lootOwnerOff = 0x04
	lootXOff     = 0x08
)

// lootHeads returns the payload offset of each type-8 record's head, walked
// as Loot walks them.
func lootHeads(m *Map, b []byte) ([]int, error) {
	head := lootHeadShort
	if m.FormatVersion >= lootWideHeadVersion {
		head = lootHeadWide
	}
	var heads []int
	off := 0
	for i := int64(0); i < int64(m.Meta.Word2C); i++ {
		if len(b)-off < head {
			return nil, errLootWalk
		}
		n := int64(binary.LittleEndian.Uint32(b[off:]))
		if n*lootElementSize > int64(len(b)-off-head) {
			return nil, errLootWalk
		}
		heads = append(heads, off)
		off += head + int(n)*lootElementSize
	}
	return heads, nil
}

type saveFormError string

func (e saveFormError) Error() string { return string(e) }

const errLootWalk saveFormError = "alm: type8 records do not fit their payload"
