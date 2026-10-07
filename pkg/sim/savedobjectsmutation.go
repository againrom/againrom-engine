package sim

import (
	"fmt"
	"slices"
)

func (w *World) savedPackOwner(i int) SavedObjectOwner {
	return SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: w.entities[i].ID}
}

// syncSavedPack records this producer's ordered live slots and load bookkeeping.
// It never changes an Item value or owner to repair a disagreement.
func (w *World) syncSavedPack(i int) {
	if w.savedObjects == nil {
		return
	}
	owner := w.savedPackOwner(i)
	c := w.savedObjects.container(owner)
	if c == nil {
		return
	}
	a := w.entities[i].ActorLoad
	c.Present = !a.Present || a.ContainerPresent
	c.Items = nil
	for _, st := range w.carried[i] {
		c.Items = append(c.Items, st.ObjectID)
		if st.ObjectID == 0 {
			c.Coverage.Unknown |= SavedUnknownContainerLoad | SavedUnknownMergePolicy
		}
	}
	if a.Present {
		c.InsertIndex, c.Accumulator = a.InsertIndex, a.Accumulator
	} else {
		c.InsertIndex, c.Accumulator = uint32(len(c.Items)), w.containerWeight(i)
		c.Coverage.Unknown |= SavedUnknownContainerLoad | SavedUnknownMergePolicy
	}
}

func (w *World) savedMutationValid() bool {
	return w.savedObjects == nil || w.validateSavedObjects() == nil
}

func (w *World) actorHasSavedItems(i int) bool {
	for _, st := range w.carried[i] {
		if st.ObjectID != 0 {
			return true
		}
	}
	for _, item := range w.equipment[i] {
		if item.ObjectID != 0 {
			return true
		}
	}
	return false
}

// Removal releases only this actor's roots. Shared Item and child nodes remain
// current until their last incoming occurrence is released.
func (w *World) retireRemovedActorObjects(id EntityID) bool {
	if w.savedObjects == nil {
		return true
	}
	r := w.savedObjects.Clone()
	for _, row := range r.Items {
		for {
			var from SavedItemLocation
			found := false
			for _, at := range r.Locations(row.ID) {
				if at.Owner.Entity == id && (at.Owner.Kind == SavedOwnerActorPack || at.Owner.Kind == SavedOwnerActorWorn) {
					from, found = at, true
					break
				}
			}
			if !found {
				break
			}
			if _, err := r.TakeWholeAt(row.ID, from, row.Value); err != nil {
				return false
			}
			if err := r.Dispose(row.ID); err != nil {
				return false
			}
		}
	}
	r.Containers = slices.DeleteFunc(r.Containers, func(c SavedObjectContainer) bool { return c.Owner.Kind == SavedOwnerActorPack && c.Owner.Entity == id })
	r.BookRoots = slices.DeleteFunc(r.BookRoots, func(root SavedBookRoot) bool { return root.Entity == id })
	r.RefreshChildLiveness()
	if r.ValidateNoInFlight() != nil {
		return false
	}
	w.savedObjects = r
	return true
}

func (w *World) equipSavedNative(i, index, slot, displace int) bool {
	if !w.sourceMutationReady(i) || index < 0 || index >= len(w.carried[i]) || slot < 1 || slot > EquipSlots || displace < 0 || displace > EquipSlots || displace == slot {
		return false
	}
	before := w.beginLoadMutation(i)
	item, ok := w.takeCarriedObject(i, index, false)
	if !ok {
		return false
	}
	held := w.equipment[i][slot-1].Clone()
	if !held.Empty() && !w.takeWornObject(i, slot, held) {
		return false
	}
	w.equipment[i][slot-1] = ItemInstance{}
	if !w.putWornObject(i, slot, item.Instance()) {
		return false
	}
	w.equipment[i][slot-1] = item.Instance()
	applyEquipmentItemState(&w.entities[i], held, item.Instance(), w.spells, w.damageObservation)
	if slot == slotWeapon+1 {
		syncWeaponItem(&w.entities[i], item.Instance())
	}
	if !held.Empty() && !w.addCarried(i, StackItem(held, 1)) {
		return false
	}
	if displace != 0 {
		other := w.equipment[i][displace-1].Clone()
		if !other.Empty() {
			if !w.takeWornObject(i, displace, other) {
				return false
			}
			w.equipment[i][displace-1] = ItemInstance{}
			applyEquipmentItemState(&w.entities[i], other, ItemInstance{}, w.spells, w.damageObservation)
			if displace == slotWeapon+1 {
				syncWeaponItem(&w.entities[i], ItemInstance{})
			}
			if !w.addCarried(i, StackItem(other, 1)) {
				return false
			}
		}
	}
	return w.finishLoadMutation(i, before)
}

