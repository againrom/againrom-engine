package game

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/random"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func (mw *mapWorld) residue() SnapshotResidue {
	r := SnapshotResidue{
		PendingGameOptions: pendingGameOptions(mw.pending),
		PendingQueue:       &SnapshotPendingQueue{Commands: append([]sim.Command(nil), mw.pending...)},
		Swing:              make(map[uint32]int, len(mw.swing)),
		Phase:              make(map[uint32]uint8, len(mw.phase)),
		GroupTag:           mw.groupTag,
	}
	if len(mw.pending) != 0 {
		r.PendingQueue.Ignored = make([]bool, len(mw.pending))
		copy(r.PendingQueue.Ignored, mw.pendingIgnored)
	}
	r.MissionLost = mw.mission != nil && mw.mission.outcome == sim.OutcomeLost
	if mw.mission != nil {
		r.PendingMessages = slices.Clone(mw.mission.pendingMessages)
		// Party order makes the envelope deterministic, without map iteration.
		for _, id := range mw.mission.guarded {
			if mw.mission.departed[id] {
				r.DepartedCharacters = append(r.DepartedCharacters, uint32(id))
			}
		}
	}
	for id := range mw.commanded {
		r.Commanded = append(r.Commanded, uint32(id))
	}
	for id, n := range mw.swing {
		r.Swing[uint32(id)] = n
	}
	for id, p := range mw.phase {
		r.Phase[uint32(id)] = uint8(p)
	}
	if mw.fog != nil {
		r.FogCols, r.FogRows = mw.fog.cols, mw.fog.rows
		r.FogExplored = append([]byte(nil), mw.fog.explored...)
		r.FogVisible = append([]byte(nil), mw.fog.visible...)
	}
	mw.actionVisuals(&r)
	return r
}

// applyResidue writes it back, after openMission has built everything else
// (plan D-3) — which is after the constructor's own tick-0 push, so what the
// first frame draws is the restored exploration and not the freshly opened
// map's.
//
// THE FOG PLANE IS REFUSED ON A DIMENSION MISMATCH rather than clamped. A
// mismatch means the map changed under the save, and overlaying the bytes would
// paint one map's exploration onto another's cells — a wrong picture nobody
// could read as wrong. Everything else in the residue is keyed by entity id and
// an id the reopened world does not hold is simply never looked up.
func (mw *mapWorld) applyResidue(r SnapshotResidue) {
	mw.restoreActionVisuals(r.SpellBolts, r.HealBursts)
	mw.restoreCastRuns(r.CastRuns)
	mw.restoreVisualIdentities(r.VisualIdentities, r.VisualNext)
	if r.PendingQueue != nil {
		mw.pending = slices.Clone(r.PendingQueue.Commands)
		mw.pendingIgnored = slices.Clone(r.PendingQueue.Ignored)
	} else {
		for _, order := range r.PendingGameOptions {
			mw.applyGameOption(order.Option, order.Value)
		}
	}
	for _, id := range r.Commanded {
		mw.commanded[sim.EntityID(id)] = true
	}
	for id, n := range r.Swing {
		mw.swing[sim.EntityID(id)] = n
	}
	for id, p := range r.Phase {
		mw.phase[sim.EntityID(id)] = sim.AttackPhase(p)
	}
	mw.groupTag = r.GroupTag
	mw.restoreNativeFog(r)
	if mw.mission != nil {
		for _, id := range r.DepartedCharacters {
			if mw.isGuarded(sim.EntityID(id)) {
				if mw.mission.departed == nil {
					mw.mission.departed = make(map[sim.EntityID]bool)
				}
				mw.mission.departed[sim.EntityID(id)] = true
			}
		}
		if r.MissionLost || mw.guardedCharacterLost() || mw.world.Outcome() == sim.OutcomeLost {
			alreadyShown := mw.mission.outcome == sim.OutcomeLost && mw.mission.outcomeShown
			mw.mission.announced, mw.mission.outcome = true, sim.OutcomeLost
			if !alreadyShown {
				mw.showOutcome()
			}
		}
	}
}

// applyExplored ORs an explored plane into the running one and reports
// whether it applied.
//
// Original Fog records reach this path after mission construction. Native
// checkpoints use restoreNativeFog instead: constructor exploration must not
// add cells to a snapshot captured between scheduled visibility samples.
//
// IT ONLY EVER ORs, WHICH IS THE ORIGINAL'S OWN LOAD ARM (TERR-FOG-145): that
// arm ORs the bit into each tile word and clears nothing, so a cell this build
// has already lit is not un-lit by a record that does not carry it.
//
// THE DIMENSIONS ARE REFUSED ON A MISMATCH rather than clamped. A mismatch
// means the plane was recorded over a different map, and overlaying the bytes
// would paint one map's exploration onto another's cells — a wrong picture
// nobody could read as wrong.
func (mw *mapWorld) applyExplored(cols, rows int, explored []byte) bool {
	if mw.fog == nil || cols != mw.fog.cols || rows != mw.fog.rows {
		return false
	}
	n := len(mw.fog.explored)
	if len(explored) < n {
		n = len(explored)
	}
	for i := 0; i < n; i++ {
		mw.fog.explored[i] |= explored[i]
	}
	// THE PUSH IS WHAT PUTS IT ON SCREEN, and it is guarded for the reason
	// every hand-built mapWorld in this package's own tests is a legal one:
	// a driver with no world and no viewer is what those fixtures are, and
	// this method must be drivable over one.
	if mw.world != nil && mw.view != nil {
		mw.push()
	}
	return true
}

