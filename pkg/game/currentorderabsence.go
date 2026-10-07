package game

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// A book can have native values without a Spell identity. Anchor that missing
// Order pointer to the emitted book slot and key, never to copied order bytes.
func currentOrderSpellKey(doc *sav.DocumentData, a *currentActionData, id sim.EntityID) (uint32, uint32, error) {
	var object uint16
	for _, row := range a.Bindings {
		if row.ID == id && !row.Structure && !row.Missing {
			object = row.Object
			break
		}
	}
	if object == 0 || int(object) > len(doc.Objects) {
		return 0, 0, fmt.Errorf("current order key absence has no ordinary actor")
	}
	record := &doc.Objects[object-1]
	raw, err := savedMotionRaw(record, "U158", 148)
	if err != nil {
		return 0, 0, err
	}
	spell := uint16(0)
	for _, cast := range a.Actions.Books {
		if cast.Caster == id {
			spell = uint16(cast.Spell)
			break
		}
	}
	if spell == 0 {
		for _, row := range a.Actions.Actors {
			if row.Entity == id && (row.PendingOrder.Kind == sim.PendingActorCast || row.PendingOrder.Kind == sim.PendingCellCast) {
				spell = row.PendingOrder.Spell
				break
			}
		}
	}
	if spell != 0 {
		refs, _ := savedObjectRefs(record, "Spells")
		if int(spell) > len(refs) || refs[spell-1] == 0 {
			return binary.LittleEndian.Uint32(raw[0x30:]), 0, nil
		}
		ref := refs[spell-1]
		if int(ref) > len(doc.Objects) || doc.Objects[ref-1].Class != "Spell" {
			return 0, 0, fmt.Errorf("current order key absence has a malformed Spell slot")
		}
		key, err := savedStructureValue(&doc.Objects[ref-1], "This")
		return binary.LittleEndian.Uint32(raw[0x30:]), key, err
	}
	return 0, 0, fmt.Errorf("current order key absence has no active book")
}

func finalizeCurrentOrderSpellAbsence(doc *sav.DocumentData) error {
	a, err := readCurrentActions(doc)
	if err != nil || a == nil {
		return err
	}
	for i := range a.Actions.Actors {
		row := &a.Actions.Actors[i]
		if row.Order == nil || row.Order.AbsentSpellKey == nil {
			continue
		}
		wire, key, err := currentOrderSpellKey(doc, a, row.Entity)
		if err != nil {
			return err
		}
		row.Order.AbsentSpellKey = nil
		if key != 0 && wire == key {
			row.Order.AbsentSpellKey = &wire
		}
	}
	raw, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return sav.SetNativeActions(&doc.State, raw)
}

func matchCurrentOrderSpellAbsence(doc *sav.DocumentData, a *currentActionData) error {
	for i := range a.Actions.Actors {
		row := &a.Actions.Actors[i]
		if row.Order == nil || row.Order.AbsentSpellKey == nil {
			continue
		}
		anchor := *row.Order.AbsentSpellKey
		if anchor == 0 {
			return fmt.Errorf("current order key absence has no emitted anchor")
		}
		wire, key, err := currentOrderSpellKey(doc, a, row.Entity)
		if err != nil {
			return err
		}
		if wire != anchor || key != anchor {
			row.Order.AbsentSpellKey = nil
		}
	}
	return nil
}
