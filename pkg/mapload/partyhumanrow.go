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
		if !ok || row <= 0 || row > maxTokenRow {
			// A hero an edition binds by identity is built from a Start_
			// template (R2-ENGINE-105), not a chargen row; he keeps the row
			// his loaded record names (R2-ENGINE-319: template load writes
			// it to +0xc).
			if row := int(member.DefinitionRow); t.Edition != nil && t.Edition.BindsByIdentity() && row > 0 && row < t.Humans.Len() {
				return row
			}
			return 0
		}
		return row
	}
	if member.CampaignHireID() != 0 {
		return int(member.DefinitionRow)
	}
	if row, ok := identityRow(member, t); ok {
		return row
	}
	if member.CompanionNPC != 0 {
		heroFemale := data.FigureDir(hero.FigureDir).Female()
		if serverID, ok := t.NPC.CampaignServerID(int32(member.CompanionNPC), 0, hero.Mage, heroFemale); ok {
			if row := data.FindHumanByServerID(t.Humans, serverID); row > 0 && row <= maxTokenRow {
				return row
			}
		}
	}
	if member.MercenaryType != 0 && member.MercenaryType != 1 && member.MercenaryType != 2 {
		if row := int(member.DefinitionRow); row > 0 && row < t.Humans.Len() {
			return row
		}
		if row := data.FindHumanByName(t.Humans, member.Name); row > 0 && row <= maxTokenRow {
			return row
		}
	}
	row := data.FindHumanByType(t.Humans, member.Class)
	if row <= 0 || row > maxTokenRow {
		return 0
	}
	return row
}

// PartyIdentity is the party identity member carries under the table's
// edition: the hero's own, or a companion's identity row (R2-ENGINE-105,
// R2-ENGINE-106). An edition with no hero identity gives none.
func PartyIdentity(member PartyMember, t *Table) (int, bool) {
	if t == nil || t.Edition == nil || !t.Edition.BindsByIdentity() {
		return 0, false
	}
	if member.StartingHero {
		return t.Edition.HeroIdentity, true
	}
	if id := member.CampaignHireID(); id != 0 {
		return id, true
	}
	if _, ok := t.Edition.PartyIdentity(member.CompanionNPC); ok && !member.Hired() {
		return member.CompanionNPC, true
	}
	return 0, false
}

// identityRow is the Humans row of the identity row a companion was built
// from.
func identityRow(member PartyMember, t *Table) (int, bool) {
	id, ok := PartyIdentity(member, t)
	if !ok || member.StartingHero {
		return 0, false
	}
	p, _ := t.Edition.PartyIdentity(id)
	row := data.FindHumanByServerID(t.Humans, p.ServerID)
	return row, row > 0 && row <= maxTokenRow
}

// partyDyingTime is the dying time SourceActorSeed reads for the binding the
// SAV writer gives this member, so a fresh member and the same member loaded
// from a SAV take one value. A member with no bound row keeps none.
func partyDyingTime(member, hero PartyMember, typeID int32, t *Table) (int32, error) {
	row := PartyHumanRow(member, hero, t)
	binding := sim.SourceBinding{Class: sim.GeneratedHumanBinding, TokenRow: uint16(row), TypeID: uint16(typeID)}
	if row <= 0 && binding.DefinitionRow() == 0 || t == nil || t.Humans == nil || int(binding.DefinitionRow()) >= t.Humans.Len() {
		return 0, nil
	}
	seed, err := SourceActorSeed(binding, t)
	if err != nil {
		return 0, err
	}
	return seed.DyingTime, nil
}
