package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// TestGuestJoinKeepsHerWornWeaponAsHeld: the town's AddHero grant of a guest
// who ended her mission takes her Weapons as held, a missing Spell included
// (ITEM-212). The control: the join build of a member constructed from his
// row would give the same staff its Spell under the same table.
func TestGuestJoinKeepsHerWornWeaponAsHeld(t *testing.T) {
	table, _ := staffTooltipTable(t, 1, "Fire Arrow", 10, 5, 3, 5, 0)
	staff := sim.ItemInstance{Code: 0x0101, Kind: 2, WeightPresent: true, Weight: 3,
		SourceEquipment: sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: 1},
		Effects:         []sim.ItemEffect{{Kind: 41, Operand: 1 | 10<<16}}}
	constructed := []sim.ItemInstance{staff.Clone()}
	joinedWornSpells(constructed, table)
	if !constructed[0].SourceEquipment.Spell.Present {
		t.Fatal("control: the table builds no Spell for the staff")
	}
	guest := mapload.PartyMember{ID: "npc:22", Name: "Guest", CompanionNPC: 22}
	guest.Worn[0], guest.WornItems[0] = staff.Code, staff
	g := &guestMission{mission: guestDecl(), member: &guest}
	s := CampaignSession{Carried: []mapload.PartyMember{{ID: "hero", StartingHero: true}}, Town: &Town{guest: g}}
	if !s.carryTownCompanion(townInstall{campaign: guestCampaign(), table: table}, 30, 22) || len(s.Carried) != 2 {
		t.Fatalf("the grant took %+v", s.Carried)
	}
	if got := s.Carried[1].WornItems[0].SourceEquipment.Spell; got.Present {
		t.Fatalf("the joined guest's staff holds Spell %+v, want none", got)
	}
}
