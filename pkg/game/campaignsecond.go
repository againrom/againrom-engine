package game

import (
	"errors"
	"fmt"

	"againrom/pkg/base"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/random"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondCampaignRules is the second game's campaign service: a scenario bank
// that names the towns and missions the party may travel to, its save
// points and the script's message queue. A session that
// holds no second-campaign state yet keeps the first game's behaviour, which
// the embedded rules give.
type secondCampaignRules struct{ firstCampaignRules }

func (secondCampaignRules) townAdapter(t *townScreen) townCampaignAdapter {
	return destinationTownCampaign{t}
}

// townSavePoint refuses a town that holds an open talk, the world map (the
// party has left every place) and a town the engine does not save yet.
func (secondCampaignRules) townSavePoint(f *FrontEnd) error {
	var c *secondCampaign
	if f.Town != nil {
		c = f.Town.second
	}
	switch {
	case c.savePoint():
		return nil
	case c.noticePending():
		return refusedTalk
	case c != nil && c.current == (partyPlace{}):
		return refusedWorldMap
	}
	return refusedHere
}

func (secondCampaignRules) captureMission(f *FrontEnd, s *Snapshot) {
	if f.live.view != nil {
		animation := f.live.view.SaveAnimation()
		s.mapAnimation = &animation
		s.mapMotion = captureCurrentMapMotion(f.live)
	}
	m := f.live.mission
	s.noticeOpen = m != nil && m.open
	s.victoryContinued = m != nil && m.delayedVictory && !m.victoryTaken && m.outcome == sim.OutcomeWon
}

// exportable admits a quiet town and a mission map without a standing
// message. A won mission is admitted once the player answered its panel
// with Continue (owner direction, DIV-4298); a lost one never is.
func (secondCampaignRules) exportable(s Snapshot) error {
	if s.second == nil {
		return refusedHere
	}
	if s.Mission == 0 {
		if s.noticeOpen {
			return refusedTalk
		}
		if s.second.validateTown() != nil || len(s.World) != 0 || len(s.Party) == 0 || s.Residue.hasMissionState() {
			return refusedHere
		}
		return s.second.validate()
	}
	var w sim.World
	if err := w.UnmarshalBinary(s.World); err != nil {
		return err
	}
	switch {
	case s.Residue.MissionLost || w.Outcome() == sim.OutcomeLost:
		return refusedLost
	case s.noticeOpen || w.Outcome() == sim.OutcomeWon && !s.victoryContinued:
		return refusedNotice
	case !secondSaveMission(int(s.Mission)) || s.second.Current != (currentSecondLocation{1, int(s.Mission)}) || s.Residue.FogVisible == nil || w.Script().Dialect() != sim.ScriptROM2:
		return refusedHere
	}
	return s.second.validate()
}

// refuseCompleted admits every save exportable admitted: the second campaign
// has no chapter record to complete.
func (secondCampaignRules) refuseCompleted(*FrontEnd, Snapshot) error { return nil }

// recordProfile: the bank states the record of a session that holds the
// destinations campaign; its town record carries its own home word, and a
// SAVE writes the offer labels as LOAD stated them.
func (secondCampaignRules) recordProfile() campaignRecordProfile {
	return campaignRecordProfile{stated: secondStatedRecord,
		statedHome:  [campaignRecordKinds]campaignHome{townRecord: homeTown},
		offerLabels: offerLabelsAsLoaded}
}

func secondStatedRecord(s Snapshot) (sav.CampaignProjection, bool) {
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

// place is the destinations campaign's current place; a session that holds
// no such campaign yet answers as the first game.
func (r secondCampaignRules) place(s *CampaignSession) partyPlace {
	if s != nil && s.Town != nil && s.Town.second != nil {
		return s.Town.second.current
	}
	return r.firstCampaignRules.place(s)
}

func (secondCampaignRules) canEnter(town *Town, n int) bool {
	return town == nil || town.second == nil || town.second.canEnter(n)
}

func (secondCampaignRules) missionParty(party []mapload.PartyMember, town *Town, table *mapload.Table) ([]mapload.PartyMember, error) {
	if town == nil || town.second == nil {
		return party, nil
	}
	c := town.second
	out := make([]mapload.PartyMember, 0, len(party)+secondHireSlots)
	seen := [secondHireSlots + 1]bool{}
	for _, p := range party {
		id := p.CampaignHireID()
		if id != 0 {
			if seen[id] || c.bank[511+id] == 0 || c.bank[531+id] == 0 {
				continue
			}
			seen[id] = true
		}
		out = append(out, p)
	}
	for id := 1; id <= secondHireSlots; id++ {
		if seen[id] || c.bank[511+id] == 0 || c.bank[531+id] == 0 {
			continue
		}
		p, ok := mapload.CampaignActor(table, mapload.CampaignActorSpec{Profile: keyedCampaignActor, Identity: id, Stage: int(c.bank[768]), Hire: true}, nil)
		if !ok {
			return nil, fmt.Errorf("campaign hire %d at stage %d has no actor template", id, c.bank[768])
		}
		out = append(out, p)
	}
	return mapload.OwnParty(out), nil
}

func (secondCampaignRules) enterMission(town *Town, n int) {
	if town != nil && town.second != nil {
		town.second.current, town.second.mapOpen = partyPlace{kind: placeMission, id: n}, nil
	}
}

func (secondCampaignRules) seatWorld(r missionRequest, ms *Mission) error {
	// A fresh mission clears its span of the campaign bank; an original
	// mission file's bank is the live one it saved and is seated unchanged.
	// An engine document's policy record has already restored the live
	// bank, which the campaign's copy of mission entry must not replace; the
	// World's loaded-record carrier names that LOAD.
	if r.snap == nil && r.town != nil && r.town.second != nil && (r.fresh() || ms.World.LoadedRecord() != sim.LoadedDocument) {
		bank := r.town.second.bank
		if r.fresh() {
			clear(bank[752:768]) // R2-ENGINE-048
		}
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

// arriveLoaded enters a loaded second-game town with its shop stock drawn
// anew from the current records before the shop can be entered: a SAV
// carries no stock (DIV-3045). A first-game town arrives as the first game's.
func (r secondCampaignRules) arriveLoaded(f *FrontEnd, c *restoreCandidate) {
	second := c.town.second
	if second == nil {
		r.firstCampaignRules.arriveLoaded(f, c)
		return
	}
	tables, ok := secondShopTables(f.Table)
	if ok {
		second.adoptLegacyHero(tables)
	}
	if second.current.kind != placeTown {
		return
	}
	if ok {
		second.shop.restock(tables, second.current.id, f.randomService().Stream(random.ShopStock).Raw)
	}
}

func (secondCampaignRules) keepsTownSurface(c *restoreCandidate) bool {
	return c.town != nil && c.town.second != nil
}

func (secondCampaignRules) keepsTownActorState() bool { return false }

func (secondCampaignRules) beforeAdvance(f *FrontEnd, n int, ms *Mission, action ui.NoticeAction) string {
	town, live := f.Town, f.live
	if town == nil || town.second == nil || live == nil || live.mission == nil {
		return ""
	}
	m := live.mission
	completion := m.kind == ui.NoticeSuccess &&
		(action == ui.NoticeAdvance && m.open || action == ui.NoticeVictory && (m.open || m.delayedVictory && !m.victoryTaken))
	if completion {
		if err := town.second.finish(n, ms.World, frontTransitions{f}.won(n, ms)); err != nil {
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
	if err := town.second.finish(n, ms.World, frontTransitions{f}.won(n, ms)); err != nil {
		return -1, err.Error()
	}
	if refused := f.CampaignSession.carryMissionHome(f.townInstall(), n, ms.Party, ms.World, ms.Start.IDs, ms.Start.Roster); refused != "" {
		return -1, refused
	}
	// The output reads the bank before the case stores (R2-ENGINE-148).
	output := town.second.complete(ms.World)
	f.CampaignSession.leftLocation(f.townInstall())
	if live := f.live; live != nil && live.view != nil {
		movie := ""
		if in := f.Archives; in != nil && in.Containers != nil {
			movie = secondGameCompletionDirectory(in.Containers, InstallTextCode(in.Containers, in.Game().Edition()), output)
		}
		live.view.SetCompletionCutscene(movie)
	}
	town.won[n] = true
	f.Offered = 0
	for _, l := range town.second.available {
		if l.kind == placeMission {
			f.Offered = l.id
			break
		}
	}
	return town.second.enteredWithoutMap(), ""
}

func (r secondCampaignRules) route(f *FrontEnd, n, successor int) winRoute {
	if f.Town == nil || f.Town.second == nil {
		return r.firstCampaignRules.route(f, n, successor)
	}
	if successor < 0 {
		return winStay
	}
	if successor > 0 {
		return winMission
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
	code := InstallTextCode(src, tableEdition(t))
	mw.mission.objectiveLabels = secondGameObjectiveLabels(src, code, ms.Number)
	mw.mission.npcKeys = secondGameNPCKeys(ms, t)
	mw.mission.failureText = secondGameFailureText(src, code, ms.Number)
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
	// The loss shows failure entry reason - 2 (R2-ENGINE-629); a reason
	// outside the entries keeps the generic body (DIV-4223).
	if r := int64(m.lossReason) - 2; m.outcome == sim.OutcomeLost && r >= 0 && r < int64(len(m.failureText)) && m.failureText[r] != "" {
		body = m.failureText[r]
	}
	return body
}

func (secondCampaignRules) eventAudience(mw *mapWorld) (EventAudience, bool) {
	return mw.secondGameAudience(), true
}

func (secondCampaignRules) objectives(mw *mapWorld, briefing string) string {
	return mw.secondGameObjectivePanel(briefing)
}

func (secondCampaignRules) generatorSetup(f *FrontEnd, setup *ui.ChargenSetup) {
	secondGeneratorSetup(f, setup)
}

func (secondCampaignRules) generatorParty(f *FrontEnd, res ui.ChargenResult) []mapload.PartyMember {
	return secondGeneratorParty(f, res)
}

// generatorBegin commits the new campaign and opens no map: the accepted
// generator shows the first town (R2-ENGINE-283).
func (secondCampaignRules) generatorBegin(f *FrontEnd, _ int) func(ui.ChargenResult) (ui.MapOpener, error) {
	return func(res ui.ChargenResult) (ui.MapOpener, error) { return nil, f.startSecondGenerated(res) }
}

func (secondCampaignRules) chat() *chatAdapter { return &secondChat }

// chatCampaign: a mission opened while the town holds the destinations
// campaign runs in campaign mode (R2-ENGINE-296).
func (secondCampaignRules) chatCampaign(town *Town) bool { return town != nil && town.second != nil }

func (secondCampaignRules) files() *gameFiles { return &secondGameFiles }

func (secondCampaignRules) eventTags() eventTags { return secondEventTags{} }

func (secondCampaignRules) newGameLimit(base.Profile) string {
	return "native character selection is unavailable; new game opens the initial campaign town with the default hero"
}

func (secondCampaignRules) captureDifficulty(f *FrontEnd, s *Snapshot, onMap bool) {
	if !onMap {
		s.Difficulty = f.Difficulty
	}
}

func (secondCampaignRules) recordDifficulty(s Snapshot, record *currentSessionData) {
	if s.Mission == 0 {
		difficulty := s.Difficulty
		record.Difficulty = &difficulty
	}
}

func (secondCampaignRules) difficultyRecord(mapload.Difficulty) (uint32, bool) { return 0, false }

// fileState reads a Bsg& file's session from the campaign tail; an empty
// shortcut array binds no cell (DIV-4153, DIV-4154). A Token in the list
// after the Sacks has no reader and refuses the file before any state
// changes (DIV-4172).
func (secondCampaignRules) fileState(sf *sav.File) (originalFileState, error) {
	if n, err := sf.ExtraTokens(); err != nil {
		return originalFileState{}, err
	} else if n != 0 {
		return originalFileState{}, fmt.Errorf("this save's world holds %d Tokens after its Sacks, which this version does not read", n)
	}
	quick, err := fileQuickSpells(sf, true)
	if err != nil {
		return originalFileState{}, err
	}
	session, err := secondFileSession(sf)
	if err != nil {
		return originalFileState{}, err
	}
	session.Game = base.GameROM2
	return originalFileState{quickSpells: quick, session: session, fileView: true}, nil
}

func (secondCampaignRules) ownsGame(g base.Game) error {
	if base.SameGame(g, base.GameROM2) {
		return nil
	}
	return errOtherGame
}

func (secondCampaignRules) censusRefusal() error { return nil }

func (secondCampaignRules) testPlayRefusal() error {
	return errors.New("Test play plays first-game maps only: this campaign has no guest missions")
}

type destinationTownCampaign struct{ t *townScreen }

func (destinationTownCampaign) chapter() int { return 0 }
func (c destinationTownCampaign) unavailableTowns() []int {
	s := c.bank()
	if s == nil {
		return nil
	}
	var ids []int
	for id := 2; id <= 3; id++ {
		if p := (partyPlace{kind: placeTown, id: id}); s.current == p || s.has(p) {
			ids = append(ids, id)
		}
	}
	return ids
}
func (c destinationTownCampaign) startCampaign() error {
	if _, err := c.t.in.Archives.Containers.ReadFile(mainPrefix + "text/town.txt"); err != nil {
		return fmt.Errorf("initial campaign town: %w", err)
	}
	return c.t.start()
}

func (c destinationTownCampaign) bank() *secondCampaign {
	if c.t.sess == nil || c.t.sess.Town == nil {
		return nil
	}
	s := c.t.sess.Town.second
	if s != nil {
		// The campaign binds the rows of a layered book; the shipped
		// book's are its default.
		s.offers = nil
		if b := c.book(); b != nil && b.catalog != secondBook().catalog {
			s.offers = b.offerRows()
		}
	}
	return s
}

// secondBook is the second game's shipped book: the book of a screen with
// no install, and the default of the campaign's inn rows.
func secondBook() *townBook { return townBooks["rom2"] }

// book is the mods' layered book when a layer applies, the profile's
// shipped book otherwise.
func (c destinationTownCampaign) book() *townBook {
	if c.t.in == nil || c.t.in.Archives == nil {
		return secondBook()
	}
	if layered := c.t.in.Campaign.Value().Towns; layered != nil && townBookOf(c.t.profile()) != nil &&
		layered.catalog.Catalog == townBookOf(c.t.profile()).catalog.Catalog {
		return layered
	}
	return townBookOf(c.t.profile())
}
func (c destinationTownCampaign) room() string {
	if s := c.bank(); s != nil {
		for name, r := range secondTownRooms {
			if s.room == r {
				return name
			}
		}
	}
	return ""
}
func (c destinationTownCampaign) setRoom(name string) {
	if s := c.bank(); s != nil {
		if r, ok := secondTownRooms[name]; ok {
			s.room = r
		}
	}
}
func (c destinationTownCampaign) townID() int {
	if s := c.bank(); s != nil && s.current.kind == placeTown {
		return s.current.id
	}
	return 0
}
func (c destinationTownCampaign) ready() bool   { return c.bank() != nil }
func (c destinationTownCampaign) present() bool { return c.townID() != 0 }
func (c destinationTownCampaign) talking() bool {
	s := c.bank()
	return s != nil && s.talk.payload != nil
}

func (c destinationTownCampaign) talkCursor() *talkCursor {
	if s := c.bank(); s != nil {
		return &s.talk
	}
	return nil
}

func (c destinationTownCampaign) endTalk() {
	s := c.bank()
	if s == nil {
		return
	}
	s.talk.close()
	if o := s.talking; o != nil {
		s.talking = nil
		c.t.endInnTalk(*o)
	}
}
func (c destinationTownCampaign) condition(name string) bool {
	if f := secondTownConditions[name]; f != nil && c.bank() != nil {
		return f(c.bank())
	}
	return false
}
func (c destinationTownCampaign) value(name string) int {
	if f := secondTownValues[name]; f != nil && c.bank() != nil {
		return f(c.bank())
	}
	return -1
}
func (c destinationTownCampaign) savePoint() bool { return c.bank().savePoint() }
func (c destinationTownCampaign) leave() {
	if s := c.bank(); s != nil {
		if s.leaveTown() {
			c.t.sess.leftLocation(c.t.in.townInstall())
		}
		s.room = secondTownSquare
	}
}

// SecondCampaignState is what a witness reads of the second campaign: the
// current place, the town room, the destinations and the hero's bag codes.
type SecondCampaignState struct {
	Kind, ID, Room int
	Available      [][2]int
	Bag            []uint16
}

// SecondState answers the second campaign's state, false outside one.
func (t *Town) SecondState() (SecondCampaignState, bool) {
	if t == nil || t.second == nil {
		return SecondCampaignState{}, false
	}
	c := t.second
	out := SecondCampaignState{Kind: c.current.kind, ID: c.current.id, Room: int(c.room)}
	for _, l := range c.available {
		out.Available = append(out.Available, [2]int{l.kind, l.id})
	}
	if len(c.party) != 0 {
		for _, it := range c.party[0].CarriedItems {
			out.Bag = append(out.Bag, it.Code)
		}
	}
	return out, true
}
