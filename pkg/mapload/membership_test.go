package mapload_test

import (
	"fmt"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
)

func TestPartyMembershipKindsAndIDs(t *testing.T) {
	for typ := 1; typ <= 15; typ++ {
		p := mapload.PartyMember{MercenaryType: uint8(typ), Temporary: true}
		m := p.Membership()
		if !m.Matches(mapload.TavernMember, typ) || !m.Hired() || !m.Temporary || m.Units != (typ <= 2) ||
			p.UnitMember() != m.Units || m.ID(2) != fmt.Sprintf("mercenary:%d:3", typ) {
			t.Fatalf("tavern %d: %+v", typ, m)
		}
	}
	for id := 1; id <= 20; id++ {
		for _, units := range []bool{false, true} {
			p := mapload.PartyMember{CampaignHire: uint8(id), CampaignUnit: units, Temporary: true}
			m := p.Membership()
			if !m.Matches(mapload.CampaignMember, id) || !p.Hired() || p.CampaignHireID() != id ||
				p.UnitMember() != units || !p.ValidMembership() || m.ID(0) != fmt.Sprintf("hire:%d", id) {
				t.Fatalf("campaign %d/%v: %+v", id, units, m)
			}
		}
	}
	for _, p := range []mapload.PartyMember{
		{CampaignHire: 21, Temporary: true}, {CampaignHire: 1}, {CampaignUnit: true},
		{CampaignHire: 1, Temporary: true, PlayerCharacter: true},
		{CampaignHire: 1, MercenaryType: 3, Temporary: true},
	} {
		if p.ValidMembership() {
			t.Fatalf("invalid membership accepted: %+v", p)
		}
	}
	for _, id := range []int{21, 22, 23, 24, 25} {
		p := mapload.PartyMember{CompanionNPC: id, PlayerCharacter: true}
		m := p.Membership()
		if !m.Matches(mapload.CompanionMember, id) || p.Hired() || p.CompanionID() != id ||
			m.ID(0) != fmt.Sprintf("npc:%d", id) {
			t.Fatalf("companion %d: %+v", id, m)
		}
	}
}

func TestPartyIdentityCodecAcceptsOnlyItsCompanionRow(t *testing.T) {
	e := base.GameROM2.Edition()
	humans := defCollection{{}, {name: "fighter", params: make([]int32, 25)}, {name: "mage", params: make([]int32, 25)}}
	humans[1].params[24], humans[2].params[24] = 10220, 10230
	table := &mapload.Table{Edition: &e, Humans: humans}
	for i, id := range []int{22, 23} {
		p := mapload.PartyMember{CompanionNPC: id, PlayerCharacter: true}
		word, ok := mapload.PartyIdentity(p, table)
		if !ok || word != id {
			t.Fatalf("encode %d: %d/%v", id, word, ok)
		}
		for _, high := range []uint32{0, 0x12340000} {
			got, ok := mapload.DecodePartyIdentity(uint32(word)|high, uint16(i+1), false, true, table)
			if !ok || got != id {
				t.Fatalf("decode %d: %d/%v", id, got, ok)
			}
		}
		for _, c := range []struct {
			row         uint16
			hero, human bool
		}{
			{uint16(2 - i), false, true}, {uint16(i + 1), true, true}, {uint16(i + 1), false, false},
		} {
			if _, ok := mapload.DecodePartyIdentity(uint32(word), c.row, c.hero, c.human, table); ok {
				t.Fatalf("invalid record accepted: identity %d record %+v", id, c)
			}
		}
		if _, ok := mapload.PartyIdentity(p, nil); ok {
			t.Fatal("role profile encoded an identity")
		}
	}
	if id, ok := mapload.PartyIdentity(mapload.PartyMember{StartingHero: true}, table); !ok || id != 21 {
		t.Fatalf("hero identity: %d/%v", id, ok)
	}
	for _, id := range []uint32{0, 20, 21, 24, 65535} {
		if _, ok := mapload.DecodePartyIdentity(id, 1, false, true, table); ok {
			t.Fatalf("unknown companion %d", id)
		}
	}
}
