package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondCampaignRules is the second game's campaign service: a scenario bank
// that names the towns and missions the party may travel to, its own town
// screen, its own save points and the script's message queue. A session that
// holds no second-campaign state yet keeps the first game's behaviour, which
// the embedded rules give.
type secondCampaignRules struct{ firstCampaignRules }

func (secondCampaignRules) townScreen(f *FrontEnd) ui.TownScreen { return f.secondCampaignScreen() }

func (secondCampaignRules) townSavePoint(f *FrontEnd) error {
	if f.Town == nil || !f.Town.second.savePoint() {
		return errSavingUnavailable
	}
	return nil
}

func (secondCampaignRules) captureMission(f *FrontEnd, s *Snapshot) {
	if f.live.view != nil {
		animation := f.live.view.SaveAnimation()
		s.mapAnimation = &animation
		s.mapMotion = captureCurrentMapMotion(f.live)
	}
	s.noticeOpen = f.live.mission != nil && f.live.mission.open
}

func (secondCampaignRules) exportable(s Snapshot) error {
	if s.second == nil || s.noticeOpen {
		return errSavingUnavailable
	}
	if s.Mission == 0 {
		if s.second.validateTown() != nil || len(s.World) != 0 || len(s.Party) == 0 || s.Residue.hasMissionState() {
			return errSavingUnavailable
		}
	} else if !secondSaveMission(int(s.Mission)) || s.second.Current != (currentSecondLocation{1, int(s.Mission)}) || s.Residue.MissionLost || s.Residue.FogVisible == nil {
		return errSavingUnavailable
	}
	if err := s.second.validate(); err != nil {
		return err
	}
	if s.Mission != 0 {
		var w sim.World
		if err := w.UnmarshalBinary(s.World); err != nil {
			return err
		}
		if w.Script().Dialect() != sim.ScriptROM2 || w.Outcome() != sim.OutcomeUndecided {
			return errSavingUnavailable
		}
	}
	return nil
}

// refuseCompleted admits every save exportable admitted: the second campaign
// has no chapter record to complete.
func (secondCampaignRules) refuseCompleted(*FrontEnd, Snapshot) error { return nil }

func (secondCampaignRules) campaignProjection(s Snapshot) (sav.CampaignProjection, bool) {
	if s.second == nil {
		return sav.CampaignProjection{}, false
	}
	return sav.CampaignProjection{Main: sav.CampaignRecord{Mission: uint32(s.Mission)}, SelectedMission: uint32(s.Mission), AutoGetMission: ^uint32(0)}, true
}

func (secondCampaignRules) validateSession(s *currentSessionData) error {
	if s.Second == nil {
		return fmt.Errorf("current save game and campaign disagree")
	}
	if err := s.Second.validate(); err != nil {
		return err
	}
	if s.Difficulty != nil && (s.Second.Current.Kind != 2 || *s.Difficulty < 0 || *s.Difficulty > mapload.DifficultyHard) {
		return fmt.Errorf("invalid current city difficulty")
	}
	return nil
}

func (secondCampaignRules) validateAuthored(doc *sav.DocumentData, a *currentActionData) error {
	if doc.Head.Mission == 0 {
		if a.Session.Difficulty != nil {
			native, err := campaignDifficulty(int64(*a.Session.Difficulty))
			if err != nil || uint32(native) != doc.Head.Difficulty {
				return fmt.Errorf("ROM2 city difficulty disagrees with its header")
			}
		}
		if doc.World != nil || a.Session.Second == nil || a.Session.Second.validateTown() != nil || a.Program != nil || a.Policy != nil || a.Fog != nil || a.Pending != nil || len(a.PendingMessages) != 0 || len(a.Actions.Actors) != 0 || len(a.Held) != 0 || len(a.Bolts) != 0 || len(a.Heals) != 0 || len(a.Runs) != 0 || len(a.Options) != 0 || len(a.Animation) != 0 || len(a.DeathAges) != 0 || a.Manifest != nil || len(a.TerminalMotions) != 0 || len(a.VisualIdentities) != 0 || a.VisualNext != 0 || a.WorldMapReturn != nil || a.Session.MissionGold != nil || len(a.Party) == 0 || len(a.Roster) != 0 {
			return fmt.Errorf("ROM2 save lacks its current town continuation")
		}
		for _, p := range a.Party {
			if p.City == nil || p.Policy == nil || p.Base == nil {
				return fmt.Errorf("ROM2 town lacks complete current city party")
			}
		}
	} else if !secondSaveMission(int(doc.Head.Mission)) || doc.World == nil || a.Session.Second == nil || a.Session.Second.Current != (currentSecondLocation{1, int(doc.Head.Mission)}) || a.Session.Second.validate() != nil || a.Program == nil || a.Program.Dialect != sim.ScriptROM2 || a.Policy == nil || a.Policy.ROM2 == nil || a.Fog == nil {
		return fmt.Errorf("ROM2 save lacks its current mission continuation")
	}
	return nil
}

func (secondCampaignRules) undecodedSave(err error) error {
	return fmt.Errorf("ROM2 LOAD requires an authored current SAV: %w", err)
}

func (secondCampaignRules) statesCampaign(s *currentSessionData) bool {
	return s != nil && s.Second != nil
}

func (secondCampaignRules) canEnter(town *Town, n int) bool {
	return town == nil || town.second == nil || town.second.canEnter(n)
}

func (secondCampaignRules) enterMission(town *Town, n int) {
	if town != nil && town.second != nil {
		town.second.current = secondLocation{kind: 1, id: n}
	}
}

