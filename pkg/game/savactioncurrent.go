package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

type currentActionBinding struct {
	ID        sim.EntityID
	Structure bool
	Object    uint16
	Missing   bool
}
type currentActionObject struct {
	ID     sim.SavedObjectID
	Object uint16
	Item   *sim.SavedItemObject   `json:",omitempty"`
	Effect *sim.SavedEffectObject `json:",omitempty"`
	Spell  *sim.SavedSpellObject  `json:",omitempty"`
}
type currentGroupContinuation struct {
	Object       uint16
	Inline       uint32
	ID           uint32
	Authored     bool
	RootOnly     bool
	RootSelector uint32
	RootMembers  []sim.EntityID
}
type currentNativeArea struct {
	Object uint16
	Policy sim.CurrentAreaPolicy
}
type currentNativeDelivery struct {
	Object uint16
	Policy sim.CurrentDeliveryPolicy
}
type currentActionData struct {
	structureGenerated   map[uint32]bool
	StructureBindings    *[]currentStructureBinding   `json:",omitempty"`
	NativeBasisWires     []currentNativeBasisWire     `json:",omitempty"`
	HeldNativeBasisWires bool                         `json:",omitempty"`
	RemovedNativeBases   []sim.NativeActorBasisRecord `json:",omitempty"`
	Version              uint32
	NativeHistoryVersion uint8              `json:",omitempty"`
	WorldMapReturn       *SnapshotMapReturn `json:",omitempty"`
	Bindings             []currentActionBinding
	Objects              []currentActionObject
	Actions              sim.ActionContinuations
	Program              *currentScriptProgram `json:",omitempty"`
	// A terminal body is owned by SAV's dead list. Its dormant action residue
	// remains with that exact root; it never resurrects a ticking Entity.
	Held                 []sim.ActorContinuation
	TerminalMotions      []currentTerminalMotion `json:",omitempty"`
	Fog                  *currentMissionFog      `json:",omitempty"`
	PendingMessages      []int32                 `json:",omitempty"`
	Options              []PendingGameOption
	Pending              *currentPendingQueue `json:",omitempty"`
	Bolts                []SnapshotSpellBolt
	Heals                []SnapshotHealBurst
	Runs                 []SnapshotCastRun
	Animation            []currentAnimationClock `json:",omitempty"`
	DeathAges            []currentDeathAge       `json:",omitempty"`
	Manifest             *currentActorManifest   `json:",omitempty"`
	VisualIdentities     []SnapshotVisualIdentity
	VisualNext           sim.EntityID
	GroupTag             uint32
	Groups               []currentGroupContinuation
	GroupHighWater       uint32
	GroupPlayers         *bool                            `json:",omitempty"`
	GroupFormations      *bool                            `json:",omitempty"`
	GroupParticipants    *bool                            `json:",omitempty"`
	PlayerIdentities     *[]currentPlayerIdentity         `json:",omitempty"`
	DepartedCharacters   []uint32                         `json:",omitempty"`
	AbsentPlayers        []currentAbsentPlayer            `json:",omitempty"`
	ArchiveCoordinates   []currentArchiveCoordinate       `json:",omitempty"`
	ActorGroups          []currentActorGroupCoordinate    `json:",omitempty"`
	PlayerSlots          []currentPlayerSlot              `json:",omitempty"`
	AbsentSessionHead    []byte                           `json:",omitempty"`
	AbsentCellTails      currentCellAbsence               `json:",omitempty"`
	AbsentCellRecords    currentCellAbsence               `json:",omitempty"`
	AbsentMotionCells    currentCellAbsence               `json:",omitempty"`
	AbsentBlocks         currentBlockAbsence              `json:",omitempty"`
	AbsentDiaries        []currentAbsentDiary             `json:",omitempty"`
	AbsentCellSacks      []currentAbsentCellSack          `json:",omitempty"`
	AbsentStructureCells []sim.CurrentAbsentStructureCell `json:",omitempty"`
	CellCosts            []sim.CurrentCellCost
	CellPlaneResidue     currentCellPlaneResidues `json:",omitempty"`
	SpellCasters         []currentSpellCaster
	EffectWidths         []currentEffectWidth             `json:",omitempty"`
	Values               map[sim.EntityID]sim.ActorValues `json:",omitempty"`
	Policy               *sim.CurrentWorldPolicy          `json:",omitempty"`
	Party                []currentPartyMember             `json:",omitempty"`
	Roster               []currentPartyMember             `json:",omitempty"`
	Inventory            *currentObjectGraph              `json:",omitempty"`
	Ownership            []currentOwnedObject             `json:",omitempty"`
	NativeAreas          []currentNativeArea              `json:",omitempty"`
	NativeDeliveries     []currentNativeDelivery          `json:",omitempty"`
	Session              *currentSessionData              `json:",omitempty"`
}

