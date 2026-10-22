package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"fmt"
	"slices"
)

// bodyRecordReferences reads a body's order words from its loaded record:
// U5C, U64 and the identity U68 names. U44 is the book selection and U40 the
// kill credit, each written from its own World state.
func bodyRecordReferences(doc *sav.DocumentData, record *sav.DocumentRecordData) [3]uint32 {
	var references [3]uint32
	for i, name := range []string{"U5C", "U64"} {
		references[i], _ = savedStructureValue(record, name)
	}
	if refs, _ := savedObjectRefs(record, "U68"); len(refs) != 0 && refs[0] != 0 && int(refs[0]) <= len(doc.Objects) {
		references[2], _ = savedStructureValue(&doc.Objects[refs[0]-1], "Identity")
	}
	return references
}

// writeBodyReferences writes a body's order words from the World. A body the
// World holds no loaded words for, one made in play, is written with
// none: no order machine ran on it. A U68 identity no object of this
// document holds is written as no object.
func writeBodyReferences(doc *sav.DocumentData, record *sav.DocumentRecordData, references [3]uint32) {
	savedObjectSetValue(record, "U5C", references[0])
	savedObjectSetValue(record, "U64", references[1])
	var object uint16
	if references[2] != 0 {
		for i := range doc.Objects {
			if key, err := savedStructureValue(&doc.Objects[i], "Identity"); err == nil && key == references[2] {
				if object != 0 {
					object = 0
					break
				}
				object = uint16(i + 1)
			}
		}
	}
	if refs, _ := savedObjectRefs(record, "U68"); object != 0 || len(refs) != 0 && refs[0] != 0 {
		savedObjectSetRefs(record, "U68", []uint16{object}, false)
	}
}

// applyOriginalDead joins a dead record with a nonzero MapUnitID to the one
// authored placement it names. ROM1 instead reconstructs and rebinds its own
// object graph; this bounded join is disclosed, and absence from the list
// never implies death.
//
// A record with MapUnitID 0 (SAV-DEADLOAD-125, promoted: ROM1's own load
// never makes this join at all) never was an authored map unit -- the
// observed producer is a hired mercenary, never an ALM placement -- and is
// carried instead as a virtual dead actor with no live entity, at a freshly
// minted ID beyond every ID this World already holds.
func applyOriginalDead(ms *Mission, source []sav.DeadActor, report *OriginalSaveResume, states ...*SnapshotSAVDocument) error {
	var state *SnapshotSAVDocument
	if len(states) > 1 {
		return fmt.Errorf("original dead actors: repeated current document")
	}
	if len(states) == 1 {
		state = states[0]
	}
	return applyOriginalDeadWithOrigins(ms, source, report, state, nil)
}

