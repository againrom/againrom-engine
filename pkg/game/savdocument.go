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
	state, origins, _ := decodeSavedRecord(raw)
	return state, origins
}

// heldDocument is a document that decoded whole but that no claim admits as
// complete: the profile places no formation block. The LOAD binds only the
// world effects through it; the loaded state keeps their family alone.
type heldDocument struct {
	doc     *sav.DocumentData
	origins []sav.DocumentObjectOrigin
}

// decodeSavedRecord is decodeSavedDocument that also returns the document a
// partial state could not admit, when it decoded.
func decodeSavedRecord(raw []byte) (*SnapshotSAVDocument, []sav.DocumentObjectOrigin, *heldDocument) {
	doc, origins, err := sav.DecodeDocumentDataWithOrigins(raw)
	var held *heldDocument
	if err == nil {
		if err = savedFormationBlock(&doc); err != nil {
			held = &heldDocument{doc: &doc, origins: origins}
		}
	}
	if err != nil {
		// Partial legacy inputs remain playable through the established import
		// path, but can never masquerade as a complete exportable document.
		reason := fmt.Sprintf("complete SAV document unavailable: %v", err)
		if len(reason) > 4096 {
			reason = reason[:4096]
		}
		return &SnapshotSAVDocument{Version: snapshotSAVDocumentVersion, Unavailable: reason}, nil, held
	}
	state := &SnapshotSAVDocument{Version: snapshotSAVDocumentVersion, Document: &doc}
	// A file decode does not fail.
	_ = decodeWorldCarriersFrom(fileRecord, carrierDecode{state: state})
	return state, origins, nil
}

// SnapshotSAVReferenceMiss is a Token Reference the loaded file held without
// a key of a record at or before its own; ROM1's LOAD keeps 0 for it.
type SnapshotSAVReferenceMiss struct {
	Class     string
	Identity  uint32
	Reference uint32
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
	// ReferenceMisses are the loaded file's Token Reference misses (SAV-1376),
	// kept from LOAD because play rewrites Document; nil writes every Reference.
	ReferenceMisses []SnapshotSAVReferenceMiss `json:",omitempty"`
	// LoadedEffects is, beside a partial state, the world-effect family its
	// record held: a document of the World effect list and the records it
	// reaches, and the WorldEffects binding the World's drivers to them. A
	// SAVE writes that family as it writes a complete document's (DIV-4175).
	// Nil when the record held none.
	LoadedEffects *SnapshotSAVDocument `json:",omitempty"`
}

// ObjectIndex is the DTO's local index, NOT SourceBinding.ArchiveIndex. A
// retired native actor keeps its explicit origin until the graph producer
// retires all of its edges; it must never resurrect the imported actor.
type SnapshotSAVActor struct {
	EntityID    sim.EntityID
	ObjectIndex uint16
	Retired     bool
	// tookTarget is that LOAD gave the actor its record's U5C as the attack
	// target: the link is then the World's to write (keepLoadedActionLinks).
	// It is not part of any encoded form.
	tookTarget bool
}

func cloneSavedDocument(src *SnapshotSAVDocument, allowCurrentSackDrift ...bool) (*SnapshotSAVDocument, error) {
	if src == nil {
		return nil, nil
	}
	if src.Version != snapshotSAVDocumentVersion {
		return nil, fmt.Errorf("saved SAV document version %d is unsupported", src.Version)
	}
	if src.Document == nil {
		if src.Unavailable == "" || len(src.Unavailable) > 4096 || strings.ContainsRune(src.Unavailable, '\x00') || len(src.Actors) != 0 || src.GroupBindings != nil || src.PlayerRoots != nil || src.PlayerPurses != nil || src.Objects != nil || src.ActorEffects != nil || src.WorldEffects != nil || src.ReferenceMisses != nil {
			return nil, fmt.Errorf("saved SAV document has invalid unavailable state")
		}
		effects, err := cloneLoadedEffects(src.LoadedEffects)
		if err != nil {
			return nil, err
		}
		return &SnapshotSAVDocument{Version: src.Version, Unavailable: src.Unavailable, LoadedEffects: effects}, nil
	}
	if src.Unavailable != "" || src.LoadedEffects != nil || len(src.Actors) > len(src.Document.Objects) || len(src.ReferenceMisses) > len(src.Document.Objects) {
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
	out := &SnapshotSAVDocument{Version: src.Version, Document: &doc, Actors: actors, GroupBindings: groups, PlayerRoots: playerRoots, PlayerPurses: purses, Objects: objects, ActorEffects: effects, WorldEffects: worldEffects,
		ReferenceMisses: slices.Clone(src.ReferenceMisses)}
	if err := remapSavedSackDocument(out, permutation); err != nil {
		return nil, err
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
	if ms.World != nil {
		ms.World.SetLoadedRecord(loadedRecordOf(state))
	}
	return decodeWorldCarriersFrom(heldRecord, carrierDecode{w: ms.World, ms: ms, state: state})
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
		clock.FullTick = sim.DerivedFullTick(clock.SubTick)
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
	if err := projectWornWeaponArm(state.Document, state.Actors, ms.World, ms.Start.ConstructionTable); err != nil {
		return nil, err
	}
	if !tableFiles(ms.Start.ConstructionTable).storedOwnWeight {
		if err := projectWornOwnWeight(state.Document, state.Actors, ms.World); err != nil {
			return nil, err
		}
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

func sortSavedActors(actors []SnapshotSAVActor) {
	slices.SortFunc(actors, func(a, b SnapshotSAVActor) int {
		if a.EntityID < b.EntityID {
			return -1
		}
		if a.EntityID > b.EntityID {
			return 1
		}
		return 0
	})
}

// savedActorBindings binds each imported actor of world to its document
// object by the exact import origin, and returns the origins by archive
// index.
func savedActorBindings(world *sim.World, origins []sav.DocumentObjectOrigin) ([]SnapshotSAVActor, map[uint16]uint16, error) {
	byArchive := make(map[uint16]uint16, len(origins))
	for _, origin := range origins {
		if origin.ArchiveIndex == 0 || origin.ObjectIndex == 0 || byArchive[origin.ArchiveIndex] != 0 {
			return nil, nil, fmt.Errorf("SAV document has an ambiguous import origin")
		}
		byArchive[origin.ArchiveIndex] = origin.ObjectIndex
	}
	var actors []SnapshotSAVActor
	for _, e := range world.Entities() {
		if e.SourceBinding.Class != 0 {
			object := byArchive[e.SourceBinding.ArchiveIndex]
			if object == 0 {
				return nil, nil, fmt.Errorf("SAV document lacks exact import origin for actor %d", e.ID)
			}
			actors = append(actors, SnapshotSAVActor{EntityID: e.ID, ObjectIndex: object})
		}
	}
	return actors, byArchive, nil
}

func importSavedDocument(ms *Mission, state *SnapshotSAVDocument, origins []sav.DocumentObjectOrigin) error {
	if state == nil || state.Document == nil {
		return restoreSavedDocument(ms, state)
	}
	bound := *state
	actors, byArchive, err := savedActorBindings(ms.World, origins)
	if err != nil {
		return err
	}
	bound.Actors = append(bound.Actors, actors...)
	sortSavedActors(bound.Actors)
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
	if err := decodeWorldCarriersFrom(documentRecord, carrierDecode{w: ms.World, ms: ms, state: &bound}); err != nil {
		return err
	}
	return importSavedPlayerPurses(ms, &bound)
}