// Snapshot captures the running game.
//
// IT REFUSES A MAP THAT IS NOT A MISSION. A map opened from the picker names no
// mission number, so there is nothing to resume it into; f.liveMission is zero
// for exactly that case and for a front end holding no map at all, which is
// what makes the refusal a property of one field rather than a rule someone
// keeps (plan D-4).
func (f *FrontEnd) Snapshot(onMap bool) (Snapshot, string, error) {
	if t := f.townUI; t != nil && t.worldMap != nil && t.worldMap.returnMission != 0 {
		return Snapshot{}, "", errors.New("return to the city before saving")
	}
	var s Snapshot
	edition := f.Base().Profile.Edition()
	s.game = edition.SaveTag
	s.randomSession = f.randomSessionNow()
	difficulty, err := campaignDifficulty(int64(f.Difficulty))
	if err != nil {
		return Snapshot{}, "", err
	}
	s.Difficulty = difficulty
	if edition.TownDifficulty && !onMap {
		s.Difficulty = f.Difficulty
	}
	fame := cloneFame(f.fame)
	s.Fame = &fame
	s.QuickSpells = f.quickSpells
	snapshotTown(f.Town, &s)
	if s.CampaignState {
		s.Campaign.MissionTime = fame.Time
		s.Campaign.ScoreEvents = fame.Events
		s.Campaign.ScoreEventsKnown = true
	}
	snapshotWorldSelectedOnce(f.townUI, &s)
	if s.CampaignState {
		s.Campaign = f.currentCampaignMarkers(s.Campaign, s.WorldSelectedOnce)
	}
	s.Party = mapload.CloneParty(f.Carried)
	s.Offered = f.Offered
	s.OriginalCity = f.originalCity.snapshot()
	s.townOptions = townApplicationOptions(s.OriginalCity)
	// A SAVE TAKEN OFF THE MAP SCREEN IS A TOWN SAVE EVEN THOUGH A DRIVER IS
	// STILL HELD. f.live is set when a map opens and is never cleared —
	// nothing on the game side runs when a map screen is left — so the
	// question "is a mission running" is the FRONT-END's to answer and it
	// answers it with the screen the menu was opened over. Reading f.live alone
	// would write a finished mission's world into a save taken in the town
	// after it.
	if !onMap || f.live == nil {
		if err := f.campaign().townSavePoint(f); err != nil {
			return Snapshot{}, "", err
		}
		s.shop = f.captureShop()
		return s, fmt.Sprintf("town - gold %d", s.Gold), nil
	}
	if f.liveMission == 0 {
		return Snapshot{}, "", errors.New("a map opened from the list is not a game and cannot be saved")
	}
	b, err := worldBytes(f.live.world)
	if err != nil {
		return Snapshot{}, "", err
	}
	s.Mission = f.liveMission
	s.World = b
	ghost := f.live.world.Ghost()
	s.ghost = &ghost
	s.Residue = f.live.residue()
	f.campaign().captureMission(f, &s)
	s.deathAges = f.live.captureCurrentDeathAges()
	s.ApplicationState, err = f.captureApplicationState(f.live)
	if err != nil {
		return Snapshot{}, "", err
	}
	if f.live.view != nil {
		// SaveApplication leaves the camera zero when the viewer has none, and
		// a zero zoom is not a camera this build can restore, so it is the
		// signal that there is nothing to record.
		if view := f.live.view.SaveApplication(); view.Zoom > 0 {
			s.CameraSet, s.CameraX, s.CameraY, s.CameraZoom = true, view.ViewX, view.ViewY, view.Zoom
		}
	}
	if f.live.mission != nil && f.live.mission.state != nil {
		s.ActorManifest, err = snapshotActorManifest(f.live.mission.state.ActorManifest, f.live.world)
		if err != nil {
			return Snapshot{}, "", err
		}
		s.SavedDocument, err = snapshotSavedDocument(f.live.mission.state)
		if err != nil {
			return Snapshot{}, "", err
		}
	}
	// THE PARTY THE MISSION WAS OPENED WITH, not f.Carried: f.Carried is
	// what the LAST finished mission handed over, and a save taken inside
	// mission n has to reopen n with the party n was started with — that is
	// what makes the reopened world's entity ids line up with the ids the
	// saved world holds (plan D-2).
	liveParty := f.liveParty
	if f.live.mission != nil {
		n := f.live.mission.entryCount
		if n > len(f.live.mission.party) {
			n = len(f.live.mission.party)
		}
		liveParty = f.live.mission.party[:n]
	}
	if len(liveParty) > 0 {
		s.Party = mapload.CloneParty(liveParty)
	}
	if f.live.mission != nil && f.live.mission.state != nil {
		s.CurrentPartyIDs = append([]sim.EntityID(nil), f.live.mission.ids[:min(len(f.live.mission.ids), len(s.Party))]...)
		s.CurrentRoster = make(map[sim.EntityID]mapload.PartyMember)
		for id, p := range f.live.mission.state.Start.Roster {
			// A healthy Start.Roster never names an entry-party id to begin
			// with (it holds NPCs and hires, not player-controlled party
			// members), but a file loaded from before this exclusion existed
			// can still carry one, because restoreCurrentPartyMembers copies a
			// saved CurrentRoster straight into Start.Roster, unfiltered. Left
			// in, it would be restated into CurrentRoster on every later SAVE
			// forever. s.CurrentPartyIDs, just computed above, already carries
			// this hero, so nothing is lost by excluding it here.
			if slices.Contains(s.CurrentPartyIDs, id) {
				continue
			}
			s.CurrentRoster[id] = mapload.CloneParty([]mapload.PartyMember{p})[0]
		}
		// Only the tail beyond the entry party (a companion syncJoinedHeroes
		// promoted mid-mission) belongs here. s.Party/s.CurrentPartyIDs above
		// already carry the entry party itself; restating it here duplicated
		// every party hero into Start.Roster on LOAD (restoreCurrentPartyMembers
		// copies a saved CurrentRoster straight into it, unfiltered), and
		// missionAppearanceArt's own Start.Roster loop then drew each one with
		// its Roster (NPC) class instead of its party (hero) body -- it has no
		// exclusion for an id that is also a current party member. A healthy
		// load's own Start.Roster never names a party hero to begin with (it
		// holds NPCs and hires, not player-controlled party members), so this
		// loop restating one from mission.party was never filling a real gap.
		for i, id := range f.live.mission.ids {
			if i < len(s.CurrentPartyIDs) {
				continue
			}
			if i < len(f.live.mission.party) {
				s.CurrentRoster[id] = mapload.CloneParty(f.live.mission.party[i : i+1])[0]
			}
		}
	}
	if s.SavedDocument != nil && !s.CampaignState {
		for _, e := range f.live.world.Entities() {
			if e.SourceBinding.Generated() {
				chapter := f.generatedCampaignChapter(s)
				s.Campaign, err = nativeCampaignProjectionForChapter(f, s, chapter)
				if err != nil {
					return Snapshot{}, "", err
				}
				s.CampaignState = true
				s.HeroGrantState, s.ConsumedHeroGrants = false, nil
				s.Campaign.SelectedMission = uint32(s.Mission)
				break
			}
		}
	}
	return s, fmt.Sprintf("mission %d - tick %d - gold %d", s.Mission, f.live.world.Tick(), s.Gold), nil
}

