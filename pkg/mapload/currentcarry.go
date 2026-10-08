package mapload

import "againrom/pkg/sim"

// MaterializePartyCarry makes the current canonical holdings explicit without
// inventing an actor load record or changing the member's spawn operands.
func MaterializePartyCarry(member PartyMember, table *Table) PartyMember {
	p := clonePartyMember(member)
	if p.Carry != nil {
		return p
	}
	items := MemberCarriedItems(p, table)
	worn := MemberItemEquipment(p, table)
	for i := range worn {
		worn[i] = SourceConstructedItem(worn[i], table)
	}
	c := &Carry{SkillXP: p.Hero.Reward().SkillXP, OrderedStacks: sim.FoldItems(items), EquippedItems: worn}
	c.ItemInstances = items[:0]
	for _, stack := range c.OrderedStacks {
		for n := uint32(0); n < stack.Count; n++ {
			c.ItemInstances = append(c.ItemInstances, stack.Instance())
			c.Items = append(c.Items, stack.Code)
		}
	}
	for slot, item := range worn {
		c.Equipped[slot] = item.Code
	}
	p.Carry = c
	return p
}
