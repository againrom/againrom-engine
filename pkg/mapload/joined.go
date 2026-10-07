package mapload

import (
	"fmt"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
)

// CampaignNPCMember builds the persistent party template behind one campaign
// NPC record. It uses the same Humans row, item-instance resolver and roster
// constructor as an NPC placement, so a town AddHero and a map-script handover
// cannot disagree about statistics, spellbook, carried items or worn effects.
//
// The returned identity is stable across missions and saves. Its localized
// display name and installed body are presentation facts applied by pkg/game;
// this tier retains the row name and exact numeric payload.
func CampaignNPCMember(t *Table, npc int32, mission int, party []PartyMember) (PartyMember, bool) {
	if t == nil || t.NPC == nil || t.Humans == nil || len(party) == 0 {
		return PartyMember{}, false
	}
	primary := party[0]
	for _, member := range party {
		if member.StartingHero {
			primary = member
			break
		}
	}
	serverID, ok := t.NPC.CampaignServerID(npc, mission, primary.Mage,
		data.FigureDir(primary.FigureDir).Female())
	if !ok || data.FindHumanByServerID(t.Humans, serverID) == data.NotFound {
		return PartyMember{}, false
	}
	// A definition-id placement reaches the same Humans row without depending
	// on the table's mission-local composed lookup. The constructor below then
	// resolves the row's complete item cells through the ordinary placement path.
	u := alm.Unit{DefID: uint32(serverID)}
	member, ok := rosterTemplate(u, t, 0)
	if !ok {
		return PartyMember{}, false
	}
	member.ID = fmt.Sprintf("npc:%d", npc)
	member.CompanionNPC = int(npc)
	member.StartingHero = false
	member.PlayerCharacter = true
	return member, true
}
