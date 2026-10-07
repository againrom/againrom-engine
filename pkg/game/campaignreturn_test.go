package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// These tests run the return-to-town operations on a bare CampaignSession and
// plain install values. No FrontEnd, screen or install is built.

// A town arrival writes the shop and the latch and touches nothing else of the
// session.
func TestArriveInTownOnASessionAloneStocksTheShop(t *testing.T) {
	s := CampaignSession{Town: NewTown(Campaign{}), Offered: 7, liveMission: 9,
		Carried: []mapload.PartyMember{{Body: "carried"}}, live: &mapWorld{}}
	shop := &Shop{}
	s.Shop = shop
	s.arriveInTown(townInstall{})
	if s.Shop == nil || s.Shop == shop {
		t.Fatal("arrival did not rebuild the shop")
	}
	if !s.Town.Open() {
		t.Fatal("arrival did not open the town")
	}
	if s.Offered != 7 || s.liveMission != 9 || s.live == nil || len(s.Carried) != 1 {
		t.Fatalf("arrival changed session state: offered %d mission %d live %v carried %d",
			s.Offered, s.liveMission, s.live != nil, len(s.Carried))
	}
}

// The chapter grant adds one companion once and reads only the table it is given.
func TestAddChapterCompanionsOnASessionAloneGrantsOnce(t *testing.T) {
	in := townInstall{table: companionTable(t),
		campaign: Campaign{Chapters: map[int]Chapter{30: {Mission: 30, AddHero: []int{22}}}}}
	s := CampaignSession{Carried: []mapload.PartyMember{{Name: "Danath", PlayerCharacter: true, StartingHero: true}}}
	s.addChapterCompanions(in, 30)
	s.addChapterCompanions(in, 30)
	if len(s.Carried) != 2 || s.Carried[1].CompanionNPC != 22 || s.Carried[1].Name != "Reniesta" {
		t.Fatalf("carried = %+v, want the original hero and one Reniesta", s.Carried)
	}
}

// The return carries the survivors and the purse home and records no win.
func TestCarryMissionHomeOnASessionAloneCarriesPartyAndPurse(t *testing.T) {
	s := CampaignSession{Town: NewTown(Campaign{})}
	ms := continuityMission(t, 10, sim.ScriptInstantWin)
	if !ms.World.SetPurse(sim.SelfSlot, 733) {
		t.Fatal("SetPurse refused SelfSlot")
	}
	if refused := s.carryMissionHome(townInstall{}, 10, ms.Party, ms.World, ms.Start.IDs, nil); refused != "" {
		t.Fatalf("carryMissionHome refused: %s", refused)
	}
	if len(s.Carried) != 1 || s.Town.gold != 733 {
		t.Fatalf("carried %d members and gold %d, want one member and 733", len(s.Carried), s.Town.gold)
	}
	if s.Town.Done(10) {
		t.Fatal("the carry recorded a win")
	}
}

// Leaving a mission needs the town open, an undecided world and a mission number.
func TestCanLeaveMissionOnASessionAlone(t *testing.T) {
	ms := continuityMission(t, 10, sim.ScriptInstantWin) // already won
	open := CampaignSession{Town: NewTown(Campaign{})}
	open.Town.Arrive()
	if open.canLeaveMission(10, ms.World) {
		t.Fatal("a decided mission may be left")
	}
	if (&CampaignSession{Town: NewTown(Campaign{})}).canLeaveMission(10, nil) {
		t.Fatal("a closed town allows leaving")
	}
}

// A won mission on a bare session: the party and purse come home, the win is
// recorded, the successor is named and the town screen is never reached.
func TestFinishMissionOnASessionAloneRecordsTheWinAndCarriesHome(t *testing.T) {
	campaign := miniCampaign(t, declaring(10, 20))
	s := CampaignSession{Town: NewTown(campaign)}
	ms := continuityMission(t, 10, sim.ScriptInstantWin)
	if !ms.World.SetPurse(sim.SelfSlot, 733) {
		t.Fatal("SetPurse refused SelfSlot")
	}
	screen := false
	next, msg := s.finishMission(townInstall{campaign: campaign}, 10, ms.Party, ms.World, ms.Start.IDs, nil,
		townArrival{arrive: func() { screen = true }, welcome: func(int) { screen = true }})
	if next != 20 || screen || !s.Town.Done(10) || len(s.Carried) != 1 || s.Town.gold != 733 {
		t.Fatalf("next %d screen %v done %v carried %d gold %d (%s)", next, screen, s.Town.Done(10), len(s.Carried), s.Town.gold, msg)
	}
}
