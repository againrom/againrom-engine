package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"encoding/binary"
	"fmt"
)

func originalGroupReference(r sav.ActorReference) sim.SavedGroupReference {
	v := sim.SavedGroupReference{Key: r.Key}
	if r.Resolved {
		v.Archive, v.Class = r.ArchiveIndex, 2
		if r.Class == "Player" {
			v.Class, v.Owner = 1, uint32(r.PlayerSlot)
		}
	}
	return v
}

// Apply after both living and late-dead admission, before candidate adoption.
// Dead identity is joined on the decoded archive identity, never MapUnitID.
func applyOriginalGroups(ms *Mission, report *OriginalSaveResume) error {
	if ms == nil || ms.actorRegistry == nil {
		return fmt.Errorf("original Groups: missing actor registry")
	}
	r := ms.actorRegistry
	bindings := map[uint16]sim.EntityID{}
	entities := map[sim.EntityID]bool{}
	for _, e := range ms.World.Entities() {
		entities[e.ID] = true
	}
	for _, b := range r.actors {
		bindings[b.Source.ArchiveIndex] = b.ID
	}
	for _, d := range ms.World.OriginalDeadActors() {
		if !entities[d.ID] {
			continue
		}
		for _, a := range r.sources {
			if a.ArchiveIndex == d.Source.ArchiveIndex && a.Identity == d.Source.Identity {
				if id, exists := bindings[a.ArchiveIndex]; exists && id != d.ID {
					return fmt.Errorf("original Groups: duplicate bound archive %d", a.ArchiveIndex)
				}
				bindings[a.ArchiveIndex] = d.ID
			}
		}
	}
	groups := make([]sim.SavedGroup, len(r.groups))
	for i, source := range r.groups {
		g := &groups[i]
		g.ID, g.Selector = source.Ordinal, source.Selector
		g.Reference, g.Owner = originalGroupReference(source.Reference), originalGroupReference(source.Owner)
		copy(g.AI[:], source.AI[:76])
		g.Words, g.Path = source.Words, source.AIWords
		for _, archive := range source.Members {
			id, bound := bindings[archive]
			g.Members = append(g.Members, sim.SavedGroupMember{Archive: archive, Entity: id, Bound: bound})
		}
	}
	var orders []sim.SavedActorOrder
	// Token Identity is the writing-process address key (SAV-PTRMAP-035),
	// independently of archive, runtime, map and native identities. The actor
	// graph has already rejected duplicate nonzero keys; only admitted actors
	// may become typed targets. Misses retain the raw dword.
	targets := map[uint32]sim.EntityID{}
	for _, a := range r.sources {
		if id, bound := bindings[a.ArchiveIndex]; bound && a.Identity != 0 {
			targets[a.Identity] = id
		}
	}
	for _, a := range r.sources {
		if id, bound := bindings[a.ArchiveIndex]; bound && a.HasOrder {
			o := sim.SavedActorOrder{Entity: id, State: a.ActorState, Patrol: a.Patrol, RepairStage: a.Stage}
			copy(o.Raw[:], a.Order[:144])
			if a.Stage == 0 && (o.State == 8 || o.State == 0x11) {
				if target, ok := targets[binary.LittleEndian.Uint32(o.Raw[0x10:])]; ok && target != id {
					o.EscortTarget, o.EscortBound = target, true
				}
			}
			orders = append(orders, o)
		}
	}
	if err := ms.World.ImportSavedGroups(groups, orders); err != nil {
		return err
	}
	ms.World.RebuildLoadedActorTraversal()
	report.GroupsRestored, report.GroupIssues = len(groups), ms.World.SavedGroupIssues()
	return nil
}
