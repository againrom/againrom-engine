package sim

import "fmt"

// WornHolder is the holder of a worn-equipment change: a World actor or a town
// city object. It answers the profile row and stores an item's Spell where it
// keeps it; when a change touches the Spell is the transaction's rule. The
// slot is the one-based equipment slot the item leaves or enters. The slot
// and container writes stay with the caller.
type WornHolder interface {
	// KeepsWeaponSpell is the second game's row: a Weapon keeps its Spell
	// when taken off, and an equip builds none.
	KeepsWeaponSpell() bool
	// DropWeaponSpell stores item as holding no Spell.
	DropWeaponSpell(slot int, item *ItemInstance) bool
	// BuildWeaponSpell gives item the Spell an equip builds from its
	// castSpell Effect; false leaves item as it was.
	BuildWeaponSpell(slot int, item *ItemInstance) bool
}

// TakeOffWorn is item leaving equipment slot slot of h. In the first game
// removal deletes a classed Weapon's Spell (ITEM-187, ITEM-DEATH-012), and the
// next equip builds a new one. It returns the item to place and false when
// the slot is no equipment slot or h refuses the Spell write; item then keeps
// whatever h already wrote.
func TakeOffWorn(h WornHolder, slot int, item ItemInstance) (ItemInstance, bool) {
	if slot < 1 || slot > EquipSlots {
		return item, false
	}
	if h.KeepsWeaponSpell() || item.SourceEquipment.Class != SourceWeapon || !item.SourceEquipment.Spell.Present {
		return item, true
	}
	ok := h.DropWeaponSpell(slot, &item)
	return item, ok
}

// WearWorn is item entering equipment slot slot of h. In the first game an
// equip builds a Weapon's Spell when it holds none, for any owner (ITEM-187);
// a script hand-over, a join and a SAVE build nothing (ITEM-212). It returns
// the item to place and whether a Spell was built.
func WearWorn(h WornHolder, slot int, item ItemInstance) (ItemInstance, bool) {
	if slot < 1 || slot > EquipSlots || h.KeepsWeaponSpell() || item.Empty() || item.SourceEquipment.Spell.Present {
		return item, false
	}
	built := h.BuildWeaponSpell(slot, &item)
	return item, built
}

// WornItemEqual reports whether a and b are one held item across worn
// changes: a Weapon's owned Spell is left out, because TakeOffWorn deletes it
// and WearWorn builds it again, so the Weapon taken off is the one worn
// before.
func WornItemEqual(a, b ItemInstance) bool {
	a, b = a.Clone(), b.Clone()
	a.SourceEquipment.Spell, b.SourceEquipment.Spell = SourceItemSpell{}, SourceItemSpell{}
	return ItemEqual(a, b)
}

// worldWornHolder is actor i of a World as a WornHolder. Its Spell store is
// the saved-object registry when the item has an identity.
type worldWornHolder struct {
	w *World
	i int
}

func (h worldWornHolder) KeepsWeaponSpell() bool { return h.w.SecondGame() }

func (h worldWornHolder) DropWeaponSpell(_ int, item *ItemInstance) bool {
	return h.w.replaceObjectSpell(item, SourceItemSpell{})
}

// BuildWeaponSpell builds the Spell from the Spells rule of the castSpell
// Effect; an item without a class takes its constructor's. Only the item and
// its registry row change: no actor load or pool is derived again.
func (h worldWornHolder) BuildWeaponSpell(_ int, item *ItemInstance) bool {
	w := h.w
	id, _, ok := item.CastSpell()
	rule, known := w.findSpell(uint32(uint8(id)))
	if !ok || !known || uint8(id) == 0 {
		return false
	}
	source := item.SourceEquipment
	if source.Class == 0 {
		source = w.itemConstructor(item.Code)
	}
	if source.Class != SourceWeapon {
		return false
	}
	source.Spell = sourceSpellFromRule(rule)
	if item.ObjectID == 0 {
		item.SourceEquipment = source
		return true
	}
	objects := w.savedObjects.Clone()
	err := objects.mutate(func(r *SavedObjects) error {
		row := r.item(item.ObjectID)
		if row == nil || row.Spell != 0 || !StackStateEqual(row.Value, r.stackForItem(*item)) {
			return fmt.Errorf("sim: stale worn Weapon")
		}
		spell := source.Spell
		source.Spell = SourceItemSpell{}
		row.Value.SourceEquipment, row.Token.T0C = source, source.DefinitionRow
		return r.putItemSpell(row, spell)
	})
	if err != nil {
		return false
	}
	w.savedObjects = objects
	*item = objects.item(item.ObjectID).Value.Instance()
	return true
}

// takeOffWorn is TakeOffWorn for actor i.
func (w *World) takeOffWorn(i, slot int, item *ItemInstance) bool {
	var ok bool
	*item, ok = TakeOffWorn(worldWornHolder{w, i}, slot, *item)
	return ok
}

// wearWorn is WearWorn for the item actor i wears in slot, after the route
// that made it worn. A built Spell that leaves the registry invalid is undone.
func (w *World) wearWorn(i, slot int) {
	if i < 0 || i >= len(w.entities) || slot < 1 || slot > EquipSlots {
		return
	}
	before, worn := w.savedObjects, w.equipment[i][slot-1]
	item, built := WearWorn(worldWornHolder{w, i}, slot, worn.Clone())
	if !built {
		return
	}
	w.equipment[i][slot-1] = item
	if !w.savedMutationValid() {
		w.savedObjects, w.equipment[i][slot-1] = before, worn
	}
}
