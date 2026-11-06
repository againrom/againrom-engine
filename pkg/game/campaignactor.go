package game

import (
	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

func (t *townScreen) campaignActor(npc int) (mapload.PartyMember, bool) {
	for _, p := range t.sess.Carried {
		if p.CompanionNPC == npc || p.CampaignHireID() == npc {
			return p, true
		}
	}
	c := t.state()
	for _, o := range c.hires() {
		if o.npc == npc {
			s := c.hireState()
			s.table = t.in.Table
			members, ok := hireMembers(&bankHires, s, npc, 1)
			if !ok {
				return mapload.PartyMember{}, false
			}
			return members[0], true
		}
	}
	if _, ok := tableEdition(t.in.Table).PartyIdentity(npc); ok {
		return t.in.townInstall().character(mapload.CharacterSpec{NPC: int32(npc)}, t.in.localizedNPCName(npc-1, ""), t.sess.Carried)
	}
	return mapload.CharacterMember(t.in.Table, mapload.CharacterSpec{Row: data.FindHumanByServerID(t.in.Table.Humans, int32(npc))}, nil)
}