// preparedMap is a fully constructed MapOpener result. Returning it through an
// opener keeps the UI seam unchanged while guaranteeing that the opener it
// receives has no map, asset, simulation or mission-construction work left to
// fail.
type preparedMap struct {
	viewer  *ui.Viewer
	tick    ui.MapTick
	order   ui.MapOrder
	cadence ui.MapCadence
	affect  ui.MapAffect
	advance ui.MapAdvance
	attack  ui.MapAttack
	grab    ui.MapGrab
	stance  ui.MapStance
	march   ui.MapMarch
}

func openPrepared(p preparedMap, commit func()) ui.MapOpener {
	// UI activation is serial today, but the opener itself is a public seam
	// value and its one-commit promise does not depend on caller scheduling.
	var committed sync.Once
	return func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect,
		ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
		committed.Do(commit)
		return p.viewer, p.tick, p.order, p.cadence, p.affect, p.advance, p.attack,
			p.grab, p.stance, p.march, nil
	}
}

type restoreCandidate struct {
	fame              SnapshotFame
	quickSpells       [4]uint32
	difficulty        mapload.Difficulty
	town              *Town
	carried           []mapload.PartyMember
	offered           int
	townOnly          bool
	worldSelectedOnce map[int]bool
	worldMapReturn    *SnapshotMapReturn
	originalCity      *originalCitySaveState

	prepared preparedMap
	activate func()
	units    *terrain.UnitSet
	// randomSession is the random session the candidate's game begins; nil
	// keeps the running one.
	randomSession *random.Session
}

func (f *FrontEnd) installCandidate(c *restoreCandidate) {
	if c.randomSession != nil {
		f.beginRandomSession(*c.randomSession)
	}
	f.CampaignSession.adopt(c)
	f.resetTownSurface(c)
	if c.townOnly {
		releaseWorldAudio(f.runtimeAudio(), f.endLive())
		f.campaign().arriveLoaded(f, c)
		return
	}
	// Mission preparation loads any uncached hero bodies into an isolated
	// bundle. The bundle becomes the front end's cache only at the same commit
	// that adopts the prepared driver. A refused candidate therefore cannot
	// change how the still-running map resolves a body on its next frame.
	f.Units = c.units
	c.activate()
	if f.live != nil && f.live.applicationState != nil {
		f.wimpyMode = normalizedWimpy(f.live.applicationState.Original.Wimpy)
	}
}

// resetTownSurface gives a loaded game a town screen with none of the previous
// game's state, then replays the candidate's own record of which world-map
// missions were selected.
func (f *FrontEnd) resetTownSurface(c *restoreCandidate) {
	if f.campaign().keepsTownSurface(c) {
		return
	}
	// A genuinely different loaded game must not inherit ANY of the previous
	// game's own townScreen session state — not only the world map
	// (`DIV-128`/`DIV-137`/`DIV-138`), but the room, conversation, shop
	// presentation and party-keyed composition caches too (round 2 of this
	// story's own adversarial review, P finding): resetForNewGame's own doc
	// enumerates the whole population.
	f.townUI.resetForNewGame()
	selected := frontWorldMapFilteredMarkers(f)
	if len(c.worldSelectedOnce) > 0 || len(selected) > 0 {
		screen := f.TownScreen().(*townScreen)
		screen.worldSelectedOnce = cloneWorldSelectedOnce(c.worldSelectedOnce)
		for _, mission := range selected {
			screen.markWorldSelected(mission)
		}
	}
}

// restoreWorldMapReturn puts a loaded town back on the world map it was saved
// returning from, or marks the first map point as reached.
func (f *FrontEnd) restoreWorldMapReturn(c *restoreCandidate) {
	if c.worldMapReturn == nil {
		return
	}
	screen := f.TownScreen().(*townScreen)
	screen.beginWorldMapReturn(c.worldMapReturn.Mission)
	if screen.worldMap != nil && screen.worldMap.returnMission != 0 {
		screen.arriveWorldMap()
	} else if f.Town.progress != nil {
		f.Town.progress.firstMapPoint = true
	}
}

