package game

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/random"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// missionRequest is one mission entry: which mission, with whom, over which
// town, and whether it starts fresh, resumes a native save (snap) or resumes
// an original save (prepare).
type missionRequest struct {
	n            int
	party        []mapload.PartyMember
	snap         *Snapshot
	initialPurse *uint32
	prepare      func(*alm.Map) (*originalFog, error)
	activate     *func()
	units        *terrain.UnitSet
	difficulty   mapload.Difficulty
	town         *Town
}

// fresh is a mission started rather than resumed.
func (r missionRequest) fresh() bool { return r.snap == nil && r.prepare == nil }

// missionPorts are the services a mission entry reaches, each through the
// narrowest thing that names it, so the entry sequence is the same whichever
// implementation is supplied. No port is the FrontEnd or a view of it.
type missionPorts struct {
	// audio is everything the entry asks of sound.
	audio missionAudio
	// install is what the entry reads from the install.
	install missionInstall
	// profile is the player's stored game options the entry applies.
	profile missionProfile
	// session is the campaign state the entry commits to.
	session *CampaignSession
	// display is the viewer settings the entry copies onto the map screen.
	display missionDisplay
	// art resolves the install's words, fonts and pictures for the viewer.
	art func() viewerArt
	// cityBase reads the open town's city graph for the party, the one part
	// of a mission start that needs the whole game state.
	cityBase cityBaseFunc
	// advance wraps the driver's own advance seam with what a finished
	// mission does next: the party carry, the successor and the screen change.
	advance func(n int, ms *Mission, advance ui.MapAdvance) ui.MapAdvance
	// random is the session's random service; in original mode a fresh
	// mission's placement and World continue its shared stream.
	random *random.Service
	// campaign is the campaign service of the install's profile; none is the
	// first game's.
	campaign campaignService
}

// rules is the entry's campaign service.
func (p missionPorts) rules() campaignService {
	if p.campaign == nil {
		return firstCampaignRules{}
	}
	return p.campaign
}

// missionInstall is every read a mission entry makes of the install. The
// production implementation is installMission.
type missionInstall interface {
	// missionTable is the definition table the mission is started over.
	missionTable() *mapload.Table
	constructCurrentBuildings(ms *Mission) error
	// decodeMission reads and decodes the map at addr into a viewer.
	decodeMission(addr string) (*MapView, error)
	// nameJoinedRoster resolves the displayed names and bodies of the roster
	// a map supplies.
	nameJoinedRoster(ms *Mission)
	// loadPartyBodies loads every party member's body art.
	loadPartyBodies(ms *Mission, units *terrain.UnitSet)
	// openMissionDriver builds the mission's driver over the started mission.
	openMissionDriver(ms *Mission, units *terrain.UnitSet, v *ui.Viewer) *mapWorld
	// missionProjectiles is the spell art bundle the driver draws with.
	missionProjectiles() *terrain.EffectSet
	// missionBriefing is the objectives text of mission n.
	missionBriefing(n int) string
}

// missionProfile is the player's stored game options a mission entry applies.
// The production implementation is PersistenceContext.
type missionProfile interface {
	applyFreshGameOptions(mw *mapWorld, incoming []mapload.PartyMember) error
	cycleRetreat(mw *mapWorld)
}

// missionDisplay is the viewer settings a mission entry copies onto the map
// screen.
type missionDisplay struct {
	pathfinding        bool
	graphics           ui.GraphicsOptions
	textSmoothing      bool
	frameSmoothing     bool
	deterministicFrame bool
	chicken            bool
}

