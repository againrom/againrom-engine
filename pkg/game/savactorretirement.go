package game

import (
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func retireCurrentActors(state *SnapshotSAVDocument, world *sim.World) error {
	groups, _, present := world.SavedGroups()
	if state.GroupBindings == nil {
		return nil
	}
	if present {
		if gap, err := projectSavedGroupRoster(state, world, groups); err != nil {
			return err
		} else if gap != "" {
			return worldSaveUnsupportedf("%s", gap)
		}
	} else if err := projectCurrentGroups(state, world); err != nil {
		return err
	}
	projectSavedPlayerCounts(state.Document)
	held := make(map[sim.EntityID]bool)
	for _, e := range world.Entities() {
		held[e.ID] = true
	}
	for _, d := range world.OriginalDeadActors() {
		held[d.ID] = true
	}
	for _, d := range world.CurrentTerminalActors() {
		held[d.ID] = true
	}
	var seeds, protected []uint16
	for _, a := range state.Actors {
		if a.Retired && !held[a.EntityID] {
			seeds = append(seeds, a.ObjectIndex)
		} else {
			protected = append(protected, a.ObjectIndex)
		}
	}
	if r := world.SavedObjects(); r != nil {
		indices := savedObjectIndices(state.Objects)
		for _, item := range r.Items {
			if !item.Retired {
				protected = append(protected, indices[item.ID])
			}
		}
	}
	retired, err := retiredActorClosure(state.Document, seeds, protected)
	if err != nil {
		return err
	}
	removed := make(map[uint16]bool)
	for _, id := range retired {
		removed[id] = true
	}
	state.Actors = slices.DeleteFunc(state.Actors, func(a SnapshotSAVActor) bool { return removed[a.ObjectIndex] })
	state.GroupBindings.Members = slices.DeleteFunc(state.GroupBindings.Members, func(m SnapshotSAVGroupMemberBinding) bool { return removed[m.ObjectIndex] })
	if state.ActorEffects != nil {
		state.ActorEffects.Rows = slices.DeleteFunc(state.ActorEffects.Rows, func(r SnapshotSAVActorEffect) bool { return removed[r.ObjectIndex] })
	}
	doc, permutation, err := sav.RetireDocumentData(*state.Document, retired)
	if err != nil {
		return err
	}
	state.Document = &doc
	if err := remapSavedSackDocument(state, permutation); err != nil {
		return err
	}
	return projectSavedGroups(state, world)
}

// Only descendants of explicitly retired actors are candidates. Every other
// object, including an unrelated orphan, remains a root for the shared-child
// test. This also handles exclusive cycles and references in inline records.
func retiredActorClosure(doc *sav.DocumentData, seeds, protected []uint16) ([]uint16, error) {
	edges := make([][]uint16, len(doc.Objects)+1)
	var refs func(*sav.DocumentRecordData) []uint16
	refs = func(r *sav.DocumentRecordData) []uint16 {
		var out []uint16
		for _, slot := range r.RefSlots {
			out = append(out, slot.Objects...)
		}
		for i := range r.Inline {
			out = append(out, refs(&r.Inline[i].Record)...)
		}
		for i := range r.Groups {
			out = append(out, refs(&r.Groups[i])...)
		}
		return out
	}
	for i := range doc.Objects {
		edges[i+1] = refs(&doc.Objects[i])
		for _, id := range edges[i+1] {
			if int(id) >= len(edges) {
				return nil, fmt.Errorf("retired actor child %d is outside document", id)
			}
		}
	}
	walk := func(roots []uint16) (map[uint16]bool, error) {
		seen := make(map[uint16]bool)
		queue := slices.Clone(roots)
		for len(queue) != 0 {
			id := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			if int(id) >= len(edges) {
				return nil, fmt.Errorf("retired actor root %d is outside document", id)
			}
			if id != 0 && !seen[id] {
				seen[id] = true
				queue = append(queue, edges[id]...)
			}
		}
		return seen, nil
	}
	candidates, err := walk(seeds)
	if err != nil {
		return nil, err
	}
	roots := append(slices.Clone(protected), doc.Players...)
	roots = append(roots, doc.DeadActors...)
	if doc.World != nil {
		roots = append(roots, doc.World.Buildings...)
		roots = append(roots, doc.World.Effects...)
		roots = append(roots, doc.World.Sacks...)
	}
	for i := range doc.Objects {
		if !candidates[uint16(i+1)] {
			roots = append(roots, uint16(i+1))
		}
	}
	kept, err := walk(roots)
	if err != nil {
		return nil, err
	}
	var out []uint16
	for i := range doc.Objects {
		id := uint16(i + 1)
		if candidates[id] && !kept[id] {
			out = append(out, id)
		}
	}
	return out, nil
}