// decodeRestore validates snapshot s and builds the candidate it describes
// from plain values: no front end, screen or sound device. It returns the
// snapshot as validated (its document owned, its party upgraded), which is
// the one a mission opener must resume. A town snapshot is complete in the
// candidate; a mission snapshot still needs its map screen built.
func decodeRestore(s Snapshot, in townInstall) (*restoreCandidate, Snapshot, error) {
	if s.CityObjects != nil {
		if err := s.CityObjects.Validate(); err != nil {
			return nil, s, err
		}
	}
	if err := validatePendingGameOptions(s.Residue.PendingGameOptions); err != nil {
		return nil, s, err
	}
	if err := validateSnapshotPending(s.Residue.PendingQueue); err != nil {
		return nil, s, err
	}
	if err := validateSnapshotFame(s.Fame); err != nil {
		return nil, s, err
	}
	if err := validateApplicationState(s.ApplicationState); err != nil {
		return nil, s, err
	}
	if err := validateNativeFogResidue(s.Residue); err != nil {
		return nil, s, err
	}
	ownedDocument, err := savedDocumentFromSnapshot(s)
	if err != nil {
		return nil, s, err
	}
	s.SavedDocument = ownedDocument
	for _, p := range s.Party {
		if err := mapload.ValidatePartyLoad(p); err != nil {
			return nil, s, err
		}
	}
	if err := validateQuickSpells(s.QuickSpells); err != nil {
		return nil, s, err
	}
	if err := validateSnapshotItemWeights(s); err != nil {
		return nil, s, err
	}
	if err := validateSnapshotBooks(s); err != nil {
		return nil, s, err
	}
	originalCity, err := originalCityFromSnapshot(s)
	if err != nil {
		return nil, s, err
	}
	if err := in.validateOriginalCityBaseline(originalCity); err != nil {
		return nil, s, err
	}
	if err := originalCity.validateSalesParty(s.Party); err != nil {
		return nil, s, err
	}
	s.Party = originalCityUpgradeParty(s.Party, originalCity)
	difficulty, err := campaignDifficulty(int64(s.Difficulty))
	if err != nil {
		return nil, s, err
	}
	worldSelectedOnce, err := worldSelectedOnceFromSnapshot(s.WorldSelectedOnce)
	if err != nil {
		return nil, s, err
	}
	if err := validateSnapshotHeroGrants(in.campaign, s); err != nil {
		return nil, s, err
	}
	if err := validateSnapshotCampaignRecords(in.campaign, s); err != nil {
		return nil, s, err
	}
	if s.CampaignState {
		progress, err := campaignProgressFromSAV(in.campaign, s.Campaign)
		if err != nil {
			return nil, s, err
		}
		if err := validateCampaignLocation(in.campaign, progress, s.Mission, "againrom save"); err != nil {
			return nil, s, err
		}
	}
	c := &restoreCandidate{
		fame:              fameFromSnapshot(s),
		quickSpells:       s.QuickSpells,
		difficulty:        difficulty,
		town:              restoreTown(in.campaign, s),
		carried:           mapload.OwnParty(s.Party),
		offered:           s.Offered,
		townOnly:          s.Mission == 0,
		worldSelectedOnce: worldSelectedOnce,
		originalCity:      originalCity,
	}
	if c.townOnly && in.campaign.completedBy(c.town) {
		return nil, s, errCompletedCampaignFile
	}
	if s.WorldMapReturn != nil {
		pending := *s.WorldMapReturn
		if s.Mission != 0 || !c.town.Open() || pending.Mission <= 0 || pending.Shown < 0 {
			return nil, s, errors.New("invalid saved world-map return")
		}
		c.worldMapReturn = &pending
	}
	// Party state is outside World and is the whole save for a town snapshot.
	// Repair it at this ownership boundary before either the town adopts it or
	// a mission derives entities from it.
	mapload.RepairLegacyParty(c.carried, in.table)
	// Saves written before MercenaryState existed carry no pool/hired arrays.
	// Infer the already-hired types from the isolated candidate party before
	// the candidate is committed. A refused mission restore must not mutate the
	// active town while performing this compatibility step.
	if !s.MercenaryState && !s.CampaignState {
		for _, member := range c.carried {
			typ := int(member.MercenaryType)
			if typ > 0 && typ < len(c.town.mercPool) {
				c.town.mercPool[typ] = 0
				c.town.mercHired[typ] = true
			}
		}
	}
	if c.townOnly {
		return c, s, nil
	}
	if len(s.World) == 0 {
		return nil, s, errors.New("save names a mission and carries no world")
	}

	return c, s, nil
}

// prepareRestore validates and constructs a snapshot without changing the
// active FrontEnd. The caller may commit a town candidate directly after this
// returns, or expose a mission candidate through openPrepared.
func (f *FrontEnd) prepareRestore(s Snapshot) (*restoreCandidate, error) {
	c, s, err := decodeRestore(s, f.townInstall())
	if err != nil {
		return nil, err
	}
	if c.townOnly {
		return c, nil
	}
	snap := s
	c.units = cloneCandidateUnits(f.Units)
	open := f.missionOpenerMode(s.Mission, c.carried, &snap, nil, nil, &c.activate, c.units, c.difficulty, c.town)
	c.prepared.viewer, c.prepared.tick, c.prepared.order, c.prepared.cadence,
		c.prepared.affect, c.prepared.advance, c.prepared.attack, c.prepared.grab,
		c.prepared.stance, c.prepared.march, err = open()
	if err != nil {
		return nil, err
	}
	if c.activate == nil {
		return nil, errors.New("prepared mission has no live-driver commit")
	}
	return c, nil
}