// enterMission builds the complete map screen for one mission and returns the
// seams the map screen runs it through. It performs every fallible step and
// commits nothing itself: the live-driver assignment goes to r.activate when
// that is set, and runs here otherwise. The sequence is admit, decode, audio,
// start, seat the world, dress the viewer, open the driver, settle what a
// resume restores, commit, and install the seams.
func enterMission(r missionRequest, ports missionPorts) (preparedMap, error) {
	addr, diff, err := admitMission(ports.rules(), r.town, r.n, !r.fresh(), r.difficulty)
	if err != nil {
		return preparedMap{}, err
	}
	mv, err := ports.install.decodeMission(addr)
	if err != nil {
		return preparedMap{}, err
	}
	ports.audio.attach(mv.Viewer)
	audioReady := false
	defer func() {
		if !audioReady {
			ports.audio.release(mv.Viewer)
		}
	}()
	ms, fog, err := startMission(r, mv, addr, diff, ports)
	if err != nil {
		return preparedMap{}, err
	}
	err = seatWorld(r, ports.rules(), ports.install.missionTable(), ms, mv.Viewer)
	if err != nil {
		return preparedMap{}, err
	}
	mv.Viewer.SetMissionCutscene(r.n)
	// The install words, fonts and pictures reach the viewer before openMission
	// builds the initial inventory descriptions. ui.flow.enter writes the words
	// again when an App adopts the viewer, but MissionOpener is also a
	// production headless seam and its returned viewer and mapWorld must not
	// expose authored English in that interval.
	ports.art().apply(mv.Viewer)

	// The view opens on the party rather than on the map's top-left corner. It
	// is ARMED here and applied when a view size is first adopted, because the
	// extent is a fraction of a window nobody has yet measured — this door
	// runs before the run loop starts.
	mv.Viewer.SetStartView(startViewCell(ms.Start))
	if r.fresh() {
		mv.Viewer.SetSAVStartView(startViewCell(ms.Start))
	}

	// THE ARCHIVE SET IS WHERE THE MISSION'S WORDS COME FROM, handed to the
	// driver here rather than reached for there: the tier that owns the
	// world opens nothing, and a driver that went looking for an install
	// itself would give one mission two behaviours depending on when it ran.
	mw := ports.install.openMissionDriver(ms, r.units, mv.Viewer)
	mw.cheats.difficulty = diff
	ports.audio.wireReplies(mw)
	mw.view.SetPathfinding(ports.display.pathfinding)
	mw.view.SetGraphicsOptions(ports.display.graphics)
	mw.view.SetTextSmoothing(ports.display.textSmoothing)
	mw.view.SetFrameSmoothing(ports.display.frameSmoothing)
	mv.Viewer.SetGameMenuContext(func() ui.GameMenuContext {
		ctx := gameMenuContext(mw, true, ports.install.missionBriefing(r.n))
		ctx.LeaveToTown = ports.session.canLeaveMission(r.n, mw.world)
		return ctx
	})
	// THE PROJECTILE ART, assigned here rather than passed through the mission
	// door: that door already takes seven arguments, and the bundle is a fact
	// about the INSTALL rather than about the mission — one load serves
	// every mission this front end opens. A driver that is never given one
	// draws no spell art.
	mw.projectiles = ports.install.missionProjectiles()
	mw.armBurstPhases()
	ports.audio.wireEffects(mw, mv.Viewer)
	if err := settleMission(r, mw, mv.Viewer, ms, fog, ports.profile); err != nil {
		return preparedMap{}, err
	}
	if r.fresh() && ports.display.chicken {
		mw.chatCommand("#Chicken")
	}
	// Commit fresh selection with the driver; LOAD keeps its validated campaign.
	commitDriver := func() {
		if r.fresh() && r.town != nil && r.town.progress != nil {
			r.town.selectMission(r.n)
		}
		releaseWorldAudio(ports.audio, ports.session.activateLive(ports.rules(), mw, r.n, ms.Party))
		mw.mission.resumed = r.snap != nil || r.prepare != nil
	}
	if r.activate == nil {
		commitDriver()
	} else {
		*r.activate = commitDriver
	}
	// THE AUTOCAST SINK, on loadMap's own terms.
	mv.Viewer.SetAutocastSink(mw.setAutocast)
	mv.Viewer.SetQuickSpells(&ports.session.quickSpells)
	mv.Viewer.SetFormationSink(mw.cycleFormation)
	mv.Viewer.SetDefendSink(mw.defend)
	mw.restoreStructureUseMetadata(ports.install.missionTable())
	mv.Viewer.SetStructureUseSink(mw.useStructure)
	mv.Viewer.SetRetreatSink(func() { ports.profile.cycleRetreat(mw) })
	mv.Viewer.SetPlayerRetreatSink(mw.playerRetreat)
	pace := mw.paced
	if ports.display.deterministicFrame {
		pace = mw.deterministicFrame
	}
	mv.Viewer.SetEntities(mw.entityDraws())
	audioReady = true
	return preparedMap{viewer: mv.Viewer, tick: pace, order: mw.enqueue, cadence: mw.setCadenceMode,
		affect: mw.affect, advance: ports.advance(r.n, ms, mw.advanceNotice), attack: mw.attackOrCast,
		grab: mw.grab, stance: mw.stance, march: mw.march}, nil
}

