package game

import (
	"image"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// This file witnesses the world map's own per-game session state
// (`worldSelectedOnce`, `worldPosition`) across every site that installs a
// genuinely DIFFERENT game into the one long-lived FrontEnd/townUI, and the
// one site that must NOT clear it: the ordinary mission-to-town return within
// one game.
//
// 1013's own round-1 adversarial review (P finding): `RestoreOriginal`'s
// between-mission (town) branch called `arriveInTown` alone, which resets
// only `worldPosition`, never `worldSelectedOnce` — a player who selects a
// picture-bearing mission, then loads an original save, saw that mission's
// marker painted in a game he never selected it in. The same gap existed in
// `RestoreOriginal`'s mid-mission branch (no reset at all) and in
// `newGameChargen`'s own Begin (a new campaign reachable after a lost
// mission's own notice returns to ScreenMenu, on the same process).

// seedWorldSession simulates "a prior game had selected a mission and moved
// the party" by writing directly into townUI's own fields, the way a real
// session leaves them and no production path resets them by itself.
func seedWorldSession(t *testing.T, f *FrontEnd) {
	t.Helper()
	f.TownScreen() // lazily builds f.townUI, the way the real flow does
	f.townUI.worldSelectedOnce = map[int]bool{5: true}
	f.townUI.worldPosition, f.townUI.worldPositionSet = image.Pt(99, 99), true
	// ROOM IS SEEDED TOO, and it is not world-map state. It stands in for the
	// other 21 fields resetForNewGame covers: an install site that reaches for
	// a narrower reset than resetForNewGame -- resetWorldMarkers and
	// resetWorldPosition are both still callable -- clears the world map and
	// leaves the room, so without this field these four tests would pass over
	// exactly the regression round 3 was written to prevent (1013 adversarial
	// review, round 3, finding W).
	f.townUI.room = roomTalk
}

func assertWorldSessionCleared(t *testing.T, f *FrontEnd, site string) {
	t.Helper()
	if f.townUI == nil {
		t.Fatalf("%s: no townUI to assert against", site)
	}
	if len(f.townUI.worldSelectedOnce) != 0 {
		t.Errorf("%s: worldSelectedOnce = %v, want cleared — a genuinely different "+
			"game must not inherit which missions the PREVIOUS one had selected",
			site, f.townUI.worldSelectedOnce)
	}
	if f.townUI.worldPositionSet {
		t.Errorf("%s: worldPositionSet is still true (position %v), want cleared",
			site, f.townUI.worldPosition)
	}
	if f.townUI.room != roomSquare {
		t.Errorf("%s: room = %v, want roomSquare: this site reset the world map "+
			"but not the rest of townScreen's per-game state; call resetForNewGame, "+
			"whose own doc enumerates the whole population",
			site, f.townUI.room)
	}
}

// This section (below) is 1019 round 2's own addition: the four tests above
// already witness townUI's own per-game state at every install site, but
// none of them asserted FrontEnd's own session fields (Carried, Offered,
// Town, live/liveMission/liveParty, Shop) — which is exactly the gap round
// 2's adversarial review found in RestoreOriginal's mid-mission arm, on a
// test file that was green while that gap shipped. seedFrontEndSession and
// assertFrontEndSessionCleared are seedWorldSession/assertWorldSessionCleared's
// own pattern, applied one layer down.

const (
	// feFenceMission is a mission number none of this file's fixture
	// campaigns (all Campaign{}, or scenarioFixture where noted) ever
	// declares, so Town.Done reading true for it can only be the seeded
	// PREVIOUS game's own mark surviving.
	feFenceMission   = 999999
	feFenceCarried   = "the previous game's own carried hero"
	feFenceLiveBody  = "the previous game's own live party"
	feFenceLiveMissn = 66
	feFenceOffered   = 77
)

// seedFrontEndSession marks f as though a PREVIOUS game had already run in
// this process: a town reached and progressed, a party carried forward, a
// next mission offered, an abandoned mission's own world still referenced,
// and a merchant stocked. Requires f.Town to be non-nil already (every
// fixture in this file sets one).
func seedFrontEndSession(t *testing.T, f *FrontEnd) (seededShop *Shop) {
	t.Helper()
	f.Town.Arrive()
	f.Town.won[feFenceMission] = true
	f.Offered = feFenceOffered
	f.Carried = []mapload.PartyMember{{Body: feFenceCarried}}
	f.live = &mapWorld{}
	f.liveMission = feFenceLiveMissn
	f.liveParty = []mapload.PartyMember{{Body: feFenceLiveBody}}
	seededShop = &Shop{}
	f.Shop = seededShop
	return seededShop
}

// assertFrontEndSessionCleared is seedFrontEndSession's own counterpart.
// carriedFromSave says whether the site under test is expected to install a
// DECODED party into Carried (RestoreOriginal's between-mission arm and
// installCandidate's two native-load arms all restore a party rather than
// starting empty) rather than leave it at its own zero value (every
// mid-mission/mission arm, whose opened mission's party is an argument and
// never Carried itself). Either way, the previous game's own sentinel member
// must be gone.
func assertFrontEndSessionCleared(t *testing.T, f *FrontEnd, site string, carriedFromSave bool, seededShop *Shop) {
	t.Helper()
	if f.Offered != 0 {
		t.Errorf("%s: Offered = %d, want 0 — the previous game's map-list bookkeeping survived",
			site, f.Offered)
	}
	if f.live != nil || f.liveMission != 0 || f.liveParty != nil {
		t.Errorf("%s: live/liveMission/liveParty = %v/%d/%v, want nil/0/nil — the abandoned "+
			"mission's own world survived", site, f.live, f.liveMission, f.liveParty)
	}
	if f.Town == nil {
		t.Fatalf("%s: Town is nil", site)
	}
	if f.Town.Done(feFenceMission) {
		t.Errorf("%s: Town still marks the previous game's own fence mission %d done",
			site, feFenceMission)
	}
	for _, m := range f.Carried {
		if m.Body == feFenceCarried {
			t.Errorf("%s: Carried still holds the previous game's own member", site)
		}
	}
	if !carriedFromSave && len(f.Carried) != 0 {
		t.Errorf("%s: Carried = %+v, want none — this site's opened mission carries its "+
			"party as an argument, not through Carried", site, f.Carried)
	}
	if f.Shop == seededShop {
		t.Errorf("%s: Shop is still the previous game's own merchant", site)
	}
}

// TestInstallCandidateClearsWorldSessionOnATownOnlyNativeLoad is the
// native-save (.ags) town-only branch of installCandidate.
func TestInstallCandidateClearsWorldSessionOnATownOnlyNativeLoad(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(Campaign{}, nil)}, CampaignSession: CampaignSession{Town: NewTown(Campaign{})}}
	seedWorldSession(t, f)
	shop := seedFrontEndSession(t, f)
	f.installCandidate(&restoreCandidate{
		town:     NewTown(Campaign{}),
		carried:  []mapload.PartyMember{{Body: "loaded"}},
		townOnly: true,
	})
	assertWorldSessionCleared(t, f, "installCandidate (town-only)")
	assertFrontEndSessionCleared(t, f, "installCandidate (town-only)", true, shop)
}

