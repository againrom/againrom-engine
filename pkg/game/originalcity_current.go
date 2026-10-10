package game

import (
	"reflect"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// currentCityHuman is the member's Human from its current class-2 actor
// load, with the runtime words that load carries.
func currentCityHuman(member mapload.PartyMember) (data.HumanState, *sav.CityHumanRuntime, bool) {
	if member.Carry == nil || member.Carry.LiveLoad == nil {
		return data.HumanState{}, nil, false
	}
	a := member.Carry.LiveLoad
	s := a.Inventory.Source
	h := mapload.SourceHumanState(s, a.Inventory.Accumulator)
	if s.Class != 2 || a.Validate() != nil || h.Hero() != member.Hero {
		return data.HumanState{}, nil, false
	}
	return h, &sav.CityHumanRuntime{Reach: s.Reach, AttackCharge: s.AttackCharge, AttackRelax: s.AttackRelax,
		HealthHundredths: a.HealthHundredths, ManaHundredths: a.ManaHundredths}, true
}

// replayedMember reports whether an unreturned member still equals the
// replay of its document baseline and admitted school and sale operations.
func (b originalCityBinding) replayedMember(member mapload.PartyMember, c Campaign) (bool, error) {
	if b.returned != nil || b.salesVersion == 0 && len(b.sales) != 0 {
		return false, nil
	}
	expected, err := b.expectedMember()
	if err != nil {
		return false, err
	}
	if nativeCityGraftableMember(member, c) {
		// A grafted companion never entered a live simulation, so the
		// replay's derived Carry, Saved, Weapon, Human and Book have no live
		// counterpart. Worn codes, Carried, KnownSpells, Hero and identity
		// stay compared.
		strip := func(m *mapload.PartyMember) {
			m.OriginalHuman, m.Weapon, m.Carry, m.Saved = nil, nil, nil, nil
			m.Book, m.SpellbookRestored, m.SpellbookPresent = sim.Spellbook{}, false, false
			m.WornItems = [sim.EquipSlots]sim.ItemInstance{}
		}
		strip(&member)
		strip(&expected)
	}
	return reflect.DeepEqual(member, expected), nil
}
