package game

import (
	"errors"
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// firstCampaignRules is the first game's campaign service: numbered chapters
// over one town, a world map between them and the chapter record in every
// save.
type firstCampaignRules struct{}

func (firstCampaignRules) townScreen(f *FrontEnd) ui.TownScreen { return f.chapterTownScreen() }

func (firstCampaignRules) townSavePoint(f *FrontEnd) error {
	if !f.Town.Open() {
		return errors.New("there is no game to save yet")
	}
	return nil
}

func (firstCampaignRules) captureMission(_ *FrontEnd, s *Snapshot) { s.noticeOpen = false }

func (firstCampaignRules) exportable(Snapshot) error { return nil }

func (firstCampaignRules) refuseCompleted(f *FrontEnd, s Snapshot) error {
	if s.Mission == 0 && f.Campaign.Value().completedBy(restoreTown(f.Campaign.Value(), s)) {
		return errCompletedCampaignSave
	}
	return nil
}

func (firstCampaignRules) campaignProjection(Snapshot) (sav.CampaignProjection, bool) {
	return sav.CampaignProjection{}, false
}

func (firstCampaignRules) validateSession(s *currentSessionData) error {
	if s.Second != nil {
		return fmt.Errorf("current save game and campaign disagree")
	}
	if s.Difficulty != nil {
		return fmt.Errorf("invalid current city difficulty")
	}
	return nil
}

func (firstCampaignRules) validateAuthored(_ *sav.DocumentData, a *currentActionData) error {
	if a != nil && (a.Session != nil && a.Session.Second != nil || a.Program != nil && a.Program.Dialect != sim.ScriptROM1 || a.Policy != nil && a.Policy.ROM2 != nil) {
		return fmt.Errorf("ROM1 save contains ROM2 continuation")
	}
	return nil
}

func (firstCampaignRules) undecodedSave(error) error { return nil }

func (firstCampaignRules) statesCampaign(*currentSessionData) bool { return false }

func (firstCampaignRules) canEnter(*Town, int) bool { return true }

func (firstCampaignRules) enterMission(*Town, int) {}

func (firstCampaignRules) seatWorld(missionRequest, *Mission) error { return nil }

func (firstCampaignRules) selectedMarkers(f *FrontEnd, src *originalSource) (map[int]bool, error) {
	return worldSelectedOnceFromSnapshot(worldMapMarkerMissions(src.campaign.town, coldWorldMapData(f.worldMapCache, f.Archives)))
}

func (firstCampaignRules) checkTownLoad(*FrontEnd, *originalSource) error { return nil }

func (firstCampaignRules) arriveLoaded(f *FrontEnd, c *restoreCandidate) {
	f.arriveInTown()
	f.restoreWorldMapReturn(c)
}

func (firstCampaignRules) keepsTownSurface(*restoreCandidate) bool { return false }

func (firstCampaignRules) beforeAdvance(*FrontEnd, int, *Mission, ui.NoticeAction) string { return "" }

func (firstCampaignRules) finishWon(f *FrontEnd, n int, ms *Mission) (int, string) {
	return f.FinishMissionWithRoster(n, ms.Party, ms.World, ms.Start.IDs, ms.Start.Roster)
}

func (firstCampaignRules) route(f *FrontEnd, n, successor int) winRoute {
	return f.CampaignSession.routeAfterWin(f.Campaign.Value(), n, successor)
}

func (firstCampaignRules) returnToTown(f *FrontEnd, n int) {
	f.TownScreen().(*townScreen).beginWorldMapReturn(n)
}

func (firstCampaignRules) openMission(*ui.Viewer) {}

func (firstCampaignRules) loadMissionText(*mapWorld, entrySource, *Mission, *mapload.Table) {}

func (firstCampaignRules) scriptMessages(*mapWorld, []int32) {}

func (firstCampaignRules) settleNotices(*mapWorld) bool { return false }

func (firstCampaignRules) acknowledgeNotice(*mapWorld) bool { return false }

func (firstCampaignRules) outcomeText(_ *mapWorld, body string) string { return body }

func (firstCampaignRules) eventAudience(*mapWorld) (EventAudience, bool) {
	return EventAudience{}, false
}

func (firstCampaignRules) objectives(_ *mapWorld, briefing string) string { return briefing }

func (firstCampaignRules) generatorPresets(f *FrontEnd, setup *ui.ChargenSetup) {
	firstGeneratorPresets(f, setup)
}

func (firstCampaignRules) generatorParty(f *FrontEnd, res ui.ChargenResult) []mapload.PartyMember {
	return firstGeneratorParty(f, res)
}

func (firstCampaignRules) generatorBegin(f *FrontEnd, mission int) func(ui.ChargenResult) (ui.MapOpener, error) {
	return func(res ui.ChargenResult) (ui.MapOpener, error) { return f.NewGameOpener(mission, res), nil }
}
