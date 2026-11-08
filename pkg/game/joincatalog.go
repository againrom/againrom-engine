package game

import (
	"slices"
	"sort"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
)

type joinCatalog []base.PartyIdentity

// addChapterCompanions grants the companions a chapter's town hands out.
func (f *FrontEnd) addChapterCompanions(chapter int) {
	if f == nil {
		return
	}
	f.CampaignSession.addChapterCompanions(f.townInstall(), chapter)
}

// joins layers the campaign's grants and mod talk conditions over profile rows.
func (in townInstall) joins() joinCatalog {
	rows := append(joinCatalog(nil), tableEdition(in.table).JoinRows()...)
	chapters := make([]int, 0, len(in.campaign.Chapters))
	for chapter := range in.campaign.Chapters {
		chapters = append(chapters, chapter)
	}
	sort.Ints(chapters)
	for _, chapter := range chapters {
		for _, id := range in.campaign.Chapters[chapter].AddHero {
			rows = append(rows, base.PartyIdentity{Identity: id, Chapter: chapter, Trigger: base.JoinChapter, TownGrant: true})
		}
	}
	if in.table != nil {
		for _, j := range in.table.Mods.Companions.Joins {
			rows = slices.DeleteFunc(rows, func(r base.PartyIdentity) bool {
				return r.TownGrant && r.Chapter == j.Chapter && r.Identity == j.Companion
			})
			rows = append(rows, base.PartyIdentity{Identity: j.Companion, Chapter: j.Chapter,
				Trigger: base.JoinTalkOpen, TalkNPC: j.Talk, Building: j.Building, TownGrant: true})
		}
	}
	return rows
}

// byServerID maps the server id of each Humans row a companion is built from
// to the identity that companion carries.
func (c joinCatalog) byServerID() map[int32]int32 {
	out := map[int32]int32{}
	for _, p := range c {
		if p.ServerID != 0 {
			out[p.ServerID] = int32(p.Identity)
		}
	}
	return out
}

func (c joinCatalog) townRow(chapter, id int) (base.PartyIdentity, bool) {
	for _, row := range c {
		if row.TownGrant && row.Chapter == chapter && row.Identity == id {
			return row, true
		}
	}
	for _, row := range c {
		if row.TownGrant && row.Identity == id {
			row.Chapter, row.Trigger = chapter, base.JoinChapter
			return row, true
		}
	}
	return base.PartyIdentity{}, false
}

func (in townInstall) holdsCompanion(chapter, npc int) bool {
	row, ok := in.joins().townRow(chapter, npc)
	return ok && row.Trigger == base.JoinTalkOpen && row.Chapter == chapter
}

func (s *CampaignSession) joinMember(in townInstall, row base.PartyIdentity) bool {
	for _, p := range s.Carried {
		if p.Membership().Matches(mapload.CompanionMember, row.Identity) {
			return false
		}
	}
	if len(s.Carried) == 0 {
		return false
	}
	var member mapload.PartyMember
	var ok bool
	if row.TownGrant {
		member, ok = s.Town.takeGuestJoining(row.Chapter, row.Identity)
	}
	if !ok {
		name := ""
		if !row.TownGrant {
			name = in.localizedNPC(row.Identity-1, "")
		}
		member, ok = in.character(mapload.CharacterSpec{NPC: int32(row.Identity), Mission: row.Chapter}, name, s.Carried)
		if ok {
			joinedWornSpells(member.WornItems[:], in.table)
		}
	}
	if !ok {
		return false
	}
	s.Carried = mapload.OwnParty(append(s.Carried, member))
	return true
}

func (s *CampaignSession) addChapterCompanions(in townInstall, chapter int) {
	if s == nil || in.table == nil {
		return
	}
	catalog := in.joins()
	grants := append([]int(nil), in.campaign.Chapters[chapter].AddHero...)
	if s.Town != nil {
		grants = s.Town.takeAddHeroesExcept(chapter, func(id int) bool {
			row, ok := catalog.townRow(chapter, id)
			return ok && row.Trigger == base.JoinTalkOpen
		})
	}
	added := false
	for _, id := range grants {
		if row, ok := catalog.townRow(chapter, id); ok && s.joinMember(in, row) {
			added = true
		}
	}
	s.settleCompanionArrival(in, added)
}

func (s *CampaignSession) joinChapterMember(in townInstall, chapter, id int) bool {
	row, ok := in.joins().townRow(chapter, id)
	return ok && s.joinMember(in, row)
}

func (s *CampaignSession) addPartyIdentity(in townInstall, id int) bool {
	row, ok := mapload.CompanionIdentity(id, in.table)
	if !ok || in.table == nil || in.table.Humans == nil {
		return false
	}
	added := s.joinMember(in, row)
	s.settleCompanionArrival(in, added)
	return added
}

func (s *CampaignSession) joinAtTalk(in townInstall, talk int) bool {
	if s == nil || in.table == nil || s.Town == nil {
		return false
	}
	added := false
	for _, row := range in.joins() {
		if row.Trigger == base.JoinTalkOpen && row.Chapter == s.Town.Chapter() &&
			row.Building == mod.BuildingTavern && row.TalkNPC == talk && s.Town.takeAddHero(row.Chapter, row.Identity) {
			if s.joinMember(in, row) {
				added = true
			}
		}
	}
	s.settleCompanionArrival(in, added)
	return added
}

func (s *CampaignSession) addTalkIdentities(in townInstall, npc, topic int) {
	s.addTriggerIdentities(in, base.JoinTalkEnd, func(rowNPC, rowTopic int) bool {
		return rowNPC == npc && rowTopic == topic
	})
}

func (s *CampaignSession) addOfferIdentities(in townInstall, trigger base.JoinTrigger, offers []innOffer) {
	s.addTriggerIdentities(in, trigger, func(rowNPC, rowTopic int) bool {
		for _, offer := range offers {
			if rowNPC == offer.npc && rowTopic == offer.topic {
				return true
			}
		}
		return false
	})
}

func (s *CampaignSession) addTriggerIdentities(in townInstall, trigger base.JoinTrigger, match func(npc, topic int) bool) {
	for _, row := range in.joins() {
		if row.Trigger == trigger && row.TalkNPC != 0 && match(row.TalkNPC, row.TalkTopic) {
			s.addPartyIdentity(in, row.Identity)
		}
	}
}

func (s *CampaignSession) addLeaveIdentities(in townInstall, bank *[innBankSlots]int32) {
	for _, row := range in.joins() {
		if row.Trigger == base.JoinLeave && row.BankSlot > 0 && row.BankSlot < len(bank) && bank[row.BankSlot] != 0 {
			s.addPartyIdentity(in, int(bank[row.BankSlot]))
		}
	}
}
