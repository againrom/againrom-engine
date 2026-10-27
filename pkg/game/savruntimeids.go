package game

import (
	"cmp"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func reserveSavedRuntimeID(used map[uint32]bool, class string, id uint32) {
	used[id] = true
	switch class {
	case "Human", "Unit", "Building", "Sack":
		used[uint32(uint16(id))] = true
	}
}

// projectCurrentRuntimeIDs writes the runtime-id edges of a SAVE from the
// World: each carried projectile's actiontarget and the selection. Every
// placeable record already holds the id its World object carries; no id is
// renumbered (SAV-1371, DIV-2504).
func projectCurrentRuntimeIDs(doc *sav.DocumentData, s Snapshot) error {
	var w sim.World
	if len(s.World) != 0 {
		if err := w.UnmarshalBinary(s.World); err != nil {
			return err
		}
	}
	actors := map[sim.EntityID]uint32{}
	for _, e := range w.Entities() {
		actors[e.ID] = e.SourceBinding.RuntimeID
	}
	// A body a dead manager holds carries its id there, not on the Entity.
	for _, d := range w.OriginalDeadActors() {
		if actors[d.ID] == 0 {
			actors[d.ID] = d.Current.RuntimeID
		}
	}
	structures := map[sim.EntityID]uint32{}
	if rows, _, present := w.SavedStructures(); present {
		for _, row := range rows {
			structures[sim.EntityID(row.ID)] = row.RuntimeID
		}
	}
	var drivers *sim.SavedWorldEffects
	if len(s.World) != 0 {
		drivers = w.SavedWorldEffectDrivers()
	}
	seenRoots := map[string]bool{}
	for _, record := range doc.State.ValueRecords {
		root, _ := splitStatePath(record.Path)
		if len(root) < 3 || seenRoots[root] || !stateRootMatches(doc.State.RootKind, record.Path, "Prj"+root[3:]) {
			continue
		}
		seenRoots[root] = true
		path := "/Prj" + root[3:] + "/actiontarget"
		if _, present, err := doc.State.ReadLeaf(path); err != nil {
			return err
		} else if !present || drivers == nil {
			continue
		}
		id, err := strconv.ParseUint(root[3:], 10, 16)
		if err != nil {
			return err
		}
		for _, d := range drivers.Projectiles {
			if d.ID != uint16(id) || !d.HasTarget || d.TargetDetached {
				continue
			}
			target, found := actors[d.Target]
			if d.TargetStructure {
				target, found = structures[d.Target]
			}
			if !found {
				continue
			}
			i := stateLeafRecords(doc.State.RootKind, doc.State.DirectoryRecords, doc.State.ValueRecords, path)[0]
			doc.State.ValueRecords[i].Value.Int32 = int32(target)
		}
	}
	app := s.ApplicationState
	if _, err := patchStateLeaf(doc.State.RootKind, doc.State.DirectoryRecords, doc.State.ValueRecords, "/Objects/Selection", func(leaf *sav.CityStateValueData) error {
		if len(leaf.Bytes)%4 != 0 {
			return fmt.Errorf("current selection has a partial runtime ID")
		}
		if app == nil || app.LocalOnly {
			return nil
		}
		for n := 0; n < len(leaf.Bytes)/4 && n < len(app.View.Selection); n++ {
			if id, found := actors[sim.EntityID(app.View.Selection[n])]; found {
				binary.LittleEndian.PutUint32(leaf.Bytes[4*n:], id)
			}
		}
		return nil
	}); err != nil {
		return err
	}
	a, err := readCurrentActions(doc)
	if err != nil || a == nil {
		return err
	}
	for id, value := range a.Values {
		value.RuntimeID = nil
		a.Values[id] = value
	}
	if a.StructureBindings != nil {
		for i := range *a.StructureBindings {
			(*a.StructureBindings)[i].RuntimeID = nil
		}
	}
	for i := range a.ArchiveCoordinates {
		row := &a.ArchiveCoordinates[i]
		if row.Generated {
			anchor := currentArchiveModeAnchor(&doc.Objects[row.Object-1])
			row.ModeAnchor = &anchor
		}
	}
	raw, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return sav.SetNativeActions(&doc.State, raw)
}

func cityRuntimeIDs(objects []sav.CityObjectData) map[uint32]bool {
	used := map[uint32]bool{0: true}
	for _, o := range objects {
		var token []byte
		switch {
		case o.Unit != nil:
			token = o.Unit.Token
		case o.Item != nil:
			token = o.Item.Token
		case o.Effect != nil:
			token = o.Effect.Token
		}
		if len(token) >= 16 {
			reserveSavedRuntimeID(used, o.Class, binary.LittleEndian.Uint32(token[12:16]))
		}
	}
	return used
}

func nextSavedRuntimeID(used map[uint32]bool) uint32 {
	id := uint32(1)
	for used[id] {
		id++
	}
	used[id] = true
	return id
}

// runtimeIDLoads lists the Token-derived records of a loaded document in
// archive order, the order World LOAD marks their saved ids (SAV-1371).
func runtimeIDLoads(doc *sav.DocumentData, origins []sav.DocumentObjectOrigin) []sim.RuntimeIDLoad {
	if doc == nil {
		return nil
	}
	ordered := slices.Clone(origins)
	slices.SortFunc(ordered, func(a, b sav.DocumentObjectOrigin) int { return cmp.Compare(a.ArchiveIndex, b.ArchiveIndex) })
	var out []sim.RuntimeIDLoad
	for _, origin := range ordered {
		if origin.ObjectIndex == 0 || int(origin.ObjectIndex) > len(doc.Objects) {
			continue
		}
		r := &doc.Objects[origin.ObjectIndex-1]
		if id, err := savedStructureValue(r, "RuntimeID"); err == nil {
			out = append(out, sim.RuntimeIDLoad{Saved: id, Sack: r.Class == "Sack"})
		}
	}
	return out
}

// restoreLoadedRuntimeIDs rebuilds the World's runtime-id allocator from the
// document a LOAD read; an original SAV and one this engine wrote load alike.
func restoreLoadedRuntimeIDs(ms *Mission, state *SnapshotSAVDocument, origins []sav.DocumentObjectOrigin) {
	if state == nil || state.Document == nil {
		return
	}
	var actors []sim.ActorRuntimeIDLoad
	for _, b := range state.Actors {
		if b.Retired || b.ObjectIndex == 0 || int(b.ObjectIndex) > len(state.Document.Objects) {
			continue
		}
		if id, err := savedStructureValue(&state.Document.Objects[b.ObjectIndex-1], "RuntimeID"); err == nil {
			actors = append(actors, sim.ActorRuntimeIDLoad{Entity: b.EntityID, Saved: id})
		}
	}
	ms.World.RestoreActorRuntimeIDs(actors)
	ms.World.RestoreRuntimeIDs(runtimeIDLoads(state.Document, origins))
}
