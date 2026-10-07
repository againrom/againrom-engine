package game

import (
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// These tests run the restore of an original SAV, the world-map marker
// population, the viewer-art resolution and the won-mission routing from plain
// values. No FrontEnd, screen, sound device or install is built.

func TestDecodeOriginalSourceReadsABetweenMissionSaveFromPlainValues(t *testing.T) {
	c, projection := restoredCampaignFixture()
	src, err := decodeOriginalSource(originalSaveWithCampaign(t, 0, projection), mapload.ModContext{}, c)
	if err != nil {
		t.Fatalf("decodeOriginalSource: %v", err)
	}
	if src.campaign.mission != 0 || src.campaign.town == nil || src.campaign.progress == nil {
		t.Fatalf("source campaign = %+v, want a restored town", src.campaign)
	}
	if src.missionDocument != nil || src.hasGround || len(src.holdings) != 0 {
		t.Fatalf("a between-mission save decoded mission state: %+v", src)
	}
	if src.offered() != 0 || src.worldMapReturn() != nil {
		t.Fatalf("offered %d, world-map return %v, want none from a file with no session supplement", src.offered(), src.worldMapReturn())
	}
}

func TestDecodeOriginalSourceRefusesBytesThatAreNotASave(t *testing.T) {
	if _, err := decodeOriginalSource([]byte("not a save"), mapload.ModContext{}, Campaign{}); err == nil {
		t.Fatal("decodeOriginalSource accepted bytes that are not a SAV")
	}
}

// The between-mission restore completes on a session of its own and reports
// the party it restored; the install it reads is plain values.
func TestRestoreOriginalTownOnASessionAloneBuildsTheTown(t *testing.T) {
	c, projection := restoredCampaignFixture()
	src, err := decodeOriginalSource(originalSaveWithCampaign(t, 0, projection), mapload.ModContext{}, c)
	if err != nil {
		t.Fatal(err)
	}
	var s CampaignSession
	if _, err := s.restoreOriginalTown(src, originalInstall{}, c); err != nil {
		t.Fatalf("restoreOriginalTown: %v", err)
	}
	if s.Town != src.campaign.town || s.Difficulty != src.campaign.difficulty || s.quickSpells != src.quickSpells {
		t.Fatalf("session town %p difficulty %v spells %v, want the decoded town, difficulty and bindings", s.Town, s.Difficulty, s.quickSpells)
	}
	if len(s.Carried) == 0 || s.Carried[0].ID == "" {
		t.Fatalf("carried = %+v, want a named restored party", s.Carried)
	}
	if s.originalCity == nil || s.live != nil || s.Shop != nil {
		t.Fatalf("originalCity %v live %v shop %v, want provenance bound and no live map", s.originalCity, s.live, s.Shop)
	}
}

// A refusal on the detached session leaves the session it was never given alone.
func TestRestoreHiredMercenariesRefusesAnOversizedPoolOnASessionAlone(t *testing.T) {
	c, _ := restoredCampaignFixture()
	s := CampaignSession{Town: NewTown(c), Carried: []mapload.PartyMember{{Name: "keep"}}}
	s.Town.mercHired[4] = true
	s.Town.mercPool[4] = legacyCityHireRebuildCap + 1
	err := s.restoreHiredMercenaries(nil)
	if err == nil || !strings.Contains(err.Error(), "over the 12-member roster cap") {
		t.Fatalf("restoreHiredMercenaries error = %v, want the roster-cap refusal", err)
	}
	if len(s.Carried) != 1 {
		t.Fatalf("carried = %d, want the refused roster untouched", len(s.Carried))
	}
	if err := (*CampaignSession)(nil).restoreHiredMercenaries(nil); err != nil {
		t.Fatalf("nil session: %v", err)
	}
}

func TestBuildMercenarySquadNeedsATableAndACount(t *testing.T) {
	if _, ok := buildMercenarySquad(nil, nil, 3, 1); ok {
		t.Fatal("a squad was built with no table")
	}
	if _, ok := buildMercenarySquad(&mapload.Table{}, nil, 3, 0); ok {
		t.Fatal("a squad was built for a count of zero")
	}
}

// The marker population reads the cache it is given by value: an untried cache
// is read from the archives for this call and stays untried.
func TestColdWorldMapDataDoesNotWarmTheCacheItIsGiven(t *testing.T) {
	var cache lazy[*worldMapAssets]
	if data := coldWorldMapData(cache, nil); data != nil {
		t.Fatalf("data = %+v, want none from an install with no archives", data)
	}
	if cache.Tried() {
		t.Fatal("the caller's cache was warmed")
	}
	want := &globalMapData{Missions: map[int]int{30: 0}}
	if got := coldWorldMapData(resolved(&worldMapAssets{data: want}, nil), nil); got != want {
		t.Fatalf("data = %p, want the cached %p", got, want)
	}
	if got := coldWorldMapData(resolved[*worldMapAssets](nil, nil), nil); got != nil {
		t.Fatalf("data = %+v, want none from a tried cache holding nothing", got)
	}
}

func TestWorldMapMarkerMissionsKeepsOnlyMissionsWithAPicture(t *testing.T) {
	data := &globalMapData{Missions: map[int]int{30: 0, 40: 1},
		Objects: []globalMapObject{{Valid: true, Picture: "mark"}, {Valid: true, Picture: "nothing"}}}
	town := NewTown(Campaign{})
	if got := worldMapMarkerMissions(town, data); len(got) != 0 {
		t.Fatalf("markers = %v, want none for a town with no selection", got)
	}
	if got := worldMapMarkerMissions(nil, data); len(got) != 0 {
		t.Fatalf("markers = %v, want none for no town", got)
	}
}

// The viewer's art resolves from the install and the presentation alone and
// warms the presentation's caches.
func TestViewerArtSourceResolvesFromTheComponentsAlone(t *testing.T) {
	in := &InstallResources{Words: ui.Words{}}
	pr := &Presentation{}
	art := viewerArtSource{in: in, pr: pr}.resolve()
	if art.dialogFrame != nil || art.paneCorners != nil || art.font != nil {
		t.Fatalf("art = %+v, want no pictures from an install with no archives", art)
	}
	if !pr.dialogArt.Tried() || !pr.characterPaneCache.Tried() || !pr.characterCornerCache.Tried() || !pr.characterFillerCache.Tried() {
		t.Fatal("resolving did not try every first-use cache once")
	}
}

func TestRouteAfterWinNamesWhereTheCampaignGoes(t *testing.T) {
	campaign := miniCampaign(t, declaring(10, 20))
	for _, tc := range []struct {
		name      string
		successor int
		open      bool
		want      winRoute
	}{
		{"unfinished win", -1, true, winStay},
		{"successor", 20, false, winMission},
		{"no successor, town open", 0, true, winTown},
		{"no successor, town closed", 0, false, winList},
	} {
		s := CampaignSession{Town: NewTown(campaign)}
		if tc.open {
			s.Town.Arrive()
		}
		if got := s.routeAfterWin(campaign, 10, tc.successor); got != tc.want {
			t.Errorf("%s: route %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestRouteAfterWinAtTheEndForgetsTheLiveMap(t *testing.T) {
	campaign := miniCampaign(t, declaring(10, 20))
	s := CampaignSession{Town: NewTown(campaign), live: &mapWorld{}, liveMission: 10}
	// A mission that is not terminal does not end the campaign.
	if got := s.routeAfterWin(campaign, 10, 0); got == winEnding || s.live == nil {
		t.Fatalf("route %d live %v, want a non-terminal win to leave the live map", got, s.live)
	}
}

func TestLeaveLiveRestoresTheEntryQuickSpells(t *testing.T) {
	ms := continuityMission(t, 10, 0)
	s := CampaignSession{Town: NewTown(Campaign{}), quickSpells: [4]uint32{9, 9, 9, 9},
		live: &mapWorld{mission: &missionNotices{entryQuickSpells: [4]uint32{1, 2, 3, 4}}}}
	if refused := s.leaveLive(townInstall{}, 10, ms); refused != "" {
		t.Fatalf("leaveLive refused: %s", refused)
	}
	if s.quickSpells != [4]uint32{1, 2, 3, 4} {
		t.Fatalf("quick spells = %v, want the bindings the mission was opened with", s.quickSpells)
	}
	var bare CampaignSession
	if refused := bare.leaveLive(townInstall{}, 10, ms); refused != "" || bare.quickSpells != [4]uint32{} {
		t.Fatalf("a session with no live map changed: %q %v", refused, bare.quickSpells)
	}
}
