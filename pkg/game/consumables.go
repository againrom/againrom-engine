package game

import (
	"againrom/pkg/sim"
	"fmt"
)

// itemCastKey is the item's description: the identity an item cast's arm
// carries to its release and a shortcut binding keeps. A world allocates object
// identities and tokens afresh, so neither is part of it. A native record
// formats by value: its pointer differs per copy.
func itemCastKey(item sim.ItemInstance) string {
	native := item.NativeRecord
	item.NativeRecord, item.ObjectID = nil, 0
	if native == nil {
		return fmt.Sprintf("%#v", item)
	}
	record := *native
	record.Token = sim.SavedObjectToken{}
	return fmt.Sprintf("%#v%+v", item, record)
}

func (mw *mapWorld) useScroll(owner uint32, index int, key string, target uint32, x, y int, atCell bool) {
	if index < 0 || index > 65535 {
		return
	}
	id := sim.EntityID(owner)
	e, ok := mw.entity(id)
	if !ok || e.Owner != sim.SelfSlot {
		return
	}
	stacks, ok := mw.world.CarriedStacks(id)
	if !ok || index >= len(stacks) || itemCastKey(stacks[index].Instance()) != key {
		return // A display arm cannot follow an index onto a replaced item.
	}
	// The two orders carry the same carried index and differ in what the other
	// two numbers are: a unit for one, a cell for the other. That is the fork a
	// single literal hid behind one X.
	cmd := sim.UseScroll(id, sim.ItemSlot(index), sim.EntityID(target))
	if atCell {
		cmd = sim.UseScrollAt(id, sim.ItemSlot(index), sim.CellPoint{X: int32(x), Y: int32(y)})
	}
	mw.pending = append(mw.pending, cmd)
	mw.commanded[id] = true
}

func inventoryItemKeys(stacks []sim.ItemStack) []string {
	keys := make([]string, len(stacks))
	for i, stack := range stacks {
		keys[i] = itemCastKey(stack.Instance())
	}
	return keys
}
