package game

import (
	"fmt"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/random"
	"againrom/pkg/sim"
)

// The operations below are the campaign half of a mission's return and of a
// town arrival. Each is a method of CampaignSession over the session's own
// state and the plain install values it is handed, so none needs a screen, a
// sound device or a FrontEnd. FrontEnd keeps the screen switch and calls these
// by name.

// townInstall is the install's definitions a town operation reads: the
// placeable table, the body list, the campaign and the localized NPC names. A session operation takes this value and
// never the install it came from.
type townInstall struct {
	table    *mapload.Table
	bodies   data.BodyList
	campaign Campaign
	npcName  func(index int, fallback string) string
	// stock is the shop-stock stream; nil draws on the campaign seed.
	stock *random.Stream
}

// townInstall is the install's definitions for a town operation.
func (in *InstallResources) townInstall() townInstall {
	return townInstall{table: in.Table, bodies: in.Bodies, campaign: in.Campaign.Value(), npcName: in.localizedNPCName}
}

// localizedNPC is the install's name for NPC index, or fallback when the
// install carries none.
func (in townInstall) localizedNPC(index int, fallback string) string {
	if in.npcName == nil {
		return fallback
	}
	return in.npcName(index, fallback)
}

// holdsCompanion reports that a mod keeps the town's grant of npc pending
// until a conversation opens. The pending grant is the AddHero array the save
// already writes, so the held state needs no field of its own.
func (in townInstall) holdsCompanion(chapter, npc int) bool {
	if in.table == nil {
		return false
	}
	_, ok := in.table.Mods.Companions.Find(chapter, npc)
	return ok
}

// arriveInTown latches the town open and stocks the merchant, which is the
// original's own SetCap-then-Generate pair (SHOP-TOWN-022). It writes campaign
// state only; the town screen's own reset on an ordinary return is the caller's.
//
// IT IS THE ONE PLACE THE SHOP IS GENERATED. Command 0x3f has two senders in the
// original — the homecoming at the end of a mission, and the load of a save that
// is not in a battle — and both do exactly SetCap and Generate. This method is
// what those two senders become here, so the assortment is re-rolled on every
// homecoming and after every load into the town, and never on walking into the
// shop room.
//
// THE CEILING IS THE LIVE CHAPTER'S. SHOP-TOWN-022 establishes the value sent is
// the NEXT mission's, because the original advances its mission record before it
// packs the command. Town.Chapter() is recomputed as the lowest offered main
// mission not yet won, so by the time this runs it already answers the next
// mission — the two agree without this method having to advance anything.
func (s *CampaignSession) arriveInTown(in townInstall) {
	s.Town.Arrive()
	ceiling := int32(s.Town.ChapterData().ShopMax)
	s.Shop = NewShop(ceiling)
	s.Shop.GenerateWith(in.table, shopSeed(s.Town.Chapter(), s.Town.finishedCount(), ceiling), in.stock)
	if s.Town.restoredCampaign() {
		s.addChapterCompanions(in, s.Town.Chapter())
	}
}

func (s *CampaignSession) addChapterCompanions(in townInstall, chapter int) {
	if s == nil || in.table == nil {
		return
	}
	grants := append([]int(nil), in.campaign.Chapters[chapter].AddHero...)
	if s.Town != nil {
		grants = s.Town.takeAddHeroesExcept(chapter, func(npc int) bool { return in.holdsCompanion(chapter, npc) })
	}
	added := false
	for _, npc := range grants {
		if s.carryTownCompanion(in, chapter, npc) {
			added = true
		}
	}
	s.settleCompanionArrival(in, added)
}

