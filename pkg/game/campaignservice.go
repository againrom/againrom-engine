package game

import (
	"errors"

	"againrom/pkg/base"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// campaignService is the campaign behaviour the two games differ in and that
// is not a value of the profile's edition. A session reads it from the
// profile; it is never re-derived from the install or from which campaign
// state happens to exist. Each implementation is stateless: the state it
// reads and writes is the session's own.
type campaignService interface {
	// townAdapter supplies the common host's campaign state.
	townAdapter(t *townScreen) townCampaignAdapter
	// place is where the session's party stands: a town or a mission.
	place(s *CampaignSession) partyPlace
	// townSavePoint refuses a save taken off the map when the town is not
	// at a save point.
	townSavePoint(f *FrontEnd) error
	// refusals is the game's save point refusal table.
	refusals() *refusalTable
	// captureMission records the mission-only state a save taken on the map
	// carries for this campaign.
	captureMission(f *FrontEnd, s *Snapshot)
	// exportable refuses a captured session this campaign cannot write.
	exportable(s Snapshot) error
	// refuseCompleted refuses a town save of a campaign already completed.
	refuseCompleted(f *FrontEnd, s Snapshot) error
	// saveForm writes a SAVE's engine document in the records the game's
	// original reads, by the game's saved form row of the document's kind;
	// false when the form is the identity form.
	saveForm(f *FrontEnd, s Snapshot, doc sav.DocumentData) (sav.DocumentData, bool, error)
	// recordProfile is the campaign's part of the campaign record a SAV
	// carries: a record it states itself, that record's home word and the
	// offer label source.
	recordProfile() campaignRecordProfile
	// validateSession checks the campaign part of a decoded session record.
	validateSession(s *currentSessionData) error
	// validateAuthored checks a save the engine wrote for this campaign.
	validateAuthored(doc *sav.DocumentData, a *currentActionData) error
	// undecodedSave is the refusal of a save whose engine document does not
	// decode; nil admits it as an original save.
	undecodedSave(err error) error
	// carryHall gives the campaign state a save is written from the
	// snapshot's Hall words, where the game's SAV keeps them.
	carryHall(s *Snapshot)
	// statesCampaign reports whether a decoded session record states the
	// campaign itself rather than through the save's chapter record.
	statesCampaign(s *currentSessionData) bool
	// canEnter reports whether mission n is a destination the town offers.
	canEnter(town *Town, n int) bool
	// enterMission records that mission n is under way.
	enterMission(town *Town, n int)
	// seatWorld gives a fresh mission's world the campaign's own state.
	seatWorld(r missionRequest, ms *Mission) error
	// selectedMarkers is the world-map record a loaded save restores.
	selectedMarkers(f *FrontEnd, src *originalSource) (map[int]bool, error)
	// arriveLoaded enters the town a loaded town save describes.
	arriveLoaded(f *FrontEnd, c *restoreCandidate)
	// keepsTownSurface reports whether a load leaves the town screen as it is.
	keepsTownSurface(c *restoreCandidate) bool
	// keepsTownActorState reports whether a member the mission's cull kept
	// brings that actor's state to town. The second game's claims do not
	// state the cull; in both games a town SAVE writes the actor state and
	// Player colour a LOAD read, else the town constructor's.
	keepsTownActorState() bool
	// beforeAdvance refuses a mission's acknowledged end before it advances;
	// what follows a win is the one finisher's (finishWin).
	beforeAdvance(f *FrontEnd, n int, ms *Mission, action ui.NoticeAction) string
	// bankWin reports whether a win, through the mission seam when seam
	// holds, takes the finisher's bank row set.
	bankWin(f *FrontEnd, seam bool) bool
	// The mission driver's campaign behaviour: the movie a departure picks,
	// and the speaker audience.
	// The per-mission text, the outcome text and the objectives panel are
	// gameFiles.objectives rows.
	openMission(v *ui.Viewer)
	messagePolicy() missionMessagePolicy
	eventAudience(mw *mapWorld) (EventAudience, bool)
	// The chat command adapter, and whether a mission opened from town runs
	// in the campaign that adapter admits commands in.
	chat() *chatAdapter
	chatCampaign(town *Town) bool

	// files are the readers of the installed files the two games lay out
	// differently.
	files() *gameFiles
	// hires is the hire state town's mission entry reads: its pools, and
	// the campaign's bank where the game keeps one.
	hires(town *Town, table *mapload.Table) hireState
	// routeStages is the route witness's step table.
	routeStages() []routeStage
	// censusRefusal refuses a census of the campaign's maps: the census
	// reports each map of a destination bank.
	censusRefusal() error
	// testPlayRefusal refuses the map editor's test play, which plays a map
	// as a guest mission after a won chapter mission.
	testPlayRefusal() error
	// testPlayParty are the companions the party window before a test play
	// offers beside the hero, in the window's order.
	testPlayParty() []testPlayCompanion
	// eventTags is how the game's event text tag bodies are read.
	eventTags() eventTags

	// newGameLimit states where a new game opens on profile p when p has no
	// character generation.
	newGameLimit(p base.Profile) string
	// captureDifficulty gives a save s taken off the map the difficulty the
	// campaign records there: the second game's, the session's own.
	captureDifficulty(f *FrontEnd, s *Snapshot, onMap bool)
	// recordDifficulty writes the difficulty a save of s carries in its
	// session record: the second game's town save, its own; the first's, none.
	recordDifficulty(s Snapshot, record *currentSessionData)
	// difficultyRecord is the campaign record word a save at difficulty d
	// carries: the first game's, d minus one; the second's keeps its word.
	difficultyRecord(d mapload.Difficulty) (uint32, bool)
	// ownsGame refuses a save whose engine record names a game other than
	// this campaign's.
	ownsGame(g base.Game) error
	// fileState is what an original save file of this game states in its
	// own records, read before any state changes; the first game's files
	// state no session (fileCampaign).
	fileState(sf *sav.File) (originalFileState, error)
}

// originalFileState is what an original save file states in its own
// records: the quick-spell bindings and, for a file whose game keeps the
// session in its campaign tail rather than in an engine document, that
// session.
type originalFileState struct {
	quickSpells [9]uint32
	session     *currentSessionData
	// fileView is that a mission file without a complete document gives the
	// first frame from its own state store.
	fileView bool
}

// fileCampaign is the campaign service of the game sf's envelope names;
// the Asg& envelope names none and is read as the first game's.
func fileCampaign(sf *sav.File) campaignService {
	return campaignOf(base.Game(sf.EnvelopeGame()))
}

// errOtherGame is a campaign service's refusal of a game not its own.
var errOtherGame = errors.New("the game is not the campaign's")

// campaignOf is the campaign service of game g. It is the one place a service
// implementation is picked.
func campaignOf(g base.Game) campaignService {
	if g.Edition().Campaign == base.CampaignDestinations {
		return secondCampaignRules{}
	}
	return firstCampaignRules{}
}

// campaign is the campaign service of the profile the install was detected as.
func (f *FrontEnd) campaign() campaignService { return campaignOf(f.Base().Profile.GameOf()) }

// campaign is the campaign service of the game the mission's table belongs to.
func (m *missionNotices) campaign() campaignService {
	if m == nil {
		return campaignOf("")
	}
	return tableCampaign(m.table)
}

// files are the file readers of the game the mission's table belongs to.
func (m *missionNotices) files() *gameFiles { return m.campaign().files() }

// sessionCampaign is the campaign service of the game a decoded session
// record names; no record is the first game's.
func sessionCampaign(s *currentSessionData) campaignService {
	if s == nil {
		return campaignOf("")
	}
	return campaignOf(s.Game)
}

// edition is the edition of the game the mission's table belongs to.
func (m *missionNotices) edition() base.Edition {
	if m == nil {
		return base.Game("").Edition()
	}
	return tableEdition(m.table)
}

// files are the file readers of the game the mission's table belongs to.
func (mw *mapWorld) files() *gameFiles { return mw.mission.files() }
