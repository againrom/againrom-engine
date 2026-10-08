package sim

// RepairNativePackCells joins independent native records after current SAV
// restoration. Aliases, bound objects and active inventory indices stay intact.
func (w *World) RepairNativePackCells(ids, blocked []EntityID) int {
	if w == nil {
		return 0
	}
	uses := w.nativeItemIdentityUses()
	for _, cast := range w.scrollCasts {
		blocked = append(blocked, cast.Caster)
	}
	merged := 0
	for _, id := range ids {
		i := indexOfEntity(w.entities, id)
		if i < 0 {
			continue
		}
		skip := false
		for _, b := range blocked {
			skip = skip || b == id
		}
		if skip {
			continue
		}
		old := w.carried[i]
		var out []ItemStack
		cursor := w.entities[i].ActorLoad.InsertIndex
		removedBefore := uint32(0)
		eligible := func(st ItemStack) bool {
			return st.ObjectID == 0 && st.Count != 0 && st.NativeRecord != nil && uses[st.NativeRecord.Token.Identity] <= 1
		}
		for j, st := range old {
			joined := false
			if eligible(st) {
				for k, held := range out {
					if eligible(held) && uint64(held.Count)+uint64(st.Count) <= uint64(^uint32(0)) && CanMergeItemValues(held.Instance(), st.Instance()) {
						out[k].Count += st.Count
						joined = true
						merged++
						if uint32(j) < cursor {
							removedBefore++
						}
						break
					}
				}
			}
			if !joined {
				out = append(out, st.Clone())
			}
		}
		if len(out) != len(old) {
			w.carried[i] = out
			w.entities[i].ActorLoad.InsertIndex = cursor - removedBefore
			w.syncSavedPack(i)
		}
	}
	return merged
}

func (w *World) nativeItemIdentityUses() map[uint32]int {
	uses := map[uint32]int{}
	observe := func(item ItemInstance) {
		if item.NativeRecord != nil && item.NativeRecord.Token.Identity != 0 {
			uses[item.NativeRecord.Token.Identity]++
		}
	}
	for i, stacks := range w.carried {
		for _, st := range stacks {
			observe(st.Instance())
		}
		for _, item := range w.equipment[i] {
			observe(item)
		}
	}
	for _, sack := range w.sacks {
		for _, item := range sack.ItemInstances {
			observe(item)
		}
	}
	for _, cast := range w.scrollCasts {
		observe(cast.Item)
	}
	return uses
}
