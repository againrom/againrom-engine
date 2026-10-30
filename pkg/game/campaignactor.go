package game

import (
	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

var keyedCampaignActor = mapload.CampaignActorProfile{
	ServerBase: 10000, ServerStride: 10, StageDivisor: 10, StageBias: -2,
	Temporary: true, RowFigure: true, RowRotation: true, RowDefinition: true,
}

func (t *townScreen) campaignActor(npc int) (mapload.PartyMember, bool) {
	for _, p := range t.sess.Carried {
		if p.CompanionNPC == npc || p.CampaignHireID() == npc {
			return p, true
		}
	}
	c := t.state()
	for _, o := range c.hires() {
		if o.npc == npc {
			return mapload.CampaignActor(t.in.Table, mapload.CampaignActorSpec{
				Profile: keyedCampaignActor, Identity: npc, Stage: int(c.bank[768]), Hire: true,
			}, nil)
		}
	}
	if _, ok := tableEdition(t.in.Table).PartyIdentity(npc); ok {
		return t.in.townInstall().character(mapload.CharacterSpec{NPC: int32(npc)}, t.in.localizedNPCName(npc-1, ""), t.sess.Carried)
	}
	return mapload.CharacterMember(t.in.Table, mapload.CharacterSpec{Row: data.FindHumanByServerID(t.in.Table.Humans, int32(npc))}, nil)
}
