package game

import (
	"fmt"
	"slices"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// A mission return keeps graph identities separately from the value-only party
// handles. The parallel runtime IDs are used before that boundary drops them.
func currentMissionCityParty(world *sim.World, state *SnapshotSAVDocument, party []mapload.PartyMember, ids []sim.EntityID, roster map[sim.EntityID]mapload.PartyMember, table *mapload.Table) ([]mapload.PartyMember, *cityObjectTopology, error) {
	carried := mapload.DropHired(mapload.CarryRoster(party, world, ids, roster))
	if world == nil {
		return carried, nil, nil
	}
	liveParty, liveIDs := mapload.CarryRosterIDs(party, world, ids, roster)
	byParty := make(map[string]sim.EntityID, len(liveIDs))
	for i, id := range liveIDs {
		if i >= len(liveParty) {
			return nil, nil, fmt.Errorf("city return has unmatched actor bindings")
		}
		if _, duplicate := byParty[liveParty[i].ID]; duplicate {
			return nil, nil, fmt.Errorf("city return has repeated party identity")
		}
		byParty[liveParty[i].ID] = id
	}
	registry := world.SavedObjects()
	if registry == nil {
		projection, err := newCityObjectProjection(nil, carried, table)
		if err != nil {
			return nil, nil, err
		}
		return carried, projection.graph, nil
	}
	living := make(map[sim.EntityID]bool)
	for _, entity := range world.Entities() {
		living[entity.ID] = entity.Alive()
	}
	var actors []cityActorEntityBinding
	for i := range carried {
		member := &carried[i]
		id, bound := byParty[member.ID]
		if !bound || !living[id] {
			continue
		}
		actors = append(actors, cityActorEntityBinding{Entity: id, PartyID: member.ID})
		// A native actor without an original load record still owns ordered,
		// counted stacks. Flattened display instances cannot describe its roots.
		if member.Carry != nil {
			if stacks, ok := world.CarriedStacks(id); ok {
				member.Carry.OrderedStacks = stacks
				for j := range stacks {
					member.Carry.OrderedStacks[j].ObjectID = 0
				}
			}
		}
	}
	graph, err := captureCityObjectTopologyFromSaved(registry, actors)
	if err != nil {
		return nil, nil, err
	}
	if err := captureMissionCityBooks(graph, registry, state, actors, carried, table); err != nil {
		return nil, nil, err
	}
	projection, err := newCityObjectProjection(graph, carried, table)
	if err != nil {
		return nil, nil, err
	}
	return carried, projection.graph, nil
}

// Archive book edges supply identities, while the carried book supplies every
// value. An independently changed book gets a new node when a weapon still owns
// the old Spell; equal independent archive nodes are never merged.
func captureMissionCityBooks(graph *cityObjectTopology, registry *sim.SavedObjects, state *SnapshotSAVDocument, actors []cityActorEntityBinding, party []mapload.PartyMember, table *mapload.Table) error {
	byEntity := make(map[sim.EntityID]uint16)
	byObject := make(map[uint16]sim.SavedObjectID)
	if state != nil && state.Document != nil {
		for _, actor := range state.Actors {
			if !actor.Retired {
				if _, duplicate := byEntity[actor.EntityID]; duplicate {
					return fmt.Errorf("city return has repeated archive actor binding")
				}
				byEntity[actor.EntityID] = actor.ObjectIndex
			}
		}
		for id, index := range savedObjectIndices(state.Objects) {
			if index != 0 {
				if byObject[index] != 0 {
					return fmt.Errorf("city return has repeated archive object binding")
				}
				byObject[index] = id
			}
		}
	}
	spells := make(map[sim.SavedObjectID]sim.SourceItemSpell)
	for _, spell := range registry.Spells {
		if !spell.Retired {
			spells[spell.ID] = spell.Value
		}
	}
	values := make(map[string][28]sim.SourceItemSpell)
	for _, member := range party {
		values[member.ID] = cityMemberBook(member, table)
	}
	type bookKey struct {
		object uint16
		value  sim.SourceItemSpell
	}
	allocated := make(map[bookKey]sim.SavedObjectID)
	nativeBooks := make(map[sim.EntityID][28]sim.SavedObjectID)
	for _, root := range registry.BookRoots {
		nativeBooks[root.Entity] = root.Slots
	}
	for _, actor := range actors {
		var refs []uint16
		if object := byEntity[actor.Entity]; object != 0 {
			if int(object) > len(state.Document.Objects) {
				return fmt.Errorf("city return archive actor is outside its document")
			}
			refs, _ = savedObjectRefs(&state.Document.Objects[object-1], "Spells")
		}
		book := cityBookTopology{PartyID: []byte(actor.PartyID)}
		for slot, value := range values[actor.PartyID] {
			if !value.Present {
				continue
			}
			var ref uint16
			if slot < len(refs) {
				ref = refs[slot]
				if ref != 0 && (int(ref) > len(state.Document.Objects) || state.Document.Objects[ref-1].Class != "Spell") {
					return fmt.Errorf("city return book has an invalid Spell edge")
				}
			}
			id := byObject[ref]
			if root, present := nativeBooks[actor.Entity]; present {
				id = root[slot]
			}
			if old, present := spells[id]; !present || old != value {
				id = 0
			}
			key := bookKey{ref, value}
			if id == 0 && ref != 0 {
				id = allocated[key]
			}
			if id == 0 {
				var err error
				id, err = mintCityObjectID(&graph.NextID)
				if err != nil {
					return err
				}
				if ref != 0 {
					allocated[key] = id
				}
			}
			book.Slots[slot] = id
			if !slices.Contains(graph.Spells, id) {
				graph.Spells = append(graph.Spells, id)
			}
		}
		for i := range graph.Books {
			if string(graph.Books[i].PartyID) == actor.PartyID {
				graph.Books[i] = book
				break
			}
		}
	}
	graph.sort()
	return graph.Validate()
}

func (s *CampaignSession) missionObjectSource(world *sim.World) *SnapshotSAVDocument {
	if s.live != nil && s.live.world == world && s.live.mission != nil && s.live.mission.state != nil {
		return s.live.mission.state.savedDocument
	}
	return nil
}
