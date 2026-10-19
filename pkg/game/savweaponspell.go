package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

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

// takenOffWeapon is a Weapon a first-game town member takes off: removal
// deletes its Spell (ITEM-187), and the next equip builds a new one.
func takenOffWeapon(item sim.ItemInstance, table *mapload.Table) sim.ItemInstance {
	if table == nil || tableFiles(table).storedWeaponSpell || item.SourceEquipment.Class != sim.SourceWeapon || !item.SourceEquipment.Spell.Present {
		return item
	}
	item = item.Clone()
	item.SourceEquipment.Spell = sim.SourceItemSpell{}
	return item
}

// dropCityItemSpell drops a first-game city Item id's Spell edge once the
// Weapon taken off holds no Spell, so the town SAVE writes it without one.
func dropCityItemSpell(g *cityObjectTopology, id sim.SavedObjectID, value sim.ItemInstance, table *mapload.Table) {
	if g == nil || table == nil || tableFiles(table).storedWeaponSpell || value.SourceEquipment.Spell.Present {
		return
	}
	for k := range g.Items {
		if g.Items[k].ID == id {
			g.Items[k].Spell = 0
		}
	}
}
