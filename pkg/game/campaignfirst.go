package game

import (
	"errors"
	"fmt"

	"againrom/pkg/base"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// firstCampaignRules is the first game's campaign service: numbered chapters
// over one town, a world map between them and the chapter record in every
// save.
type firstCampaignRules struct{}

func (firstCampaignRules) townAdapter(t *townScreen) townCampaignAdapter {
	return chapterTownCampaign{t}
}

func (firstCampaignRules) townSavePoint(f *FrontEnd) error {
	if !f.Town.Open() {
		return errors.New("there is no game to save yet")
	}
	return nil
}

func (firstCampaignRules) captureMission(_ *FrontEnd, s *Snapshot) { s.noticeOpen = false }

func (firstCampaignRules) exportable(Snapshot) error { return nil }

func (firstCampaignRules) refuseCompleted(f *FrontEnd, s Snapshot) error {
	if s.Mission != 0 {
		return nil
	}
	t, err := restoreTown(f.Campaign.Value(), s)
	if err != nil {
		return err
	}
	if f.Campaign.Value().completedBy(t) {
		return errCompletedCampaignSave
	}
	return nil
}

// recordProfile: the chapter progress states every record, and a SAVE
// writes the current chapter's offer labels.
func (firstCampaignRules) recordProfile() campaignRecordProfile {
	return campaignRecordProfile{offerLabels: offerLabelsOfChapter}
}

func (r firstCampaignRules) validateSession(s *currentSessionData) error {
	if r.ownsGame(s.Game) != nil {
		return fmt.Errorf("invalid current save game")
	}
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

func (firstCampaignRules) missionParty(party []mapload.PartyMember, _ *Town, _ *mapload.Table) ([]mapload.PartyMember, error) {
	return party, nil
}

// place: the party stands in its live mission while one is open, and
// otherwise in the town its town state belongs to once the campaign has
// reached the town; before that it stands nowhere.
func (firstCampaignRules) place(s *CampaignSession) partyPlace {
	switch {
	case s == nil:
		return partyPlace{}
	case s.live != nil && s.liveMission > 0:
		return partyPlace{kind: placeMission, id: s.liveMission}
	case s.Town.Open():
		return partyPlace{kind: placeTown, id: s.Town.townID()}
	}
	return partyPlace{}
}

func (firstCampaignRules) enterMission(*Town, int) {}

func (firstCampaignRules) seatWorld(missionRequest, *Mission) error { return nil }

func (firstCampaignRules) selectedMarkers(f *FrontEnd, src *originalSource) (map[int]bool, error) {
	return worldSelectedOnceFromSnapshot(worldMapMarkerMissions(src.campaign.town, coldWorldMapData(f.worldMapCache, f.Archives)))
}

func (firstCampaignRules) arriveLoaded(f *FrontEnd, c *restoreCandidate) {
	f.arriveInTown()
	f.restoreWorldMapReturn(c)
}

func (firstCampaignRules) keepsTownSurface(*restoreCandidate) bool { return false }

func (firstCampaignRules) keepsTownActorState() bool { return true }

func (firstCampaignRules) beforeAdvance(*FrontEnd, int, *Mission, ui.NoticeAction) string { return "" }

func (firstCampaignRules) finishWon(f *FrontEnd, n int, ms *Mission) (int, string) {
	return f.FinishMissionWithRoster(n, ms.Party, ms.World, ms.Start.IDs, ms.Start.Roster)
}

func (firstCampaignRules) route(f *FrontEnd, n, successor int) winRoute {
	return f.CampaignSession.routeAfterWin(f.Campaign.Value(), n, successor)
}

func (firstCampaignRules) returnToTown(f *FrontEnd, n int) {
	f.TownScreen().(*townScreen).beginWorldMapReturn(f.Town.guestMission().returnMission(n))
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

func (firstCampaignRules) generatorSetup(f *FrontEnd, setup *ui.ChargenSetup) {
	firstGeneratorPresets(f, setup)
}

func (firstCampaignRules) generatorParty(f *FrontEnd, res ui.ChargenResult) []mapload.PartyMember {
	return firstGeneratorParty(f, res)
}

func (firstCampaignRules) generatorBegin(f *FrontEnd, mission int) func(ui.ChargenResult) (ui.MapOpener, error) {
	return func(res ui.ChargenResult) (ui.MapOpener, error) { return f.NewGameOpener(mission, res), nil }
}

func (firstCampaignRules) chat() *chatAdapter { return &firstChat }

// chatCampaign: the first game's adapter has no campaign rule.
func (firstCampaignRules) chatCampaign(*Town) bool { return false }

func (firstCampaignRules) files() *gameFiles { return &firstGameFiles }

func (firstCampaignRules) eventTags() eventTags { return firstEventTags{} }

func (firstCampaignRules) newGameLimit(p base.Profile) string {
	return fmt.Sprintf("no character generation; new game opens mission %d with the default party", p.Mission())
}

func (firstCampaignRules) captureDifficulty(*FrontEnd, *Snapshot, bool) {}

func (firstCampaignRules) recordDifficulty(Snapshot, *currentSessionData) {}

func (firstCampaignRules) difficultyRecord(d mapload.Difficulty) (uint32, bool) {
	return uint32(d) - 1, true
}

// fileState reads the shortcut cells; the file states no session.
func (firstCampaignRules) fileState(sf *sav.File) (originalFileState, error) {
	quick, err := originalQuickSpells(sf)
	return originalFileState{quickSpells: quick}, err
}

func (firstCampaignRules) ownsGame(g base.Game) error {
	if base.SameGame(g, base.GameROM1) {
		return nil
	}
	return errOtherGame
}

func (firstCampaignRules) testPlayRefusal() error { return nil }

func (firstCampaignRules) censusRefusal() error {
	return errors.New("the census needs a second-game root")
}

type chapterTownCampaign struct{ t *townScreen }

func (c chapterTownCampaign) chapter() int          { return c.t.sess.Town.Chapter() }
func (chapterTownCampaign) unavailableTowns() []int { return nil }
func (chapterTownCampaign) startCampaign() error {
	return errors.New("campaign does not start in town")
}

func (c chapterTownCampaign) book() *townBook {
	if c.t.sess != nil && c.t.sess.Town != nil && c.t.sess.Town.camp.Towns != nil {
		return c.t.sess.Town.camp.Towns
	}
	return townBookOf(c.t.profile())
}
func (c chapterTownCampaign) room() string { return townRoomName(c.t.room) }
func (c chapterTownCampaign) setRoom(name string) {
	if r, ok := townRoomNames[name]; ok {
		c.t.room = r
	}
}
func (c chapterTownCampaign) townID() int {
	if c.t.sess != nil && c.t.sess.Town != nil {
		return c.t.sess.Town.townID()
	}
	return chapterTown
}
func (c chapterTownCampaign) ready() bool   { return c.t.sess != nil && c.t.sess.Town != nil }
func (c chapterTownCampaign) present() bool { return true }
func (c chapterTownCampaign) talking() bool { return c.t.room == roomTalk }

func (c chapterTownCampaign) talkCursor() *talkCursor {
	if !c.talking() {
		return nil
	}
	return &c.t.talkPage
}

// endTalk returns to the opening room; the room popup keeps its lifetime.
func (c chapterTownCampaign) endTalk() {
	c.t.room = c.t.dialogueRoom()
	c.t.mercenaryTalk = 0
	c.t.rereadTip(c.t.roomName())
}
func (c chapterTownCampaign) condition(name string) bool {
	return name == "gate-open" && c.ready() && (c.t.sess.Town.gateMission() != -1 || len(c.t.sess.Town.travelTowns()) > 0)
}
func (c chapterTownCampaign) value(name string) int {
	if !c.ready() {
		return -1
	}
	switch name {
	case "tavern-offers":
		return countOffering(c.t.sess.Town.Offers(TownTavern))
	case "shop-offers":
		return len(c.t.sess.Town.Offers(TownShop))
	case "school-offers":
		return len(c.t.sess.Town.Offers(TownSchool))
	case "missions":
		return len(c.t.sess.Town.Available())
	}
	return -1
}
func (c chapterTownCampaign) savePoint() bool     { return c.ready() && c.t.sess.Town.Open() }
func (chapterTownCampaign) leave()                {}
func (chapterTownCampaign) bank() *secondCampaign { return nil }