// carryTownCompanion appends the companion a town grants to the carried party
// and reports whether it did. Only the grant of npc 22 is a town companion;
// a companion already carried is not added twice.
func (s *CampaignSession) carryTownCompanion(in townInstall, chapter, npc int) bool {
	if npc != townGrantedCompanion {
		return false
	}
	for _, p := range s.Carried {
		if p.CompanionNPC == npc {
			return false
		}
	}
	if len(s.Carried) == 0 {
		return false
	}
	member, ok := mapload.CampaignNPCMember(in.table, int32(npc), chapter, s.Carried)
	if !ok {
		return false
	}
	selector := 0
	if data.FigureDir(member.FigureDir).Female() {
		selector++
	}
	if member.Mage {
		selector += 2
	}
	fallback := member.Name
	for _, base := range data.ChargenBaseNames() {
		if member.Name == base || strings.HasPrefix(member.Name, base+"_") {
			fallback = strings.TrimPrefix(base, "PC_")
			break
		}
	}
	member.Name = in.localizedNPC(20+selector, fallback)
	body, dir, class, matched := data.HeroAppearance(in.bodies, mapload.EquipmentFromParty(member), member.Mage, false)
	if matched {
		member.Body, member.BodyDir, member.Class = string(body), dir, class
	}
	s.Carried = mapload.OwnParty(append(s.Carried, member))
	return true
}

// settleCompanionArrival gives a party grown by a town companion the city
// state every carried member has.
func (s *CampaignSession) settleCompanionArrival(in townInstall, added bool) {
	// The mission-return projection is built before the chapter grant is
	// appended. Rebuild the detached current city topology once so every
	// carried member has a party root before shop or school mutations run.
	if added && s.Town != nil && s.Town.cityObjects != nil {
		if projection, err := newCityObjectProjection(s.Town.cityObjects, s.Carried, in.table); err == nil {
			s.Town.cityObjects = projection.graph
		}
	}
	if added {
		s.Town.settleCityGroups(s.Carried)
	}
}

// finishMission is the campaign's whole answer to a won mission: the party and
// purse carried home, the win recorded, and the return city kept for the town
// SAVE when the mission leaves no successor to open. The town screen and the
// companion grant are reached only through arrival.
func (s *CampaignSession) finishMission(in townInstall, n int, party []mapload.PartyMember,
	w *sim.World, ids []sim.EntityID, roster map[sim.EntityID]mapload.PartyMember,
	arrival townArrival) (next int, message string) {
	defer func() {
		if next == 0 {
			s.retainReturnCity(in, n, w)
		}
	}()
	if refused := s.carryMissionHome(in, n, party, w, ids, roster); refused != "" {
		return -1, refused
	}
	return s.recordWin(in.campaign, n, party, w, ids, roster, arrival)
}

// carryMissionHome is the party-and-purse half of a mission's return: the
// survivors become the town's party and city objects, the hired pools take the
// tally, and the purse becomes the town balance. It records no win. A non-empty
// result is the refusal.
func (s *CampaignSession) carryMissionHome(in townInstall, n int, party []mapload.PartyMember,
	w *sim.World, ids []sim.EntityID, roster map[sim.EntityID]mapload.PartyMember) string {
	// MERC-DEATH-006 tallies the living mercenaries before the cull resets a
	// fallen body, so a hired body the boundary raises still leaves the pool.
	mercenaries := liveMercenaries(party, w, ids)
	// A failed source inverse must not discard its live effect lifetime or
	// publish town rewards. Keep the complete mission for its lossless SAVE.
	if w != nil {
		if err := w.NormalizeMissionSurvivors(sim.SelfSlot); err != nil {
			return fmt.Sprintf("mission %d return refused: %v", n, err)
		}
	}
	carryParty, carryIDs := party, ids
	if w != nil && s.live != nil && s.live.world == w && s.live.mission != nil {
		carryParty, carryIDs = withoutDeparted(party, ids, s.live.mission.departed)
	}
	carryParty, carryIDs = withoutFallenBodies(carryParty, carryIDs, w)
	carried, cityObjects, err := currentMissionCityParty(w, s.missionObjectSource(w), carryParty, carryIDs, roster, in.table)
	if err != nil {
		return fmt.Sprintf("mission %d return: %v", n, err)
	}

	// THE TAVERN ECONOMY'S END-OF-MISSION HALF RUNS BEFORE THE CARRY, and the
	// order is the original's own (MERC-DEATH-006): the live units are tallied
	// per type first, the pool is merged from that tally and the hire flags are
	// cleared, and only then does the cull drop every mercenary. A tally taken
	// after the cull would count the survivors of its own cull, which is zero
	// for every type.
	//
	// IT NEEDS A WORLD. With none there is no live population to count, so the
	// merge is skipped rather than run against an all-zero tally that would
	// wipe every hired pool. Every production boundary has one -- continuity
	// returns early on a nil world, AdvanceLine builds one, and the resume path
	// hands the live one over -- and the callers that pass nil are the tools
	// that open a town without playing a mission.
	if w != nil && s.Town != nil {
		s.Town.mercenaryBoundary(mercenaries)
	}
	// PARTY-ENDCULL-026 expires attached effects before restoring the pools.
	// Keep the real actor mapping until current-source return admission ends.
	packs := captureCityItemGraphs(party, w, ids)
	worn := captureCityEquipmentGraphs(party, w, ids)
	s.Carried = carried
	if s.Town != nil && w != nil {
		s.Town.cityObjects = cityObjects
	}
	s.Town.settleCityGroups(s.Carried)
	s.captureMissionReturn(n, party, w, ids, w != nil, packs, worn)
	// Mission loot and the purse carried in become the town balance before the
	// mission's payment is applied. Both are the same participant-owned u32.
	if w != nil && s.Town != nil {
		s.Town.gold = int(w.Purse(sim.SelfSlot))
		s.Town.carryKnowledge(w)
	}
	if address, ok := MissionMap(n); ok && s.Town != nil {
		s.Town.lastMap = originalMapName(address)
	}
	return ""
}

