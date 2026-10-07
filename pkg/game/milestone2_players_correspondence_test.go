package game

import (
	"fmt"

	"againrom/pkg/sim"
)

// baseline is the first complete snapshot, already checked against original
// origins. It freezes source correspondence before native projection can add
// objects. Current Player/actor bindings provide only local indices; expected
// Player and Diary values continue to come from players1154Expected(raw).
// Scope is the Player/Diary subjects, not every original object in the graph.
func players1154SnapshotOrigins(want players1154Source, original map[uint16]uint16, baseline, current *SnapshotSAVDocument) (map[uint16]uint16, error) {
	if baseline == nil || current == nil || baseline.Document == nil || current.Document == nil || baseline.GroupBindings == nil || current.GroupBindings == nil {
		return nil, fmt.Errorf("Player/Diary continuation has no complete identity bindings")
	}
	type identity struct {
		kind       uint8 // 1: Player ID; 2: actor EntityID; 3: unresolved member handle
		player     uint32
		entity     sim.EntityID
		unresolved uint16
	}
	identities := func(state *SnapshotSAVDocument) (map[uint16]identity, map[identity]uint16, error) {
		byObject, byIdentity := map[uint16]identity{}, map[identity]uint16{}
		add := func(index uint16, id identity) error {
			if index == 0 || int(index) > len(state.Document.Objects) {
				return fmt.Errorf("invalid continuation object %d", index)
			}
			if old, ok := byObject[index]; ok && old != id {
				return fmt.Errorf("distinct continuation identities collapse into object %d", index)
			}
			if old := byIdentity[id]; old != 0 && old != index {
				return fmt.Errorf("continuation identity %+v repeats on objects %d/%d", id, old, index)
			}
			byObject[index], byIdentity[id] = id, index
			return nil
		}
		for _, p := range state.GroupBindings.Players {
			if p.ID == 0 {
				return nil, nil, fmt.Errorf("zero continuation Player identity")
			}
			if err := add(p.ObjectIndex, identity{kind: 1, player: p.ID}); err != nil {
				return nil, nil, err
			}
		}
		for _, a := range state.Actors {
			if err := add(a.ObjectIndex, identity{kind: 2, entity: a.EntityID}); err != nil {
				return nil, nil, err
			}
		}
		for _, m := range state.GroupBindings.Members {
			id := identity{kind: 2, entity: m.EntityID}
			if !m.Bound {
				id = identity{kind: 3, unresolved: m.UnresolvedHandle}
			}
			if err := add(m.ObjectIndex, id); err != nil {
				return nil, nil, err
			}
		}
		return byObject, byIdentity, nil
	}
	baselineIDs, _, err := identities(baseline)
	if err != nil {
		return nil, err
	}
	_, currentObjects, err := identities(current)
	if err != nil {
		return nil, err
	}
	join, used := map[uint16]uint16{}, map[uint16]uint16{}
	bind := func(archive, local uint16) error {
		if archive == 0 || local == 0 || int(local) > len(current.Document.Objects) {
			return fmt.Errorf("missing continuation origin %d -> %d", archive, local)
		}
		if prior := join[archive]; prior != 0 && prior != local {
			return fmt.Errorf("source alias %d split into current objects %d/%d", archive, prior, local)
		}
		if prior := used[local]; prior != 0 && prior != archive {
			return fmt.Errorf("distinct source identities %d/%d collapse into current object %d", prior, archive, local)
		}
		join[archive], used[local] = local, archive
		return nil
	}
	bindOwner := func(archive uint16) error {
		id, found := baselineIDs[original[archive]]
		if !found {
			return fmt.Errorf("source owner archive %d has no frozen Player/actor identity", archive)
		}
		return bind(archive, currentObjects[id])
	}
	for archive := range want.players {
		if err := bindOwner(archive); err != nil {
			return nil, err
		}
	}
	for _, diary := range want.diaries {
		loc := diary.location
		if err := bindOwner(loc.OwnerArchiveIndex); err != nil {
			return nil, err
		}
		if loc.OwnerClass == "Player" || loc.Off < 0 {
			continue
		}
		owner := current.Document.Objects[join[loc.OwnerArchiveIndex]-1]
		var targets []uint16
		found := 0
		for _, slot := range owner.RefSlots {
			if slot.Name == "Diary" {
				targets, found = slot.Objects, found+1
			}
		}
		if found != 1 || len(targets) != 1 {
			return nil, fmt.Errorf("owner archive %d lost its exact Diary edge", loc.OwnerArchiveIndex)
		}
		if err := bind(loc.ArchiveIndex, targets[0]); err != nil {
			return nil, err
		}
	}
	return join, nil
}

// Use the existing comparator after correspondence construction. Null/repeated
// root slots, Player count, scalar values, inline Diaries and tagged Diary
// arrays/counts remain independent raw expectations. Never replace those checks
// with a before/after Document equality comparison.