func (w *World) unequipSavedNative(i, slot int) bool {
	if slot < 1 || slot > EquipSlots || !w.hasActorContainer(i) || !w.sourceMutationReady(i) || w.equipment[i][slot-1].Empty() {
		return false
	}
	before := w.beginLoadMutation(i)
	item := w.equipment[i][slot-1].Clone()
	if !w.takeWornObject(i, slot, item) {
		return false
	}
	w.equipment[i][slot-1] = ItemInstance{}
	applyEquipmentItemState(&w.entities[i], item, ItemInstance{}, w.spells, w.damageObservation)
	if slot == slotWeapon+1 {
		syncWeaponItem(&w.entities[i], ItemInstance{})
	}
	if !w.addCarried(i, StackItem(item, 1)) {
		return false
	}
	return w.finishLoadMutation(i, before)
}

// takeCarriedObject distinguishes whole moves from one-unit split virtuals.
// Its caller owns the surrounding World candidate and applies actor load once.
func (w *World) takeCarriedObject(i, index int, whole bool, otherActive ...int) (ItemStack, bool) {
	if index < 0 || index >= len(w.carried[i]) {
		return ItemStack{}, false
	}
	old := w.carried[i][index]
	if old.Code == 0 || old.Count == 0 {
		return ItemStack{}, false
	}
	out := old.Clone()
	if !whole {
		out.Count = 1
	}
	if old.ObjectID != 0 {
		if w.savedObjects == nil {
			return ItemStack{}, false
		}
		var id SavedObjectID
		var err error
		if whole {
			id, err = w.savedObjects.TakeWholeAt(old.ObjectID, SavedItemLocation{w.savedPackOwner(i), uint32(index)}, old)
		} else {
			id, err = w.savedObjects.TakeOneAt(old.ObjectID, SavedItemLocation{w.savedPackOwner(i), uint32(index)}, old)
		}
		if err != nil || !w.fillSplitObjectSpell(id) {
			return ItemStack{}, false
		}
		out = w.savedObjects.item(id).Value.Clone()
		if !whole && old.Count > 1 && !w.syncSavedItemViews(old.ObjectID, old, i, otherActive...) {
			return ItemStack{}, false
		}
	}
	if whole || old.Count == 1 {
		w.carried[i] = slices.Delete(w.carried[i], index, index+1)
	} else if old.ObjectID == 0 {
		w.carried[i][index].Count--
	}
	if old.ObjectID != 0 && w.entities[i].ActorLoad.Present {
		w.entities[i].ActorLoad.InsertIndex = uint32(index)
	}
	return out, true
}

func (w *World) fillSplitObjectSpell(id SavedObjectID) bool {
	row := w.savedObjects.item(id)
	if row.Spell == 0 || row.Origin.Kind != SavedObjectSplit || row.Coverage.Unknown&SavedUnknownSpellInitialization == 0 {
		return true
	}
	child := w.savedObjects.spell(row.Spell)
	rule, ok := w.findSpell(uint32(child.Value.ID))
	if !ok {
		return true // native placeholder stays explicitly uncovered
	}
	value := sourceSpellFromRule(rule)
	child.Value, row.Value.SourceEquipment.Spell = value, value
	child.Coverage.Unknown &^= SavedUnknownSpellInitialization
	row.Coverage.Unknown &^= SavedUnknownSpellInitialization
	return true
}

func sourceSpellFromRule(rule SpellRule) SourceItemSpell {
	defensive := uint8(0)
	if rule.Defensive {
		defensive = 1
	}
	return SourceItemSpell{Present: true, ID: uint8(rule.ID), Range: rule.MaxRange, Defensive: defensive, ManaCost: uint16(rule.ManaCost)}
}

func (w *World) putCarriedObject(i int, item ItemStack, otherActive ...int) bool {
	return w.putCarriedObjectAt(i, item, -1, 0, otherActive...)
}