func worldSelectedOnceFromSnapshot(missions []int) (map[int]bool, error) {
	if len(missions) == 0 {
		return nil, nil
	}
	out := make(map[int]bool, len(missions))
	for _, mission := range missions {
		if mission <= 0 {
			return nil, fmt.Errorf("save world-map marker history contains invalid mission %d", mission)
		}
		if out[mission] {
			return nil, fmt.Errorf("save world-map marker history contains duplicate mission %d", mission)
		}
		out[mission] = true
	}
	return out, nil
}

func cloneWorldSelectedOnce(src map[int]bool) map[int]bool {
	if len(src) == 0 {
		return nil
	}
	out := make(map[int]bool, len(src))
	for mission, selected := range src {
		if selected {
			out[mission] = true
		}
	}
	return out
}

// cloneCandidateUnits isolates the only mutable maps in a unit bundle. Class
// records and cached body records are immutable after loading, so their
// pointers may be shared. LoadHeroBody can then add a candidate-only body
// without writing through to the active map's cache.
func cloneCandidateUnits(src *terrain.UnitSet) *terrain.UnitSet {
	if src == nil {
		return nil
	}
	out := *src
	if src.Bodies != nil {
		out.Bodies = make(map[string]*terrain.UnitClass, len(src.Bodies))
		for key, body := range src.Bodies {
			out.Bodies[key] = body
		}
	}
	return &out
}

// SaveSeams builds the three function values pkg/ui takes.
//
// NOTHING THAT CROSSES HERE NAMES A WORLD, A MISSION, A PARTY OR A TOWN. A
// label and a name are strings, "the loaded game is in the town" is a bool, and
// a map opener is the shape three doors in that package already accept — so the
// tier that may import the render tier and no other gains a save menu without
// gaining any way to describe a simulation value.
//
// now IS INJECTED so a test can name the file a save lands in without waiting a
// second for the clock to move.
// orig IS READ-ONLY AND MAY BE ZERO. An OriginalStore with no directory lists
// nothing, so a build pointed at no install behaves exactly as this seam did
// before the original format could be read at all.
func (f *FrontEnd) SaveSeams(store SaveStore, orig OriginalStore, now func() time.Time, observers ...func(string, Snapshot)) (ui.SaveGame, ui.SaveList, ui.LoadGame) {
	if now == nil {
		now = time.Now
	}
	save := func(onMap bool) (string, error) {
		s, label, err := f.Snapshot(onMap)
		if err != nil {
			return "", err
		}
		notifySaveCapture(observers, "manual", s)
		encoded, err := encodeSaveLabel(label, f.textSelector())
		if err != nil {
			return "", err
		}
		b, err := f.ExportCurrentSave(s, encoded)
		if err != nil {
			return "", err
		}
		return store.WriteOriginal(orig.Dir, b)
	}
	// OURS FIRST, THE ORIGINAL GAME'S AFTER, and the two are NOT interleaved by
	// time. A player who has just saved wants his own file at the top, and the
	// install's saves are a fixed set he did not produce in this session —
	// merging them by mtime would push what he just wrote below a file from
	// another program's last run. Each group keeps its own newest-first order.
	list := func() []ui.SaveEntry {
		var out []ui.SaveEntry
		// Decode stored labels at the seam; SaveStore.List retains header bytes.
		if ents, err := store.List(); err == nil {
			for _, e := range ents {
				name := e.Name
				if IsOriginal(name) {
					name = localOriginalSaveToken(name)
				}
				out = append(out, ui.SaveEntry{Name: name, Label: drawableLabel(e.Label)})
			}
		}
		for _, e := range orig.List() {
			out = append(out, ui.SaveEntry{Name: e.Name, Label: e.Label})
		}
		return out
	}
	// SAV IS THE ONLY SAVE FORMAT. A local row carries a source tag, an install
	// row is a bare .sav name, and every other name is refused. The name is the
	// opaque token pkg/ui passed back unread.
	load := func(name string) (ui.MapOpener, bool, error) {
		if localName, ok := localOriginalSaveName(name); ok {
			b, err := store.Read(localName)
			if err != nil {
				return nil, false, err
			}
			return f.RestoreOriginal(b)
		}
		if IsOriginal(name) {
			b, err := orig.Read(name)
			if err != nil {
				return nil, false, err
			}
			return f.RestoreOriginal(b)
		}
		return nil, false, fmt.Errorf("%q is not a SAV file", name)
	}
	return save, list, load
}

// A local authored .sav and a read-only install .sav may lawfully share the
// same game#### name. The load window returns only one opaque string, so local
// rows carry a source tag while retaining the .sav suffix that identifies their
// decoder. SaveGame still reports the real disk name after publication.
const localOriginalSavePrefix = "local-sav:"

func localOriginalSaveToken(name string) string { return localOriginalSavePrefix + name }

func localOriginalSaveName(token string) (string, bool) {
	if !strings.HasPrefix(token, localOriginalSavePrefix) {
		return "", false
	}
	name := strings.TrimPrefix(token, localOriginalSavePrefix)
	return name, name != "" && name != "." && name != ".." && IsOriginal(name) && name == filepath.Base(name)
}

