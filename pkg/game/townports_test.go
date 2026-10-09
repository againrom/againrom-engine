package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func bareTownScreen(t *testing.T) *townScreen {
	t.Helper()
	c := townCampaign(t)
	town := NewTown(c)
	town.Arrive()
	return &townScreen{
		sess: &CampaignSession{Town: town, Carried: []mapload.PartyMember{{ID: "a"}}},
		in:   &InstallResources{Campaign: resolved(c, nil)},
		pc:   &PersistenceContext{},
	}
}

func TestTownScreenFooterReadsSessionAndInstallPorts(t *testing.T) {
	s := bareTownScreen(t)
	s.sess.Town.gold = 321
	got := s.Footer()
	if len(got) != 2 || got[0][:8] != "gold 321" {
		t.Fatalf("footer = %q", got)
	}
}

func TestTownScreenNextPartyPrefersTheCarriedParty(t *testing.T) {
	s := bareTownScreen(t)
	if got := s.nextParty(); len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("nextParty = %+v", got)
	}
}

func TestUnboundTownScreenAnswersNothing(t *testing.T) {
	s := &townScreen{}
	if v := s.TownSquareView(); v.Scene != nil || v.Font != nil {
		t.Fatalf("an unbound screen drew %+v", v)
	}
	s.loadTip(roomSquare)
}

func TestCityBookCandidateIsDetachedUntilCommitted(t *testing.T) {
	s := bareTownScreen(t)
	live := s.sess
	cand := live.cityBookCandidate()
	cand.Town.gold = 5
	cand.Carried[0].ID = "changed"
	if live.Town.gold == 5 || live.Carried[0].ID != "a" {
		t.Fatal("the candidate shares town or party storage with the live session")
	}
	live.commitCityBookCandidate(cand)
	if live.Town.gold != 5 || live.Carried[0].ID != "changed" {
		t.Fatalf("commit left gold %d, member %q", live.Town.gold, live.Carried[0].ID)
	}
}

func TestWithCityBookMutationWithoutAGraphRunsOnTheScreenItself(t *testing.T) {
	s := bareTownScreen(t)
	var seen *townScreen
	s.withCityBookMutation(func(n *townScreen) ui.TownAction {
		seen = n
		return ui.TownAction{}
	})
	if seen != s {
		t.Fatal("the action ran on a copy although the town holds no city graph")
	}
}

func TestPersistenceContextKeepsTheTipPreference(t *testing.T) {
	p := &PersistenceContext{}
	if p.tipsOffNow() {
		t.Fatal("tips start suppressed")
	}
	p.setTipsOff(true)
	if !p.tipsOffNow() {
		t.Fatal("the preference was not kept")
	}
}

func TestPresentationArtCachesResolveOncePerInstall(t *testing.T) {
	p := &Presentation{}
	in := &InstallResources{}
	first := p.shopArt(in)
	if first == nil || p.shopArt(in) != first {
		t.Fatal("the shop art is not cached after its first use")
	}
	if p.tipArt(in) != nil || p.documentFont(in) != nil {
		t.Fatal("an install with no archives produced tip art or a document font")
	}
	if !p.tipArtCache.Tried() || !p.docFontCache.Tried() {
		t.Fatal("a failed first use was not recorded")
	}
}
