package game

import (
	"encoding/binary"
	"fmt"
	"math"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// OriginalStateData retains the full signed application domains. A native
// boolean or clamped speed is a display interpretation, not authority to
// replace the original value during conversion.
type OriginalStateData struct {
	Wimpy, ShowHP, FlyingHP, Formation, Speed, ShowTimeFlow int32
	InventoryOpen, BookOpen, Pressed, ViewX, ViewY          int32
	Selection                                               []uint32
}

// SnapshotApplicationState is additive native persistence. Original and Baseline
// preserve source values only while their actual UI producer remains unchanged.
// The current value is always View, never the historical Document state store.
type SnapshotApplicationState struct {
	// LocalOnly captures the three presentation settings of a native mission.
	// Other application fields come from the document when exporting SAV.
	LocalOnly     bool
	Version       uint32
	View          ui.SaveApplicationState
	Baseline      ui.SaveApplicationState
	Original      OriginalStateData
	WimpyBaseline int
}

const applicationStateVersion uint32 = 1

// localOnlyApplicationView retains only the three presentation settings. Other
// fields are neutral valid metadata, not a second native application contract.
func localOnlyApplicationView(v ui.SaveApplicationState) ui.SaveApplicationState {
	return ui.SaveApplicationState{ShowHealth: v.ShowHealth, FlyingHP: v.FlyingHP, TimeFlow: v.TimeFlow,
		Zoom: 1, PeriodUS: terrain.CadencePeriod(terrain.DefaultCadenceRung)}
}

func cloneApplicationState(s *SnapshotApplicationState) *SnapshotApplicationState {
	if s == nil {
		return nil
	}
	out := *s
	// Gob represents an empty slice as nil. Keep one native representation so
	// a source-free native checkpoint preserves the full Snapshot value too.
	out.View.Selection = append([]uint32(nil), s.View.Selection...)
	out.Baseline.Selection = append([]uint32(nil), s.Baseline.Selection...)
	out.Original.Selection = append([]uint32(nil), s.Original.Selection...)
	return &out
}

func validateApplicationState(s *SnapshotApplicationState) error {
	if s == nil {
		return nil
	}
	if s.Version != applicationStateVersion {
		return fmt.Errorf("saved application version %d is unsupported", s.Version)
	}
	if err := ui.ValidateSaveApplication(s.View); err != nil {
		return err
	}
	if err := ui.ValidateSaveApplication(s.Baseline); err != nil {
		return err
	}
	if len(s.Original.Selection) > 1<<16 {
		return fmt.Errorf("original application selection exceeds 65536 actors")
	}
	if s.WimpyBaseline < 0 || s.WimpyBaseline > 2 {
		return fmt.Errorf("saved application has invalid native retreat baseline")
	}
	return nil
}

func readOriginalStateLeaf(doc *sav.DocumentData, path string, kind uint32) (sav.CityStateValueData, error) {
	var value sav.CityStateValueData
	found := false
	for _, record := range doc.State.ValueRecords {
		if record.Path != path {
			continue
		}
		if found || record.Value.Kind != kind {
			return value, fmt.Errorf("original application has ambiguous or invalid %s", path)
		}
		found, value = true, record.Value
	}
	if !found {
		return value, fmt.Errorf("original application is missing %s", path)
	}
	if kind == 2 && len(value.Bytes) != 0 || kind == 6 && (value.Int32 != 0 || len(value.Bytes)%4 != 0) {
		return value, fmt.Errorf("original application has invalid storage for %s", path)
	}
	return value, nil
}

func readOriginalApplicationState(doc *sav.DocumentData) (OriginalStateData, error) {
	var state OriginalStateData
	if doc == nil {
		return state, fmt.Errorf("original application document is absent")
	}
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
		value, err := readOriginalStateLeaf(doc, leaf.path, 2)
		if err != nil {
			return state, err
		}
		*leaf.dst = value.Int32
	}
	selection, err := readOriginalStateLeaf(doc, "/Objects/Selection", 6)
	if err != nil {
		return state, err
	}
	if len(selection.Bytes)/4 > 1<<16 {
		return state, fmt.Errorf("original application selection exceeds 65536 actors")
	}
	state.Selection = make([]uint32, len(selection.Bytes)/4)
	seen := make(map[uint32]bool, len(state.Selection))
	for i := range state.Selection {
		id := binary.LittleEndian.Uint32(selection.Bytes[4*i:])
		if seen[id] {
			return state, fmt.Errorf("original application repeats selection runtime ID %d", id)
		}
		state.Selection[i], seen[id] = id, true
	}
	if _, err := originalPressedSpellID(state.Pressed); err != nil {
		return state, err
	}
	return state, nil
}

