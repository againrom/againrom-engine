package game

import (
	"fmt"

	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// The operations below are the campaign half of a game's lifecycle. Each is a
// method of CampaignSession or a function of plain values, so none needs an
// install, a screen or a sound device: starting a campaign, admitting a
// mission, adopting a prepared candidate and activating a live map are decided
// on the session's own state. FrontEnd keeps the transitions that span
// systems and calls these by name.

// newCampaignCandidate is the complete state of a campaign that has just been
// started: a fresh town over the install's campaign, the difficulty the
// player chose, a private copy of the unit bundle, and fame that has begun.
// It reads no session and writes none, so a refused start leaves the running
// campaign untouched.
func newCampaignCandidate(campaign Campaign, units *terrain.UnitSet, level int64) (*restoreCandidate, error) {
	difficulty, err := campaignDifficulty(level)
	if err != nil {
		return nil, err
	}
	return &restoreCandidate{
		town:       NewTown(campaign),
		difficulty: difficulty,
		units:      cloneCandidateUnits(units),
		fame:       SnapshotFame{Known: true},
	}, nil
}

// admitMission decides whether mission n may be entered over town and names
// the map it loads. resuming is true for a load, which carries its own
// mission number and is not held to the restored main progress. The order of
// the refusals is the entry's order: difficulty, progress, mission number.
func admitMission(town *Town, n int, resuming bool, level mapload.Difficulty) (address string, difficulty mapload.Difficulty, err error) {
	difficulty, err = campaignDifficulty(int64(level))
	if err != nil {
		return "", 0, err
	}
	if !resuming && town != nil && town.second != nil && !town.second.canEnter(n) {
		return "", 0, fmt.Errorf("mission %d is not an available campaign destination", n)
	}
	if !resuming && !town.canOpenMission(n) {
		return "", 0, fmt.Errorf("mission %d is below restored main progress %d", n, town.Chapter())
	}
	address, ok := MissionMap(n)
	if !ok {
		return "", 0, fmt.Errorf("mission %d: not a campaign mission number", n)
	}
	return address, difficulty, nil
}

// clear drops everything the previous game left in the session, so the next
// game starts exactly as the first one did. The town is rebuilt over the
// install's campaign rather than zeroed.
func (s *CampaignSession) clear(campaign Campaign) {
	s.detachFameObserver()
	s.fame = SnapshotFame{}
	s.quickSpells = [4]uint32{}
	s.Carried = nil
	s.Difficulty = 0 // Normal, as on a zero-value FrontEnd.
	s.Offered = 0
	s.Town = NewTown(campaign)
	s.originalCity = nil
	s.live, s.liveMission, s.liveParty = nil, 0, nil
	s.Shop = nil
}

// adopt commits a fully prepared candidate as the session's game. It is the
// one write of the campaign state a start or a load makes; every fallible step
// happened before it.
func (s *CampaignSession) adopt(c *restoreCandidate) {
	s.detachFameObserver()
	s.fame = cloneFame(c.fame)
	s.quickSpells = c.quickSpells
	s.Difficulty = c.difficulty
	s.Shop = nil
	s.Town = c.town
	s.Carried = c.carried
	s.Offered = c.offered
	// Only a fully validated candidate may supply provenance. Old native and
	// mission candidates supply nil, never the previous session's document.
	s.originalCity = c.originalCity
}

// endLive forgets the live map and returns it, so the caller can release what
// the map held. The session keeps no reference to a map that was ended.
func (s *CampaignSession) endLive() *mapWorld {
	outgoing := s.live
	s.live, s.liveMission, s.liveParty = nil, 0, nil
	return outgoing
}

// activateLive makes mw the live map of mission n and returns the map it
// replaced, or nil when there was none or mw was already live. Fame is bound to the new world, the town learns which
// mission is under way, and the quick-spell bindings the mission was opened
// with are recorded on it.
//
// party is the mission's own slice and not a copy: the running mission writes
// through it and a save must read what it holds now.
func (s *CampaignSession) activateLive(mw *mapWorld, n int, party []mapload.PartyMember) (outgoing *mapWorld) {
	outgoing = s.live
	mapload.NameParty(party)
	s.bindFameObserver(mw)
	s.Town.activateMission(n)
	s.live, s.liveMission, s.liveParty = mw, n, party
	if s.Town != nil && s.Town.second != nil {
		s.Town.second.current = secondLocation{kind: 1, id: n}
	}
	if mw != nil && mw.mission != nil {
		mw.mission.entryQuickSpells = s.quickSpells
	}
	if outgoing == mw {
		return nil
	}
	return outgoing
}

// nextParty is the party the next mission opened by number starts with: the
// one carried out of the last won mission, or the default the caller derives.
func (s *CampaignSession) nextParty(fallback func() []mapload.PartyMember) []mapload.PartyMember {
	if len(s.Carried) > 0 {
		return s.Carried
	}
	return fallback()
}

// townArrival is how a win reaches the town screen, supplied by the caller
// because opening the town spans the shop, the screen and the party: arrive
// is the economic arrival that stocks the merchant, and welcome grants the
// companions a chapter's town hands out.
type townArrival struct {
	arrive  func()
	welcome func(chapter int)
}

// recordWin is what winning mission n does to the campaign: the town's record
// of the win and its payment, fame, and the answer to what follows. It returns
// a positive mission number to open now, or zero beside the sentence the map
// list or the town states. The party and the purse have already been carried
// home by the caller.
func (s *CampaignSession) recordWin(campaign Campaign, n int, party []mapload.PartyMember,
	w *sim.World, ids []sim.EntityID, roster map[sim.EntityID]mapload.PartyMember,
	arrival townArrival) (next int, message string) {
	// THE TOWN'S RECORD OF THE WIN, WRITTEN BESIDE THE PARTY AND FOR ITS
	// REASON: unconditionally, before anything else is decided, because a
	// campaign this tree could not read still finished this mission and still
	// owes its payment. Won() is idempotent and pays once, so no path below can
	// double it.
	//
	// It does NOT open the town.
	payment := s.Town.campaignPayment(n)
	firstCompletion := !s.Town.Done(n)
	restoredSide, restoredAccepted := s.Town.Won(n)
	if firstCompletion && s.Town.Done(n) {
		s.completeFame(campaign, n, party, w, ids, roster)
	}
	transitionReward := campaign.TransitionRewards[n]
	completed := func(rest string) string {
		prefix := fmt.Sprintf("mission %d won", n)
		if payment > 0 {
			prefix += fmt.Sprintf(" - scenario payment +%d gold", payment)
		}
		if transitionReward > 0 {
			prefix += fmt.Sprintf(" - campaign-transition reward +%d gold", transitionReward)
		}
		return prefix + " - " + rest
	}
	if s.Town.restoredCampaign() {
		if !restoredAccepted {
			s.Offered = s.Town.selectedMission()
			return 0, fmt.Sprintf("mission %d is below restored main progress %d", n, s.Town.Chapter())
		}
		if restoredSide {
			s.Offered = s.Town.selectedMission()
			if s.Town.Open() {
				if s.Town.progress != nil {
					s.Town.progress.firstMapPoint = true
				}
				arrival.arrive()
				return 0, completed(fmt.Sprintf("your party carries over; restored main mission %d remains current", s.Town.Chapter()))
			}
			if _, addressable := MissionMap(s.Offered); addressable {
				return s.Offered, completed(fmt.Sprintf("your party carries over; restored main mission %d follows and opens now", s.Offered))
			}
			return 0, completed(fmt.Sprintf("your party carries over; restored main mission %d remains available", s.Offered))
		}
	}

	if v, ok := campaign.AutoAdvance(n); ok {
		if _, addressable := MissionMap(v); addressable {
			s.Offered = 0
			return v, completed(fmt.Sprintf("your party carries over; mission %d follows and opens now", v))
		}
		s.Offered = 0
		return 0, completed(fmt.Sprintf("your party carries over; the campaign declares mission %d follows and this tree can open no such mission: choose a row", v))
	}

	offer, ok := campaign.NextMission(n)
	if !ok {
		s.Offered = 0
		if len(campaign.Main) == 0 {
			return 0, completed("your party carries over; this install declares no campaign, so nothing names what follows")
		}
		return 0, completed("your party carries over; it is the last mission the campaign declares")
	}
	s.Offered = offer.Mission
	if offer.Town {
		// THE TOWN IS OPENED HERE AND NOWHERE ELSE. This is the campaign's own
		// boundary, already computed by NextMission, and the sentence it used to
		// carry — that the town is not built — is what this story replaces
		// with a screen. Arrive() latches, so a second town chapter reached later
		// is the same open town.
		if s.Town.progress != nil {
			s.Town.progress.firstMapPoint = true
		}
		arrival.arrive()
		if !s.Town.restoredCampaign() {
			arrival.welcome(offer.Mission)
		}
		return 0, completed("your party carries over; the town takes it from here")
	}
	return 0, completed(fmt.Sprintf("your party carries over; mission %d follows: choose its row", offer.Mission))
}
