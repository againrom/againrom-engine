package game

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"slices"

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
			if row.Entity == id && (row.PendingOrder.IsCastRow()) {
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

// finalizeCurrentOrderProjection records, per actor order, the bytes the
// emitted U158 holds where the World's order differs: the writer projects a
// victim's minted key or a reach from typed current state. The continuation
// restores the inner order, progress and retry bytes and the absent spell
// key itself, so those are not listed.
func finalizeCurrentOrderProjection(doc *sav.DocumentData, a *currentActionData, world *sim.World) error {
	_, orders, present := world.SavedGroups()
	if !present {
		return nil
	}
	byEntity := make(map[sim.EntityID]*sim.SavedActorOrder, len(orders))
	for i := range orders {
		byEntity[orders[i].Entity] = &orders[i]
	}
	restored := func(at int, spellAbsent bool) bool {
		return at == 8 || at == 9 || at == 0x15 || spellAbsent && at >= 0x30 && at < 0x34
	}
	for i := range a.Actions.Actors {
		row := &a.Actions.Actors[i]
		if row.Order == nil {
			continue
		}
		row.Order.Projected = nil
		o := byEntity[row.Entity]
		if o == nil {
			continue
		}
		var object uint16
		for _, b := range a.Bindings {
			if b.ID == row.Entity && !b.Structure && !b.Missing {
				object = b.Object
				break
			}
		}
		if object == 0 || int(object) > len(doc.Objects) {
			continue
		}
		wire, err := savedMotionRaw(&doc.Objects[object-1], "U158", 148)
		if err != nil {
			continue
		}
		spellAbsent := row.Order.AbsentSpellKey != nil
		for at := 0; at < len(o.Raw); {
			if wire[at] == o.Raw[at] || restored(at, spellAbsent) {
				at++
				continue
			}
			end := at
			for end < len(o.Raw) && wire[end] != o.Raw[end] && !restored(end, spellAbsent) {
				end++
			}
			row.Order.Projected = append(row.Order.Projected, sim.ActionOrderBytes{At: uint8(at), Wire: slices.Clone(wire[at:end]), World: slices.Clone(o.Raw[at:end])})
			at = end
		}
	}
	return nil
}