// TestInstallCandidateClearsWorldSessionOnAMissionNativeLoad is the
// native-save mission branch of installCandidate — the reset runs before the
// townOnly/mission split, so it must fire here too even though this branch
// never itself touches the town screen.
func TestInstallCandidateClearsWorldSessionOnAMissionNativeLoad(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(Campaign{}, nil)}, CampaignSession: CampaignSession{Town: NewTown(Campaign{})}}
	seedWorldSession(t, f)
	seedFrontEndSession(t, f) // Shop's own return value is unused: see below
	activated := false
	f.installCandidate(&restoreCandidate{
		town:     NewTown(Campaign{}),
		carried:  []mapload.PartyMember{{Body: "loaded"}},
		townOnly: false,
		activate: func() { activated = true },
	})
	if !activated {
		t.Fatal("installCandidate did not run the mission branch's own activate")
	}
	assertWorldSessionCleared(t, f, "installCandidate (mission)")
	// live/liveMission/liveParty are NOT asserted cleared here: this branch's
	// own activate() is a test stub that never calls liveDriver, so the field
	// state after it proves nothing about production, where the real
	// activate always installs the NEW mission's own live world (verified by
	// reading liveDriver, resume.go:416-419: it sets live/liveMission/liveParty
	// unconditionally and touches nothing else).
	//
	// Shop is likewise NOT asserted here, and this is a verified absence of
	// requirement rather than an oversight: liveDriver never touches f.Shop,
	// and arriveInTown (frontend.go:434-447, the ONE place f.Shop is ever
	// assigned) always rebuilds it from scratch on the next town arrival,
	// discarding whatever it held. The previous game's stale merchant is
	// unreachable in between: the player is inside the newly opened mission,
	// never at ScreenTown, until the next arriveInTown overwrites it. What
	// must still be gone here is Offered/Town/Carried, which installCandidate
	// sets unconditionally before the townOnly/mission split.
	if f.Offered != 0 {
		t.Errorf("installCandidate (mission): Offered = %d, want 0", f.Offered)
	}
	if f.Town.Done(feFenceMission) {
		t.Errorf("installCandidate (mission): Town still marks the previous game's own fence mission done")
	}
	for _, m := range f.Carried {
		if m.Body == feFenceCarried {
			t.Errorf("installCandidate (mission): Carried still holds the previous game's own member")
		}
	}
}