func applyOriginalDeadWithOrigins(ms *Mission, source []sav.DeadActor, report *OriginalSaveResume, state *SnapshotSAVDocument, origins []sav.DocumentObjectOrigin) error {
	if ms == nil || ms.Map == nil || ms.World == nil {
		return fmt.Errorf("original dead actors: no map/world")
	}
	retained, err := currentRetainedDeadKeys(state)
	if err != nil {
		return err
	}
	currentTerminal, err := currentTerminalActorKeys(state)
	if err != nil {
		return err
	}
	var current []sim.EntityID
	authored := make(map[uint16]int)
	for _, u := range ms.Map.Units {
		authored[u.UnitID]++
	}
	byMap := make(map[uint16][]sim.EntityID)
	for _, e := range ms.World.Entities() {
		byMap[e.MapUnitID] = append(byMap[e.MapUnitID], e.ID)
	}
	party := make(map[sim.EntityID]bool)
	for _, id := range ms.Start.IDs {
		party[id] = true
	}
	next, nextOK := ms.World.NextEntityID()
	batch := make([]sim.OriginalDeadActor, 0, len(source))
	type terminalBody struct {
		id     sim.EntityID
		dead   sav.DeadActor
		weapon bool
	}
	var terminal []terminalBody
	for _, d := range source {
		if currentTerminal[d.Identity] {
			continue
		}
		binding, bound := ms.actorRegistry.actor(d.Off)
		currentBody := bound && binding.Source.CurrentEntity && retained[d.Identity]
		if bound && binding.Source.CurrentEntity && !currentBody {
			continue
		}
		if d.Stage == 1 {
			binding, ok := ms.actorRegistry.actor(d.Off)
			if !ok || !binding.Source.Dying() || binding.Source.Identity != d.Identity || binding.Source.ArchiveIndex != d.ArchiveIndex {
				return fmt.Errorf("original dying root %#x lacks exact full actor binding", d.Identity)
			}
			continue // full profile, equipment and timer use the ordinary actor importer
		}
		var id sim.EntityID
		if currentBody {
			id = binding.ID
			current = append(current, id)
			delete(retained, d.Identity)
		} else if d.MapUnitID == 0 {
			if !nextOK {
				return fmt.Errorf("original dead actor %#x: entity namespace exhausted", d.Identity)
			}
			id = next
			if next == ^sim.EntityID(0) {
				nextOK = false
			} else {
				next++
			}
			// TypeID is the class this virtual corpse is drawn as
			// (world.go's missionAppearanceArt). Recorded here, against the
			// same id ImportOriginalDeadActors admits below, because this
			// is the one place the freshly parsed archive field and the
			// freshly minted id are both in hand; sim.OriginalDeadSource
			// itself carries no class-resolving field (originaldead.go's
			// own doc).
			if ms.DeadArt == nil {
				ms.DeadArt = map[sim.EntityID]uint16{}
			}
			ms.DeadArt[id] = d.TypeID
		} else {
			ids := byMap[d.MapUnitID]
			if authored[d.MapUnitID] != 1 || len(ids) != 1 || party[ids[0]] || originalPartyCarriesMapUnit(ms.Party, d.MapUnitID) {
				return fmt.Errorf("original dead actor %#x: MapUnitID %d is missing, ambiguous or party-bound", d.Identity, d.MapUnitID)
			}
			id = ids[0]
		}
		terminalWeapon := d.Class == "Unit" && d.Stage == 5 && d.Worn == 1 && d.HeldWeapon.Present
		effectsOnly := d.Carried == 0 && (d.Worn == 0 || terminalWeapon) && retainedDeadEffectsCount(state, origins, d) == d.Effects
		if (d.Effects != 0 || d.Carried != 0 || (d.Worn != 0 && !terminalWeapon)) && !effectsOnly && !retainedTerminalDeadContents(state, origins, d) {
			return fmt.Errorf("original dead actor %#x: unsupported effects/carried/worn contents", d.Identity)
		}
		var heldWeapon sim.OriginalDeadWeapon
		if terminalWeapon {
			heldWeapon = sim.OriginalDeadWeapon(d.HeldWeapon)
		}
		if d.Stage == 5 {
			terminal = append(terminal, terminalBody{id, d, terminalWeapon})
		}
		var class uint8
		switch d.Class {
		case "Unit":
			class = 1
		case "Human":
			class = 2
		case "Humanoid":
			class = 3
		}
		maxHP := int32(d.MaxHP)
		if currentBody {
			if health, present := ms.actorRegistry.health[id]; present {
				maxHP = health.MaxHP
			}
		}
		batch = append(batch, sim.OriginalDeadActor{ID: id, Source: sim.OriginalDeadSource{
			Identity: d.Identity, ArchiveIndex: d.ArchiveIndex, MapUnitID: d.MapUnitID, Class: class,
			TerrainKey: d.TerrainKey, OwnerKey: d.OwnerKey, References: d.References,
			ContainerPresent: d.ContainerPresent, ContainerTail: d.ContainerTail,
			MaxHP: maxHP, MaxHPPresent: true,
			HeldWeapon: heldWeapon,
			State:      sim.DeadActorState{RuntimeID: d.RuntimeID, Cell: d.Cell, FineX: d.FineX, FineY: d.FineY, Stage: d.Stage, HP: d.HP, Timer: d.Timer},
		}})
	}
	if len(retained) != 0 {
		return fmt.Errorf("current retained actor has no materialized ordinary body")
	}
	if err := ms.World.ImportOriginalDeadActors(batch, current...); err != nil {
		return fmt.Errorf("original dead actors: %w", err)
	}
	// A terminal record's items stay passive on the World (sim.PassiveBody):
	// SAVE writes them back and nothing in play reads them.
	var kept []sim.PassiveBody
	for _, body := range terminal {
		if state == nil {
			break
		}
		contents := passiveRecordContents(state.Document, deadRecordObject(state, origins, body.dead), !body.weapon)
		if passiveBodyEmpty(contents) {
			continue
		}
		contents.ID = body.id
		kept = append(kept, contents)
	}
	if len(kept) != 0 {
		if err := ms.World.RestorePassiveBodies(kept); err != nil {
			return fmt.Errorf("original dead actor contents: %w", err)
		}
	}
	for _, d := range source {
		switch {
		case d.Stage == 1:
			// The full actor importer reports first-stage roots as dying.
		case d.MapUnitID == 0:
			report.UnboundRestored++
		case d.Stage == 5:
			report.TerminalRestored++
		default:
			report.CorpsesRestored++
		}
	}
	return nil
}

