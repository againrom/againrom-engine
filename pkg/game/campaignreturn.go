package game

import (
	"fmt"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/herokind"
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
	stock           *random.Stream
	stockMultiplier int32
}

// townInstall is the install's definitions for a town operation.
func (in *InstallResources) townInstall() townInstall {
	return townInstall{table: in.Table, bodies: in.Bodies, campaign: in.Campaign.Value(), npcName: in.localizedNPCName}
}

func (f *FrontEnd) townInstall() townInstall {
	return f.runtime.townInstall(&f.InstallResources)
}

// townInstall is the install's definitions for a town operation with the
// launch's shop stock multiplier.
func (s *runtimeSwitches) townInstall(in *InstallResources) townInstall {
	ti := in.townInstall()
	if s != nil {
		ti.stockMultiplier = s.shopStockMultiplier
	}
	return ti
}

// localizedNPC is the install's name for NPC index, or fallback when the
// install carries none.
func (in townInstall) localizedNPC(index int, fallback string) string {
	if in.npcName == nil {
		return fallback
	}
	return in.npcName(index, fallback)
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
	s.Shop = s.Town.newShop(in)
	if s.Town.restoredCampaign() {
		s.addChapterCompanions(in, s.Town.Chapter())
	}
}

// newShop is the town's one stock builder: the chapter's ceiling and the
// install's multiplier, drawn from in.stock or, when it is nil, from the
// seed of the chapter and the missions finished.
func (t *Town) newShop(in townInstall) *Shop {
	ceiling := int32(t.ChapterData().ShopMax)
	shop := NewShop(ceiling)
	shop.stockMultiplier = in.stockMultiplier
	ids := []int{t.townID()}
	for _, entry := range t.camp.Towns.chapterTowns() {
		ids = append(ids, entry.ID)
	}
	shop.location = t.townID()
	shop.deriveRecords(ids)
	if shop.profile().fillOnArrival {
		shop.GenerateWith(in.table, shopSeed(t.Chapter(), t.finishedCount(), ceiling), in.stock)
	}
	return shop
}

// townCompanion is the member the town of chapter grants as companion npc to
// party: the campaign's definition row for that party, named and drawn as the
// install names and draws it.
func townCompanion(in townInstall, chapter, npc int, party []mapload.PartyMember) (mapload.PartyMember, bool) {
	return in.character(mapload.CharacterSpec{NPC: int32(npc), Mission: chapter}, "", party)
}

// character is the member s names, built by the one builder against party,
// named and drawn as the install names and draws it: an npc record by the
// town's companion line, any other member by name when it is set.
func (in townInstall) character(s mapload.CharacterSpec, name string, party []mapload.PartyMember) (mapload.PartyMember, bool) {
	member, ok := mapload.CharacterMember(in.table, s, party)
	if !ok {
		return mapload.PartyMember{}, false
	}
	_, controlled := tableEdition(in.table).PartyIdentity(int(s.NPC))
	// A record that is one definition row is one named person: the town's
	// companion line names only the composed archetypes.
	_, direct := in.table.NPC.ServerID(s.NPC)
	if s.NPC != 0 && s.Row == 0 && !controlled && !(direct && name != "") {
		member.Name = in.companionName(member)
	} else if name != "" {
		member.Name = name
	}
	if look := heroAppearance(nil, 0, member, in.bodies, heroCreated); look.Matched {
		member.Body, member.BodyDir, member.Class = string(look.Body), look.Dir, look.Class
	}
	return member, true
}

// companionName is the npcnames line the town names a companion by: the line
// of the composed archetype (female +1, mage +2) after line 21.
func (in townInstall) companionName(member mapload.PartyMember) string {
	selector := herokind.Of(member.Mage, data.FigureDir(member.FigureDir).Female()).Index()
	fallback := member.Name
	for _, base := range data.ChargenBaseNames() {
		if member.Name == base || strings.HasPrefix(member.Name, base+"_") {
			fallback = strings.TrimPrefix(base, "PC_")
			break
		}
	}
	return in.localizedNPC(20+selector, fallback)
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
	if !arrival.carried {
		if refused := s.carryMissionHome(in, n, party, w, ids, roster); refused != "" {
			return -1, refused
		}
	} else if address, ok := MissionMap(n); ok && s.Town != nil {
		s.Town.lastMap = originalMapName(address)
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
		_ = seatTownCarriers(townCarry, carrierTown{t: s.Town, w: w})
		if s.live != nil && s.live.world == w && s.live.mission != nil && s.live.mission.state != nil {
			s.Town.playerColor = uint8(missionPlayerColor(s.live.mission.state.Map, sim.SelfSlot))
		}
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
	a := campaign.after(n, s.Town.currentMain(), s.Town.restoredCampaign(), campaignExit{
		successor: successor, stage: s.Town.campaignStage(), done: s.Town.Done(n), open: s.Town.Open(),
	})
	if a.screen == winEnding {
		s.endLive()
	}
	return a.screen
}

// leaveLive is what leaving the live mission n without completing it does to
// the campaign. A mission restored from a SAV has no town to leave: its party,
// as it stands, goes home through the carry a won mission uses, with no win,
// payment or fame recorded, and a refusal of that carry is returned. The
// quick-spell bindings go back to those the mission was opened with.
func (s *CampaignSession) leaveLive(in townInstall, n int, ms *Mission, restart bool) (refused string) {
	if s.live != nil && s.live.mission != nil && s.live.mission.resumed {
		if refused := s.carryMissionHome(in, n, ms.Party, ms.World, ms.Start.IDs, ms.Start.Roster); refused != "" {
			return refused
		}
		s.retainReturnCity(in, n, ms.World)
	}
	if s.live != nil && s.live.mission != nil {
		s.quickSpells = s.live.mission.entryQuickSpells
	}
	if err := s.returnToEntry(in.table, n, restart); err != nil {
		return err.Error()
	}
	return ""
}

// returnToEntry is the one return of mission n, left without completion, to
// the state at its entry, in every game. The entry is data the session's
// item-home row keeps: where the row's game departs its town onto a map of
// places (partyItemProfile.departs), the departure record undoes its places
// and answers whether the party is home; with no departure the party never
// left its town and is home unless the mission restarts. At home the party
// and the bag are those the departure kept, else the arrival walk's.
func (s *CampaignSession) returnToEntry(table *mapload.Table, n int, restart bool) error {
	h := s.partyItems(table)
	home := !restart
	if c := h.departure(); c != nil {
		home = c.returnFrom(n, restart)
	}
	if !home {
		return nil
	}
	return h.returnEntry()
}