// drawableLabel substitutes for the runes the load window's font cannot draw.
//
// THE FONT DRAWS A BLANK, NOT A SUBSTITUTE. The window draws through
// ebitenutil.DebugPrintAt, whose atlas is 256 cells indexed by the rune;
// U+2014 resolves to a source rectangle outside it, SubImage intersects to
// empty, and the draw is a degenerate quad. Nothing panics and nothing
// appears, so the row reads "mission 30  tick 176  gold 70" with the
// separators silently gone (1032 adversarial pass 3, F-8, which read the
// mechanism to the leaf after pass 2 described it as junk glyphs; it is not).
//
// ONLY U+2014 IS ATTESTED IN THE CORPUS. The en dash and the ellipsis are
// handled with it because they are the same family and each has one
// unambiguous ASCII spelling; any other rune outside the drawable range
// becomes '?', which is asciiLabel's own choice for its own reason
// (originalsave.go): one mark per rune, so a name's length and shape survive
// and the row stays recognisable beside its neighbours.
//
// A GENUINELY EXTERNAL ORIGINAL SAVE (orig.List(), the install's own
// archives root) DOES NOT COME THROUGH HERE: it is appended straight from
// e.Label a few lines below, never through this function.
func drawableLabel(s string) string {
	drawable := true
	for _, r := range s {
		if r < 0x20 || r >= 0x7f {
			drawable = false
			break
		}
	}
	if drawable {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 0x20 && r < 0x7f:
			b.WriteRune(r)
		case r == '\u2014' || r == '\u2013':
			b.WriteByte('-')
		case r == '\u2026':
			b.WriteString("...")
		default:
			b.WriteByte('?')
		}
	}
	return b.String()
}

// resumeWorld replaces a freshly started mission's world with a saved one (plan
// D-2).
//
// THE WORLD IS UNMARSHALLED INTO THE ONE StartMissionFrom BUILT rather than
// beside it, so every pointer the mission already handed out still names the
// world the driver will run.
//
// sim.CheckSaveForm REFUSES EVERY BYTE FORM THIS BUILD DOES NOT READ, with a
// sentence written for the player. It is not wrapped, so the load window shows
// it unedited.
func resumeWorld(ms *Mission, s *Snapshot, table *mapload.Table) error {
	if ms.World == nil {
		return errors.New("mission started with no world")
	}
	if err := sim.CheckSaveForm(s.World); err != nil {
		return err
	}
	// What the started mission already knows, taken BEFORE the unmarshal
	// replaces it.
	structures := ms.World.Structures()
	freshScript := ms.World.Script()

	if err := ms.World.UnmarshalBinary(s.World); err != nil {
		return fmt.Errorf("save's world half: %w", err)
	}
	ms.World.ResetLoadedAreaCosts()
	if !sim.HasStructureBlockingForm(s.World) {
		ms.World.RestoreStructureBlocking(structures)
	}
	if s.Residue.FogVisible != nil {
		b := ms.World.Bounds()
		if s.Residue.FogCols != int(b.Width) || s.Residue.FogRows != int(b.Height) {
			return errors.New("saved visibility belongs to a different map extent")
		}
	}
	if err := restoreActorManifest(ms, s.ActorManifest); err != nil {
		return err
	}
	document := s.SavedDocument
	if document == nil && ms.World.SavedObjects() != nil {
		if err := validateCurrentMissionRegistry(s, ms.World); err != nil {
			return err
		}
	} else if err := restoreSavedDocument(ms, document); err != nil {
		return err
	}
	// A current SAV may intentionally carry an absent live session head while
	// its retained document already contains the constructor timer.  Restore
	// the world-side absence before any later SAVE; otherwise the native
	// checkpoint would silently turn the retained constructor bytes into live
	// simulation state on LOAD.
	if document != nil && document.Document != nil {
		actions, err := readCurrentActions(document.Document)
		if err != nil {
			return err
		}
		if actions != nil {
			if err := restoreAbsentSessionHead(ms.World, document.Document, actions.AbsentSessionHead); err != nil {
				return err
			}
		}
	}
	if err := rebuildActorPeople(ms, table); err != nil {
		return err
	}
	// Current-form saves written by the old linked-item loader need no byte
	// migration. Repair its exact scalar underflow signature in memory and
	// price each affected item from this installation's definition table.
	sim.RepairLegacyLinkedItems(ms.World, func(item sim.ItemInstance) int32 {
		return mapload.RepriceItemInstance(item, table)
	})
	if document == nil {
		sim.RepairLegacyUnitCapacity(ms.World, data.UnitCapacity())
	}
	// Older current-form saves may contain a player command in the retained
	// lifecycle that caused the reported repeated Fire Ball. Preserve at most
	// its first pending release; map-owned AI orders remain retained.
	ms.World.RestoreOneShotPlayerCasts()

	// Old native mission saves contain the old compiled program. Repair only
	// its exact missing-companion signature, preserving execution state and
	// all existing bindings. Original-SAV reconstruction has a separate path.
	if (ms.Number == 30 || ms.Number == 130) && ms.Map != nil {
		refs := campaignScriptRefs(ms.Map, table, ms.Party)
		if refs.HasCompanion {
			// The old program built every node; so does this compile.
			refs.Companion, refs.HasCompanion, refs.Roster = 0, false, false
			if legacy, _, err := mapload.CompileScript(ms.Map, refs); err == nil {
				ms.World.RestoreScriptBindings(legacy, freshScript)
			}
		}
	}
	// Resolve hero ordinals an older build left unbound; never refuses the load.
	_ = restoreCurrentScriptBindings(ms, table)
	return nil
}

// liveDriver records which driver a save would reach (plan D-4).
//
// IT KEEPS THE MISSION'S OWN SLICE AND NO LONGER A COPY OF IT (1005 round 2,
// seventh pass, R1). party here is Mission.Party — the very slice the running
// mission writes through: switchInventorySubject and rearm both raise
// PartyMember.WeaponMaterialized on it (world.go), and openMission seeds it.
// Taking mapload.OwnParty's deep copy meant Snapshot (resume.go) wrote the
// party AS IT STOOD AT THE OPEN, so every mid-mission write to a member record
// was correct in memory and absent from the save: reloading restored a member
// whose starting weapon had never been taken off, and the retired display
// fallback came back with it. mapload.NameParty gives the same stable
// identities OwnParty did, in place, so nothing downstream sees a different
// member id.
func (f *FrontEnd) liveDriver(mw *mapWorld, n int, party []mapload.PartyMember) {
	releaseWorldAudio(f.runtimeAudio(), f.activateLive(f.campaign(), mw, n, party))
}

