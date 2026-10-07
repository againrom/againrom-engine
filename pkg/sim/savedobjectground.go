package sim

import (
	"encoding/binary"
	"slices"
)

func (w *World) savedGroundIndex(x, y int32) int {
	for i, s := range w.sacks {
		if s.X == x && s.Y == y {
			return i
		}
	}
	return -1
}

func (w *World) syncSavedSackSlots(j int) bool {
	s := &w.sacks[j]
	c := w.savedObjects.container(SavedObjectOwner{Kind: SavedOwnerSack, Object: s.ObjectID})
	c.Items = nil
	for at := 0; at < len(s.ItemInstances); {
		id := s.ItemInstances[at].ObjectID
		c.Items = append(c.Items, id)
		if id == 0 {
			at++
			c.Coverage.Unknown |= SavedUnknownContainerLoad | SavedUnknownMergePolicy
		} else {
			row := w.savedObjects.item(id)
			if row == nil || row.Value.Count == 0 || uint64(row.Value.Count) > uint64(len(s.ItemInstances)-at) {
				return false
			}
			for k := uint32(0); k < row.Value.Count; k++ {
				if s.ItemInstances[at+int(k)].ObjectID != id {
					return false
				}
			}
			at += int(row.Value.Count)
		}
	}
	return true
}

// New native Sacks have native identities but no fabricated original Token
// constructor fields or cell registration keys beyond the ITEM-SACK-010
// value slot (SackTokenValue), which every successful mutation below
// recomputes rather than leaving at its zero construction value. Existing
// source Sacks retain their exact row. The enclosing item operation owns
// this whole candidate.
func (w *World) putGroundObject(x, y int32, gold uint32, item ItemStack, adopt ...bool) bool {
	return w.putGroundObjectAt(-1, x, y, gold, item, adopt...)
}

func (w *World) putGroundObjectAt(active int, x, y int32, gold uint32, item ItemStack, adopt ...bool) bool {
	if w.savedObjects == nil || sackFault(w.bounds, Sack{X: x, Y: y}) != nil {
		return false
	}
	j := w.savedGroundIndex(x, y)
	fresh := j < 0
	if fresh {
		w.pourSack(x, y, gold, nil)
		j = w.savedGroundIndex(x, y)
	} else {
		w.sacks[j].Gold += gold
	}
	s := &w.sacks[j]
	if s.ObjectID == 0 {
		id, err := w.savedObjects.mint()
		if err != nil {
			return false
		}
		s.ObjectID = id
		// SAV-655, SAV-1093: an original Sack token carries mask 2 at
		// Token+0x18; its runtime id comes from the allocator.
		token := SavedObjectToken{RuntimeID: w.constructedSackRuntimeID(), T18: 2}
		binary.LittleEndian.PutUint16(token.Position[2:], uint16(x)|uint16(y)<<8)
		w.savedObjects.Sacks = append(w.savedObjects.Sacks, SavedSackObject{ID: id, Origin: SavedObjectOrigin{Kind: SavedObjectGenerated}, Token: token, Gold: s.Gold, Coverage: SavedObjectCoverage{Unknown: SavedUnknownToken}})
		w.savedObjects.SackRoots = append(w.savedObjects.SackRoots, id)
		w.savedObjects.Containers = append(w.savedObjects.Containers, SavedObjectContainer{Owner: SavedObjectOwner{Kind: SavedOwnerSack, Object: id}, Present: true, InsertIndex: 10000, Coverage: SavedObjectCoverage{Unknown: SavedUnknownContainerLoad | SavedUnknownMergePolicy}})
	}
	if !w.syncSavedSackSlots(j) {
		return false
	}
	c := w.savedObjects.container(SavedObjectOwner{Kind: SavedOwnerSack, Object: s.ObjectID})
	w.savedObjects.sack(s.ObjectID).Gold = s.Gold
	if item.Count == 0 {
		w.savedObjects.sack(s.ObjectID).Token.T1C = w.sackRecordValue(*s)
		return true
	}
	if item.ObjectID == 0 {
		for range item.Count {
			s.ItemInstances = append(s.ItemInstances, item.Instance())
			c.Items = append(c.Items, 0)
		}
		c.Coverage.Unknown |= SavedUnknownContainerLoad | SavedUnknownMergePolicy
		c.weight(item, item.Count, true)
		s.Items = itemCodes(s.ItemInstances)
		w.savedObjects.sack(s.ObjectID).Token.T1C = w.sackRecordValue(*s)
		return true
	}
	policy := SavedMergeNativeRetention
	if fresh || len(adopt) != 0 && adopt[0] {
		policy = SavedMergeNone
	}
	// Nonmerge ground insertions append. The retained original cursor remains
	// an operand, not a claimed reconstruction of an unobserved callback.
	c.InsertIndex = uint32(len(c.Items))
	unbound := make([]ItemInstance, 0)
	for _, value := range s.ItemInstances {
		if value.ObjectID == 0 {
			unbound = append(unbound, value)
		}
	}
	previous := make(map[SavedObjectID]ItemStack, len(c.Items))
	for _, id := range c.Items {
		if row := w.savedObjects.item(id); row != nil {
			previous[id] = row.Value.Clone()
		}
	}
	retained, err := w.savedObjects.Insert(item.ObjectID, c.Owner, policy)
	if err != nil {
		return false
	}
	if retained != item.ObjectID && !w.syncSavedItemViews(retained, previous[retained], active) {
		return false
	}
	c = w.savedObjects.container(SavedObjectOwner{Kind: SavedOwnerSack, Object: s.ObjectID})
	s.ItemInstances = nil
	for _, id := range c.Items {
		if id == 0 {
			if len(unbound) == 0 {
				return false
			}
			s.ItemInstances = append(s.ItemInstances, unbound[0].Clone())
			unbound = unbound[1:]
			continue
		}
		value := w.savedObjects.item(id).Value
		for range value.Count {
			s.ItemInstances = append(s.ItemInstances, value.Instance())
		}
	}
	s.Items = itemCodes(s.ItemInstances)
	w.savedObjects.sack(s.ObjectID).Token.T1C = w.sackRecordValue(*s)
	return len(unbound) == 0
}

