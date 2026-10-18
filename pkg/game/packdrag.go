package game

import (
	"slices"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func (t *shopController) shopMoveWithinPack(source, target int, whole bool) ui.TownAction {
	stacks := t.shopPackStacks()
	if source == target || source < 0 || target < 0 || source >= len(stacks) || target >= len(stacks) {
		return ui.TownAction{}
	}
	return t.withCityShopMutation(func(n *shopController) (ui.TownAction, error) {
		return ui.TownAction{}, n.cityShopMovePack(source, target, whole)
	})
}

func (t *shopController) cityShopMovePack(source, target int, whole bool) error {
	from, incoming, err := t.cityShopPackSelection(source)
	if err != nil {
		return err
	}
	to, held, err := t.cityShopPackSelection(target)
	if err != nil {
		return err
	}
	held.ObjectID, incoming.ObjectID = to.ID, from.ID
	plan, ok := sim.PlanPackDrag(held, incoming, whole, func(v sim.ItemInstance) sim.ItemInstance {
		return cityConstructItem(v, t.in.Table)
	})
	if !ok {
		return shopRefusal("pack move refused")
	}
	memberIndex := t.shopMemberIndex()
	if !plan.Merge {
		member := t.sess.Carried[memberIndex]
		stacks := slices.Clone(cityMemberStacks(member, t.in.Table))
		root, err := cityMutationParty(t.sess.Town.cityObjects, member.ID)
		if err != nil {
			return err
		}
		stacks[source], stacks[target] = stacks[target], stacks[source]
		root.Pack[source], root.Pack[target] = root.Pack[target], root.Pack[source]
		return t.cityShopSetPack(memberIndex, stacks, true)
	}
	item, err := t.cityShopTakePack(source, plan.Count == incoming.Count, false)
	if err != nil {
		return err
	}
	if plan.Count == incoming.Count && source < target {
		target--
	}
	to, held, err = t.cityShopPackSelection(target)
	if err != nil {
		return err
	}
	destination := shopItemFromInstance(held.Instance(), int32(held.Count))
	destination.cityID = to.ID
	if _, err := t.cityShopMerge(destination, item); err != nil {
		return err
	}
	return t.cityShopSetPack(memberIndex, cityMemberStacks(t.sess.Carried[memberIndex], t.in.Table), true)
}
