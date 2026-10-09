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

// currentTerminalBodies builds the body of each departed authored actor and
// departed party member. The World holds its terminal tuple and native basis;
// its placement, or its party member's roster entry, constructs the rest
// through the map and the installed tables as a fresh mission does. A
// terminal row with neither has no record.
func currentTerminalBodies(w *sim.World, m *alm.Map, t *mapload.Table, diff mapload.Difficulty, party map[sim.EntityID]mapload.PartyMember) (map[sim.EntityID]currentTerminalBody, error) {
	out := map[sim.EntityID]currentTerminalBody{}
	rows := w.CurrentTerminalActors()
	placed := false
	for _, row := range rows {
		placed = placed || row.MapUnitID != 0
	}
	var spawn *sim.World
	if placed {
		var err error
		if spawn, err = mapload.FromALMWith(m, t, diff); err != nil {
			return nil, fmt.Errorf("departed actor placements: %w", err)
		}
	}
	bases := map[sim.EntityID]sim.NativeActorBasis{}
	for _, row := range w.RemovedNativeActorBases() {
		bases[row.ID] = row.Basis
	}
	for _, row := range rows {
		var body *currentTerminalBody
		if row.MapUnitID != 0 {
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
		} else if member, ok := party[row.ID]; ok {
			body = departedMemberBody(m, t, diff, member)
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

// departedMemberBody constructs a departed party member as a mission start
// constructs him, or nothing when the start cannot.
func departedMemberBody(m *alm.Map, t *mapload.Table, diff mapload.Difficulty, member mapload.PartyMember) *currentTerminalBody {
	world, start, err := mapload.StartMission(m, t, diff, []mapload.PartyMember{member})
	if err != nil || len(start.IDs) != 1 {
		return nil
	}
	e, ok := world.Entity(start.IDs[0])
	if !ok {
		return nil
	}
	worn, _ := world.EquippedItems(e.ID)
	return &currentTerminalBody{entity: e, worn: worn}
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

// placementActors constructs the map's placed actors as a fresh mission does,
// keyed by map unit ID. A map unit ID placed twice maps to nothing.
func placementActors(m *alm.Map, t *mapload.Table, diff mapload.Difficulty) (map[uint16]sim.Entity, error) {
	spawn, err := mapload.FromALMWith(m, t, diff)
	if err != nil {
		return nil, fmt.Errorf("retained dead actor placements: %w", err)
	}
	out, seen := map[uint16]sim.Entity{}, map[uint16]int{}
	for _, e := range spawn.Entities() {
		if e.MapUnitID != 0 {
			seen[e.MapUnitID]++
			out[e.MapUnitID] = e
		}
	}
	for id, n := range seen {
		if n > 1 {
			delete(out, id)
		}
	}
	return out, nil
}
