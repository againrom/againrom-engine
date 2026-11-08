package game

import (
	"againrom/pkg/mapload"
)

func (t *townScreen) campaignActor(npc int) (mapload.PartyMember, bool) {
	p := innActor(bankInnProfile, npc, innActorInput{party: t.sess.Carried, campaign: t.state(), install: t.in})
	return p.member, p.known
}
