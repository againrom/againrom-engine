package game

import (
	"againrom/pkg/sim"
	"fmt"
)

func (mw *mapWorld) useScroll(owner uint32, index int, key string, target uint32, x, y int, atCell bool) {
	if index < 0 || index > 65535 {
		return
	}
	id := sim.EntityID(owner)
	e, ok := mw.entity(id)
	if !ok || e.Owner != sim.SelfSlot {
		return
	}
	items, ok := mw.world.CarriedItems(id)
	if !ok || index >= len(items) || fmt.Sprintf("%#v", items[index]) != key {
		return // A display arm cannot follow an index onto a replaced item.
	}
	// The two orders carry the same carried index and differ in what the other
	// two numbers are: a unit for one, a cell for the other. That is the fork a
	// single literal hid behind one X.
	cmd := sim.UseScroll(id, sim.ItemSlot(index), sim.EntityID(target))
	if atCell {
		cmd = sim.UseScrollAt(id, sim.ItemSlot(index), sim.CellPoint{X: int32(x), Y: int32(y)})
	}
	mw.cancelPickup(id)
	mw.pending = append(mw.pending, cmd)
	mw.commanded[id] = true
}
