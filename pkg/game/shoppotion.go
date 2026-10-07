package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func (t *townScreen) shopPotionUser(item sim.ItemInstance) (*mapload.PartyMember, mapload.PartyMember, ui.TownAction, bool) {
	if item.Kind == 4 {
		return nil, mapload.PartyMember{}, ui.TownAction{Msg: "use a scroll on the mission map"}, true
	}
	if item.Kind != 3 {
		return nil, mapload.PartyMember{}, ui.TownAction{}, false
	}
	p := t.shopPartyMember(t.shopMemberIndex())
	if p == nil {
		return nil, mapload.PartyMember{}, ui.TownAction{}, true
	}
	result, ok := mapload.ApplyTownPotion(*p, item, t.in.Table)
	if !ok {
		return nil, mapload.PartyMember{}, ui.TownAction{Msg: "cannot use that potion"}, true
	}
	return p, result, ui.TownAction{}, true
}

func commitTownPotion(member *mapload.PartyMember, result mapload.PartyMember) bool {
	if result.Carry != nil && result.Carry.LiveLoad != nil && result.Carry.LiveLoad.Inventory.Source.Class != 0 {
		return mapload.CommitSourceTownPotion(member, result)
	}
	// The transaction already removed one item from member's pack. Do not
	// overwrite that updated container with the preflight copy's inventory.
	member.RetireOriginalHuman()
	member.Hero, member.PotionEffect = result.Hero, result.PotionEffect
	if member.Saved != nil && result.Saved != nil {
		member.Saved.HP, member.Saved.Mana = result.Saved.HP, result.Saved.Mana
	}
	if member.Carry != nil && member.Carry.LiveLoad != nil && result.Carry != nil && result.Carry.LiveLoad != nil {
		state := *member.Carry.LiveLoad
		state.Capacity, state.Speed, state.Movement = result.Carry.LiveLoad.Capacity, result.Carry.LiveLoad.Speed, sim.HumanMovement{}
		state.Load = state.Inventory.CurrentLoad()
		member.Carry.LiveLoad = &state
	}
	return true
}
