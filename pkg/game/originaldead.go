package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"fmt"
)

func (mw *mapWorld) retainNewSourceDeadReferences(first int) {
	if mw.mission == nil || mw.mission.state == nil || mw.mission.state.savedDocument == nil || mw.mission.state.savedDocument.Document == nil {
		return
	}
	doc := mw.mission.state.savedDocument.Document
	for _, dead := range mw.world.OriginalDeadActors()[first:] {
		var record *sav.DocumentRecordData
		for i := range doc.Objects {
			r := &doc.Objects[i]
			if r.Class != savedActorClass(dead.Source.Class) {
				continue
			}
			identity, _ := savedStructureValue(r, "Identity")
			if identity != dead.Source.Identity {
				continue
			}
			if record != nil {
				record = nil
				break
			}
			record = r
		}
		if record == nil {
			continue
		}
		var references [5]uint32
		for i, name := range []string{"U5C", "U64", "U44", "", "U40"} {
			if name != "" {
				references[i], _ = savedStructureValue(record, name)
			}
		}
		if refs, _ := savedObjectRefs(record, "U68"); len(refs) != 0 && refs[0] != 0 && int(refs[0]) <= len(doc.Objects) {
			references[3], _ = savedStructureValue(&doc.Objects[refs[0]-1], "Identity")
		}
		mw.world.RetainOriginalDeadReferences(dead.ID, dead.Source.Identity, references)
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
		if (d.Effects != 0 || d.Carried != 0 || (d.Worn != 0 && !terminalWeapon)) && !retainedTerminalDeadContents(state, origins, d) {
			return fmt.Errorf("original dead actor %#x: unsupported effects/carried/worn contents", d.Identity)
		}
		var heldWeapon sim.OriginalDeadWeapon
		if terminalWeapon {
			heldWeapon = sim.OriginalDeadWeapon(d.HeldWeapon)
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
		batch = append(batch, sim.OriginalDeadActor{ID: id, Source: sim.OriginalDeadSource{
			Identity: d.Identity, ArchiveIndex: d.ArchiveIndex, MapUnitID: d.MapUnitID, Class: class,
			TerrainKey: d.TerrainKey, OwnerKey: d.OwnerKey, References: d.References,
			ContainerPresent: d.ContainerPresent, ContainerTail: d.ContainerTail,
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
	if dead.Stage != 5 || dead.Identity == 0 || dead.ArchiveIndex == 0 || state == nil || state.Document == nil {
		return false
	}
	var object uint16
	for _, origin := range origins {
		if origin.ArchiveIndex == dead.ArchiveIndex {
			if object != 0 || origin.ObjectIndex == 0 || int(origin.ObjectIndex) > len(state.Document.Objects) {
				return false
			}
			object = origin.ObjectIndex
		}
	}
	if object == 0 {
		return false
	}
	for _, origin := range origins {
		if origin.ObjectIndex == object && origin.ArchiveIndex != dead.ArchiveIndex {
			return false
		}
	}
	record := &state.Document.Objects[object-1]
	identity, err := savedStructureValue(record, "Identity")
	if err != nil || identity != dead.Identity || record.Class != dead.Class {
		return false
	}
	stage, err := savedStructureValue(record, "Stage")
	if err != nil || stage != uint32(dead.Stage) {
		return false
	}
	found := false
	for _, root := range state.Document.DeadActors {
		if root == 0 || int(root) > len(state.Document.Objects) {
			return false
		}
		if root == object {
			found = true
			continue
		}
		key, err := savedStructureValue(&state.Document.Objects[root-1], "Identity")
		if err == nil && key == dead.Identity {
			return false
		}
	}
	return found
}
