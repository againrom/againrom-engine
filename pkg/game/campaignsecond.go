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
	case f.Town.secondSavePoint():
		return nil
	case f.Town.secondNoticePending():
		return secondRefusals.refuse(refusedTalk)
	case c != nil && c.current == (partyPlace{}):
		return secondRefusals.refuse(refusedWorldMap)
	}
	return secondRefusals.refuse(refusedHere)
}

func (secondCampaignRules) refusals() *refusalTable { return secondRefusals }

func (secondCampaignRules) captureMission(f *FrontEnd, s *Snapshot) {
	if f.live.view != nil {
		animation := f.live.view.SaveAnimation()
		s.mapAnimation = &animation
		s.mapMotion = captureCurrentMapMotion(f.live)
	}
	if bank, ok := f.live.world.ROM2ScenarioState(); ok {
		s.liveBank = &bank
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
		return secondRefusals.refuse(refusedHere)
	}
	if s.Mission == 0 {
		if s.noticeOpen {
			return secondRefusals.refuse(refusedTalk)
		}
		if s.second.validateTown() != nil || len(s.World) != 0 || len(s.Party) == 0 || s.Residue.hasMissionState() {
			return secondRefusals.refuse(refusedHere)
		}
		return s.second.validate()
	}
	var w sim.World
	if err := w.UnmarshalBinary(s.World); err != nil {
		return err
	}
	switch {
	case s.Residue.MissionLost || w.Outcome() == sim.OutcomeLost:
		return secondRefusals.refuse(refusedLost)
	case s.noticeOpen || w.Outcome() == sim.OutcomeWon && !s.victoryContinued:
		return secondRefusals.refuse(refusedNotice)
	case !secondSaveMission(int(s.Mission)) || s.second.Current != (currentSecondLocation{1, int(s.Mission)}) || s.Residue.FogVisible == nil || w.Script().Dialect() != sim.ScriptROM2:
		return secondRefusals.refuse(refusedHere)
	}
	return s.second.validate()
}

// refuseCompleted refuses a town save at the bank's end stage, the campaign
// the ending leaves no save point after (DIV-906).
func (secondCampaignRules) refuseCompleted(f *FrontEnd, s Snapshot) error {
	if s.Mission == 0 && s.second != nil && f.Campaign.Value().endedAt(s.second.Bank[768]) {
		return secondRefusals.refuse(refusedCompleted)
	}
	return nil
}

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

func (secondCampaignRules) carryHall(s *Snapshot) {
	if s.second != nil && s.Fame != nil && s.Fame.Known {
		s.second.Kept = s.second.Kept.withHall(s.Fame.Time, s.Fame.Events)
	}
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
			enterLocation(&bank)
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
		second.shop.stockMultiplier = f.townInstall().stockMultiplier
		second.shop.drawStock(tables, second.current.id, f.randomService().Stream(random.ShopStock).Raw)
	}
}

func (secondCampaignRules) keepsTownSurface(c *restoreCandidate) bool {
	return c.town != nil && c.town.second != nil
}

func (secondCampaignRules) keepsTownActorState() bool { return false }

// bankWin holds only for a win through the seam on a town the bank stands for.
func (secondCampaignRules) bankWin(f *FrontEnd, seam bool) bool {
	return seam && f.Town != nil && f.Town.second != nil
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
		if err := town.second.finish(n, ms.World, frontTransitions{f}.won(n, ms)); err != nil {
			return err.Error()
		}
	}
	return ""
}

// openMission leaves the completion movie unset: the departure output
// selects it at acknowledgement.
func (secondCampaignRules) openMission(v *ui.Viewer) { v.SetCompletionCutscene("") }

func (secondCampaignRules) messagePolicy() missionMessagePolicy { return missionMessagePolicies[1] }

func (secondCampaignRules) eventAudience(mw *mapWorld) (EventAudience, bool) {
	return mw.secondGameAudience(), true
}

func (secondCampaignRules) chat() *chatAdapter { return &secondChat }

// chatCampaign: a mission opened while the town holds the destinations
// campaign runs in campaign mode (R2-ENGINE-296).
func (secondCampaignRules) chatCampaign(town *Town) bool { return town != nil && town.second != nil }

func (secondCampaignRules) files() *gameFiles { return &secondGameFiles }

// hires reads the campaign's bank when the town holds the campaign; a town
// without it enters as the first game's.
func (secondCampaignRules) hires(town *Town, table *mapload.Table) hireState {
	if town == nil || town.second == nil {
		return hireState{town: town, table: table}
	}
	s := town.second.hireState()
	s.town, s.table = town, table
	return s
}

func (secondCampaignRules) routeStages() []routeStage { return secondRouteStages }

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
// shortcut array binds no cell (DIV-4153, DIV-4154).
func (secondCampaignRules) fileState(sf *sav.File) (originalFileState, error) {
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
	if err := installText(c.t.in.Archives.Containers, townTextSource.at(), TextCode{}).err(); err != nil {
		return fmt.Errorf("initial campaign town: %w", err)
	}
	return c.t.start()
}

// townShop is the campaign, whose one shop stocks the party's town.
func (c destinationTownCampaign) townShop() *Shop {
	if c.t.sess == nil {
		return nil
	}
	if s := c.bank(); s != nil {
		return &s.shop
	}
	return nil
}

// innInput is the campaign's inn input: the inn's options and its hire
// options, which the options already carry.
func (c destinationTownCampaign) innInput(innProfile) (options, hires []innOffer) {
	s := c.bank()
	if s == nil {
		return nil, nil
	}
	return s.innOptions(), s.hires()
}

func (c destinationTownCampaign) bank() *secondCampaign {
	if c.t.sess == nil || c.t.sess.Town == nil {
		return nil
	}
	s := c.t.sess.Town.second
	if s != nil {
		// The rows bind once and share the bank; the Town keeps its own
		// row, so its ID stays the chapter town's.
		tw := c.t.sess.Town
		if b := c.book(); b != nil && tw.bankOffers != s {
			tw.resetTownStates(b.catalog.Towns, s)
		}
		if tw.bankOffers != nil {
			s = tw.bankOffers
		}
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
	return s != nil && c.t.talkPage().payload != nil
}

func (c destinationTownCampaign) talkSeated() bool { return c.bank() != nil }

func (c destinationTownCampaign) endTalk() {
	s := c.bank()
	if s == nil {
		return
	}
	c.t.talkPage().close()
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
func (c destinationTownCampaign) savePoint() bool {
	return c.bank() != nil && c.t.sess.Town.secondSavePoint()
}
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

// campaignStage is the bank's stage, zero on a town without a bank.
func (t *Town) campaignStage() int32 {
	if t == nil || t.second == nil {
		return 0
	}
	return t.second.bank[768]
}

// saveForm writes doc in the second game's saved form of its kind.
func (secondCampaignRules) saveForm(f *FrontEnd, s Snapshot, doc sav.DocumentData) (sav.DocumentData, bool, error) {
	return secondSavedForms[savedFormKind(s)].write(f, s, doc)
}