func originalPressedSpellID(index int32) (uint32, error) {
	if index == -1 {
		return 0, nil
	}
	if index < 0 || index >= int32(len(originalBookIDs)) {
		return 0, fmt.Errorf("original application current spell index %d outside -1 or 0..23", index)
	}
	return uint32(originalBookIDs[index]), nil
}

// Decode runs only after proving their total, so malformed counts cannot ask
// sav.File.Fog's legacy convenience reader to allocate an unchecked plane.
func originalFogPlane(doc *sav.DocumentData, cells int) ([]byte, error) {
	first, err := readOriginalStateLeaf(doc, "/Fog/FirstState", 2)
	if err != nil {
		return nil, err
	}
	data, err := readOriginalStateLeaf(doc, "/Fog/Data", 6)
	if err != nil {
		return nil, err
	}
	if first.Int32 != 0 && first.Int32 != 0x8000 {
		return nil, fmt.Errorf("original Fog/FirstState is not bit 15")
	}
	if len(data.Bytes) > maxSaveBytes {
		return nil, fmt.Errorf("original Fog runs exceed native save bound")
	}
	total := 0
	for i := 0; i < len(data.Bytes); i += 4 {
		run := int32(binary.LittleEndian.Uint32(data.Bytes[i:]))
		if run < 0 || int64(total)+int64(run) > maxSaveBytes {
			return nil, fmt.Errorf("original Fog run %d is negative or exceeds the plane bound", i/4)
		}
		total += int(run)
	}
	if total == 0 || cells >= 0 && total != cells {
		return nil, fmt.Errorf("original Fog covers %d cells, want %d", total, cells)
	}
	// A negative cell count requests validation without allocating the plane.
	if cells < 0 {
		return nil, nil
	}
	plane := make([]byte, total)
	at, explored := 0, first.Int32 != 0
	for i := 0; i < len(data.Bytes); i += 4 {
		n := int(binary.LittleEndian.Uint32(data.Bytes[i:]))
		if explored {
			for j := at; j < at+n; j++ {
				plane[j] = 1
			}
		}
		at, explored = at+n, !explored
	}
	return plane, nil
}

