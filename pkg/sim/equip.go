package sim

const EquipSlots = 12

func (w *World) equip(i, cindex, slot, displace int) {
	if w.entities[i].ActorLoad.Source.Class != 0 {
		n := w.sourceMutationCopy(i)
		if n.sourceEquipCommand(i, cindex) && n.savedMutationValid() {
			*w = n
			w.clearFelled(i)
		}
		return
	}
	if w.actorHasSavedItems(i) {
		n := w.sourceMutationCopy(i)
		if n.equipSavedNative(i, cindex, slot, displace) && n.savedMutationValid() {
			*w = n
			w.clearFelled(i)
		}
		return
	}
	if !w.sourceMutationReady(i) {
		return
	}
	if cindex < 0 || cindex >= len(w.carried[i]) {
		return
	}
	if w.carried[i][cindex].Code == 0 || w.carried[i][cindex].Count == 0 {
		return
	}
	if slot < 1 || slot > EquipSlots {
		return
	}
	if displace < 0 || displace > EquipSlots || displace == slot {
		return
	}
	before := w.beginLoadMutation(i)
	if w.entities[i].ActorLoad.Present {
		w.entities[i].ActorLoad.InsertIndex = uint32(cindex)
	}
	item := w.carried[i][cindex].Instance()
	held := w.equipment[i][slot-1]
	var other ItemInstance
	if displace != 0 {
		other = w.equipment[i][displace-1]
	}
	w.equipment[i][slot-1] = item.Clone()
	applyEquipmentItemState(&w.entities[i], held, item, w.spells, w.damageObservation)
	if slot == slotWeapon+1 {
		syncWeaponItem(&w.entities[i], item)
	}
	switch {
	case w.carried[i][cindex].Count > 1:
		w.carried[i][cindex].Count--
		if !held.Empty() {
			w.addCarried(i, StackItem(held, 1))
		}
	case !held.Empty():
		w.carried[i][cindex] = StackItem(held, 1)
	default:
		w.carried[i] = append(w.carried[i][:cindex], w.carried[i][cindex+1:]...)
	}
	if !other.Empty() {
		w.equipment[i][displace-1] = ItemInstance{}
		applyEquipmentItemState(&w.entities[i], other, ItemInstance{}, w.spells, w.damageObservation)
		if displace == slotWeapon+1 {
			syncWeaponItem(&w.entities[i], ItemInstance{})
		}
		w.addCarried(i, StackItem(other, 1))
	}
	if !w.entities[i].ActorLoad.Present && !hasNullOrderedSlot(w.carried[i]) {
		w.carried[i] = foldContainer(w.carried[i])
	}
	w.finishLoadMutation(i, before)
	// An item replacement can remove more current health than its replacement
	// supplies. Normalize death only after the complete container/equipment move
	// so clearFelled drops the final loadout once and no later writer restores it.
	w.clearFelled(i)
}

// unequip is the unequip command's own act (0151, defect 4): equipment slot
// slot's code moves back into i's container as one unit, and the slot is
// left at the zero code — equip's own spelling of empty.
//
// TWO THINGS REFUSE IT, each leaving i's equipment and container
// byte-for-byte what they were: slot outside 1 to 12, and a slot already at
// the zero code — there is nothing to take off.
//
// THE ITEM IS FOLDED IN AT THE CONTAINER'S TAIL, equip's own displaced-code
// append (equip, above) reused rather than a third way of putting a code
// into a container: it merges into an element already holding that code, or
// takes a fresh place at the tail when none does (foldContainer). The
// container carries no capacity limit in this build — no slot count and no
// refusal on weight anywhere (foldContainer's own doc, whose weight has ONE
// consumer and it is the speed penalty) — so unlike equip's own
// cindex-and-slot refusals, there is no third refusal here for "the
// container cannot take it": once the slot holds a real code this move
// always succeeds. Taking off slot 1 leaves a worn slot-2 shield where it is:
// a shield is worn on its own.
//
// NOTHING ELSE ABOUT THE ENTITY MOVES — not health, position, order block,
// combat block or tick, equip's own rule restated for its inverse. The combat
// block recompute is a pkg/game call site outside this package (equip's own
// doc), unchanged by this function.
func (w *World) unequip(i, slot int) {
	if w.entities[i].ActorLoad.Source.Class != 0 {
		n := w.sourceMutationCopy(i)
		if _, ok := n.sourceUnequipCommand(i, slot, true); ok && n.savedMutationValid() {
			*w = n
			w.clearFelled(i)
		}
		return
	}
	if w.actorHasSavedItems(i) {
		n := w.sourceMutationCopy(i)
		if n.unequipSavedNative(i, slot) && n.savedMutationValid() {
			*w = n
			w.clearFelled(i)
		}
		return
	}
	if !w.sourceMutationReady(i) {
		return
	}
	if slot < 1 || slot > EquipSlots || !w.hasActorContainer(i) {
		return
	}
	item := w.equipment[i][slot-1]
	if item.Empty() {
		return
	}
	before := w.beginLoadMutation(i)
	w.equipment[i][slot-1] = ItemInstance{}
	applyEquipmentItemState(&w.entities[i], item, ItemInstance{}, w.spells, w.damageObservation)
	if slot == slotWeapon+1 {
		syncWeaponItem(&w.entities[i], ItemInstance{})
	}
	w.addCarried(i, StackItem(item, 1))
	w.finishLoadMutation(i, before)
	w.clearFelled(i)
}

// Equipped returns id's twelve equipment slots, in slot order, as a copy —
// Carried's own reason: mutating the result reaches nothing. UNLIKE Carried
// it costs no fresh append to get that guarantee, because [EquipSlots]uint16
// is an ARRAY: Go copies one by assignment, so returning w.equipment[i]
// directly is already the copy the guarantee asks for. The second result is
// false when id names no entity this world holds, and the first is then the
// zero array — every slot empty, which is indistinguishable from an entity
// this world does hold that has equipped nothing, on the same ground a
// felled entity's zero HP is not distinguishable from one built at zero: a
// caller that needs to tell the two apart asks Entities or Carried first.
func (w *World) Equipped(id EntityID) ([EquipSlots]uint16, bool) {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return [EquipSlots]uint16{}, false
	}
	return equipmentCodes(w.equipment[i]), true
}

// EquippedItems returns the twelve complete instances as a deep copy.
func (w *World) EquippedItems(id EntityID) ([EquipSlots]ItemInstance, bool) {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return [EquipSlots]ItemInstance{}, false
	}
	return cloneEquipment(w.equipment[i]), true
}