func readCurrentActions(doc *sav.DocumentData) (*currentActionData, error) {
	if doc == nil {
		return nil, nil
	}
	b, present, err := sav.NativeActions(doc.State)
	if err != nil || !present {
		return nil, err
	}
	var a currentActionData
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&a); err != nil {
		return nil, fmt.Errorf("current actions: %w", err)
	}
	var tail any
	if err := d.Decode(&tail); err != io.EOF {
		return nil, fmt.Errorf("current actions contain trailing data")
	}
	if a.Version != 1 || a.NativeHistoryVersion > 1 || len(a.Bindings) > 131072 || len(a.Actions.Actors) > 32767 || len(a.Groups) > 65534 || len(a.CellCosts) > 65536 || len(a.SpellCasters) > 65534 || len(a.AbsentStructureCells) > 65536 {
		return nil, fmt.Errorf("current action version/population is invalid")
	}
	if a.Program != nil && (doc.World == nil || a.Policy == nil) {
		return nil, fmt.Errorf("current script program lacks world/register policy")
	}
	if err := validateCurrentDepartedCharacters(a.DepartedCharacters); err != nil {
		return nil, err
	}
	if err := matchCurrentNativeBasis(doc, &a); err != nil {
		return nil, err
	}
	if p := a.WorldMapReturn; p != nil && (doc.Head.Mission != 0 || p.Mission <= 0 || p.Shown < 0) {
		return nil, fmt.Errorf("current world-map return is invalid")
	}
	if _, err := matchingCurrentEffectWidths(doc, a.EffectWidths); err != nil {
		return nil, err
	}
	if len(a.EffectWidths) != 0 && a.Values == nil {
		return nil, fmt.Errorf("current effect widths lack actor values")
	}
	if _, err := matchingCurrentPlaneResidue(doc, a.CellPlaneResidue); err != nil {
		return nil, err
	}
	if (a.GroupPlayers == nil) != (a.GroupFormations == nil) || a.GroupPlayers != nil && (a.Groups == nil || *a.GroupFormations && !*a.GroupPlayers) {
		return nil, fmt.Errorf("current Group carrier presence is incomplete or conflicting")
	}
	if a.GroupParticipants != nil && a.GroupPlayers == nil && a.PlayerIdentities == nil {
		return nil, fmt.Errorf("current Participant presence lacks exact Player identities")
	}
	if err := validatePendingGameOptions(a.Options); err != nil {
		return nil, err
	}
	if err := validateCurrentPending(a.Pending); err != nil {
		return nil, err
	}
	if a.Fog != nil {
		if err := a.Fog.validate(); err != nil {
			return nil, err
		}
	}
	if len(a.PendingMessages) > 65536 {
		return nil, fmt.Errorf("current mission notice queue exceeds its bound")
	}
	if err := validateCurrentSession(a.Session); err != nil {
		return nil, err
	}
	if err := validateAbsentSessionHead(a.AbsentSessionHead, doc); err != nil {
		return nil, err
	}
	if err := validateCurrentCarrierAbsence(doc, &a); err != nil {
		return nil, err
	}
	if err := validateCurrentAbsentPlayers(doc, &a); err != nil {
		return nil, err
	}
	if err := validateCurrentArchiveCoordinates(doc, &a); err != nil {
		return nil, err
	}
	if err := validateCurrentStructureBindings(doc, &a); err != nil {
		return nil, err
	}
	if err := validateCurrentActorManifest(&a); err != nil {
		return nil, err
	}
	if err := validateActionVisuals(a.Bolts, a.Heals); err != nil {
		return nil, err
	}
	if err := validateCastRuns(a.Runs); err != nil {
		return nil, err
	}
	if err := validateCurrentAnimation(a.Animation, a.Runs); err != nil {
		return nil, err
	}
	actorBindings := make(map[sim.EntityID]bool, len(a.Bindings))
	for _, binding := range a.Bindings {
		if !binding.Structure {
			actorBindings[binding.ID] = true
		}
	}
	for _, row := range a.Animation {
		if !actorBindings[row.Entity] {
			return nil, fmt.Errorf("current animation actor lacks a binding")
		}
	}
	if err := validateCurrentDeathAges(a.DeathAges, actorBindings); err != nil {
		return nil, err
	}
	for _, raw := range a.DepartedCharacters {
		if !actorBindings[sim.EntityID(raw)] {
			return nil, fmt.Errorf("current departed character lacks an actor binding")
		}
	}
	ids, labels := map[sim.EntityID]bool{}, map[sim.EntityID]bool{}
	for _, v := range a.VisualIdentities {
		if ids[v.Entity] || labels[v.Label] {
			return nil, fmt.Errorf("repeated current visual identity")
		}
		ids[v.Entity], labels[v.Label] = true, true
	}
	return &a, nil
}

func validateCurrentDepartedCharacters(rows []uint32) error {
	if len(rows) > 131072 {
		return fmt.Errorf("current departed character population is invalid")
	}
	seen := make(map[uint32]bool, len(rows))
	for _, id := range rows {
		if seen[id] {
			return fmt.Errorf("current departed character identity is repeated")
		}
		seen[id] = true
	}
	return nil
}

func restoreCurrentDepartedCharacters(mw *mapWorld, rows []uint32, identities map[sim.EntityID]sim.EntityID) error {
	if len(rows) == 0 {
		return nil
	}
	if mw == nil || mw.mission == nil {
		return fmt.Errorf("current departed characters lack a mission")
	}
	guarded := make(map[sim.EntityID]bool, len(mw.mission.guarded))
	for _, id := range mw.mission.guarded {
		guarded[id] = true
	}
	seen := make(map[sim.EntityID]bool, len(rows))
	resolved := make([]sim.EntityID, 0, len(rows))
	for _, raw := range rows {
		old := sim.EntityID(raw)
		id, ok := identities[old]
		if !ok {
			return fmt.Errorf("current departed character %d lacks an actor binding", old)
		}
		if !guarded[id] {
			return fmt.Errorf("current departed character %d is not guarded", id)
		}
		if seen[id] {
			return fmt.Errorf("current departed character identity is repeated")
		}
		seen[id] = true
		resolved = append(resolved, id)
	}
	if mw.mission.departed == nil {
		mw.mission.departed = make(map[sim.EntityID]bool)
	}
	for _, id := range resolved {
		mw.mission.departed[id] = true
	}
	return nil
}

// currentTypedIdentityIndex resolves a key within its record class.
func currentTypedIdentityIndex(doc *sav.DocumentData, class string, key uint32) (uint16, error) {
	if key == 0 {
		return 0, nil
	}
	if doc == nil || class == "" {
		return 0, fmt.Errorf("current typed identity %#x lacks an expected SAV class", key)
	}
	var index uint16
	for i, record := range doc.Objects {
		if record.Class != class {
			continue
		}
		for _, value := range record.Values {
			if (value.Name != "Identity" && value.Name != "This") || value.Value != key {
				continue
			}
			if index != 0 && index != uint16(i+1) {
				return 0, fmt.Errorf("current SAV identity %#x is ambiguous within class %s", key, class)
			}
			index = uint16(i + 1)
		}
	}
	return index, nil
}

func sourceActorDocumentClass(class uint8) string {
	switch class {
	case 1, sim.GeneratedUnitBinding:
		return "Unit"
	case 2, sim.GeneratedHumanBinding:
		return "Human"
	case 3:
		return "Humanoid"
	default:
		return ""
	}
}