func validateOriginalApplicationSource(doc *sav.DocumentData) error {
	if doc == nil {
		return nil
	}
	if _, _, err := sav.NativeRandomState(doc.State); err != nil {
		return err
	}
	if _, err := readOriginalApplicationState(doc); err != nil {
		return err
	}
	_, err := originalFogPlane(doc, -1)
	return err
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

func applicationSelection(runtime []uint32, byEntity map[uint32]uint32) ([]uint32, error) {
	selected := make([]uint32, len(runtime))
	for i, id := range runtime {
		found := 0
		for entity, candidate := range byEntity {
			if candidate == id {
				selected[i], found = entity, found+1
			}
		}
		if found == 0 && id != 0 && id <= 0xffff {
			for entity, candidate := range byEntity {
				if uint32(uint16(candidate)) == id {
					selected[i], found = entity, found+1
				}
			}
		}
		if found != 1 {
			return nil, fmt.Errorf("application selection runtime ID %d has %d live bindings", id, found)
		}
	}
	slices.Sort(selected)
	for i := 1; i < len(selected); i++ {
		if selected[i] == selected[i-1] {
			return nil, fmt.Errorf("application selection repeats native actor %d", selected[i])
		}
	}
	return selected, nil
}

func normalizedWimpy(raw int32) int {
	if raw >= 0 && raw <= 2 {
		return int(raw)
	}
	return 0
}

func (mw *mapWorld) restoreApplicationState(s *SnapshotApplicationState, original bool) error {
	if mw == nil || mw.view == nil {
		return fmt.Errorf("application restore needs a live map viewer")
	}
	s = cloneApplicationState(s)
	if s == nil {
		// Legacy native snapshots did not record the current application. Their
		// retained original Document cannot establish what the player changed.
		// Preserve their nil additive field and ordinary native continuation.
		if !original {
			return nil
		}
		if mw.mission == nil || mw.mission.state == nil || mw.mission.state.savedDocument == nil || mw.mission.state.savedDocument.Document == nil {
			return nil
		}
		document := mw.mission.state.savedDocument
		raw, err := readOriginalApplicationState(document.Document)
		if err != nil {
			return err
		}
		ids, err := applicationActorIDs(document.Document, document.Actors, mw.world)
		if err != nil {
			return err
		}
		selection, err := applicationSelection(raw.Selection, ids)
		if err != nil {
			return err
		}
		view := mw.view.SaveApplication()
		view.Selection, view.ViewX, view.ViewY, view.Zoom = selection, float64(raw.ViewX), float64(raw.ViewY), 1
		view.InventoryOpen, view.SpellBookOpen = raw.InventoryOpen != 0, raw.BookOpen != 0
		view.ShowHealth, view.FlyingHP, view.TimeFlow = raw.ShowHP != 0, raw.FlyingHP != 0, raw.ShowTimeFlow != 0
		view.PressedSpell, _ = originalPressedSpellID(raw.Pressed)
		view.PeriodUS, view.Unpaced = terrain.SpeedIndexPeriod(int(raw.Speed)), false
		actions, err := readCurrentActions(document.Document)
		if err != nil {
			return err
		}
		applyCurrentView(&view, raw, actions)
		s = &SnapshotApplicationState{Version: applicationStateVersion, View: view, Baseline: view, Original: raw, WimpyBaseline: normalizedWimpy(raw.Wimpy)}
	}
	if err := validateApplicationState(s); err != nil {
		return err
	}
	if original {
		if mw.fog == nil {
			return fmt.Errorf("original application has no native fog plane")
		}
		plane, err := originalFogPlane(mw.mission.state.savedDocument.Document, len(mw.fog.explored))
		if err != nil {
			return err
		}
		copy(mw.fog.explored, plane)
		for i := range mw.fog.visible {
			mw.fog.visible[i] &= plane[i]
		}
		mw.push()
	}
	if s.LocalOnly {
		// Native option persistence must not adopt the full imported-application
		// restore path: that also changes selection, spell, panels and camera.
		mw.applyGameOption(ui.GameOptionDayNight, boolOption(s.View.TimeFlow))
		mw.applyGameOption(ui.GameOptionHealth, boolOption(s.View.ShowHealth))
		mw.applyGameOption(ui.GameOptionDamage, boolOption(s.View.FlyingHP))
		mw.view.RestoreScheduledLightClock(mw.world.Tick())
		mw.applicationState = s
		return nil
	}
	if err := mw.view.RestoreSaveApplication(s.View); err != nil {
		return err
	}
	// The new selection changes the displayed book. That push clears the old
	// spell as usual, then the saved Pressed field is installed on the new book.
	mw.push()
	if err := mw.view.RestoreSaveApplication(s.View); err != nil {
		return err
	}
	// SetTimeFlow forces a relight at the current tick. LOAD restores the last
	// scheduled lighting cache after installing the saved time-flow setting.
	mw.view.RestoreScheduledLightClock(mw.world.Tick())
	mw.setCadenceMode(s.View.PeriodUS, s.View.PlayerPaused, s.View.Unpaced, false)
	if original {
		actions, err := readCurrentActions(mw.mission.state.savedDocument.Document)
		if err != nil {
			return err
		}
		if actions != nil && actions.Session != nil && actions.Session.View != nil && actions.Session.View.Animation != nil {
			if err := mw.view.RestoreAnimation(*actions.Session.View.Animation); err != nil {
				return err
			}
		}
	}
	mw.applicationState = s
	return nil
}

func (f *FrontEnd) captureApplicationState(mw *mapWorld) (*SnapshotApplicationState, error) {
	if mw == nil || mw.view == nil {
		return nil, nil
	}
	out := cloneApplicationState(mw.applicationState)
	if (out == nil || out.LocalOnly) && mw.mission != nil && mw.mission.state != nil && mw.mission.state.savedDocument != nil && mw.mission.state.savedDocument.Document != nil {
		raw, err := readOriginalApplicationState(mw.mission.state.savedDocument.Document)
		if err != nil {
			return nil, err
		}
		if out == nil {
			out = &SnapshotApplicationState{Version: applicationStateVersion, Baseline: mw.view.SaveApplication(), WimpyBaseline: normalizedWimpy(raw.Wimpy)}
		} else {
			raw.Wimpy, raw.Formation = out.Original.Wimpy, out.Original.Formation
		}
		// A newly captured selection comes from the viewer, including empty.
		raw.Selection, out.Baseline.Selection = nil, nil
		out.Original, out.LocalOnly = raw, false
		out.Baseline.ShowHealth, out.Baseline.FlyingHP, out.Baseline.TimeFlow = raw.ShowHP != 0, raw.FlyingHP != 0, raw.ShowTimeFlow != 0
		out.Baseline.InventoryOpen, out.Baseline.SpellBookOpen = raw.InventoryOpen != 0, raw.BookOpen != 0
	}
	if out == nil {
		out = &SnapshotApplicationState{Version: applicationStateVersion, WimpyBaseline: f.wimpyMode,
			Original: OriginalStateData{Wimpy: int32(f.wimpyMode), Speed: int32(nearestOriginalSpeed(mw.clock.Period()))},
			Baseline: ui.SaveApplicationState{Zoom: 1, PeriodUS: mw.clock.Period()}}
	}
	out.View = mw.view.SaveApplication()
	// MapCadence's world-side reader is the authority even before App adoption.
	out.View.PeriodUS, out.View.Unpaced = mw.clock.Period(), mw.unpaced
	if out.LocalOnly {
		out.Original.ShowHP, out.Original.FlyingHP, out.Original.ShowTimeFlow = boolToInt32(out.View.ShowHealth), boolToInt32(out.View.FlyingHP), boolToInt32(out.View.TimeFlow)
		out.LocalOnly = false
	}
	if f.wimpyMode != out.WimpyBaseline {
		out.Original.Wimpy = int32(f.wimpyMode)
		out.WimpyBaseline = normalizedWimpy(int32(f.wimpyMode))
	}
	out = cloneApplicationState(out)
	if err := validateApplicationState(out); err != nil {
		return nil, err
	}
	return out, nil
}

func (mw *mapWorld) rememberApplicationFormation(value int32) {
	if mw != nil && mw.applicationState != nil {
		mw.applicationState.Original.Formation = value
	}
}

func boolToInt32(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

// viewOriginFloor is the original client's own scroll-origin lower bound on
// each axis, never itself relief-aware (SESS-VIEW-030, High): the outermost
// 8 cells of a map are never the top-left of its viewport, on either axis,
// regardless of terrain height in the visible columns. This engine's own
// displaced-mode camera clamp derives its floor from the projected playable
// terrain instead, which correctly lets the player look at raised ground
// near a jagged top edge but, on that same edge, can position the camera
// one row higher than the original's flat 8-cell rule ever does: a discovered
// corpus census decoding `/View/X` and `/View/Y` from every plain-dated
// `gameversions/saves/<date>/*.sav` file (62 files, every date directory,
// non-recursive so a nested experiment subdirectory is not walked) found
// every single one either at or above 8 on BOTH axes, or at exactly (0, 0)
// on both -- never a nonzero value under 8 on either axis alone. The exact
// zero pair belongs to the corpus's own ROM1-written town-save population,
// which carries no world camera to record; see originalViewOrigin below for
// why that pair is left unfloored rather than treated as a corpus
// counterexample. Written here,
// not in the live camera clamp: the reported defect is what this engine's
// own SAV writer puts in the file the original client then loads, not a
// live in-session scroll limit this story's evidence reaches.
const viewOriginFloor = 8

// originalViewOrigin floors a genuinely established camera and leaves an
// exact (0, 0) pair alone. The camera exists as soon as a mission's Viewer
// is constructed (terrain.Project seeds it whenever altitude data validates),
// so a Zoom or projection check cannot tell a laid-out viewport from one that
// never received its own Layout call. Every synthetic-document test fixture
// in this tree that skips Layout carries exactly ViewX=0, ViewY=0, the same
// exact pair the corpus census backing viewOriginFloor found on every town
// save that has no world camera to record -- a zero PAIR is this build's own
// "no camera captured" value, not a position the original's floor was ever
// meant to correct, and the census never found a genuine file at a nonzero
// value under 8 on one axis alone.
func originalViewOrigin(x, y float64) (int32, int32) {
	if x == 0 && y == 0 {
		return 0, 0
	}
	return originalViewFloor(x), originalViewFloor(y)
}

func originalViewFloor(cells float64) int32 {
	if cells < viewOriginFloor {
		return viewOriginFloor
	}
	return int32(math.Round(cells))
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
	pressed, err := quickSpellsToOriginalIndices([4]uint32{now.PressedSpell})
	if err != nil {
		return out, worldSaveUnsupportedf("current spell %d has no original book index", now.PressedSpell)
	}
	out.Pressed = pressed[0]
	return out, nil
}

func nearestOriginalSpeed(period int) int {
	best, distance := terrain.SpeedIndexMin, math.MaxInt
	for index := terrain.SpeedIndexMin; index <= terrain.SpeedIndexMax; index++ {
		delta := period - terrain.SpeedIndexPeriod(index)
		if delta < 0 {
			delta = -delta
		}
		if delta < distance {
			best, distance = index, delta
		}
	}
	return best
}

func projectApplicationState(doc *sav.DocumentData, s Snapshot, world *sim.World) error {
	if doc == nil || s.SavedDocument == nil || world == nil {
		return fmt.Errorf("current application lacks exact document bindings")
	}
	ids, err := applicationActorIDs(doc, s.SavedDocument.Actors, world)
	if err != nil {
		return err
	}
	app := s.ApplicationState
	currentSelection := app != nil && !app.LocalOnly
	if app == nil || app.LocalOnly {
		if err := validateApplicationState(app); err != nil {
			return err
		}
		original, err := readOriginalApplicationState(doc)
		if err != nil {
			return err
		}
		selection, err := applicationSelection(original.Selection, ids)
		if err != nil {
			return err
		}
		pressed, err := originalPressedSpellID(original.Pressed)
		if err != nil {
			return err
		}
		base := ui.SaveApplicationState{Selection: selection, ViewX: float64(original.ViewX), ViewY: float64(original.ViewY), Zoom: 1,
			InventoryOpen: original.InventoryOpen != 0, SpellBookOpen: original.BookOpen != 0,
			ShowHealth: original.ShowHP != 0, FlyingHP: original.FlyingHP != 0, TimeFlow: original.ShowTimeFlow != 0,
			PressedSpell: pressed, PeriodUS: terrain.SpeedIndexPeriod(int(original.Speed))}
		now := base
		wimpyBaseline := normalizedWimpy(original.Wimpy)
		if app != nil {
			now.ShowHealth, now.FlyingHP, now.TimeFlow = app.View.ShowHealth, app.View.FlyingHP, app.View.TimeFlow
			original.Wimpy, original.Formation = app.Original.Wimpy, app.Original.Formation
			wimpyBaseline = app.WimpyBaseline
		}
		if s.CameraSet {
			now.ViewX, now.ViewY, now.Zoom = s.CameraX, s.CameraY, s.CameraZoom
		}
		app = &SnapshotApplicationState{Version: applicationStateVersion, View: now, Baseline: base,
			Original: original, WimpyBaseline: wimpyBaseline}
	}
	raw, err := applicationCurrentRaw(app)
	if err != nil {
		return err
	}
	selection := make([]uint32, len(app.View.Selection))
	for i, entity := range app.View.Selection {
		id, found := ids[entity]
		if !found {
			return worldSaveUnsupportedf("selected native actor %d has no original runtime binding", entity)
		}
		// Current actor bindings already identify the selected objects. The
		// final runtime projection assigns their unique ordinary IDs.
		selection[i] = id
	}
	if !currentSelection && slices.Equal(app.View.Selection, app.Baseline.Selection) {
		mapped, err := applicationSelection(raw.Selection, ids)
		if err != nil {
			return err
		}
		if !slices.Equal(mapped, app.View.Selection) {
			return fmt.Errorf("current application selection baseline disagrees with runtime bindings")
		}
		selection = slices.Clone(raw.Selection)
	}
	shortcuts, err := quickSpellsToOriginalIndices(s.QuickSpells)
	if err != nil {
		return worldSaveUnsupportedf("quick spells: %v", err)
	}
	r := s.Residue
	if r.FogCols <= 0 || r.FogRows <= 0 || uint64(r.FogCols) > uint64(maxSaveBytes)/uint64(r.FogRows) || len(r.FogExplored) != r.FogCols*r.FogRows {
		return worldSaveUnsupportedf("current Fog exploration has invalid dimensions or count")
	}
	if b := world.Bounds(); int64(r.FogCols) != int64(b.Width) || int64(r.FogRows) != int64(b.Height) {
		return worldSaveUnsupportedf("current Fog dimensions disagree with World")
	}
	if err := validateNativeFogResidue(r); err != nil {
		return err
	}
	for i, bit := range r.FogExplored {
		if bit > 1 {
			return fmt.Errorf("current Fog exploration cell %d is not a bit", i)
		}
	}
	first := int32(r.FogExplored[0]) * 0x8000
	var runs []uint32
	start := 0
	for i := 1; i <= len(r.FogExplored); i++ {
		if i == len(r.FogExplored) || r.FogExplored[i] != r.FogExplored[start] {
			runs = append(runs, uint32(i-start))
			start = i
		}
	}
	ints := map[string]int32{
		"/CurrentState/InBattle": 1, "/GameOptions/Wimpy": raw.Wimpy, "/GameOptions/ShowHP": raw.ShowHP,
		"/GameOptions/FlyingHP": raw.FlyingHP, "/GameOptions/Formation": raw.Formation, "/GameOptions/Speed": raw.Speed,
		"/GameOptions/ShowTimeFlow": raw.ShowTimeFlow, "/Inventory/IsOpen": raw.InventoryOpen,
		"/SpellBook/IsOpen": raw.BookOpen, "/SpellBook/Pressed": raw.Pressed, "/View/X": raw.ViewX, "/View/Y": raw.ViewY,
		"/Fog/FirstState": first,
	}
	arrays := map[string][]uint32{"/Fog/Data": runs, "/Objects/Selection": selection,
		"/SpellBook/Shortcuts": {uint32(shortcuts[0]), uint32(shortcuts[1]), uint32(shortcuts[2]), uint32(shortcuts[3])}}
	// Validate all addressed leaves before adopting even one change.
	for path := range ints {
		if _, err := readOriginalStateLeaf(doc, path, 2); err != nil {
			return err
		}
	}
	for path := range arrays {
		if _, err := readOriginalStateLeaf(doc, path, 6); err != nil {
			return err
		}
	}
	records := slices.Clone(doc.State.ValueRecords)
	for i := range records {
		if value, ok := ints[records[i].Path]; ok {
			records[i].Value = sav.CityStateValueData{Kind: 2, Int32: value}
		}
		if values, ok := arrays[records[i].Path]; ok {
			data := make([]byte, 4*len(values))
			for j, value := range values {
				binary.LittleEndian.PutUint32(data[4*j:], value)
			}
			records[i].Value = sav.CityStateValueData{Kind: 6, Bytes: data}
		}
	}
	doc.State.ValueRecords = records
	return nil
}

// Legacy bounded imports without complete Document authority keep their old
// bootstrap. Only an exact authored leaf can install a different native stream.
func restoreOriginalRandomState(ms *Mission) error {
	if ms.savedDocument == nil || ms.savedDocument.Document == nil {
		return nil
	}
	value, present, err := sav.NativeRandomState(ms.savedDocument.Document.State)
	if err != nil {
		return err
	}
	if present {
		ms.World.RestoreRandomState(value)
	}
	return nil
}
