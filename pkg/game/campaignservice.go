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
	// townScreen is the screen the town is shown on.
	townScreen(f *FrontEnd) ui.TownScreen
	// townSavePoint refuses a save taken off the map when the town is not
	// at a save point.
	townSavePoint(f *FrontEnd) error
	// captureMission records the mission-only state a save taken on the map
	// carries for this campaign.
	captureMission(f *FrontEnd, s *Snapshot)
	// exportable refuses a captured session this campaign cannot write.
	exportable(s Snapshot) error
	// refuseCompleted refuses a town save of a campaign already completed.
	refuseCompleted(f *FrontEnd, s Snapshot) error
	// campaignProjection is the campaign record a save of s carries when the
	// campaign states it itself rather than through its chapters.
	campaignProjection(s Snapshot) (sav.CampaignProjection, bool)
	// validateSession checks the campaign part of a decoded session record.
	validateSession(s *currentSessionData) error
	// validateAuthored checks a save the engine wrote for this campaign.
	validateAuthored(doc *sav.DocumentData, a *currentActionData) error
	// undecodedSave is the refusal of a save whose engine document does not
	// decode; nil admits it as an original save.
	undecodedSave(err error) error
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
	// checkTownLoad refuses a town save the install cannot continue.
	checkTownLoad(f *FrontEnd, src *originalSource) error
	// arriveLoaded enters the town a loaded town save describes.
	arriveLoaded(f *FrontEnd, c *restoreCandidate)
	// keepsTownSurface reports whether a load leaves the town screen as it is.
	keepsTownSurface(c *restoreCandidate) bool
	// The four steps of a mission's acknowledged end.
	beforeAdvance(f *FrontEnd, n int, ms *Mission, action ui.NoticeAction) string
	finishWon(f *FrontEnd, n int, ms *Mission) (int, string)
	route(f *FrontEnd, n, successor int) winRoute
	returnToTown(f *FrontEnd, n int)
	// The mission driver's campaign behaviour: the movie a departure picks,
	// the per-mission text, the script message queue, the notice flow, the
	// outcome text, the speaker audience and the objectives panel.
	openMission(v *ui.Viewer)
	loadMissionText(mw *mapWorld, src entrySource, ms *Mission, t *mapload.Table)
	scriptMessages(mw *mapWorld, messages []int32)
	settleNotices(mw *mapWorld) bool
	acknowledgeNotice(mw *mapWorld) bool
	outcomeText(mw *mapWorld, body string) string
	eventAudience(mw *mapWorld) (EventAudience, bool)
	objectives(mw *mapWorld, briefing string) string
	// The character generator's campaign hooks: what the setup takes from the
	// campaign (the pictures' statistic presets), the party a result makes,
	// and what an accepted result opens or commits.
	generatorSetup(f *FrontEnd, setup *ui.ChargenSetup)
	generatorParty(f *FrontEnd, res ui.ChargenResult) []mapload.PartyMember
	generatorBegin(f *FrontEnd, mission int) func(ui.ChargenResult) (ui.MapOpener, error)

	// The chat command adapter, and whether a mission opened from town runs
	// in the campaign that adapter admits commands in.
	chat() *chatAdapter
	chatCampaign(town *Town) bool

	// files are the readers of the installed files the two games lay out
	// differently.
	files() *gameFiles
	// censusRefusal refuses a census of the campaign's maps: the census
	// reports each map of a destination bank.
	censusRefusal() error
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
	// ownsGame refuses a save whose engine record names a game other than
	// this campaign's.
	ownsGame(g base.Game) error
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