// retainReturnCity keeps the city a mission left for the town SAVE after the
// party is home, when the mission came from a current-source city document.
func (s *CampaignSession) retainReturnCity(in townInstall, n int, w *sim.World) {
	if s.Town != nil && s.Town.Open() && s.Town.progress != nil && s.originalCity == nil &&
		w != nil && s.live != nil && s.live.world == w && s.liveMission == n &&
		s.live.mission != nil && s.live.mission.state != nil && s.live.mission.state.savedDocument != nil &&
		!generatedCurrentMissionDocument(s.live.mission.state.savedDocument) {
		if err := s.retainMissionCity(in, n, w); err != nil {
			s.originalCity = &originalCitySaveState{unavailable: fmt.Errorf("mission return city: %w", err)}
		}
	}
}

// winRoute is where the campaign sends the player after a won mission.
type winRoute int

const (
	// winList leaves the driver's own destination, the map list, in place.
	winList winRoute = iota
	// winStay keeps the banner open: the win could not be finished.
	winStay
	// winEnding is the end of the campaign.
	winEnding
	// winMission opens the successor mission directly.
	winMission
	// winTown travels to the town, which is standing already.
	winTown
)

// routeAfterWin answers where winning mission n leads, given successor, the
// answer finishMission returned: negative for a win that could not be
// finished, positive for a mission to open and zero for none.
//
// The end of the campaign forgets the live map. A win with no successor to
// open ends in the town once the campaign has reached one. The test is the
// town's own latch and not this mission's number, so the arm that opened the
// town and the arm that goes there are the same fact read twice. No opener
// crosses: the town is a screen the front end already installed, standing
// whether a mission is running or not.
func (s *CampaignSession) routeAfterWin(campaign Campaign, n, successor int) winRoute {
	if successor < 0 {
		return winStay
	}
	if campaign.TerminalMission(n) && s.Town.Done(n) {
		s.endLive()
		return winEnding
	}
	if successor > 0 {
		return winMission
	}
	if s.Town.Open() {
		return winTown
	}
	return winList
}

// leaveLive is what leaving the live mission n without completing it does to
// the campaign. A mission restored from a SAV has no town to leave: its party,
// as it stands, goes home through the carry a won mission uses, with no win,
// payment or fame recorded, and a refusal of that carry is returned. The
// quick-spell bindings go back to those the mission was opened with.
func (s *CampaignSession) leaveLive(in townInstall, n int, ms *Mission) (refused string) {
	if s.live != nil && s.live.mission != nil && s.live.mission.resumed {
		if refused := s.carryMissionHome(in, n, ms.Party, ms.World, ms.Start.IDs, ms.Start.Roster); refused != "" {
			return refused
		}
		s.retainReturnCity(in, n, ms.World)
	}
	if s.live != nil && s.live.mission != nil {
		s.quickSpells = s.live.mission.entryQuickSpells
	}
	return ""
}
