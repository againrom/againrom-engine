package mapload

import (
	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// PartyHumanRow is the installed Humans row a party member's actor is bound
// to: the chargen row for the starting hero, the campaign row for a companion,
// the hire row for a Human mercenary, else the first row of his TypeID. Zero
// is no row. The SAV writer stores this row and the mission start reads the
// member's dying time from it (DAT-SCHEMA-007).
func PartyHumanRow(member, hero PartyMember, t *Table) int {
	if t == nil || t.Humans == nil {
		return 0
	}
	if member.StartingHero {
		fig := data.FigureDir(member.FigureDir)
		_, row, ok := data.ChargenBase(t.Humans, fig.Mage(), fig.Female())
		if !ok || row <= 0 || row > 255 {
			return 0
		}
		return row
	}
	if member.CompanionNPC != 0 {
		heroFemale := data.FigureDir(hero.FigureDir).Female()
		if serverID, ok := t.NPC.CampaignServerID(int32(member.CompanionNPC), 0, hero.Mage, heroFemale); ok {
			if row := data.FindHumanByServerID(t.Humans, serverID); row > 0 && row <= 255 {
				return row
			}
		}
	}
	if member.MercenaryType != 0 && member.MercenaryType != 1 && member.MercenaryType != 2 {
		if row := int(member.DefinitionRow); row > 0 && row < t.Humans.Len() {
			return row
		}
		if row := data.FindHumanByName(t.Humans, member.Name); row > 0 && row <= 255 {
			return row
		}
	}
	row := data.FindHumanByType(t.Humans, member.Class)
	if row <= 0 || row > 255 {
		return 0
	}
	return row
}

// partyDyingTime is the dying time SourceActorSeed reads for the binding the
// SAV writer gives this member, so a fresh member and the same member loaded
// from a SAV take one value. A member with no bound row keeps none.
func partyDyingTime(member, hero PartyMember, typeID int32, t *Table) (int32, error) {
	row := PartyHumanRow(member, hero, t)
	binding := sim.SourceBinding{Class: sim.GeneratedHumanBinding, TokenRow: uint8(row), TypeID: uint16(typeID)}
	if row <= 0 && binding.DefinitionRow() == 0 || t == nil || t.Humans == nil || int(binding.DefinitionRow()) >= t.Humans.Len() {
		return 0, nil
	}
	seed, err := SourceActorSeed(binding, t)
	if err != nil {
		return 0, err
	}
	return seed.DyingTime, nil
}