func (w *World) savedDropCell(i int, x, y int32) (int32, int32, bool) {
	return w.dropWindowCell(i, x, y)
}

func (w *World) dropSavedCarried(i, index int, x, y int32) bool {
	x, y, ok := w.savedDropCell(i, x, y)
	if !ok || !w.sourceMutationReady(i) {
		return false
	}
	before := w.beginLoadMutation(i)
	item, ok := w.takeCarriedObject(i, index, false)
	if !ok || !w.putGroundObjectAt(i, x, y, 0, item) || !w.finishLoadMutation(i, before) {
		return false
	}
	return w.savedMutationValid()
}

func (w *World) dropSavedPack(i int, gold uint32) bool {
	return w.dropSavedPackAt(i, gold, w.entities[i].X, w.entities[i].Y)
}

// A newly created Sack adopts the whole source container's ordered objects.
// An existing Sack receives the repeated one-unit drain instead.
func (w *World) dropSavedPackAt(i int, gold uint32, x, y int32) bool {
	if sackFault(w.bounds, Sack{X: x, Y: y}) != nil {
		return false
	}
	existing := w.savedGroundIndex(x, y) >= 0
	var adopted *SavedObjectContainer
	if c := w.savedObjects.container(w.savedPackOwner(i)); c != nil && !existing {
		copy := *c
		copy.Items = slices.Clone(c.Items)
		adopted = &copy
	}
	first := true
	for len(w.carried[i]) > 0 {
		item, ok := w.takeCarriedObject(i, 0, !existing)
		if !ok || !w.putGroundObjectAt(i, x, y, gold, item, !existing) {
			return false
		}
		gold, first = 0, false
	}
	if first && gold != 0 && !w.putGroundObjectAt(i, x, y, gold, ItemStack{}) {
		return false
	}
	if adopted != nil && !first {
		s := w.sacks[w.savedGroundIndex(x, y)]
		c := w.savedObjects.container(SavedObjectOwner{Kind: SavedOwnerSack, Object: s.ObjectID})
		c.InsertIndex, c.Accumulator, c.Coverage = adopted.InsertIndex, adopted.Accumulator, adopted.Coverage
	}
	return true
}

func (w *World) disposeSavedPack(i int) bool {
	for len(w.carried[i]) > 0 {
		item, ok := w.takeCarriedObject(i, 0, true)
		if !ok || item.ObjectID != 0 && w.savedObjects.Dispose(item.ObjectID) != nil {
			return false
		}
	}
	return true
}

func (w *World) terminalSavedNative(i int, x, y int32, validCell bool) bool {
	e := &w.entities[i]
	if !validCell && !e.SuppressCorpseLoot {
		return true
	}
	before := w.beginLoadMutation(i)
	for _, slot := range []int{2, 1, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12} {
		item := w.equipment[i][slot-1].Clone()
		if item.Empty() {
			continue
		}
		if slot == 1 && item.innateWeapon() {
			continue
		}
		if !w.takeWornObject(i, slot, item) {
			return false
		}
		w.equipment[i][slot-1] = ItemInstance{}
		if !w.addCarried(i, StackItem(item, 1)) {
			return false
		}
	}
	syncWeaponItem(e, ItemInstance{})
	if e.SuppressCorpseLoot && !w.disposeSavedPack(i) {
		return false
	}
	if validCell && !w.dropSavedPackAt(i, w.deathGold(i), x, y) {
		return false
	}
	if e.ActorLoad.Present {
		e.ActorLoad.ContainerPresent, e.ActorLoad.InsertIndex = true, 10000
	}
	return w.finishLoadMutation(i, before)
}
