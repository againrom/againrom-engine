package game

import (
	"cmp"
	"fmt"
	"slices"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func restoreCurrentActorIdentities(ms *Mission, a *currentActionData) error {
	byObject := map[uint16]sim.EntityID{}
	for _, b := range ms.savedDocument.Actors {
		if !b.Retired {
			byObject[b.ObjectIndex] = b.EntityID
		}
	}
	for _, d := range ms.World.OriginalDeadActors() {
		for i, r := range ms.savedDocument.Document.Objects {
			key, err := savedStructureValue(&r, "Identity")
			if err == nil && key == d.Source.Identity {
				byObject[uint16(i+1)] = d.ID
			}
		}
	}
	terminalRows, err := currentTerminalImportRows(ms, a)
	if err != nil {
		return err
	}
	terminal := map[sim.EntityID]bool{}
	for _, row := range terminalRows {
		terminal[row.ID] = true
	}
	// Terminal rows are in the saved ID namespace and the constructed World in
	// the construction namespace. A constructor no SAV actor binds is an
	// authored placement whose actor has left: retire it before the IDs meet.
	bound := map[sim.EntityID]bool{}
	for _, id := range byObject {
		bound[id] = true
	}
	var unbound []sim.EntityID
	for _, e := range ms.World.Entities() {
		if e.SourceBinding.Class == 0 && e.MapUnitID != 0 && !bound[e.ID] {
			unbound = append(unbound, e.ID)
		}
	}
	if err := ms.World.RetireUnboundConstructors(unbound); err != nil {
		return err
	}
	ids := map[sim.EntityID]sim.EntityID{}
	reserved := map[sim.EntityID]bool{}
	var next uint64
	for _, b := range a.Bindings {
		if b.Structure {
			continue
		}
		if reserved[b.ID] || b.Missing && b.Object != 0 {
			return fmt.Errorf("invalid current actor identity binding")
		}
		reserved[b.ID] = true
		next = max(next, uint64(b.ID)+1)
		if terminal[b.ID] && !bound[b.ID] {
			// A retired constructor's references keep following the terminal
			// ID when no bound actor holds that construction ID.
			ids[b.ID] = b.ID
		}
		if b.Missing || terminal[b.ID] {
			continue
		}
		old, ok := byObject[b.Object]
		if _, repeated := ids[old]; !ok || repeated || b.Object == 0 {
			return fmt.Errorf("current actor identity has no unique ordinary object")
		}
		ids[old] = b.ID
	}
	refs := ms.World.ActorIdentityReferences()
	ms.currentActorIdentityFields(func(id *sim.EntityID) { refs = append(refs, *id) })
	slices.Sort(refs)
	refs = slices.Compact(refs)
	for _, old := range refs {
		if _, ok := ids[old]; ok {
			continue
		}
		id := old
		if reserved[id] {
			for next <= uint64(^sim.EntityID(0)) && reserved[sim.EntityID(next)] {
				next++
			}
			if next > uint64(^sim.EntityID(0)) {
				return fmt.Errorf("current actor identity namespace exhausted")
			}
			id = sim.EntityID(next)
			next++
		}
		ids[old], reserved[id] = id, true
	}
	var floor *uint64
	if a.Policy != nil {
		floor = a.Policy.EntityIDFloor
	}
	if err := ms.World.RestoreActorIdentities(ids, floor); err != nil {
		return err
	}
	if len(terminalRows) != 0 {
		if err := ms.World.ImportCurrentTerminalActors(terminalRows); err != nil {
			return err
		}
	}
	ms.Start.IDs = slices.Clone(ms.Start.IDs)
	ms.ActorManifest = cloneActorManifest(ms.ActorManifest)
	if ms.actorRegistry != nil {
		r := *ms.actorRegistry
		r.actors = slices.Clone(r.actors)
		ms.actorRegistry = &r
	}
	ms.currentActorIdentityFields(func(id *sim.EntityID) { *id = ids[*id] })
	if ms.Start.Roster != nil {
		roster := make(map[sim.EntityID]mapload.PartyMember, len(ms.Start.Roster))
		for id, p := range ms.Start.Roster {
			roster[ids[id]] = p
		}
		ms.Start.Roster = roster
	}
	if ms.DeadArt != nil {
		art := make(map[sim.EntityID]uint16, len(ms.DeadArt))
		for id, v := range ms.DeadArt {
			art[ids[id]] = v
		}
		ms.DeadArt = art
	}
	slices.SortFunc(ms.savedDocument.Actors, func(x, y SnapshotSAVActor) int { return cmp.Compare(x.EntityID, y.EntityID) })
	if g := ms.savedDocument.GroupBindings; g != nil {
		slices.SortFunc(g.Members, savedGroupMemberCompare)
	}
	return nil
}

// Map keys are visited as values for collection, then rebuilt by the caller.
func (ms *Mission) currentActorIdentityFields(ref func(*sim.EntityID)) {
	for i := range ms.Start.IDs {
		ref(&ms.Start.IDs[i])
	}
	for id := range ms.Start.Roster {
		ref(&id)
	}
	for id := range ms.DeadArt {
		ref(&id)
	}
	if ms.ActorManifest != nil {
		for i := range ms.ActorManifest.Actors {
			ref(&ms.ActorManifest.Actors[i].ID)
		}
	}
	if ms.actorRegistry != nil {
		for i := range ms.actorRegistry.actors {
			ref(&ms.actorRegistry.actors[i].ID)
		}
	}
	s := ms.savedDocument
	for i := range s.Actors {
		ref(&s.Actors[i].EntityID)
	}
	if s.GroupBindings != nil {
		for i := range s.GroupBindings.Members {
			m := &s.GroupBindings.Members[i]
			if m.Bound {
				ref(&m.EntityID)
			}
		}
	}
	if s.ActorEffects != nil {
		for i := range s.ActorEffects.Rows {
			ref(&s.ActorEffects.Rows[i].Entity)
		}
	}
	if s.Objects != nil {
		for i := range s.Objects.Containers {
			o := &s.Objects.Containers[i].Owner
			if o.Kind == sim.SavedOwnerActorPack || o.Kind == sim.SavedOwnerActorWorn {
				ref(&o.Entity)
			}
		}
	}
}
