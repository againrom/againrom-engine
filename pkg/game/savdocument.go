package game

import (
	"fmt"
	"slices"
	"strings"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

const snapshotSAVDocumentVersion uint32 = 1

func decodeSavedDocument(raw []byte) (*SnapshotSAVDocument, []sav.DocumentObjectOrigin) {
	doc, origins, err := sav.DecodeDocumentDataWithOrigins(raw)
	if err != nil {
		// Partial legacy inputs remain playable through the established import
		// path, but can never masquerade as a complete exportable document.
		reason := fmt.Sprintf("complete SAV document unavailable: %v", err)
		if len(reason) > 4096 {
			reason = reason[:4096]
		}
		return &SnapshotSAVDocument{Version: snapshotSAVDocumentVersion, Unavailable: reason}, nil
	}
	return &SnapshotSAVDocument{Version: snapshotSAVDocumentVersion, Document: &doc}, origins
}

// SnapshotSAVDocument is the complete semantic SAV graph, not a source file.
// Live values are projected from World at Snapshot; unknown/source-only state
// remains explicit. Its presence alone does not admit original-format export.
// Unavailable preserves an existing partial-import boundary when the input
// lacks the complete container grammar; it never hides an invalid native DTO.
type SnapshotSAVDocument struct {
	Version       uint32
	Document      *sav.DocumentData
	Actors        []SnapshotSAVActor
	Unavailable   string
	GroupBindings *SnapshotSAVGroupBindings
	PlayerRoots   *[]SnapshotSAVGroupPlayerBinding `json:",omitempty"`
	PlayerPurses  *SnapshotSAVPlayerPurses
	Objects       *SnapshotSAVObjectBindings
	ActorEffects  *SnapshotSAVActorEffects
	WorldEffects  *SnapshotSAVWorldEffects
	// ActionTick is the World tick at which the actor orders were restored
	// from the loaded Document. While the World is still at that tick, an
	// order field the World holds no value for keeps the loaded bytes.
	ActionTick    uint64
	ActionTickSet bool
}

// ObjectIndex is the DTO's local index, NOT SourceBinding.ArchiveIndex. A
// retired native actor keeps its explicit origin until the graph producer
// retires all of its edges; it must never resurrect the imported actor.
type SnapshotSAVActor struct {
	EntityID    sim.EntityID
	ObjectIndex uint16
	Retired     bool
}

func cloneSavedDocument(src *SnapshotSAVDocument, allowCurrentSackDrift ...bool) (*SnapshotSAVDocument, error) {
	if src == nil {
		return nil, nil
	}
	if src.Version != snapshotSAVDocumentVersion {
		return nil, fmt.Errorf("saved SAV document version %d is unsupported", src.Version)
	}
	if src.Document == nil {
		if src.Unavailable == "" || len(src.Unavailable) > 4096 || strings.ContainsRune(src.Unavailable, '\x00') || len(src.Actors) != 0 || src.GroupBindings != nil || src.PlayerRoots != nil || src.PlayerPurses != nil || src.Objects != nil || src.ActorEffects != nil || src.WorldEffects != nil {
			return nil, fmt.Errorf("saved SAV document has invalid unavailable state")
		}
		return &SnapshotSAVDocument{Version: src.Version, Unavailable: src.Unavailable}, nil
	}
	if src.Unavailable != "" || len(src.Actors) > len(src.Document.Objects) {
		return nil, fmt.Errorf("saved SAV document has conflicting state or too many actor bindings")
	}
	doc, permutation, err := sav.ReindexDocumentData(*src.Document)
	if err != nil {
		return nil, fmt.Errorf("saved SAV document: %w", err)
	}
	if doc.World == nil || doc.Head.Mission == 0 {
		return nil, fmt.Errorf("saved mission SAV document has no world")
	}
	actors := slices.Clone(src.Actors)
	seenObjects := make(map[uint16]bool, len(actors))
	for i, actor := range actors {
		if i > 0 && actors[i-1].EntityID >= actor.EntityID || actor.ObjectIndex == 0 || int(actor.ObjectIndex) > len(doc.Objects) || seenObjects[actor.ObjectIndex] {
			return nil, fmt.Errorf("saved SAV document has ambiguous actor bindings")
		}
		seenObjects[actor.ObjectIndex] = true
		switch src.Document.Objects[actor.ObjectIndex-1].Class {
		case "Unit", "Human", "Humanoid":
		default:
			return nil, fmt.Errorf("saved SAV actor %d binds a non-actor object", actor.EntityID)
		}
	}
	groups, err := cloneSavedGroupBindings(src.GroupBindings, src.Document, actors)
	if err != nil {
		return nil, err
	}
	playerRoots, err := cloneCurrentPlayerRoots(src.PlayerRoots, src.Document)
	if err != nil {
		return nil, err
	}
	purses, err := cloneSavedPlayerPurses(src.PlayerPurses, src.Document, groups)
	if err != nil {
		return nil, err
	}
	objects, err := cloneSavedObjectBindings(src.Objects, src.Document, allowCurrentSackDrift...)
	if err != nil {
		return nil, err
	}
	effects, err := cloneSavedActorEffects(src.ActorEffects, src.Document, actors)
	if err != nil {
		return nil, err
	}
	worldEffects, err := cloneSavedWorldEffects(src.WorldEffects, src.Document)
	if err != nil {
		return nil, err
	}
	out := &SnapshotSAVDocument{Version: src.Version, Document: &doc, Actors: actors, GroupBindings: groups, PlayerRoots: playerRoots, PlayerPurses: purses, Objects: objects, ActorEffects: effects, WorldEffects: worldEffects, ActionTick: src.ActionTick, ActionTickSet: src.ActionTickSet}
	if err := remapSavedSackDocument(out, permutation); err != nil {
		return nil, err
	}
	return out, nil
}

func savedDocumentFromSnapshot(s Snapshot) (*SnapshotSAVDocument, error) {
	if s.SavedDocument == nil {
		// Exact native formations are only minted with document bindings.
		// Legacy absence remains valid; a retained current carrier cannot lose
		// its independent binding authority at the envelope seam.
		if len(s.World) != 0 && s.World[0] >= firstSavedFormationWorldForm {
			var world sim.World
			if err := sim.CheckSaveForm(s.World); err != nil {
				return nil, err
			}
			if err := world.UnmarshalBinary(s.World); err != nil {
				return nil, err
			}
			if err := validateSavedWorldEffectsWorld(nil, &world); err != nil {
				return nil, err
			}
			if err := savedFormationWorld(nil, &world, false); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
	if s.Mission <= 0 || len(s.World) == 0 || s.OriginalCity != nil {
		return nil, fmt.Errorf("saved SAV document requires one mission world")
	}
	out, err := cloneSavedDocument(s.SavedDocument)
	if err != nil {
		return nil, err
	}
	if out.Document != nil && (uint64(s.Mission) != uint64(out.Document.Head.Mission) || int64(s.Difficulty) != int64(out.Document.Head.Difficulty)) {
		return nil, fmt.Errorf("saved SAV document names a different mission or difficulty")
	}
	if out.PlayerPurses != nil || out.Objects != nil || out.GroupBindings != nil || out.PlayerRoots != nil || len(s.World) != 0 && s.World[0] >= firstSavedFormationWorldForm {
		validate := func(world *sim.World) error {
			if err := savedFormationWorld(out, world, false); err != nil {
				return err
			}
			if err := validateSavedGroupBindingWorld(out, world); err != nil {
				return err
			}
			if err := validateCurrentPlayerRoots(out, world); err != nil {
				return err
			}
			if err := savedPlayerPurseWorld(out, world, true); err != nil {
				return err
			}
			if err := savedAutoHealingWorld(out, world, false); err != nil {
				return err
			}
			if err := validateSavedObjectBindingWorld(out, world); err != nil {
				return err
			}
			if err := validateSavedWorldEffectsWorld(out, world); err != nil {
				return err
			}
			return validateSavedActorEffectsWorld(out, world)
		}
		if err := sim.CheckSaveForm(s.World); err != nil {
			return nil, err
		}
		var world sim.World
		if err := world.UnmarshalBinary(s.World); err != nil {
			return nil, err
		}
		if err := validate(&world); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func savedActorClass(class uint8) string {
	switch class {
	case 1, sim.GeneratedUnitBinding:
		return "Unit"
	case 2, sim.GeneratedHumanBinding:
		return "Human"
	case 3:
		return "Humanoid"
	}
	return ""
}

func validateSavedDocumentWorld(state *SnapshotSAVDocument, world *sim.World) error {
	if state == nil || state.Document == nil {
		return nil
	}
	if world == nil {
		return fmt.Errorf("saved SAV document has no native world")
	}
	if err := validateCurrentPlayerRoots(state, world); err != nil {
		return err
	}
	clock, present := world.SessionClock()
	if present && (clock.SubTick != state.Document.Head.CounterA || clock.FullTick != state.Document.Head.CounterB) || !present && uint32(world.Tick()) != state.Document.Head.CounterA {
		return fmt.Errorf("saved SAV document clock differs from its native world")
	}
	entities := make(map[sim.EntityID]sim.Entity)
	for _, e := range world.Entities() {
		entities[e.ID] = e
	}
	for _, binding := range state.Actors {
		e, exists := entities[binding.EntityID]
		if binding.Retired {
			if exists {
				return fmt.Errorf("saved SAV retired actor %d still exists", binding.EntityID)
			}
			continue
		}
		if !exists || !currentActorRecordMatches(world, e, state.Document.Objects[binding.ObjectIndex-1]) {
			return fmt.Errorf("saved SAV actor %d has no matching native actor", binding.EntityID)
		}
		if err := e.SourceBinding.Validate(e); err != nil {
			return err
		}
		if e.SourceBinding.Generated() {
			r := &state.Document.Objects[binding.ObjectIndex-1]
			for _, field := range []sav.DocumentValueData{
				{Name: "Identity", Value: e.SourceBinding.Identity},
				{Name: "RuntimeID", Value: e.SourceBinding.RuntimeID},
				{Name: "T0C", Value: uint32(e.SourceBinding.TokenRow)},
				{Name: "T0E", Value: uint32(e.ActorLoad.Source.TypeID)},
				{Name: "U4C", Value: uint32(e.SourceBinding.ClassFlags)},
			} {
				v, err := savedStructureValue(r, field.Name)
				if field.Name == "U4C" {
					// Bit 3 is live on-map presence, not constructor authority.
					v &^= sav.ActorOffMapFlag
				}
				if err != nil || v != field.Value {
					return fmt.Errorf("generated SAV actor %d %s differs from constructor authority", e.ID, field.Name)
				}
			}
		}
		delete(entities, binding.EntityID)
	}
	// Current construction completes missing bindings, including later spawns
	// which already have a constructor basis. This validates retained bindings;
	// projectCurrentActions checks complete coverage on the constructed graph.
	return nil
}

func restoreSavedDocument(ms *Mission, source *SnapshotSAVDocument) error {
	return restoreSavedDocumentDecoded(ms, source, false)
}

func restoreSavedDocumentDecoded(ms *Mission, source *SnapshotSAVDocument, decodedLegacy bool) error {
	state, err := cloneSavedDocument(source)
	if err != nil {
		return err
	}
	if err := validateSavedDocumentWorld(state, ms.World); err != nil {
		return err
	}
	if err := validateSavedGroupBindingWorld(state, ms.World); err != nil {
		return err
	}
	if err := savedFormationWorld(state, ms.World, false); err != nil {
		return err
	}
	if err := validateSavedActorMotionsWorld(state, ms.World); err != nil {
		return err
	}
	if err := savedPlayerPurseWorld(state, ms.World, true); err != nil {
		return err
	}
	if err := savedAutoHealingWorld(state, ms.World, false); err != nil {
		return err
	}
	if err := validateSavedObjectBindingWorldDecoded(state, ms.World, decodedLegacy); err != nil {
		return err
	}
	if err := validateSavedActorEffectsWorld(state, ms.World); err != nil {
		return err
	}
	if err := validateSavedWorldEffectsWorld(state, ms.World); err != nil {
		return err
	}
	ms.savedDocument = state
	ms.restoreDeadArt()
	return nil
}

func (ms *Mission) restoreDeadArt() {
	if ms.savedDocument == nil || ms.savedDocument.Document == nil {
		return
	}
	for _, dead := range ms.World.OriginalDeadActors() {
		if dead.Source.MapUnitID != 0 {
			continue
		}
		for i := range ms.savedDocument.Document.Objects {
			record := &ms.savedDocument.Document.Objects[i]
			if record.Class != savedActorClass(dead.Source.Class) {
				continue
			}
			key, err := savedStructureValue(record, "Identity")
			if err != nil || key != dead.Source.Identity {
				continue
			}
			typeID, err := savedStructureValue(record, "T0E")
			if err == nil {
				if ms.DeadArt == nil {
					ms.DeadArt = map[sim.EntityID]uint16{}
				}
				ms.DeadArt[dead.ID] = uint16(typeID)
			}
		}
	}
}

// fresh forwards to projectSavedActorValues below, under the same optional,
// default-false idiom: see that function's own doc comment for what it
// gates and why only currentWorldDocument's own recompute call ever sets it.
func snapshotSavedDocument(ms *Mission, fresh ...bool) (*SnapshotSAVDocument, error) {
	state, err := cloneSavedDocument(ms.savedDocument, true)
	if err != nil || state == nil || state.Document == nil {
		return state, err
	}
	clock, present := ms.World.SessionClock()
	if !present {
		clock.SubTick = uint32(ms.World.Tick())
		clock.FullTick = clock.SubTick / 16
		if clock.SubTick%16 == 15 {
			clock.FullTick++
		}
	}
	state.Document.Head.CounterA, state.Document.Head.CounterB = clock.SubTick, clock.FullTick
	state.Document.World.Session.Raw08 = savedSessionHead(ms.World.RawSessionHead(), state.Document.World.Session.Raw08)
	state.Document.World.Session.RawA828 = ms.World.RawSessionMid()
	state.Document.World.Session.Won, state.Document.World.Session.Lost = ms.World.ScriptCounters()
	for i := range state.Document.World.Session.Latches {
		state.Document.World.Session.Latches[i] = 0
		if ms.World.ScriptLatched(int32(i)) {
			state.Document.World.Session.Latches[i] = 1
		}
	}
	relations := ms.World.Relations()
	for from := range state.Document.World.Session.Diplomacy {
		for to := range state.Document.World.Session.Diplomacy[from] {
			state.Document.World.Session.Diplomacy[from][to] = relations.Byte(uint32(from), uint32(to))
		}
	}
	live := make(map[sim.EntityID]bool)
	for _, e := range ms.World.Entities() {
		live[e.ID] = true
	}
	for i := range state.Actors {
		if !live[state.Actors[i].EntityID] {
			state.Actors[i].Retired = true
		}
	}
	// Establish dead roots here too, before projectSavedGroups below rebuilds
	// live-only Group membership: a retired actor's object may have had no
	// other root, and rootDeadActors' own add() is dedup-checked so a later
	// materializeCurrentWorld pass repeating this call is a harmless no-op.
	if err := rootDeadActors(state, ms.World); err != nil {
		return nil, err
	}
	if err := validateSavedDocumentWorld(state, ms.World); err != nil {
		return nil, err
	}
	if err := projectSavedActorValues(state.Document, state.Actors, ms.World, fresh...); err != nil {
		return nil, err
	}
	if err := projectSavedActorMotions(state, ms.World); err != nil {
		return nil, err
	}
	if err := projectSavedStructures(state.Document, ms.World); err != nil {
		return nil, err
	}
	if err := projectSavedGroups(state, ms.World); err != nil {
		return nil, err
	}
	if err := savedFormationWorld(state, ms.World, true); err != nil {
		return nil, err
	}
	if err := projectSavedPlayerParticipants(state, ms.World); err != nil {
		return nil, err
	}
	if err := projectSavedPlayerPurses(state, ms.World); err != nil {
		return nil, err
	}
	if err := savedAutoHealingWorld(state, ms.World, true); err != nil {
		return nil, err
	}
	if err := projectSavedSackObjects(state, ms.World, ms.Start.ConstructionTable); err != nil {
		return nil, err
	}
	if err := projectSavedActorEffects(state, ms.World); err != nil {
		return nil, err
	}
	if err := projectSavedWorldEffects(state, ms.World); err != nil {
		return nil, err
	}
	doc, permutation, err := sav.ReindexDocumentData(*state.Document)
	if err != nil {
		return nil, err
	}
	state.Document = &doc
	if err := remapSavedSackDocument(state, permutation); err != nil {
		return nil, err
	}
	return state, nil
}

func importSavedDocument(ms *Mission, state *SnapshotSAVDocument, origins []sav.DocumentObjectOrigin) error {
	if state == nil || state.Document == nil {
		return restoreSavedDocument(ms, state)
	}
	byArchive := make(map[uint16]uint16, len(origins))
	for _, origin := range origins {
		if origin.ArchiveIndex == 0 || origin.ObjectIndex == 0 || byArchive[origin.ArchiveIndex] != 0 {
			return fmt.Errorf("SAV document has an ambiguous import origin")
		}
		byArchive[origin.ArchiveIndex] = origin.ObjectIndex
	}
	bound := *state
	for _, e := range ms.World.Entities() {
		if e.SourceBinding.Class != 0 {
			object := byArchive[e.SourceBinding.ArchiveIndex]
			if object == 0 {
				return fmt.Errorf("SAV document lacks exact import origin for actor %d", e.ID)
			}
			bound.Actors = append(bound.Actors, SnapshotSAVActor{EntityID: e.ID, ObjectIndex: object})
		}
	}
	slices.SortFunc(bound.Actors, func(a, b SnapshotSAVActor) int {
		if a.EntityID < b.EntityID {
			return -1
		}
		if a.EntityID > b.EntityID {
			return 1
		}
		return 0
	})
	var err error
	bound.GroupBindings, err = importSavedGroupBindings(state.Document, ms.World, byArchive)
	if err != nil {
		return err
	}
	if err := importSavedPlayerParticipants(&bound, ms.World); err != nil {
		return err
	}
	if err := importSavedActorMotions(ms, &bound); err != nil {
		return err
	}
	if err := importSavedDiaries(ms, &bound); err != nil {
		return err
	}
	return importSavedPlayerPurses(ms, &bound)
}