// startMission turns the decoded map into a started mission and grants what
// entering it grants. A resume's positions are written into the map first, in
// the one window where its unit records are still records.
func startMission(r missionRequest, mv *MapView, addr string, diff mapload.Difficulty, ports missionPorts) (*Mission, *originalFog, error) {
	table := ports.install.missionTable()
	// THE ORIGINAL SAVE'S POSITIONS ARE WRITTEN INTO THE MAP HERE, in the
	// one window where the map's unit records are still records: after the
	// decode that produced them and before the start that turns them into
	// world entities.
	var fog *originalFog
	if r.prepare != nil {
		var err error
		if fog, err = r.prepare(mv.Map); err != nil {
			return nil, nil, err
		}
	}
	if r.snap != nil {
		// An imported mission removed its party's former ALM placements
		// before assigning entity IDs. Rebuild that same placement set
		// before native World bytes replace the temporary world; otherwise
		// party IDs and the person roster name different saved entities.
		withdrawSavedPartyPlacements(mv.Map, r.party)
	}
	entryParty := r.party
	if r.fresh() {
		// A fresh start drops the placements the original loader refuses.
		// A resume rebuilds its placement set from the saved actors.
		mapload.WithdrawBorderPlacements(mv.Map)
		entryParty = make([]mapload.PartyMember, len(r.party))
		for i, member := range r.party {
			entryParty[i] = mapload.MaterializePartyCarry(member, table)
		}
	}
	var draws *sim.Draws
	if r.fresh() {
		draws = missionRandom(ports.random)
	}
	ms, err := startMissionFromWith(mv.Map, addr, r.n, table, diff, entryParty, draws)
	if err != nil {
		return nil, nil, err
	}
	ports.install.nameJoinedRoster(ms)
	if r.fresh() {
		if err := seedCurrentCityObjects(ms, r.town, table, ports.cityBase); err != nil {
			return nil, nil, err
		}
		if err := ports.install.constructCurrentBuildings(ms); err != nil {
			return nil, nil, err
		}
		for i, id := range ms.Start.IDs {
			if i >= len(ms.Party) || ms.Party[i].Carry == nil {
				continue
			}
			if e, ok := ms.World.Entity(id); ok {
				ms.Party[i].Carry.NativeHistory = mapload.CaptureNativeCarryHistory(e)
			}
		}
	}
	// THE CAMPAIGN GRANTS ITS DOCUMENTS AT MISSION ENTRY (REG-SCN-097,
	// MISSION-DOC-021). The registry's AddTextDocument and AddPictureDocument
	// keys are per-`[Mission<n>]` section, and the original reads the
	// collection back in its two mission-entry routines, so entering the
	// mission is where the append belongs.
	//
	// IT IS HERE, AFTER THE MISSION HAS STARTED, so a mission that failed
	// to load grants nothing. The append is idempotent twice over: it
	// refuses a mission number no higher than the last one it ran for
	// (REG-SCN-097's own "only grows" guard, Medium — Town.CollectDocuments
	// states the grade), and it deduplicates on the (value, kind) pair.
	// A resumed save carries its own mission number, so re-entering the
	// mission a save was taken in re-grants nothing.
	r.town.CollectDocuments(r.n)
	// The original session lands as soon as the map has become a world, before
	// openMission below constructs NewAnnouncer and before a gameplay tick can
	// run. The same post-start carrier then restores equipment for actors which
	// are not persistent party members (hired mission actors are the shipped
	// case). Any refusal returns before this candidate can become the live map.
	if fog != nil && fog.after != nil {
		if err := fog.after(ms); err != nil {
			return nil, nil, err
		}
	}
	// Every party member's saved or newly assembled body must exist before
	// openMission takes its tick-zero art snapshot. Startup used to preload
	// only the process's default hero in NewFrontEnd. A restored hero with a
	// different saved appearance, and an AddHero companion, therefore opened
	// through the naked class fallback and was repainted only after a paced
	// refresh. Loading the exact body already carried by each PartyMember
	// preserves identity and makes the first visible frame the saved/new
	// member rather than a template-shaped intermediate.
	ports.install.loadPartyBodies(ms, r.units)
	return ms, fog, nil
}

