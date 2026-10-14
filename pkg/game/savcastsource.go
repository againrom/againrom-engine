package game

import (
	"encoding/binary"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// sealCastSources is the last actor pass of a SAVE, after the document's keys
// are complete. LOAD maps U64 to its Spell record or to 0 (SAV-1262), and a
// cast at order progress 2 reads it at phase 0 and at the phase-5 zero, where
// a null Spell faults (MAGIC-376, MAGIC-377, SAV-1264). A weapon borrow in U68
// carries its WeaponSpell key in U64 (ITEM-CASTSTATE-056); a current book
// cast keeps its Spell's completed key in U64 and U44 (MAGIC-377, DIV-3330).
// An actor at progress 2 with neither U64 nor U68 is written in the words
// the original writes when no cast is live: progress 0, state 0, phase 0,
// countdown 0 (DIV-3356). At another progress the executor replaces the
// state before any read (SAV-1264), and the words stay. A current scroll cast keeps its words: what the original
// holds for one is Unknown (SAV-1264), and LOAD reads them.
func sealCastSources(doc *sav.DocumentData) {
	books, scrolls := currentCastObjects(doc)
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Human" && r.Class != "Unit" || scrolls[uint16(i+1)] {
			continue
		}
		state, errState := savedMotionRaw(r, "U54", 4)
		phase, errPhase := savedMotionRaw(r, "U58", 4)
		order, errOrder := savedMotionRaw(r, "U158", 148)
		spell, errSpell := savedStructureValue(r, "U64")
		if errState != nil || errPhase != nil || errOrder != nil || errSpell != nil {
			continue
		}
		borrow := uint16(0)
		if refs, _ := savedObjectRefs(r, "U68"); len(refs) == 1 && int(refs[0]) <= len(doc.Objects) {
			borrow = refs[0]
		}
		if borrow != 0 && doc.Objects[borrow-1].Class == "Weapon" && spell == 0 {
			key := uint32(0)
			if refs, _ := savedObjectRefs(&doc.Objects[borrow-1], "WeaponSpell"); len(refs) == 1 && refs[0] != 0 && int(refs[0]) <= len(doc.Objects) {
				key, _ = savedStructureValue(&doc.Objects[refs[0]-1], "This")
			}
			if key != 0 {
				savedObjectSetValue(r, "U64", key)
				spell = key
			} else {
				savedObjectSetRefs(r, "U68", []uint16{0}, false)
				borrow = 0
			}
		}
		code, sub := binary.LittleEndian.Uint32(state), binary.LittleEndian.Uint32(phase)
		if key := books[uint16(i+1)]; spell == 0 && borrow == 0 && key != 0 && (code == 0xd || code == 0xe || order[9] == 2) {
			savedObjectSetValue(r, "U64", key)
			savedObjectSetValue(r, "U44", key)
			spell = key
		}
		if spell != 0 || borrow != 0 || order[9] != 2 {
			continue
		}
		order[9] = 0
		if (code == 0xd || code == 0xe) && (sub == 0 || sub == 5) {
			copy(state, make([]byte, 4))
			copy(phase, make([]byte, 4))
			savedObjectSetValue(r, "U6C", 0)
		}
	}
}

// currentCastObjects maps each actor object with a current book cast to its
// Spell's key and names each actor object with a current scroll cast, read
// from the action continuation the same SAVE wrote.
func currentCastObjects(doc *sav.DocumentData) (map[uint16]uint32, map[uint16]bool) {
	keys, scrolls := map[uint16]uint32{}, map[uint16]bool{}
	a, err := readCurrentActions(doc)
	if err != nil || a == nil {
		return keys, scrolls
	}
	books, scrollCasters := map[sim.EntityID]bool{}, map[sim.EntityID]bool{}
	for _, c := range a.Actions.Books {
		books[c.Caster] = true
	}
	for _, c := range a.Actions.Scrolls {
		scrollCasters[c.Caster] = true
	}
	for _, row := range a.Bindings {
		if row.Structure || row.Missing {
			continue
		}
		scrolls[row.Object] = scrolls[row.Object] || scrollCasters[row.ID]
		if !books[row.ID] {
			continue
		}
		if _, key, err := currentOrderSpellKey(doc, a, row.ID); err == nil && key != 0 {
			keys[row.Object] = key
		}
	}
	return keys, scrolls
}

// savedWeaponCastBorrow reports a charge whose actor holds a weapon-borne
// cast: U68 borrows a Weapon carrying a WeaponSpell and U64 holds the Spell
// (ITEM-CASTSTATE-056). Such a charge ends in a release, not a blow.
func savedWeaponCastBorrow(doc *sav.DocumentData, r *sav.DocumentRecordData) bool {
	spell, _ := savedStructureValue(r, "U64")
	refs, _ := savedObjectRefs(r, "U68")
	if spell == 0 || len(refs) != 1 || refs[0] == 0 || int(refs[0]) > len(doc.Objects) {
		return false
	}
	weapon := &doc.Objects[refs[0]-1]
	carried, _ := savedObjectRefs(weapon, "WeaponSpell")
	return weapon.Class == "Weapon" && len(carried) == 1 && carried[0] != 0
}
