package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// shopPotionUser refuses a scroll. A usable potion is consumed by the caller and
// changes nothing: the town has no clock (DIV-1691).
func (t *townScreen) shopPotionUser(item sim.ItemInstance) (*mapload.PartyMember, ui.TownAction, bool) {
	if item.Kind == 4 {
		return nil, ui.TownAction{}, true
	}
	if item.Kind != 3 {
		return nil, ui.TownAction{}, false
	}
	p := t.shopPartyMember(t.shopMemberIndex())
	if p == nil || !sim.UsablePotion(item) {
		return nil, ui.TownAction{}, true
	}
	return p, ui.TownAction{}, true
}
