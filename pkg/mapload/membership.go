package mapload

import (
	"fmt"

	"againrom/pkg/base"
	"againrom/pkg/data"
)

type MembershipKind uint8

const (
	FullMember MembershipKind = iota
	TavernMember
	CampaignMember
	CompanionMember
)

// Membership is a party member's kind, identity, collection and continuity.
// PartyMember's legacy fields are the wire representation of this value.
type Membership struct {
	Kind      MembershipKind
	Identity  int
	Units     bool
	Temporary bool
}

func (p PartyMember) Membership() Membership {
	m := Membership{Temporary: p.Temporary}
	switch {
	case p.MercenaryType != 0:
		m.Kind, m.Identity = TavernMember, int(p.MercenaryType)
		m.Units = TableEdition(nil).TavernUnit(m.Identity)
	case p.Temporary && p.CampaignHire >= 1 && p.CampaignHire <= 20:
		m.Kind, m.Identity, m.Units = CampaignMember, int(p.CampaignHire), p.CampaignUnit
	case p.Temporary && p.CampaignHire == 0 && p.CampaignUnit && p.DefinitionRow != 0:
		m.Kind, m.Units = CampaignMember, true
	case p.CompanionNPC != 0:
		m.Kind, m.Identity = CompanionMember, p.CompanionNPC
	}
	return m
}

func (m Membership) Is(kind MembershipKind) bool { return m.Kind == kind }
func (m Membership) Matches(kind MembershipKind, id int) bool {
	return m.Kind == kind && m.Identity == id
}
func (m Membership) Hired() bool { return m.Is(TavernMember) || m.Is(CampaignMember) }

// Hired recognizes tavern and campaign hires; script companions are full heroes.
func (p PartyMember) Hired() bool      { return p.Membership().Hired() }
func (p PartyMember) UnitMember() bool { return p.Membership().Units }
func (p PartyMember) TavernType() uint8 {
	if m := p.Membership(); m.Is(TavernMember) {
		return uint8(m.Identity)
	}
	return 0
}
func (p PartyMember) CampaignHireID() int {
	if m := p.Membership(); m.Is(CampaignMember) {
		return m.Identity
	}
	return 0
}
func (p PartyMember) CompanionID() int {
	if m := p.Membership(); m.Is(CompanionMember) {
		return m.Identity
	}
	return 0
}

func (p PartyMember) ValidMembership() bool {
	return !(p.CampaignHire > 20 || p.CampaignUnit && p.CampaignHire == 0 ||
		p.CampaignHire != 0 && (!p.Temporary || p.MercenaryType != 0 || p.PlayerCharacter))
}

// ID builds the stable identity for a membership; index orders a tavern squad.
func (m Membership) ID(index int) string {
	switch m.Kind {
	case CompanionMember:
		return fmt.Sprintf("npc:%d", m.Identity)
	case TavernMember:
		return fmt.Sprintf("mercenary:%d:%d", m.Identity, index+1)
	case CampaignMember:
		return fmt.Sprintf("hire:%d", m.Identity)
	}
	return ""
}

func inferredPartyID(p PartyMember, index int) string {
	if p.StartingHero {
		return "hero"
	}
	if m := p.Membership(); m.Is(CompanionMember) || m.Is(TavernMember) {
		return m.ID(index)
	}
	if p.PlayerCharacter && p.Name != "" {
		return "player:" + stablePartyName(p.Name)
	}
	return fmt.Sprintf("temporary:%d", index+1)
}

// PartyIdentity encodes the identity word using the profile's identity rows.
func PartyIdentity(p PartyMember, t *Table) (int, bool) {
	source, e := heroBand(t)
	if !source.identities() {
		return 0, false
	}
	if p.StartingHero {
		return e.HeroIdentity, true
	}
	m := p.Membership()
	if m.Is(CampaignMember) {
		return m.Identity, true
	}
	if _, ok := CompanionIdentity(m.Identity, t); m.Is(CompanionMember) && ok {
		return m.Identity, true
	}
	return 0, false
}

func CompanionIdentity(id int, t *Table) (base.PartyIdentity, bool) {
	source, e := heroBand(t)
	p, ok := e.PartyIdentity(id)
	return p, source.identities() && ok && id != e.HeroIdentity
}

// DecodePartyIdentity validates the companion identity word and its Humans row.
func DecodePartyIdentity(word uint32, row uint16, hero, human bool, t *Table) (int, bool) {
	if hero || !human || t == nil || t.Humans == nil {
		return 0, false
	}
	id := int(uint16(word))
	p, ok := CompanionIdentity(id, t)
	if !ok || int(row) != data.FindHumanByServerID(t.Humans, p.ServerID) {
		return 0, false
	}
	return id, true
}