// TestRestoreOriginalClearsWorldSessionBetweenMissions is RestoreOriginal's
// own town branch (an original save taken between missions, mission 0).
func TestRestoreOriginalClearsWorldSessionBetweenMissions(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(Campaign{}, nil)}, CampaignSession: CampaignSession{Town: NewTown(Campaign{}), Carried: []mapload.PartyMember{{Body: "existing"}}}}
	seedWorldSession(t, f)
	shop := seedFrontEndSession(t, f)
	open, town, err := f.RestoreOriginal(savedFile(0, tenActors()))
	if err != nil {
		t.Fatalf("RestoreOriginal: %v", err)
	}
	if open != nil || !town {
		t.Fatalf("load returned opener %v town=%v, want nil/true", open != nil, town)
	}
	assertWorldSessionCleared(t, f, "RestoreOriginal (between missions)")
	// carriedFromSave=true: this arm decodes its own party from the save and
	// overwrites Carried with it (originalsave.go, RestoreParty's result),
	// same as installCandidate's two native-load arms.
	assertFrontEndSessionCleared(t, f, "RestoreOriginal (between missions)", true, shop)
}

// TestRestoreOriginalClearsWorldSessionMidMission is RestoreOriginal's own
// mission branch (an original save taken inside mission 10). Preparation must
// preserve the old session; adopting the prepared opener resets it exactly once.
func TestRestoreOriginalClearsWorldSessionMidMission(t *testing.T) {
	f := missionFrontEnd(t)
	f.Town = NewTown(Campaign{})
	seedWorldSession(t, f)
	shop := seedFrontEndSession(t, f)
	previousTown, previousLive := f.Town, f.live
	open, town, err := f.RestoreOriginal(savedFile(10, nil))
	if err != nil {
		t.Fatalf("RestoreOriginal: %v", err)
	}
	if open == nil || town {
		t.Fatalf("load returned opener %v town=%v, want non-nil/false", open != nil, town)
	}
	if f.Town != previousTown || f.live != previousLive || f.Shop != shop ||
		f.Offered != feFenceOffered || !f.townUI.worldSelectedOnce[5] || f.townUI.room != roomTalk {
		t.Fatal("original preparation changed the previous session before commit")
	}
	if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
		t.Fatal(err)
	}
	assertWorldSessionCleared(t, f, "RestoreOriginal (mid-mission)")
	if f.Town == previousTown || f.Town.Done(feFenceMission) || len(f.Carried) != 0 ||
		f.Offered != 0 || f.Shop == shop || f.live == previousLive || f.live == nil || f.liveMission != 10 {
		t.Fatal("opened original game kept the previous campaign or failed to install its live driver")
	}
	for _, member := range f.liveParty {
		if member.Body == feFenceCarried || member.Body == feFenceLiveBody {
			t.Fatal("original fallback retained the previous game's party")
		}
	}
	loadedLive := f.live
	f.Offered = 23
	if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
		t.Fatal(err)
	}
	if f.live != loadedLive || f.Offered != 23 {
		t.Fatal("reusing the prepared original opener committed twice")
	}
}

func TestRestoreOriginalMissingMapPreservesSession(t *testing.T) {
	f := missionFrontEnd(t) // contains map 10, not map 20
	f.Town = NewTown(Campaign{})
	seedWorldSession(t, f)
	seedFrontEndSession(t, f)
	before, beforeUI := *f, *f.townUI
	open, town, err := f.RestoreOriginal(savedFile(20, tenActors()))
	if err == nil || !strings.Contains(err.Error(), "read scenario/20.alm") || open != nil || town {
		t.Fatalf("missing original map = opener:%t town:%t error:%v", open != nil, town, err)
	}
	after := *f.townUI
	after.openMission, beforeUI.openMission = nil, nil
	if !reflect.DeepEqual(*f, before) || !reflect.DeepEqual(after, beforeUI) {
		t.Fatal("failed original mission opener changed the previous session")
	}
}

