package sim

import "slices"

// TradeCarried applies one Shop deal to hero's pack and its owner's purse as
// one atomic operation that advances no clock, the rule UseCarriedPotion and
// CheatAddItem follow. Each sold value names a pack element in CarriedStacks
// order and takes one unit from it; each bought stack joins the pack as a
// pickup does; the purse then holds purse. A refused part leaves the world
// unchanged and reports false.
func (w *World) TradeCarried(hero EntityID, sold []int, bought []ItemStack, purse uint32) bool {
	i := indexOfEntity(w.entities, hero)
	if i < 0 || w.entities[i].Owner >= relationSlots || !w.hasActorContainer(i) || !w.sourceMutationReady(i) {
		return false
	}
	for _, stack := range bought {
		if stack.ObjectID != 0 || stack.Code == 0 || stack.Count == 0 || stack.Count > 65535 || len(stack.Effects) > 4096 || stack.Instance().ValidateWeight() != nil {
			return false
		}
	}
	if len(bought) > 0 && !w.cheatHoldingCapacity(bought, nil, false) {
		return false
	}
	order := slices.Clone(sold)
	slices.Sort(order)
	slices.Reverse(order)
	for k := 0; k < len(order); {
		index, run := order[k], 1
		for k+run < len(order) && order[k+run] == index {
			run++
		}
		if index < 0 || index >= len(w.carried[i]) || uint64(w.carried[i][index].Count) < uint64(run) {
			return false
		}
		k += run
	}
	n := w.sourceMutationCopy(i)
	before := n.beginLoadMutation(i)
	for _, index := range order {
		item, ok := n.takeCarriedObject(i, index, false)
		if !ok || (item.ObjectID != 0 && n.savedObjects.Dispose(item.ObjectID) != nil) {
			return false
		}
	}
	for _, stack := range bought {
		if !n.addCarried(i, stack) {
			return false
		}
	}
	if !n.finishLoadMutation(i, before) || !n.savedMutationValid() {
		return false
	}
	n.purses[n.entities[i].Owner] = purse
	if len(bought) > 0 {
		n.publishActorItems(i, SelfSlot, false)
	}
	*w = n
	return true
}
