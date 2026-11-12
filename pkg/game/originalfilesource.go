package game

import (
	"encoding/binary"
	"fmt"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// originalFileSource is one place an original file states a value the game
// adopts: a between-mission file's Player groups, a mission's application
// state. A rule that differs between the sources is a column every row
// carries, so a repair to reading or identity has one row to go to and the
// callers name a phase, never a game. A column the source does not state is
// nil. The second game's actor graph and session are the full inputs; the
// first game's city and Player are the same inputs in reduced form.
type originalFileSource struct {
	name string
	// townHolds is this row's test of a between-mission file; the first row
	// that holds is the town's group source, and none holding is no source.
	townHolds func(originalTownInputs) bool
	// townGroups reads the groups the town adopts.
	townGroups func(originalTownInputs) ([]cityLiveGroup, error)
	// missionHolds is this row's test of a mission LOAD at the read moment,
	// after the import has bound the file's actors.
	missionHolds func(*originalSource) bool
	// readApplication is the row's read of the mission's application state
	// at that moment.
	readApplication func(*originalSource, *Mission) (*fileApplication, error)
	// frameHolds is this row's test of a mission's first frame.
	frameHolds func(ms *Mission, original bool) bool
	// frame is what the row restores on the first frame.
	frame              *firstFrameSource
	captureHolds       func(*mapWorld) bool
	captureApplication func(*mapWorld) (OriginalStateData, error)
	adoptMission       func(*originalSource, *Mission)
}

// originalTownInputs is what a between-mission LOAD states about its file
// when it chooses a group source.
type originalTownInputs struct {
	src          *originalSource
	session      *CampaignSession
	hasCurrent   bool
	cityDocument originalCityDocument
	cityErr      error
	actions      *currentActionData
	party        []mapload.PartyMember
	chars        []sav.Character
}

// originalFileSources is the table: one row per source, in the order a load
// tries them.
func originalFileSources() []originalFileSource {
	return []originalFileSource{
		{
			name:      "current SAV document",
			townHolds: func(in originalTownInputs) bool { return in.hasCurrent && in.cityErr == nil },
			townGroups: func(in originalTownInputs) ([]cityLiveGroup, error) {
				if p, ok := in.cityDocument.(interface{ Data() sav.CityData }); ok {
					return cityGroupsFromCurrent(p.Data(), in.actions), nil
				}
				return nil, nil
			},
		},
		{
			name:      "file session and actor graph",
			townHolds: func(in originalTownInputs) bool { return in.src.campaign.fileSession != nil },
			townGroups: func(in originalTownInputs) ([]cityLiveGroup, error) {
				graph, err := in.src.sf.ActorGraph()
				if err != nil {
					return nil, fmt.Errorf("original between-mission save: %w", err)
				}
				return cityGroupsFromGraph(graph, in.party, in.chars), nil
			},
		},
		{
			name:      "city provenance",
			townHolds: func(in originalTownInputs) bool { return !in.hasCurrent },
			townGroups: func(in originalTownInputs) ([]cityLiveGroup, error) {
				return cityGroupsFromLoaded(in.party, in.session.originalCity.snapshot()), nil
			},
		},
		{
			name:       "decoded document",
			frameHolds: func(_ *Mission, original bool) bool { return original },
			frame:      documentFirstFrame(),
		},
		{
			name:         "file state store",
			missionHolds: func(src *originalSource) bool { return src.fileView && src.missionDocument.Document == nil },
			readApplication: func(src *originalSource, ms *Mission) (*fileApplication, error) {
				return readFileApplication(src.sf, ms.World)
			},
			frameHolds: func(ms *Mission, _ bool) bool { return ms.fileApplication != nil },
			frame:      fileStoreFirstFrame(),
		},
		{
			name: "captured mission document",
			captureHolds: func(mw *mapWorld) bool {
				return (mw.applicationState == nil || mw.applicationState.LocalOnly) && mw.mission != nil && mw.mission.state != nil && mw.mission.state.savedDocument != nil && mw.mission.state.savedDocument.Document != nil
			},
			captureApplication: func(mw *mapWorld) (OriginalStateData, error) {
				return readOriginalApplicationState(mw.mission.state.savedDocument.Document)
			},
		},
		{
			name: "building caster list",
			adoptMission: func(src *originalSource, ms *Mission) {
				applyOriginalBuildingCasters(ms, src.virtualCasters)
			},
		},
	}
}

// adoptTownGroups is the between-mission LOAD's adoption of the town's groups
// from the source the table chooses for its file.
func (s *CampaignSession) adoptTownGroups(in originalTownInputs) error {
	s.Town.cityGroups, s.Town.cityGroupsLoaded = nil, true
	row := originalTownSource(in)
	if row == nil {
		return nil
	}
	groups, err := row.townGroups(in)
	if err != nil {
		return err
	}
	s.Town.cityGroups = groups
	return nil
}

// originalTownSource is the first row of the table holding for a
// between-mission file; nil where the file states no group source.
func originalTownSource(in originalTownInputs) *originalFileSource {
	for _, row := range originalFileSources() {
		if row.townHolds != nil && row.townHolds(in) {
			return &row
		}
	}
	return nil
}

// readMissionApplication is the read moment of a mission LOAD: the source the
// table names reads the application state the first frame later adopts.
func (src *originalSource) readMissionApplication(ms *Mission) error {
	for _, row := range originalFileSources() {
		if row.missionHolds == nil || !row.missionHolds(src) {
			continue
		}
		app, err := row.readApplication(src, ms)
		if err != nil {
			return err
		}
		ms.fileApplication = app
		return nil
	}
	return nil
}

func (src *originalSource) adoptMissionState(ms *Mission) {
	for _, row := range originalFileSources() {
		if row.adoptMission != nil {
			row.adoptMission(src, ms)
		}
	}
}

func readOriginalApplicationState(doc *sav.DocumentData) (OriginalStateData, error) {
	if doc == nil {
		return OriginalStateData{}, fmt.Errorf("original application document is absent")
	}
	return originalApplicationStateOf(doc.State, originalBookIDs)
}

// originalApplicationStateOf reads the application leaves of a state store;
// rows is the book slot table the Pressed index names.
func originalApplicationStateOf(store sav.DocumentStateData, rows []uint16) (OriginalStateData, error) {
	var state OriginalStateData
	for _, leaf := range []struct {
		path string
		dst  *int32
	}{
		{"/GameOptions/Wimpy", &state.Wimpy}, {"/GameOptions/ShowHP", &state.ShowHP},
		{"/GameOptions/FlyingHP", &state.FlyingHP}, {"/GameOptions/Formation", &state.Formation},
		{"/GameOptions/Speed", &state.Speed}, {"/GameOptions/ShowTimeFlow", &state.ShowTimeFlow},
		{"/Inventory/IsOpen", &state.InventoryOpen}, {"/SpellBook/IsOpen", &state.BookOpen},
		{"/SpellBook/Pressed", &state.Pressed}, {"/View/X", &state.ViewX}, {"/View/Y", &state.ViewY},
	} {
		value, err := readStateLeaf(store, leaf.path, 2)
		if err != nil {
			return state, err
		}
		*leaf.dst = value.Int32
	}
	selection, err := readStateLeaf(store, "/Objects/Selection", 6)
	if err != nil {
		return state, err
	}
	if len(selection.Bytes)/4 > 1<<16 {
		return state, fmt.Errorf("original application selection exceeds 65536 actors")
	}
	state.Selection = make([]uint32, len(selection.Bytes)/4)
	// A repeated id is checked against the actors that carry it when the
	// selection is bound (applicationSelection).
	for i := range state.Selection {
		state.Selection[i] = binary.LittleEndian.Uint32(selection.Bytes[4*i:])
	}
	if _, err := pressedSpellID(rows, state.Pressed); err != nil {
		return state, err
	}
	return state, nil
}

// readFileApplication reads a mission file's application state for a world
// whose actors the file's import has bound.
func readFileApplication(sf *sav.File, w *sim.World) (*fileApplication, error) {
	store, present, err := sf.StateData()
	if err != nil || !present {
		return nil, err
	}
	raw, err := originalApplicationStateOf(store, data.SpellPresentationOf(w.SpellGame()).BookRows)
	if err != nil {
		return nil, err
	}
	ids := map[uint32]uint32{}
	for _, e := range w.Entities() {
		if e.Alive() && e.SourceBinding.Class >= 1 && e.SourceBinding.Class <= 3 {
			ids[uint32(e.ID)] = e.SourceBinding.RuntimeID
		}
	}
	return &fileApplication{raw: raw, ids: ids, structures: applicationStructureIDs(w)}, nil
}

// Actor bindings keep runtime IDs distinct from saved addresses and object
// indices. DIV-1184 records the selection policy.
func applicationActorIDs(doc *sav.DocumentData, bindings []SnapshotSAVActor, world *sim.World) (map[uint32]uint32, error) {
	if doc == nil || world == nil {
		return nil, fmt.Errorf("application selection needs a document and World")
	}
	entities := make(map[sim.EntityID]sim.Entity)
	for _, e := range world.Entities() {
		entities[e.ID] = e
	}
	actions, err := readCurrentActions(doc)
	if err != nil {
		return nil, err
	}
	coordinates := map[uint16]struct {
		Entity sim.EntityID
		Value  sim.ActorRuntimeCoordinate
	}{}
	if actions != nil {
		for _, binding := range actions.Bindings {
			value, present := actions.Values[binding.ID]
			if binding.Structure || binding.Missing || !present || !value.SourceBound || value.RuntimeID == nil {
				continue
			}
			if _, duplicate := coordinates[binding.Object]; duplicate {
				return nil, fmt.Errorf("application selection repeats a current runtime coordinate")
			}
			coordinates[binding.Object] = struct {
				Entity sim.EntityID
				Value  sim.ActorRuntimeCoordinate
			}{binding.ID, *value.RuntimeID}
		}
	}
	byEntity := make(map[uint32]uint32, len(bindings))
	seenObjects := make(map[uint16]bool, len(bindings))
	for _, binding := range bindings {
		if binding.Retired {
			continue
		}
		if binding.ObjectIndex == 0 || int(binding.ObjectIndex) > len(doc.Objects) || seenObjects[binding.ObjectIndex] {
			return nil, fmt.Errorf("application selection has an invalid or repeated actor object binding")
		}
		e, exists := entities[binding.EntityID]
		record := &doc.Objects[binding.ObjectIndex-1]
		if !exists || !currentActorRecordMatches(world, e, *record) {
			return nil, fmt.Errorf("application selection actor %d has no exact live binding", binding.EntityID)
		}
		id, err := savedStructureValue(record, "RuntimeID")
		if err != nil {
			return nil, fmt.Errorf("application selection actor %d has no runtime ID: %w", binding.EntityID, err)
		}
		if e.SourceBinding.Class != 0 && id != e.SourceBinding.RuntimeID {
			coordinate, ok := coordinates[binding.ObjectIndex]
			if !ok || coordinate.Entity != binding.EntityID || coordinate.Value.Wire != id || coordinate.Value.Value != e.SourceBinding.RuntimeID || coordinate.Value.Wire > 65535 {
				return nil, fmt.Errorf("application selection actor %d disagrees with its source runtime ID", binding.EntityID)
			}
		}
		if _, duplicate := byEntity[uint32(binding.EntityID)]; duplicate {
			return nil, fmt.Errorf("application selection repeats native actor %d", binding.EntityID)
		}
		byEntity[uint32(binding.EntityID)] = id
		seenObjects[binding.ObjectIndex] = true
	}
	return byEntity, nil
}

// fileStoreFirstFrame is a mission file without a complete document: the
// state store's application leaves, read at LOAD with the actor IDs of the
// import's source bindings (readFileApplication). The store holds no engine
// action leaf, so it supplies no current view and no animation; its fog
// plane is the opener's, and the first frame adds none.
func fileStoreFirstFrame() *firstFrameSource {
	return &firstFrameSource{
		application: func(mw *mapWorld) (*fileApplication, []uint16, error) {
			return mw.mission.state.fileApplication, mw.spells().BookRows, nil
		},
	}
}

// documentFirstFrame is an original save's decoded document: its
// application leaves with the actor IDs of the document's actor table, its
// current view and animation, and its fog plane over the opener's.
func documentFirstFrame() *firstFrameSource {
	return &firstFrameSource{
		application: documentApplication,
		current: func(mw *mapWorld) (*currentActionData, error) {
			return readCurrentActions(mw.mission.state.savedDocument.Document)
		},
		fog: func(mw *mapWorld, cells int) ([]byte, error) {
			return originalFogPlane(mw.mission.state.savedDocument.Document, cells)
		},
		animation: true,
	}
}

func documentApplication(mw *mapWorld) (*fileApplication, []uint16, error) {
	document := mw.mission.state.savedDocument
	if document == nil || document.Document == nil {
		return nil, nil, nil
	}
	raw, err := readOriginalApplicationState(document.Document)
	if err != nil {
		return nil, nil, err
	}
	ids, err := applicationActorIDs(document.Document, document.Actors, mw.world)
	if err != nil {
		return nil, nil, err
	}
	return &fileApplication{raw: raw, ids: ids, structures: applicationStructureIDs(mw.world)}, originalBookIDs, nil
}

// firstFrameSourceOf is the source of a mission's first frame, the table's
// first row holding for the mission: the decoded document of an original
// save, else a mission file's state store; nil where the mission has neither.
func firstFrameSourceOf(ms *Mission, original bool) *firstFrameSource {
	if ms == nil {
		return nil
	}
	for _, row := range originalFileSources() {
		if row.frameHolds != nil && row.frameHolds(ms, original) {
			return row.frame
		}
	}
	return nil
}

// cityGroupsFromLoaded is the live membership of a town loaded from an
// original file: each loaded group with the party members it held, in loaded
// order, resolved through the provenance's identity bindings. It is nil when
// the provenance is unavailable.
func cityGroupsFromLoaded(party []mapload.PartyMember, loaded *SnapshotOriginalCity) []cityLiveGroup {
	if loaded == nil || loaded.Unavailable != "" {
		return nil
	}
	source := loaded.Document
	boundTo := map[uint32]string{}
	for _, b := range loaded.Bindings {
		boundTo[b.Identity] = b.PartyID
	}
	member := map[string]bool{}
	for _, p := range party {
		member[p.ID] = true
	}
	return cityGroupsOver(loadedPlayerGroups(source), func(a uint16) (string, bool) {
		if a == 0 || int(a) > len(source.Objects) || source.Objects[a-1].Unit == nil || len(source.Objects[a-1].Unit.Token) != 37 {
			return "", false
		}
		id := boundTo[binary.LittleEndian.Uint32(source.Objects[a-1].Unit.Token[29:33])]
		return id, member[id]
	})
}

// cityGroupsFromGraph is the live membership of a town loaded from a file
// whose city has no first-game records: each group of the hero's Player with
// the party members its actors are, an actor named by its identity key
// through chars, which parallels party.
func cityGroupsFromGraph(graph sav.SavedActorGraph, party []mapload.PartyMember, chars []sav.Character) []cityLiveGroup {
	if len(party) != len(chars) {
		return nil
	}
	byIdentity := map[uint32]int{}
	for i, c := range chars {
		byIdentity[c.Key] = i
	}
	var key uint32
	byArchive := map[uint16]string{}
	for _, a := range graph.Actors {
		i, ok := byIdentity[a.Identity]
		if !ok {
			continue
		}
		byArchive[a.ArchiveIndex] = party[i].ID
		if party[i].StartingHero {
			key = a.Owner.Key
		}
	}
	var groups []sav.CityGroupData
	for _, g := range graph.Groups {
		if key != 0 && g.Owner.Key == key {
			groups = append(groups, sav.CityGroupData{Words20: g.Words, Raw80: g.AI[:], Words3c: g.AIWords,
				Actors: g.Members, F1c: g.Selector, F40: g.Reference.Key, F44: g.Owner.Key})
		}
	}
	return cityGroupsOver(loadedGroups(key, groups), func(a uint16) (string, bool) {
		id, ok := byArchive[a]
		return id, ok
	})
}

// cityGroupsFromCurrent is the live membership of a town loaded from a current
// SAV: each saved group with the party members its actors are bound to. The
// binding is the document's own, so it holds with or without original
// provenance.
func cityGroupsFromCurrent(city sav.CityData, a *currentActionData) []cityLiveGroup {
	if a == nil {
		return nil
	}
	entityID := make(map[sim.EntityID]string, len(a.Party))
	for _, p := range a.Party {
		entityID[p.Entity] = string(p.ID)
	}
	objectID := map[uint16]string{}
	for _, b := range a.Bindings {
		if id, ok := entityID[b.ID]; ok && !b.Structure && !b.Missing {
			objectID[b.Object] = id
		}
	}
	return cityGroupsOver(loadedPlayerGroups(city), func(object uint16) (string, bool) {
		id, ok := objectID[object]
		return id, ok
	})
}

func applicationCurrentRaw(app *SnapshotApplicationState) (OriginalStateData, error) {
	if app == nil {
		return OriginalStateData{}, worldSaveUnsupportedf("current application state is absent")
	}
	if err := validateApplicationState(app); err != nil {
		return OriginalStateData{}, err
	}
	out, now, base := app.Original, app.View, app.Baseline
	// Preserve full source domains while their native interpretation is unchanged.
	if now.ShowHealth != base.ShowHealth {
		out.ShowHP = boolToInt32(now.ShowHealth)
	}
	if now.FlyingHP != base.FlyingHP {
		out.FlyingHP = boolToInt32(now.FlyingHP)
	}
	if now.TimeFlow != base.TimeFlow {
		out.ShowTimeFlow = boolToInt32(now.TimeFlow)
	}
	if now.InventoryOpen != base.InventoryOpen {
		out.InventoryOpen = boolToInt32(now.InventoryOpen)
	}
	if now.SpellBookOpen != base.SpellBookOpen {
		out.BookOpen = boolToInt32(now.SpellBookOpen)
	}
	if now.PeriodUS != terrain.SpeedIndexPeriod(int(out.Speed)) {
		out.Speed = int32(nearestOriginalSpeed(now.PeriodUS))
	}
	out.ViewX, out.ViewY = originalViewOrigin(now.ViewX, now.ViewY)
	pressed, err := quickSpellsToOriginalIndices([9]uint32{now.PressedSpell})
	if err != nil {
		return out, worldSaveUnsupportedf("current spell %d has no original book index", now.PressedSpell)
	}
	out.Pressed = pressed[0]
	return out, nil
}
