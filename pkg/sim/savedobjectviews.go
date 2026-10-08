package sim

import "slices"

// Worn and session views have no quantity field. Their exact node owns Count;
// the instance still supplies every value that the view actually represents.
func (r *SavedObjects) stackForItem(item ItemInstance) ItemStack {
	count := uint32(1)
	if r != nil {
		if row := r.item(item.ObjectID); row != nil {
			count = row.Value.Count
		}
	}
	return StackItem(item, count)
}

// A scalar change is a node operation. Every native occurrence observes it;
// the caller still owns the surrounding detached World transaction. This is
// native alias policy, not a claim about original mutation callbacks.
func (w *World) syncSavedItemViews(id SavedObjectID, before ItemStack, active int, otherActive ...int) bool {
	row := w.savedObjects.item(id)
	if row == nil || row.Retired {
		return false
	}
	for i := range w.entities {
		load := w.beginLoadMutation(i)
		changed := false
		for j, old := range w.carried[i] {
			if old.ObjectID == id {
				w.carried[i][j] = row.Value.Clone()
				changed = true
			}
		}
		for j, old := range w.equipment[i] {
			if old.ObjectID != id {
				continue
			}
			item := row.Value.Instance()
			w.equipment[i][j] = item.Clone()
			w.applyEquipmentItemState(&w.entities[i], old, item, w.spells, w.damageObservation)
			if j == slotWeapon {
				syncWeaponItem(&w.entities[i], item)
			}
			changed = true
		}
		if changed && i != active && !slices.Contains(otherActive, i) && !w.finishLoadMutation(i, load) {
			return false
		}
	}
	for i := range w.sacks {
		s := &w.sacks[i]
		var units []ItemInstance
		for at := 0; at < len(s.ItemInstances); {
			old := s.ItemInstances[at]
			if old.ObjectID != id {
				units = append(units, old.Clone())
				at++
				continue
			}
			if before.Count == 0 || uint64(before.Count) > uint64(len(s.ItemInstances)-at) {
				return false
			}
			for j := uint32(0); j < before.Count; j++ {
				if s.ItemInstances[at+int(j)].ObjectID != id {
					return false
				}
			}
			at += int(before.Count)
			for j := uint32(0); j < row.Value.Count; j++ {
				units = append(units, row.Value.Instance().Clone())
			}
		}
		s.ItemInstances = units
		s.Items = itemCodes(units)
	}
	for i := range w.scrollCasts {
		if w.scrollCasts[i].Item.ObjectID == id {
			w.scrollCasts[i].Item = row.Value.Instance().Clone()
		}
	}
	return true
}
