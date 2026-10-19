package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// joinedWornSpells gives a first-game member constructed from his definition
// row, a tavern hire or a town companion who was not a guest, each worn
// Weapon's Spell as the Human constructor's template equip builds it
// (ITEM-203, ITEM-212). A member who already exists keeps his Weapons as held.
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

// equippedWeapon is a Weapon a first-game town member puts on: equip builds
// its Spell from the first castSpell Effect when it holds none (ITEM-187); a
// spell id without an installed Spells row keeps the gap.
func equippedWeapon(item sim.ItemInstance, table *mapload.Table) sim.ItemInstance {
	if table == nil || tableFiles(table).storedWeaponSpell || item.SourceEquipment.Class != sim.SourceWeapon || item.SourceEquipment.Spell.Present {
		return item
	}
	spell, err := nativeCityWeaponSpell(item, true, table)
	if err != nil || !spell.Present {
		return item
	}
	item.SourceEquipment.Spell = spell
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