// seatWorld puts the world the mission runs in its starting state: a native
// resume substitutes the saved world, anything else is seeded with the
// between-mission purse.
func seatWorld(r missionRequest, rules campaignService, table *mapload.Table, ms *Mission, v *ui.Viewer) error {
	if err := rules.seatWorld(r, ms); err != nil {
		return err
	}
	// THE SAVED WORLD REPLACES THE STARTED ONE HERE, between the start
	// and openMission (0143 plan D-2): the driver below is then built
	// over the resumed world exactly as it is built over a fresh one,
	// and every id the mission already handed out still names the world
	// the driver will run.
	if r.snap != nil {
		if err := resumeWorld(ms, r.snap, table); err != nil {
			return err
		}
		// A fresh viewer has no prior scheduled cache. Restore the native
		// save's last cadence before openMission projects its current tick.
		v.RestoreScheduledLightClock(ms.World.Tick())
		return nil
	}
	// Seed a fresh mission from the between-mission purse. Exact Player
	// bindings have already restored current Money in fog.after; a
	// participant shortcut must not overwrite it or invent a SelfSlot.
	if ms.savedDocument == nil || ms.savedDocument.PlayerPurses == nil {
		gold := uint32(r.town.Gold())
		if r.initialPurse != nil {
			gold = *r.initialPurse
		}
		ms.World.SetPurse(sim.SelfSlot, gold)
	}
	if d := r.town.knowledgeDiaries(); r.fresh() && len(d) != 0 {
		ms.World.SetSavedDiaries(d)
	}
	if _, hasClock := ms.World.SessionClock(); hasClock {
		// DIV-779: original SAV does not establish the incoming light
		// cache. Choose the standard scheduled cache at its retained S,
		// matching native continuation without an invented world tick.
		v.RestoreScheduledLightClock(ms.World.Tick())
	}
	return nil
}

// settleMission applies what a resume restores on top of the opened driver:
// the residue, the explored map, the application state and the camera of a
// native save, the explored map and action view of an original save, the
// fresh-game options of a new start, and the disclosure of an older byte form.
func settleMission(r missionRequest, mw *mapWorld, v *ui.Viewer, ms *Mission, fog *originalFog, profile missionProfile) error {
	// THE RESIDUE IS APPLIED LAST (0143 plan D-3): after the
	// constructor's own tick-0 push, so the restored exploration is what
	// the first frame draws.
	if r.snap != nil {
		mw.applyResidue(r.snap.Residue)
	}
	// THE ORIGINAL SAVE'S EXPLORED MAP GOES IN HERE, beside the residue and on
	// the residue's own timing: after the constructor's tick-0 push, so the
	// first frame draws the ground the save had already uncovered rather than
	// the freshly opened map's.
	if fog != nil {
		applied := mw.applyExplored(fog.cols, fog.rows, fog.cells)
		if fog.done != nil {
			fog.done(applied)
		}
	}
	var application *SnapshotApplicationState
	if r.snap != nil {
		application = r.snap.ApplicationState
	}
	if err := mw.restoreApplicationState(application, r.snap == nil && r.prepare != nil && ms.savedDocument != nil && ms.savedDocument.Document != nil); err != nil {
		return err
	}
	if r.snap == nil && r.prepare != nil {
		if err := mw.restoreOriginalActionView(); err != nil {
			return err
		}
		mw.seedRestoredRuns()
	}
	// A snapshot with no application record carries the camera on its own,
	// and so does one whose record is the local-only game-option form, whose
	// own restore deliberately leaves camera, selection and spell alone.
	// A camera outside its range must not stop a map from opening: it is
	// where the player was looking, not game state, so a rejected value
	// keeps the map's start position, which is what happened before the
	// field existed.
	if r.snap != nil && r.snap.CameraSet && mw.view != nil && (application == nil || application.LocalOnly) {
		_ = mw.view.RestoreCamera(r.snap.CameraX, r.snap.CameraY, r.snap.CameraZoom)
	}
	if r.fresh() {
		if err := profile.applyFreshGameOptions(mw, ms.Party); err != nil {
			return err
		}
		if len(mw.pending) != 0 {
			mw.rememberApplicationFormation(int32(mw.formationSetting()))
		}
	}
	return nil
}