// LiveWorld is the world the open map screen is running, for a developer tool
// that has no window to read one off (cmd/savecheck).
//
// IT IS A READ AND NOTHING ELSE. Nothing in the game calls it; it exists because
// `go test ./...` is green with no install, so the only place a save taken in a
// SHIPPED mission can be shown to come back is a tool pointed at a real one.
func (f *FrontEnd) LiveWorld() (*sim.World, bool) {
	if f.live == nil || f.live.world == nil {
		return nil, false
	}
	return f.live.world, true
}

// LiveNotice reports the open map screen's own notice state — its text, its
// kind and whether one is open — for a developer tool with no window to read
// one off (cmd/savecheck), the same reason LiveWorld exists. It is what a
// 1032 B3 load disclosure reaches: openLoadNotice (world.go) calls the same
// mw.view.SetDialogue every mission-script dialogue calls, and this reads it
// back through the viewer's own state rather than a second field.
func (f *FrontEnd) LiveNotice() (text string, kind ui.NoticeKind, open bool) {
	if f.live == nil {
		return "", 0, false
	}
	return f.live.view.NoticeState()
}

// NoticePage is one page of an open notice and how it measures against the
// window that draws it: the wrapped lines its text produces, and how many of
// those the window's text area draws.
//
// Produced and Drawn differ only for a page the window CLIPS, which is the
// state 1032's first review found on 51 of the owner's 52 loadable saves and
// which no witness in this tree could see: LiveNotice reports the string that
// was pushed, and a pushed string is complete whatever was drawn.
type NoticePage struct {
	Text     string
	Produced int
	Drawn    int
}

// LiveNoticePages is every page of the notice the open map screen is paging
// through, each measured against that screen's own notice window.
//
// It is LiveNotice's counterpart for the whole notice rather than the page on
// screen, and it exists for the same reason LiveWorld does: `go test ./...` is
// green with no install, so the only place the REAL font's line count can be
// read is a developer tool pointed at an install (cmd/savecheck).
//
// An empty result means no notice is paging: either none is open, or the one
// that is came out of an event file and is paged by EventPart instead.
func (f *FrontEnd) LiveNoticePages() []NoticePage {
	if f.live == nil || f.live.mission == nil || f.live.view == nil {
		return nil
	}
	pages := f.live.mission.pages
	out := make([]NoticePage, 0, len(pages))
	for _, p := range pages {
		produced, drawn := f.live.view.DialogueClip(p)
		out = append(out, NoticePage{Text: p, Produced: produced, Drawn: drawn})
	}
	return out
}

// LiveParty returns the stable party backing the open real-mission driver. It
// is a deep copy for the same reason NextParty is: savecheck can reconstruct a
// normal mission from an original save's decoded party without gaining a write
// path into the live session.
func (f *FrontEnd) LiveParty() []mapload.PartyMember {
	if f.live == nil {
		return nil
	}
	if f.live.mission != nil {
		return mapload.CloneParty(f.live.mission.party)
	}
	return mapload.CloneParty(f.liveParty)
}

// LiveHeadlessSnapshot is the map-screen projection used by no-window
// developer tools. Keeping the ui.Screen value on this side of the game seam
// lets those tools read the running mission without importing the Client tier.
func (f *FrontEnd) LiveHeadlessSnapshot() HeadlessState {
	return f.HeadlessSnapshot(ui.ScreenMap)
}

// LiveDoll reports how the actual map-unit compositor resolves one live actor.
// It is read-only developer evidence for original-save equipment restoration:
// the same installed archive, figure identity and composeUnitFigure routine the
// visible map uses, reduced to digests so no game image is written or returned.
func (f *FrontEnd) LiveDoll(entity uint32) (LiveDollEvidence, bool) {
	if f.live == nil || f.live.world == nil {
		return LiveDollEvidence{}, false
	}
	id := sim.EntityID(entity)
	fig, ok := f.live.figures[id]
	if !ok {
		return LiveDollEvidence{}, false
	}
	slots, ok := f.live.world.Equipped(id)
	if !ok {
		return LiveDollEvidence{}, false
	}
	return composeDollEvidence(f.live.archive(), slots, fig), true
}

// LiveHealFeedback reports the semantic Heal bursts and the actual decoded
// sheet instances the open map would submit this frame. It is developer-only,
// read-only evidence: the returned counts come after observeCasts has consumed
// the simulation event, and no particle or sheet state is exposed for mutation.
func (f *FrontEnd) LiveHealFeedback() (bursts, sprites int) {
	if f.live == nil {
		return 0, 0
	}
	return len(f.live.healBursts), len(f.live.healSpriteDraws())
}

// LiveAdvance runs the open map screen's own logic tick n times, unpaced. Only
// the last tick projects to the viewer: a projection replaces the previous one,
// and the state a projection keeps across ticks runs on every tick
// (pushHeldState).
//
// IT IS THE DRIVER'S TICK AND NOT THE FRAME'S. mw.paced is gated on a wall
// clock, so a tool calling it in a loop advances nothing at all — which is
// exactly what the first run of cmd/savecheck did, and is why this exists as a
// named reader rather than as a loop in the tool.
func (f *FrontEnd) LiveAdvance(n int) {
	if f.live == nil {
		return
	}
	for i := 0; i < n; i++ {
		f.live.tickStep(nil, i == n-1)
	}
}

