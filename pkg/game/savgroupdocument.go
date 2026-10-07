package game

import (
	"encoding/binary"
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// projectSavedGroups writes the Group graph and every actor order, then the
// order of every body from current state (projectDeadActorOrders).
func projectSavedGroups(state *SnapshotSAVDocument, world *sim.World) error {
	if err := projectSavedGroupGraph(state, world); err != nil {
		return err
	}
	return projectDeadActorOrders(state, world)
}

// projectDeadActorOrders writes, on every Group path, the order of each body
// whose record still names a living-only state: U50 becomes the body's
// current state and the U158_90 ring is emptied. A record keeps whatever
// order it was loaded or last written with unless a writer replaces it, and
// death clears the entity's patrol and escort (DIV-1432). A body's list under
// any other state is its retained order and stays.
func projectDeadActorOrders(state *SnapshotSAVDocument, world *sim.World) error {
	if state == nil || state.Document == nil || world == nil {
		return nil
	}
	bound := make(map[sim.EntityID]uint16, len(state.Actors))
	for _, a := range state.Actors {
		if !a.Retired && a.ObjectIndex != 0 {
			bound[a.EntityID] = a.ObjectIndex
		}
	}
	if state.GroupBindings != nil {
		for _, m := range state.GroupBindings.Members {
			if _, ok := bound[m.EntityID]; m.Bound && !ok && m.ObjectIndex != 0 {
				bound[m.EntityID] = m.ObjectIndex
			}
		}
	}
	for _, e := range world.Entities() {
		index := bound[e.ID]
		if e.Alive() || index == 0 || int(index) > len(state.Document.Objects) {
			continue
		}
		r := &state.Document.Objects[index-1]
		u50, err := savedActorRaw(r, "U50", 4)
		if err != nil || !sim.LivingOnlyActorState(binary.LittleEndian.Uint32(u50)) {
			continue
		}
		next, sites, err := cloneSavedGroupFieldRecord(r)
		if err != nil {
			return err
		}
		word, err := sites.fixedRaw(&next, "U50", 4)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(word, uint32(e.ActorState))
		if err := sites.wordList(&next, "U158_90", nil); err != nil {
			return fmt.Errorf("actor %d body order: %w", e.ID, err)
		}
		*r = next
	}
	return nil
}

// projectSavedGroupGraph changes the graph and EVERY external local-index binding
// together. Unsupported constructors/lifecycle remain an explicit export gap,
// never a reason to lose ordinary native SAVE. Invalid persisted metadata still
// refuses; it is not repaired from the retained source graph.
func projectSavedGroupGraph(state *SnapshotSAVDocument, world *sim.World) error {
	if state == nil || state.Document == nil || world == nil {
		return fmt.Errorf("saved SAV Groups require a document and world")
	}
	if state.GroupBindings == nil {
		return nil
	}
	next, err := cloneSavedDocument(state)
	if err != nil {
		return err
	}
	unavailable := func(reason string) error {
		// Only the coverage marker changes. All graph/binding edits were made
		// in the independent candidate, including failed late reindexing.
		bindings, err := cloneSavedGroupBindings(state.GroupBindings, state.Document, state.Actors)
		if err != nil {
			return err
		}
		if len(reason) > 4096 {
			reason = reason[:4096]
		}
		bindings.Unavailable = reason
		state.GroupBindings = bindings
		return nil
	}
	groups, orders, present := world.SavedGroups()
	if !present {
		return projectCurrentGroups(state, world)
	}
	gap, err := projectSavedGroupRoster(next, world, groups)
	if err != nil {
		return err
	}
	if gap != "" {
		return unavailable(gap)
	}
	boundMembers := make(map[sim.EntityID]uint16)
	for _, m := range next.GroupBindings.Members {
		if m.Bound {
			boundMembers[m.EntityID] = m.ObjectIndex
		}
	}
	actorObjects := make(map[sim.EntityID]uint16, len(next.Actors))
	for _, a := range next.Actors {
		if !a.Retired {
			actorObjects[a.EntityID] = a.ObjectIndex
		}
	}
	projectSavedPlayerCounts(next.Document)
	orderEntities := world.Entities()
	for _, order := range orders {
		index := actorObjects[order.Entity]
		if index == 0 {
			index = boundMembers[order.Entity]
		}
		if index == 0 {
			return unavailable(fmt.Sprintf("actor %d order lacks a persisted exact binding", order.Entity))
		}
		gap, err := projectSavedCurrentOrder(next, orderEntities, index, order, world)
		if err != nil {
			return err
		}
		if gap != "" {
			return unavailable(gap)
		}
	}
	doc, permutation, err := sav.ReindexDocumentData(*next.Document)
	if err != nil {
		return unavailable(fmt.Sprintf("current Group graph requires lifecycle support: %v", err))
	}
	next.Document = &doc
	if err := remapSavedSackDocument(next, permutation); err != nil {
		return err
	}
	if err := savedGroupReferenceOrder(next); err != nil {
		return unavailable(err.Error())
	}
	next.GroupBindings.Unavailable = ""
	if _, err := cloneSavedGroupBindings(next.GroupBindings, next.Document, next.Actors); err != nil {
		return err
	}
	state.Document, state.Actors, state.GroupBindings = next.Document, next.Actors, next.GroupBindings
	state.Objects = next.Objects
	state.ActorEffects = next.ActorEffects
	state.WorldEffects = next.WorldEffects
	return nil
}

func projectSavedPlayerCounts(doc *sav.DocumentData) {
	for i := range doc.Objects {
		player := &doc.Objects[i]
		if player.Class != "Player" {
			continue
		}
		var members uint32
		for _, g := range player.Groups {
			for _, field := range g.Counts {
				if field.Name == "Actors" {
					members += field.Count
				}
			}
		}
		for j := range player.Counts {
			switch player.Counts[j].Name {
			case "Actors":
				player.Counts[j].Count = members
			case "Groups":
				player.Counts[j].Count = uint32(len(player.Groups))
			}
		}
	}
}

func savedGroupReferenceKey(doc *sav.DocumentData, binding SnapshotSAVGroupReferenceBinding, ref sim.SavedGroupReference) (uint32, error) {
	if binding.Key != ref.Key || binding.Class != ref.Class || binding.Owner != ref.Owner {
		return 0, fmt.Errorf("current Group reference requires a new exact binding")
	}
	if binding.Class == 0 {
		return 0, nil
	} // LOAD resolved null, not the old unresolved address.
	record := &doc.Objects[binding.ObjectIndex-1]
	field := "Identity"
	if record.Class == "Player" {
		field = "This"
	}
	key, err := savedStructureValue(record, field)
	if err != nil || key == 0 {
		return 0, fmt.Errorf("Group reference target lacks an encodable identity")
	}
	// Keys will eventually be reminted by the full writer. Until then an exact
	// local binding is insufficient if two distinct objects share its wire key.
	for i, object := range doc.Objects {
		for _, value := range object.Values {
			if (value.Name == "This" || value.Name == "Identity") && value.Value == key && i+1 != int(binding.ObjectIndex) {
				return 0, fmt.Errorf("Group reference requires collision-free wire keys")
			}
		}
	}
	return key, nil
}

// A resolved key must still name an object encountered by the end of that
// inline Group. Reordering must not silently turn a resolved link into null.
func savedGroupReferenceOrder(state *SnapshotSAVDocument) error {
	type site struct {
		player uint16
		inline uint32
	}
	bindings := make(map[site]SnapshotSAVGroupBinding)
	for _, g := range state.GroupBindings.Groups {
		bindings[site{g.PlayerObject, g.InlineIndex}] = g
	}
	seen := make(map[uint16]bool)
	var walkObject func(uint16) error
	var walkRecord func(*sav.DocumentRecordData, uint16) error
	walkObject = func(index uint16) error {
		if index == 0 || seen[index] {
			return nil
		}
		seen[index] = true
		return walkRecord(&state.Document.Objects[index-1], index)
	}
	walkRecord = func(record *sav.DocumentRecordData, index uint16) error {
		for _, refs := range record.RefSlots {
			for _, target := range refs.Objects {
				if err := walkObject(target); err != nil {
					return err
				}
			}
		}
		for i := range record.Inline {
			if err := walkRecord(&record.Inline[i].Record, 0); err != nil {
				return err
			}
		}
		for i := range record.Groups {
			if err := walkRecord(&record.Groups[i], 0); err != nil {
				return err
			}
			g := bindings[site{index, uint32(i)}]
			for _, ref := range []SnapshotSAVGroupReferenceBinding{g.Reference, g.Owner} {
				if ref.ObjectIndex != 0 && !seen[ref.ObjectIndex] {
					return fmt.Errorf("Group %d reference requires a different serialization order", g.ID)
				}
			}
		}
		return nil
	}
	for _, player := range state.Document.Players {
		if err := walkObject(player); err != nil {
			return err
		}
	}
	return nil
}
