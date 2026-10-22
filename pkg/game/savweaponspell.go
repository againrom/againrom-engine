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

// townWornHolder is a town member's worn slot as a sim.WornHolder. Its Spell
// store is the item value; g and id name the item's city node when the town
// keeps its city topology, whose Spell edge then goes with the value. A
// table that keeps a stored Spell, or none, is the second game's row.
type townWornHolder struct {
	table *mapload.Table
	g     *cityObjectTopology
	id    sim.SavedObjectID
}

func (h townWornHolder) KeepsWeaponSpell() bool {
	return h.table == nil || tableFiles(h.table).storedWeaponSpell
}

func (h townWornHolder) DropWeaponSpell(_ int, item *sim.ItemInstance) bool {
	*item = item.Clone()
	item.SourceEquipment.Spell = sim.SourceItemSpell{}
	h.dropNodeSpell()
	return true
}

// dropNodeSpell clears the Spell edge of the item's city node.
func (h townWornHolder) dropNodeSpell() {
	if h.g == nil {
		return
	}
	for k := range h.g.Items {
		if h.g.Items[k].ID == h.id {
			h.g.Items[k].Spell = 0
		}
	}
}

// BuildWeaponSpell builds the Spell from the first castSpell Effect; a spell
// id without an installed Spells row keeps the gap.
func (h townWornHolder) BuildWeaponSpell(_ int, item *sim.ItemInstance) bool {
	built := *item
	if built.SourceEquipment.Class == 0 {
		built = mapload.SourceConstructedItem(built, h.table)
	}
	if built.SourceEquipment.Class != sim.SourceWeapon {
		return false
	}
	spell, err := nativeCityWeaponSpell(built, true, h.table)
	if err != nil || !spell.Present {
		return false
	}
	built.SourceEquipment.Spell = spell
	*item = built
	return true
}

// takeOffTown is sim.TakeOffWorn for a town member's item leaving slot; in the
// first game a node edge whose value holds no Spell is cleared, for any class.
func takeOffTown(h townWornHolder, slot int, item sim.ItemInstance) sim.ItemInstance {
	item, _ = sim.TakeOffWorn(h, slot, item)
	if !h.KeepsWeaponSpell() && !item.SourceEquipment.Spell.Present {
		h.dropNodeSpell()
	}
	return item
}

// wearTown is sim.WearWorn for a town member's item entering slot.
func wearTown(table *mapload.Table, slot int, item sim.ItemInstance) sim.ItemInstance {
	item, _ = sim.WearWorn(townWornHolder{table: table}, slot, item)
	return item
}