func retainedTerminalDeadContents(state *SnapshotSAVDocument, origins []sav.DocumentObjectOrigin, dead sav.DeadActor) bool {
	return dead.Stage == 5 && deadRecordObject(state, origins, dead) != 0
}

// deadRecordObject is the one loaded object a dead record names: its archive
// index maps to exactly one object, that object is a DeadActors root with the
// record's identity, class and stage, and no other root holds the identity.
// Zero is no such object.
func deadRecordObject(state *SnapshotSAVDocument, origins []sav.DocumentObjectOrigin, dead sav.DeadActor) uint16 {
	if dead.Identity == 0 || dead.ArchiveIndex == 0 || state == nil || state.Document == nil {
		return 0
	}
	var object uint16
	for _, origin := range origins {
		if origin.ArchiveIndex == dead.ArchiveIndex {
			if object != 0 || origin.ObjectIndex == 0 || int(origin.ObjectIndex) > len(state.Document.Objects) {
				return 0
			}
			object = origin.ObjectIndex
		}
	}
	if object == 0 {
		return 0
	}
	for _, origin := range origins {
		if origin.ObjectIndex == object && origin.ArchiveIndex != dead.ArchiveIndex {
			return 0
		}
	}
	record := &state.Document.Objects[object-1]
	identity, err := savedStructureValue(record, "Identity")
	if err != nil || identity != dead.Identity || record.Class != dead.Class {
		return 0
	}
	stage, err := savedStructureValue(record, "Stage")
	if err != nil || stage != uint32(dead.Stage) {
		return 0
	}
	found := false
	for _, root := range state.Document.DeadActors {
		if root == 0 || int(root) > len(state.Document.Objects) {
			return 0
		}
		if root == object {
			found = true
			continue
		}
		key, err := savedStructureValue(&state.Document.Objects[root-1], "Identity")
		if err == nil && key == dead.Identity {
			return 0
		}
	}
	if !found {
		return 0
	}
	return object
}

// retainedDeadEffectsCount is the number of Effects a dead record keeps
// unexecuted, or -1 when it keeps none this way. A DeadActors root in no
// Group is never on the tick list (SAV-1171), and only the actor tick runs an
// attachment (MAGIC-CONSUME-144), so its Effects stay as stored. The writer
// re-emits them from the loaded document (retainedDeadEffects).
func retainedDeadEffectsCount(state *SnapshotSAVDocument, origins []sav.DocumentObjectOrigin, dead sav.DeadActor) int {
	object := deadRecordObject(state, origins, dead)
	if object == 0 {
		return -1
	}
	effects, ok := retainedDeadEffects(state.Document, object)
	if !ok {
		return -1
	}
	return len(effects)
}

// retainedDeadEffects reads the Effects of the DeadActors root object when
// that root is in no Group and owns each Effect alone, one per spell id.
func retainedDeadEffects(doc *sav.DocumentData, object uint16) ([]sim.SavedEffectObject, bool) {
	if doc == nil || object == 0 || int(object) > len(doc.Objects) || !slices.Contains(doc.DeadActors, object) {
		return nil, false
	}
	incoming := savedDocumentIncoming(doc)
	refs, found := savedObjectRefs(&doc.Objects[object-1], "Effects")
	if !found || len(refs) == 0 || incoming[object] != 1 {
		return nil, false
	}
	var out []sim.SavedEffectObject
	spells := map[uint8]bool{}
	for _, index := range refs {
		if index == 0 || int(index) > len(doc.Objects) || incoming[index] != 1 {
			return nil, false
		}
		effect, err := savedEffectRecord(&doc.Objects[index-1])
		if err != nil || effect.E0C > 31 || spells[effect.E0C] {
			return nil, false
		}
		spells[effect.E0C] = true
		out = append(out, effect)
	}
	return out, true
}
