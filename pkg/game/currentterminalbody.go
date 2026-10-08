package game

import (
	"fmt"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// currentTerminalBody is the record source of a departed authored actor.
type currentTerminalBody struct {
	entity sim.Entity
	worn   [sim.EquipSlots]sim.ItemInstance
}

// currentTerminalBodies builds the body of each departed authored actor. The
// World holds its terminal tuple and native basis; its placement, through the
// map and the installed tables, constructs the rest as a fresh mission does.
// A terminal row with no placement has no record.
func currentTerminalBodies(w *sim.World, m *alm.Map, t *mapload.Table, diff mapload.Difficulty) (map[sim.EntityID]currentTerminalBody, error) {
	out := map[sim.EntityID]currentTerminalBody{}
	rows := w.CurrentTerminalActors()
	placed := false
	for _, row := range rows {
		placed = placed || row.MapUnitID != 0
	}
	if !placed {
		return out, nil
	}
	spawn, err := mapload.FromALMWith(m, t, diff)
	if err != nil {
		return nil, fmt.Errorf("departed actor placements: %w", err)
	}
	bases := map[sim.EntityID]sim.NativeActorBasis{}
	for _, row := range w.RemovedNativeActorBases() {
		bases[row.ID] = row.Basis
	}
	for _, row := range rows {
		if row.MapUnitID == 0 {
			continue
		}
		var body *currentTerminalBody
		for _, e := range spawn.Entities() {
			if e.MapUnitID != row.MapUnitID {
				continue
			}
			if body != nil {
				return nil, fmt.Errorf("departed actor %d has an ambiguous placement", row.ID)
			}
			worn, _ := spawn.EquippedItems(e.ID)
			body = &currentTerminalBody{entity: e, worn: worn}
		}
		if body == nil {
			continue
		}
		e := &body.entity
		e.ID, e.X, e.Y, e.HP, e.Decay = row.ID, int32(row.Cell&255), int32(row.Cell>>8), row.HP, sim.DecayStage(row.Stage)
		e.NativeBasis = bases[row.ID]
		for slot := range body.worn {
			if row.Worn&(1<<slot) == 0 {
				body.worn[slot] = sim.ItemInstance{}
			}
		}
		out[row.ID] = *body
	}
	return out, nil
}

// currentTerminalWorn writes the items a departed actor's body wears and
// references them from its record.
func (b *generatedDocumentBuilder) currentTerminalWorn(index uint16, worn [sim.EquipSlots]sim.ItemInstance, w *sim.World) error {
	weights := map[uint16]sim.ItemWeight{}
	for _, weight := range w.ItemWeights() {
		weights[weight.Code] = weight
	}
	owner, err := savedStructureValue(&b.doc.Objects[index-1], "Reference")
	if err != nil {
		return err
	}
	var refs [sim.EquipSlots]uint16
	for slot, item := range worn {
		if item.Empty() {
			continue
		}
		if refs[slot], err = b.item(currentItemRecordValue(item, true, weights, b.table), 1, owner); err != nil {
			return err
		}
	}
	r := &b.doc.Objects[index-1]
	mustSetRefs(r, "HeldWeapon", []uint16{refs[0]})
	mustSetRefs(r, "HeldShield", []uint16{refs[1]})
	if r.Class != "Unit" {
		refs[0], refs[1] = 0, 0
		mustSetRefs(r, "Worn", refs[:])
	}
	return nil
}
