package sim

// dropToGround is ITEM-DROP-008's own geometry, shared by both drop sources
// (1005 round 2): the caller's requested cell (destX, destY) is honoured
// only within a Chebyshev distance of 2 of the ENTITY'S OWN CURRENT CELL,
// read here at APPLICATION TIME rather than at the time the drag was
// released — the same "read fresh, not from the request" rule equip.go
// already applies to a container index. Outside that window, on a cell
// sackFault refuses, or on a cell Sack registration refuses
// (groundBlockedSackCell), the drop lands at the entity's own cell.
// That own cell is used even when it is itself ground-blocked (DIV-1413).
//
// THE TEST IS TWO INDEPENDENT abs(...) > 2 COMPARISONS (ITEM-DROP-008), so
// the window is a 5x5 square centred on the entity and not a circle: a
// request 2 cells east and 2 cells north of the entity is inside it, one 3
// cells east and 0 north is not.
//
// IT IS NEVER REFUSED FOR A REQUESTED DESTINATION (ITEM-DROP-008). A
// destination this world cannot encode falls back to the entity's own cell
// rather than dropping nothing.
//
// THE FALLBACK IS ITSELF RE-CHECKED: placeAt and the move step keep a live
// entity inside the map, but a Sack is never planted where decodeSacks would
// refuse it. On that branch the item, already taken off the entity, is
// discarded: a corrupt entity position, not a requested destination.
//
// GOLD IS ALWAYS ZERO. ITEM-DROP-008 puts gold on a different opcode
// (destination 3 pairs only with a container element or an equipment slot
// here); the purse drops through KindPlayerDropGold (dropgold.go).
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
	dx, dy := destX-x, destY-y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx > 2 || dy > 2 || w.groundBlockedSackCell(destX, destY) || sackFault(w.bounds, Sack{X: destX, Y: destY}) != nil {
		destX, destY = x, y
	}
	return destX, destY, sackFault(w.bounds, Sack{X: destX, Y: destY}) == nil
}

// dropRing is the order DropLanding searches a dropper's eight neighbours:
// south first, because a sack on a later row is drawn in front of the dropper,
// then the sides, then the rows behind it.
var dropRing = [8][2]int32{{0, 1}, {1, 0}, {-1, 0}, {1, 1}, {-1, 1}, {0, -1}, {1, -1}, {-1, -1}}

// DropLanding is the cell a drop aimed at (x, y) should be commanded to, so
// that the thrown item lies in a sack the player can see. dropToGround would
// plant the sack on the cell dropWindowCell names; a new sack on a cell a
// living unit occupies is drawn beneath that unit and cannot be seen. In that
// case the landing is the first neighbour of the dropper, in dropRing order,
// that holds no living unit, is not ground-blocked and holds no sack; failing
// that, the first that holds a sack; with none, the cell dropWindowCell names,
// so the item is never lost. A cell that already holds a sack is kept: a drop
// pours into it, one sack per cell. The answer is false for an entity the world
// does not hold or one whose own cell is off the map, with the request echoed.
func (w *World) DropLanding(id EntityID, x, y int32) (CellPoint, bool) {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return CellPoint{X: x, Y: y}, false
	}
	cx, cy, ok := w.dropWindowCell(i, x, y)
	if !ok {
		return CellPoint{X: x, Y: y}, false
	}
	if w.savedGroundIndex(cx, cy) >= 0 || !w.livingUnitAt(cx, cy) {
		return CellPoint{X: cx, Y: cy}, true
	}
	ox, oy := w.entities[i].X, w.entities[i].Y
	for _, empty := range [2]bool{true, false} {
		for _, d := range dropRing {
			nx, ny := ox+d[0], oy+d[1]
			if sackFault(w.bounds, Sack{X: nx, Y: ny}) != nil || w.groundBlockedSackCell(nx, ny) || w.livingUnitAt(nx, ny) {
				continue
			}
			if empty && w.savedGroundIndex(nx, ny) >= 0 {
				continue
			}
			return CellPoint{X: nx, Y: ny}, true
		}
	}
	return CellPoint{X: cx, Y: cy}, true
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

// dropFromContainer is KindDropCarried's own act: entity index i's container
// ELEMENT at cindex gives up ONE UNIT to the ground (source code 2 of
// ITEM-CMD-007), on equip's own element-not-unit rule — cindex names a
// place in w.carried[i], not a position in its flat expansion.
//
// ONE THING REFUSES IT, leaving i's container byte-for-byte what it was:
// cindex outside the container's own element bounds. AN ENTITY THE WORLD
// DOES NOT HOLD is refused already, by stepWorld's own entity lookup before
// this is ever reached (equip's own precedent).
//
// AT A COUNT ABOVE 1 the element stays in its own place at count minus 1
// (equip's own arithmetic for a source that loses a unit); AT A COUNT OF 1
// the element is removed entirely, on 0112's own rule for an emptied place.
// NO FOLD IS NEEDED: removing a unit can never create a second element
// holding a code some other element already holds, unlike equip's own
// displaced-code append.
func (w *World) dropFromContainer(i, cindex int, destX, destY int32) {
	if w.savedObjects != nil && w.actorHasSavedItems(i) {
		n := w.sourceMutationCopy(i)
		if n.dropSavedCarried(i, cindex, destX, destY) {
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
	w.finishLoadMutation(i, before)
	w.dropToGround(i, item, destX, destY)
	// Finish the item move before normalizing a resulting health loss, so the
	// death transition sees the final equipment and container state.
	w.clearFelled(i)
}