// TestNewGameChargenClearsWorldSession is newGameChargen's own Begin, reached
// from NEW GAME — reachable again on the same process after a lost mission's
// own notice returns to ScreenMenu, so a fresh character must not inherit a
// PREVIOUS game's world map session either.
func TestNewGameChargenClearsWorldSession(t *testing.T) {
	f := missionFrontEnd(t)
	f.Table, f.Humans = &mapload.Table{}, fourBaseHumans()
	f.Bodies, f.Town = data.NewBodyList("unarmed", "mage"), NewTown(Campaign{})
	f.Maps = []MapEntry{{Source: "10.alm", Name: "B", Mission: 10}}
	seedWorldSession(t, f)
	shop := seedFrontEndSession(t, f)
	previousTown, previousLive := f.Town, f.live
	e := f.newGameChargen(0)
	if e == nil || e.Begin == nil {
		t.Fatal("newGameChargen(0) did not claim the one mission row")
	}
	open, err := e.Begin(ui.ChargenResult{Choices: []int{1, 1, 2}, Stats: []int{30, 20, 18, 16}})
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if open == nil {
		t.Fatal("begin answered no opener")
	}
	if f.Town != previousTown || f.live != previousLive || f.Shop != shop {
		t.Fatal("draft Begin changed the previous session before opening the mission")
	}
	if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
		t.Fatal(err)
	}
	assertWorldSessionCleared(t, f, "newGameChargen")
	// carriedFromSave=false: a fresh character has no prior save to decode a
	// party from; ChargenParty builds the new party and it is passed to
	// MissionOpenerWith as an argument, never through Carried.
	if f.Town == previousTown || f.Town.Done(feFenceMission) || len(f.Carried) != 0 ||
		f.Offered != 0 || f.Shop == shop || f.live == previousLive || f.liveMission != 10 {
		t.Fatal("opened new game kept the previous campaign or failed to install the new driver")
	}
}

// TestArriveInTownAloneDoesNotClearTheMarkerCache is the survival side of the
// same rule: arriveInTown is the ORDINARY mission-to-town return within one
// game (winning a mission, or loading a save made in town within the SAME
// native/original resume that already reset the session once). It must not
// itself drop worldSelectedOnce, or a mission selected earlier this game
// would lose its marker on every homecoming.
func TestArriveInTownAloneDoesNotClearTheMarkerCache(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(Campaign{}, nil)}, CampaignSession: CampaignSession{Town: NewTown(Campaign{})}}
	seedWorldSession(t, f)
	f.Offered = feFenceOffered
	f.Carried = []mapload.PartyMember{{Body: feFenceCarried}}
	f.live = &mapWorld{}
	f.liveMission = feFenceLiveMissn
	f.liveParty = []mapload.PartyMember{{Body: feFenceLiveBody}}
	shop := &Shop{}
	f.Shop = shop
	f.arriveInTown()
	if len(f.townUI.worldSelectedOnce) == 0 {
		t.Fatal("arriveInTown alone cleared the marker cache; it must survive an ordinary " +
			"mission-to-town return within one game")
	}
	// The position DOES reset on every arrival, by design (`DIV-137`) —
	// unlike the marker cache, it carries no game-identity distinction.
	if f.townUI.worldPositionSet {
		t.Fatal("arriveInTown did not reset the position on an ordinary return")
	}
	// Survival side of 1019 round 2's own addition: an ordinary mission-to-
	// town return within ONE game must not drop FrontEnd's own session
	// fields either, or every homecoming would look like a new game. Shop is
	// the one exception BY DESIGN: arriveInTown always rebuilds it fresh
	// (frontend.go:440-441), so it never survives one arrival to the next,
	// win or lose, same game or not.
	if f.Offered != feFenceOffered {
		t.Errorf("arriveInTown alone cleared Offered (%d), want it untouched", f.Offered)
	}
	if f.live == nil || f.liveMission != feFenceLiveMissn || f.liveParty == nil {
		t.Error("arriveInTown alone cleared live/liveMission/liveParty, want them untouched")
	}
	found := false
	for _, m := range f.Carried {
		if m.Body == feFenceCarried {
			found = true
		}
	}
	if !found {
		t.Error("arriveInTown alone cleared Carried, want it untouched")
	}
	if f.Shop == shop {
		t.Error("arriveInTown did not rebuild Shop on an ordinary return")
	}
}
