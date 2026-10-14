package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// savedWornWeaponSpells is the World a first-game SAVE writes: every worn
// Weapon with a castSpell Effect holds its Spell (DIV-3362). A plain worn item
// takes the operands the item projection writes for a held item. The live
// World keeps its state; an unchanged World returns its own bytes.
func savedWornWeaponSpells(raw []byte, table *mapload.Table) ([]byte, error) {
	var w sim.World
	if err := w.UnmarshalBinary(raw); err != nil {
		return nil, err
	}
	if w.BuildWornWeaponSpells(wornItemConstructor(&w, table)) == 0 {
		return raw, nil
	}
	return w.MarshalBinary()
}

// joinedWornSpells gives a first-game member who joins in the town each worn
// Weapon's Spell as a mission start builds it, so the town state is the one
// its SAVE writes (DIV-3362).
func joinedWornSpells(worn []sim.ItemInstance, table *mapload.Table) {
	if table == nil || tableFiles(table).storedWeaponSpell {
		return
	}
	for slot := range worn {
		if _, _, ok := worn[slot].CastSpell(); ok {
			worn[slot] = mapload.SourceEquippedItem(worn[slot], table)
		}
	}
}

// buildMissionWornSpells gives a first-game mission World, new or loaded,
// each party actor's worn classed Weapon its Spell before the mission runs,
// so the World SAVE writes is the one that plays (DIV-3362).
func buildMissionWornSpells(ms *Mission, table *mapload.Table) {
	if ms == nil || ms.World == nil || table == nil || tableFiles(table).storedWeaponSpell {
		return
	}
	ms.World.BuildWornWeaponSpells(wornItemConstructor(ms.World, table))
}

// wornItemConstructor gives a plain worn item the operands the item
// projection writes for a held item.
func wornItemConstructor(w *sim.World, table *mapload.Table) func(sim.ItemInstance) sim.ItemInstance {
	weights := make(map[uint16]sim.ItemWeight)
	for _, weight := range w.ItemWeights() {
		weights[weight.Code] = weight
	}
	return func(item sim.ItemInstance) sim.ItemInstance {
		return currentItemRecordValue(item, true, weights, table)
	}
}
