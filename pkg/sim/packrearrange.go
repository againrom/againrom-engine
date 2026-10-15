package sim

type PackDragPlan struct {
	Merge bool
	Count uint32
	Form  ItemInstance
}

// PlanPackDrag selects the quantity and equal-stack merge or whole-source swap.
func PlanPackDrag(held, incoming ItemStack, whole bool, construct func(ItemInstance) ItemInstance) (PackDragPlan, bool) {
	if held.Count == 0 || incoming.Count == 0 || held.Code == 0 || incoming.Code == 0 || held.ObjectID != 0 && held.ObjectID == incoming.ObjectID {
		return PackDragPlan{}, false
	}
	form, join := EffectiveJoin(held.Instance(), incoming.Instance(), construct)
	count := uint32(1)
	if whole || incoming.Count == 1 {
		count = incoming.Count
	}
	if !join && count != incoming.Count || join && uint64(held.Count)+uint64(count) > uint64(^uint32(0)) {
		return PackDragPlan{}, false
	}
	return PackDragPlan{Merge: join, Count: count, Form: form}, true
}

func (w *World) rearrangeCarried(i, source, target int, whole bool) {
	if source == target || source < 0 || target < 0 || source >= len(w.carried[i]) || target >= len(w.carried[i]) || !w.sourceMutationReady(i) {
		return
	}
	held, incoming := w.carried[i][target], w.carried[i][source]
	construct := func(v ItemInstance) ItemInstance {
		if len(w.mergeOperands) == 0 {
			return v
		}
		if built := w.sourceConstructItem(v); built.SourceEquipment.Class != 0 {
			return built
		}
		return constructItemWith(w.mergeOperands, v)
	}
	plan, ok := PlanPackDrag(held, incoming, whole, construct)
	if !ok {
		return
	}
	n := w.sourceMutationCopy(i)
	before := n.beginLoadMutation(i)
	if !plan.Merge {
		n.carried[i][source], n.carried[i][target] = n.carried[i][target], n.carried[i][source]
	} else {
		item, ok := n.takeCarriedObject(i, source, plan.Count == incoming.Count)
		if !ok {
			return
		}
		if plan.Count == incoming.Count && source < target {
			target--
		}
		v := StackItem(plan.Form, item.Count)
		v.ObjectID = item.ObjectID
		if held.ObjectID != 0 && item.ObjectID != 0 {
			ok = n.mergeBoundIntoHeld(i, target, item, v)
		} else {
			ok = n.mergeIntoHeld(i, target, v, 0)
		}
		if !ok {
			return
		}
	}
	if n.finishLoadMutation(i, before) && n.savedMutationValid() {
		*w = n
	}
}