func (w *World) putCarriedObjectAt(i int, item ItemStack, index int, merge SavedObjectID, otherActive ...int) bool {
	if w.savedObjects == nil || item.ObjectID == 0 || !w.hasActorContainer(i) {
		return false
	}
	if w.savedObjects.container(w.savedPackOwner(i)) == nil {
		w.savedObjects.Containers = append(w.savedObjects.Containers, SavedObjectContainer{Owner: w.savedPackOwner(i)})
	}
	w.syncSavedPack(i)
	if index >= 0 {
		w.savedObjects.container(w.savedPackOwner(i)).InsertIndex = uint32(index)
	}
	oldValues := make(map[SavedObjectID]ItemStack)
	for _, row := range w.savedObjects.Items {
		oldValues[row.ID] = row.Value.Clone()
	}
	id, at, err := w.savedObjects.insertSelectedAt(item.ObjectID, w.savedPackOwner(i), SavedMergeNativeRetention, merge, index >= 0)
	if err != nil {
		return false
	}
	row := w.savedObjects.item(id)
	if id != item.ObjectID {
		return w.syncSavedItemViews(id, oldValues[id], i, otherActive...)
	}
	if uint64(at) > uint64(len(w.carried[i])) {
		return false
	}
	w.carried[i] = slices.Insert(w.carried[i], int(at), row.Value.Clone())
	return true
}

func (w *World) retireSessionObject(item ItemInstance, handles ...uint64) bool {
	if item.ObjectID == 0 {
		return true
	}
	if w.savedObjects == nil {
		return false
	}
	r := w.savedObjects.Clone()
	row := r.item(item.ObjectID)
	handle := r.sessionHandle(item.ObjectID, handles...)
	if row == nil || handle == 0 {
		return false
	}
	if _, err := r.ImportExternal(handle, r.stackForItem(item)); err != nil || r.Dispose(item.ObjectID) != nil {
		return false
	}
	w.savedObjects = r
	return true
}

func (w *World) takeWornObject(i, slot int, item ItemInstance) bool {
	if item.ObjectID == 0 {
		return true
	}
	_, err := w.savedObjects.TakeWhole(item.ObjectID, SavedObjectOwner{Kind: SavedOwnerActorWorn, Entity: w.entities[i].ID, Slot: uint32(slot)}, w.savedObjects.stackForItem(item))
	return err == nil
}

func (w *World) putWornObject(i, slot int, item ItemInstance) bool {
	if item.ObjectID == 0 {
		return true
	}
	row := w.savedObjects.item(item.ObjectID)
	if row == nil || !StackStateEqual(row.Value, w.savedObjects.stackForItem(item)) {
		return false
	}
	_, err := w.savedObjects.Insert(item.ObjectID, SavedObjectOwner{Kind: SavedOwnerActorWorn, Entity: w.entities[i].ID, Slot: uint32(slot)}, SavedMergeNone)
	return err == nil
}

// replaceObjectSpell is the nested lifecycle of ITEM-SPELLMOVE-132. Known
// constructor fields come from SpellRule; the runtime pointer remains unknown.
func (w *World) replaceObjectSpell(item *ItemInstance, value SourceItemSpell) bool {
	if item.ObjectID == 0 {
		item.SourceEquipment.Spell = value
		return true
	}
	before := w.savedObjects.stackForItem(*item)
	err := w.savedObjects.mutate(func(r *SavedObjects) error {
		row := r.item(item.ObjectID)
		if row == nil || !StackStateEqual(row.Value, r.stackForItem(*item)) {
			return fmt.Errorf("sim: stale Weapon Spell producer")
		}
		row.Spell = 0
		row.Value.SourceEquipment.Spell = value
		row.Coverage.Unknown &^= SavedUnknownSpellInitialization
		if value.Present {
			id, err := r.mint()
			if err != nil {
				return err
			}
			row.Spell = id
			r.Spells = append(r.Spells, SavedSpellObject{ID: id, Origin: SavedObjectOrigin{Kind: SavedObjectGenerated}, Value: value, Coverage: SavedObjectCoverage{Unknown: SavedUnknownIdentity}})
		}
		r.RefreshChildLiveness()
		return nil
	})
	if err != nil {
		return false
	}
	item.SourceEquipment.Spell = value
	return w.syncSavedItemViews(item.ObjectID, before, -1)
}
