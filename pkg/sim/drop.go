package sim

import "encoding/binary"

// dropToGround shares ITEM-DROP-008's landing rule between both item sources.
// A refused aim falls back to the dropper's current cell, including when that
// cell is ground-blocked (DIV-1413). Gold uses its separate command.
func (w *World) dropToGround(i int, item ItemInstance, destX, destY int32) {
	x, y, ok := w.dropWindowCell(i, destX, destY)
	if !ok {
		return
	}
	w.pourSack(x, y, 0, []ItemInstance{item.Clone()})
}

// dropWindowCell is the cell entity i's drop lands on and whether a Sack may
// stand there: the request inside the window, else the entity's own cell.
func (w *World) dropWindowCell(i int, destX, destY int32) (int32, int32, bool) {
	x, y := w.entities[i].X, w.entities[i].Y
	dx, dy := int64(destX)-int64(x), int64(destY)-int64(y)
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx > 2 || dy > 2 || w.refusedDropCell(destX, destY) {
		destX, destY = x, y
	}
	return destX, destY, sackFault(w.bounds, Sack{X: destX, Y: destY}) == nil
}

// DropLanding previews the same landing rule the item command applies.
// An absent or off-map dropper echoes the request with a false answer.
func (w *World) DropLanding(id EntityID, x, y int32) (CellPoint, bool) {
	i := indexOfEntity(w.entities, id)
	if i < 0 || !w.dropperOnMap(i) {
		return CellPoint{X: x, Y: y}, false
	}
	cx, cy, ok := w.dropWindowCell(i, x, y)
	if !ok {
		return CellPoint{X: x, Y: y}, false
	}
	return CellPoint{X: cx, Y: cy}, true
}

func (w *World) refusedDropCell(x, y int32) bool {
	if sackFault(w.bounds, Sack{X: x, Y: y}) != nil || w.groundBlockedSackCell(x, y) {
		return true
	}
	if w.savedGroundIndex(x, y) >= 0 {
		return false
	}
	if w.livingUnitAt(x, y) {
		return true
	}
	key, ok := savedPlaneKey(x, y)
	if !ok {
		return false
	}
	if _, occupied := w.structureSlots[key]; occupied {
		return true
	}
	if cell := w.motionCell(key); cell != nil && binary.LittleEndian.Uint32(cell.Payload[16:]) != 0 {
		return true
	}
	for _, cell := range w.savedCellRecords {
		if cell.Cell == key && cell.Sack != 0 {
			return true
		}
	}
	return false
}

// dropperOnMap reports whether entity i's own cell can hold a Sack. A drop from
// an entity that fails it is refused before anything leaves the entity.
func (w *World) dropperOnMap(i int) bool {
	return sackFault(w.bounds, Sack{X: w.entities[i].X, Y: w.entities[i].Y}) == nil
}

// livingUnitAt reports whether a living unit on the map covers (x, y) with its
// footprint.
func (w *World) livingUnitAt(x, y int32) bool {
	for _, e := range w.entities {
		if e.OffMap || !e.Alive() {
			continue
		}
		n := footprintSide(e.TokenSize)
		if x >= e.X && x < e.X+n && y >= e.Y && y < e.Y+n {
			return true
		}
	}
	return false
}

// groundBlockedSackCell is SAV-SACKENTRY-590's refusal test, Dynamic bit 0.
// Without saved planes the grid's ground bit is the byte a SAVE writes there.
func (w *World) groundBlockedSackCell(x, y int32) bool {
	if w.savedCellPlanes != nil {
		key, ok := savedPlaneKey(x, y)
		return ok && w.savedCellPlanes.Dynamic[key]&1 != 0
	}
	i, ok := w.cellIndex(x, y)
	return ok && i < len(w.grid) && w.grid[i]&blockGround != 0
}

