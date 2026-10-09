package game

import (
	"cmp"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func bindCurrentSavedGroup(state *SnapshotSAVDocument, world *sim.World, g sim.SavedGroup, players map[uint32]uint16) (SnapshotSAVGroupBinding, error) {
	b := SnapshotSAVGroupBinding{ID: g.ID, ContainerID: g.ContainerID, PlayerObject: players[g.ContainerID]}
	bind := func(ref sim.SavedGroupReference) (SnapshotSAVGroupReferenceBinding, error) {
		v := SnapshotSAVGroupReferenceBinding{Key: ref.Key, Class: ref.Class, Owner: ref.Owner}
		if ref.Class == 0 {
			return v, nil
		}
		for i := range state.Document.Objects {
			r := &state.Document.Objects[i]
			field := "Identity"
			if ref.Class == 1 {
				if r.Class != "Player" {
					continue
				}
				field = "This"
			}
			key, err := savedStructureValue(r, field)
			if err == nil && key == ref.Key {
				if v.ObjectIndex != 0 {
					return v, fmt.Errorf("current Group reference has repeated identity")
				}
				v.ObjectIndex = uint16(i + 1)
			}
		}
		if v.ObjectIndex == 0 && ref.Class == 1 {
			// A document built from the World mints its own Player keys. The
			// Player owner reference names its slot, which is current state.
			for _, p := range state.GroupBindings.Players {
				slot, err := savedStructureValue(&state.Document.Objects[p.ObjectIndex-1], "Slot")
				if err == nil && uint32(uint16(slot)) == ref.Owner {
					if v.ObjectIndex != 0 {
						return v, fmt.Errorf("current Group Player reference has repeated slot")
					}
					v.ObjectIndex = p.ObjectIndex
				}
			}
		}
		if v.ObjectIndex == 0 {
			return v, fmt.Errorf("current Group reference has no exact object")
		}
		return v, nil
	}
	var err error
	if b.Reference, err = bind(g.Reference); err != nil {
		return b, err
	}
	if b.Owner, err = bind(g.Owner); err != nil {
		return b, err
	}
	if b.PlayerObject == 0 && g.ContainerID == 0 {
		if b.Owner.Class == 1 {
			b.PlayerObject = b.Owner.ObjectIndex
		}
		for _, member := range g.Members {
			if b.PlayerObject != 0 || !member.Bound {
				continue
			}
			for _, e := range world.Entities() {
				if e.ID != member.Entity {
					continue
				}
				for _, p := range state.GroupBindings.Players {
					slot, err := savedStructureValue(&state.Document.Objects[p.ObjectIndex-1], "Slot")
					if err == nil && uint32(uint16(slot)) == e.Owner {
						if b.PlayerObject != 0 {
							return b, fmt.Errorf("current Group lacks exact enclosing Player")
						}
						b.PlayerObject = p.ObjectIndex
					}
				}
			}
		}
		if b.PlayerObject == 0 && len(state.GroupBindings.Players) > 0 {
			b.PlayerObject = state.GroupBindings.Players[0].ObjectIndex
		}
	}
	if b.PlayerObject == 0 {
		return b, fmt.Errorf("current Group lacks its enclosing Player")
	}
	return b, nil
}

// A partial native registry can coexist with live actors outside every Group.
// Ordinary SAV still needs a root for them. Only this absent topology uses a
// neutral transport Group; existing current Groups retain all their fields.
func appendCurrentActorRoots(state *SnapshotSAVDocument, world *sim.World) error {
	type player struct {
		binding   SnapshotSAVGroupPlayerBinding
		slot, key uint32
	}
	var players []player
	rooted, usedIDs := map[uint16]bool{}, map[uint32]bool{}
	for _, id := range state.Document.DeadActors {
		rooted[id] = true
	}
	for _, b := range state.GroupBindings.Groups {
		usedIDs[b.ID] = true
		r := &state.Document.Objects[b.PlayerObject-1].Groups[b.InlineIndex]
		refs, ok := savedObjectRefs(r, "Actors")
		if !ok {
			return fmt.Errorf("current Group lacks Actors roots")
		}
		for _, id := range refs {
			rooted[id] = true
		}
	}
	seen := map[uint16]bool{}
	for _, p := range state.GroupBindings.Players {
		if p.ObjectIndex == 0 || int(p.ObjectIndex) > len(state.Document.Objects) || seen[p.ObjectIndex] {
			return fmt.Errorf("current actor roots have invalid Player bindings")
		}
		seen[p.ObjectIndex] = true
		r := &state.Document.Objects[p.ObjectIndex-1]
		slot, err := savedStructureValue(r, "Slot")
		if err != nil || slot > 0xffff || r.Class != "Player" {
			return fmt.Errorf("current actor root Player has invalid owner slot")
		}
		key, err := savedStructureValue(r, "This")
		if err != nil || key == 0 {
			return fmt.Errorf("current actor root Player lacks identity")
		}
		players = append(players, player{p, slot, key})
	}
	actors := map[sim.EntityID]uint16{}
	for _, a := range state.Actors {
		if !a.Retired {
			actors[a.EntityID] = a.ObjectIndex
		}
	}
	type site struct {
		player   uint16
		selector uint32
	}
	type group struct {
		player  player
		current sim.SavedGroup
		refs    []uint16
	}
	var groups []group
	sites := map[site]int{}
	for _, e := range world.Entities() {
		index := actors[e.ID]
		if index == 0 || rooted[index] {
			continue
		}
		if int(index) > len(state.Document.Objects) {
			return fmt.Errorf("current actor root has invalid actor binding")
		}
		actor := &state.Document.Objects[index-1]
		key, err := savedStructureValue(actor, "Reference")
		if err != nil {
			return err
		}
		owner, matches, exact := player{}, 0, false
		for _, p := range players {
			if p.slot != e.Owner {
				continue
			}
			matches++
			if p.key == key {
				if exact {
					return fmt.Errorf("current actor root has repeated Player identity")
				}
				owner, exact = p, true
			} else if !exact {
				owner = p
			}
		}
		if matches == 0 || matches > 1 && !exact {
			return fmt.Errorf("current actor %d lacks an exact current owner root", e.ID)
		}
		s := site{owner.binding.ObjectIndex, e.Group}
		at, found := sites[s]
		if !found {
			id := uint32(1)
			for usedIDs[id] {
				id++
			}
			usedIDs[id] = true
			g := sim.SavedGroup{ID: id, Selector: e.Group}
			g.AI[0x45] = 1
			at, sites[s] = len(groups), len(groups)
			groups = append(groups, group{player: owner, current: g})
		}
		groups[at].refs = append(groups[at].refs, index)
		groups[at].current.Members = append(groups[at].current.Members, sim.SavedGroupMember{Entity: e.ID, Bound: true})
		rooted[index] = true
	}
	for _, g := range groups {
		r := newSavedGroupRecord()
		if err := projectSavedGroupFields(&r, g.current, g.refs, 0, g.player.key); err != nil {
			return err
		}
		p := g.player.binding
		player := &state.Document.Objects[p.ObjectIndex-1]
		b := SnapshotSAVGroupBinding{ID: g.current.ID, PlayerObject: p.ObjectIndex, InlineIndex: uint32(len(player.Groups)), ContainerID: p.ID, RootOnly: true,
			Owner: SnapshotSAVGroupReferenceBinding{Class: 1, Owner: g.player.slot, Key: g.player.key, ObjectIndex: p.ObjectIndex}}
		player.Groups = append(player.Groups, r)
		state.GroupBindings.Groups = append(state.GroupBindings.Groups, b)
		for _, index := range g.refs {
			if err := savedActorSetValue(&state.Document.Objects[index-1], "Reference", g.player.key); err != nil {
				return err
			}
		}
	}
	slices.SortFunc(state.GroupBindings.Groups, func(a, b SnapshotSAVGroupBinding) int { return cmp.Compare(a.ID, b.ID) })
	return nil
}

func validateActorRootGroup(doc *sav.DocumentData, b SnapshotSAVGroupBinding) error {
	neutral, err := actorRootGroupNeutral(doc, b)
	if err != nil {
		return err
	}
	if !neutral {
		return fmt.Errorf("current actor root has conflicting native Group absence")
	}
	return nil
}

func actorRootGroupNeutral(doc *sav.DocumentData, b SnapshotSAVGroupBinding) (bool, error) {
	if b.Authored || b.Reference != (SnapshotSAVGroupReferenceBinding{}) || b.Owner.Class != 1 || b.Owner.ObjectIndex != b.PlayerObject {
		return false, nil
	}
	r := &doc.Objects[b.PlayerObject-1].Groups[b.InlineIndex]
	ref, err := savedStructureValue(r, "G40")
	if err != nil || ref != 0 {
		return false, err
	}
	owner, err := savedStructureValue(r, "G44")
	if err != nil || owner != b.Owner.Key {
		return false, err
	}
	var neutral [80]byte
	neutral[0x45] = 1
	for _, raw := range r.Raw {
		switch raw.Name {
		case "G3C":
			if !slices.Equal(raw.Bytes, neutral[:]) {
				return false, nil
			}
		case "G20", "G4C":
			if len(raw.Bytes) != 0 {
				return false, nil
			}
		}
	}
	refs, ok := savedObjectRefs(r, "Actors")
	if !ok || len(refs) == 0 {
		return false, nil
	}
	seen := map[uint16]bool{}
	for _, index := range refs {
		if index == 0 || int(index) > len(doc.Objects) || seen[index] {
			return false, nil
		}
		seen[index] = true
		key, err := savedStructureValue(&doc.Objects[index-1], "Reference")
		if err != nil || key != b.Owner.Key {
			return false, err
		}
	}
	return true, nil
}
