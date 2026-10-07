package sim

// sourceMutationCopy isolates the owners touched by an item producer. The
// caller publishes this candidate only after all source callbacks succeed.
// Death normalization runs after commit, never against the shallow shared
// sack/script/AI owners of this bounded candidate.
func (w *World) sourceMutationCopy(i int) World {
	n := *w
	if w.damageObservation != nil {
		n.damageObservation = &damageObservation{events: append([]DamageEvent(nil), w.damageObservation.events...)}
	}
	n.savedGroups = cloneSavedGroups(w.savedGroups)
	n.savedObjects = w.savedObjects.Clone()
	n.savedMotion = cloneActorMotions(w.savedMotion)
	if w.savedCellPlanes != nil {
		planes := *w.savedCellPlanes
		n.savedCellPlanes = &planes
	}
	n.sacks = make([]Sack, len(w.sacks))
	for j, sack := range w.sacks {
		n.sacks[j] = makeSack(sack.X, sack.Y, sack.Gold, sack.ItemInstances)
		n.sacks[j].ObjectID = sack.ObjectID
	}
	n.grid = append([]byte(nil), w.grid...)
	n.cost = append([]byte(nil), w.cost...)
	n.cellTails = append([]cellTail(nil), w.cellTails...)
	n.savedStructureCells = append([]SavedStructureCell(nil), w.savedStructureCells...)
	n.rebuildStructureSlots()
	n.entities = append([]Entity(nil), w.entities...)
	n.carried = append([][]ItemStack(nil), w.carried...)
	for j := range n.carried {
		n.carried[j] = cloneStacks(w.carried[j])
	}
	n.equipment = append([][EquipSlots]ItemInstance(nil), w.equipment...)
	for j := range n.equipment {
		n.equipment[j] = cloneEquipment(w.equipment[j])
	}
	n.attached = append([]attachedEffect(nil), w.attached...)
	n.scrollCasts = append([]ScrollCast(nil), w.scrollCasts...)
	n.routes = append([][]cell(nil), w.routes...)
	return n
}