func (secondCampaignRules) seatWorld(r missionRequest, ms *Mission) error {
	if r.fresh() && r.town != nil && r.town.second != nil {
		bank := r.town.second.bank
		clear(bank[752:768])
		if !ms.World.SetROM2ScenarioState(bank) {
			return fmt.Errorf("campaign mission has no ROM2 scenario bank")
		}
	}
	return nil
}

func (r secondCampaignRules) selectedMarkers(f *FrontEnd, src *originalSource) (map[int]bool, error) {
	if src.campaign.town.second == nil {
		return r.firstCampaignRules.selectedMarkers(f, src)
	}
	return nil, nil
}

func (secondCampaignRules) checkTownLoad(f *FrontEnd, src *originalSource) error {
	if src.campaign.town.second != nil {
		if payload, err := readSecondTownTalk(&f.InstallResources, "npc517talk10"); err != nil || payload == nil {
			if err == nil {
				err = fmt.Errorf("initial inn conversation is unavailable")
			}
			return err
		}
	}
	return nil
}

func (r secondCampaignRules) arriveLoaded(f *FrontEnd, c *restoreCandidate) {
	if c.town.second == nil {
		r.firstCampaignRules.arriveLoaded(f, c)
	}
}

func (secondCampaignRules) keepsTownSurface(c *restoreCandidate) bool {
	return c.town != nil && c.town.second != nil
}

func (secondCampaignRules) beforeAdvance(f *FrontEnd, n int, ms *Mission, action ui.NoticeAction) string {
	town, live := f.Town, f.live
	if town == nil || town.second == nil || live == nil || live.mission == nil {
		return ""
	}
	m := live.mission
	completion := m.kind == ui.NoticeSuccess &&
		(action == ui.NoticeAdvance && m.open || action == ui.NoticeVictory && (m.open || m.delayedVictory && !m.victoryTaken))
	if completion {
		if err := town.second.finish(n, ms.World); err != nil {
			return err.Error()
		}
	}
	return ""
}

func (r secondCampaignRules) finishWon(f *FrontEnd, n int, ms *Mission) (int, string) {
	town := f.Town
	if town == nil || town.second == nil {
		return r.firstCampaignRules.finishWon(f, n, ms)
	}
	if err := town.second.finish(n, ms.World); err != nil {
		return -1, err.Error()
	}
	if refused := f.CampaignSession.carryMissionHome(f.townInstall(), n, ms.Party, ms.World, ms.Start.IDs, ms.Start.Roster); refused != "" {
		return -1, refused
	}
	// The output reads the bank before the case stores (R2-ENGINE-148).
	output := town.second.complete(ms.World)
	if live := f.live; live != nil && live.view != nil {
		movie := ""
		if in := f.Archives; in != nil && in.Containers != nil {
			movie = secondGameCompletionDirectory(in.Containers, output)
		}
		live.view.SetCompletionCutscene(movie)
	}
	town.won[n] = true
	f.Offered = 0
	for _, l := range town.second.available {
		if l.kind == 1 {
			f.Offered = l.id
			break
		}
	}
	return 0, ""
}

func (r secondCampaignRules) route(f *FrontEnd, n, successor int) winRoute {
	if f.Town == nil || f.Town.second == nil {
		return r.firstCampaignRules.route(f, n, successor)
	}
	if successor < 0 {
		return winStay
	}
	return winTown
}

func (r secondCampaignRules) returnToTown(f *FrontEnd, n int) {
	if f.Town == nil || f.Town.second == nil {
		r.firstCampaignRules.returnToTown(f, n)
		return
	}
	releaseWorldAudio(f.runtimeAudio(), f.endLive())
}

// openMission leaves the completion movie unset: the departure output
// selects it at acknowledgement.
func (secondCampaignRules) openMission(v *ui.Viewer) { v.SetCompletionCutscene("") }

func (secondCampaignRules) loadMissionText(mw *mapWorld, src entrySource, ms *Mission, t *mapload.Table) {
	mw.mission.objectiveLabels = secondGameObjectiveLabels(src, ms.Number)
	mw.mission.npcKeys = secondGameNPCKeys(ms, t)
	mw.mission.failureText = secondGameFailureText(src, ms.Number)
}

func (secondCampaignRules) scriptMessages(mw *mapWorld, messages []int32) {
	mw.observeSecondGameMessages(messages)
}

func (secondCampaignRules) settleNotices(mw *mapWorld) bool {
	mw.settleSecondGameNotices()
	return true
}

func (secondCampaignRules) acknowledgeNotice(mw *mapWorld) bool {
	mw.closeNotice()
	mw.settleSecondGameNotices()
	return true
}

func (secondCampaignRules) outcomeText(mw *mapWorld, body string) string {
	m := mw.mission
	if m.outcome == sim.OutcomeLost {
		_, reason := mw.world.ScriptCounters()
		if reason >= 2 && uint64(reason-2) < uint64(len(m.failureText)) && m.failureText[reason-2] != "" {
			body = m.failureText[reason-2]
		}
	}
	return body
}

func (secondCampaignRules) eventAudience(mw *mapWorld) (EventAudience, bool) {
	return mw.secondGameAudience(), true
}

func (secondCampaignRules) objectives(mw *mapWorld, briefing string) string {
	return mw.secondGameObjectivePanel(briefing)
}

func (secondCampaignRules) generatorPresets(f *FrontEnd, setup *ui.ChargenSetup) {}

func (secondCampaignRules) generatorParty(f *FrontEnd, res ui.ChargenResult) []mapload.PartyMember {
	return firstGeneratorParty(f, res)
}

func (secondCampaignRules) generatorBegin(f *FrontEnd, mission int) func(ui.ChargenResult) (ui.MapOpener, error) {
	return nil
}
