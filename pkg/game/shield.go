package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// normalizeMissionShieldLoadouts repairs a restored live world before any
// mission projection reads it. Invalid shields move to their owner's pack and
// the party record and derived combat block follow the same complete instances.
func normalizeMissionShieldLoadouts(ms *Mission, t *mapload.Table) {
	if ms == nil || ms.World == nil {
		return
	}
	n := len(ms.Party)
	if len(ms.Start.IDs) < n {
		n = len(ms.Start.IDs)
	}
	for i := 0; i < n; i++ {
		id := ms.Start.IDs[i]
		equipped, ok := ms.World.EquippedItems(id)
		if !ok {
			continue
		}
		carried, ok := ms.World.CarriedItems(id)
		if !ok || !mapload.NormalizeShieldLoadout(&equipped, &carried, t) {
			continue
		}
		if !ms.World.ReplaceStock(sim.Stock{ID: id, ItemInstances: carried, EquippedItems: equipped}) {
			continue
		}
		syncMissionPartyStock(&ms.Party[i], equipped, carried)
		member := &ms.Party[i]
		loadout := mapload.PartyLoadout(*member, t)
		applyRearmLoadout(ms.World, id, member.Hero, member.Profile, loadout, equipped, t)
	}
}

func syncMissionPartyStock(member *mapload.PartyMember, equipped [sim.EquipSlots]sim.ItemInstance, carried []sim.ItemInstance) {
	if member == nil {
		return
	}
	var equippedCodes [sim.EquipSlots]uint16
	for i := range equipped {
		equipped[i] = equipped[i].Clone()
		equippedCodes[i] = equipped[i].Code
	}
	carriedItems := make([]sim.ItemInstance, len(carried))
	carriedCodes := make([]uint16, len(carried))
	for i := range carried {
		carriedItems[i] = carried[i].Clone()
		carriedCodes[i] = carried[i].Code
	}
	if member.Carry != nil {
		member.Carry.EquippedItems = equipped
		member.Carry.Equipped = equippedCodes
		member.Carry.ItemInstances = carriedItems
		member.Carry.Items = carriedCodes
		return
	}
	member.WornItems = equipped
	member.Worn = equippedCodes
	member.CarriedItems = carriedItems
	member.Carried = carriedCodes
}
