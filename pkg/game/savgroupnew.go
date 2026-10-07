package game

import (
	"cmp"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// This is a current native-state producer, not a claim about original heap
// defaults. All 76 AI bytes, both lists and the selector come from the current
// registry (DIV788). SAV-GRPAI-563 proves that LOAD replaces the final four AI
// pointer bytes; zero is the encoder's deterministic transport spelling there.
func newSavedGroupRecord() sav.DocumentRecordData {
	return sav.DocumentRecordData{
		Class:    "Group",
		Values:   []sav.DocumentValueData{{Name: "G1C"}, {Name: "G40"}, {Name: "G44"}},
		Raw:      []sav.DocumentRawData{{Name: "G20"}, {Name: "G3C", Bytes: make([]byte, 80)}, {Name: "G4C"}},
		Counts:   []sav.DocumentCountData{{Name: "Actors"}, {Name: "G20"}, {Name: "G4C"}},
		RefSlots: []sav.DocumentRefsData{{Name: "Actors"}},
	}
}

// Generated native command Groups carry a typed slot, not a source address.
// Their exact native container identifies the Player that append installed as
// owner. Without a native Player carrier, the current typed slot must identify
// one ordinary transport Player. This is separate from imported owner precedence
// (SAV-GRPOWNER-561), where owner and enclosing Player can disagree.
func validateAuthoredSavedGroupAuthority(state *SnapshotSAVDocument, binding SnapshotSAVGroupBinding, group sim.SavedGroup) error {
	if !binding.Authored || !group.Authored || binding.ContainerID != group.ContainerID ||
		binding.ContainerID == 0 && !state.GroupBindings.PlayersConstructed ||
		group.Reference != (sim.SavedGroupReference{}) || binding.Reference != (SnapshotSAVGroupReferenceBinding{}) ||
		group.Owner.Class != 1 || group.Owner.Key != 0 || group.Owner.Archive != 0 ||
		binding.Owner.Class != 1 || binding.Owner.ObjectIndex != binding.PlayerObject || binding.Owner.Owner != group.Owner.Owner {
		return fmt.Errorf("Group %d lacks exact native command reference authority", group.ID)
	}
	slot, err := savedStructureValue(&state.Document.Objects[binding.PlayerObject-1], "Slot")
	if err != nil || uint32(uint16(slot)) != group.Owner.Owner {
		return fmt.Errorf("Group %d owner differs from its native command Player", group.ID)
	}
	return nil
}

func bindAuthoredSavedGroup(state *SnapshotSAVDocument, group sim.SavedGroup, players map[uint32]uint16) (SnapshotSAVGroupBinding, error) {
	b := SnapshotSAVGroupBinding{ID: group.ID, Authored: true, ContainerID: group.ContainerID, PlayerObject: players[group.ContainerID]}
	if b.ContainerID == 0 && state.GroupBindings.PlayersConstructed && group.Owner.Class == 1 {
		for _, player := range state.GroupBindings.Players {
			slot, err := savedStructureValue(&state.Document.Objects[player.ObjectIndex-1], "Slot")
			if err != nil {
				return b, err
			}
			if uint32(uint16(slot)) != group.Owner.Owner {
				continue
			}
			if b.PlayerObject != 0 {
				return b, fmt.Errorf("Group %d has ambiguous current Player owner", group.ID)
			}
			b.PlayerObject = player.ObjectIndex
		}
	}
	if !state.GroupBindings.PlayersPresent || b.ContainerID == 0 && !state.GroupBindings.PlayersConstructed || b.PlayerObject == 0 {
		return b, fmt.Errorf("Group %d lacks an exact native Player container", group.ID)
	}
	key, err := savedStructureValue(&state.Document.Objects[b.PlayerObject-1], "This")
	if err != nil || key == 0 {
		return b, fmt.Errorf("Group %d Player lacks a serializable identity", group.ID)
	}
	b.Owner = SnapshotSAVGroupReferenceBinding{ObjectIndex: b.PlayerObject, Class: 1, Key: key, Owner: group.Owner.Owner}
	return b, validateAuthoredSavedGroupAuthority(state, b, group)
}

// Rebuild only inline Player lists from the authoritative current registry.
// Removed Groups consume no archive identity. The enclosing transaction's
// reindex operation still refuses an edit that would orphan any graph object.
func projectSavedGroupRoster(state *SnapshotSAVDocument, world *sim.World, groups []sim.SavedGroup) (string, error) {
	bindings := state.GroupBindings
	old := make(map[uint32]SnapshotSAVGroupBinding, len(bindings.Groups))
	for _, b := range bindings.Groups {
		if !b.RootOnly {
			old[b.ID] = b
		}
	}
	players := make(map[uint32]uint16, len(bindings.Players))
	for _, p := range bindings.Players {
		players[p.ID] = p.ObjectIndex
	}
	currentPlayers, present := world.SavedGroupPlayers()
	if present != (bindings.PlayersPresent && !bindings.PlayersConstructed) || present && len(currentPlayers) != len(nativeSavedPlayerBindings(bindings)) {
		return "native Player containers lack persisted exact bindings", nil
	}
	for _, p := range currentPlayers {
		index := players[p.ID]
		if index == 0 {
			return "native Player lacks a persisted exact binding", nil
		}
		slot, err := savedStructureValue(&state.Document.Objects[index-1], "Slot")
		if err != nil || p.Slot != uint32(uint16(slot)) {
			return "native Player slot differs from its exact document object", nil
		}
	}
	bound, unresolved := make(map[sim.EntityID]uint16), make(map[uint16]uint16)
	for _, m := range bindings.Members {
		if m.Bound {
			bound[m.EntityID] = m.ObjectIndex
		} else {
			unresolved[m.UnresolvedHandle] = m.ObjectIndex
		}
	}
	actors := make(map[sim.EntityID]uint16, len(state.Actors))
	entities := make(map[sim.EntityID]sim.Entity)
	for _, e := range world.Entities() {
		entities[e.ID] = e
	}
	for _, a := range state.Actors {
		if !a.Retired {
			actors[a.EntityID] = a.ObjectIndex
		}
	}
	byPlayer := make(map[uint16][]sav.DocumentRecordData)
	var next []SnapshotSAVGroupBinding
	for _, g := range groups {
		b, existed := old[g.ID]
		var record sav.DocumentRecordData
		if existed {
			if bindings.FormationsPresent && g.OwnerID != savedFormationOwnerID(bindings, b.Owner) {
				return fmt.Sprintf("Group %d changed exact formation owner", g.ID), nil
			}
			if g.Authored != b.Authored || g.ContainerID != b.ContainerID {
				return fmt.Sprintf("Group %d changed container or construction authority", g.ID), nil
			}
			record = state.Document.Objects[b.PlayerObject-1].Groups[b.InlineIndex]
		} else {
			var err error
			if g.Authored {
				b, err = bindAuthoredSavedGroup(state, g, players)
			} else {
				b, err = bindCurrentSavedGroup(state, world, g, players)
			}
			if err != nil {
				return err.Error(), nil
			}
			record = newSavedGroupRecord()
		}
		if g.Authored {
			if err := validateAuthoredSavedGroupAuthority(state, b, g); err != nil {
				return err.Error(), nil
			}
		}
		owner := g.Owner
		if g.Authored {
			// Translate typed native owner to the separately bound wire key.
			// The native record keeps its zero source key unchanged.
			owner.Key = b.Owner.Key
		}
		refKey, err := savedGroupReferenceKey(state.Document, b.Reference, g.Reference)
		if err != nil {
			return err.Error(), nil
		}
		ownerKey, err := savedGroupReferenceKey(state.Document, b.Owner, owner)
		if err != nil {
			return err.Error(), nil
		}
		members := make([]uint16, len(g.Members))
		for i, m := range g.Members {
			if m.Bound {
				members[i] = bound[m.Entity]
				if members[i] == 0 && actors[m.Entity] != 0 {
					members[i], bound[m.Entity] = actors[m.Entity], actors[m.Entity]
					bindings.Members = append(bindings.Members, SnapshotSAVGroupMemberBinding{EntityID: m.Entity, ObjectIndex: members[i], Bound: true})
				}
			} else {
				members[i] = unresolved[m.Archive]
			}
			if members[i] == 0 && (m.Bound || m.Archive != 0) {
				return fmt.Sprintf("Group %d member lacks a persisted exact binding", g.ID), nil
			}
			if g.Authored && m.Bound {
				e, exists := entities[m.Entity]
				if !exists || e.Owner != g.Owner.Owner {
					return fmt.Sprintf("Group %d member has different current ownership", g.ID), nil
				}
				// Ordinary append's owner is current; PARTY-JOIN-025 changes
				// actor+14 before adding the handover Group to its destination.
				// Do not retain a pre-handover Token owner in the output.
				if err := savedActorSetValue(&state.Document.Objects[members[i]-1], "Reference", ownerKey); err != nil {
					return "", err
				}
			}
		}
		current := g
		current.Authored = false // only after constructor/reference authority above
		if err := projectSavedGroupFields(&record, current, members, refKey, ownerKey); err != nil {
			return "", err
		}
		b.InlineIndex = uint32(len(byPlayer[b.PlayerObject]))
		byPlayer[b.PlayerObject] = append(byPlayer[b.PlayerObject], record)
		next = append(next, b)
	}
	seen := make(map[uint16]bool)
	for _, index := range state.Document.Players {
		if index != 0 && !seen[index] {
			state.Document.Objects[index-1].Groups = byPlayer[index]
			seen[index] = true
		}
	}
	slices.SortFunc(next, func(a, b SnapshotSAVGroupBinding) int { return cmp.Compare(a.ID, b.ID) })
	bindings.Groups = next
	slices.SortFunc(bindings.Members, savedGroupMemberCompare)
	return "", appendCurrentActorRoots(state, world)
}