// LiveAdvanceCasts is LiveAdvance with the applied-cast observations returned
// to a no-window developer instrument. It runs the exact driver tick and merely
// copies the per-tick observations that the map renderer already consumed.
func (f *FrontEnd) LiveAdvanceCasts(n int) []sim.CastEvent {
	if f.live == nil {
		return nil
	}
	var out []sim.CastEvent
	for i := 0; i < n; i++ {
		f.live.tickStep(func(events []sim.CastEvent) {
			out = append(out, events...)
		}, i == n-1)
	}
	return out
}

// LiveDamage and LiveKill are the script's health write, for real-mission
// no-window witnesses; LiveKill writes -1. The write lands at once, between
// ticks, and shares no code with the map's diagnostic affect key.
func (f *FrontEnd) LiveDamage(entity uint32, amount int32) {
	if f.live == nil || amount <= 0 {
		return
	}
	_ = f.live.world.HeadlessDamage(sim.EntityID(entity), amount)
}

func (f *FrontEnd) LiveKill(entity uint32) {
	if f.live == nil {
		return
	}
	e, ok := f.live.world.Entity(sim.EntityID(entity))
	if !ok || e.Dead() {
		return
	}
	_ = f.live.world.HeadlessDamage(e.ID, e.HP+1)
}

// LiveAutocast submits the same canonical toggle command as the spellbook's
// right-click route. It is used by savecheck to arm shipped Heal in a loaded
// mission without mutating the entity from the client/tool tier.
func (f *FrontEnd) LiveAutocast(entity uint32, spell uint16) {
	if f.live == nil {
		return
	}
	f.live.pending = append(f.live.pending, sim.Autocast(sim.EntityID(entity), sim.SpellID(spell)))
}

// LiveOrder issues one move order into the open map screen, which is what marks
// an entity COMMANDED — the residue field whose loss is behavioural. A tool
// needs it to have something for the round trip to carry.
func (f *FrontEnd) LiveOrder(entity uint32, x, y int) {
	if f.live == nil {
		return
	}
	f.live.enqueue(entity, x, y)
}

// LiveExplored is how many cells of the open map screen's explored plane are
// explored, and how many the plane holds.
//
// IT IS LiveWorld'S SHAPE AND EXISTS FOR LiveWorld'S REASON. `go test ./...` is
// green with no install, so the only place a plane restored from a SHIPPED
// save can be counted against the file that carried it is a tool pointed at a
// real one. It reads the plane the map screen is actually drawing from, not the
// bytes the restore was handed, which is what makes it a check of the restore
// rather than of the decode.
func (f *FrontEnd) LiveExplored() (set, total int) {
	if f.live == nil || f.live.fog == nil {
		return 0, 0
	}
	for _, v := range f.live.fog.explored {
		if v != 0 {
			set++
		}
	}
	return set, len(f.live.fog.explored)
}

// LiveCommanded is how many entities the open map screen has been ordered to
// move, which is the residue field whose loss is behavioural rather than
// cosmetic. It is LiveWorld's shape and exists for LiveWorld's reason.
func (f *FrontEnd) LiveCommanded() int {
	if f.live == nil {
		return 0
	}
	return len(f.live.commanded)
}

// CampaignProgressEvidence is the read-only campaign projection exposed to
// no-window developer tools. It contains no install bytes or presentation art.
type CampaignProgressEvidence struct {
	Restored    bool
	Main        int
	Selected    int
	Children    []int
	Announced   []int
	InnMission  []int
	ShopMission []int
	TCMission   []int
	Documents   int
	Markers     []int
	MissionTime uint32
}

// LiveCampaignProgress reports the campaign state installed behind the live
// mission. A fresh campaign reports Restored false.
func (f *FrontEnd) LiveCampaignProgress() CampaignProgressEvidence {
	if f == nil || f.Town == nil || f.Town.progress == nil {
		return CampaignProgressEvidence{}
	}
	p := f.Town.progress
	out := CampaignProgressEvidence{
		Restored: true, Main: p.main.mission, Selected: p.selected,
		InnMission:  append([]int(nil), p.innMission...),
		ShopMission: append([]int(nil), p.shopMission...),
		TCMission:   append([]int(nil), p.tcMission...), Documents: len(f.Town.documents),
		Markers: p.selectedMarkers(), MissionTime: p.missionTime,
	}
	if p.main.announced {
		out.Announced = append(out.Announced, p.main.mission)
	}
	for _, child := range p.children {
		out.Children = append(out.Children, child.mission)
		if child.announced {
			out.Announced = append(out.Announced, child.mission)
		}
	}
	sortInts(out.Children)
	sortInts(out.Announced)
	return out
}

// LiveCompleteCampaign runs the same campaign-completion method the map
// driver's win continuation calls, without synthesizing an outcome. It exists
// for static/headless lawful-install witnesses that need to inspect the loaded
// campaign transition without controlling a desktop window.
func (f *FrontEnd) LiveCompleteCampaign() (int, string, error) {
	if f == nil || f.live == nil || f.live.world == nil || f.liveMission <= 0 {
		return 0, "", errors.New("no live campaign mission")
	}
	ids := []sim.EntityID(nil)
	if f.live.mission != nil {
		ids = append(ids, f.live.mission.ids...)
	}
	party := f.liveParty
	var roster map[sim.EntityID]mapload.PartyMember
	if f.live.mission != nil {
		party = f.live.mission.party
		if f.live.mission.state != nil {
			roster = f.live.mission.state.Start.Roster
		}
	}
	next, line := f.FinishMissionWithRoster(f.liveMission, party, f.live.world, ids, roster)
	if next < 0 {
		return 0, line, errors.New(line)
	}
	return next, line, nil
}
