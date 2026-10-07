package game

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

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

func runtimePlaceable(class string) bool {
	switch class {
	case "Unit", "Humanoid", "Human", "Building", "Outpost", "Tavern", "Shop", "Sack":
		return true
	}
	return false
}

// Runtime IDs occupy a separate namespace from archive addresses and item
// tokens. The last projection rewrites only placeables and their runtime edges.
func projectCurrentRuntimeIDs(doc *sav.DocumentData, s Snapshot) error {
	oldObjects := map[uint32][]uint16{}
	for i, r := range doc.Objects {
		if !runtimePlaceable(r.Class) {
			continue
		}
		id, err := savedStructureValue(&r, "RuntimeID")
		if err != nil {
			return err
		}
		oldObjects[id] = append(oldObjects[id], uint16(i+1))
	}
	a, err := readCurrentActions(doc)
	if err != nil {
		return err
	}
	actors := map[sim.EntityID]uint16{}
	if a != nil {
		for _, b := range a.Bindings {
			if !b.Structure && !b.Missing {
				actors[b.ID] = b.Object
			}
		}
	}
	var w sim.World
	var drivers *sim.SavedWorldEffects
	if len(s.World) > 0 {
		if err := w.UnmarshalBinary(s.World); err != nil {
			return err
		}
		drivers = w.SavedWorldEffectDrivers()
	}
	used := map[uint32]bool{0: true}
	pinned := map[uint16]uint32{}
	registry := w.SavedObjects()
	if a != nil && registry != nil {
		sacks := map[sim.SavedObjectID]bool{}
		for _, sack := range registry.Sacks {
			if !sack.Retired {
				sacks[sack.ID] = true
			}
		}
		for _, row := range a.Ownership {
			if row.Kind != 4 || row.Object == 0 || !sacks[row.ID] {
				continue
			}
			if int(row.Object) > len(doc.Objects) || doc.Objects[row.Object-1].Class != "Sack" {
				return fmt.Errorf("current Sack runtime binding is invalid")
			}
			id, err := savedStructureValue(&doc.Objects[row.Object-1], "RuntimeID")
			if err != nil {
				return err
			}
			pinned[row.Object] = id
			reserveSavedRuntimeID(used, "Sack", id)
		}
	}
	targets := map[int]uint16{}
	for i, leaf := range doc.State.ValueRecords {
		if !strings.HasPrefix(leaf.Path, "/Prj") || !strings.HasSuffix(leaf.Path, "/actiontarget") {
			continue
		}
		old := uint32(leaf.Value.Int32)
		if old == 0 {
			continue
		}
		var object uint16
		if drivers != nil {
			id, err := strconv.ParseUint(strings.TrimSuffix(strings.TrimPrefix(leaf.Path, "/Prj"), "/actiontarget"), 10, 16)
			if err != nil {
				return err
			}
			for _, d := range drivers.Projectiles {
				if d.ID == uint16(id) && d.HasTarget && !d.TargetDetached {
					object = actors[d.Target]
				}
			}
		} else if len(oldObjects[old]) == 1 {
			object = oldObjects[old][0]
		}
		if object == 0 {
			used[old], used[uint32(uint16(old))] = true, true
		} else {
			targets[i] = object
		}
	}
	nativePin := pinNativeProjectileTargets(&w, drivers, actors, used)
	retained := map[uint32]uint16{}
	for i, r := range doc.Objects {
		object := uint16(i + 1)
		_, isPinned := pinned[object]
		_, isNativePin := nativePin[object]
		if !runtimePlaceable(r.Class) || isPinned || isNativePin {
			continue
		}
		id, _ := savedStructureValue(&r, "RuntimeID")
		if id == 0 || id > 65535 || used[id] {
			continue
		}
		retained[id] = object
		used[id] = true
	}
	byObject := map[uint16]uint32{}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if !runtimePlaceable(r.Class) {
			continue
		}
		if id, ok := pinned[uint16(i+1)]; ok {
			byObject[uint16(i+1)] = id
			continue
		}
		if id, ok := nativePin[uint16(i+1)]; ok {
			byObject[uint16(i+1)] = id
			savedObjectSetValue(r, "RuntimeID", id)
			continue
		}
		old, _ := savedStructureValue(r, "RuntimeID")
		stage, _ := savedStructureValue(r, "Stage")
		if old == 0 && stage == 5 {
			byObject[uint16(i+1)] = 0
			continue
		}
		id := old
		if retained[id] != uint16(i+1) {
			id = nextSavedRuntimeID(used)
		}
		if id > 65535 {
			return fmt.Errorf("current placeable runtime namespace exhausted")
		}
		byObject[uint16(i+1)] = id
		savedObjectSetValue(r, "RuntimeID", id)
	}
	for i := range doc.State.ValueRecords {
		leaf := &doc.State.ValueRecords[i]
		if object := targets[i]; object != 0 {
			leaf.Value.Int32 = int32(byObject[object])
		}
		if leaf.Path != "/Objects/Selection" {
			continue
		}
		if len(leaf.Value.Bytes)%4 != 0 {
			return fmt.Errorf("current selection has a partial runtime ID")
		}
		for n := 0; n < len(leaf.Value.Bytes)/4; n++ {
			old := binary.LittleEndian.Uint32(leaf.Value.Bytes[4*n:])
			object := uint16(0)
			if s.ApplicationState != nil && !s.ApplicationState.LocalOnly && n < len(s.ApplicationState.View.Selection) {
				object = actors[sim.EntityID(s.ApplicationState.View.Selection[n])]
			}
			if object == 0 && len(oldObjects[old]) == 1 {
				object = oldObjects[old][0]
			}
			if object == 0 || byObject[object] == 0 {
				return fmt.Errorf("current selection has no unique placeable binding")
			}
			binary.LittleEndian.PutUint32(leaf.Value.Bytes[4*n:], byObject[object])
		}
	}
	if a != nil {
		sourceCoordinates := make(map[sim.EntityID]uint32)
		for _, e := range w.Entities() {
			if e.SourceBinding.Class == 0 {
				continue
			}
			sourceCoordinates[e.ID] = e.SourceBinding.RuntimeID
			object := actors[e.ID]
			wire, found := byObject[object]
			value, present := a.Values[e.ID]
			if !found || !present || !value.SourceBound {
				return fmt.Errorf("current runtime coordinate lacks an exact actor value binding")
			}
			value.RuntimeID = nil
			if wire != e.SourceBinding.RuntimeID {
				value.RuntimeID = &sim.ActorRuntimeCoordinate{Wire: wire, Value: e.SourceBinding.RuntimeID}
			}
			a.Values[e.ID] = value
		}
		for _, dead := range w.OriginalDeadActors() {
			if dead.Current.Stage == 5 {
				continue
			}
			wire, found := byObject[actors[dead.ID]]
			if !found || wire == 0 || dead.Current.RuntimeID != dead.Source.State.RuntimeID {
				return fmt.Errorf("current retained runtime coordinate lacks an exact body binding")
			}
			if source, present := sourceCoordinates[dead.ID]; present && source != dead.Current.RuntimeID {
				return fmt.Errorf("current actor and retained body runtime coordinates disagree")
			}
			value := a.Values[dead.ID]
			if value.RuntimeID != nil && (value.RuntimeID.Wire != wire || value.RuntimeID.Value != dead.Current.RuntimeID) {
				return fmt.Errorf("current actor and retained body runtime coordinates disagree")
			}
			if wire != dead.Current.RuntimeID {
				value.RuntimeID = &sim.ActorRuntimeCoordinate{Wire: wire, Value: dead.Current.RuntimeID}
				a.Values[dead.ID] = value
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
	return nil
}

func savedRuntimeIDs(objects []sav.DocumentRecordData) map[uint32]bool {
	used := map[uint32]bool{0: true}
	for _, r := range objects {
		if id, err := savedStructureValue(&r, "RuntimeID"); err == nil {
			reserveSavedRuntimeID(used, r.Class, id)
		}
	}
	return used
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

// pinNativeProjectileTargets chooses the runtime identity of each native actor
// a carried projectile is aimed at: the entity key the World's record already
// holds (sim.ProjectileTargetKey), so a SAVE then LOAD returns the same record.
// An actor is left to the ordinary allocator when its key is out of the
// namespace, a pinned Sack or an earlier pin holds it, or the record holds a
// restored Document identity instead of the entity key. Another object that
// held the key is given a new identity by the ordinary allocator.
func pinNativeProjectileTargets(w *sim.World, drivers *sim.SavedWorldEffects, actors map[sim.EntityID]uint16, used map[uint32]bool) map[uint16]uint32 {
	pins := map[uint16]uint32{}
	if drivers == nil {
		return pins
	}
	records := map[uint16]int32{}
	for _, p := range w.SavedProjectiles().Items {
		records[p.ID] = p.ActionTarget
	}
	for _, d := range drivers.Projectiles {
		if d.Retired || !d.HasTarget || d.TargetDetached {
			continue
		}
		var target sim.Entity
		found := false
		for _, e := range w.Entities() {
			if e.ID == d.Target {
				target, found = e, true
				break
			}
		}
		if !found || target.SourceBinding.Class != 0 && target.SourceBinding.RuntimeID != 0 {
			continue
		}
		object := actors[target.ID]
		key := uint32(sim.ProjectileTargetKey(target))
		if object == 0 || key == 0 || key > 65535 || used[key] || records[d.ID] != int32(key) {
			continue
		}
		pins[object] = key
		reserveSavedRuntimeID(used, "Unit", key)
	}
	return pins
}