// Bind to the current archive before further graph projection. The format
// layer relocates these addresses with every reindex/retirement; final key
// completion is independent. Constructor objects may still have a zero wire key.
func projectCurrentActions(doc *sav.DocumentData, state *SnapshotSAVDocument, w *sim.World, s Snapshot, table *mapload.Table, captured ...sim.CurrentWorldPolicy) error {
	a := currentActionData{Version: 1, Actions: w.Actions(), RemovedNativeBases: w.RemovedNativeActorBases(), Options: s.Residue.PendingGameOptions,
		Bolts: s.Residue.SpellBolts, Heals: s.Residue.HealBursts, Runs: s.Residue.CastRuns, GroupTag: s.Residue.GroupTag, VisualIdentities: s.Residue.VisualIdentities, VisualNext: s.Residue.VisualNext}
	if w.Script().Dialect() == sim.ScriptROM2 {
		a.Fog = captureCurrentMissionFog(s.Residue)
	}
	a.PendingMessages = slices.Clone(s.Residue.PendingMessages)
	a.Program = captureCurrentScriptProgram(w)
	if _, err := a.Program.compile(); err != nil {
		return err
	}
	a.DepartedCharacters = append([]uint32(nil), s.Residue.DepartedCharacters...)
	if err := validateCurrentDepartedCharacters(a.DepartedCharacters); err != nil {
		return err
	}
	a.Animation = mergeCurrentMapMotion(captureCurrentAnimation(s.Residue), s.mapMotion)
	if err := validateCurrentAnimation(a.Animation, a.Runs); err != nil {
		return err
	}
	a.DeathAges = s.deathAges
	a.AbsentSessionHead = captureAbsentSessionHead(doc, w)
	policy := w.CurrentPolicy()
	if len(captured) != 0 {
		policy = captured[0]
	}
	base := sim.Terrain{}
	if s.terrainBase != nil {
		base = *s.terrainBase
	}
	policy.KeepTerrainDeviations(base, w.Bounds())
	a.Policy = &policy
	a.Values = make(map[sim.EntityID]sim.ActorValues)
	for _, e := range w.Entities() {
		v := e.Values()
		if policy.GroupCarrier || e.Decay >= sim.DecayBones {
			group := e.Group
			v.NativeGroup = &group
		}
		a.Values[e.ID] = v
	}
	for _, dead := range w.OriginalDeadActors() {
		if dead.Source.MapUnitID != 0 || dead.Current.Stage < 2 || dead.Current.Stage >= 5 ||
			dead.Source.State.Stage != dead.Current.Stage || dead.Source.State.HP == dead.Current.HP {
			continue
		}
		value := a.Values[dead.ID]
		value.DeadSourceHealth = &sim.ActorDeadSourceHealth{Wire: dead.Current.HP, Value: dead.Source.State.HP}
		a.Values[dead.ID] = value
	}
	if err := captureCurrentTerminalValues(w, &a); err != nil {
		return err
	}
	var objectErr error
	a.Inventory, a.Ownership, objectErr = captureCurrentObjects(state, w)
	if objectErr != nil {
		return objectErr
	}
	a.Actions.Reservations = sim.ActionReservations{}
	for i, id := range s.CurrentPartyIDs {
		if i < len(s.Party) {
			a.Party = append(a.Party, captureCurrentParty(id, s.Party[i]))
		}
	}
	for id, p := range s.CurrentRoster {
		a.Roster = append(a.Roster, captureCurrentParty(id, p))
	}
	slices.SortFunc(a.Roster, func(x, y currentPartyMember) int { return int(x.Entity) - int(y.Entity) })
	if planes, ok := w.SavedCellPlanes(); ok {
		_, cells, _, _ := w.SavedActorMotions()
		for _, cell := range cells {
			a.CellCosts = append(a.CellCosts, sim.CurrentCellCost{Cell: cell.Cell, Cost: planes.Cost[cell.Cell]})
		}
	}
	if _, cells, present := w.SavedStructures(); present {
		current := make(map[uint16]bool, len(cells))
		for _, cell := range cells {
			current[cell.Cell] = true
		}
		last := make(map[uint16]int, len(doc.World.Cells))
		for i, cell := range doc.World.Cells {
			last[cell.Cell] = i
		}
		for i, cell := range doc.World.Cells {
			if !current[cell.Cell] && last[cell.Cell] == i {
				if cell.Building != 0 {
					return fmt.Errorf("transported structure cell has no current binding")
				}
				a.AbsentStructureCells = append(a.AbsentStructureCells, sim.CurrentAbsentStructureCell{Cell: cell.Cell, Cost: cell.Cost, Static: cell.Static})
			}
		}
	}
	groups, _, hasGroups := w.SavedGroups()
	if err := captureCurrentPlayerIdentities(state, w, &a); err != nil {
		return err
	}
	if hasGroups && state.GroupBindings != nil {
		_, players := w.SavedGroupPlayers()
		_, formations := w.SavedPlayerFormations()
		a.GroupPlayers, a.GroupFormations = &players, &formations
		_, participants := w.PlayerParticipants()
		a.GroupParticipants = &participants
		a.Groups = make([]currentGroupContinuation, 0, len(groups))
		a.GroupHighWater = w.GroupHighWater()
		for _, g := range groups {
			found := false
			for _, b := range state.GroupBindings.Groups {
				if b.ID == g.ID {
					a.Groups = append(a.Groups, currentGroupContinuation{Object: b.PlayerObject, Inline: b.InlineIndex, ID: g.ID, Authored: g.Authored})
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("current Group lost its exact binding")
			}
		}
		for _, b := range state.GroupBindings.Groups {
			if b.RootOnly {
				if err := validateActorRootGroup(doc, b); err != nil {
					return err
				}
				r := &doc.Objects[b.PlayerObject-1].Groups[b.InlineIndex]
				selector, err := savedStructureValue(r, "G1C")
				if err != nil {
					return err
				}
				row := currentGroupContinuation{Object: b.PlayerObject, Inline: b.InlineIndex, ID: b.ID, RootOnly: true, RootSelector: selector}
				refs, _ := savedObjectRefs(r, "Actors")
				for _, object := range refs {
					found := false
					for _, actor := range state.Actors {
						if actor.ObjectIndex == object && !actor.Retired {
							row.RootMembers = append(row.RootMembers, actor.EntityID)
							found = true
							break
						}
					}
					if !found {
						return fmt.Errorf("current actor root lacks a live member binding")
					}
				}
				a.Groups = append(a.Groups, row)
			}
		}
	}
	seenIdentity := map[uint32]uint16{}
	for i, r := range doc.Objects {
		for _, v := range r.Values {
			if (v.Name != "Identity" && v.Name != "This") || v.Value == 0 {
				continue
			}
			if seenIdentity[v.Value] != 0 {
				return fmt.Errorf("current action object identity is ambiguous")
			}
			seenIdentity[v.Value] = uint16(i + 1)
		}
	}
	terminal := map[sim.EntityID]bool{}
	for _, e := range w.Entities() {
		terminal[e.ID] = !e.Alive() && e.Decay >= 2
	}
	byActor := map[sim.EntityID]uint16{}
	for _, b := range state.Actors {
		if !b.Retired {
			if b.ObjectIndex == 0 || int(b.ObjectIndex) > len(doc.Objects) {
				return fmt.Errorf("current action actor lost its object")
			}
			byActor[b.EntityID] = b.ObjectIndex
		}
	}
	terminalBindings, err := currentTerminalActorBindings(state, w)
	if err != nil {
		return err
	}
	for id, object := range terminalBindings {
		if object == 0 || int(object) > len(doc.Objects) {
			return fmt.Errorf("current terminal actor %d has no exact object", id)
		}
		if old, ok := byActor[id]; ok && old != object {
			return fmt.Errorf("current terminal actor %d conflicts with object %d", id, old)
		}
		byActor[id] = object
	}
	// An already imported held body has no ticking action producer. Retain its
	// own last residue by the exact current dead-root key, after current state.
	previous, err := readCurrentActions(state.Document)
	if err != nil {
		return err
	}
	byStructure := map[sim.EntityID]uint16{}
	structures, _, _ := w.SavedStructures()
	for _, v := range structures {
		index, err := currentTypedIdentityIndex(doc, savedStructureClass(v.Class), v.SourceKey)
		if err != nil {
			return err
		}
		byStructure[sim.EntityID(v.ID)] = index
	}
	// A world created by the engine can hold live structures before it has any
	// imported SavedStructures provenance. currentSpatial writes those
	// structures to World.Buildings in the world's stable structure order, so
	// bind action endpoints to the records this producer has just written.
	for i, v := range w.Structures() {
		if _, ok := byStructure[sim.EntityID(v.ID)]; ok || i >= len(state.Document.World.Buildings) {
			continue
		}
		byStructure[sim.EntityID(v.ID)] = state.Document.World.Buildings[i]
	}
	for _, dead := range w.OriginalDeadActors() {
		if dead.Current.Stage >= 2 {
			class := sourceActorDocumentClass(dead.Source.Class)
			if class == "" {
				return fmt.Errorf("current action dead actor %d has no SAV class binding", dead.ID)
			}
			index, err := currentTypedIdentityIndex(doc, class, dead.Source.Identity)
			if err != nil {
				return err
			}
			byActor[dead.ID] = index
		}
	}
	// An inert body's old endpoints use the old carrier's local labels. Resolve
	// every one through its bound object before assigning this save's labels.
	// Missing endpoints get distinct local labels without mutating the World.
	oldLabels := map[[2]uint32]sim.EntityID{}
	nextLabel := sim.EntityID(0)
	for id := range byActor {
		if id >= nextLabel {
			nextLabel = id + 1
		}
	}
	for id := range byStructure {
		if id >= nextLabel {
			nextLabel = id + 1
		}
	}
	oldRef := func(id sim.EntityID, structure bool) (sim.EntityID, error) {
		kind := uint32(0)
		if structure {
			kind = 1
		}
		label := [2]uint32{kind, uint32(id)}
		if v, ok := oldLabels[label]; ok {
			return v, nil
		}
		for _, binding := range previous.Bindings {
			if binding.ID != id || binding.Structure != structure {
				continue
			}
			current := byActor
			if structure {
				current = byStructure
			}
			if !binding.Missing && binding.Object > 0 && int(binding.Object) <= len(state.Document.Objects) {
				prior := &state.Document.Objects[binding.Object-1]
				key, e := savedStructureValue(prior, "Identity")
				if e != nil {
					return 0, e
				}
				currentIndex, e := currentTypedIdentityIndex(doc, prior.Class, key)
				if e != nil {
					return 0, e
				}
				for candidate, index := range current {
					if index != 0 && index == currentIndex {
						oldLabels[label] = candidate
						return candidate, nil
					}
				}
			}
			if nextLabel == 0 {
				return 0, fmt.Errorf("current action local label namespace exhausted")
			}
			v := nextLabel
			nextLabel++
			oldLabels[label] = v
			return v, nil
		}
		return 0, fmt.Errorf("held action endpoint lacks its previous binding")
	}
	for _, dead := range w.OriginalDeadActors() {
		if dead.Current.Stage < 2 {
			continue
		}
		index := byActor[dead.ID]
		if index == 0 {
			return fmt.Errorf("held action body lost its SAV root")
		}
		byActor[dead.ID] = index
		if previous == nil || terminal[dead.ID] {
			continue
		}
		for _, v := range previous.Held {
			for _, binding := range previous.Bindings {
				if binding.Structure || binding.ID != v.Entity || binding.Object == 0 || int(binding.Object) > len(state.Document.Objects) {
					continue
				}
				key, e := savedStructureValue(&state.Document.Objects[binding.Object-1], "Identity")
				if e == nil && key == dead.Source.Identity {
					copy := sim.ActionContinuations{Actors: []sim.ActorContinuation{v}}
					if err := copy.RemapActors(oldRef); err != nil {
						return err
					}
					a.Held = append(a.Held, copy.Actors[0])
				}
			}
		}
	}
	type typed struct {
		id        sim.EntityID
		structure bool
	}
	seen := map[typed]bool{}
	ref := func(id sim.EntityID, structure bool) (sim.EntityID, error) {
		k := typed{id, structure}
		if seen[k] {
			return id, nil
		}
		seen[k] = true
		index := byActor[id]
		if structure {
			index = byStructure[id]
		}
		a.Bindings = append(a.Bindings, currentActionBinding{ID: id, Structure: structure, Object: index, Missing: index == 0})
		return id, nil
	}
	if err := a.Actions.RemapActors(ref); err != nil {
		return err
	}
	if err := a.Program.remap(ref); err != nil {
		return err
	}
	for _, raw := range a.DepartedCharacters {
		if _, err := ref(sim.EntityID(raw), false); err != nil {
			return err
		}
	}
	for _, body := range w.OriginalDeadActors() {
		if _, err := ref(body.ID, false); err != nil {
			return err
		}
	}
	// Terminal actors are reached only through Values. Visiting it in ascending
	// ID keeps one state's Bindings in one order (DIV-1501).
	for _, id := range slices.Sorted(maps.Keys(a.Values)) {
		if _, err := ref(id, false); err != nil {
			return err
		}
	}
	if registry := w.SavedObjects(); registry != nil {
		var owners []sim.SavedObjectOwner
		for _, container := range registry.Containers {
			owners = append(owners, container.Owner)
		}
		for _, root := range registry.ItemRoots {
			owners = append(owners, root.Owner)
		}
		for _, root := range registry.BookRoots {
			if _, err := ref(root.Entity, false); err != nil {
				return err
			}
		}
		for _, owner := range owners {
			if owner.Kind == sim.SavedOwnerActorPack || owner.Kind == sim.SavedOwnerActorWorn {
				if _, err := ref(owner.Entity, false); err != nil {
					return err
				}
			}
		}
	}
	for _, p := range append(slices.Clone(a.Party), a.Roster...) {
		if _, err := ref(p.Entity, false); err != nil {
			return err
		}
	}
	if err := (&sim.ActionContinuations{Actors: a.Held}).RemapActors(ref); err != nil {
		return err
	}
	for _, r := range a.Runs {
		if _, err := ref(r.Entity, false); err != nil {
			return err
		}
	}
	for _, r := range a.Animation {
		if _, err := ref(r.Entity, false); err != nil {
			return err
		}
	}
	for _, r := range a.DeathAges {
		if _, err := ref(r.Entity, false); err != nil {
			return err
		}
	}
	for _, v := range a.VisualIdentities {
		if _, err := ref(v.Entity, false); err != nil {
			return err
		}
	}
	for _, v := range a.Actions.Actors {
		if byActor[v.Entity] == 0 {
			return fmt.Errorf("current action actor has no exact document binding")
		}
	}
	for _, area := range w.CellEffects() {
		if area.HasCaster {
			if _, err := ref(area.Caster, false); err != nil {
				return err
			}
		}
	}
	for _, d := range w.NativeSpellDeliverySaveStates() {
		if d.HasCaster {
			if _, err := ref(d.Caster, false); err != nil {
				return err
			}
		}
		if d.HasTarget {
			if _, err := ref(d.Target, false); err != nil {
				return err
			}
		}
	}
	if state.Objects != nil {
		objectIndices := savedObjectIndices(state.Objects)
		children := map[sim.SavedObjectID]bool{}
		for _, r := range a.Actions.Reservations.Effects {
			children[r.ID] = true
		}
		for _, r := range a.Actions.Reservations.Spells {
			children[r.ID] = true
		}
		for id := range children {
			index := objectIndices[id]
			if index == 0 {
				continue
			}
			if int(index) > len(doc.Objects) {
				return fmt.Errorf("current reserved object has no archive binding")
			}
			a.Objects = append(a.Objects, currentActionObject{ID: id, Object: index})
		}
		slices.SortFunc(a.Objects, func(x, y currentActionObject) int {
			if x.ID < y.ID {
				return -1
			}
			if x.ID > y.ID {
				return 1
			}
			return 0
		})
	}
	if err := validateActionVisuals(a.Bolts, a.Heals); err != nil {
		return err
	}
	if err := validateCastRuns(a.Runs); err != nil {
		return err
	}
	if g := w.SavedSpellGraph(); g != nil && state.WorldEffects != nil {
		for i, n := range g.Nodes {
			if !n.Retired && n.HasCaster {
				if i >= len(state.WorldEffects.SpellNodes) {
					return fmt.Errorf("current spell caster lost its node binding")
				}
				a.SpellCasters = append(a.SpellCasters, currentSpellCaster{Object: state.WorldEffects.SpellNodes[i].ObjectIndex, Caster: byActor[n.Caster]})
			}
		}
	}
	a.Manifest, err = captureCurrentActorManifest(doc, state, w, s.ActorManifest)
	if err != nil {
		return err
	}
	projectCurrentHiredNames(doc, &a, table)
	if err := bindCurrentPartyRecords(&a, doc); err != nil {
		return err
	}
	if err := stripCurrentPartyValues(&a, w, table); err != nil {
		return err
	}
	if err := bindCurrentBookPolicies(doc, state, &a); err != nil {
		return err
	}
	if err := projectCurrentPlayerSlots(doc, &a, w); err != nil {
		return err
	}
	if err := captureCurrentAbsentPlayers(doc, state, &a); err != nil {
		return err
	}
	if err := captureCurrentArchiveCoordinates(doc, state, w, &a); err != nil {
		return err
	}
	if err := captureCurrentStructureBindings(doc, w, &a); err != nil {
		return err
	}
	captureCurrentManaReserves(doc, state, w, &a)
	if err := captureCurrentDead(doc, state, w, &a); err != nil {
		return err
	}
	if err := captureCurrentTerminalMotions(w, &a); err != nil {
		return err
	}
	if err := captureCurrentEffectWidths(doc, state, w, &a); err != nil {
		return err
	}
	a.Pending, err = projectCurrentPending(s.Residue, byActor, byStructure)
	if err != nil {
		return err
	}
	if err := captureCurrentNativeBasis(doc, &a); err != nil {
		return err
	}
	b, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return sav.SetNativeActions(&doc.State, b)
}

func resolveCurrentActions(ms *Mission, a *currentActionData) (map[sim.EntityID]sim.EntityID, error) {
	state := ms.savedDocument
	actors := map[uint16]sim.EntityID{}
	for _, b := range state.Actors {
		if !b.Retired {
			actors[b.ObjectIndex] = b.EntityID
		}
	}
	for _, dead := range ms.World.OriginalDeadActors() {
		class := sourceActorDocumentClass(dead.Source.Class)
		object, err := currentTypedIdentityIndex(state.Document, class, dead.Source.Identity)
		if err != nil {
			return nil, err
		}
		if object != 0 {
			actors[object] = dead.ID
		}
	}
	terminal, err := currentTerminalActorObjects(state, ms.World)
	if err != nil {
		return nil, err
	}
	for object, id := range terminal {
		if old, ok := actors[object]; ok && old != id {
			return nil, fmt.Errorf("current terminal actor object %d conflicts with actor %d", object, old)
		}
		actors[object] = id
	}
	structures := map[uint16]sim.EntityID{}
	ss, _, _ := ms.World.SavedStructures()
	for _, v := range ss {
		object, err := currentTypedIdentityIndex(state.Document, savedStructureClass(v.Class), v.SourceKey)
		if err != nil {
			return nil, err
		}
		if object != 0 {
			structures[object] = sim.EntityID(v.ID)
		}
	}
	actorMap, structureMap := map[sim.EntityID]sim.EntityID{}, map[sim.EntityID]sim.EntityID{}
	seenObjects := map[[2]uint32]bool{}
	// Missing actors keep their saved identity; missing structures get a local
	// tombstone. Both stay reserved and absent until the normal lifetime gate;
	// LOAD neither cancels nor refunds the action.
	for _, b := range a.Bindings {
		out, objects := actorMap, actors
		kind := uint32(0)
		if b.Structure {
			out, objects, kind = structureMap, structures, 1
		}
		if _, ok := out[b.ID]; ok {
			return nil, fmt.Errorf("repeated current action identity")
		}
		if b.Missing {
			if b.Object != 0 {
				return nil, fmt.Errorf("missing current action identity has an object")
			}
			id := b.ID
			if b.Structure {
				var ok bool
				id, ok = ms.World.NextEntityID()
				if !ok {
					return nil, fmt.Errorf("current action tombstone namespace exhausted")
				}
			}
			ms.World.ReserveEntityIDs([]sim.EntityID{id})
			out[b.ID] = id
			continue
		}
		id, ok := objects[b.Object]
		k := [2]uint32{kind, uint32(b.Object)}
		if !ok || b.Object == 0 || seenObjects[k] {
			return nil, fmt.Errorf("current action object binding is absent or repeated: id %d kind %d object %d", b.ID, kind, b.Object)
		}
		seenObjects[k] = true
		out[b.ID] = id
	}
	ref := func(id sim.EntityID, structure bool) (sim.EntityID, error) {
		m := actorMap
		if structure {
			m = structureMap
		}
		n, ok := m[id]
		if !ok {
			return 0, fmt.Errorf("current action endpoint lacks a binding")
		}
		return n, nil
	}
	if err := a.Actions.RemapActors(ref); err != nil {
		return nil, err
	}
	if a.Program != nil {
		if err := a.Program.remap(ref); err != nil {
			return nil, err
		}
	}
	for i := range a.Groups {
		for j, id := range a.Groups[i].RootMembers {
			n, err := ref(id, false)
			if err != nil {
				return nil, err
			}
			a.Groups[i].RootMembers[j] = n
		}
	}
	owner := func(o *sim.SavedObjectOwner) error {
		if o.Kind == sim.SavedOwnerActorPack || o.Kind == sim.SavedOwnerActorWorn {
			var err error
			o.Entity, err = ref(o.Entity, false)
			return err
		}
		return nil
	}
	for i := range a.Ownership {
		if err := owner(&a.Ownership[i].Owner); err != nil {
			return nil, err
		}
	}
	for i := range a.NativeAreas {
		p := &a.NativeAreas[i].Policy
		if p.HasCaster {
			var err error
			p.Caster, err = ref(p.Caster, false)
			if err != nil {
				return nil, err
			}
		}
	}
	for i := range a.NativeDeliveries {
		p := &a.NativeDeliveries[i].Policy
		if p.HasCaster {
			var err error
			p.Caster, err = ref(p.Caster, false)
			if err != nil {
				return nil, err
			}
		}
		if p.MissingTarget != nil {
			id, err := ref(*p.MissingTarget, false)
			if err != nil {
				return nil, err
			}
			p.MissingTarget = &id
		}
	}
	if a.Inventory != nil {
		for i := range a.Inventory.Containers {
			if err := owner(&a.Inventory.Containers[i].Owner); err != nil {
				return nil, err
			}
		}
		for i, entity := range a.Inventory.BookActors {
			var err error
			a.Inventory.BookActors[i], err = ref(entity, false)
			if err != nil {
				return nil, err
			}
		}
	}
	if a.Values != nil {
		next := make(map[sim.EntityID]sim.ActorValues, len(a.Values))
		live := make(map[sim.EntityID]int32)
		for _, entity := range ms.World.Entities() {
			live[entity.ID] = entity.HP
		}
		for id, v := range a.Values {
			n, err := ref(id, false)
			if err != nil {
				return nil, err
			}
			for _, b := range a.Bindings {
				hp, present := live[n]
				if b.ID == id && !b.Structure && !b.Missing && present {
					record := &state.Document.Objects[b.Object-1]
					stage, err := savedStructureValue(record, "Stage")
					if err != nil {
						return nil, err
					}
					dwell, err := savedStructureValue(record, "U6C")
					if err != nil {
						return nil, err
					}
					v.Decay = sim.DecayStage(stage)
					if hp > 0 && stage == 1 {
						// Current health supersedes stale Stage 1 after healing.
						v.Decay, v.Dwell = sim.DecayNone, 0
					} else if stage != 0 {
						v.Dwell = uint16(int8(dwell))
					} else {
						v.Dwell = 0
					}
					break
				}
			}
			next[n] = v
		}
		a.Values = next
	}
	if a.Manifest != nil {
		for i := range a.Manifest.Actors {
			var err error
			a.Manifest.Actors[i].Entity, err = ref(a.Manifest.Actors[i].Entity, false)
			if err != nil {
				return nil, err
			}
		}
	}
	for _, rows := range [][]currentPartyMember{a.Party, a.Roster} {
		for i := range rows {
			id, err := ref(rows[i].Entity, false)
			if err != nil {
				return nil, err
			}
			rows[i].Entity = id
		}
	}
	if err := (&sim.ActionContinuations{Actors: a.Held}).RemapActors(ref); err != nil {
		return nil, err
	}
	for i := range a.RemovedNativeBases {
		id, err := ref(a.RemovedNativeBases[i].ID, false)
		if err != nil {
			return nil, err
		}
		a.RemovedNativeBases[i].ID = id
	}
	slices.SortFunc(a.RemovedNativeBases, func(x, y sim.NativeActorBasisRecord) int {
		if x.ID < y.ID {
			return -1
		}
		if x.ID > y.ID {
			return 1
		}
		return 0
	})
	return actorMap, nil
}

func restoreOriginalActions(ms *Mission, table *mapload.Table) error {
	if ms.savedDocument == nil {
		return nil
	}
	a, err := readCurrentActions(ms.savedDocument.Document)
	if err != nil {
		return err
	}
	if a == nil {
		return reconcileSavedTerminalRegistry(ms)
	}
	original := ms
	staged := *ms
	raw, err := ms.World.MarshalBinary()
	if err != nil {
		return err
	}
	world := *ms.World
	staged.World = &world
	if err := staged.World.UnmarshalBinary(raw); err != nil {
		return err
	}
	staged.savedDocument, err = cloneSavedDocument(ms.savedDocument)
	if err != nil {
		return err
	}
	ms = &staged
	if a.Program != nil {
		// Local map-only endpoints must not enter the saved identity namespace.
		if err := restoreCurrentScriptProgram(ms, table, &currentScriptProgram{Dialect: a.Program.Dialect}); err != nil {
			return err
		}
	}
	absentPlayers, err := matchCurrentAbsentPlayers(ms.savedDocument.Document, a)
	if err != nil {
		return err
	}
	if err := restoreCurrentActorIdentities(ms, a); err != nil {
		return err
	}
	archiveCoordinates, err := prepareCurrentArchiveCoordinates(ms, a)
	if err != nil {
		return err
	}
	if err := restoreCurrentPlayerSlots(ms, a.PlayerSlots); err != nil {
		return err
	}
	if err := restoreCurrentPlayerIdentities(ms, a); err != nil {
		return err
	}
	if err := bindCurrentPartyRecords(a, ms.savedDocument.Document); err != nil {
		return err
	}
	actorIDs, err := resolveCurrentActions(ms, a)
	if err != nil {
		return err
	}
	terminalMotions, err := readCurrentTerminalMotions(ms, a, actorIDs)
	if err != nil {
		return err
	}
	if a.Policy != nil {
		if err := ms.World.RestoreCurrentClock(a.Policy.TickHigh, a.Policy.ClockKnown); err != nil {
			return err
		}
		if err := ms.World.RestoreCurrentSpellDurations(a.Policy.SpellPolicies); err != nil {
			return err
		}
	}
	if err := restoreCurrentAreas(ms, a.NativeAreas); err != nil {
		return err
	}
	if err := restoreCurrentSpellCasters(ms, a.SpellCasters); err != nil {
		return err
	}
	if err := restoreCurrentSpellDeliveries(ms, a.NativeDeliveries); err != nil {
		return err
	}
	// Remove transport-only containment before restoring native dispatch order.
	// A native registry with absent Players may order Groups across wire roots.
	if a.GroupPlayers != nil && a.GroupFormations != nil {
		if err := ms.World.RestoreGroupCarrierPresence(*a.GroupPlayers, *a.GroupFormations); err != nil {
			return err
		}
	}
	if a.GroupParticipants != nil && !*a.GroupParticipants {
		if err := ms.World.RestorePlayerParticipants(nil, false); err != nil {
			return err
		}
	}
	if a.PlayerIdentities == nil && (a.GroupParticipants == nil || !*a.GroupParticipants) {
		ms.World.RestoreCurrentPlayerRegistryAbsent()
	}
	if a.Groups != nil {
		bindings := ms.savedDocument.GroupBindings
		if bindings == nil || len(a.Groups) != len(bindings.Groups) {
			return fmt.Errorf("incomplete current Group bindings")
		}
		var rows []sim.GroupContinuation
		next := make([]SnapshotSAVGroupBinding, 0, len(a.Groups))
		current, _, _ := ms.World.SavedGroups()
		highWater := a.GroupHighWater
		nextID := highWater
		for _, row := range a.Groups {
			nextID = max(nextID, row.ID)
		}
		for _, row := range a.Groups {
			found := false
			for _, b := range bindings.Groups {
				if b.PlayerObject == row.Object && b.InlineIndex == row.Inline {
					if row.RootOnly {
						neutral, err := actorRootGroupNeutral(ms.savedDocument.Document, b)
						if err != nil {
							return err
						}
						var members []sim.EntityID
						var selector uint32
						for _, g := range current {
							if g.ID == b.ID {
								selector = g.Selector
								for _, m := range g.Members {
									if !m.Bound {
										neutral = false
									}
									members = append(members, m.Entity)
								}
							}
						}
						if !neutral || selector != row.RootSelector || !slices.Equal(members, row.RootMembers) {
							if nextID == ^uint32(0) {
								return fmt.Errorf("current ordinary Group exhausted native identity space")
							}
							// An ordinary edit makes this an actual current Group.
							// The absence leaf never replaces its AI or membership.
							nextID++
							highWater = nextID
							row.ID, row.RootOnly, row.Authored = nextID, false, false
							for _, id := range members {
								if v, ok := a.Values[id]; ok {
									v.NativeGroup = nil
									a.Values[id] = v
								}
							}
						}
					}
					rows = append(rows, sim.GroupContinuation{Group: b.ID, ID: row.ID, Authored: row.Authored, RootOnly: row.RootOnly})
					b.ID, b.Authored, b.RootOnly = row.ID, row.Authored, row.RootOnly
					if b.RootOnly {
						if err := validateActorRootGroup(ms.savedDocument.Document, b); err != nil {
							return err
						}
					}
					next = append(next, b)
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("current Group archive binding is absent")
			}
		}
		absent := absentCurrentPlayerIDs(bindings, next, absentPlayers)
		if err := ms.World.RestoreGroupContinuations(rows, highWater, absent...); err != nil {
			return err
		}
		slices.SortFunc(next, func(x, y SnapshotSAVGroupBinding) int { return int(x.ID) - int(y.ID) })
		bindings.Groups = next
	}
	if a.GroupPlayers != nil && a.GroupFormations != nil {
		bindings := ms.savedDocument.GroupBindings
		bindings.PlayersConstructed, bindings.FormationsPresent = !*a.GroupPlayers, *a.GroupFormations
		if !*a.GroupPlayers {
			for i := range bindings.Groups {
				if !bindings.Groups[i].RootOnly {
					bindings.Groups[i].ContainerID = 0
				}
			}
		}
	}
	if err := ms.World.RestoreAbsentStructureCells(a.AbsentStructureCells); err != nil {
		return err
	}
	live := map[sim.EntityID]bool{}
	for _, e := range ms.World.Entities() {
		live[e.ID] = true
	}
	for _, v := range a.Held {
		if live[v.Entity] {
			v.ImportedMotion = false
			a.Actions.Actors = append(a.Actions.Actors, v)
		}
	}
	objects := map[sim.SavedObjectID]sim.SavedObjectID{}
	if a.Inventory != nil {
		objects, err = restoreCurrentObjects(ms, a.Inventory, a.Ownership, table)
		if err != nil {
			return err
		}
	} else if ms.savedDocument.Objects != nil {
		indices := savedObjectIndices(ms.savedDocument.Objects)
		for _, b := range a.Objects {
			if b.ID == 0 || objects[b.ID] != 0 || b.Object == 0 {
				return fmt.Errorf("invalid shared reservation identity")
			}
			for id, index := range indices {
				if index == b.Object {
					objects[b.ID] = id
				}
			}
			if objects[b.ID] == 0 {
				return fmt.Errorf("shared reservation object was not imported")
			}
		}
	}
	if a.Policy != nil {
		regs := ms.savedDocument.Document.World.Session.Results
		a.Policy.CurrentRegisters = &regs
	}
	if err := matchCurrentStationaryPositions(ms, a); err != nil {
		return err
	}
	matchCurrentManaReserves(ms, a)
	if err := matchCurrentBookSelection(ms.savedDocument.Document, a); err != nil {
		return err
	}
	if err := matchCurrentOrderSpellAbsence(ms.savedDocument.Document, a); err != nil {
		return err
	}
	if err := matchCurrentEffectWidths(ms, a); err != nil {
		return err
	}
	if err := ms.World.RestoreCurrentContinuation(a.Policy, a.Values, a.Actions, objects, terminalMotions...); err != nil {
		return err
	}
	if err := ms.World.RestoreNativeActorBases(a.RemovedNativeBases); err != nil {
		return err
	}
	if err := restoreLegacyNativeObservations(ms, a); err != nil {
		return err
	}
	if err := restoreCurrentArchiveCoordinates(ms, a, archiveCoordinates); err != nil {
		return err
	}
	if err := restoreAbsentSessionHead(ms.World, ms.savedDocument.Document, a.AbsentSessionHead); err != nil {
		return err
	}
	if err := restoreCurrentCarrierAbsence(ms.World, ms.savedDocument.Document, a); err != nil {
		return err
	}
	if a.Policy != nil && ms.savedDocument.WorldEffects != nil {
		meta := ms.savedDocument.WorldEffects
		if a.Policy.EffectDriverCarrier != nil && !*a.Policy.EffectDriverCarrier {
			meta.Areas, meta.ProjectileIDs, meta.Projectiles = nil, nil, false
		}
		if a.Policy.SpellGraphCarrier != nil && !*a.Policy.SpellGraphCarrier {
			meta.SpellNodes = nil
		}
	}
	if a.Policy != nil && !a.Policy.GroupCarrier && ms.savedDocument.GroupBindings != nil {
		ms.savedDocument.GroupBindings.FormationsPresent = false
	}
	repairCurrentNativePacks(ms, a, actorIDs)
	if err := restoreCurrentPartyMembers(ms, a, table); err != nil {
		return err
	}
	if a.Program == nil {
		if err := restoreCurrentScriptProgram(ms, table, nil); err != nil {
			return err
		}
	}
	if err := restoreCurrentActorManifest(ms, a, table); err != nil {
		return err
	}
	if a.Program != nil {
		if err := restoreCurrentScriptProgram(ms, table, a.Program); err != nil {
			return err
		}
		// Best effort: a failure keeps the saved program and never refuses the load.
		_ = restoreCurrentScriptBindings(ms, table)
	}
	planeResidue, err := matchingCurrentPlaneResidue(ms.savedDocument.Document, a.CellPlaneResidue)
	if err != nil {
		return err
	}
	if err := ms.World.RestoreCurrentCellCosts(a.CellCosts, planeResidue...); err != nil {
		return err
	}
	// Extend detached bindings for reserved objects created by the same import.
	// They have no ordinary archive root and remain exclusively session-owned.
	r := ms.World.SavedObjects()
	if r != nil && ms.savedDocument.Objects != nil {
		appendMissing := func(rows *[]SnapshotSAVObjectBinding, id sim.SavedObjectID) {
			for _, b := range *rows {
				if b.ID == id {
					return
				}
			}
			*rows = append(*rows, SnapshotSAVObjectBinding{ID: id, Unavailable: savedActiveScrollUnavailable})
		}
		for _, v := range r.Items {
			appendMissing(&ms.savedDocument.Objects.Items, v.ID)
		}
		for _, v := range r.Effects {
			appendMissing(&ms.savedDocument.Objects.Effects, v.ID)
		}
		for _, v := range r.Spells {
			appendMissing(&ms.savedDocument.Objects.Spells, v.ID)
		}
	}
	if err := reconcileSavedTerminalRegistry(ms); err != nil {
		return err
	}
	if err := bindCurrentPlayerRoots(ms.savedDocument, ms.World); err != nil {
		return err
	}
	*original.World = *ms.World
	ms.World = original.World
	*original = *ms
	return nil
}

func (mw *mapWorld) restoreOriginalActionView() error {
	ms := mw.mission.state
	if ms.savedDocument == nil {
		return nil
	}
	a, err := readCurrentActions(ms.savedDocument.Document)
	if err != nil || a == nil {
		return err
	}
	// View-only identity resolution must not reserve missing targets again.
	byObject := map[uint16]sim.EntityID{}
	for _, b := range ms.savedDocument.Actors {
		if !b.Retired {
			byObject[b.ObjectIndex] = b.EntityID
		}
	}
	for _, dead := range ms.World.OriginalDeadActors() {
		for i, r := range ms.savedDocument.Document.Objects {
			key, err := savedStructureValue(&r, "Identity")
			if err == nil && key == dead.Source.Identity {
				byObject[uint16(i+1)] = dead.ID
			}
		}
	}
	terminal, err := currentTerminalActorObjects(ms.savedDocument, ms.World)
	if err != nil {
		return err
	}
	for object, id := range terminal {
		if old, ok := byObject[object]; ok && old != id {
			return fmt.Errorf("current terminal actor object %d conflicts with actor %d", object, old)
		}
		byObject[object] = id
	}
	ids := map[sim.EntityID]sim.EntityID{}
	for _, b := range a.Bindings {
		if !b.Structure && b.Missing {
			ids[b.ID] = b.ID
		}
		if !b.Structure && !b.Missing {
			id, ok := byObject[b.Object]
			if ok {
				ids[b.ID] = id
			}
		}
	}
	if err := restoreCurrentDepartedCharacters(mw, a.DepartedCharacters, ids); err != nil {
		return err
	}
	for i := range a.Runs {
		id, ok := ids[a.Runs[i].Entity]
		if !ok {
			return fmt.Errorf("current cast animation actor is absent")
		}
		a.Runs[i].Entity = id
	}
	var visualIDs []SnapshotVisualIdentity
	for _, v := range a.VisualIdentities {
		if id, ok := ids[v.Entity]; ok {
			v.Entity = id
			visualIDs = append(visualIDs, v)
		}
	}
	if a.Pending != nil {
		if err := mw.restoreCurrentPending(a.Pending, byObject); err != nil {
			return err
		}
	} else {
		for _, o := range a.Options {
			mw.applyGameOption(o.Option, o.Value)
		}
	}
	mw.restoreActionVisuals(a.Bolts, a.Heals)
	mw.restoreCastRuns(a.Runs)
	if err := mw.restoreCurrentAnimation(a.Animation, ids); err != nil {
		return err
	}
	if err := mw.restoreCurrentDeathAges(a.DeathAges, ids); err != nil {
		return err
	}
	mw.restoreVisualIdentities(visualIDs, a.VisualNext)
	mw.groupTag = a.GroupTag
	if a.Fog != nil && !mw.restoreNativeFog(a.Fog.residue()) {
		return fmt.Errorf("current visibility belongs to a different mission extent")
	}
	if mw.mission != nil {
		mw.mission.pendingMessages = slices.Clone(a.PendingMessages)
	}
	if len(a.DeathAges) != 0 {
		mw.view.SetEntities(mw.entityDraws())
	}
	return nil
}