// dropFromContainer moves one unit or the whole selected element. The source
// remains unchanged when either its index or the landing cell is invalid.
func (w *World) dropFromContainer(i, cindex int, destX, destY int32, whole bool) {
	if w.savedObjects != nil && w.actorHasSavedItems(i) {
		n := w.sourceMutationCopy(i)
		if n.dropSavedCarried(i, cindex, destX, destY, whole) {
			*w = n
		}
		return
	}
	if !w.sourceMutationReady(i) || !w.dropperOnMap(i) {
		return
	}
	if cindex < 0 || cindex >= len(w.carried[i]) {
		return
	}
	if whole {
		n := w.sourceMutationCopy(i)
		x, y, valid := n.dropWindowCell(i, destX, destY)
		if !valid {
			return
		}
		before := n.beginLoadMutation(i)
		item, ok := n.takeCarriedObject(i, cindex, true)
		if !ok {
			return
		}
		if !n.finishLoadMutation(i, before) {
			return
		}
		items := make([]ItemInstance, item.Count)
		for j := range items {
			items[j] = item.Instance()
		}
		n.pourSack(x, y, 0, items)
		*w = n
		return
	}
	item := w.carried[i][cindex].Instance()
	if !w.consumeCarriedUnit(i, cindex) {
		return
	}
	w.dropToGround(i, item, destX, destY)
}

// dropFromEquipment is KindDropWorn's own act: equipment slot slot's code
// moves to the ground (source code 1 of ITEM-CMD-007), unequip's own move
// (equip.go) with pourSack in place of the container append. Dropping slot 1
// leaves a worn slot-2 shield where it is.
//
// TWO THINGS REFUSE IT, each leaving i's equipment byte-for-byte what it
// was: slot outside 1 to 12, and a slot already at the zero code — there is
// nothing to drop, unequip's own two refusals restated for this move.
func (w *World) dropFromEquipment(i, slot int, destX, destY int32) {
	if w.savedObjects != nil && w.actorHasSavedItems(i) {
		n := w.sourceMutationCopy(i)
		x, y, valid := n.savedDropCell(i, destX, destY)
		if !valid || slot < 1 || slot > EquipSlots || n.equipment[i][slot-1].Empty() {
			return
		}
		before := n.beginLoadMutation(i)
		var item ItemInstance
		var ok bool
		source := n.entities[i].ActorLoad.Source.Class != 0
		if source {
			item, ok = n.sourceUnequipCommand(i, slot, false)
		} else {
			item = n.equipment[i][slot-1].Clone()
			ok = n.takeWornObject(i, slot, item)
			if ok {
				n.equipment[i][slot-1] = ItemInstance{}
				n.applyEquipmentItemState(&n.entities[i], item, ItemInstance{}, n.spells, n.damageObservation)
				if slot == slotWeapon+1 {
					syncWeaponItem(&n.entities[i], ItemInstance{})
				}
				ok = n.takeOffWeaponSpell(&item)
			}
		}
		if !ok || !n.putGroundObjectAt(i, x, y, 0, StackItem(item, 1)) || !source && !n.finishLoadMutation(i, before) || !n.savedMutationValid() {
			return
		}
		*w = n
		w.clearFelled(i)
		return
	}
	if !w.dropperOnMap(i) {
		return
	}
	if w.entities[i].ActorLoad.Source.Class != 0 {
		n := w.sourceMutationCopy(i)
		if item, ok := n.sourceUnequipCommand(i, slot, false); ok {
			*w = n
			w.dropToGround(i, item, destX, destY)
			w.clearFelled(i)
		}
		return
	}
	if !w.sourceMutationReady(i) {
		return
	}
	if slot < 1 || slot > EquipSlots {
		return
	}
	item := w.equipment[i][slot-1].Clone()
	if item.Empty() {
		return
	}
	before := w.beginLoadMutation(i)
	w.equipment[i][slot-1] = ItemInstance{}
	w.applyEquipmentItemState(&w.entities[i], item, ItemInstance{}, w.spells, w.damageObservation)
	if slot == slotWeapon+1 {
		syncWeaponItem(&w.entities[i], ItemInstance{})
	}
	w.takeOffWeaponSpell(&item)
	w.finishLoadMutation(i, before)
	w.dropToGround(i, item, destX, destY)
	// Finish the item move before normalizing a resulting health loss, so the
	// death transition sees the final equipment and container state.
	w.clearFelled(i)
}
